// Package bootstrap implements the container-side process entrypoint.
//
// The parent process re-execs the yacr binary into the target namespaces and
// invokes this package when [IsBootstrap] is true. This mirrors runc's
// internal init path: it is not a user-facing CLI command.
package bootstrap

import (
	"fmt"
	"os"
	"syscall"
)

const (
	// ArgName is the argv token used when re-execing for bootstrap.
	ArgName = "init"
	// EnvMarker must be set so accidental "yacr init" from a shell does nothing.
	EnvMarker = "_YACR_BOOTSTRAP"
)

// IsBootstrap reports whether this process is the container bootstrap re-exec.
func IsBootstrap() bool {
	if os.Getenv(EnvMarker) != "1" {
		return false
	}
	return len(os.Args) > 1 && os.Args[1] == ArgName
}

// Run sets up the container rootfs and execs the user process.
// It does not return on success.
func Run() int {
	hostname, rootfs, command, args, err := parseArgs(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := setup(hostname, rootfs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	argv := append([]string{command}, args...)
	if err := syscall.Exec(command, argv, []string{}); err != nil {
		fmt.Fprintf(os.Stderr, "execve: %v\n", err)
		return 1
	}
	return 1
}

func parseArgs(args []string) (hostname, rootfs, command string, cmdArgs []string, err error) {
	hostname = "container"
	rootfs = "rootfs"

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++
			break
		}
		switch arg {
		case "--hostname":
			if i+1 >= len(args) {
				return "", "", "", nil, fmt.Errorf("missing value for --hostname")
			}
			hostname = args[i+1]
			i += 2
		case "--rootfs":
			if i+1 >= len(args) {
				return "", "", "", nil, fmt.Errorf("missing value for --rootfs")
			}
			rootfs = args[i+1]
			i += 2
		default:
			return "", "", "", nil, fmt.Errorf("unknown bootstrap flag %q", arg)
		}
	}

	if i >= len(args) {
		return "", "", "", nil, fmt.Errorf("usage: yacr %s [--hostname name] [--rootfs path] -- <command> [args...]", ArgName)
	}

	return hostname, rootfs, args[i], args[i+1:], nil
}
