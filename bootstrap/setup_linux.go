package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// DefaultMounts are the temporary built-in mounts applied after pivot_root.
// They are placeholders until OCI mounts[] is wired in.
// sysfs and devpts are best-effort (often fail in user namespaces).
var DefaultMounts = []struct {
	Source string
	Target string
	FSType string
	Flags  uintptr
	Data   string
	// Optional marks mounts that may fail (e.g. in userns); errors are ignored.
	Optional bool
}{
	{Source: "proc", Target: "/proc", FSType: "proc"},
	{Source: "tmpfs", Target: "/dev", FSType: "tmpfs", Data: "mode=755"},
	{Source: "sysfs", Target: "/sys", FSType: "sysfs", Optional: true},
	{Source: "devpts", Target: "/dev/pts", FSType: "devpts", Data: "newinstance,ptmxmode=0666,mode=0620", Optional: true},
}

func setup(cfg Config) error {
	rootfs, err := filepath.Abs(cfg.RootFS)
	if err != nil {
		return fmt.Errorf("abs rootfs: %w", err)
	}

	if err := ensureRootfsMount(rootfs); err != nil {
		return err
	}

	// Make mount propagation private so container mounts stay isolated.
	// Best-effort: may fail in some restricted environments.
	_ = unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, "")

	if err := pivotRoot(rootfs); err != nil {
		// Fall back to chroot so tests/dev still work without full privileges.
		if ferr := chrootFallback(rootfs); ferr != nil {
			return fmt.Errorf("pivot_root: %w (chroot fallback: %v)", err, ferr)
		}
	}
	if err := applyDefaultMounts(); err != nil {
		return err
	}

	if cfg.Hostname != "" {
		if err := unix.Sethostname([]byte(cfg.Hostname)); err != nil {
			return fmt.Errorf("sethostname: %w", err)
		}
	}
	return nil
}

func ensureRootfsMount(rootfs string) error {
	mounted, err := isMountPoint(rootfs)
	if err != nil {
		return fmt.Errorf("check mountpoint %s: %w", rootfs, err)
	}
	if mounted {
		return nil
	}
	if err := unix.Mount(rootfs, rootfs, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return fmt.Errorf("bind-mount rootfs onto itself: %w", err)
	}
	return nil
}

func isMountPoint(path string) (bool, error) {
	var st, pst unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return false, err
	}
	parent := filepath.Dir(path)
	if err := unix.Stat(parent, &pst); err != nil {
		return false, err
	}
	// Different device number means path is a mount point.
	// Note: same-device bind mounts are not detected; ensureRootfsMount
	// will bind again, which is harmless for our use.
	return st.Dev != pst.Dev, nil
}

func pivotRoot(rootfs string) error {
	putOld := filepath.Join(rootfs, ".oldroot")
	if err := os.Mkdir(putOld, 0o700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("mkdir put_old: %w", err)
	}

	if err := unix.PivotRoot(rootfs, putOld); err != nil {
		_ = os.Remove(putOld)
		return fmt.Errorf("pivot_root: %w", err)
	}
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}

	putOldInNew := "/.oldroot"
	if err := unix.Unmount(putOldInNew, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("umount put_old: %w", err)
	}
	if err := os.Remove(putOldInNew); err != nil {
		return fmt.Errorf("remove put_old: %w", err)
	}
	return nil
}

func chrootFallback(rootfs string) error {
	if err := unix.Chroot(rootfs); err != nil {
		return fmt.Errorf("chroot %s: %w", rootfs, err)
	}
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}
	return nil
}

func applyDefaultMounts() error {
	for _, m := range DefaultMounts {
		if err := os.MkdirAll(m.Target, 0o755); err != nil {
			if m.Optional {
				continue
			}
			return fmt.Errorf("mkdir %s: %w", m.Target, err)
		}
		if err := unix.Mount(m.Source, m.Target, m.FSType, m.Flags, m.Data); err != nil {
			if m.Optional {
				continue
			}
			return fmt.Errorf("mount %s: %w", m.Target, err)
		}
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
// The parent must also set ConfigFDEnv to the ExtraFiles fd of the config pipe.
func Env() []string {
	return append(os.Environ(), EnvMarker+"=1")
}
