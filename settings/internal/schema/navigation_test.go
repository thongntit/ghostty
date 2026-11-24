package schema

import "testing"

func TestNavigationCoversEmbeddedCatalog(t *testing.T) {
	catalog, err := EmbeddedCatalog()
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	if err := ValidateNavigation(catalog.Options); err != nil {
		t.Fatalf("validate embedded navigation: %v", err)
	}

	sections := NavigationSections()
	if len(sections) < 10 {
		t.Fatalf("navigation sections = %d, want a broad category surface", len(sections))
	}
	seen := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		if section.Name == "" {
			t.Fatal("navigation section has an empty name")
		}
		if _, ok := seen[section.Name]; ok {
			t.Fatalf("duplicate navigation section %q", section.Name)
		}
		seen[section.Name] = struct{}{}
	}
}

func TestNavigationCategoryUsesDedicatedAreas(t *testing.T) {
	tests := map[string]string{
		"font-family":            "Fonts",
		"theme":                  "Colors",
		"keybind":                "Keybindings",
		"working-directory":      "Shell",
		"clipboard-read":         "Clipboard",
		"mouse-reporting":        "Mouse",
		"gtk-tabs-location":      "GTK",
		"macos-icon":             "macOS",
		"linux-cgroup":           "Linux",
		"window-theme":           "Window",
		"scrollback-limit-lines": "Terminal",
		"language":               "Application",
	}
	for key, want := range tests {
		if got := NavigationCategoryForKey(key); got != want {
			t.Errorf("NavigationCategoryForKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestNavigationKeepsUnknownOptionInCustom(t *testing.T) {
	option := Option{Key: "future-option", Category: "Custom"}
	if got := NavigationCategory(option); got != "Custom" {
		t.Fatalf("unknown option category = %q, want Custom", got)
	}
	if err := ValidateNavigation([]Option{option}); err != nil {
		t.Fatalf("validate custom option: %v", err)
	}
}
