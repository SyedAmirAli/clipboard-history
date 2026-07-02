package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RuntimeStats returns lightweight process/DB stats for the UI's status
// display: the total number of history rows, and — only when requested (the
// "Show memory usage" setting) — resident memory of clipd plus its direct
// children (the WebKit web/network processes). Skipping the /proc scan when
// the display is off keeps the idle cost at a single COUNT(*).
func (s *Service) RuntimeStats(includeMemory bool) (map[string]any, error) {
	total, err := s.store.CountItems()
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"rssBytes":   int64(0),
		"totalItems": total,
	}
	if includeMemory {
		out["rssBytes"] = processTreeRSS()
	}
	return out, nil
}

// processTreeRSS sums VmRSS of this process and its direct children.
// Best-effort: unreadable /proc entries are simply skipped.
func processTreeRSS() int64 {
	self := os.Getpid()
	total := readVmRSS(self)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return total
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if ppidOf(pid) == self {
			total += readVmRSS(pid)
		}
	}
	return total
}

// readVmRSS returns a process's resident set size in bytes (0 on failure).
func readVmRSS(pid int) int64 {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, found := strings.CutPrefix(line, "VmRSS:"); found {
			fields := strings.Fields(v) // e.g. ["123456", "kB"]
			if len(fields) >= 1 {
				if kb, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
					return kb * 1024
				}
			}
		}
	}
	return 0
}

// ppidOf returns a process's parent pid (0 on failure). Field 4 of
// /proc/<pid>/stat, after the parenthesised (and possibly space-containing)
// command name.
func ppidOf(pid int) int {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	// Skip past ") " that closes the comm field, then state + ppid follow.
	i := strings.LastIndexByte(string(data), ')')
	if i < 0 {
		return 0
	}
	fields := strings.Fields(string(data)[i+1:])
	if len(fields) < 2 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return ppid
}
