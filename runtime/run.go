package runtime

import (
	"fmt"
	"os"

	"github.com/jaylate/yacr/runtime/resources"
)

func RunWithCgroups(command string, commandArgs []string, cfg *ContainerConfig, limits resources.ResourceLimits) error {
	mgr, err := resources.NewCgroupsManager("", limits)

	if err != nil {
		fmt.Fprintf(os.Stderr, "NewCgroupsManager error: %v\n", err)
		return err
	}

	executor := NewLinuxExecutorWithCgroups(cfg, mgr)

	err = executor.Execute(command, commandArgs...)

	mgr.DestroyRuntime()
	return err
}

func Run(command string, commandArgs []string, cfg *ContainerConfig) (err error) {
	if cfg == nil {
		return RunWithExecutor(command, commandArgs, NewLinuxExecutor(nil))
	}

	hasLimits := cfg.Limits.MemoryBytes != 0 || cfg.Limits.CPUCores != 0 || cfg.Limits.PIDsMax != 0
	if !hasLimits {
		return RunWithExecutor(command, commandArgs, NewLinuxExecutor(cfg))
	}

	var mgr *resources.CgroupsManager
	mgr, err = resources.NewCgroupsManager("", cfg.Limits)
	if err != nil {
		return err
	}
	defer func() {
		if destroyErr := mgr.DestroyRuntime(); destroyErr != nil && err == nil {
			err = destroyErr
		}
	}()

	executor := NewLinuxExecutorWithCgroups(cfg, mgr)
	err = RunWithExecutor(command, commandArgs, executor)
	return
}

func RunWithExecutor(command string, commandArgs []string, executor ProcessExecutor) error {
	return executor.Execute(command, commandArgs...)
}
