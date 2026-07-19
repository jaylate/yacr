package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func setup(cfg Config) error {
	rootfs, err := filepath.Abs(cfg.RootFS)
	if err != nil {
		return fmt.Errorf("abs rootfs: %w", err)
	}

	// Best-effort: isolate mount propagation.
	_ = unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, "")

	// Bind-mount host /proc into the rootfs before pivot/chroot. Kernels that
	// emit "VFS: Mount too revealing" reject a fresh proc mount in a userns
	// unless a proc mount already exists in the mount tree (rootless path).
	if err := prepareProc(rootfs); err != nil {
		return err
	}

	// Populate /dev while host device nodes are still reachable (rootless:
	// bind-mount instead of mknod).
	if err := prepareDev(rootfs); err != nil {
		return err
	}

	// Prefer pivot_root; fall back to chroot when bind/pivot is denied.
	if err := tryPivotRoot(rootfs); err != nil {
		if ferr := chrootFallback(rootfs); ferr != nil {
			return fmt.Errorf("enter rootfs: pivot_root: %v; chroot: %w", err, ferr)
		}
	}

	// Replace the bind-mounted proc with a pid-ns-local instance when allowed.
	remountProc()

	// pts/shm/mqueue/sys + Podman-compatible /dev symlinks.
	finishDev()

	if cfg.Hostname != "" {
		if err := unix.Sethostname([]byte(cfg.Hostname)); err != nil {
			return fmt.Errorf("sethostname: %w", err)
		}
	}
	return nil
}

// prepareProc bind-mounts the caller's /proc into rootfs/proc so rootless
// setups can later remount a real proc after pivot_root/chroot.
func prepareProc(rootfs string) error {
	target := filepath.Join(rootfs, "proc")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("mkdir proc: %w", err)
	}
	if err := unix.Mount("/proc", target, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		// Soft: some environments already have a usable proc or deny the bind;
		// remountProc / later mounts may still succeed.
		return nil
	}
	return nil
}

// remountProc mounts a fresh proc on /proc (pid-namespace aware).
// If the kernel denies it, the bind from prepareProc is left in place.
func remountProc() {
	const flags = uintptr(unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_NODEV)
	_ = unix.Mount("proc", "/proc", "proc", flags, "")
}

// tryPivotRoot bind-mounts rootfs (required by pivot_root) then pivots.
func tryPivotRoot(rootfs string) error {
	if err := ensureRootfsMount(rootfs); err != nil {
		return err
	}
	return pivotRoot(rootfs)
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
