package schema

import "strings"

// ControlKind describes the interaction model for a value. It is deliberately
// separate from ValueKind: Ghostty has string-shaped values that still have a
// safe, structured editor (for example resource limits and font metrics).
type ControlKind string

const (
	ControlRaw            ControlKind = "raw"
	ControlToggle         ControlKind = "toggle"
	ControlSelect         ControlKind = "select"
	ControlMultiSelect    ControlKind = "multi-select"
	ControlNumber         ControlKind = "number"
	ControlColor          ControlKind = "color"
	ControlDuration       ControlKind = "duration"
	ControlPath           ControlKind = "path"
	ControlTheme          ControlKind = "theme"
	ControlFont           ControlKind = "font"
	ControlFontStyle      ControlKind = "font-style"
	ControlMetric         ControlKind = "metric"
	ControlResource       ControlKind = "resource"
	ControlPair           ControlKind = "pair"
	ControlKeybind        ControlKind = "keybind"
	ControlCommand        ControlKind = "command"
	ControlCommandPalette ControlKind = "command-palette"
	ControlRepeatable     ControlKind = "repeatable"
)

// Control is optional in generated catalogs for backwards compatibility. The
// resolver below provides a reviewed fallback for older catalogs and tests
// that construct Options inline.
func (o Option) ControlModel() ControlKind {
	if o.Control != "" {
		return o.Control
	}
	return ControlFor(o)
}

// ControlFor returns the reviewed editor for an option. Key-specific rules are
// kept here instead of being scattered through Bubble Tea event handlers.
func ControlFor(option Option) ControlKind {
	if option.Multiple {
		return ControlMultiSelect
	}
	switch option.Kind {
	case KindBoolean:
		return ControlToggle
	case KindNumber:
		return ControlNumber
	case KindColor:
		return ControlColor
	case KindDuration:
		return ControlDuration
	case KindPath:
		return ControlPath
	case KindEnum:
		return ControlSelect
	case KindKeybind:
		return ControlKeybind
	case KindCommand:
		return ControlCommand
	case KindRepeatable:
		return repeatableControl(option.Key)
	}

	switch option.Key {
	case "theme":
		return ControlTheme
	case "font-style", "font-style-bold", "font-style-italic", "font-style-bold-italic":
		return ControlFontStyle
	case "scrollback-limit-lines", "linux-cgroup-memory-limit", "linux-cgroup-processes-limit":
		return ControlResource
	case "mouse-scroll-multiplier", "font-variation-bold", "font-variation-italic", "font-variation-bold-italic":
		return ControlPair
	}
	if strings.HasPrefix(option.Key, "adjust-") {
		return ControlMetric
	}
	if strings.HasPrefix(option.Key, "font-family") || option.Key == "window-title-font-family" {
		return ControlFont
	}
	return ControlRaw
}

func repeatableControl(key string) ControlKind {
	switch key {
	case "font-family":
		return ControlFont
	case "font-variation":
		return ControlPair
	case "font-codepoint-map", "clipboard-codepoint-map":
		return ControlPair
	case "palette":
		return ControlPair
	case "env":
		return ControlPair
	case "key-remap":
		return ControlPair
	case "keybind":
		return ControlKeybind
	case "command-palette-entry":
		return ControlCommandPalette
	case "config-file", "custom-shader", "gtk-custom-css":
		return ControlPath
	default:
		return ControlRepeatable
	}
}

// GroupForKey supplies the second-level navigation labels for large areas.
// These labels are intentionally stable and user-facing; they are not derived
// from prose that may change between Ghostty releases.
func GroupForKey(key string) string {
	switch {
	case strings.HasPrefix(key, "font-family"), key == "font-size", strings.HasPrefix(key, "font-style"):
		return "Families & styles"
	case strings.HasPrefix(key, "font-feature"), strings.HasPrefix(key, "font-variation"), key == "freetype-load-flags":
		return "Features & axes"
	case strings.HasPrefix(key, "font-codepoint"), key == "clipboard-codepoint-map":
		return "Codepoint mapping"
	case strings.HasPrefix(key, "adjust-"):
		return "Metrics"
	case key == "theme", strings.HasPrefix(key, "background"), strings.HasPrefix(key, "foreground"), key == "palette", strings.HasPrefix(key, "cursor-"), strings.HasPrefix(key, "search-"), strings.HasPrefix(key, "window-titlebar-"):
		return "Theme & colors"
	case strings.HasPrefix(key, "clipboard-"), strings.HasPrefix(key, "selection-"):
		return "Clipboard & selection"
	case key == "command", key == "initial-command", key == "env", key == "working-directory", key == "term":
		return "Commands & environment"
	case key == "shell-integration", key == "shell-integration-features":
		return "Shell integration"
	case strings.HasPrefix(key, "window-position"), strings.HasPrefix(key, "window-width"), strings.HasPrefix(key, "window-height"), strings.HasPrefix(key, "window-padding"):
		return "Geometry"
	case strings.HasPrefix(key, "quick-terminal"):
		return "Quick terminal"
	case strings.Contains(key, "split"), strings.Contains(key, "tab"):
		return "Tabs & splits"
	case strings.HasPrefix(key, "keybind"), key == "key-remap", key == "command-palette-entry":
		return "Shortcuts"
	case strings.HasPrefix(key, "scrollback"), key == "cursor-style", key == "cursor-style-blink":
		return "Display & scrollback"
	case strings.HasPrefix(key, "clipboard"), strings.HasPrefix(key, "selection"):
		return "Clipboard & selection"
	case strings.HasPrefix(key, "mouse"), strings.HasPrefix(key, "link"), strings.Contains(key, "click"):
		return "Mouse & links"
	case strings.HasPrefix(key, "gtk-"):
		return "GTK"
	case strings.HasPrefix(key, "macos-"):
		return "macOS"
	case strings.HasPrefix(key, "linux-"):
		return "Linux"
	default:
		return "General"
	}
}

// ApplyControlMetadata fills generated metadata while keeping hand-written
// tests and older catalogs compatible.
func ApplyControlMetadata(option *Option) {
	if option == nil {
		return
	}
	if option.Control == "" {
		option.Control = ControlFor(*option)
	}
	if option.Group == "" {
		option.Group = GroupForKey(option.Key)
	}
}
