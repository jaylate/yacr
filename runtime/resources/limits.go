package resources

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type LimitsConfig struct {
	Memory   string
	CPUCores string
	PIDsMax  string
}

type LimitParser struct{}

func NewLimitParser() *LimitParser {
	return &LimitParser{}
}

func (p *LimitParser) ParseConfig(cfg LimitsConfig) (ResourceLimits, error) {
	limits := ResourceLimits{}

	if cfg.Memory != "" {
		memoryBytes, ok := ParseMemoryString(cfg.Memory)
		if !ok {
			return limits, fmt.Errorf("invalid memory value: %s (use format like 100M, 1G, or max)", cfg.Memory)
		}
		limits.MemoryBytes = memoryBytes
	}

	if cfg.CPUCores != "" {
		if cfg.CPUCores == "max" {
			limits.CPUCores = 0
		} else {
			cpuStr, ok := ParseCPUString(cfg.CPUCores)
			if !ok {
				return limits, fmt.Errorf("invalid CPU value: %s (use format like 0.5, 2, or max)", cfg.CPUCores)
			}
			var quota int
			_, err := fmt.Sscanf(cpuStr, "%d", &quota)
			if err != nil {
				return limits, fmt.Errorf("invalid CPU value: %s", cfg.CPUCores)
			}
			limits.CPUCores = float64(quota) / 100000
		}
	}

	if cfg.PIDsMax != "" {
		pidsMax, ok := ParsePIDsString(cfg.PIDsMax)
		if !ok {
			return limits, fmt.Errorf("invalid PIDs value: %s (use format like 50 or max)", cfg.PIDsMax)
		}
		limits.PIDsMax = pidsMax
	}

	return limits, nil
}

func ParseMemoryString(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if s == "max" {
		return 0, true
	}

	lastChar := s[len(s)-1]
	if lastChar >= '0' && lastChar <= '9' {
		num, err := strconv.ParseInt(s, 10, 64)
		if err != nil || num <= 0 {
			return 0, false
		}
		return num, true
	}

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
	if err != nil || size <= 0 {
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
	if err != nil || num <= 0 || math.IsNaN(num) || math.IsInf(num, 0) {
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
