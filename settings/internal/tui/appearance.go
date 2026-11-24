package tui

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/thongntit/ghostty/settings/internal/configgraph"
)

// ThemeColors is the subset of Ghostty's theme data needed for an honest
// in-app preview. Empty fields remain unknown; the playground never fills
// them with guessed colors.
type ThemeColors struct {
	Background          string
	Foreground          string
	Cursor              string
	CursorText          string
	SelectionBackground string
	SelectionForeground string
	Palette             []string
}

// ThemePreview is supplied by the host after resolving Ghostty's own theme
// files. A name may remain in the inventory while Available is false when its
// source cannot be found on this machine.
type ThemePreview struct {
	Name      string
	Colors    ThemeColors
	Source    string
	Available bool
	Error     string
}

type appearanceLook struct {
	Name      string
	Colors    ThemeColors
	Source    string
	Available bool
	Fallback  bool
	Partial   bool
	Reason    string
}

func (m Model) updateAppearanceHome(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, defaultKeyMap.Quit):
		return m, m.requestQuit()
	case key.Matches(keyMsg, defaultKeyMap.Up):
		if m.appearanceHomeIndex > 0 {
			m.appearanceHomeIndex--
		}
	case key.Matches(keyMsg, defaultKeyMap.Down):
		if m.appearanceHomeIndex < 2 {
			m.appearanceHomeIndex++
		}
	case key.Matches(keyMsg, defaultKeyMap.Enter):
		switch m.appearanceHomeIndex {
		case 0:
			return m, m.beginGlobalSearch()
		case 1:
			m.beginThemeBrowse()
		case 2:
			m.openFullEditor()
		}
	case keyMsg.String() == "/":
		return m, m.beginGlobalSearch()
	case keyMsg.String() == "t":
		m.beginThemeBrowse()
	case keyMsg.String() == "g" || keyMsg.String() == "c":
		m.openFullEditor()
	case key.Matches(keyMsg, defaultKeyMap.Preview):
		m.openPreview()
	case key.Matches(keyMsg, defaultKeyMap.Save):
		m.saveNow()
	case key.Matches(keyMsg, defaultKeyMap.Help):
		m.help = !m.help
	}
	return m, nil
}

func (m *Model) beginThemeBrowse() {
	if !m.appearanceHome {
		m.appearanceHome = true
	}
	m.browseLevel = browseThemes
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.themeSearching = false
	m.themeQuery.Blur()
	m.themeQuery.SetValue("")
	m.applyThemeFilter("")
}

func (m *Model) beginThemeSearch() tea.Cmd {
	m.themeSearching = true
	m.themeQuery.CursorEnd()
	return m.themeQuery.Focus()
}

func (m *Model) updateThemeBrowse(msg tea.Msg) (Model, tea.Cmd) {
	if m.themeSearching {
		return m.updateThemeSearch(msg)
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return *m, nil
	}
	switch {
	case key.Matches(keyMsg, defaultKeyMap.Quit):
		return *m, m.requestQuit()
	case keyMsg.String() == "esc" || keyMsg.String() == "h":
		m.returnToAppearanceHome()
	case keyMsg.String() == "/" || keyMsg.String() == "s":
		return *m, m.beginThemeSearch()
	case key.Matches(keyMsg, defaultKeyMap.Up):
		m.moveTheme(-1)
	case key.Matches(keyMsg, defaultKeyMap.Down):
		m.moveTheme(1)
	case key.Matches(keyMsg, defaultKeyMap.Left):
		m.moveTheme(-1)
	case key.Matches(keyMsg, defaultKeyMap.Right):
		m.moveTheme(1)
	case key.Matches(keyMsg, defaultKeyMap.Enter):
		m.stageSelectedTheme()
	case keyMsg.String() == "g" || keyMsg.String() == "c":
		m.openFullEditor()
	case key.Matches(keyMsg, defaultKeyMap.Preview):
		m.openPreview()
	case key.Matches(keyMsg, defaultKeyMap.Save):
		m.saveNow()
	case key.Matches(keyMsg, defaultKeyMap.Help):
		m.help = !m.help
	}
	return *m, nil
}

