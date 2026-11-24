package tui

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/thongntit/ghostty/settings/internal/configdoc"
	"github.com/thongntit/ghostty/settings/internal/configgraph"
	"github.com/thongntit/ghostty/settings/internal/schema"
)

func TestEditPreviewKeepsOriginalUntouched(t *testing.T) {
	original := configdoc.Parse([]byte("theme = old\nfont-size = 14\n"))
	model := NewModel(schema.BootstrapOptions(), "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	if model.Mode() != ModeEdit {
		t.Fatalf("expected edit mode, got %v", model.Mode())
	}
	model.input.SetValue("rose-pine")
	model, _ = model.Update(special(tea.KeyEnter))
	if model.Mode() != ModeBrowse || !model.HasChanges() {
		t.Fatalf("expected staged browse state, mode=%v changes=%v", model.Mode(), model.HasChanges())
	}

	model, _ = model.Update(printable("p"))
	if model.Mode() != ModePreview {
		t.Fatalf("expected preview mode, got %v", model.Mode())
	}
	if !strings.Contains(model.preview.GetContent(), "DRY RUN — no file will be written") {
		t.Fatal("preview did not identify dry-run mode")
	}
	if !strings.Contains(model.preview.GetContent(), "theme = rose-pine") {
		t.Fatal("preview did not contain the staged value")
	}

	if got, want := model.OriginalBytes(), original.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("original document changed:\n got %q\nwant %q", got, want)
	}
	if got, want := string(model.DraftBytes()), "theme = rose-pine\nfont-size = 14\n"; got != want {
		t.Fatalf("unexpected draft:\n got %q\nwant %q", got, want)
	}
}

func TestResetRevertAndRepeatableEditor(t *testing.T) {
	original := configdoc.Parse([]byte("theme = old\nfont-size = 14\nkeybind = ctrl+a=first\n"))
	model := NewModel(schema.BootstrapOptions(), "fixture.ghostty", original)

	model, _ = model.Update(printable("r"))
	if got := string(model.DraftBytes()); !strings.Contains(got, "theme =\n") {
		t.Fatalf("reset did not stage an explicit default: %q", got)
	}
	model, _ = model.Update(printable("u"))
	if got := string(model.DraftBytes()); got != string(original.Bytes()) {
		t.Fatalf("revert did not restore original draft:\n got %q\nwant %q", got, original.Bytes())
	}

	for index := 0; index < 5; index++ {
		model, _ = model.Update(special(tea.KeyDown))
	}
	if model.SelectedKey() != "keybind" {
		t.Fatalf("expected keybind selection, got %q", model.SelectedKey())
	}
	model, _ = model.Update(printable("e"))
	if model.Mode() != ModeEdit {
		t.Fatalf("repeatable option did not open an editor: mode=%v status=%q", model.Mode(), model.Status())
	}
	model, _ = model.Update(special(tea.KeyEscape))
	if model.Mode() != ModeBrowse {
		t.Fatalf("repeatable editor did not cancel: mode=%v", model.Mode())
	}
}

func TestInvalidValueStaysInEditAndDirtyQuitConfirms(t *testing.T) {
	original := configdoc.Parse([]byte("theme = old\nfont-size = 14\n"))
	model := NewModel(schema.BootstrapOptions(), "fixture.ghostty", original)
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(printable("e"))
	model.input.SetValue("NaN")
	model, cmd := model.Update(special(tea.KeyEnter))
	if cmd != nil || model.Mode() != ModeEdit || model.EditError() == nil {
		t.Fatalf("invalid value did not stay in edit: mode=%v err=%v cmd=%v", model.Mode(), model.EditError(), cmd)
	}

	model.input.SetValue("16")
	model, _ = model.Update(special(tea.KeyEnter))
	model, cmd = model.Update(printable("q"))
	if cmd != nil || model.Mode() != ModeConfirmQuit {
		t.Fatalf("dirty quit skipped confirmation: mode=%v cmd=%v", model.Mode(), cmd)
	}
	model, _ = model.Update(printable("n"))
	if model.Mode() != ModeBrowse {
		t.Fatalf("cancelled quit did not return to browse: %v", model.Mode())
	}
	model, _ = model.Update(printable("q"))
	model, cmd = model.Update(printable("y"))
	if cmd == nil {
		t.Fatal("confirmed quit did not return a quit command")
	}
}

