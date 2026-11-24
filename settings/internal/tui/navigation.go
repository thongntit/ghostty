package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/thongntit/ghostty/settings/internal/configdoc"
	"github.com/thongntit/ghostty/settings/internal/schema"
)

type browseLevel uint8

const (
	browseHome browseLevel = iota
	browseCategories
	browseOptions
	browseThemes
)

type browseScope uint8

const (
	browseScopeCategory browseScope = iota
	browseScopeConfigured
	browseScopeAll
	browseScopeSearch
)

type browseSection struct {
	Key         string
	Label       string
	Description string
	OptionCount int
	Configured  int
}

// EnableCategoryNavigation opts the application host into the category-first
// experience while keeping NewModel's flat behavior available to embedders
// and focused model tests that rely on the original API contract.
func (m *Model) EnableCategoryNavigation() {
	m.categoryNavigation = true
	m.appearanceHome = false
	m.categoryReturnHome = false
	m.browseLevel = browseCategories
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.searching = false
	m.search.SetValue("")
	m.visible = []int{}
	m.selected = 0
	m.categoryIndex = m.initialCategoryIndex()
}

// EnableAppearancePlayground makes the appearance playground the app's first
// screen while preserving the existing category-first editor behind it. The
// basic NewModel API intentionally remains unchanged for embedders and tests.
func (m *Model) EnableAppearancePlayground() {
	if !m.categoryNavigation {
		m.EnableCategoryNavigation()
	}
	m.appearanceHome = true
	m.appearanceHomeIndex = 0
	m.categoryReturnHome = false
	m.browseLevel = browseHome
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.visible = nil
	m.selected = 0
	m.themeSearching = false
	m.themeQuery.SetValue("")
	m.themeQuery.Blur()
	m.themeIndex = -1
	m.themeVisible = nil
}

func (m Model) initialCategoryIndex() int {
	sections := m.browseSections()
	if len(m.configuredKeys()) > 0 {
		for index, section := range sections {
			if section.Key == "configured" {
				return index
			}
		}
	}
	for index, section := range sections {
		if section.Key != "configured" && section.Key != "all" {
			return index
		}
	}
	return 0
}

func (m Model) browseSections() []browseSection {
	configured := m.configuredKeys()
	counts := make(map[string]int)
	configuredCounts := make(map[string]int)
	for _, option := range m.options {
		category := m.optionCategory(option)
		counts[category]++
		if _, ok := configured[option.Key]; ok {
			configuredCounts[category]++
		}
	}

	sections := []browseSection{
		{Key: "configured", Label: "Configured", Description: "Settings explicitly present in the current config graph.", OptionCount: len(configured), Configured: len(configured)},
		{Key: "all", Label: "All Settings", Description: "The complete Ghostty catalog; use this when you need an uncommon option.", OptionCount: len(m.options), Configured: len(configured)},
	}
	for _, section := range schema.NavigationSections() {
		if count := counts[section.Name]; count > 0 {
			sections = append(sections, browseSection{
				Key:         section.Name,
				Label:       section.Name,
				Description: section.Description,
				OptionCount: count,
				Configured:  configuredCounts[section.Name],
			})
		}
	}
	if count := counts["Custom"]; count > 0 {
		sections = append(sections, browseSection{
			Key:         "Custom",
			Label:       "Custom",
			Description: "Unknown or version-specific keys preserved from the loaded config.",
			OptionCount: count,
			Configured:  configuredCounts["Custom"],
		})
	}
	return sections
}

func (m Model) optionCategory(option schema.Option) string {
	return schema.NavigationCategory(option)
}

func (m Model) configuredKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	add := func(document configdoc.Document) {
		for _, node := range document.Nodes {
			if node.Kind == configdoc.AssignmentNode {
				keys[node.Key] = struct{}{}
			}
		}
	}
	if m.graph != nil {
		for _, document := range m.fileDrafts {
			add(document)
		}
		return keys
	}
	if m.draft != nil {
		add(*m.draft)
	}
	return keys
}

