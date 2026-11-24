package app

import (
	"fmt"
	"os"
	"strconv"

	"github.com/thongntit/ghostty/settings/internal/tui"
)

// hostReloadNotifier is disabled for standalone editor launches. When the app
// host supplies a PID, malformed values are reported in the editor status
// after a successful save instead of failing the save itself.
func hostReloadNotifier() tui.ReloadFunc {
	value := os.Getenv("GHOSTTY_SETTINGS_RELOAD_PID")
	if value == "" {
		return nil
	}
	pid, err := strconv.ParseInt(value, 10, 32)
	if err != nil || pid <= 1 {
		return func() error { return fmt.Errorf("GHOSTTY_SETTINGS_RELOAD_PID must be a positive process ID") }
	}
	return func() error { return signalGhostty(int(pid)) }
}
