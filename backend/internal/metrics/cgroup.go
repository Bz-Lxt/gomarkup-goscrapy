package metrics

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cgroupFS = "/sys/fs/cgroup"
	procStat = "/proc/stat"
)

type CPUSample struct {
	UsageUsec uint64
	At        time.Time
}

func ReadMemoryBytes() (uint64, bool) {
	for _, p := range []string{
		filepath.Join(cgroupFS, "memory.current"),
		filepath.Join(cgroupFS, "memory/memory.usage_in_bytes"),
	} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func ReadMemoryLimitBytes() (uint64, bool) {
	for _, p := range []string{
		filepath.Join(cgroupFS, "memory.max"),
		filepath.Join(cgroupFS, "memory/memory.limit_in_bytes"),
	} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(b))
		if s == "" || s == "max" {
			return 0, false
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err == nil && n > 0 && n < 1<<62 {
			return n, true
		}
	}
	return 0, false
}

func ReadCPUUsageUsec() (uint64, bool) {
	p := filepath.Join(cgroupFS, "cpu.stat")
	f, err := os.Open(p)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "usage_usec") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				n, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return n, true
				}
			}
		}
	}
	return 0, false
}

func CPUPercent(prev CPUSample, nowUsec uint64, now time.Time) float64 {
	if prev.UsageUsec == 0 || now.Before(prev.At) || now.Equal(prev.At) {
		return 0
	}
	elapsed := now.Sub(prev.At).Seconds()
	if elapsed <= 0 {
		return 0
	}
	deltaSec := float64(nowUsec-prev.UsageUsec) / 1e6
	pct := (deltaSec / elapsed) * 100
	if pct < 0 {
		return 0
	}
	if pct > 400 {
		return 400
	}
	return pct
}

func CgroupAvailable() bool {
	_, err := os.Stat(filepath.Join(cgroupFS, "cpu.stat"))
	return err == nil
}
