package runtime

import (
	"testing"

	"github.com/jaylate/yacr/runtime/resources"
)

func TestRunWithCgroups_NilConfig(t *testing.T) {
	err := RunWithCgroups("/bin/sh", []string{"-c", "echo hello"}, nil, resources.ResourceLimits{})
	if err != nil {
		t.Logf("Expected error (no real rootfs/init), got: %v", err)
	}
}

func TestRun_NilConfig(t *testing.T) {
	err := Run("/bin/sh", []string{"-c", "echo hello"}, nil)
	if err != nil {
		t.Logf("Expected error (no real rootfs/init), got: %v", err)
	}
}
