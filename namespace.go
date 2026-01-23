package main

import (
    "log"
    "os"
    "os/exec"
    "syscall"
)

func executeInNamespace(name string, arg ...string) {
    cmd := exec.Command(name, arg...)
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

    err := cmd.Start()
    if err != nil {
        log.Fatal(err)
    }

    err = cmd.Wait()
    if err != nil {
        log.Fatal(err)
    }
}
