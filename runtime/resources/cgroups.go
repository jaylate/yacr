package resources

import (
	"os"
	"fmt"
)

type ResourceLimits struct {
	MemoryBytes int64   // 0 = unlimited
	CPUCores    float64 // 0 = unlimited
	PIDsMax     int     // 0 = unlimited
}

type CgroupManager interface {
	Create(id string, limits ResourceLimits) error
	AddProcess(pid int) error
	Destroy(id string) error
}

type CgroupsManager struct {
	basePath string
	limits   ResourceLimits
}

func NewCgroupsManager(basePath string, limits ResourceLimits) *CgroupsManager {
	return &CgroupsManager{basePath, limits}
}

func (m *CgroupsManager) Create(id string, limits ResourceLimits) error { return nil }
func (m *CgroupsManager) AddProcess(pid int) error                      { return nil }
func (m *CgroupsManager) Destroy(id string) error                       { return nil }

func ParseMemoryString(s string) (int64, bool) { return -1, false }
func ParseCPUString(s string) (string, bool)   { return "", false }
func ParsePIDsString(s string) (int, bool)     { return 0, false }
func DetectCgroupVersion() string              { return "" }
func DetectUserCgroupPath() string {
	currentUid := os.Getuid()
	return fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice/", currentUid, currentUid)
}
func IsCgroupWritable(path string) bool { return false }
