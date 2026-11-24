package schema

import "testing"

func TestControlForCoversFriendlyScalarEditors(t *testing.T) {
	tests := []struct {
		key     string
		kind    ValueKind
		control ControlKind
	}{
		{key: "confirm-close-surface", kind: KindBoolean, control: ControlToggle},
		{key: "window-theme", kind: KindEnum, control: ControlSelect},
		{key: "adjust-font-baseline", kind: KindString, control: ControlMetric},
		{key: "scrollback-limit-lines", kind: KindString, control: ControlResource},
		{key: "mouse-scroll-multiplier", kind: KindString, control: ControlPair},
		{key: "font-style", kind: KindString, control: ControlFontStyle},
		{key: "font-family-bold", kind: KindString, control: ControlFont},
	}
	for _, test := range tests {
		option := Option{Key: test.key, Kind: test.kind}
		if got := option.ControlModel(); got != test.control {
			t.Errorf("ControlModel(%q) = %q, want %q", test.key, got, test.control)
		}
	}
}

func TestEmbeddedCatalogHasReviewedControlAndGroupMetadata(t *testing.T) {
	catalog, err := EmbeddedCatalog()
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	for _, option := range catalog.Options {
		if option.ControlModel() == "" {
			t.Errorf("%s has no control model", option.Key)
		}
		if option.Group == "" {
			t.Errorf("%s has no navigation group", option.Key)
		}
	}
}

func TestControlValidationAcceptsMetricAndResourceValues(t *testing.T) {
	metric := Option{Key: "adjust-font-baseline", Kind: KindString, Control: ControlMetric}
	for _, value := range []string{"0px", "-1.5px", "5%"} {
		if err := metric.Validate(value); err != nil {
			t.Errorf("metric %q rejected: %v", value, err)
		}
	}
	resource := Option{Key: "scrollback-limit-lines", Kind: KindString, Control: ControlResource}
	for _, value := range []string{"unlimited", "1000"} {
		if err := resource.Validate(value); err != nil {
			t.Errorf("resource %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"-1px", "five", "1.5"} {
		if err := resource.Validate(value); err == nil {
			t.Errorf("invalid resource %q accepted", value)
		}
	}
}
