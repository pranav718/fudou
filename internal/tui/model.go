package tui

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranav718/fudou/internal/coordinator"
	"github.com/pranav718/fudou/internal/metadata"
)

type Config struct {
	CoordinatorURL  string
	UserID          string
	RefreshInterval time.Duration
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.RefreshInterval < time.Second {
		return errors.New("refresh interval must be at least 1s")
	}
	client, err := NewClient(cfg.CoordinatorURL, cfg.UserID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, err = tea.NewProgram(newModel(ctx, client, cfg), tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}

type backend interface {
	Snapshot(context.Context) (Snapshot, error)
	Upload(context.Context, string) (coordinator.BackupResult, error)
	Restore(context.Context, metadata.FileRecord, string, string) error
	Delete(context.Context, string) error
}

type page int

const (
	overviewPage page = iota
	filesPage
	nodesPage
)

type dialog int

const (
	noDialog dialog = iota
	helpDialog
	detailDialog
	uploadDialog
	restoreDialog
	deleteDialog
	keyDialog
	saveKeyDialog
)

type refreshTick time.Time
type snapshotMsg struct {
	data Snapshot
	err  error
}
type operationMsg struct {
	kind   dialog
	backup coordinator.BackupResult
	label  string
	err    error
}
type activity struct {
	at    time.Time
	text  string
	isErr bool
}

type model struct {
	ctx           context.Context
	client        backend
	cfg           Config
	width         int
	height        int
	page          page
	data          Snapshot
	lastSync      time.Time
	connectionErr error
	refreshing    bool
	spin          spinner.Model
	selected      [3]int
	queries       [3]string
	searching     bool
	search        textinput.Model
	dialog        dialog
	fields        []textinput.Model
	focus         int
	dialogErr     string
	targetFile    metadata.FileRecord
	targetNode    metadata.NodeRecord
	backup        coordinator.BackupResult
	busy          bool
	busyLabel     string
	events        []activity
}

func newModel(ctx context.Context, client backend, cfg Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle
	search := textinput.New()
	search.Prompt = " / "
	search.Placeholder = "Filter by name, ID, owner, or address…"
	search.CharLimit = 256
	return model{ctx: ctx, client: client, cfg: cfg, width: 100, height: 32, spin: s, search: search, refreshing: true}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.fetch(), m.tick(), m.spin.Tick)
}

func (m model) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		data, err := m.client.Snapshot(ctx)
		return snapshotMsg{data, err}
	}
}

