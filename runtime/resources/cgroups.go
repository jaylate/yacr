package resources

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
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
	DeleteCgroupHierarchy() error
}

type CgroupsManager struct {
	basePath string
	limits   ResourceLimits
}

func NewCgroupsManager(basePath string, limits ResourceLimits) (*CgroupsManager, error) {
	if basePath == "" {
		basePath = DetectUserCgroupPath()
	}

	// Check if base path exists and is writable
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cgroup base path does not exist: %s", basePath)
	}
	if !IsCgroupWritable(basePath) {
		return nil, fmt.Errorf("cgroup base path is not writable: %s", basePath)
	}

	m := CgroupsManager{basePath, limits}

	// Create yacr cgroup manually
	yacrPath := filepath.Join(basePath, "yacr")
	err := os.Mkdir(yacrPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create yacr cgroup directory: %w", err)
	}

	// Enable controllers for child cgroups before adding any process
	// This must be done while the cgroup is empty
	err = os.WriteFile(filepath.Join(yacrPath, "cgroup.subtree_control"), []byte("+cpu +io +memory +pids\n"), 0644)
	if err != nil {
		os.RemoveAll(yacrPath)
		return nil, fmt.Errorf("failed to enable subtree_control: %w", err)
	}

	m.basePath = yacrPath
	err = m.writeLimits(limits)
	if err != nil {
		os.RemoveAll(yacrPath)
		return nil, fmt.Errorf("failed to write limits to yacr cgroup: %w", err)
	}

	return &m, nil
}

func (m *CgroupsManager) Create(containerId string, limits ResourceLimits) error {
	containerCgroupPath := filepath.Join(m.basePath, containerId)

	err := os.Mkdir(containerCgroupPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create container cgroup directory: %w", err)
	}

	memoryString := "max"
	if limits.MemoryBytes != 0 {
		memoryString = fmt.Sprintf("%d", limits.MemoryBytes)
	}
	err = os.WriteFile(
		filepath.Join(containerCgroupPath, "memory.max"),
		[]byte(fmt.Sprintf("%s\n", memoryString)),
		0644,
	)
	if err != nil {
		os.RemoveAll(containerCgroupPath)
		return fmt.Errorf("failed to write memory.max: %w", err)
	}

	cpuString, ok := "", false
	if limits.CPUCores == 0 {
		cpuString, ok = ParseCPUString("max")
	} else {
		cpuString, ok = ParseCPUString(strconv.FormatFloat(limits.CPUCores, 'f', -1, 64))
	}
	if !ok {
		os.RemoveAll(containerCgroupPath)
		return fmt.Errorf("failed to parse CPU limits: invalid value")
	}

	err = os.WriteFile(
		filepath.Join(containerCgroupPath, "cpu.max"),
		[]byte(fmt.Sprintf("%s\n", cpuString)),
		0644,
	)
	if err != nil {
		os.RemoveAll(containerCgroupPath)
		return fmt.Errorf("failed to write cpu.max: %w", err)
	}

	pidString := "max"
	if limits.PIDsMax != 0 {
		pidString = fmt.Sprintf("%d", limits.PIDsMax)
	}
	err = os.WriteFile(
		filepath.Join(containerCgroupPath, "pids.max"),
		[]byte(fmt.Sprintf("%s\n", pidString)),
		0644,
	)
	if err != nil {
		os.RemoveAll(containerCgroupPath)
		return fmt.Errorf("failed to write pids.max: %w", err)
	}

	return nil
}

func (m *CgroupsManager) writeLimits(limits ResourceLimits) error {
	memoryString := "max"
	if limits.MemoryBytes != 0 {
		memoryString = fmt.Sprintf("%d", limits.MemoryBytes)
	}
	err := os.WriteFile(
		filepath.Join(m.basePath, "memory.max"),
		[]byte(fmt.Sprintf("%s\n", memoryString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to write memory.max: %w", err)
	}

	cpuString, ok := "", false
	if limits.CPUCores == 0 {
		cpuString, ok = ParseCPUString("max")
	} else {
		cpuString, ok = ParseCPUString(strconv.FormatFloat(limits.CPUCores, 'f', -1, 64))
	}
	if !ok {
		return fmt.Errorf("failed to parse CPU limits: invalid value")
	}

	err = os.WriteFile(
		filepath.Join(m.basePath, "cpu.max"),
		[]byte(fmt.Sprintf("%s\n", cpuString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to write cpu.max: %w", err)
	}

	pidString := "max"
	if limits.PIDsMax != 0 {
		pidString = fmt.Sprintf("%d", limits.PIDsMax)
	}
	err = os.WriteFile(
		filepath.Join(m.basePath, "pids.max"),
		[]byte(fmt.Sprintf("%s\n", pidString)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to write pids.max: %w", err)
	}

	return nil
}

func (m *CgroupsManager) AddProcess(containerId string, pid int) error {
	path := filepath.Join(m.basePath, containerId)

	err := os.WriteFile(
		filepath.Join(path, "cgroup.procs"),
		[]byte(fmt.Sprintf("%s\n", strconv.Itoa(pid))),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	return nil
}
func (m *CgroupsManager) Destroy(containerId string) error {
	path := filepath.Join(m.basePath, containerId)

	// Read PIDs from cgroup.procs
	data, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read cgroup.procs: %w", err)
	}

	// Migrate processes to parent cgroup by writing one PID at a time
	if len(data) > 0 {
		var parentPath string
		if path == m.basePath {
			// Destroying the runtime cgroup itself; migrate to the real parent
			parentPath = filepath.Dir(m.basePath)
		} else {
			parentPath = m.basePath
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			pid, err := strconv.Atoi(line)
			if err != nil {
				return fmt.Errorf("invalid PID %q in cgroup.procs: %w", line, err)
			}

			pidData := []byte(fmt.Sprintf("%d\n", pid))
			if err := os.WriteFile(filepath.Join(parentPath, "cgroup.procs"), pidData, 0644); err != nil {
				return fmt.Errorf("failed to migrate PID %d to parent cgroup: %w", pid, err)
			}
		}
	}

	// Remove the directory
	return os.RemoveAll(path)
}
func (m *CgroupsManager) DeleteCgroupHierarchy() error {
	return m.Destroy("")
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
