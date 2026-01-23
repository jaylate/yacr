package main

import (
    "log"
    "os"
    "os/exec"
    "syscall"
)

func executeInNamespace(name string, arg ...string) {
    cmd := exec.Command("./bin/init", name)
    cmd.Args = append(cmd.Args, arg...)
    cmd.SysProcAttr = &syscall.SysProcAttr {
	Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS,
	UidMappings: []syscall.SysProcIDMap {
	    {
                ContainerID: 0,
	        HostID: os.Getuid(),
		Size: 1,
	    },
	},
	GidMappings: []syscall.SysProcIDMap {
	    {
                ContainerID: 0,
	        HostID: os.Getgid(),
		Size: 1,
	    },
	},
    }

    cmd.Stdout = os.Stdout
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        log.Fatalf("Failed to start child process: %v", err)
    }

    if err := cmd.Wait(); err != nil {
        log.Fatalf("Child process exited with error: %v", err)
    }
}
