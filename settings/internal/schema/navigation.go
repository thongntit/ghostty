package schema

import (
	"fmt"
	"strings"
)

// NavigationSection is an ordered top-level area in the guided editor. The
// catalog still owns the complete option surface; this is the product-facing
// information architecture used to keep that surface approachable.
type NavigationSection struct {
	Name        string
	Description string
}

var navigationSections = []NavigationSection{
	{Name: "Application", Description: "General behavior, startup, shutdown, and updates."},
	{Name: "Terminal", Description: "Terminal rendering, scrollback, bell, and protocol behavior."},
	{Name: "Shell", Description: "Shell integration, commands, environment, and working directory."},
	{Name: "Clipboard", Description: "Clipboard permissions, paste safety, and selection behavior."},
	{Name: "Window", Description: "Window layout, tabs, splits, titles, and quick terminal behavior."},
	{Name: "Colors", Description: "Themes, colors, palettes, contrast, and background imagery."},
	{Name: "Fonts", Description: "Font families, styles, shaping, and metric adjustments."},
	{Name: "Keybindings", Description: "Keyboard shortcuts, remaps, and command palette entries."},
	{Name: "Mouse", Description: "Mouse reporting, links, scrolling, and pointer actions."},
	{Name: "GTK", Description: "GTK-specific window, tab, toolbar, and CSS settings."},
	{Name: "Linux", Description: "Linux-specific process and resource settings."},
	{Name: "macOS", Description: "macOS-specific window, icon, shortcut, and input settings."},
}

// NavigationSections returns the stable order used by the category screen.
func NavigationSections() []NavigationSection {
	return append([]NavigationSection(nil), navigationSections...)
}

// NavigationCategory returns the reviewed product category for an option.
// Unknown options are intentionally kept under Custom so the editor can
// preserve and expose version-specific settings without hiding them.
func NavigationCategory(option Option) string {
	if option.Category == "Custom" {
		return "Custom"
	}
	return NavigationCategoryForKey(option.Key)
}

// NavigationCategoryForKey maps the known Ghostty grammar to the guided
// editor's top-level areas. Prefixes are used only where Ghostty itself has a
// stable namespace (gtk-, linux-, macos-, font-, window-); ambiguous settings
// are listed explicitly below.
func NavigationCategoryForKey(key string) string {
	switch {
	case strings.HasPrefix(key, "macos-"):
		return "macOS"
	case strings.HasPrefix(key, "gtk-"):
		return "GTK"
	case strings.HasPrefix(key, "linux-"):
		return "Linux"
	case key == "keybind", key == "key-remap", key == "command-palette-entry":
		return "Keybindings"
	case strings.HasPrefix(key, "font-"), strings.HasPrefix(key, "adjust-"):
		return "Fonts"
	case key == "freetype-load-flags", key == "grapheme-width-method":
		return "Fonts"
	case key == "theme", key == "background", key == "foreground", key == "background-image",
		key == "background-image-opacity", key == "background-image-position", key == "background-image-fit",
		key == "background-image-repeat", key == "selection-foreground", key == "selection-background",
		key == "selection-word-chars", key == "minimum-contrast", key == "palette",
		key == "palette-generate", key == "palette-harmonious", key == "cursor-color",
		key == "cursor-opacity", key == "cursor-text", key == "search-foreground",
		key == "search-background", key == "search-selected-foreground", key == "search-selected-background",
		key == "unfocused-split-opacity", key == "unfocused-split-fill", key == "bold-color",
		key == "window-titlebar-background", key == "window-titlebar-foreground":
		return "Colors"
	case key == "clipboard-codepoint-map", key == "clipboard-read", key == "clipboard-write",
		key == "clipboard-write-limit-bytes", key == "clipboard-trim-trailing-spaces",
		key == "clipboard-paste-protection", key == "clipboard-paste-bracketed-safe",
		key == "copy-on-select", key == "selection-clear-on-typing", key == "selection-clear-on-copy":
		return "Clipboard"
	case key == "mouse-hide-while-typing", key == "mouse-shift-capture", key == "mouse-reporting",
		key == "mouse-scroll-multiplier", key == "cursor-click-to-move", key == "link-url",
		key == "link-osc8", key == "link-previews", key == "right-click-action",
		key == "middle-click-action", key == "click-repeat-interval":
		return "Mouse"
	case key == "split-divider-color", key == "split-preserve-zoom", key == "maximize", key == "fullscreen",
		key == "title", key == "class", key == "x11-instance-name", key == "window-padding-x",
		key == "window-padding-y", key == "window-padding-balance", key == "window-padding-color",
		key == "window-vsync", key == "window-inherit-working-directory", key == "tab-inherit-working-directory",
		key == "split-inherit-working-directory", key == "window-inherit-font-size", key == "window-decoration",
		key == "window-title-font-family", key == "window-subtitle", key == "window-theme",
		key == "window-colorspace", key == "window-height", key == "window-width",
		key == "window-position-x", key == "window-position-y", key == "window-save-state",
		key == "window-step-resize", key == "window-new-tab-position", key == "window-show-tab-bar",
		key == "drag-handle", key == "resize-overlay", key == "resize-overlay-position",
		key == "resize-overlay-duration", key == "focus-follows-mouse", strings.HasPrefix(key, "quick-terminal-"):
		return "Window"
	case key == "command", key == "initial-command", key == "env", key == "working-directory",
		key == "shell-integration", key == "shell-integration-features", key == "term":
		return "Shell"
	case key == "scroll-to-bottom", key == "scrollback-limit-bytes", key == "scrollback-limit-lines",
		key == "scrollback-compression", key == "scrollbar", key == "cursor-style",
		key == "cursor-style-blink", key == "bell-features", key == "bell-audio-path",
		key == "bell-audio-volume", key == "vt-kam-allowed", key == "image-storage-limit",
		key == "adjust-cell-width", key == "input",
		key == "faint-opacity":
		return "Terminal"
	default:
		return "Application"
	}
}

// ValidateNavigation confirms that every catalog option resolves to exactly
// one known area. Custom is the only dynamic category and is accepted for
// options discovered directly in a user's config.
func ValidateNavigation(options []Option) error {
	known := make(map[string]struct{}, len(navigationSections)+1)
	for _, section := range navigationSections {
		known[section.Name] = struct{}{}
	}
	known["Custom"] = struct{}{}
	seen := make(map[string]string, len(options))
	for _, option := range options {
		if strings.TrimSpace(option.Key) == "" {
			return fmt.Errorf("navigation option key is empty")
		}
		category := NavigationCategory(option)
		if _, ok := known[category]; !ok {
			return fmt.Errorf("option %q resolves to unknown navigation category %q", option.Key, category)
		}
		if previous, ok := seen[option.Key]; ok {
			return fmt.Errorf("option %q appears more than once (%s and %s)", option.Key, previous, category)
		}
		seen[option.Key] = category
	}
	return nil
}