func (m Model) optionIndexesForScope(scope browseScope, category string) []int {
	configured := m.configuredKeys()
	indexes := make([]int, 0, len(m.options))
	for index, option := range m.options {
		include := false
		switch scope {
		case browseScopeConfigured:
			_, include = configured[option.Key]
		case browseScopeAll:
			include = true
		case browseScopeCategory:
			include = m.optionCategory(option) == category
		}
		if include {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (m *Model) setBrowseScope(scope browseScope, category string, preferredKey string) {
	m.browseLevel = browseOptions
	m.browseScope = scope
	m.activeCategory = category
	m.visible = m.optionIndexesForScope(scope, category)
	m.selected = 0
	if preferredKey != "" {
		m.selectVisibleKey(preferredKey)
	}
}

func (m *Model) openFullEditor() {
	if !m.categoryNavigation {
		return
	}
	m.categoryReturnHome = m.appearanceHome
	m.browseLevel = browseCategories
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.visible = nil
	m.selected = 0
	m.categoryIndex = m.initialCategoryIndex()
}

func (m *Model) returnToAppearanceHome() {
	if !m.appearanceHome {
		return
	}
	m.browseLevel = browseHome
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.visible = nil
	m.selected = 0
	m.categoryReturnHome = false
	m.themeSearching = false
	m.themeQuery.Blur()
}

func (m *Model) selectVisibleKey(keyValue string) {
	if keyValue == "" {
		return
	}
	for position, index := range m.visible {
		if m.options[index].Key == keyValue {
			m.selected = position
			return
		}
	}
}

func (m *Model) openSelectedSection() {
	sections := m.browseSections()
	if m.categoryIndex < 0 || m.categoryIndex >= len(sections) {
		return
	}
	section := sections[m.categoryIndex]
	switch section.Key {
	case "configured":
		m.setBrowseScope(browseScopeConfigured, "", "")
	case "all":
		m.setBrowseScope(browseScopeAll, "", "")
	default:
		m.setBrowseScope(browseScopeCategory, section.Key, "")
	}
}

func (m *Model) openConfigured() {
	m.setBrowseScope(browseScopeConfigured, "", m.SelectedKey())
}

func (m *Model) openAll() {
	m.setBrowseScope(browseScopeAll, "", m.SelectedKey())
}

func (m *Model) returnToCategories() {
	if !m.categoryNavigation {
		return
	}
	previousKey := m.activeCategory
	if previousKey == "" {
		switch m.browseScope {
		case browseScopeConfigured:
			previousKey = "configured"
		case browseScopeAll:
			previousKey = "all"
		}
	}
	m.browseLevel = browseCategories
	m.browseScope = browseScopeAll
	m.activeCategory = ""
	m.visible = []int{}
	m.selected = 0
	m.searching = false
	m.search.SetValue("")
	m.search.Blur()
	m.selectCategoryKey(previousKey)
}

func (m *Model) selectCategoryKey(keyValue string) {
	sections := m.browseSections()
	for index, section := range sections {
		if section.Key == keyValue {
			m.categoryIndex = index
			return
		}
	}
	m.categoryIndex = m.initialCategoryIndex()
}

func (m *Model) beginGlobalSearch() tea.Cmd {
	if m.browseLevel == browseOptions && m.browseScope == browseScopeSearch {
		m.searching = true
		m.search.CursorEnd()
		return m.search.Focus()
	}
	m.searchReturnLevel = m.browseLevel
	m.searchReturnScope = m.browseScope
	m.searchReturnIndex = m.categoryIndex
	m.searchReturnCategory = m.activeCategory
	m.searchReturnKey = m.SelectedKey()
	m.browseLevel = browseOptions
	m.browseScope = browseScopeSearch
	m.activeCategory = ""
	m.visible = []int{}
	m.selected = 0
	m.search.SetValue("")
	m.searching = true
	m.search.CursorEnd()
	return m.search.Focus()
}

func (m *Model) updateNavigationSearch(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.restoreSearchContext()
			return *m, nil
		case "enter":
			query := strings.TrimSpace(m.search.Value())
			m.search.Blur()
			m.searching = false
			if query == "" {
				m.restoreSearchContext()
				return *m, nil
			}
			m.applyNavigationSearch(query)
			return *m, nil
		}
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.applyNavigationSearch(m.search.Value())
	return *m, cmd
}

func (m *Model) restoreSearchContext() {
	m.search.Blur()
	m.searching = false
	m.search.SetValue("")
	m.browseLevel = m.searchReturnLevel
	m.browseScope = m.searchReturnScope
	m.categoryIndex = m.searchReturnIndex
	m.activeCategory = m.searchReturnCategory
	if m.browseLevel != browseOptions {
		m.visible = []int{}
		m.selected = 0
		return
	}
	m.visible = m.optionIndexesForScope(m.browseScope, m.activeCategory)
	m.selected = 0
	m.selectVisibleKey(m.searchReturnKey)
}

func (m *Model) applyNavigationSearch(query string) {
	previousKey := m.SelectedKey()
	m.visible = []int{}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		m.selected = 0
		return
	}
	tokens := strings.Fields(query)
	var matches [3][]int
	for index, option := range m.options {
		name := strings.ToLower(friendlyOptionName(option.Key))
		optionKey := strings.ToLower(option.Key)
		if name == query || optionKey == query {
			matches[0] = append(matches[0], index)
			continue
		}
		searchable := strings.ToLower(strings.Join([]string{
			m.optionCategory(option), option.Category, optionGroup(option), name, optionKey, option.Description,
		}, " "))
		matched := true
		nameMatched := true
		for _, token := range tokens {
			if !strings.Contains(searchable, token) {
				matched = false
				break
			}
			nameMatched = nameMatched && (strings.Contains(name, token) || strings.Contains(optionKey, token))
		}
		if matched && nameMatched {
			matches[1] = append(matches[1], index)
		} else if matched {
			matches[2] = append(matches[2], index)
		}
	}
	for _, group := range matches {
		m.visible = append(m.visible, group...)
	}
	m.selected = 0
	if !m.searching {
		m.selectVisibleKey(previousKey)
	}
}

func (m *Model) refreshBrowse(preferredKey string) {
	if !m.categoryNavigation || m.browseLevel != browseOptions {
		return
	}
	if m.browseScope == browseScopeSearch {
		m.applyNavigationSearch(m.search.Value())
		return
	}
	m.visible = m.optionIndexesForScope(m.browseScope, m.activeCategory)
	m.selected = 0
	m.selectVisibleKey(preferredKey)
}

func (m Model) updateCategoryBrowse(msg tea.Msg) (Model, tea.Cmd) {
	if m.searching {
		return m.updateNavigationSearch(msg)
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, defaultKeyMap.Quit):
		return m, m.requestQuit()
	case (keyMsg.String() == "esc" || keyMsg.String() == "h" || keyMsg.String() == "g") && m.categoryReturnHome:
		m.returnToAppearanceHome()
	case keyMsg.String() == "/":
		return m, m.beginGlobalSearch()
	case key.Matches(keyMsg, defaultKeyMap.Up):
		if m.categoryIndex > 0 {
			m.categoryIndex--
		}
	case key.Matches(keyMsg, defaultKeyMap.Down):
		if m.categoryIndex < len(m.browseSections())-1 {
			m.categoryIndex++
		}
	case key.Matches(keyMsg, defaultKeyMap.Enter), keyMsg.String() == "l":
		m.openSelectedSection()
	case keyMsg.String() == "c":
		m.openConfigured()
	case keyMsg.String() == "a":
		m.openAll()
	case key.Matches(keyMsg, defaultKeyMap.Preview):
		m.openPreview()
	case key.Matches(keyMsg, defaultKeyMap.Save):
		m.saveNow()
	case key.Matches(keyMsg, defaultKeyMap.Help):
		m.help = !m.help
	}
	return m, nil
}

