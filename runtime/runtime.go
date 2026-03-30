package runtime

import (
	"fmt"
	"os/exec"

	"github.com/jaylate/yacr/runtime/resources"
)

type NamespaceManager interface {
	Create(config ContainerConfig, cmd *exec.Cmd) error
}

type Runtime struct {
	cgroupManager    resources.CgroupManager
	namespaceManager NamespaceManager
}

func CreateRuntime(limits resources.ResourceLimits) (*Runtime, error) {
	var cgroupManager resources.CgroupManager

	hasLimits := limits.MemoryBytes != 0 || limits.CPUCores != 0 || limits.PIDsMax != 0
	if hasLimits {
		var err error
		cgroupManager, err = resources.NewCgroupsManager("", limits)
		if err != nil {
			return nil, fmt.Errorf("failed to create cgroup manager: %w", err)
		}
	}

	return &Runtime{
		cgroupManager:    cgroupManager,
		namespaceManager: &LinuxNamespaceManager{},
	}, nil
}

func (r *Runtime) DeleteRuntime() error {
	if r.cgroupManager != nil {
		return r.cgroupManager.DeleteCgroupHierarchy()
	}
	return nil
}
