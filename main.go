package main

import (
	"fmt"
	"io"
	"os"

	"github.com/jaylate/yacr/cmd"
	"github.com/jaylate/yacr/runtime"
	"github.com/jaylate/yacr/runtime/resources"
)

type runtimeRunner func(command string, args []string, cfg *runtime.ContainerConfig) error

func main() {
	os.Exit(run(os.Args, os.Stderr, runtime.Run))
}

func run(args []string, stderr io.Writer, runner runtimeRunner) int {
	cfg, err := cmd.ParseArgsWithWriter(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		cmd.Usage(stderr)
		return 1
	}

	if cfg != nil && cfg.Help {
		return 0
	}

	if cfg == nil {
		fmt.Fprintln(stderr, "Invalid configuration")
		return 1
	}

	parser := resources.NewLimitParser()
	limits, err := parser.ParseConfig(resources.LimitsConfig{
		Memory:   cfg.Memory,
		CPUCores: cfg.CPUCores,
		PIDsMax:  cfg.PIDsMax,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	containerCfg := &runtime.ContainerConfig{
		RootFS:   cfg.RootFS,
		Hostname: cfg.Hostname,
		Limits:   limits,
	}

	if err := runner(cfg.Command, cfg.Args, containerCfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}
