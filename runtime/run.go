package runtime

import (
	"github.com/jaylate/yacr/runtime/resources"
)

func RunWithCgroups(command string, commandArgs []string, cfg *ContainerConfig, limits resources.ResourceLimits) error {
	if cfg == nil {
		cfg = DefaultContainerConfig()
	}

	rt, err := CreateRuntime(limits)
	if err != nil {
		return err
	}
	defer rt.DeleteRuntime()

	container, err := rt.CreateContainer(*cfg)
	if err != nil {
		return err
	}

	return container.StartContainer(command, commandArgs...)
}

func Run(command string, commandArgs []string, cfg *ContainerConfig) (err error) {
	if cfg == nil {
		cfg = DefaultContainerConfig()
	}

	rt, err := CreateRuntime(cfg.Limits)
	if err != nil {
		return err
	}
	defer rt.DeleteRuntime()

	container, err := rt.CreateContainer(*cfg)
	if err != nil {
		return err
	}

	return container.StartContainer(command, commandArgs...)
}
