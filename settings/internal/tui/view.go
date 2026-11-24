package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/thongntit/ghostty/settings/internal/schema"
)

func (m Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	return view
}

func (m Model) render() string {
	session := "Guided editor · preview before save"
	if m.saveFunc != nil {
		session = "Guided editor · saves are confirmed explicitly"
	}
	if m.appearanceHome {
		session = "Appearance playground · explore safely, save explicitly"
	}
	header := titleStyle.Render("Ghostty Config TUI") + "\n" + mutedStyle.Render(session)
	searchBar := m.searchBar()
	if m.appearanceHome && m.mode == ModeBrowse {
		header = m.renderAppearanceHeader()
		if m.browseLevel == browseHome || m.browseLevel == browseThemes {
			searchBar = ""
		}
	}

	var body string
	switch m.mode {
	case ModeEdit:
		body = m.renderEdit()
	case ModePreview:
		body = m.renderPreview()
	case ModeConfirmQuit:
		body = m.renderConfirmQuit()
	default:
		if m.categoryNavigation && m.browseLevel == browseHome {
			body = m.renderAppearanceHome()
		} else if m.categoryNavigation && m.browseLevel == browseThemes {
			body = m.renderThemeBrowse()
		} else if m.categoryNavigation && m.browseLevel == browseCategories {
			body = m.renderCategories()
		} else if m.categoryNavigation && m.browseScope == browseScopeSearch {
			body = m.renderOptions()
		} else {
			body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderOptions(), m.renderDetails())
		}
	}

	config := m.configPath
	if config == "" {
		config = "(unknown)"
	}
	if m.appearanceHome && m.width > 8 && (m.browseLevel != browseOptions || m.browseScope == browseScopeSearch) {
		config = truncateDisplay(config, m.width-8)
	}
	footer := m.footer()
	if m.width > 0 {
		footer = truncateDisplay(footer, m.width)
	}
	return strings.Join([]string{
		header,
		searchBar,
		"",
		body,
		"",
		accentStyle.Render("Config: ") + config,
		mutedStyle.Render(m.fileSummary()),
		footer,
	}, "\n")
}

func (m Model) renderOptions() string {
	title := "Options"
	if m.categoryNavigation {
		switch m.browseScope {
		case browseScopeConfigured:
			title += " · Configured"
		case browseScopeAll:
			title += " · All Settings"
		case browseScopeSearch:
			title += " · Search results"
		case browseScopeCategory:
			if m.activeCategory != "" {
				title += " · " + m.activeCategory
			}
		}
	}
	rows := []string{accentStyle.Render(title)}
	visible := m.visibleOptionIndexes()
	if len(visible) == 0 {
		message := "No options match the search"
		if m.categoryNavigation && m.browseScope == browseScopeSearch && strings.TrimSpace(m.search.Value()) == "" {
			message = "Type to search all settings"
		} else if m.categoryNavigation && m.browseScope == browseScopeConfigured {
			message = "No settings are configured yet"
		}
		rows = append(rows, mutedStyle.Render(message))
		return panelStyle.Render(strings.Join(rows, "\n"))
	}

	limit := m.optionRowLimit()
	start := maxInt(0, m.selected-limit/2)
	if start+limit > len(visible) {
		start = maxInt(0, len(visible)-limit)
	}
	end := minInt(len(visible), start+limit)
	if start > 0 {
		rows = append(rows, mutedStyle.Render("↑ more"))
	}
	for position := start; position < end; position++ {
		index := visible[position]
		option := m.options[index]
		if m.categoryNavigation && m.browseScope == browseScopeCategory {
			group := optionGroup(option)
			if position == start || optionGroup(m.options[visible[position-1]]) != group {
				rows = append(rows, mutedStyle.Render("— "+group+" —"))
			}
		}
		marker := "  "
		if position == m.selected {
			marker = selectedStyle.Render("▸ ")
		}
		state := m.optionStatus(option)
		rows = append(rows, marker+fmt.Sprintf("%-24s %s", friendlyOptionName(option.Key), mutedStyle.Render(state)))
	}
	if end < len(visible) {
		rows = append(rows, mutedStyle.Render("↓ more"))
	}
	content := strings.Join(rows, "\n")
	if m.categoryNavigation && m.browseScope == browseScopeSearch && m.width > panelStyle.GetHorizontalFrameSize() {
		content = truncateLines(content, m.width-panelStyle.GetHorizontalFrameSize())
	}
	return panelStyle.Render(content)
}