func (m *Model) updateThemeSearch(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.themeQuery.SetValue("")
			m.themeSearching = false
			m.themeQuery.Blur()
			m.applyThemeFilter("")
			return *m, nil
		case "enter":
			query := strings.TrimSpace(m.themeQuery.Value())
			m.themeSearching = false
			m.themeQuery.Blur()
			m.applyThemeFilter(query)
			return *m, nil
		}
	}
	var cmd tea.Cmd
	m.themeQuery, cmd = m.themeQuery.Update(msg)
	m.applyThemeFilter(m.themeQuery.Value())
	return *m, cmd
}

func (m *Model) moveTheme(delta int) {
	if len(m.themeVisible) == 0 {
		return
	}
	if m.themeIndex < 0 {
		m.themeIndex = 0
		return
	}
	m.themeIndex = (m.themeIndex + delta + len(m.themeVisible)) % len(m.themeVisible)
}

func (m *Model) applyThemeFilter(query string) {
	previous := m.selectedThemeName()
	names := m.themeNames()
	query = strings.TrimSpace(query)
	m.themeVisible = m.themeVisible[:0]
	for _, name := range names {
		if query == "" || containsFold(name, query) {
			m.themeVisible = append(m.themeVisible, name)
		}
	}
	m.themeIndex = -1
	for index, name := range m.themeVisible {
		if name == previous {
			m.themeIndex = index
			break
		}
	}
	if m.themeIndex < 0 {
		baseline := m.baselineThemeName()
		for index, name := range m.themeVisible {
			if name == baseline {
				m.themeIndex = index
				break
			}
		}
	}
	if m.themeIndex < 0 && len(m.themeVisible) > 0 {
		m.themeIndex = 0
	}
}

func (m Model) themeNames() []string {
	seen := make(map[string]struct{})
	names := make([]string, 0, len(m.themeChoices)+len(m.themePreviews)+1)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, name := range m.themeChoices {
		add(name)
	}
	previewNames := make([]string, 0, len(m.themePreviews))
	for name := range m.themePreviews {
		previewNames = append(previewNames, name)
	}
	sort.Strings(previewNames)
	for _, name := range previewNames {
		add(name)
	}
	add(m.baselineThemeName())
	return names
}

func (m Model) selectedThemeName() string {
	if m.themeIndex < 0 || m.themeIndex >= len(m.themeVisible) {
		return ""
	}
	return m.themeVisible[m.themeIndex]
}

func (m *Model) stageSelectedTheme() {
	name := m.selectedThemeName()
	if name == "" {
		m.status = "No Ghostty themes are available to stage"
		return
	}
	if m.readOnly || !m.CanEdit("theme") {
		m.status = "Theme editing is unavailable for this host"
		return
	}
	if err := m.stageScalar("theme", name); err != nil {
		m.status = "Theme not staged: " + err.Error()
		return
	}
	m.returnToAppearanceHome()
	if m.HasChanges() {
		m.status = "Staged theme " + name + " in memory; save explicitly to write it"
	} else {
		m.status = "Theme unchanged: " + name
	}
}

func (m Model) baselineThemeName() string {
	assignments := m.originalAssignments("theme")
	if len(assignments) == 0 {
		return ""
	}
	return strings.TrimSpace(assignments[len(assignments)-1].Value)
}

func (m Model) candidateThemeName() string {
	assignments := m.optionAssignments("theme")
	if len(assignments) == 0 {
		return m.baselineThemeName()
	}
	return strings.TrimSpace(assignments[len(assignments)-1].Value)
}

