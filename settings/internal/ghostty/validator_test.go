package ghostty

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateConfigPassesCandidateAsOneArgument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX shell")
	}
	binary := filepath.Join(t.TempDir(), "ghostty")
	candidate := filepath.Join(t.TempDir(), "candidate config.ghostty")
	t.Setenv("EXPECTED_CANDIDATE", candidate)
	script := `#!/bin/sh
if [ "$#" -ne 2 ] || [ "$1" != "+validate-config" ] || [ "$2" != "--config-file=$EXPECTED_CANDIDATE" ]; then
    echo "unexpected validator arguments" >&2
    exit 2
fi
if [ "$REJECT_CANDIDATE" = 1 ]; then
    echo "unrelated startup log" >&2
    echo "invalid setting value"
    exit 1
fi
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REJECT_CANDIDATE", "0")
	if err := ValidateConfig(context.Background(), binary, candidate); err != nil {
		t.Fatalf("valid candidate rejected: %v", err)
	}
	t.Setenv("REJECT_CANDIDATE", "1")
	err := ValidateConfig(context.Background(), binary, candidate)
	if err == nil || !strings.Contains(err.Error(), "invalid setting value") ||
		strings.Contains(err.Error(), "unrelated startup log") {
		t.Fatalf("expected a clear validation diagnostic, got %v", err)
	}
}
