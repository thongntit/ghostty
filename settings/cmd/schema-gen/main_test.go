package main

import (
	"strings"
	"testing"

	"github.com/thongntit/ghostty/settings/internal/schema"
)

func TestParseVersionIgnoresDiagnosticPrefix(t *testing.T) {
	got, err := parseVersion([]byte("error: SentryInitFailed\nGhostty 1.3.2-main-+abc\n"))
	if err != nil {
		t.Fatalf("parse version: %v", err)
	}
	if got != "1.3.2-main-+abc" {
		t.Fatalf("version = %q", got)
	}
}

func TestSourceRevisionExtractsBuildRevision(t *testing.T) {
	if got := sourceRevision("1.3.2-main-+44f2a44df"); got != "44f2a44df" {
		t.Fatalf("source revision = %q", got)
	}
	if got := sourceRevision("1.3.2"); got != "" {
		t.Fatalf("release source revision = %q, want empty", got)
	}
}

func TestCatalogFromDocsDeduplicatesAndClassifiesOptions(t *testing.T) {
	docs := []byte("# A theme.\ntheme =\n\n# Toggle it.\nenable-feature = true\n\n# Font size.\nfont-size = 13\n\n# A documented enum.\ncursor-style = block\n\n# A color without the word color in its key.\nselection-foreground = #ffffff\n\n# A documented list.\nfont-synthetic-style = bold\n\n# Additional configuration files to read.\nconfig-file = ?optional.conf\n\n# Binding.\nkeybind = ctrl+a=ignore\n\n# duplicate docs entry\ntheme =\n")
	catalog, err := catalogFromDocs(docs, "test")
	if err != nil {
		t.Fatalf("catalog from docs: %v", err)
	}
	if len(catalog.Options) != 8 {
		t.Fatalf("options = %d, want 8", len(catalog.Options))
	}
	byKey := make(map[string]schema.Option, len(catalog.Options))
	for _, option := range catalog.Options {
		byKey[option.Key] = option
	}
	if byKey["enable-feature"].Kind != schema.KindBoolean {
		t.Fatalf("boolean classification = %q", byKey["enable-feature"].Kind)
	}
	if byKey["config-file"].Kind != schema.KindRepeatable {
		t.Fatalf("config-file classification = %q", byKey["config-file"].Kind)
	}
	if byKey["keybind"].Kind != schema.KindKeybind {
		t.Fatalf("keybind classification = %q", byKey["keybind"].Kind)
	}
	if byKey["keybind"].Edit != schema.EditRepeatable || byKey["theme"].Edit != schema.EditScalar {
		t.Fatalf("editor classification = keybind:%q theme:%q", byKey["keybind"].Edit, byKey["theme"].Edit)
	}
	if byKey["font-size"].Step == nil || *byKey["font-size"].Step != 0.5 {
		t.Fatalf("font-size step = %+v, want 0.5", byKey["font-size"].Step)
	}
	if byKey["cursor-style"].Kind != schema.KindEnum || len(byKey["cursor-style"].Values) != 4 {
		t.Fatalf("cursor-style metadata = %+v", byKey["cursor-style"])
	}
	if byKey["selection-foreground"].Kind != schema.KindColor {
		t.Fatalf("selection-foreground classification = %q", byKey["selection-foreground"].Kind)
	}
	if !byKey["font-synthetic-style"].Multiple || byKey["font-synthetic-style"].Values[0] != "bold" {
		t.Fatalf("font-synthetic-style metadata = %+v", byKey["font-synthetic-style"])
	}
	if !strings.Contains(byKey["theme"].Docs, "#theme") {
		t.Fatalf("theme docs link = %q", byKey["theme"].Docs)
	}
}

func TestCatalogFromDocsClassifiesOptionalMenuBarIconAsToggle(t *testing.T) {
	docs := []byte("# When unset, the icon follows macos-hidden. Set this to `true` to show it, or `false` to hide it.\nmacos-menu-bar-icon = \n")
	catalog, err := catalogFromDocs(docs, "test")
	if err != nil {
		t.Fatalf("catalog from docs: %v", err)
	}
	if len(catalog.Options) != 1 {
		t.Fatalf("options = %d, want 1", len(catalog.Options))
	}
	option := catalog.Options[0]
	if option.Kind != schema.KindBoolean || option.ControlModel() != schema.ControlToggle {
		t.Fatalf("optional toggle metadata = kind:%q control:%q", option.Kind, option.ControlModel())
	}
	for _, value := range []string{"true", "false"} {
		if err := option.Validate(value); err != nil {
			t.Errorf("explicit value %q rejected: %v", value, err)
		}
	}
	if err := option.Validate("maybe"); err == nil {
		t.Error("invalid explicit value accepted")
	}
}