func (m Model) themeLook(name string, draft bool) appearanceLook {
	name = strings.TrimSpace(name)
	look := appearanceLook{Name: name}
	if preview, ok := m.themePreviews[name]; ok {
		look.Colors = cloneThemeColors(preview.Colors)
		look.Source = preview.Source
		look.Available = preview.Available
		look.Reason = preview.Error
	}

	lookup := m.originalAssignments
	if draft {
		lookup = m.optionAssignments
	}
	for key, value := range map[string]*string{
		"background":           &look.Colors.Background,
		"foreground":           &look.Colors.Foreground,
		"cursor-color":         &look.Colors.Cursor,
		"cursor-text":          &look.Colors.CursorText,
		"selection-background": &look.Colors.SelectionBackground,
		"selection-foreground": &look.Colors.SelectionForeground,
	} {
		assignments := lookup(key)
		if len(assignments) == 0 {
			continue
		}
		if value != nil {
			*value = strings.TrimSpace(assignments[len(assignments)-1].Value)
		}
	}
	applyPaletteOverrides(&look.Colors, lookup("palette"))
	look.Partial = palettePreviewPartial(look.Colors)
	sourceAvailable := look.Available
	if normalizedPreviewColor(look.Colors.Background) != "" && normalizedPreviewColor(look.Colors.Foreground) != "" {
		if !sourceAvailable {
			look.Fallback = true
		}
		if !sourceAvailable && look.Source == "" {
			look.Source = "configured Ghostty color overrides"
			look.Reason = "Theme source unavailable; showing configured color overrides"
		}
		look.Available = true
	} else {
		look.Available = false
		if look.Reason == "" {
			if name == "" {
				look.Reason = "Ghostty did not provide theme colors for the current configuration"
			} else {
				look.Reason = "Theme colors are unavailable from Ghostty on this machine"
			}
		}
	}
	return look
}

func cloneThemeColors(colors ThemeColors) ThemeColors {
	colors.Palette = append([]string(nil), colors.Palette...)
	return colors
}

func applyPaletteOverrides(colors *ThemeColors, assignments []configgraph.Assignment) {
	if colors == nil || len(assignments) == 0 {
		return
	}
	if len(colors.Palette) == 0 {
		colors.Palette = make([]string, 16)
	}
	for _, assignment := range assignments {
		indexText, value, ok := strings.Cut(strings.TrimSpace(assignment.Value), "=")
		if !ok {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(indexText))
		if err != nil || index < 0 || index >= len(colors.Palette) {
			continue
		}
		colors.Palette[index] = strings.TrimSpace(value)
	}
}

func palettePreviewPartial(colors ThemeColors) bool {
	if len(colors.Palette) == 0 {
		return false
	}
	if len(colors.Palette) < 16 {
		return true
	}
	for _, value := range colors.Palette {
		if normalizedPreviewColor(value) == "" {
			return true
		}
	}
	return false
}

func (m Model) currentThemeLook() appearanceLook {
	return m.themeLook(m.baselineThemeName(), false)
}

func (m Model) candidateThemeLook() appearanceLook {
	return m.themeLook(m.candidateThemeName(), true)
}

func (m Model) hoverThemeLook() appearanceLook {
	name := m.selectedThemeName()
	if name == "" {
		name = m.candidateThemeName()
	}
	return m.themeLook(name, true)
}

func normalizedPreviewColor(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		value = "#" + value[2:]
	}
	if len(value) == 6 {
		value = "#" + value
	}
	if !strings.HasPrefix(value, "#") {
		return ""
	}
	if len(value) != 7 {
		return ""
	}
	for _, char := range value[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return ""
		}
	}
	return value
}

