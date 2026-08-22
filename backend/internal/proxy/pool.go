package proxy

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"

	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

const hitsKey = "goscrapy:proxy:hits"

var mockProxies = []string{
	"http://mock-proxy-1:8000",
	"http://mock-proxy-2:8000",
	"http://mock-proxy-3:8000",
}

type entry struct {
	url       string
	hits      int64
	fails     int64
	consec    int
	evicted   bool
	lastError string
	lastUsed  string
}

type Pool struct {
	mode      string
	failMax   int
	rdb       *redis.Client
	mu        sync.Mutex
	entries   []*entry
	cursor    uint64
	evictions int64
}

func (p *Pool) BindRedis(rdb *redis.Client) {
	if p != nil {
		p.rdb = rdb
	}
}

func New(mode string, realList []string) *Pool {
	p := &Pool{mode: strings.ToLower(mode), failMax: 3}
	p.Reset(realList)
	return p
}

func (p *Pool) Mode() string { return p.mode }

func (p *Pool) Reset(realList []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	srcs := mockProxies
	if p.mode == "real" {
		srcs = realList
	}
	p.entries = make([]*entry, 0, len(srcs))
	for _, u := range srcs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		p.entries = append(p.entries, &entry{url: u})
	}
}

func (p *Pool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	alive := p.aliveLocked()
	if len(alive) == 0 {
		p.reviveLocked()
		alive = p.aliveLocked()
	}
	if len(alive) == 0 {
		return ""
	}
	i := atomic.AddUint64(&p.cursor, 1)
	e := alive[int(i-1)%len(alive)]
	e.hits++
	e.lastUsed = timeutil.Format(timeutil.Now())
	p.incrLocked(e.url, 1)
	return e.url
}

func (p *Pool) incrLocked(url string, delta int64) {
	if p.rdb == nil || url == "" {
		return
	}
	_ = p.rdb.HIncrBy(context.Background(), hitsKey, url, delta).Err()
}

// Dial 返回实际用于 HTTP 出口的代理。mock 模式只记账、不劫持流量。
func (p *Pool) Dial() string {
	if p == nil {
		return ""
	}
	u := p.Next()
	if p.mode != "real" {
		return ""
	}
	return u
}

func (p *Pool) Report(proxyURL string, ok bool, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e.url != proxyURL {
			continue
		}
		if ok {
			e.consec = 0
			e.lastError = ""
			return
		}
		e.fails++
		e.consec++
		e.lastError = errMsg
		if e.consec >= p.failMax && !e.evicted {
			e.evicted = true
			p.evictions++
		}
		return
	}
}

func (p *Pool) Snapshot() []model.ProxyView {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]model.ProxyView, 0, len(p.entries))
	var remote map[string]string
	if p.rdb != nil {
		remote, _ = p.rdb.HGetAll(context.Background(), hitsKey).Result()
	}
	for _, e := range p.entries {
		hits := e.hits
		if remote != nil {
			if v := remote[e.url]; v != "" {
				if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > hits {
					hits = n
				}
			}
		}
		out = append(out, model.ProxyView{
			URL:        e.url,
			Healthy:    !e.evicted,
			Hits:       hits,
			Fails:      e.fails,
			Evicted:    e.evicted,
			LastError:  e.lastError,
			LastUsedAt: e.lastUsed,
		})
	}
	return out
}

func (p *Pool) Evictions() int64 {
	return p.evictions
}

func (p *Pool) aliveLocked() []*entry {
	out := make([]*entry, 0, len(p.entries))
	for _, e := range p.entries {
		if !e.evicted {
			out = append(out, e)
		}
	}
	return out
}

func (p *Pool) reviveLocked() {
	for _, e := range p.entries {
		e.evicted = false
		e.consec = 0
	}
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}
