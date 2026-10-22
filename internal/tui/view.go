package tui

import (
	"fmt"
	"math"
	"strings"

	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	ink    = lipgloss.AdaptiveColor{Light: "#243047", Dark: "#DBE4F3"}
	muted  = lipgloss.AdaptiveColor{Light: "#65748B", Dark: "#7F8DA7"}
	line   = lipgloss.AdaptiveColor{Light: "#CED7E5", Dark: "#303B52"}
	violet = lipgloss.AdaptiveColor{Light: "#673AC6", Dark: "#B69BFF"}
	cyan   = lipgloss.AdaptiveColor{Light: "#007A89", Dark: "#67DDE8"}
	green  = lipgloss.AdaptiveColor{Light: "#167544", Dark: "#81D8A0"}
	amber  = lipgloss.AdaptiveColor{Light: "#936000", Dark: "#F3C77A"}
	red    = lipgloss.AdaptiveColor{Light: "#BC354F", Dark: "#F58AA2"}
	shade  = lipgloss.AdaptiveColor{Light: "#EAE5FA", Dark: "#2B2543"}

	textStyle   = lipgloss.NewStyle().Foreground(ink)
	dimStyle    = lipgloss.NewStyle().Foreground(muted)
	accentStyle = lipgloss.NewStyle().Foreground(violet)
	goodStyle   = lipgloss.NewStyle().Foreground(green)
	warnStyle   = lipgloss.NewStyle().Foreground(amber)
	errorStyle  = lipgloss.NewStyle().Foreground(red)
	titleStyle  = lipgloss.NewStyle().Foreground(ink).Bold(true)
)

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}

func clip(s string, width int) string {
	return ansi.Truncate(s, max(0, width), "…")
}

func cell(s string, width int) string {
	return lipgloss.NewStyle().Width(max(0, width)).Render(clip(s, width))
}

func panel(content string, width, height int, focused bool) string {
	border := line
	if focused {
		border = violet
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(0, 1).Width(max(0, width-4)).Height(max(0, height-2)).Render(content)
}

func fit(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:max(0, height)]
	}
	for i := range lines {
		lines[i] = cell(lines[i], width)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", max(0, width)))
	}
	return strings.Join(lines, "\n")
}