func (m Model) renderAppearanceHeader() string {
	status := "colors only  ·  Enter stages  ·  Ctrl+S saves"
	if m.HasChanges() {
		status = "staged in memory  ·  Ctrl+S saves  ·  no live reload"
		if m.reloadFunc != nil {
			status = "staged in memory  ·  Ctrl+S saves and applies"
		}
	} else if len(m.lastSave) > 0 {
		status = "Ctrl+Z undoes the last save"
	}
	return strings.Join([]string{
		lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("Ghostty Config TUI"), "  ", appearanceCurrentTag.Render("SAFE PREVIEW")),
		mutedStyle.Render(status),
	}, "\n")
}

func (m Model) renderAppearanceHome() string {
	current := m.currentThemeLook()
	candidate := m.candidateThemeLook()
	if m.compactAppearanceLayout() {
		return m.renderCompactAppearanceHome(current, candidate)
	}

	totalWidth := m.appearanceTotalWidth()
	if totalWidth <= 0 {
		totalWidth = 104
	}
	controlsWidth := maxInt(28, totalWidth/3)
	previewWidth := totalWidth - controlsWidth - 2
	label, preview := "CURRENT LOOK", current
	if candidate.Name != current.Name {
		label, preview = "STAGED PREVIEW", candidate
	}
	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderAppearanceControls(controlsWidth),
		"  ",
		renderLookPanel(label, preview, previewWidth),
	)
	rows := []string{
		appearanceKickerStyle.Render("Appearance playground  /  Safe preview"),
		"",
		main,
		"",
		appearanceHintStyle.Render("Theme colors previewed here · Ctrl+S saves"),
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderCompactAppearanceHome(current, candidate appearanceLook) string {
	width := m.appearanceTotalWidth()
	if width <= 0 {
		width = 72
	}
	rows := []string{
		appearanceKickerStyle.Render("Appearance playground  /  Safe preview"),
	}
	rows = append(rows, m.appearanceActions()...)
	rows = append(rows,
		"",
		appearanceCurrentTag.Render("CURRENT")+"  "+truncateDisplay(displayThemeName(current.Name), maxInt(16, width-14)),
		renderCompactTerminalPreview(current, width-appearanceCardStyle.GetHorizontalFrameSize()),
	)
	if candidate.Name != current.Name {
		rows = append(rows, appearanceCandidateTag.Render("STAGED")+"  "+truncateDisplay(displayThemeName(candidate.Name), maxInt(16, width-14)))
		rows = append(rows, renderCompactTerminalPreview(candidate, width-appearanceCardStyle.GetHorizontalFrameSize()))
	}
	if m.status != "" {
		contentWidth := width - appearanceCardStyle.GetHorizontalFrameSize()
		rows = append(rows, "", accentStyle.Render(truncateDisplay(m.status, contentWidth)))
	}
	return appearanceCardStyle.Padding(0, 2).Width(width).Render(strings.Join(rows, "\n"))
}

func (m Model) renderAppearanceAction(index int, label, shortcut string) string {
	marker := "  "
	if index == m.appearanceHomeIndex {
		marker = selectedStyle.Render("▸ ")
	}
	return marker + label + "  " + appearanceHintStyle.Render(shortcut)
}

func (m Model) appearanceActions() []string {
	return []string{
		m.renderAppearanceAction(0, "Search all settings", "/"),
		m.renderAppearanceAction(1, "Browse themes", "t"),
		m.renderAppearanceAction(2, "Categories", "g"),
	}
}

func (m Model) renderAppearanceControls(width int) string {
	contentWidth := width - appearanceCardStyle.GetHorizontalFrameSize()
	rows := append([]string{appearanceCardTitle.Render("Settings"), ""}, m.appearanceActions()...)
	if m.status != "" {
		rows = append(rows, "", accentStyle.Render(truncateDisplay(m.status, contentWidth)))
	}
	return appearanceCardStyle.Width(width).Render(strings.Join(rows, "\n"))
}

func (m Model) renderThemeBrowse() string {
	rows := []string{
		appearanceKickerStyle.Render("THEME LAB  /  CURRENT ⇄ CANDIDATE"),
		appearanceHeroStyle.Render("Browse the Ghostty palette."),
	}
	if m.themeSearching {
		rows = append(rows, m.themeQuery.View())
	} else if m.compactAppearanceLayout() {
		rows = append(rows, appearanceHintStyle.Render("↑/↓ preview · enter stage · / search · esc back"))
	} else {
		rows = append(rows, appearanceHintStyle.Render("↑/↓/←/→ preview immediately · enter stages · / searches · esc returns"))
	}
	if len(m.themeVisible) == 0 {
		rows = append(rows, "", renderUnavailableCard("No Ghostty theme inventory is available.", "Use g for the full editor; theme remains raw-editable."))
		return strings.Join(rows, "\n")
	}
	limit := m.themeRowLimit()
	if m.compactAppearanceLayout() {
		limit = minInt(limit, 2)
	}
	start := maxInt(0, m.themeIndex-limit/2)
	if start+limit > len(m.themeVisible) {
		start = maxInt(0, len(m.themeVisible)-limit)
	}
	end := minInt(len(m.themeVisible), start+limit)
	if start > 0 {
		rows = append(rows, mutedStyle.Render("↑ more"))
	}
	currentName := m.baselineThemeName()
	for index := start; index < end; index++ {
		name := m.themeVisible[index]
		rows = append(rows, m.renderThemeRow(name, index, currentName))
	}
	if end < len(m.themeVisible) && !m.compactAppearanceLayout() {
		rows = append(rows, appearanceHintStyle.Render("↓ more themes"))
	}
	if m.compactAppearanceLayout() {
		return m.renderCompactThemeBrowse(rows)
	}
	totalWidth := m.appearanceTotalWidth()
	listRows := rows[2:]
	if totalWidth <= 0 {
		list := appearanceCardStyle.Render(strings.Join(listRows, "\n"))
		compare := renderThemeCompare("Current", m.currentThemeLook(), "Candidate", m.hoverThemeLook())
		return strings.Join([]string{strings.Join(rows[:2], "\n"), lipgloss.JoinHorizontal(lipgloss.Top, list, compare)}, "\n")
	}
	listWidth := m.themeListWidth(totalWidth)
	list := renderThemeListPanel(listRows, listWidth)
	compareWidth := maxInt(40, totalWidth-listWidth-2)
	compare := renderThemeCompare("Current", m.currentThemeLook(), "Candidate", m.hoverThemeLook(), compareWidth)
	return strings.Join([]string{
		strings.Join(rows[:2], "\n"),
		lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", compare),
		appearanceHintStyle.Render("Candidate preview is transient until Enter · no config writes while browsing."),
	}, "\n")
}

func (m Model) renderThemeRow(name string, index int, currentName string) string {
	preview := m.themePreviews[name]
	swatch := renderThemeSwatch(preview)
	tag := ""
	if name == currentName {
		tag = "  " + appearanceCurrentTag.Render("CURRENT")
	}
	if !preview.Available {
		tag += "  " + appearanceHintStyle.Render("unavailable")
	}
	label := swatch + "  " + name + tag
	if index == m.themeIndex {
		return appearanceSelectedRow.Render("▸ " + label)
	}
	return "  " + label
}

func renderThemeSwatch(preview ThemePreview) string {
	if !preview.Available {
		return errorStyle.Render("?")
	}
	background := normalizedPreviewColor(preview.Colors.Background)
	foreground := normalizedPreviewColor(preview.Colors.Foreground)
	if background == "" || foreground == "" {
		return errorStyle.Render("?")
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(background)).Foreground(lipgloss.Color(foreground)).Render("●")
}

func (m Model) renderCompactThemeBrowse(rows []string) string {
	width := m.appearanceTotalWidth()
	if width <= 0 {
		width = 72
	}
	rows = append(rows,
		"",
		appearanceCurrentTag.Render("CURRENT")+"  "+truncateDisplay(displayThemeName(m.currentThemeLook().Name), width-12),
		renderCompactTerminalPreview(m.currentThemeLook(), width-appearanceCardStyle.GetHorizontalFrameSize()),
		appearanceCandidateTag.Render("CANDIDATE")+"  "+truncateDisplay(displayThemeName(m.hoverThemeLook().Name), width-14),
		renderCompactTerminalPreview(m.hoverThemeLook(), width-appearanceCardStyle.GetHorizontalFrameSize()),
	)
	return appearanceCardStyle.Padding(0, 2).Width(width).Render(strings.Join(rows, "\n"))
}

func renderUnavailableCard(title, detail string) string {
	return appearanceCardStyle.Render(strings.Join([]string{
		errorStyle.Render("PREVIEW UNAVAILABLE"),
		appearanceHeroStyle.Render(title),
		appearanceHintStyle.Render(detail),
	}, "\n"))
}

// compactAppearanceLayout reserves the intentionally dense fallback for a
// genuinely small terminal such as 80x24. The normal 120x40 experience keeps
// the side-by-side previews with representative terminal content visible.
func (m Model) compactAppearanceLayout() bool {
	if m.height <= 0 {
		return m.width > 0 && m.width < 100
	}
	return m.height <= 24 || (m.width > 0 && m.width < 100)
}

func (m Model) appearanceTotalWidth() int {
	if m.width <= 0 {
		return 0
	}
	return maxInt(40, m.width-8)
}

func (m Model) appearanceTextWidth() int {
	if m.width <= 0 {
		return 0
	}
	return maxInt(20, m.width-12)
}

func (m Model) appearanceCompareWidth() int {
	return m.appearanceTotalWidth()
}

func (m Model) themeListWidth(total int) int {
	if total <= 0 {
		return 30
	}
	return maxInt(24, minInt(32, total/4))
}

func truncateDisplay(value string, width int) string {
	if width <= 0 || ansi.StringWidth(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

func truncateLines(value string, width int) string {
	if width <= 0 {
		return value
	}
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = truncateDisplay(line, width)
	}
	return strings.Join(lines, "\n")
}

func renderThemeListPanel(rows []string, width int) string {
	if width <= 0 {
		return appearanceCardStyle.Render(strings.Join(rows, "\n"))
	}
	contentWidth := maxInt(12, width-appearanceCardStyle.GetHorizontalFrameSize())
	for index, row := range rows {
		rows[index] = truncateDisplay(row, contentWidth)
	}
	return appearanceCardStyle.Width(width).Render(strings.Join(rows, "\n"))
}

func renderCompactTerminalPreview(look appearanceLook, width int) string {
	background := normalizedPreviewColor(look.Colors.Background)
	foreground := normalizedPreviewColor(look.Colors.Foreground)
	if !look.Available || background == "" || foreground == "" {
		reason := look.Reason
		if reason == "" {
			reason = "Ghostty theme colors are unavailable"
		}
		return errorStyle.Render(truncateDisplay("Preview unavailable · "+reason, width))
	}
	base := ansi.NewStyle().ForegroundColor(lipgloss.Color(foreground)).BackgroundColor(lipgloss.Color(background))
	return strings.Join([]string{
		renderPreviewLine(base, width, "  ❯ ghostty +list-themes"),
		renderPreviewLine(base, width, renderPaletteRow(look.Colors.Palette, 0, width)),
	}, "\n")
}

func renderPreviewLine(style ansi.Style, width int, content string) string {
	content = truncateDisplay(content, width)
	content += strings.Repeat(" ", maxInt(0, width-ansi.StringWidth(content)))
	// Swatches and samples reset the terminal style after their own colors.
	// Restore the preview colors so subsequent text stays on the same canvas.
	return style.Styled(strings.ReplaceAll(content, ansi.ResetStyle, ansi.ResetStyle+style.String()))
}

// Use solid glyphs for swatches so each ANSI color remains visible even when
// a terminal renderer trims trailing spaces.
func renderPaletteRow(palette []string, start int, width int) string {
	if start >= len(palette) {
		return ""
	}
	colors := palette[start:minInt(start+8, len(palette))]
	chipWidth := maxInt(1, (width-len(colors)-1)/len(colors))
	blocks := make([]string, 0, len(colors))
	for _, value := range colors {
		text := strings.Repeat("█", chipWidth)
		if color := normalizedPreviewColor(value); color != "" {
			chip := lipgloss.Color(color)
			blocks = append(blocks, lipgloss.NewStyle().Foreground(chip).Background(chip).Render(text))
		} else {
			blocks = append(blocks, strings.Repeat("·", chipWidth))
		}
	}
	return "  " + strings.Join(blocks, " ")
}

func (m Model) themeRowLimit() int {
	if m.height <= 0 {
		return 10
	}
	return maxInt(4, minInt(12, m.height-14))
}

func displayThemeName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Ghostty default"
	}
	return name
}

