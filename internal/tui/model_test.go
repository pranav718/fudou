package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/metadata"
)

func testModel() model {
	m := newModel(context.Background(), nil, Config{CoordinatorURL: "http://localhost:8080", RefreshInterval: 5 * time.Second})
	m.refreshing = false
	m.lastSync = time.Now()
	m.data.Files = []metadata.FileRecord{{ID: "a", Filename: "report.pdf", UserID: "alice", Size: 1024, ChunkCount: 1}, {ID: "b", Filename: "photo.jpg", UserID: "bob"}}
	m.data.Nodes = []metadata.NodeRecord{{ID: "node-1", Address: "http://localhost:9001", Status: "online", Capacity: 10000, UsedBytes: 100}}
	return m
}

func press(m model, key string) model {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg.Type = tea.KeyEnter
	case "esc":
		msg.Type = tea.KeyEsc
	default:
		msg.Type, msg.Runes = tea.KeyRunes, []rune(key)
	}
	updated, _ := m.Update(msg)
	return updated.(model)
}

func TestNavigationSearchAndDeleteConfirmation(t *testing.T) {
	m := press(testModel(), "2")
	if m.page != filesPage {
		t.Fatal("2 should open files")
	}
	m = press(m, "/")
	m = press(m, "alice")
	m = press(m, "enter")
	if files := m.filteredFiles(); len(files) != 1 || files[0].ID != "a" {
		t.Fatalf("search results: %v", files)
	}
	m = press(m, "x")
	if m.dialog != deleteDialog || m.busy {
		t.Fatal("delete must require confirmation before any request")
	}
	m = press(m, "n")
	if m.dialog != noDialog {
		t.Fatal("n should cancel deletion")
	}
	m = press(m, "esc")
	if len(m.filteredFiles()) != 2 {
		t.Fatal("esc should clear search")
	}
}

func TestRefreshPreservesSelection(t *testing.T) {
	m := testModel()
	m.page, m.selected[filesPage] = filesPage, 1
	data := m.data
	data.Files = []metadata.FileRecord{data.Files[1], data.Files[0]}
	updated, _ := m.Update(snapshotMsg{data: data})
	file, ok := updated.(model).currentFile()
	if !ok || file.ID != "b" {
		t.Fatal("refresh changed selected file")
	}
}

func TestKeyAcknowledgementAndPrivateSave(t *testing.T) {
	m := testModel()
	m.dialog, m.backup = keyDialog, coordinator.BackupResult{KeyHex: strings.Repeat("a", 64)}
	m = press(m, "esc")
	if m.dialog != keyDialog {
		t.Fatal("key must not be silently dismissed")
	}
	m = press(m, "enter")
	if m.dialog != noDialog || m.backup.KeyHex != "" {
		t.Fatal("acknowledgement must clear key from model")
	}
	path := filepath.Join(t.TempDir(), "backup.key")
	result := coordinator.BackupResult{KeyHex: strings.Repeat("b", 64)}
	if err := saveBackupKey(path, result); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("key permissions: %v", info.Mode())
	}
	if err := saveBackupKey(path, coordinator.BackupResult{KeyHex: "overwrite"}); err == nil {
		t.Fatal("existing key file was overwritten")
	}
}

func TestAllScreensFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 15}, {60, 22}, {80, 24}, {100, 32}, {160, 48}} {
		for p := overviewPage; p <= nodesPage; p++ {
			for d := noDialog; d <= saveKeyDialog; d++ {
				m := testModel()
				m.width, m.height, m.page, m.dialog = size[0], size[1], p, d
				m.targetFile, m.targetNode = m.data.Files[0], m.data.Nodes[0]
				m.backup = coordinator.BackupResult{Filename: "report.pdf", KeyHex: strings.Repeat("a", 64)}
				m.setFields([]string{"key", "path"}, []string{"key", "path"}, []string{"", "./restored.pdf"})
				view := m.View()
				if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
					t.Fatalf("size %v page %d dialog %d rendered %dx%d", size, p, d, lipgloss.Width(view), lipgloss.Height(view))
				}
				if d == keyDialog && size[0] >= 60 && size[1] >= 22 {
					if strings.Count(ansi.Strip(view), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") != 2 {
						t.Fatal("full key must remain visible at all supported sizes")
					}
				}
			}
		}
	}
}

func TestCleanRemovesTerminalEscapes(t *testing.T) {
	if got := clean("\x1b[31mname\x1b[0m\n\x07\u202e"); got != "name" {
		t.Fatalf("unsafe filename: %q", got)
	}
}