func (m Model) updateOptionBrowse(msg tea.Msg) (Model, tea.Cmd) {
	if m.searching {
		return m.updateNavigationSearch(msg)
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, defaultKeyMap.Quit):
		return m, m.requestQuit()
	case keyMsg.String() == "/":
		return m, m.beginGlobalSearch()
	case keyMsg.String() == "esc":
		if m.browseScope == browseScopeSearch {
			m.restoreSearchContext()
		} else {
			m.returnToCategories()
		}
	case keyMsg.String() == "backspace" || keyMsg.String() == "h":
		m.returnToCategories()
	case keyMsg.String() == "c":
		m.openConfigured()
	case keyMsg.String() == "a":
		m.openAll()
	case keyMsg.String() == "g":
		m.returnToCategories()
	case key.Matches(keyMsg, defaultKeyMap.Up):
		if m.selected > 0 {
			m.selected--
		}
	case key.Matches(keyMsg, defaultKeyMap.Down):
		if m.selected < len(m.visibleOptionIndexes())-1 {
			m.selected++
		}
	case key.Matches(keyMsg, defaultKeyMap.Enter), key.Matches(keyMsg, defaultKeyMap.Edit):
		return m, m.beginEdit()
	case key.Matches(keyMsg, defaultKeyMap.Reset):
		m.resetSelected()
	case key.Matches(keyMsg, defaultKeyMap.Undo):
		m.revertSelected()
	case key.Matches(keyMsg, defaultKeyMap.Preview):
		m.openPreview()
	case key.Matches(keyMsg, defaultKeyMap.Save):
		m.saveNow()
	case key.Matches(keyMsg, defaultKeyMap.Help):
		m.help = !m.help
	}
	return m, nil
}