func lookSourceLabel(look appearanceLook) string {
	if look.Available {
		if look.Fallback {
			return "Configured colors · theme file unavailable"
		}
		if look.Name == "" {
			return "Ghostty default configuration"
		}
		if look.Source == "" {
			return "Ghostty theme data"
		}
		return shortThemeSource(look.Source)
	}
	reason := shortThemeReason(look.Reason)
	if reason == "" {
		reason = "Ghostty theme colors unavailable"
	}
	return "Preview unavailable · " + reason
}

func shortThemeSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "Ghostty theme data"
	}
	if strings.EqualFold(source, "configured Ghostty color overrides") {
		return "Configured colors"
	}
	normalized := strings.ToLower(strings.ReplaceAll(source, "\\", "/"))
	if strings.Contains(normalized, "/.config/ghostty/themes/") || strings.Contains(normalized, "/application support/com.mitchellh.ghostty/themes/") || strings.Contains(normalized, "/library/application support/com.mitchellh.ghostty/themes/") {
		return "Configured theme file"
	}
	if strings.Contains(normalized, "/resources/ghostty/themes/") || strings.Contains(normalized, "/share/ghostty/themes/") || strings.Contains(normalized, "/usr/share/ghostty/themes/") || strings.Contains(normalized, "/usr/local/share/ghostty/themes/") {
		return "Ghostty bundled theme file"
	}
	if strings.Contains(normalized, "theme file") {
		return "Ghostty theme file"
	}
	return "Ghostty theme file"
}

func shortThemeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	lower := strings.ToLower(reason)
	if strings.Contains(lower, "not found") || strings.Contains(lower, "no such file") || strings.Contains(lower, "cannot locate") || strings.Contains(lower, "source missing") || strings.Contains(lower, "unavailable") {
		return "Theme data unavailable on this machine"
	}
	if strings.Contains(reason, "/") || strings.Contains(reason, "\\") {
		return "Theme data unavailable on this machine"
	}
	return reason
}

func renderThemeCompare(leftLabel string, left appearanceLook, rightLabel string, right appearanceLook, widths ...int) string {
	totalWidth := 0
	if len(widths) > 0 {
		totalWidth = widths[0]
	}
	if totalWidth <= 0 {
		leftPanel := renderLookPanel(leftLabel, left)
		rightPanel := renderLookPanel(rightLabel, right)
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	}
	panelWidth := maxInt(20, (totalWidth-2)/2)
	leftPanel := renderLookPanel(leftLabel, left, panelWidth)
	rightPanel := renderLookPanel(rightLabel, right, panelWidth)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}

func renderLookPanel(label string, look appearanceLook, widths ...int) string {
	panelWidth := 0
	if len(widths) > 0 {
		panelWidth = widths[0]
	}
	textWidth := panelWidth
	if panelWidth > 0 {
		textWidth = maxInt(12, panelWidth-appearanceCardStyle.GetHorizontalFrameSize())
	}
	preview := renderTerminalPreview(look, textWidth)
	if textWidth > 0 {
		preview = truncateLines(preview, textWidth)
	}
	tag := appearanceCandidateTag
	if strings.Contains(strings.ToUpper(label), "CURRENT") {
		tag = appearanceCurrentTag
	}
	rows := []string{
		tag.Render(truncateDisplay(label, maxInt(8, textWidth-2))),
		selectedStyle.Render(truncateDisplay(displayThemeName(look.Name), textWidth)),
		appearanceHintStyle.Render(truncateDisplay(lookSourceLabel(look), textWidth)),
		"",
		preview,
	}
	content := strings.Join(rows, "\n")
	if panelWidth > 0 {
		return appearanceCardStyle.Width(panelWidth).Render(content)
	}
	return appearanceCardStyle.Render(content)
}