func (m model) View() string {
	if m.width < 60 || m.height < 22 {
		return lipgloss.Place(max(1, m.width), max(1, m.height), lipgloss.Center, lipgloss.Center,
			fit("◈ fudou\n\nA little more room, please.\nResize to at least 60 × 22.\nCtrl+C to quit", max(1, m.width), min(5, max(1, m.height))))
	}
	w, h := min(140, m.width-4), m.height-10
	brand := accentStyle.Bold(true).Render("◈ fudou") + dimStyle.Render("  /  distributed vault")
	status := goodStyle.Render("● LIVE")
	if m.connectionErr != nil {
		status = errorStyle.Render("● RECONNECTING")
	} else if m.lastSync.IsZero() {
		status = warnStyle.Render(m.spin.View() + " CONNECTING")
	} else if m.refreshing {
		status = accentStyle.Render(m.spin.View() + " SYNCING")
	}
	header := cell(brand, w-lipgloss.Width(status)) + status
	subtitle := dimStyle.Render(clip(clean(m.cfg.CoordinatorURL)+"  ·  "+m.scopeLabel(), w))
	tabs := []string{"1  Overview", fmt.Sprintf("2  Files %d", len(m.data.Files)), fmt.Sprintf("3  Nodes %d", len(m.data.Nodes))}
	for i := range tabs {
		style := dimStyle.Padding(0, 2)
		if page(i) == m.page {
			style = accentStyle.Bold(true).Background(shade).Padding(0, 2)
		}
		tabs[i] = style.Render(tabs[i])
	}
	var body string
	if m.dialog != noDialog {
		body = m.dialogView(w, h)
	} else {
		switch m.page {
		case overviewPage:
			body = m.overviewView(w, h)
		case filesPage:
			body = m.filesView(w, h)
		case nodesPage:
			body = m.nodesView(w, h)
		}
	}
	footer := keyHint("tab", "navigate") + "   " + keyHint("u", "backup") + "   " + keyHint("r", "refresh") + "   " + keyHint("?", "help") + "   " + keyHint("q", "quit")
	if m.page != overviewPage {
		footer = keyHint("↑↓", "select") + "  " + keyHint("/", "filter") + "  " + keyHint("enter", "details") + "  " + keyHint("u", "backup") + "  " + keyHint("?", "help") + "  " + keyHint("q", "quit")
	}
	bottom := dimStyle.Render("AES-256-GCM  ·  SHA-256 verified restores")
	if m.connectionErr != nil {
		bottom = errorStyle.Render(clip(clean(m.connectionErr.Error()), w))
	} else if !m.lastSync.IsZero() {
		bottom = dimStyle.Render(fmt.Sprintf("Synced %s  ·  auto-refresh %s", m.lastSync.Format("15:04:05"), m.cfg.RefreshInterval))
	}
	screen := strings.Join([]string{header, subtitle, "", strings.Join(tabs, " "), "", fit(body, w, h), "", clip(footer, w), clip(bottom, w)}, "\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(screen)
}

func keyHint(key, description string) string {
	return accentStyle.Render(key) + dimStyle.Render(" "+description)
}

func (m model) metricsView(w int) string {
	metrics := m.data.Metrics
	values := []string{fmt.Sprint(metrics.TotalFiles), byteSize(int64(metrics.TotalBytes)), fmt.Sprintf("%d / %d", metrics.ActiveNodes, len(m.data.Nodes)), fmt.Sprintf("%d×", metrics.ReplicationFactor)}
	if m.lastSync.IsZero() {
		values = []string{"—", "—", "—", "—"}
	}
	titles := []string{"BACKED UP", "LOGICAL SIZE", "NODES ONLINE", "REPLICATION"}
	details := []string{"files in cluster", "before replication", "heartbeat status", "target copies"}
	colors := []lipgloss.TerminalColor{violet, cyan, green, amber}
	cards := make([]string, 4)
	cardWidth := (w - 3) / 4
	for i := range cards {
		cw := cardWidth
		if i == 3 {
			cw = w - 3 - cardWidth*3
		}
		content := dimStyle.Render(clip(titles[i], cw-4)) + "\n" + lipgloss.NewStyle().Bold(true).Foreground(colors[i]).Render(values[i])
		if cw >= 20 {
			content += "\n" + dimStyle.Render(details[i])
		}
		cards[i] = panel(content, cw, 5, false)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cards[0], " ", cards[1], " ", cards[2], " ", cards[3])
}

func (m model) overviewView(w, h int) string {
	metrics := m.data.Metrics
	top := m.metricsView(w)
	remaining := h - 6
	if h >= 19 {
		capacity := titleStyle.Render("Storage fabric") + dimStyle.Render("  ·  physical capacity across registered nodes") + "\n"
		capacity += usageBar(metrics.TotalUsed, metrics.TotalCapacity, max(10, w-37)) + "  " + dimStyle.Render(byteSize(metrics.TotalUsed)+" / "+byteSize(metrics.TotalCapacity))
		health := "No nodes registered · start storage nodes to accept backups"
		if len(m.data.Nodes) > 0 {
			health = fmt.Sprintf("%d online · %d offline · target replication %d×", metrics.ActiveNodes, max(0, len(m.data.Nodes)-metrics.ActiveNodes), metrics.ReplicationFactor)
		}
		capacity += "\n" + dimStyle.Render(health)
		top += "\n\n" + panel(capacity, w, 5, false)
		remaining -= 7
	}
	if w >= 98 && remaining >= 5 {
		leftWidth := w * 3 / 5
		return top + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top,
			m.recentView(leftWidth, remaining), " ", m.activityView(w-leftWidth-1, remaining))
	}
	return top + "\n\n" + m.recentView(w, max(4, remaining))
}

func (m model) recentView(w, h int) string {
	rows := []string{titleStyle.Render("Recent backups") + dimStyle.Render("  ·  enter to inspect")}
	if len(m.data.Files) == 0 {
		rows = append(rows, "", dimStyle.Render("Your vault is ready for its first file."), keyHint("u", "back up a local file"))
	} else {
		limit := max(1, h-4)
		start := max(0, m.selected[overviewPage]-limit+1)
		for i := start; i < min(len(m.data.Files), start+limit); i++ {
			file := m.data.Files[i]
			prefix := "  "
			style := textStyle
			if i == m.selected[overviewPage] {
				prefix, style = "› ", accentStyle.Bold(true)
			}
			rows = append(rows, style.Render(prefix+cell(clean(file.Filename), max(5, w-18)))+dimStyle.Render(cell(byteSize(file.Size), 10)))
		}
		rows = append(rows, dimStyle.Render(fmt.Sprintf("%d files in this view · 2 opens the full catalog", len(m.data.Files))))
	}
	return panel(strings.Join(rows, "\n"), w, h, true)
}

func (m model) activityView(w, h int) string {
	rows := []string{titleStyle.Render("Session activity"), ""}
	for i, event := range m.events {
		if i >= max(0, h-5) {
			break
		}
		dot := goodStyle.Render("· ")
		if event.isErr {
			dot = errorStyle.Render("! ")
		}
		rows = append(rows, dimStyle.Render(event.at.Format("15:04 "))+dot+textStyle.Render(clip(clean(event.text), w-13)))
	}
	if len(m.events) == 0 {
		rows = append(rows, dimStyle.Render("Waiting for coordinator…"))
	}
	return panel(strings.Join(rows, "\n"), w, h, false)
}