func (m Model) categoryBrowseDescription() string {
	sections := m.browseSections()
	if m.categoryIndex < 0 || m.categoryIndex >= len(sections) {
		return ""
	}
	return sections[m.categoryIndex].Description
}

func (m Model) categoryBrowseTitle() string {
	sections := m.browseSections()
	if m.categoryIndex < 0 || m.categoryIndex >= len(sections) {
		return "Categories"
	}
	return fmt.Sprintf("Categories · %s", sections[m.categoryIndex].Label)
}

func (m Model) renderCategories() string {
	sections := m.browseSections()
	rows := []string{accentStyle.Render("Categories")}
	if len(sections) == 0 {
		rows = append(rows, mutedStyle.Render("No settings available"))
		return panelStyle.Render(strings.Join(rows, "\n"))
	}
	limit := m.categoryRowLimit()
	start := maxInt(0, m.categoryIndex-limit/2)
	if start+limit > len(sections) {
		start = maxInt(0, len(sections)-limit)
	}
	end := minInt(len(sections), start+limit)
	if start > 0 {
		rows = append(rows, mutedStyle.Render("↑ more"))
	}
	for index := start; index < end; index++ {
		section := sections[index]
		marker := "  "
		if index == m.categoryIndex {
			marker = selectedStyle.Render("▸ ")
		}
		summary := fmt.Sprintf("%d configured · %d options", section.Configured, section.OptionCount)
		rows = append(rows, marker+fmt.Sprintf("%-16s %s", section.Label, mutedStyle.Render(summary)))
	}
	if end < len(sections) {
		rows = append(rows, mutedStyle.Render("↓ more"))
	}
	if m.categoryIndex >= 0 && m.categoryIndex < len(sections) {
		rows = append(rows, "", mutedStyle.Render(sections[m.categoryIndex].Description))
	}
	return panelStyle.Render(strings.Join(rows, "\n"))
}

func scopeLabel(scope browseScope) string {
	switch scope {
	case browseScopeConfigured:
		return "Configured"
	case browseScopeAll:
		return "All Settings"
	case browseScopeSearch:
		return "Search results"
	default:
		return "Category"
	}
}
