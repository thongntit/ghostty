package ghostty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseThemeListStripsSourceAnnotationsAndDeduplicates(t *testing.T) {
	got := parseThemeList([]byte("Catppuccin Mocha (resources)\nMy Theme (config)\nCatppuccin Mocha (resources)\n\n"))
	want := []string{"Catppuccin Mocha", "My Theme"}
	if len(got) != len(want) {
		t.Fatalf("theme count = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("theme %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParseColorListKeepsNamesWithSpaces(t *testing.T) {
	got := parseColorList([]byte("alice blue = #f0f8ff\nAliceBlue = #f0f8ff\nnot a color\n"))
	if len(got) != 2 {
		t.Fatalf("color count = %d, want 2: %+v", len(got), got)
	}
	if got[0].Name != "alice blue" || got[0].Value != "#f0f8ff" {
		t.Fatalf("first color = %+v", got[0])
	}
}

func TestParseFontListKeepsFamiliesAndSkipsFaces(t *testing.T) {
	got := parseFontList([]byte("Andale Mono\n  Andale Mono\n\nMenlo\n\tMenlo Bold\nMenlo\nerror: SentryInitFailed\n"))
	want := []string{"Andale Mono", "Menlo"}
	if len(got) != len(want) {
		t.Fatalf("font count = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("font %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParseActionListSkipsDiagnosticsAndDeduplicates(t *testing.T) {
	got := parseActionList([]byte("error: SentryInitFailed\nnew_window\ncopy_to_clipboard\nnew_window\n\n"))
	want := []string{"new_window", "copy_to_clipboard"}
	if len(got) != len(want) {
		t.Fatalf("action count = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("action %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParseThemeColorsUsesGhosttyAssignments(t *testing.T) {
	colors, err := parseThemeColors([]byte("# source theme\nbackground = #101820\nforeground = #f2f4f8\npalette = 1=#ff0000\npalette = 15=#ffffff\ncursor-color = #00ff00\n"))
	if err != nil {
		t.Fatalf("parse theme colors: %v", err)
	}
	if colors.Background != "#101820" || colors.Foreground != "#f2f4f8" || colors.Cursor != "#00ff00" {
		t.Fatalf("theme colors = %+v", colors)
	}
	if len(colors.Palette) != 16 || colors.Palette[1] != "#ff0000" || colors.Palette[15] != "#ffffff" {
		t.Fatalf("theme palette = %#v", colors.Palette)
	}
}

func TestResolveThemeReadsOnlyThemeSourceAndReportsUnavailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	themeDir := filepath.Join(home, "config", "ghostty", "themes")
	if err := os.MkdirAll(themeDir, 0o700); err != nil {
		t.Fatalf("mkdir themes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "Local Theme"), []byte("background = #000000\nforeground = #ffffff\n"), 0o600); err != nil {
		t.Fatalf("write theme: %v", err)
	}

	preview, err := ResolveTheme("/does/not/exist/ghostty", "Local Theme")
	if err != nil || !preview.Available || preview.Source == "" {
		t.Fatalf("resolved preview = %+v, err=%v", preview, err)
	}

	missing, err := ResolveTheme("/does/not/exist/ghostty", "Missing Theme")
	if err == nil || missing.Available || missing.Error == "" {
		t.Fatalf("missing preview = %+v, err=%v", missing, err)
	}
}

func TestResolveDefaultThemeUsesGhosttyDefaultConfig(t *testing.T) {
	command := filepath.Join(t.TempDir(), "ghostty")
	script := "#!/bin/sh\n[ \"$1\" = +show-config ] && [ \"$2\" = --default ] && [ \"$3\" = --no-pager ] || exit 2\ncat <<'EOF'\nbackground = #282c34\nforeground = #ffffff\npalette = 0=#1d1f21\npalette = 15=#eaeaea\nEOF\n"
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatalf("write Ghostty command: %v", err)
	}

	preview, err := ResolveDefaultTheme(command)
	if err != nil {
		t.Fatalf("resolve default theme: %v", err)
	}
	if preview.Name != "" || !preview.Available || preview.Source == "" {
		t.Fatalf("default preview = %+v", preview)
	}
	if preview.Colors.Background != "#282c34" || preview.Colors.Foreground != "#ffffff" || len(preview.Colors.Palette) != 16 || preview.Colors.Palette[15] != "#eaeaea" {
		t.Fatalf("default colors = %+v", preview.Colors)
	}
}