func TestBooleanEditorCyclesAndStagesTypedValue(t *testing.T) {
	original := configdoc.Parse([]byte("confirm-close-surface = false\n"))
	options := []schema.Option{{
		Key:         "confirm-close-surface",
		Category:    "General",
		Kind:        schema.KindBoolean,
		Edit:        schema.EditScalar,
		Description: "Confirm before closing.",
	}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	if model.Mode() != ModeEdit {
		t.Fatalf("expected edit mode, got %v", model.Mode())
	}
	model, _ = model.Update(special(tea.KeyRight))
	if got := model.input.Value(); got != "true" {
		t.Fatalf("boolean toggle value = %q, want true", got)
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "confirm-close-surface = true\n" {
		t.Fatalf("unexpected boolean draft: %q", got)
	}
}

func TestBooleanEditorShowsDefaultStateWithoutForcingText(t *testing.T) {
	options := []schema.Option{{
		Key:   "confirm-close-surface",
		Kind:  schema.KindBoolean,
		Edit:  schema.EditScalar,
		Group: "General",
	}}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse(nil))
	model, _ = model.Update(printable("e"))
	if !model.toggleMode || model.input.Focused() {
		t.Fatalf("boolean editor should be a focused toggle, mode=%v focused=%v", model.toggleMode, model.input.Focused())
	}
	if !strings.Contains(model.render(), "inherit default") {
		t.Fatalf("toggle view did not explain the default state:\n%s", model.render())
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if model.HasChanges() {
		t.Fatal("accepting an untouched default toggle created a change")
	}
}

func TestMetricEditorUsesUnitAndAmountControls(t *testing.T) {
	original := configdoc.Parse([]byte("adjust-font-baseline = 1px\n"))
	options := []schema.Option{{Key: "adjust-font-baseline", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model, _ = model.Update(printable("e"))
	if !model.metricMode {
		t.Fatal("metric editor did not open")
	}
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyUp))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "adjust-font-baseline = 2%\n" {
		t.Fatalf("metric draft = %q", got)
	}
}

func TestResourceEditorSeparatesDefaultUnlimitedAndExactValue(t *testing.T) {
	original := configdoc.Parse([]byte("scrollback-limit-lines = unlimited\n"))
	options := []schema.Option{{Key: "scrollback-limit-lines", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model, _ = model.Update(printable("e"))
	if !model.resourceMode || model.resourceChoice != 1 {
		t.Fatalf("resource editor state = mode %v choice %d", model.resourceMode, model.resourceChoice)
	}
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyUp))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "scrollback-limit-lines = 1\n" {
		t.Fatalf("resource draft = %q", got)
	}
}

func TestScalarPairEditorHandlesStructuredMouseMultiplier(t *testing.T) {
	original := configdoc.Parse([]byte("mouse-scroll-multiplier = precision:1,discrete:3\n"))
	options := []schema.Option{{Key: "mouse-scroll-multiplier", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model, _ = model.Update(printable("e"))
	if !model.scalarPairMode || model.formLeft != "1" || model.formRight != "3" {
		t.Fatalf("pair editor state = mode %v left %q right %q", model.scalarPairMode, model.formLeft, model.formRight)
	}
	model.input.SetValue("2")
	model, _ = model.Update(special(tea.KeyEnter))
	model.input.SetValue("4")
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "mouse-scroll-multiplier = precision:2,discrete:4\n" {
		t.Fatalf("pair draft = %q", got)
	}
}

func TestMultiSelectPreservesUnknownTokens(t *testing.T) {
	original := configdoc.Parse([]byte("shell-integration-features = cursor,no-future-token\n"))
	options := []schema.Option{{Key: "shell-integration-features", Kind: schema.KindString, Multiple: true, Values: []string{"cursor", "title"}, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "shell-integration-features = no-cursor,no-future-token\n" {
		t.Fatalf("unknown token was lost: %q", got)
	}
}

func TestFriendlyThemeChooserStagesSelectionWithoutRawSyntax(t *testing.T) {
	original := configdoc.Parse([]byte("theme = old\n"))
	options := []schema.Option{{
		Key:         "theme",
		Category:    "Appearance",
		Kind:        schema.KindString,
		Edit:        schema.EditScalar,
		Description: "A theme to use.",
	}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetThemeChoices([]string{"Rose Pine", "Tokyo Night"})

	model, _ = model.Update(printable("e"))
	if model.choiceMode != choiceTheme {
		t.Fatalf("expected theme chooser, got %v", model.choiceMode)
	}
	model, _ = model.Update(printable("tokyo"))
	model, _ = model.Update(special(tea.KeyEnter))
	if model.Mode() != ModeBrowse {
		t.Fatalf("theme chooser did not return to browse: %v", model.Mode())
	}
	if got := string(model.DraftBytes()); got != "theme = Tokyo Night\n" {
		t.Fatalf("unexpected chosen theme draft: %q", got)
	}
	if got := model.optionStatus(options[0]); got != "Tokyo Night" {
		t.Fatalf("staged theme status = %q, want Tokyo Night", got)
	}
}

func TestFriendlyColorChooserStagesSwatchValue(t *testing.T) {
	original := configdoc.Parse([]byte("background = 282c34\n"))
	options := []schema.Option{{
		Key:         "background",
		Category:    "Appearance",
		Kind:        schema.KindColor,
		Edit:        schema.EditScalar,
		Description: "Background color.",
	}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetColorChoices([]ColorChoice{{Name: "Ocean", Value: "#123456"}})

	model, _ = model.Update(printable("e"))
	if model.choiceMode != choiceColor {
		t.Fatalf("expected color chooser, got %v", model.choiceMode)
	}
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if lines := len(strings.Split(model.render(), "\n")); lines > 24 {
		t.Fatalf("color chooser rendered %d lines on an 80x24 terminal", lines)
	}
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "background = #123456\n" {
		t.Fatalf("unexpected chosen color draft: %q", got)
	}
}

func TestFontSizeStepperStagesHalfPoint(t *testing.T) {
	original := configdoc.Parse([]byte("font-size = 13\n"))
	step := 0.5
	options := []schema.Option{{
		Key:         "font-size",
		Category:    "Appearance",
		Kind:        schema.KindNumber,
		Edit:        schema.EditScalar,
		Step:        &step,
		Description: "Font size in points.",
	}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "font-size = 13.5\n" {
		t.Fatalf("unexpected stepped font size draft: %q", got)
	}
	model, _ = model.Update(printable("p"))
	if !strings.Contains(model.preview.GetContent(), "font-size = 13.5") {
		t.Fatalf("numeric preview did not include staged value: %q", model.preview.GetContent())
	}
}

func TestOpacityStepperStagesExactDecimalValue(t *testing.T) {
	original := configdoc.Parse([]byte("background-opacity = 0.95\n"))
	min := 0.0
	max := 1.0
	step := 0.05
	options := []schema.Option{{
		Key:         "background-opacity",
		Category:    "Colors",
		Kind:        schema.KindNumber,
		Edit:        schema.EditScalar,
		Min:         &min,
		Max:         &max,
		Step:        &step,
		Description: "Background opacity.",
	}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyLeft))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "background-opacity = 0.9\n" {
		t.Fatalf("unexpected stepped opacity draft: %q", got)
	}
}

func TestFriendlyEditorDoesNotDirtyOnNoopSelection(t *testing.T) {
	original := configdoc.Parse([]byte("background = #123456\n"))
	options := []schema.Option{{Key: "background", Kind: schema.KindColor, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetColorChoices([]ColorChoice{{Name: "Ocean", Value: "#123456"}})

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyEnter))
	if model.HasChanges() {
		t.Fatal("reselecting the current color created a staged change")
	}
	if got := model.Status(); got != "No change for background" {
		t.Fatalf("noop status = %q", got)
	}
}

func TestEnumChooserStagesDocumentedValue(t *testing.T) {
	original := configdoc.Parse([]byte("window-theme = auto\n"))
	options := []schema.Option{{Key: "window-theme", Kind: schema.KindEnum, Edit: schema.EditScalar, Values: []string{"auto", "system", "light"}}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	if model.choiceMode != choiceEnum {
		t.Fatalf("expected enum chooser, got %v", model.choiceMode)
	}
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "window-theme = system\n" {
		t.Fatalf("enum draft = %q", got)
	}
}

func TestMultiSelectStagesEnabledAndDisabledTokens(t *testing.T) {
	original := configdoc.Parse([]byte("shell-integration-features = cursor,no-title\n"))
	options := []schema.Option{{Key: "shell-integration-features", Kind: schema.KindString, Multiple: true, Values: []string{"cursor", "title"}, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	if !model.multiMode || model.multiIndex != 0 {
		t.Fatalf("multi editor did not open: mode=%v index=%d", model.multiMode, model.multiIndex)
	}
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "shell-integration-features = no-cursor,no-title\n" {
		t.Fatalf("multi draft = %q", got)
	}
}

func TestMultiSelectNoopDoesNotCreateExplicitAssignment(t *testing.T) {
	original := configdoc.Parse([]byte(""))
	options := []schema.Option{{Key: "font-synthetic-style", Kind: schema.KindString, Multiple: true, Values: []string{"bold", "italic"}, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyEnter))
	if model.HasChanges() {
		t.Fatalf("opening and accepting an untouched multi-select dirtied the draft: %q", model.DraftBytes())
	}
}

func TestDurationPickerStagesAmountAndUnit(t *testing.T) {
	original := configdoc.Parse([]byte("undo-timeout = 5s\n"))
	options := []schema.Option{{Key: "undo-timeout", Kind: schema.KindDuration, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(special(tea.KeyRight))
	model, _ = model.Update(special(tea.KeyUp))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "undo-timeout = 6m\n" {
		t.Fatalf("duration draft = %q", got)
	}
}

func TestCompoundDurationFallsBackWithoutTruncating(t *testing.T) {
	original := configdoc.Parse([]byte("undo-timeout = 1s 200ms\n"))
	options := []schema.Option{{Key: "undo-timeout", Kind: schema.KindDuration, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model, _ = model.Update(printable("e"))
	if !model.durationRawMode || model.durationMode {
		t.Fatalf("compound duration modes = raw %v structured %v", model.durationRawMode, model.durationMode)
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != string(original.Bytes()) {
		t.Fatalf("compound duration was rewritten: %q", got)
	}
	if model.HasChanges() {
		t.Fatal("no-op compound duration edit created a change")
	}
}

func TestPairFormStagesEnvironmentValue(t *testing.T) {
	original := configdoc.Parse([]byte("env = OLD=value\n"))
	options := []schema.Option{{Key: "env", Kind: schema.KindRepeatable, Edit: schema.EditRepeatable}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(printable("a"))
	if model.repeatableForm != repeatableFormPair {
		t.Fatalf("expected pair form, got %v", model.repeatableForm)
	}
	model.input.SetValue("FOO")
	model, _ = model.Update(special(tea.KeyEnter))
	model.input.SetValue("bar")
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "env = OLD=value\nenv = FOO=bar\n" {
		t.Fatalf("pair draft = %q", got)
	}
}

func TestKeybindFormUsesActionPickerAndStagesValue(t *testing.T) {
	original := configdoc.Parse([]byte("keybind = ctrl+a=ignore\n"))
	options := []schema.Option{{Key: "keybind", Kind: schema.KindKeybind, Edit: schema.EditRepeatable}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetActionChoices([]string{"ignore", "reload_config"})

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(printable("e"))
	if model.repeatableForm != repeatableFormKeybind {
		t.Fatalf("expected keybind form, got %v", model.repeatableForm)
	}
	model.input.SetValue("ctrl+r")
	model, _ = model.Update(special(tea.KeyEnter))
	if model.choiceMode != choiceAction {
		t.Fatalf("expected action chooser, got %v", model.choiceMode)
	}
	model, _ = model.Update(printable("reload"))
	model, _ = model.Update(special(tea.KeyEnter))
	model.input.SetValue("")
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "keybind = ctrl+r=reload_config\n" {
		t.Fatalf("keybind draft = %q", got)
	}
}

func TestFontFamilyRepeatableUsesFontPicker(t *testing.T) {
	original := configdoc.Parse([]byte("font-family = Menlo\n"))
	options := []schema.Option{{Key: "font-family", Kind: schema.KindRepeatable, Edit: schema.EditRepeatable}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetFontChoices([]string{"Menlo", "Monaco"})

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(printable("e"))
	if model.choiceMode != choiceFont {
		t.Fatalf("expected font chooser, got %v", model.choiceMode)
	}
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "font-family = Monaco\n" {
		t.Fatalf("font picker draft = %q", got)
	}
}

func TestPathPickerSelectsWorkingDirectory(t *testing.T) {
	directory := t.TempDir()
	configPath := directory + "/config.ghostty"
	original := configdoc.Parse([]byte("working-directory = " + directory + "\n"))
	options := []schema.Option{{Key: "working-directory", Kind: schema.KindPath, Edit: schema.EditScalar}}
	model := NewModel(options, configPath, original)

	model, _ = model.Update(printable("e"))
	if !model.pathMode || !model.pathDirectory {
		t.Fatalf("path picker did not open: mode=%v directory=%v", model.pathMode, model.pathDirectory)
	}
	// The selected directory is represented by `s`; it avoids depending on
	// the host's temporary directory contents.
	model, _ = model.Update(printable("s"))
	if got := string(model.DraftBytes()); got != "working-directory = "+shortenHomePath(directory)+"\n" {
		t.Fatalf("path picker draft = %q", got)
	}
}

func TestCommandPaletteFormUsesQuotedFieldsAndActionPicker(t *testing.T) {
	original := configdoc.Parse([]byte(`command-palette-entry = title:"Old",description:"Old description",action:"new_tab"
`))
	options := []schema.Option{{Key: "command-palette-entry", Kind: schema.KindRepeatable, Edit: schema.EditRepeatable}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetActionChoices([]string{"new_tab", "reload_config"})

	model, _ = model.Update(printable("e"))
	model, _ = model.Update(printable("e"))
	if model.repeatableForm != repeatableFormCommandPalette {
		t.Fatalf("expected command palette form, got %v", model.repeatableForm)
	}
	model.input.SetValue("New, title")
	model, _ = model.Update(special(tea.KeyEnter))
	model.input.SetValue("A description")
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(printable("reload"))
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(special(tea.KeyEnter))
	if got := string(model.DraftBytes()); got != "command-palette-entry = title:\"New, title\",description:\"A description\",action:\"reload_config\"\n" {
		t.Fatalf("command palette draft = %q", got)
	}
}

func TestSuccessfulSavePromotesDraftBaseline(t *testing.T) {
	original := configdoc.Parse([]byte("theme = old\n"))
	model := NewModel([]schema.Option{{Key: "theme", Kind: schema.KindString, Edit: schema.EditScalar}}, "fixture.ghostty", original)
	var calls [][]FileSnapshot
	model.SetSaveFunc(func(changes []FileSnapshot) error {
		calls = append(calls, changes)
		return nil
	})

	if err := model.stageScalar("theme", "new"); err != nil {
		t.Fatalf("stage first value: %v", err)
	}
	model.saveNow()
	if model.HasChanges() {
		t.Fatal("successful save left the first draft dirty")
	}
	if len(calls) != 1 || string(calls[0][0].Candidate) != "theme = new\n" {
		t.Fatalf("first save snapshots = %+v", calls)
	}

	if err := model.stageScalar("theme", "newer"); err != nil {
		t.Fatalf("stage second value: %v", err)
	}
	model.saveNow()
	if model.HasChanges() {
		t.Fatal("successful second save left the draft dirty")
	}
	if len(calls) != 2 || string(calls[1][0].Original) != "theme = new\n" || string(calls[1][0].Candidate) != "theme = newer\n" {
		t.Fatalf("second save snapshots = %+v", calls)
	}
}

func TestFriendlyOptionName(t *testing.T) {
	for key, want := range map[string]string{
		"font-size":    "Font Size",
		"gtk-titlebar": "GTK Titlebar",
		"macos-window": "macOS Window",
		"window-title": "Window Title",
	} {
		if got := friendlyOptionName(key); got != want {
			t.Errorf("friendlyOptionName(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestGraphResetStagesAgainstSourceDocument(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\n"))
	graph := configgraph.Graph{
		Roots:       []string{"config.ghostty"},
		Files:       []configgraph.File{{Path: "config.ghostty", Document: original}},
		Assignments: []configgraph.Assignment{{Key: "theme", Value: "dark", Path: "config.ghostty", Line: 1}},
	}
	options := []schema.Option{{Key: "theme", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewGraphModel(options, "config.ghostty", original, graph)
	model, _ = model.Update(printable("r"))
	if !model.HasChanges() {
		t.Fatal("graph reset did not stage a change")
	}
	if got := string(model.DraftBytes()); got != "theme =\n" {
		t.Fatalf("graph reset draft = %q", got)
	}
}

func TestGraphModelSearchesCatalogAndShowsEffectiveSource(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\nfont-size = 14\n"))
	graph := configgraph.Graph{
		Roots: []string{"config.ghostty"},
		Files: []configgraph.File{{Path: "config.ghostty", Document: original}},
		Assignments: []configgraph.Assignment{
			{Key: "theme", Value: "dark", Path: "config.ghostty", Line: 1},
			{Key: "font-size", Value: "14", Path: "config.ghostty", Line: 2},
		},
	}
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditReadOnlyRepeatable, Description: "Theme."},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditReadOnlyRepeatable, Description: "Font size."},
	}
	model := NewGraphModel(options, "config.ghostty", original, graph)

	model, _ = model.Update(printable("/"))
	model.search.SetValue("font")
	model, _ = model.Update(special(tea.KeyEnter))
	if got := model.SelectedKey(); got != "font-size" {
		t.Fatalf("search selected %q, want font-size", got)
	}
	if got := model.optionStatus(options[1]); got != "14" {
		t.Fatalf("effective option status = %q, want 14", got)
	}
	if got := model.optionSource(options[1]); got != "Effective source: config.ghostty:2" {
		t.Fatalf("effective source = %q", got)
	}
	model, _ = model.Update(printable("p"))
	if !strings.Contains(model.preview.GetContent(), "Effective Ghostty configuration graph") || !strings.Contains(model.preview.GetContent(), "config.ghostty:2") {
		t.Fatal("graph preview did not include effective source information")
	}
}

func TestGraphScalarEditTargetsEffectiveSourceFile(t *testing.T) {
	rootPath := "root.ghostty"
	childPath := "child.ghostty"
	root := configdoc.Parse([]byte("theme = dark\nconfig-file = child.ghostty\n"))
	child := configdoc.Parse([]byte("theme = light\n"))
	graph := configgraph.Graph{
		Roots: []string{rootPath},
		Files: []configgraph.File{
			{Path: rootPath, Document: root},
			{Path: childPath, Document: child},
		},
		Assignments: []configgraph.Assignment{
			{Key: "theme", Value: "dark", Path: rootPath, Line: 1, SourceFile: 0},
			{Key: "theme", Value: "light", Path: childPath, Line: 1, SourceFile: 1},
		},
	}
	options := []schema.Option{{Key: "theme", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewGraphModel(options, rootPath, root, graph)
	if err := model.stageScalar("theme", "rose-pine"); err != nil {
		t.Fatalf("stage graph scalar: %v", err)
	}
	if got := string(model.fileDrafts[0].Bytes()); got != string(root.Bytes()) {
		t.Fatalf("root changed instead of effective source: %q", got)
	}
	if got := string(model.fileDrafts[1].Bytes()); got != "theme = rose-pine\n" {
		t.Fatalf("child draft = %q", got)
	}
	if got := model.optionStatus(options[0]); got != "rose-pine (2 assignments)" {
		t.Fatalf("effective graph status = %q", got)
	}
}

func TestRepeatableEditorUpdatesAndAddsAcrossFiles(t *testing.T) {
	rootPath := "root.ghostty"
	childPath := "child.ghostty"
	root := configdoc.Parse([]byte("config-file = child.ghostty\n"))
	child := configdoc.Parse([]byte("keybind = ctrl+a=one\nkeybind = ctrl+b=two\n"))
	graph := configgraph.Graph{
		Roots: []string{rootPath},
		Files: []configgraph.File{
			{Path: rootPath, Document: root},
			{Path: childPath, Document: child},
		},
		Assignments: []configgraph.Assignment{
			{Key: "keybind", Value: "ctrl+a=one", Path: childPath, Line: 1, SourceFile: 1},
			{Key: "keybind", Value: "ctrl+b=two", Path: childPath, Line: 2, SourceFile: 1},
		},
	}
	options := []schema.Option{{Key: "keybind", Kind: schema.KindKeybind, Edit: schema.EditRepeatable}}
	model := NewGraphModel(options, rootPath, root, graph)
	items := model.repeatableItemsFor("keybind")
	items = append(items[1:], repeatableItem{Key: "keybind", Path: rootPath, Value: "ctrl+c=three"})
	if err := model.stageRepeatable("keybind", items); err != nil {
		t.Fatalf("stage repeatable graph value: %v", err)
	}
	if got := string(model.fileDrafts[1].Bytes()); got != "keybind = ctrl+b=two\n" {
		t.Fatalf("child repeatable draft = %q", got)
	}
	if got := string(model.fileDrafts[0].Bytes()); got != "config-file = child.ghostty\nkeybind = ctrl+c=three\n" {
		t.Fatalf("root repeatable draft = %q", got)
	}
}

func TestRepeatableEditorAddsValueThroughListControls(t *testing.T) {
	original := configdoc.Parse([]byte("keybind = ctrl+a=one\n"))
	options := []schema.Option{{Key: "keybind", Kind: schema.KindKeybind, Edit: schema.EditRepeatable}}
	model := NewModel(options, "fixture.ghostty", original)

	model, _ = model.Update(printable("e"))
	if model.Mode() != ModeEdit || !model.repeatableMode {
		t.Fatalf("repeatable editor did not open: mode=%v repeatable=%v", model.Mode(), model.repeatableMode)
	}
	model, _ = model.Update(printable("a"))
	if !model.repeatableEditing {
		t.Fatal("add control did not open the value input")
	}
	model.input.SetValue("ctrl+b=two")
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(special(tea.KeyEnter))
	if model.Mode() != ModeBrowse || !model.HasChanges() {
		t.Fatalf("repeatable list did not stage: mode=%v changes=%v", model.Mode(), model.HasChanges())
	}
	if got := string(model.DraftBytes()); got != "keybind = ctrl+a=one\nkeybind = ctrl+b=two\n" {
		t.Fatalf("repeatable list draft = %q", got)
	}
}

func TestCategoryNavigationStartsWithConfiguredScope(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\nfont-size = 14\n"))
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditScalar},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditScalar},
		{Key: "window-theme", Category: "Window", Kind: schema.KindEnum, Edit: schema.EditScalar, Values: []string{"auto", "dark"}},
	}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()

	if model.browseLevel != browseCategories {
		t.Fatalf("initial browse level = %v, want categories", model.browseLevel)
	}
	if !strings.Contains(model.View().Content, "Categories") || strings.Contains(model.View().Content, "Options · All Settings") {
		t.Fatalf("category-first view did not hide the flat catalog:\n%s", model.View().Content)
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if model.browseLevel != browseOptions || model.browseScope != browseScopeConfigured {
		t.Fatalf("configured section did not open: level=%v scope=%v", model.browseLevel, model.browseScope)
	}
	if got := len(model.visibleOptionIndexes()); got != 2 {
		t.Fatalf("configured options = %d, want 2", got)
	}
	if got := model.SelectedKey(); got != "theme" {
		t.Fatalf("configured selection = %q, want theme", got)
	}
}

func TestCategoryNavigationOpensAllAndReturnsToCategories(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\n"))
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditScalar},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditScalar},
		{Key: "keybind", Category: "Input", Kind: schema.KindKeybind, Edit: schema.EditRepeatable},
	}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()
	model, _ = model.Update(printable("a"))
	if model.browseScope != browseScopeAll || len(model.visibleOptionIndexes()) != len(options) {
		t.Fatalf("all settings scope = %v with %d options", model.browseScope, len(model.visibleOptionIndexes()))
	}
	model, _ = model.Update(printable("h"))
	if model.browseLevel != browseCategories {
		t.Fatalf("back navigation level = %v, want categories", model.browseLevel)
	}
}

func TestCategoryNavigationSearchesAllFieldsAndRestoresContext(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\n"))
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditScalar, Description: "Theme name."},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditScalar, Description: "Font size in points."},
		{Key: "keybind", Category: "Input", Kind: schema.KindKeybind, Edit: schema.EditRepeatable, Description: "Keyboard shortcut."},
	}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()
	model, _ = model.Update(printable("/"))
	model.search.SetValue("font size")
	model, _ = model.Update(special(tea.KeyEnter))
	if model.browseScope != browseScopeSearch || model.SelectedKey() != "font-size" {
		t.Fatalf("search scope=%v selected=%q, want search/font-size", model.browseScope, model.SelectedKey())
	}
	model, _ = model.Update(special(tea.KeyEscape))
	if model.browseLevel != browseCategories {
		t.Fatalf("search cancel level = %v, want categories", model.browseLevel)
	}
}

func TestCategoryNavigationFiltersAReviewedCategory(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\n"))
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditScalar},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditScalar},
		{Key: "window-theme", Category: "Window", Kind: schema.KindEnum, Edit: schema.EditScalar, Values: []string{"auto", "dark"}},
	}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()
	sections := model.browseSections()
	for index, section := range sections {
		if section.Key == "Colors" {
			model.categoryIndex = index
			break
		}
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if model.browseScope != browseScopeCategory || model.activeCategory != "Colors" {
		t.Fatalf("category scope = %v/%q", model.browseScope, model.activeCategory)
	}
	if len(model.visibleOptionIndexes()) != 1 || model.SelectedKey() != "theme" {
		t.Fatalf("colors options = %v, want theme only", model.visibleOptionIndexes())
	}
}

func TestAppearancePlaygroundPreviewsWithoutStagingOrWriting(t *testing.T) {
	original := configdoc.Parse([]byte("theme = Current\n"))
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar, Description: "A theme to use."}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetThemeChoices([]string{"Current", "Rose Pine"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#111111", Foreground: "#eeeeee"}},
		{Name: "Rose Pine", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#191724", Foreground: "#e0def4"}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if model.browseLevel != browseHome || !strings.Contains(model.View().Content, "Appearance playground") {
		t.Fatalf("appearance home not shown: level=%v\n%s", model.browseLevel, model.View().Content)
	}
	model, _ = model.Update(printable("t"))
	if model.browseLevel != browseThemes || model.selectedThemeName() != "Current" {
		t.Fatalf("theme browse state = level %v selected %q", model.browseLevel, model.selectedThemeName())
	}
	before := model.DraftBytes()
	var saves int
	model.SetSaveFunc(func([]FileSnapshot) error {
		saves++
		return nil
	})
	model, _ = model.Update(special(tea.KeyDown))
	if got := model.selectedThemeName(); got != "Rose Pine" {
		t.Fatalf("hover selection = %q, want Rose Pine", got)
	}
	if !bytes.Equal(model.DraftBytes(), before) || model.HasChanges() {
		t.Fatalf("hover preview mutated draft: draft=%q changes=%v", model.DraftBytes(), model.HasChanges())
	}
	if saves != 0 {
		t.Fatalf("hover preview invoked save %d times", saves)
	}
	if !strings.Contains(model.View().Content, "Rose Pine") || !strings.Contains(model.View().Content, "Candidate") {
		t.Fatalf("candidate preview was not rendered:\n%s", model.View().Content)
	}

	model, _ = model.Update(special(tea.KeyEnter))
	if model.browseLevel != browseHome || !model.HasChanges() {
		t.Fatalf("explicit selection did not stage theme: level=%v changes=%v", model.browseLevel, model.HasChanges())
	}
	if got := string(model.DraftBytes()); got != "theme = Rose Pine\n" {
		t.Fatalf("staged theme draft = %q", got)
	}
	if !strings.Contains(model.View().Content, "Staged theme Rose Pine") {
		t.Fatalf("staging status was not visible on appearance home:\n%s", model.View().Content)
	}
	if lines := len(strings.Split(model.View().Content, "\n")); lines > 40 {
		t.Fatalf("staged appearance home rendered %d lines in 120x40", lines)
	}
	if saves != 0 {
		t.Fatalf("staging invoked save %d times", saves)
	}
}

func TestAppearanceThemeSearchAndUnavailableFallback(t *testing.T) {
	original := configdoc.Parse([]byte("theme = custom-local\n"))
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar, Description: "A theme to use."}}
	model := NewModel(options, "fixture.ghostty", original)
	model.SetThemeChoices([]string{"Current Night", "Rose Pine"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current Night", Available: false, Error: "source missing"},
		{Name: "Rose Pine", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#191724", Foreground: "#e0def4"}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(printable("t"))
	if !strings.Contains(model.View().Content, "Preview unavailable") || !strings.Contains(model.View().Content, "No colors are guessed here") {
		t.Fatalf("unavailable preview was not explicit:\n%s", model.View().Content)
	}
	model, _ = model.Update(printable("/"))
	model.themeQuery.SetValue("rose")
	model, _ = model.Update(special(tea.KeyEnter))
	if len(model.themeVisible) != 1 || model.selectedThemeName() != "Rose Pine" {
		t.Fatalf("theme search result = %v selected=%q", model.themeVisible, model.selectedThemeName())
	}
	if model.HasChanges() {
		t.Fatal("theme search staged a change")
	}
}

func TestAppearancePreviewHonorsPaletteOverridesAndRejectsUnknownColors(t *testing.T) {
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	original := configdoc.Parse([]byte("theme = Current\npalette = 1=#010203\npalette = 3=terminal-red\n"))
	model := NewModel(options, "fixture.ghostty", original)
	palette := make([]string, 16)
	for index := range palette {
		palette[index] = "#101010"
	}
	model.SetThemePreviews([]ThemePreview{{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{
		Background: "#111111", Foreground: "#eeeeee", Palette: palette,
	}}})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	look := model.currentThemeLook()
	if look.Colors.Palette[1] != "#010203" || look.Colors.Palette[3] != "terminal-red" {
		t.Fatalf("explicit palette overrides were not applied: %#v", look.Colors.Palette)
	}
	if !look.Partial {
		t.Fatal("named palette override was presented as a complete preview")
	}
	if !strings.Contains(model.View().Content, "Palette preview partial") {
		t.Fatalf("partial palette warning was not visible:\n%s", model.View().Content)
	}

	named := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\nbackground = terminal-black\nforeground = #ffffff\n")))
	named.SetThemePreviews([]ThemePreview{{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{
		Background: "#111111", Foreground: "#eeeeee",
	}}})
	named.EnableCategoryNavigation()
	named.EnableAppearancePlayground()
	if named.currentThemeLook().Available {
		t.Fatal("named explicit background was rendered as an accurate color preview")
	}
	if !strings.Contains(named.View().Content, "Preview unavailable") || !strings.Contains(named.View().Content, "No colors are guessed") {
		t.Fatalf("unknown color fallback was not explicit:\n%s", named.View().Content)
	}

	partial := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\n")))
	partial.SetThemePreviews([]ThemePreview{{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{
		Background: "#111111", Foreground: "#eeeeee", Cursor: "#ffcc00", CursorText: "terminal-black",
		SelectionBackground: "#333333", SelectionForeground: "terminal-white",
	}}})
	partial.EnableCategoryNavigation()
	partial.EnableAppearancePlayground()
	content := partial.View().Content
	if !strings.Contains(content, "Cursor preview partial") || !strings.Contains(content, "Selection preview partial") {
		t.Fatalf("unknown optional colors were not labeled as partial:\n%s", content)
	}
}

func TestAppearanceHomeKeepsFullEditorReachable(t *testing.T) {
	original := configdoc.Parse([]byte("theme = Current\n"))
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyDown))
	model, _ = model.Update(special(tea.KeyEnter))
	if model.browseLevel != browseCategories || !strings.Contains(model.View().Content, "Categories") {
		t.Fatalf("full editor was not reachable: level=%v\n%s", model.browseLevel, model.View().Content)
	}
}

func TestAppearanceHomeFitsSmallTerminal(t *testing.T) {
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\n")))
	model.SetThemeChoices([]string{"Current"})
	model.SetThemePreviews([]ThemePreview{{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#111111", Foreground: "#eeeeee"}}})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	lines := len(strings.Split(model.View().Content, "\n"))
	if lines > 24 {
		t.Fatalf("appearance home rendered %d lines in a 80x24 terminal:\n%s", lines, model.View().Content)
	}
	content := model.View().Content
	assertANSIViewFits(t, content, 80, 24)
	plain := ansi.Strip(content)
	controls, preview := strings.Index(plain, "Search all settings"), strings.Index(plain, "CURRENT")
	if controls < 0 || preview <= controls {
		t.Fatalf("compact layout must show controls above the preview:\n%s", plain)
	}
}

func TestAppearanceThemeBrowseFitsSmallTerminal(t *testing.T) {
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\n")))
	model.SetThemeChoices([]string{"Current", "Rose Pine", "Tokyo Night", "Nord", "Solarized"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#111111", Foreground: "#eeeeee"}},
		{Name: "Rose Pine", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#191724", Foreground: "#e0def4"}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, _ = model.Update(printable("t"))
	lines := len(strings.Split(model.View().Content, "\n"))
	if lines > 24 {
		t.Fatalf("theme browse rendered %d lines in a 80x24 terminal:\n%s", lines, model.View().Content)
	}
	assertANSIViewFits(t, model.View().Content, 80, 24)
}

func TestAppearanceHomeStandardLayoutShowsRepresentativePreviews(t *testing.T) {
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\n")))
	model.SetThemeChoices([]string{"Current", "Rose Pine"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current", Available: true, Source: "/Applications/Ghostty.app/Contents/Resources/ghostty/themes/Catppuccin Macchiato", Colors: ThemeColors{
			Background: "#111111", Foreground: "#eeeeee", Cursor: "#ffcc00", CursorText: "#111111",
			SelectionBackground: "#333333", SelectionForeground: "#ffffff",
			Palette: []string{"#000000", "#ff5555", "#50fa7b", "#f1fa8c", "#8be9fd", "#ff79c6", "#8be9fd", "#f8f8f2"},
		}},
		{Name: "Rose Pine", Available: true, Source: "/Users/hiraism/Library/Application Support/com.mitchellh.ghostty/themes/Rose Pine", Colors: ThemeColors{
			Background: "#191724", Foreground: "#e0def4", Cursor: "#ebbcba", CursorText: "#191724",
			SelectionBackground: "#403d52", SelectionForeground: "#e0def4",
			Palette: []string{"#26233a", "#eb6f92", "#9ccfd8", "#f6c177", "#31748f", "#c4a7e7"},
		}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	content := model.View().Content
	lines := len(strings.Split(content, "\n"))
	if model.compactAppearanceLayout() {
		t.Fatal("120x40 unexpectedly selected compact appearance layout")
	}
	for _, row := range strings.Split(ansi.Strip(content), "\n") {
		if preview := strings.Index(row, "CURRENT LOOK"); preview >= 0 {
			if controls := strings.Index(row, "Settings"); controls < 0 || controls >= preview {
				t.Fatalf("wide layout must show settings left of the preview: %q", row)
			}
		}
	}
	if !strings.Contains(content, "Current") || !strings.Contains(content, "Browse themes") || !strings.Contains(content, "highlighted text") {
		t.Fatalf("standard preview is missing its theme controls or selection sample:\n%s", content)
	}
	if !strings.Contains(content, "48;2;0;0;0") {
		t.Fatalf("standard preview is missing its ANSI palette swatches:\n%s", content)
	}
	if !strings.Contains(content, "Ghostty bundled theme file") || strings.Contains(content, "/Applications/Ghostty.app/") || strings.Contains(content, "/Users/hiraism/") {
		t.Fatalf("standard preview exposed a raw source path or omitted short provenance:\n%s", content)
	}
	if !strings.Contains(content, "Theme colors previewed here") {
		t.Fatalf("standard preview omitted its preview scope:\n%s", content)
	}
	if lines > 40 {
		t.Fatalf("standard appearance home rendered %d lines in 120x40:\n%s", lines, content)
	}
	assertANSIViewFits(t, content, 120, 40)
}

func TestAppearanceThemeBrowserStandardLayoutKeepsBothPreviewsVisible(t *testing.T) {
	options := []schema.Option{{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar}}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current Theme\n")))
	model.SetThemeChoices([]string{"Current Theme", "Candidate Theme With A Long Name"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current Theme", Available: true, Source: "/Applications/Ghostty.app/Contents/Resources/ghostty/themes/Current Theme", Colors: ThemeColors{Background: "#111111", Foreground: "#eeeeee", Cursor: "#ffcc00", CursorText: "#111111", SelectionBackground: "#333333", SelectionForeground: "#ffffff", Palette: []string{"#000000", "#ff5555", "#50fa7b", "#f1fa8c", "#8be9fd", "#ff79c6", "#8be9fd", "#f8f8f2"}}},
		{Name: "Candidate Theme With A Long Name", Available: true, Source: "/Users/hiraism/Library/Application Support/com.mitchellh.ghostty/themes/Candidate Theme With A Long Name", Colors: ThemeColors{Background: "#191724", Foreground: "#e0def4", Cursor: "#ebbcba", CursorText: "#191724", SelectionBackground: "#403d52", SelectionForeground: "#e0def4", Palette: []string{"#26233a", "#eb6f92", "#9ccfd8", "#f6c177", "#31748f", "#c4a7e7"}}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, _ = model.Update(printable("t"))
	model, _ = model.Update(special(tea.KeyDown))
	content := model.View().Content
	if !strings.Contains(content, "Current Theme") || !strings.Contains(content, "Candidate Theme With A Long Name") {
		t.Fatalf("theme browser lost current/candidate names:\n%s", content)
	}
	if !strings.Contains(content, "Candidate") || !strings.Contains(content, "highlighted text") || !strings.Contains(content, "48;2;0;0;0") {
		t.Fatalf("theme browser lost candidate terminal details:\n%s", content)
	}
	if strings.Contains(content, "/Applications/Ghostty.app/") || strings.Contains(content, "/Users/hiraism/") {
		t.Fatalf("theme browser exposed a raw source path:\n%s", content)
	}
	assertANSIViewFits(t, content, 120, 40)
}

func TestAppearanceTerminalPreviewFillsTheThemeSurface(t *testing.T) {
	look := appearanceLook{
		Name:      "Rose Pine",
		Available: true,
		Colors: ThemeColors{
			Background:          "#191724",
			Foreground:          "#e0def4",
			Cursor:              "#ebbcba",
			CursorText:          "#191724",
			SelectionBackground: "#403d52",
			SelectionForeground: "#e0def4",
			Palette:             []string{"#26233a", "#eb6f92", "#9ccfd8", "#f6c177"},
		},
	}
	const width = 34
	rendered := renderTerminalPreview(look, width)
	lines := strings.Split(rendered, "\n")
	if len(lines) < 7 {
		t.Fatalf("terminal preview has %d lines, want representative content:\n%s", len(lines), rendered)
	}
	for index, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("preview line %d has width %d, want full surface width %d: %q", index+1, got, width, line)
		}
	}
	if !strings.Contains(rendered, "highlighted text") {
		t.Fatalf("preview omitted its selection sample:\n%s", rendered)
	}
	if !strings.Contains(rendered, "48;2;38;35;58") {
		t.Fatalf("preview omitted its palette background token:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\x1b[m\x1b[38;2;224;222;244;48;2;25;23;36m cursor") {
		t.Fatalf("selection or cursor styles did not restore the theme canvas:\n%s", rendered)
	}
}

func TestAppearanceShortcutsMatchEachSurface(t *testing.T) {
	options := []schema.Option{
		{Key: "theme", Category: "Colors", Kind: schema.KindString, Edit: schema.EditScalar},
		{Key: "font-synthetic-style", Category: "Fonts", Kind: schema.KindString, Edit: schema.EditScalar, Description: "Depends on the current font size."},
		{Key: "font-size", Category: "Fonts", Kind: schema.KindNumber, Edit: schema.EditScalar},
	}
	model := NewModel(options, "fixture.ghostty", configdoc.Parse([]byte("theme = Current\n")))
	model.SetThemeChoices([]string{"Current", "Rose Pine"})
	model.SetThemePreviews([]ThemePreview{
		{Name: "Current", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#111111", Foreground: "#eeeeee"}},
		{Name: "Rose Pine", Available: true, Source: "Ghostty theme file", Colors: ThemeColors{Background: "#191724", Foreground: "#e0def4"}},
	})
	model.EnableCategoryNavigation()
	model.EnableAppearancePlayground()
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	home := model.View().Content
	if !strings.Contains(home, "Search all settings") {
		t.Fatalf("home did not show global search:\n%s", home)
	}
	model, _ = model.Update(special(tea.KeyEnter))
	if !model.searching || !strings.Contains(ansi.Strip(model.View().Content), "Search settings by name or keyword") {
		t.Fatalf("the selected home action did not open a visible search field:\n%s", ansi.Strip(model.View().Content))
	}
	model, _ = model.Update(tea.PasteMsg{Content: "font size"})
	if got := model.SelectedKey(); got != "font-size" {
		t.Fatalf("live global search selected %q, want font-size", got)
	}
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assertANSIViewFits(t, model.View().Content, 80, 24)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, _ = model.Update(special(tea.KeyEnter))
	model, _ = model.Update(printable("/"))
	if !model.searching || model.search.Value() != "font size" {
		t.Fatal("reopening search did not keep the query")
	}
	model, _ = model.Update(special(tea.KeyEscape))
	if model.browseLevel != browseHome || model.HasChanges() {
		t.Fatal("closing the search did not return to the unchanged home")
	}
	model, _ = model.Update(printable("/"))
	if !model.searching {
		t.Fatal("slash did not open global search from home")
	}
	model, _ = model.Update(special(tea.KeyEscape))
	model, _ = model.Update(printable("t"))
	model, _ = model.Update(special(tea.KeyDown))
	themes := model.View().Content
	for _, shortcut := range []string{"/ search themes", "ctrl+f search all settings", "enter stage"} {
		if !strings.Contains(themes, shortcut) {
			t.Fatalf("theme browser omitted %q shortcut:\n%s", shortcut, themes)
		}
	}
	model, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: 'f', Mod: tea.ModCtrl}))
	if !model.searching || model.browseScope != browseScopeSearch {
		t.Fatal("ctrl+f did not open global search from the theme browser")
	}
	model, _ = model.Update(special(tea.KeyEscape))
	if model.browseLevel != browseThemes || model.selectedThemeName() != "Rose Pine" {
		t.Fatal("closing global search did not restore the highlighted theme")
	}
	model, _ = model.Update(printable("/"))
	if !model.themeSearching || model.searching {
		t.Fatal("slash in the theme browser should search themes")
	}
}

func assertANSIViewFits(t *testing.T, content string, width, height int) {
	t.Helper()
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		t.Fatalf("rendered %d lines, want <= %d", len(lines), height)
	}
	for index, line := range lines {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("rendered line %d has ANSI-aware width %d, want <= %d: %q", index+1, got, width, line)
		}
	}
}

func TestConfiguredScopeUpdatesAfterStagingFromAllSettings(t *testing.T) {
	original := configdoc.Parse([]byte("theme = dark\n"))
	options := []schema.Option{
		{Key: "theme", Category: "Appearance", Kind: schema.KindString, Edit: schema.EditScalar},
		{Key: "font-size", Category: "Appearance", Kind: schema.KindNumber, Edit: schema.EditScalar},
	}
	model := NewModel(options, "fixture.ghostty", original)
	model.EnableCategoryNavigation()
	model, _ = model.Update(printable("a"))
	model.selected = 1
	model, _ = model.Update(printable("e"))
	model.input.SetValue("14")
	model, _ = model.Update(special(tea.KeyEnter))
	if len(model.configuredKeys()) != 2 {
		t.Fatalf("configured keys after staging = %v, want two keys", model.configuredKeys())
	}
	model, _ = model.Update(printable("h"))
	model, _ = model.Update(printable("c"))
	if model.browseScope != browseScopeConfigured || len(model.visibleOptionIndexes()) != 2 {
		t.Fatalf("configured scope after staging = %v with %d options", model.browseScope, len(model.visibleOptionIndexes()))
	}
}

func printable(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value, Code: rune(value[0])})
}

func special(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}
