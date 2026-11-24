package schema

import (
	"bytes"
	_ "embed"
)

// optionsJSON is a fallback for standalone builds. Fork builds regenerate it
// from the Ghostty executable being packaged.
//
//go:embed options.json
var optionsJSON []byte

// EmbeddedCatalog loads the catalog shipped with the application.
func EmbeddedCatalog() (Catalog, error) {
	return Load(bytes.NewReader(optionsJSON))
}