func renderTerminalPreview(look appearanceLook, widths ...int) string {
	width := 0
	if len(widths) > 0 {
		width = widths[0]
	}
	background := normalizedPreviewColor(look.Colors.Background)
	foreground := normalizedPreviewColor(look.Colors.Foreground)
	if !look.Available || background == "" || foreground == "" {
		reason := look.Reason
		if reason == "" {
			reason = "Ghostty theme colors are unavailable"
		}
		return strings.Join([]string{
			errorStyle.Render("Preview unavailable"),
			mutedStyle.Render(reason),
			mutedStyle.Render("No colors are guessed here."),
		}, "\n")
	}

	base := ansi.NewStyle().ForegroundColor(lipgloss.Color(foreground)).BackgroundColor(lipgloss.Color(background))
	line := func(content string) string {
		return renderPreviewLine(base, width, content)
	}
	paletteToken := func(index int, content string) string {
		if index < 0 || index >= len(look.Colors.Palette) {
			return content
		}
		color := normalizedPreviewColor(look.Colors.Palette[index])
		if color == "" {
			return content
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(content)
	}
	lines := []string{
		line(""),
		line("  " + paletteToken(4, "❯") + " ghostty +list-themes"),
		line("  " + paletteToken(2, "✓ ready") + "  " + paletteToken(6, "themes") + "  " + paletteToken(3, "⌘ + ,")),
		line(""),
	}
	cursor := normalizedPreviewColor(look.Colors.Cursor)
	cursorText := normalizedPreviewColor(look.Colors.CursorText)
	if cursor != "" && cursorText != "" {
		cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(cursor)).Foreground(lipgloss.Color(cursorText))
		lines = append(lines, line("  "+cursorStyle.Render(" │ ")+" cursor"))
	} else if cursor != "" || cursorText != "" {
		lines = append(lines, line("Cursor preview partial · unknown or named color omitted"))
	}
	selectionBackground := normalizedPreviewColor(look.Colors.SelectionBackground)
	selectionForeground := normalizedPreviewColor(look.Colors.SelectionForeground)
	if selectionBackground != "" && selectionForeground != "" {
		selection := lipgloss.NewStyle().Background(lipgloss.Color(selectionBackground)).Foreground(lipgloss.Color(selectionForeground))
		lines = append(lines, line("  "+selection.Render(" highlighted text ")))
	} else if selectionBackground != "" || selectionForeground != "" {
		lines = append(lines, line("Selection preview partial · unknown or named color omitted"))
	}
	if len(look.Colors.Palette) > 0 {
		lines = append(lines, line(""))
		for start := 0; start < len(look.Colors.Palette); start += 8 {
			lines = append(lines, line(renderPaletteRow(look.Colors.Palette, start, maxInt(width, 24))))
		}
		if look.Partial {
			lines = append(lines, line("Palette preview partial · unknown or named colors omitted"))
		}
	}
	lines = append(lines, line(""))
	return strings.Join(lines, "\n")
}
