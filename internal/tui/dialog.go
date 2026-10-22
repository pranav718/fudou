package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pranav718/fudou/internal/coordinator"
)

func (m model) dialogView(w, h int) string {
	width := min(w, 82)
	inner := width - 6
	var title, subtitle, hint string
	var rows []string
	switch m.dialog {
	case helpDialog:
		title, subtitle = "Make yourself at home.", "A keyboard-first control room for your storage cluster."
		rows = []string{
			"tab / shift+tab    Next / previous view",
			"1 / 2 / 3          Overview / files / nodes",
			"↑ ↓ or j k         Select a file or node",
			"/                  Search files or nodes; esc clears",
			"u                  Back up a local file",
			"enter              Inspect selected file or node",
			"d / x              Restore / delete selected file",
			"r                  Refresh cluster data",
			"q / ctrl+c         Quit / emergency quit",
		}
		hint = keyHint("esc", "back to dashboard")
	case detailDialog:
		if m.page == nodesPage {
			n := m.targetNode
			title, subtitle = clean(n.ID), "Storage node · live cluster member"
			rows = []string{"Address    " + clean(n.Address), "Status     " + clean(n.Status), "Capacity   " + byteSize(n.Capacity), "Used       " + byteSize(n.UsedBytes), "Last seen  " + n.LastSeen.Local().Format("02 Jan 2006 15:04:05 MST"), "", usageBar(n.UsedBytes, n.Capacity, inner-2)}
			hint = keyHint("esc", "close")
		} else {
			f := m.targetFile
			title, subtitle = clean(f.Filename), "Encrypted backup · file details"
			rows = []string{"ID       " + clean(f.ID), "Owner    " + clean(f.UserID), "Size     " + byteSize(f.Size), fmt.Sprintf("Chunks   %d", f.ChunkCount), "Type     " + clean(f.MimeType), "Created  " + f.CreatedAt.Local().Format("02 Jan 2006 15:04"), "SHA-256  " + clean(f.Checksum)}
			hint = keyHint("d", "restore") + "   " + keyHint("x", "delete") + "   " + keyHint("esc", "close")
		}
	case uploadDialog:
		title, subtitle = "Back up a file", "Encrypted with AES-256-GCM, then replicated across your cluster."
		rows = []string{dimStyle.Render("Local file path"), m.fields[0].View(), "", dimStyle.Render("Supports absolute, relative, and ~/ paths. No shell escaping needed."), warnStyle.Render("You will receive a one-time decryption key. Keep it safe.")}
		hint = keyHint("enter", "start backup") + "   " + keyHint("esc", "cancel")
	case restoreDialog:
		title, subtitle = "Restore your file", clean(m.targetFile.Filename)
		rows = []string{dimStyle.Render("Decryption key · 64 hex characters"), m.fields[0].View(), dimStyle.Render("Save to · existing files are never overwritten"), m.fields[1].View(), "", dimStyle.Render("Size and SHA-256 are verified before the file is published.")}
		hint = keyHint("tab", "next field") + "   " + keyHint("enter", "continue") + "   " + keyHint("esc", "cancel")
	case deleteDialog:
		title, subtitle = "Delete this backup?", clean(m.targetFile.Filename)
		rows = []string{errorStyle.Render("This permanently removes the file and its stored replicas."), "", "File ID  " + clean(m.targetFile.ID), "Size     " + byteSize(m.targetFile.Size)}
		hint = keyHint("y", "delete permanently") + "   " + keyHint("n / esc", "keep file")
	case keyDialog:
		title, subtitle = "Backup complete. Your key is yours to keep.", clean(m.backup.Filename)
		key := m.backup.KeyHex
		if inner < 64 && len(key) == 64 {
			key = key[:32] + "\n" + key[32:]
		}
		rows = []string{warnStyle.Render("The coordinator does not retain this key."), "", accentStyle.Bold(true).Render(key), "", dimStyle.Render("Save it now. Without it, this backup cannot be restored.")}
		hint = keyHint("s", "save key to a private file") + "   " + keyHint("enter", "I have saved my key")
	case saveKeyDialog:
		title, subtitle = "Save your decryption key", "Creates a new private file (0600). Existing files are never overwritten."
		rows = []string{dimStyle.Render("Key file path"), m.fields[0].View(), "", warnStyle.Render("Keep this file separate from the backup. Anyone with it can decrypt.")}
		hint = keyHint("enter", "save key") + "   " + keyHint("esc", "back to key")
	}
	content := titleStyle.Render(clip(title, inner)) + "\n" + dimStyle.Render(clip(subtitle, inner)) + "\n\n" + strings.Join(rows, "\n")
	if m.busy {
		content += "\n\n" + accentStyle.Render(m.spin.View()+" "+m.busyLabel)
	} else if m.dialogErr != "" {
		content += "\n\n" + warnStyle.Render(clip(clean(m.dialogErr), inner))
	}
	content += "\n\n" + hint
	// Keep actions visible on small terminals without cutting the key or inputs.
	if lipgloss.Height(content)+2 > h {
		lines := strings.Split(content, "\n")
		compact := make([]string, 0, len(lines))
		for _, line := range lines {
			if line != "" {
				compact = append(compact, line)
			}
		}
		content = strings.Join(compact, "\n")
	}
	box := panel(fit(content, width-4, min(h-2, lipgloss.Height(content))), width, min(h, lipgloss.Height(content)+2), true)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

func saveBackupKey(path string, result coordinator.BackupResult) error {
	path, err := expandClientPath(path)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create key file: %w", err)
	}
	if _, err := fmt.Fprintln(file, result.KeyHex); err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("write key file: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("close key file: %w", err)
	}
	return nil
}
