//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package app

import "fmt"

func signalGhostty(pid int) error {
	return fmt.Errorf("SIGUSR2 notification is unsupported on this platform")
}
