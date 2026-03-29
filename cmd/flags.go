package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrNoSubcommand = errors.New("No subcommand provided")
	ErrInvalidUsage = errors.New("Invalid usage")
)

type RunConfig struct {
	Command  string
	Args     []string
	RootFS   string
	Hostname string
	Memory   string
	CPUCores string
	PIDsMax  string
	Help     bool
}

func Usage(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintln(w, `Usage: yacr [options] run <command> [arguments]

Run a command in a containerized environment.

Options:
  -h, --help     Show this help message
  --hostname     Set container hostname (default: container)
  --rootfs       Set root filesystem path (default: rootfs)
  --memory       Set memory limit (e.g., 100M, 1G, max)
  --cpus         Set CPU limit (e.g., 0.5, 2, max)
  --pids         Set PID limit (e.g., 50, max)

Commands:
  run            Run a command in a new environment

Examples:
  yacr run /bin/sh
  yacr --hostname myhost --memory 512M --cpus 2 run /bin/sh -l`)
}

func ParseArgs(args []string) (*RunConfig, error) {
	return ParseArgsWithWriter(args, nil)
}

func ParseArgsWithWriter(args []string, w io.Writer) (*RunConfig, error) {
	if w == nil {
		w = os.Stderr
	}

	cfg := &RunConfig{
		RootFS:   "rootfs",
		Hostname: "container",
	}

	i := 1
	for i < len(args) {
		arg := args[i]

		if arg == "--" {
			i++
			break
		}

		if arg == "--help" || arg == "-h" {
			cfg.Help = true
			i++
			continue
		}

		if arg == "--hostname" {
			if i+1 >= len(args) || isMissingFlagValue(args[i+1]) {
				return nil, fmt.Errorf("%w: Missing value for --hostname", ErrInvalidUsage)
			}
			cfg.Hostname = args[i+1]
			i += 2
			continue
		}

		if arg == "--rootfs" {
			if i+1 >= len(args) || isMissingFlagValue(args[i+1]) {
				return nil, fmt.Errorf("%w: Missing value for --rootfs", ErrInvalidUsage)
			}
			cfg.RootFS = args[i+1]
			i += 2
			continue
		}

		if arg == "--memory" {
			if i+1 >= len(args) || isMissingFlagValue(args[i+1]) {
				return nil, fmt.Errorf("%w: Missing value for --memory", ErrInvalidUsage)
			}
			cfg.Memory = args[i+1]
			i += 2
			continue
		}

		if arg == "--cpus" {
			if i+1 >= len(args) || isMissingFlagValue(args[i+1]) {
				return nil, fmt.Errorf("%w: Missing value for --cpus", ErrInvalidUsage)
			}
			cfg.CPUCores = args[i+1]
			i += 2
			continue
		}

		if arg == "--pids" {
			if i+1 >= len(args) || isMissingFlagValue(args[i+1]) {
				return nil, fmt.Errorf("%w: Missing value for --pids", ErrInvalidUsage)
			}
			cfg.PIDsMax = args[i+1]
			i += 2
			continue
		}

		break
	}

	if cfg.Help {
		Usage(w)
		return cfg, nil
	}

	if i >= len(args) {
		return nil, ErrNoSubcommand
	}

	subcommand := args[i]
	i++

	if subcommand == "run" {
		if i >= len(args) {
			return nil, ErrInvalidUsage
		}

		cfg.Command = args[i]
		cfg.Args = args[i+1:]
		return cfg, nil
	}

	return nil, fmt.Errorf("%w: Unknown subcommand %q", ErrInvalidUsage, subcommand)
}

func isMissingFlagValue(value string) bool {
	switch value {
	case "run", "--", "--help", "-h", "--hostname", "--rootfs", "--memory", "--cpus", "--pids":
		return true
	default:
		return false
	}
}
