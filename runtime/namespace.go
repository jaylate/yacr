package runtime

import (
	"os"
	"os/exec"
	"syscall"
)

// defaultCloneFlags is the Podman/runc-like namespace set for a rootless container.
const defaultCloneFlags = syscall.CLONE_NEWUSER |
	syscall.CLONE_NEWUTS |
	syscall.CLONE_NEWIPC |
	syscall.CLONE_NEWPID |
	syscall.CLONE_NEWNET |
	syscall.CLONE_NEWNS |
	syscall.CLONE_NEWCGROUP

type LinuxNamespaceManager struct{}

func (s *LinuxNamespaceManager) Create(cfg ContainerConfig, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: defaultCloneFlags,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	return nil
}