func (m model) searchView() string {
	if m.searching {
		return accentStyle.Render(m.search.View()) + dimStyle.Render("  enter to apply · esc to finish")
	}
	query := m.queries[m.page]
	if query != "" {
		return accentStyle.Render(" / "+clean(query)) + dimStyle.Render("   esc to clear")
	}
	return dimStyle.Render(" / Search by name, ID, owner, or address")
}

func (m model) filesView(w, h int) string {
	files := m.filteredFiles()
	rows := []string{titleStyle.Render("File catalog") + dimStyle.Render(fmt.Sprintf("  %d of %d files", len(files), len(m.data.Files))), m.searchView(), ""}
	wide := w >= 94
	nameWidth := w - 40
	if wide {
		nameWidth -= 19
	}
	heading := "  " + cell("NAME", nameWidth) + cell("SIZE", 11) + cell("CHUNKS", 9)
	if wide {
		heading += cell("OWNER", 19)
	}
	rows = append(rows, dimStyle.Render(heading+"CREATED"), dimStyle.Render(strings.Repeat("─", w-4)))
	limit := max(1, h-10)
	selected := m.selected[filesPage]
	start := max(0, selected-limit+1)
	for i := start; i < min(len(files), start+limit); i++ {
		f := files[i]
		prefix := "  "
		style := textStyle
		if i == selected {
			prefix, style = "› ", accentStyle.Bold(true).Background(shade)
		}
		row := prefix + cell(clean(f.Filename), nameWidth) + cell(byteSize(f.Size), 11) + cell(fmt.Sprint(f.ChunkCount), 9)
		if wide {
			row += cell(clean(f.UserID), 19)
		}
		row += f.CreatedAt.Format("02 Jan 15:04")
		rows = append(rows, style.Render(cell(row, w-4)))
	}
	if len(files) == 0 {
		message := "No matching files. Press esc to clear the filter."
		if m.queries[filesPage] == "" {
			message = "No backups yet. Press u to back up your first file."
		}
		rows = append(rows, "", dimStyle.Render(message))
	}
	for len(rows) < h-4 {
		rows = append(rows, "")
	}
	position := "0 / 0"
	if len(files) > 0 {
		position = fmt.Sprintf("%d / %d", selected+1, len(files))
	}
	rows = append(rows, keyHint("d", "restore")+"   "+keyHint("x", "delete")+"   "+keyHint("enter", "details")+"   "+dimStyle.Render(position))
	return panel(strings.Join(rows, "\n"), w, h, true)
}

func (m model) nodesView(w, h int) string {
	nodes := m.filteredNodes()
	rows := []string{titleStyle.Render("Storage nodes") + dimStyle.Render(fmt.Sprintf("  %d registered", len(m.data.Nodes))), m.searchView(), ""}
	nameWidth := max(12, (w-4)/4)
	rows = append(rows, dimStyle.Render("  "+cell("NODE", nameWidth)+cell("STATUS", 12)+"STORAGE UTILIZATION"), dimStyle.Render(strings.Repeat("─", w-4)))
	limit := max(1, h-10)
	selected := m.selected[nodesPage]
	start := max(0, selected-limit+1)
	for i := start; i < min(len(nodes), start+limit); i++ {
		n := nodes[i]
		prefix, style := "  ", textStyle
		if i == selected {
			prefix, style = "› ", accentStyle.Bold(true)
		}
		status := errorStyle.Render(cell("● offline", 12))
		if n.Status == "online" {
			status = goodStyle.Render(cell("● online", 12))
		}
		barWidth := max(5, w-nameWidth-39)
		rows = append(rows, style.Render(prefix+cell(clean(n.ID), nameWidth))+status+usageBar(n.UsedBytes, n.Capacity, barWidth)+dimStyle.Render("  "+byteSize(n.UsedBytes)+" / "+byteSize(n.Capacity)))
	}
	if len(nodes) == 0 {
		rows = append(rows, "", dimStyle.Render("No nodes found. Start nodes with make run-node1 (and 2, 3)."))
	}
	for len(rows) < h-4 {
		rows = append(rows, "")
	}
	rows = append(rows, keyHint("enter", "inspect node")+dimStyle.Render("   Heartbeats drive online / offline status"))
	return panel(strings.Join(rows, "\n"), w, h, true)
}

func byteSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	x := float64(n)
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	for _, unit := range units {
		x /= 1024
		if x < 1024 || unit == "EiB" {
			return fmt.Sprintf("%.1f %s", x, unit)
		}
	}
	return "—"
}

func usageBar(used, capacity int64, width int) string {
	ratio := 0.0
	if capacity > 0 {
		ratio = math.Max(0, math.Min(1, float64(used)/float64(capacity)))
	}
	filled := int(math.Round(ratio * float64(width)))
	color := cyan
	if ratio > .85 {
		color = amber
	}
	if ratio > .95 {
		color = red
	}
	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("━", filled)) + dimStyle.Render(strings.Repeat("─", max(0, width-filled)))
}
