package resources

import (
	"fmt"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"strconv"
	"strings"
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

func ParseMemoryString(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if s == "max" {
		return 0, true
	}

	// Check if last char is a unit
	lastChar := s[len(s)-1]
	if lastChar >= '0' && lastChar <= '9' {
		// Plain number (e.g., "1024")
		num, err := strconv.ParseInt(s, 10, 64)
		return num, err == nil
	}

	// Has unit suffix
	unitTable := map[string]int64{
		"B": 1,
		"K": 1024,
		"M": 1024 * 1024,
		"G": 1024 * 1024 * 1024,
		"T": 1024 * 1024 * 1024 * 1024,
	}
	unitValue, ok := unitTable[string(lastChar)]
	if !ok {
		return 0, false
	}

	size, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return 0, false
	}
	return size * unitValue, true
}
func ParseCPUString(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "max" {
		return "max 100000", true
	}

	num, err := strconv.ParseFloat(s, 64)
	if err != nil || num <= 0 {
		return "", false
	}

	quota := int64(math.Round(num * 100000))
	return fmt.Sprintf("%d 100000", quota), true
}
func ParsePIDsString(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "max" {
		return 0, true
	}

	num, err := strconv.Atoi(s)
	if err != nil || num <= 0 {
		return 0, false
	}
	return num, true
}

func DetectCgroupVersion() string { return "" }
func DetectUserCgroupPath() string {
	currentUid := os.Getuid()
	return fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice/", currentUid, currentUid)
}
func IsCgroupWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}
