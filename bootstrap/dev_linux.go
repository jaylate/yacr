package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// hostDevices are bound from the host into the container /dev before
// pivot_root/chroot so rootless (no mknod) setups still get a Podman-like /dev.
var hostDevices = []struct {
	Name   string
	Source string // host path; empty means Name under /dev
}{
	{Name: "null"},
	{Name: "zero"},
	{Name: "full"},
	{Name: "random"},
	{Name: "urandom"},
	{Name: "tty"},
	{Name: "console", Source: "/dev/console"}, // falls back to /dev/tty if missing
}

// prepareDev mounts a tmpfs at rootfs/dev and bind-mounts standard devices
// from the host. Must run before pivot_root/chroot while host /dev is visible.
func prepareDev(rootfs string) error {
	dev := filepath.Join(rootfs, "dev")
	if err := os.MkdirAll(dev, 0o755); err != nil {
		return fmt.Errorf("mkdir dev: %w", err)
	}
	if err := unix.Mount("tmpfs", dev, "tmpfs", unix.MS_NOSUID|unix.MS_STRICTATIME, "mode=755"); err != nil && err != unix.EBUSY {
		return fmt.Errorf("mount /dev tmpfs: %w", err)
	}

	for _, d := range []string{"pts", "shm", "mqueue"} {
		if err := os.MkdirAll(filepath.Join(dev, d), 0o755); err != nil {
			return fmt.Errorf("mkdir /dev/%s: %w", d, err)
		}
	}

	for _, d := range hostDevices {
		src := d.Source
		if src == "" {
			src = filepath.Join("/dev", d.Name)
		}
		dst := filepath.Join(dev, d.Name)
		if d.Name == "console" {
			bindHostDeviceWithFallback(src, "/dev/tty", dst)
			continue
		}
		_ = bindHostDevice(src, dst)
	}
	return nil
}

func bindHostDevice(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	_ = f.Close()
	if err := unix.Mount(src, dst, "", unix.MS_BIND, ""); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

func bindHostDeviceWithFallback(primary, fallback, dst string) {
	if err := bindHostDevice(primary, dst); err == nil {
		return
	}
	_ = bindHostDevice(fallback, dst)
}

// finishDev sets up /dev submounts and Podman-compatible symlinks after the
// container root is entered.
func finishDev() {
	_ = mountOptional("devpts", "/dev/pts", "devpts", unix.MS_NOSUID|unix.MS_NOEXEC,
		"newinstance,ptmxmode=0666,mode=0620")
	_ = mountOptional("tmpfs", "/dev/shm", "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC,
		"mode=1777,size=65536k")
	_ = mountOptional("mqueue", "/dev/mqueue", "mqueue", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "")
	_ = mountOptional("sysfs", "/sys", "sysfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC|unix.MS_RDONLY, "")

	// ptmx → pts/ptmx (create after devpts is mounted).
	_ = os.Remove("/dev/ptmx")
	_ = os.Symlink("pts/ptmx", "/dev/ptmx")

	_ = os.Remove("/dev/fd")
	_ = os.Symlink("/proc/self/fd", "/dev/fd")
	_ = os.Remove("/dev/stdin")
	_ = os.Symlink("fd/0", "/dev/stdin")
	_ = os.Remove("/dev/stdout")
	_ = os.Symlink("fd/1", "/dev/stdout")
	_ = os.Remove("/dev/stderr")
	_ = os.Symlink("fd/2", "/dev/stderr")
	_ = os.Remove("/dev/core")
	_ = os.Symlink("/proc/kcore", "/dev/core")
}

func mountOptional(source, target, fstype string, flags uintptr, data string) error {
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	if err := unix.Mount(source, target, fstype, flags, data); err != nil {
		if err == unix.EBUSY {
			return nil
		}
		return err
	}
	return nil
}
