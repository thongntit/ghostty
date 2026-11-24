package ghostty

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thongntit/ghostty/settings/internal/configdoc"
)

const actionTimeout = 3 * time.Second

// ColorChoice is one color name and its canonical RGB value as reported by
// Ghostty. The name is the value users can select in a color editor.
type ColorChoice struct {
	Name  string
	Value string
}

// ThemeColors contains the color fields needed for a truthful terminal
// preview. Values are the canonical strings from the Ghostty theme source;
// an empty field means that the source did not define that field.
type ThemeColors struct {
	Background          string
	Foreground          string
	Cursor              string
	CursorText          string
	SelectionBackground string
	SelectionForeground string
	Palette             []string
}

// ThemePreview is a theme name together with colors resolved from Ghostty's
// own theme data. Available is false when the name is known but its source
// could not be located or parsed, so consumers can show an honest fallback
// instead of inventing a swatch.
type ThemePreview struct {
	Name      string
	Colors    ThemeColors
	Source    string
	Available bool
	Error     string
}

// ListThemes asks the installed Ghostty binary for its available theme names.
// The command includes both user-configured themes and bundled resources; an
// empty result is an error so callers can fall back to raw input explicitly.
func ListThemes(binary string) ([]string, error) {
	output, err := actionOutput(binary, "+list-themes", "--plain")
	if err != nil {
		return nil, fmt.Errorf("list Ghostty themes: %w", err)
	}
	return parseThemeList(output), nil
}

// ResolveTheme reads a named theme from the installed Ghostty resources or
// the user's Ghostty theme directory. It never invokes Ghostty with a config
// mutation and never writes a file. Theme names are resolved as exact file
// names because that is the format emitted by +list-themes.
func ResolveTheme(binary, name string) (ThemePreview, error) {
	preview := ThemePreview{Name: name}
	if strings.TrimSpace(name) == "" {
		preview.Error = "theme name is empty"
		return preview, fmt.Errorf("theme name is empty")
	}

	paths := themePaths(binary, name)
	var lastErr error
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		colors, err := parseThemeColors(data)
		if err != nil {
			lastErr = fmt.Errorf("parse theme %q: %w", name, err)
			continue
		}
		if !themeColorsUsable(colors) {
			lastErr = fmt.Errorf("theme %q does not define background and foreground", name)
			continue
		}
		preview.Colors = colors
		preview.Source = path
		preview.Available = true
		return preview, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("theme %q was not found", name)
	}
	preview.Error = lastErr.Error()
	return preview, lastErr
}

// ResolveThemes resolves the names returned by ListThemes. Names remain in
// the result even when a source file is unavailable, which lets the UI offer
// the complete authoritative inventory while labeling individual previews.
func ResolveThemes(binary string, names []string) []ThemePreview {
	previews := make([]ThemePreview, 0, len(names))
	for _, name := range names {
		preview, _ := ResolveTheme(binary, name)
		previews = append(previews, preview)
	}
	return previews
}

// ResolveDefaultTheme asks Ghostty to print its authoritative default
// configuration and extracts the colors used by the terminal preview. The
// empty name identifies the no-explicit-theme state in the appearance UI.
func ResolveDefaultTheme(binary string) (ThemePreview, error) {
	preview := ThemePreview{}
	output, err := actionOutput(binary, "+show-config", "--default", "--no-pager")
	if err != nil {
		preview.Error = fmt.Sprintf("show Ghostty default config: %v", err)
		return preview, fmt.Errorf("show Ghostty default config: %w", err)
	}
	colors, err := parseThemeColors(output)
	if err != nil {
		preview.Error = fmt.Sprintf("parse Ghostty default colors: %v", err)
		return preview, fmt.Errorf("parse Ghostty default colors: %w", err)
	}
	if !themeColorsUsable(colors) {
		preview.Error = "Ghostty default config does not define background and foreground"
		return preview, fmt.Errorf("%s", preview.Error)
	}
	preview.Colors = colors
	preview.Source = "Ghostty +show-config --default"
	preview.Available = true
	return preview, nil
}

func parseThemeColors(data []byte) (ThemeColors, error) {
	document := configdoc.Parse(data)
	colors := ThemeColors{}
	for _, node := range document.Nodes {
		if node.Kind != configdoc.AssignmentNode {
			continue
		}
		value := strings.TrimSpace(node.Value)
		switch node.Key {
		case "background":
			colors.Background = value
		case "foreground":
			colors.Foreground = value
		case "cursor-color":
			colors.Cursor = value
		case "cursor-text":
			colors.CursorText = value
		case "selection-background":
			colors.SelectionBackground = value
		case "selection-foreground":
			colors.SelectionForeground = value
		case "palette":
			paletteIndex, paletteValue, ok := strings.Cut(value, "=")
			if !ok {
				continue
			}
			paletteIndex = strings.TrimSpace(paletteIndex)
			paletteValue = strings.TrimSpace(paletteValue)
			if paletteIndex == "" || paletteValue == "" {
				continue
			}
			if len(colors.Palette) == 0 {
				colors.Palette = make([]string, 16)
			}
			index, err := strconv.Atoi(paletteIndex)
			if err != nil || index < 0 || index >= 16 {
				continue
			}
			colors.Palette[index] = paletteValue
		}
	}
	if !themeColorsHaveColorValues(colors) {
		return ThemeColors{}, fmt.Errorf("no usable color assignments")
	}
	return colors, nil
}

