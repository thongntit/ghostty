package tui

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/thongntit/ghostty/settings/internal/schema"
)

func valueHint(option schema.Option) string {
	if len(option.Values) > 0 {
		separator := " | "
		if option.Multiple {
			separator = ", "
		}
		return strings.Join(option.Values, separator)
	}
	switch option.Kind {
	case schema.KindBoolean:
		return "true or false"
	case schema.KindNumber:
		return "finite number"
	case schema.KindColor:
		return "#RRGGBB or named color"
	case schema.KindPath:
		return "path"
	case schema.KindDuration:
		return "0, 250ms, or 1s 200ms"
	case schema.KindEnum:
		return "choose one"
	default:
		switch option.ControlModel() {
		case schema.ControlMetric:
			return "signed adjustment, e.g. 1px or 5%"
		case schema.ControlResource:
			return "non-negative integer or unlimited"
		case schema.ControlPair:
			return "structured key=value"
		default:
			return "enter a value"
		}
	}
}

func defaultColorChoices() []ColorChoice {
	return []ColorChoice{
		{Name: "Black", Value: "#000000"},
		{Name: "White", Value: "#ffffff"},
		{Name: "Red", Value: "#f38ba8"},
		{Name: "Orange", Value: "#fab387"},
		{Name: "Yellow", Value: "#f9e2af"},
		{Name: "Green", Value: "#a6e3a1"},
		{Name: "Cyan", Value: "#94e2d5"},
		{Name: "Blue", Value: "#89b4fa"},
		{Name: "Purple", Value: "#cba6f7"},
		{Name: "Ghostty dark", Value: "#282c34"},
		{Name: "cell-foreground", Value: "cell-foreground"},
		{Name: "cell-background", Value: "cell-background"},
	}
}

func friendlyOptionName(key string) string {
	initialisms := map[string]string{
		"gtk":   "GTK",
		"macos": "macOS",
		"ssh":   "SSH",
		"url":   "URL",
		"vt":    "VT",
		"x11":   "X11",
	}
	parts := strings.Split(key, "-")
	for index, part := range parts {
		if replacement, ok := initialisms[part]; ok {
			parts[index] = replacement
			continue
		}
		runes := []rune(part)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		parts[index] = string(runes)
	}
	return strings.Join(parts, " ")
}

func numberStep(option schema.Option) float64 {
	if option.Step != nil && *option.Step > 0 && !math.IsNaN(*option.Step) && !math.IsInf(*option.Step, 0) {
		return *option.Step
	}
	return 1
}

func numberStepLabel(option schema.Option) string {
	return strconv.FormatFloat(numberStep(option), 'f', -1, 64)
}

// steppedNumberValue applies a numeric step using decimal arithmetic. Float64
// addition is intentionally avoided here because common config steps such as
// 0.05 cannot be represented exactly in binary (0.95 - 0.05 can otherwise
// become 0.8999999999999999).
func steppedNumberValue(value string, step float64, delta int) (string, bool) {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return "", false
	}
	if math.IsNaN(step) || math.IsInf(step, 0) {
		return "", false
	}
	if adjusted := number + float64(delta)*step; math.IsInf(adjusted, 0) {
		return "", false
	}

	// Convert the parsed values back to their shortest decimal forms before
	// building the exact rationals. This preserves the user's meaningful
	// decimal value without importing the float64 representation error.
	numberText := strconv.FormatFloat(number, 'f', -1, 64)
	stepText := strconv.FormatFloat(step, 'f', -1, 64)
	numberRat, ok := new(big.Rat).SetString(numberText)
	if !ok {
		return "", false
	}
	stepRat, ok := new(big.Rat).SetString(stepText)
	if !ok {
		return "", false
	}
	adjustment := new(big.Rat).Mul(stepRat, new(big.Rat).SetInt64(int64(delta)))
	result := new(big.Rat).Add(numberRat, adjustment)

	// Both inputs are finite decimal strings, so this precision is enough to
	// render an exact result for the supported decimal steps. Trim trailing
	// zeroes below to keep values like 0.90 user-friendly as 0.9.
	precision := maxInt(decimalPlaces(numberText), decimalPlaces(stepText))
	return trimDecimalZeros(result.FloatString(precision)), true
}

func decimalPlaces(value string) int {
	separator := strings.IndexByte(value, '.')
	if separator < 0 {
		return 0
	}
	return len(value) - separator - 1
}

func trimDecimalZeros(value string) string {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(value, "0")
		value = strings.TrimRight(value, ".")
	}
	if value == "-0" {
		return "0"
	}
	return value
}

func (m Model) editorLabel(option schema.Option) string {
	switch option.ControlModel() {
	case schema.ControlMultiSelect:
		return "multi-select list"
	case schema.ControlTheme:
		if len(m.themeChoices) == 0 {
			return "theme chooser + raw fallback"
		}
		return "theme chooser"
	case schema.ControlColor:
		return "color chooser"
	case schema.ControlFont:
		if len(m.fontChoices) == 0 {
			return "font chooser + raw fallback"
		}
		return "font chooser"
	case schema.ControlFontStyle:
		return "font style chooser"
	case schema.ControlSelect:
		return "enum chooser"
	case schema.ControlNumber:
		return "numeric stepper"
	case schema.ControlDuration:
		return "duration picker"
	case schema.ControlMetric:
		return "metric stepper"
	case schema.ControlResource:
		return "limit stepper"
	case schema.ControlPath:
		return "path browser"
	case schema.ControlToggle:
		return "on/off toggle"
	case schema.ControlPair:
		return "structured pair form"
	case schema.ControlKeybind:
		return "keybinding list"
	case schema.ControlCommandPalette:
		return "command palette form"
	case schema.ControlCommand:
		return "command input"
	case schema.ControlRepeatable:
		return "repeatable list"
	default:
		return "raw text (open-ended)"
	}
}

func optionGroup(option schema.Option) string {
	if option.Group != "" {
		return option.Group
	}
	return schema.GroupForKey(option.Key)
}

func (m Model) optionStatus(option schema.Option) string {
	assignments := m.optionAssignments(option.Key)
	if len(assignments) == 0 {
		return "not set"
	}
	if option.Repeatable() {
		return fmt.Sprintf("%d value(s)", len(assignments))
	}
	effective := assignments[len(assignments)-1]
	if effective.Empty {
		return "default"
	}
	if len(assignments) > 1 {
		return fmt.Sprintf("%s (%d assignments)", effective.Value, len(assignments))
	}
	return effective.Value
}

func (m Model) draftOptionStatus(option schema.Option) string {
	return m.optionStatus(option)
}

func (m Model) optionSource(option schema.Option) string {
	if m.graph == nil {
		return ""
	}
	assignments := m.optionAssignments(option.Key)
	if len(assignments) == 0 {
		return "No assignment in loaded config graph"
	}
	effective := assignments[len(assignments)-1]
	return fmt.Sprintf("Effective source: %s:%d", effective.Path, effective.Line)
}
