// Package bootstrap implements the container-side process entrypoint.
//
// The parent process re-execs the yacr binary into the target namespaces and
// invokes this package when [IsBootstrap] is true. This mirrors runc's
// internal init path: it is not a user-facing CLI command.
//
// Configuration is passed as JSON over a pipe whose fd is in [ConfigFDEnv].
// The parent should pass only [ArgName] as argv after argv0, set [EnvMarker],
// attach the config pipe via ExtraFiles, and set ConfigFDEnv to that fd number.
package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
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
	cfg, err := readConfigFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := setup(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if cfg.GID != 0 {
		if err := unix.Setgid(int(cfg.GID)); err != nil {
			fmt.Fprintf(os.Stderr, "setgid: %v\n", err)
			return 1
		}
	}
	if cfg.UID != 0 {
		if err := unix.Setuid(int(cfg.UID)); err != nil {
			fmt.Fprintf(os.Stderr, "setuid: %v\n", err)
			return 1
		}
	}

	if err := unix.Chdir(cfg.WorkDir); err != nil {
		fmt.Fprintf(os.Stderr, "chdir %s: %v\n", cfg.WorkDir, err)
		return 1
	}

	env := cfg.Env
	if env == nil {
		env = []string{}
	}

	argv := append([]string{cfg.Command}, cfg.Args...)
	if err := syscall.Exec(cfg.Command, argv, env); err != nil {
		fmt.Fprintf(os.Stderr, "execve: %v\n", err)
		return 1
	}
	return 1
}

func readConfigFromEnv() (Config, error) {
	fdStr := os.Getenv(ConfigFDEnv)
	if fdStr == "" {
		return Config{}, fmt.Errorf("missing %s", ConfigFDEnv)
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", ConfigFDEnv, err)
	}
	f := os.NewFile(uintptr(fd), "bootstrap-config")
	if f == nil {
		return Config{}, fmt.Errorf("invalid %s fd %d", ConfigFDEnv, fd)
	}
	defer f.Close()
	return ReadConfig(f)
}
