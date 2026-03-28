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
	Create(containerId string, limits ResourceLimits) error
	AddProcess(containerId string, pid int) error
	Destroy(containerId string) error
}

type CgroupsManager struct {
	basePath string
	limits   ResourceLimits
}

func NewCgroupsManager(basePath string, limits ResourceLimits) *CgroupsManager {
	return &CgroupsManager{basePath, limits}
}

func (m *CgroupsManager) Create(containerId string, limits ResourceLimits) error {
	containerCgroupPath := m.basePath + "/" + containerId

	err := os.Mkdir(containerCgroupPath, 0755)
	if err != nil {
		return fmt.Errorf("CgroupsManagerCreate: %w", err)
	}

	memoryString := "max"
	if limits.MemoryBytes != 0 {
		memoryString = fmt.Sprintf("%d", limits.MemoryBytes)
	}
	err = os.WriteFile(
		containerCgroupPath+"/memory.max",
		[]byte(fmt.Sprintf("%s\n", memoryString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("CgroupsManagerCreate: %w", err)
	}

	cpuString, ok := "", false
	if limits.CPUCores == 0 {
		cpuString, ok = ParseCPUString("max")
	} else {
		cpuString, ok = ParseCPUString(strconv.FormatFloat(limits.CPUCores, 'f', -1, 64))
	}
	if !ok {
		return fmt.Errorf("CgroupsManagerCreate: ParseCPUString")
	}

	err = os.WriteFile(
		containerCgroupPath+"/cpu.max",
		[]byte(fmt.Sprintf("%s\n", cpuString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("CgroupsManagerCreate: %w", err)
	}

	pidString := "max"
	if limits.PIDsMax != 0 {
		pidString = fmt.Sprintf("%d", limits.PIDsMax)
	}
	err = os.WriteFile(
		containerCgroupPath+"/pids.max",
		[]byte(fmt.Sprintf("%s\n", pidString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("CgroupsManagerCreate: %w", err)
	}

	return nil
}
func (m *CgroupsManager) AddProcess(containerId string, pid int) error {
	err := os.WriteFile(
		m.basePath+"/"+containerId+"/cgroup.procs",
		[]byte(fmt.Sprintf("%s\n", strconv.Itoa(pid))),
		0644,
	)
	if err != nil {
		return fmt.Errorf("CgroupManagerAddProcess: %w", err)
	}

	return nil
}
func (m *CgroupsManager) Destroy(containerId string) error {
	path := m.basePath + "/" + containerId

	// Read PIDs from cgroup.procs
	data, err := os.ReadFile(path + "/cgroup.procs")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("CgroupManagerDestroy: %w", err)
	}

	// Migrate processes to parent cgroup
	if len(data) > 0 {
		parentPath := m.basePath // parent is m.basePath
		// Write PIDs to parent cgroup.procs
		err = os.WriteFile(parentPath+"/cgroup.procs", data, 0644)
		if err != nil {
			return fmt.Errorf("CgroupManagerDestroy: %w", err)
		}
	}

	// Remove the directory
	return os.RemoveAll(path)
}

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

func DetectCgroupVersion() string {
	data, _ := os.ReadFile("/proc/mounts")
	if strings.Contains(string(data), "cgroup2") {
		return "v2"
	}
	return "v1"
}
func DetectUserCgroupPath() string {
	currentUid := os.Getuid()
	return fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice", currentUid, currentUid)
}
func IsCgroupWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}