func (m model) tick() tea.Cmd {
	return tea.Tick(m.cfg.RefreshInterval, func(t time.Time) tea.Msg { return refreshTick(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.search.Width = max(10, min(m.width-14, 100))
		for i := range m.fields {
			m.fields[i].Width = max(10, min(m.width-18, 66))
		}
		return m, nil
	case refreshTick:
		cmds := []tea.Cmd{m.tick()}
		if !m.refreshing {
			m.refreshing = true
			cmds = append(cmds, m.fetch())
		}
		return m, tea.Batch(cmds...)
	case snapshotMsg:
		m.refreshing = false
		wasOffline := m.connectionErr != nil
		m.connectionErr = msg.err
		if msg.err == nil {
			fileID, nodeID := m.selectedIDs()
			first := m.lastSync.IsZero()
			m.data, m.lastSync = msg.data, time.Now()
			m.reselect(fileID, nodeID)
			if first || wasOffline {
				m.addEvent("Connected to coordinator", false)
			}
		} else if !wasOffline {
			m.addEvent("Coordinator unreachable · retrying automatically", true)
		}
		return m, nil
	case operationMsg:
		m.busy = false
		if msg.err != nil {
			m.dialogErr = msg.err.Error()
			m.addEvent("Operation failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.addEvent(msg.label, false)
		m.dialogErr = ""
		if msg.kind == uploadDialog {
			m.backup = msg.backup
			m.dialog = keyDialog
			m.fields = nil
		} else if msg.kind == saveKeyDialog {
			m.dialog = keyDialog
			m.fields = nil
			m.dialogErr = "Key saved securely. Press Enter to return."
		} else {
			m.closeDialog()
		}
		if !m.refreshing {
			m.refreshing = true
			return m, m.fetch()
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.dialog != noDialog {
			return m.updateDialog(msg)
		}
		if m.searching {
			switch msg.String() {
			case "enter", "esc":
				m.searching = false
				m.search.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				m.queries[m.page] = m.search.Value()
				m.selected[m.page] = 0
				return m, cmd
			}
		}
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab", "right", "l":
			m.page = (m.page + 1) % 3
		case "shift+tab", "left", "h":
			m.page = (m.page + 2) % 3
		case "1", "2", "3":
			m.page = page(msg.String()[0] - '1')
		case "down", "j":
			m.selected[m.page] = min(m.selected[m.page]+1, max(0, m.rowCount()-1))
		case "up", "k":
			m.selected[m.page] = max(0, m.selected[m.page]-1)
		case "home", "g":
			m.selected[m.page] = 0
		case "end", "G":
			m.selected[m.page] = max(0, m.rowCount()-1)
		case "pgdown", "ctrl+d":
			m.selected[m.page] = min(m.selected[m.page]+m.visibleRows(), max(0, m.rowCount()-1))
		case "pgup", "ctrl+u":
			m.selected[m.page] = max(0, m.selected[m.page]-m.visibleRows())
		case "/":
			if m.page != overviewPage {
				m.searching = true
				m.search.SetValue(m.queries[m.page])
				return m, m.search.Focus()
			}
		case "esc":
			m.queries[m.page] = ""
			m.selected[m.page] = 0
		case "r":
			if !m.refreshing {
				m.refreshing = true
				return m, m.fetch()
			}
		case "?":
			m.dialog = helpDialog
		case "u":
			m.dialog = uploadDialog
			return m, m.setFields([]string{"Local file path"}, []string{"~/Documents/backup.zip"}, nil)
		case "enter":
			if m.page == nodesPage {
				nodes := m.filteredNodes()
				if len(nodes) > 0 {
					m.targetNode = nodes[m.selected[nodesPage]]
					m.dialog = detailDialog
				}
			} else if file, ok := m.currentFile(); ok {
				m.targetFile, m.dialog = file, detailDialog
			}
		case "d":
			if file, ok := m.currentFile(); ok {
				return m.beginRestore(file)
			}
		case "x":
			if file, ok := m.currentFile(); ok {
				m.targetFile, m.dialog = file, deleteDialog
			}
		}
		return m, nil
	}
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	cmds = append(cmds, cmd)
	if m.searching {
		m.search, cmd = m.search.Update(msg)
		cmds = append(cmds, cmd)
	}
	if len(m.fields) > 0 && !m.busy {
		m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) addEvent(text string, isErr bool) {
	m.events = append([]activity{{time.Now(), text, isErr}}, m.events...)
	if len(m.events) > 8 {
		m.events = m.events[:8]
	}
}

func (m model) filteredFiles() []metadata.FileRecord {
	query := strings.ToLower(m.queries[filesPage])
	files := make([]metadata.FileRecord, 0, len(m.data.Files))
	for _, f := range m.data.Files {
		if strings.Contains(strings.ToLower(f.Filename+" "+f.ID+" "+f.UserID+" "+f.MimeType), query) {
			files = append(files, f)
		}
	}
	return files
}

func (m model) filteredNodes() []metadata.NodeRecord {
	query := strings.ToLower(m.queries[nodesPage])
	nodes := make([]metadata.NodeRecord, 0, len(m.data.Nodes))
	for _, n := range m.data.Nodes {
		if strings.Contains(strings.ToLower(n.ID+" "+n.Address+" "+n.Status), query) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (m model) currentFile() (metadata.FileRecord, bool) {
	if m.page == nodesPage {
		return metadata.FileRecord{}, false
	}
	files := m.data.Files
	if m.page == filesPage {
		files = m.filteredFiles()
	}
	index := m.selected[m.page]
	if index < 0 || index >= len(files) {
		return metadata.FileRecord{}, false
	}
	return files[index], true
}

func (m model) selectedIDs() (string, string) {
	var fileID, nodeID string
	files, nodes := m.filteredFiles(), m.filteredNodes()
	if i := m.selected[filesPage]; i < len(files) {
		fileID = files[i].ID
	}
	if i := m.selected[nodesPage]; i < len(nodes) {
		nodeID = nodes[i].ID
	}
	return fileID, nodeID
}

func (m *model) reselect(fileID, nodeID string) {
	files, nodes := m.filteredFiles(), m.filteredNodes()
	m.selected[filesPage] = min(m.selected[filesPage], max(0, len(files)-1))
	m.selected[nodesPage] = min(m.selected[nodesPage], max(0, len(nodes)-1))
	m.selected[overviewPage] = min(m.selected[overviewPage], max(0, len(m.data.Files)-1))
	for i, file := range files {
		if file.ID == fileID {
			m.selected[filesPage] = i
		}
	}
	for i, node := range nodes {
		if node.ID == nodeID {
			m.selected[nodesPage] = i
		}
	}
}

func (m model) rowCount() int {
	switch m.page {
	case filesPage:
		return len(m.filteredFiles())
	case nodesPage:
		return len(m.filteredNodes())
	default:
		return len(m.data.Files)
	}
}

func (m model) visibleRows() int { return max(1, m.height-20) }

func (m *model) setFields(labels, placeholders, values []string) tea.Cmd {
	m.fields = make([]textinput.Model, len(labels))
	m.focus, m.dialogErr = 0, ""
	for i := range labels {
		input := textinput.New()
		input.Prompt = "› "
		input.Placeholder = placeholders[i]
		input.CharLimit = 4096
		input.Width = max(10, min(m.width-18, 66))
		if i < len(values) {
			input.SetValue(values[i])
		}
		m.fields[i] = input
	}
	return m.fields[0].Focus()
}

func (m model) beginRestore(file metadata.FileRecord) (tea.Model, tea.Cmd) {
	m.targetFile, m.dialog = file, restoreDialog
	name := filepath.Base(file.Filename)
	if name == "." || name == string(filepath.Separator) || name == ".." {
		name = "restored-file"
	}
	cmd := m.setFields([]string{"Decryption key", "Save to"}, []string{"64-character hexadecimal key", "Destination path"}, []string{"", "./" + name})
	m.fields[0].EchoMode = textinput.EchoPassword
	m.fields[0].EchoCharacter = '•'
	m.fields[0].CharLimit = 64
	return m, cmd
}

func (m *model) closeDialog() {
	m.dialog, m.dialogErr, m.fields = noDialog, "", nil
	m.backup = coordinator.BackupResult{}
}

func (m model) updateDialog(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := key.String()
	if m.busy {
		return m, nil
	}
	if m.dialog == keyDialog {
		switch k {
		case "enter":
			m.closeDialog()
		case "s":
			m.dialog = saveKeyDialog
			name := filepath.Base(m.backup.Filename) + ".key"
			return m, m.setFields([]string{"Key file path"}, []string{"./backup.key"}, []string{"./" + name})
		case "esc", "q":
			m.dialogErr = "Save this key first, then press Enter to acknowledge."
		}
		return m, nil
	}
	if k == "esc" || (k == "q" && len(m.fields) == 0) {
		if m.dialog == saveKeyDialog {
			m.dialog, m.fields, m.dialogErr = keyDialog, nil, ""
		} else {
			m.closeDialog()
		}
		return m, nil
	}
	switch m.dialog {
	case helpDialog:
		if k == "?" || k == "enter" {
			m.closeDialog()
		}
		return m, nil
	case detailDialog:
		if k == "enter" {
			m.closeDialog()
		} else if m.page != nodesPage {
			if k == "d" {
				return m.beginRestore(m.targetFile)
			} else if k == "x" {
				m.dialog = deleteDialog
			}
		}
		return m, nil
	case deleteDialog:
		if k == "y" {
			m.busy, m.busyLabel = true, "Deleting file and replicas…"
			file := m.targetFile
			return m, func() tea.Msg {
				err := m.client.Delete(m.ctx, file.ID)
				return operationMsg{kind: deleteDialog, label: "Deleted " + file.Filename, err: err}
			}
		} else if k == "n" {
			m.closeDialog()
		}
		return m, nil
	}
	if k == "tab" || k == "shift+tab" || k == "down" || k == "up" {
		m.fields[m.focus].Blur()
		step := 1
		if k == "shift+tab" || k == "up" {
			step = len(m.fields) - 1
		}
		m.focus = (m.focus + step) % len(m.fields)
		return m, m.fields[m.focus].Focus()
	}
	if k == "enter" {
		if m.focus < len(m.fields)-1 {
			m.fields[m.focus].Blur()
			m.focus++
			return m, m.fields[m.focus].Focus()
		}
		return m.submit()
	}
	var cmd tea.Cmd
	m.fields[m.focus], cmd = m.fields[m.focus].Update(key)
	return m, cmd
}

func (m model) submit() (tea.Model, tea.Cmd) {
	for _, field := range m.fields {
		if strings.TrimSpace(field.Value()) == "" {
			m.dialogErr = "Please complete all fields."
			return m, nil
		}
	}
	m.dialogErr = ""
	switch m.dialog {
	case uploadDialog:
		path := m.fields[0].Value()
		m.busy, m.busyLabel = true, "Encrypting and distributing backup…"
		return m, func() tea.Msg {
			result, err := m.client.Upload(m.ctx, path)
			return operationMsg{kind: uploadDialog, backup: result, label: "Backed up " + result.Filename, err: err}
		}
	case restoreDialog:
		key, destination := strings.TrimSpace(m.fields[0].Value()), m.fields[1].Value()
		if decoded, err := hex.DecodeString(key); err != nil || len(decoded) != 32 {
			m.dialogErr = "The key must contain exactly 64 hexadecimal characters."
			return m, nil
		}
		m.fields[0].SetValue("")
		m.busy, m.busyLabel = true, "Restoring and verifying SHA-256…"
		file := m.targetFile
		return m, func() tea.Msg {
			err := m.client.Restore(m.ctx, file, key, destination)
			return operationMsg{kind: restoreDialog, label: "Restored " + file.Filename + " → " + destination, err: err}
		}
	case saveKeyDialog:
		path, result := m.fields[0].Value(), m.backup
		m.busy, m.busyLabel = true, "Saving key securely…"
		return m, func() tea.Msg {
			err := saveBackupKey(path, result)
			return operationMsg{kind: saveKeyDialog, label: "Key saved → " + path, err: err}
		}
	}
	return m, nil
}

func (m model) connectionLabel() string {
	if m.connectionErr != nil {
		return "RECONNECTING"
	}
	if m.lastSync.IsZero() {
		return "CONNECTING"
	}
	return "LIVE"
}

func (m model) scopeLabel() string {
	if m.cfg.UserID == "" {
		return "all owners"
	}
	return fmt.Sprintf("owner: %s", clean(m.cfg.UserID))
}
