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

	limits, err := parseLimits(cfg)
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

func parseLimits(cfg *cmd.RunConfig) (resources.ResourceLimits, error) {
	limits := resources.ResourceLimits{}

	if cfg.Memory != "" {
		memoryBytes, ok := resources.ParseMemoryString(cfg.Memory)
		if !ok {
			return limits, fmt.Errorf("invalid memory value: %s (use format like 100M, 1G, or max)", cfg.Memory)
		}
		limits.MemoryBytes = memoryBytes
	}

	if cfg.CPUCores != "" {
		cpuStr, ok := resources.ParseCPUString(cfg.CPUCores)
		if !ok {
			return limits, fmt.Errorf("invalid CPU value: %s (use format like 0.5, 2, or max)", cfg.CPUCores)
		}
		// ParseCPUString returns "100000 100000" for "1.0", need to extract the quota
		var quota int
		_, err := fmt.Sscanf(cpuStr, "%d", &quota)
		if err != nil {
			return limits, fmt.Errorf("invalid CPU value: %s", cfg.CPUCores)
		}
		limits.CPUCores = float64(quota) / 100000
	}

	if cfg.PIDsMax != "" {
		pidsMax, ok := resources.ParsePIDsString(cfg.PIDsMax)
		if !ok {
			return limits, fmt.Errorf("invalid PIDs value: %s (use format like 50 or max)", cfg.PIDsMax)
		}
		limits.PIDsMax = pidsMax
	}

	return limits, nil
}