func (m Model) renderDetails() string {
	option, ok := m.selectedOption()
	if !ok {
		return panelStyle.Render("No schema options loaded")
	}

	rows := []string{
		accentStyle.Render(m.optionCategory(option)),
		selectedStyle.Render(friendlyOptionName(option.Key)),
		mutedStyle.Render(option.Key),
		mutedStyle.Render("Group: " + optionGroup(option)),
		"",
		option.Description,
		"",
		"Config value: " + m.optionStatus(option),
		mutedStyle.Render("Editor: " + m.editorLabel(option)),
	}
	if option.Default != "" {
		rows = append(rows, mutedStyle.Render("Ghostty default: "+option.Default))
	}
	if len(option.Availability) > 0 {
		rows = append(rows, mutedStyle.Render("Available on: "+strings.Join(option.Availability, ", ")))
	}
	if option.Docs != "" {
		rows = append(rows, mutedStyle.Render("Docs: "+option.Docs))
	}
	if source := m.optionSource(option); source != "" {
		rows = append(rows, mutedStyle.Render(source))
	}
	if m.graph != nil {
		draftGraph := m.draftGraph()
		rows = append(rows, mutedStyle.Render(fmt.Sprintf("Config graph: %d files · %d effective assignments", len(draftGraph.Files), len(draftGraph.Assignments))))
		if len(m.graph.Diagnostics) > 0 {
			rows = append(rows, errorStyle.Render(fmt.Sprintf("Diagnostics: %d (press p for details)", len(m.graph.Diagnostics))))
		}
	}
	if m.graph != nil {
		if len(m.optionAssignments(option.Key)) > 1 {
			rows = append(rows, mutedStyle.Render("Multiple assignments; editor targets the effective source."))
		}
	} else {
		if len(m.optionAssignments(option.Key)) > 1 {
			rows = append(rows, mutedStyle.Render("Multiple assignments; editor targets the last one."))
		}
	}
	if m.status != "" {
		rows = append(rows, "", accentStyle.Render(m.status))
	}
	return panelStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderEdit() string {
	option, _ := m.selectedOption()
	if m.repeatableMode {
		return m.renderRepeatableEdit(option)
	}
	if m.toggleMode {
		return m.renderToggleEdit(option)
	}
	if m.multiMode {
		return m.renderMultiEdit(option)
	}
	if m.metricMode {
		return m.renderMetricEdit(option)
	}
	if m.resourceMode {
		return m.renderResourceEdit(option)
	}
	if m.scalarPairMode {
		return m.renderScalarPairEdit(option)
	}
	if m.durationMode {
		return m.renderDurationEdit(option)
	}
	if m.pathMode {
		return m.renderPathEdit(option)
	}
	if m.choiceMode != choiceNone {
		return m.renderChoiceEdit(option)
	}
	rows := []string{
		accentStyle.Render("Edit value"),
		selectedStyle.Render(friendlyOptionName(option.Key)),
		mutedStyle.Render(option.Key),
		mutedStyle.Render(option.Description),
		mutedStyle.Render("Expected: " + valueHint(option)),
		"",
		m.input.View(),
	}
	if m.durationRawMode {
		rows = append(rows, mutedStyle.Render("Compound duration · kept as raw text to preserve all components"))
	}
	if m.numberMode {
		rows = append(rows, mutedStyle.Render("←/→ adjust by "+numberStepLabel(option)+" · ctrl+r raw numeric input"))
	}
	if m.editError != nil {
		rows = append(rows, "", errorStyle.Render(m.editError.Error()))
	}
	shortcut := "enter stage · esc cancel"
	if len(option.Values) > 0 || option.Kind == schema.KindBoolean {
		shortcut = "←/→ choose · enter stage · esc cancel"
	}
	rows = append(rows, "", mutedStyle.Render(shortcut))
	return panelStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderRepeatableEdit(option schema.Option) string {
	if m.choiceMode != choiceNone {
		return m.renderChoiceEdit(option)
	}
	if m.pathMode {
		return m.renderPathEdit(option)
	}
	if m.repeatableForm != repeatableFormNone {
		return m.renderRepeatableForm(option)
	}
	rows := []string{
		accentStyle.Render("Edit " + friendlyOptionName(option.Key)),
		mutedStyle.Render(option.Key),
		mutedStyle.Render(option.Description),
		"",
	}
	if m.repeatableEditing {
		rows = append(rows,
			selectedStyle.Render("Value"),
			m.input.View(),
			mutedStyle.Render("Enter a complete one-line Ghostty value."),
		)
		if m.editError != nil {
			rows = append(rows, errorStyle.Render(m.editError.Error()))
		}
		rows = append(rows, mutedStyle.Render("enter update · esc back to list"))
		return panelStyle.Render(strings.Join(rows, "\n"))
	}
	rows = append(rows, mutedStyle.Render(fmt.Sprintf("%d value(s)", len(m.repeatableItems))))
	if len(m.repeatableItems) == 0 {
		rows = append(rows, mutedStyle.Render("No values. Press a to add one."))
	} else {
		limit := m.repeatableRowLimit()
		start := maxInt(0, m.repeatableIndex-limit/2)
		if start+limit > len(m.repeatableItems) {
			start = maxInt(0, len(m.repeatableItems)-limit)
		}
		end := minInt(len(m.repeatableItems), start+limit)
		if start > 0 {
			rows = append(rows, mutedStyle.Render("↑ more"))
		}
		for index := start; index < end; index++ {
			item := m.repeatableItems[index]
			marker := "  "
			if index == m.repeatableIndex {
				marker = selectedStyle.Render("▸ ")
			}
			location := "new"
			if item.Existing {
				location = fmt.Sprintf("%s:%d", filepath.Base(item.Path), item.Line)
			}
			rows = append(rows, marker+fmt.Sprintf("%-30s %s", item.Value, mutedStyle.Render(location)))
		}
		if end < len(m.repeatableItems) {
			rows = append(rows, mutedStyle.Render("↓ more"))
		}
	}
	return panelStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) renderChoiceEdit(option schema.Option) string {
	title := "Choose " + friendlyOptionName(option.Key)
	switch m.choiceMode {
	case choiceFont:
		title = "Choose font family"
	case choiceAction:
		title = "Choose action"
	case choiceEnum:
		title = "Choose " + friendlyOptionName(option.Key)
	}
	rows := []string{
		accentStyle.Render(title),
		mutedStyle.Render(option.Key),
		mutedStyle.Render(option.Description),
		"",
		"Current: " + m.input.Value(),
	}
	if m.choiceMode == choiceColor {
		rows[len(rows)-1] += "  " + colorSwatch(m.input.Value())
	}
	rows = append(rows, "", m.choiceQuery.View(), "")
	if len(m.choiceMatch) == 0 {
		rows = append(rows, mutedStyle.Render("No matching values"))
	} else {
		limit := m.choiceRowLimit()
		start := maxInt(0, m.choiceIndex-limit/2)
		if start+limit > len(m.choiceMatch) {
			start = maxInt(0, len(m.choiceMatch)-limit)
		}
		end := minInt(len(m.choiceMatch), start+limit)
		if start > 0 {
			rows = append(rows, mutedStyle.Render("↑ more"))
		}
		for position := start; position < end; position++ {
			index := m.choiceMatch[position]
			marker := "  "
			if position == m.choiceIndex {
				marker = selectedStyle.Render("▸ ")
			}
			name, value := m.choiceAt(index)
			label := name
			if m.choiceMode == choiceColor {
				label = fmt.Sprintf("%-24s %s %s", name, value, colorSwatch(value))
			}
			rows = append(rows, marker+label)
		}
		if end < len(m.choiceMatch) {
			rows = append(rows, mutedStyle.Render("↓ more"))
		}
	}
	return panelStyle.Render(strings.Join(rows, "\n"))
}

func colorSwatch(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 6 {
		value = "#" + value
	}
	if len(value) != 7 || value[0] != '#' {
		return mutedStyle.Render("[no swatch]")
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(value)).Render("  ")
}

func (m Model) renderPreview() string {
	header := strings.Join([]string{
		accentStyle.Render("Preview"),
		selectedStyle.Render("DRY RUN — no file will be written"),
	}, "\n")
	return panelStyle.Render(header + "\n\n" + m.preview.View())
}

func (m Model) renderConfirmQuit() string {
	message := "Save staged changes before quitting?\n\n" + selectedStyle.Render("y") + " save and quit    " + selectedStyle.Render("d") + " discard and quit    " + mutedStyle.Render("esc back")
	return panelStyle.Render(message)
}

func (m Model) footer() string {
	if m.help {
		if m.categoryNavigation && m.mode == ModeBrowse {
			if m.browseLevel == browseHome {
				return mutedStyle.Render("/ or ctrl+f search all settings · t themes · g categories · ctrl+z undo last save · q quit · ? close help")
			}
			if m.browseLevel == browseThemes {
				return mutedStyle.Render("/ search themes · ctrl+f search all settings · enter stage · ctrl+z undo last save · esc back · ? close help")
			}
			if m.browseLevel == browseCategories {
				return mutedStyle.Render("↑/k ↓/j move · enter/l open · / search · c configured · a all · p preview · ctrl+z undo last save · q quit · ? close help")
			}
			return mutedStyle.Render("↑/k ↓/j move · enter/e edit · esc/h categories · c configured · a all · g categories · r reset · u revert · p preview · ctrl+s save · ctrl+z undo save · q quit")
		}
		return mutedStyle.Render("↑/k ↓/j move · enter/e edit · r reset · u revert · p preview · ctrl+s save · ctrl+z undo save · ? close help · q quit")
	}
	switch m.mode {
	case ModeEdit:
		if m.repeatableMode {
			if m.repeatableForm != repeatableFormNone {
				return mutedStyle.Render("tab/enter next field · ctrl+r raw value · esc back to list")
			}
			if m.pathMode {
				return mutedStyle.Render("↑/↓ browse · enter open/select · s select folder · ctrl+r raw path · esc back")
			}
			if m.choiceMode != choiceNone {
				return mutedStyle.Render("↑/↓ choose · type to filter · enter select · ctrl+r raw · esc back to list")
			}
			if m.repeatableEditing {
				return mutedStyle.Render("enter update · esc back to list")
			}
			return mutedStyle.Render("↑/k ↓/j select · a add · e edit · d delete · r clear · enter stage · esc cancel")
		}
		if m.choiceMode != choiceNone {
			return mutedStyle.Render("↑/↓ choose · type to filter · enter select · ctrl+r raw · esc cancel")
		}
		if m.toggleMode {
			return mutedStyle.Render("space/←/→ cycle default/off/on · enter stage · ctrl+r raw · esc cancel")
		}
		if m.multiMode {
			return mutedStyle.Render("↑/↓ select item · ←/→/space cycle · ctrl+r raw list · enter stage · esc cancel")
		}
		if m.metricMode {
			return mutedStyle.Render("←/→ unit · ↑/↓ amount · ctrl+r raw metric · enter stage · esc cancel")
		}
		if m.resourceMode {
			return mutedStyle.Render("←/→/space mode · ↑/↓ value · ctrl+r raw limit · enter stage · esc cancel")
		}
		if m.scalarPairMode {
			return mutedStyle.Render("tab/enter next field · ctrl+r raw pair · esc cancel")
		}
		if m.durationMode {
			return mutedStyle.Render("←/→ unit · ↑/↓ amount · ctrl+r raw duration · enter stage · esc cancel")
		}
		if m.pathMode {
			return mutedStyle.Render("↑/↓ browse · enter open/select · s select folder · ctrl+r raw path · esc cancel")
		}
		if m.numberMode {
			option, ok := m.selectedOption()
			if ok {
				return mutedStyle.Render("←/→ adjust by " + numberStepLabel(option) + " · ctrl+r raw input · enter stage · esc cancel")
			}
			return mutedStyle.Render("←/→ adjust · ctrl+r raw input · enter stage · esc cancel")
		}
		option, ok := m.selectedOption()
		if ok && (len(option.Values) > 0 || option.Kind == schema.KindBoolean) {
			return mutedStyle.Render("←/→ choose · enter stage · esc cancel")
		}
		return mutedStyle.Render("enter stage · esc cancel")
	case ModePreview:
		return mutedStyle.Render("↑/k ↓/j scroll · pgup/pgdn page · ctrl+z undo last save · esc back · q quit")
	case ModeConfirmQuit:
		return mutedStyle.Render("y save and quit · d discard and quit · esc back")
	default:
		if m.categoryNavigation {
			if m.browseLevel == browseHome {
				return mutedStyle.Render("/ search all settings · ↑/↓ move · enter select · ? help · q quit")
			}
			if m.browseLevel == browseThemes {
				return mutedStyle.Render("/ search themes · ctrl+f search all settings · ↑/↓ preview · enter stage · esc back")
			}
			if m.browseLevel == browseCategories {
				return mutedStyle.Render("↑/k ↓/j move · enter/l open · / search · c configured · a all · p preview · ? help · q quit")
			}
			if m.browseScope == browseScopeSearch {
				if m.searching {
					return mutedStyle.Render("Type to filter · enter focus results · esc back")
				}
				return mutedStyle.Render("↑/↓ select · enter edit · / refine search · esc back")
			}
			return mutedStyle.Render("↑/k ↓/j move · / search · enter/e edit · esc/h categories · c configured · a all · g categories · r reset · u revert · p preview · ctrl+s save · ctrl+z undo save · ? help · q quit")
		}
		return mutedStyle.Render("↑/k ↓/j move · / search · enter/e edit · r reset · u revert · p preview · ctrl+s save · ctrl+z undo save · ? help · q quit")
	}
}

func (m Model) searchBar() string {
	if m.mode != ModeBrowse {
		return ""
	}
	if m.searching {
		return m.search.View()
	}
	if m.categoryNavigation {
		if m.browseLevel == browseHome {
			return mutedStyle.Render("Appearance playground · enter select · t browse themes · g full editor")
		}
		if m.browseLevel == browseThemes {
			if m.themeSearching {
				return m.themeQuery.View()
			}
			if query := strings.TrimSpace(m.themeQuery.Value()); query != "" {
				return mutedStyle.Render("Theme search: " + query + " · / edit · esc back")
			}
			return mutedStyle.Render("Themes · ↑/↓/←/→ preview · enter stage · / search · esc back")
		}
		if m.browseLevel == browseCategories {
			return mutedStyle.Render("/ search all settings · enter/l open · c configured · a all")
		}
		if m.browseScope == browseScopeSearch && strings.TrimSpace(m.search.Value()) != "" {
			return mutedStyle.Render("Search: " + strings.TrimSpace(m.search.Value()) + " · esc return")
		}
		if m.activeCategory != "" {
			return mutedStyle.Render("Categories › " + m.activeCategory + " · / search all settings · esc categories")
		}
		return mutedStyle.Render("Categories › " + scopeLabel(m.browseScope) + " · / search all settings · esc categories")
	}
	return mutedStyle.Render("/ search options")
}

func (m Model) optionRowLimit() int {
	if m.height <= 0 {
		return 12
	}
	return maxInt(5, m.height-14)
}

func (m Model) categoryRowLimit() int {
	if m.height <= 0 {
		return 12
	}
	return maxInt(5, minInt(12, m.height-14))
}

func (m Model) choiceRowLimit() int {
	if m.height <= 0 {
		return 6
	}
	return maxInt(3, minInt(8, m.height-21))
}

func (m Model) repeatableRowLimit() int {
	if m.height <= 0 {
		return 8
	}
	return maxInt(3, minInt(10, m.height-13))
}

func (m Model) fileSummary() string {
	if m.saveFunc == nil {
		return "Preview only; changes stay in memory until this host provides saving"
	}
	if m.HasChanges() {
		return fmt.Sprintf("%d file(s) changed in memory; ctrl+s or y confirms the write", len(m.changedFileSnapshots()))
	}
	return "No staged changes"
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
