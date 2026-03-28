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

func Run(command string, commandArgs []string, cfg *ContainerConfig) error {
	mgr, err := resources.NewCgroupsManager("", cfg.Limits)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewCgroupsManager error: %v\n", err)
		return err
	}
	executor := NewLinuxExecutorWithCgroups(cfg, mgr)
	err = RunWithExecutor(command, commandArgs, executor)

	mgr.DestroyRuntime()
	return err
}

func RunWithExecutor(command string, commandArgs []string, executor ProcessExecutor) error {
	return executor.Execute(command, commandArgs...)
}
