package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"

	"goscrapy/internal/logger"
)

type Snapshot struct {
	CPU         float64
	MemoryMB    float64
	PagesPerMin float64
	FailRate    float64
	Source      string
}

type Collector struct {
	mu      sync.Mutex
	prevCPU CPUSample
	pages   atomic.Int64
	fails   atomic.Int64
	window  []pageHit
	source  string
}

type pageHit struct {
	at   time.Time
	fail bool
}

func NewCollector() *Collector {
	c := &Collector{}
	if usec, ok := ReadCPUUsageUsec(); ok {
		c.prevCPU = CPUSample{UsageUsec: usec, At: time.Now()}
		c.source = "cgroup"
	} else {
		c.source = "gopsutil"
	}
	return c
}

func (c *Collector) ObservePage(fail bool) {
	c.pages.Add(1)
	if fail {
		c.fails.Add(1)
	}
	c.mu.Lock()
	c.window = append(c.window, pageHit{at: time.Now(), fail: fail})
	c.mu.Unlock()
}

func (c *Collector) Snapshot() Snapshot {
	now := time.Now()
	cpuPct, memMB, src := c.usage()
	c.mu.Lock()
	cut := now.Add(-60 * time.Second)
	kept := c.window[:0]
	var pages, fails int
	for _, h := range c.window {
		if h.at.After(cut) {
			kept = append(kept, h)
			pages++
			if h.fail {
				fails++
			}
		}
	}
	c.window = kept
	c.mu.Unlock()
	ppm := float64(pages)
	failRate := 0.0
	if pages > 0 {
		failRate = float64(fails) / float64(pages)
	}
	return Snapshot{
		CPU:         round1(cpuPct),
		MemoryMB:    round1(memMB),
		PagesPerMin: ppm,
		FailRate:    round3(failRate),
		Source:      src,
	}
}

func (c *Collector) usage() (cpuPct, memMB float64, src string) {
	if usec, ok := ReadCPUUsageUsec(); ok {
		c.mu.Lock()
		cpuPct = CPUPercent(c.prevCPU, usec, time.Now())
		c.prevCPU = CPUSample{UsageUsec: usec, At: time.Now()}
		c.mu.Unlock()
		src = "cgroup"
		if b, ok := ReadMemoryBytes(); ok {
			memMB = float64(b) / (1024 * 1024)
			return cpuPct, memMB, src
		}
	}
	src = "gopsutil"
	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		cpuPct = pcts[0]
	} else if err != nil {
		logger.Named("metrics").Debug("gopsutil cpu failed")
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		memMB = float64(vm.Used) / (1024 * 1024)
	}
	return cpuPct, memMB, src
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
func round3(v float64) float64 { return float64(int(v*1000+0.5)) / 1000 }
