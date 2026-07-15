package bootstrap

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func setup(hostname, rootfs string) error {
	if err := unix.Sethostname([]byte(hostname)); err != nil {
		return fmt.Errorf("sethostname: %w", err)
	}

	if err := unix.Chroot(rootfs); err != nil {
		return fmt.Errorf("chroot %s: %w", rootfs, err)
	}
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}

	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}
	return nil
}

// Self returns the path of the current executable for re-exec.
func Self() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return path, nil
}

// Env returns the environment for a bootstrap child process.
func Env() []string {
	return append(os.Environ(), EnvMarker+"=1")
}

// CommandArgs builds argv for a bootstrap re-exec, excluding argv0.
func CommandArgs(hostname, rootfs, command string, args []string) []string {
	out := []string{
		ArgName,
		"--hostname", hostname,
		"--rootfs", rootfs,
		"--",
		command,
	}
	return append(out, args...)
}