func themeColorsHaveColorValues(colors ThemeColors) bool {
	if colors.Background != "" || colors.Foreground != "" || colors.Cursor != "" || colors.SelectionBackground != "" {
		return true
	}
	for _, value := range colors.Palette {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func themeColorsUsable(colors ThemeColors) bool {
	return strings.TrimSpace(colors.Background) != "" && strings.TrimSpace(colors.Foreground) != ""
}

func themePaths(binary, name string) []string {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	paths := make([]string, 0, 12)
	addRoot := func(root string) {
		if root == "" {
			return
		}
		candidate := filepath.Clean(filepath.Join(root, name))
		// Theme names are emitted as file names. Reject traversal so an
		// unusual config value cannot turn a read-only resolver into a
		// filesystem escape hatch.
		relative, err := filepath.Rel(filepath.Clean(root), candidate)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return
		}
		for _, existing := range paths {
			if existing == candidate {
				return
			}
		}
		paths = append(paths, candidate)
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			addRoot(filepath.Join(xdg, "ghostty", "themes"))
		} else {
			addRoot(filepath.Join(home, ".config", "ghostty", "themes"))
		}
		addRoot(filepath.Join(home, "Library", "Application Support", "com.mitchellh.ghostty", "themes"))
	}

	if binary != "" {
		binaryPath, err := filepath.Abs(binary)
		if err == nil {
			binaryPaths := []string{binaryPath}
			if resolved, resolveErr := filepath.EvalSymlinks(binaryPath); resolveErr == nil && resolved != binaryPath {
				binaryPaths = append(binaryPaths, resolved)
			}
			for _, path := range binaryPaths {
				binaryDir := filepath.Dir(path)
				addRoot(filepath.Join(binaryDir, "..", "Resources", "ghostty", "themes"))
				addRoot(filepath.Join(binaryDir, "..", "share", "ghostty", "themes"))
				addRoot(filepath.Join(binaryDir, "..", "..", "share", "ghostty", "themes"))
				addRoot(filepath.Join(binaryDir, "..", "..", "Resources", "ghostty", "themes"))
			}
		}
	}
	addRoot("/usr/share/ghostty/themes")
	addRoot("/usr/local/share/ghostty/themes")
	return paths
}

// ListColors asks the installed Ghostty binary for its named X11 colors.
func ListColors(binary string) ([]ColorChoice, error) {
	output, err := actionOutput(binary, "+list-colors", "--plain")
	if err != nil {
		return nil, fmt.Errorf("list Ghostty colors: %w", err)
	}
	return parseColorList(output), nil
}

// ListFonts asks Ghostty for the installed font families. Ghostty prints a
// family heading followed by indented faces; family headings are the values
// accepted by font-family and are the least surprising choices to show in a
// config editor.
func ListFonts(binary string) ([]string, error) {
	output, err := actionOutput(binary, "+list-fonts")
	if err != nil {
		return nil, fmt.Errorf("list Ghostty fonts: %w", err)
	}
	return parseFontList(output), nil
}

// ListActions asks Ghostty for the action names accepted by keybind and
// command-palette-entry. Arguments remain a separate form field because they
// are action-specific and may be arbitrary strings.
func ListActions(binary string) ([]string, error) {
	output, err := actionOutput(binary, "+list-actions")
	if err != nil {
		return nil, fmt.Errorf("list Ghostty actions: %w", err)
	}
	return parseActionList(output), nil
}

func actionOutput(binary string, args ...string) ([]byte, error) {
	if strings.TrimSpace(binary) == "" {
		return nil, fmt.Errorf("Ghostty executable is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	output, err := command.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return output, nil
}

func parseThemeList(output []byte) []string {
	choices := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		// `+list-themes` annotates each row with `(config)` or `(resources)`.
		// Strip only that trailing source annotation, preserving spaces in the
		// actual theme name.
		if marker := strings.LastIndex(line, " ("); marker >= 0 && strings.HasSuffix(line, ")") {
			line = strings.TrimSpace(line[:marker])
		}
		if line == "" {
			continue
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		choices = append(choices, line)
	}
	return choices
}

func parseColorList(output []byte) []ColorChoice {
	choices := make([]ColorChoice, 0)
	seen := make(map[string]struct{})
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		separator := strings.Index(line, "=")
		if separator <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		if name == "" || value == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		choices = append(choices, ColorChoice{Name: name, Value: value})
	}
	return choices
}

func parseFontList(output []byte) []string {
	choices := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawLine := range strings.Split(string(output), "\n") {
		if rawLine == "" || strings.HasPrefix(rawLine, " ") || strings.HasPrefix(rawLine, "\t") {
			continue
		}
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "error:") {
			continue
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		choices = append(choices, line)
	}
	return choices
}

func parseActionList(output []byte) []string {
	choices := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "error:") {
			continue
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		choices = append(choices, line)
	}
	return choices
}
