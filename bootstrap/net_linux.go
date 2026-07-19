package bootstrap

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// setupLoopback brings up the loopback interface in the container network
// namespace. Required after CLONE_NEWNET; without it nothing can bind to 127.0.0.1.
func setupLoopback() error {
	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_IP)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}
	defer unix.Close(sock)

	ifr, err := unix.NewIfreq("lo")
	if err != nil {
		return fmt.Errorf("ifreq lo: %w", err)
	}
	if err := unix.IoctlIfreq(sock, unix.SIOCGIFFLAGS, ifr); err != nil {
		return fmt.Errorf("SIOCGIFFLAGS lo: %w", err)
	}
	ifr.SetUint16(ifr.Uint16() | unix.IFF_UP)
	if err := unix.IoctlIfreq(sock, unix.SIOCSIFFLAGS, ifr); err != nil {
		return fmt.Errorf("SIOCSIFFLAGS lo: %w", err)
	}
	return nil
}
