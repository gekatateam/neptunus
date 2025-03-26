//go:build linux

package memory

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func sysTotalMemory() uint64 {
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}
	// If this is a 32-bit system, then these fields are
	// uint32 instead of uint64.
	// So we always convert to uint64 to match signature.
	memTotal := uint64(in.Totalram) * uint64(in.Unit)

	// cgroup v1 detection
	if limitRaw, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if limit, err := strconv.ParseUint(strings.TrimSuffix(string(limitRaw), "\n"), 10, 64); err == nil {
			if limit < memTotal {
				return limit
			}
		}
	}

	// cgroup v2 detection
	if limitRaw, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		if limit, err := strconv.ParseUint(strings.TrimSuffix(string(limitRaw), "\n"), 10, 64); err == nil {
			if limit < memTotal {
				return limit
			}
		}
	}

	return memTotal
}
