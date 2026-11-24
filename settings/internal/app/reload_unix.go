//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package app

import "syscall"

func signalGhostty(pid int) error { return syscall.Kill(pid, syscall.SIGUSR2) }
