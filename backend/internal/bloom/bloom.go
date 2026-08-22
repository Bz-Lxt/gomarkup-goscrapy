package bloom

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Filter is a Redis-bitmap bloom filter namespaced per task: bloom:task:{id}.
type Filter struct {
	rdb *redis.Client
	m   uint64
	k   int
}

func New(rdb *redis.Client, m uint64, k int) *Filter {
	if m == 0 {
		m = DefaultM
	}
	if k <= 0 {
		k = DefaultK
	}
	return &Filter{rdb: rdb, m: m, k: k}
}

func Key(taskID int64) string {
	return fmt.Sprintf("bloom:task:%d", taskID)
}

func (f *Filter) M() uint64 { return f.m }
func (f *Filter) K() int    { return f.k }

func (f *Filter) Test(ctx context.Context, taskID int64, url string) (bool, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return true, nil
	}
	key := Key(taskID)
	offs := positions([]byte(url), f.m, f.k)
	pipe := f.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(offs))
	for i, off := range offs {
		cmds[i] = pipe.GetBit(ctx, key, int64(off))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return false, fmt.Errorf("bloom test: %w", err)
	}
	for _, c := range cmds {
		bit, err := c.Result()
		if err != nil && err != redis.Nil {
			return false, err
		}
		if bit == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (f *Filter) Add(ctx context.Context, taskID int64, url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	key := Key(taskID)
	offs := positions([]byte(url), f.m, f.k)
	pipe := f.rdb.Pipeline()
	for _, off := range offs {
		pipe.SetBit(ctx, key, int64(off), 1)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("bloom add: %w", err)
	}
	return nil
}

// AddIfFresh returns true when the URL was not present and is now added.
func (f *Filter) AddIfFresh(ctx context.Context, taskID int64, url string) (fresh bool, err error) {
	seen, err := f.Test(ctx, taskID, url)
	if err != nil {
		return false, err
	}
	if seen {
		return false, nil
	}
	if err := f.Add(ctx, taskID, url); err != nil {
		return false, err
	}
	return true, nil
}

func (f *Filter) Drop(ctx context.Context, taskID int64) error {
	return f.rdb.Del(ctx, Key(taskID)).Err()
}

func (f *Filter) BitCount(ctx context.Context, taskID int64) (int64, error) {
	return f.rdb.BitCount(ctx, Key(taskID), nil).Result()
}

// MemoryFilter is an in-process bitmap used by unit tests (no Redis).
type MemoryFilter struct {
	m    uint64
	k    int
	mu   sync.Mutex
	bits []byte
}

func NewMemory(m uint64, k int) *MemoryFilter {
	if m == 0 {
		m = 1 << 16
	}
	if k <= 0 {
		k = DefaultK
	}
	return &MemoryFilter{m: m, k: k, bits: make([]byte, (m+7)/8)}
}

func (f *MemoryFilter) Test(url string) bool {
	offs := positions([]byte(url), f.m, f.k)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, off := range offs {
		if !bitGet(f.bits, off) {
			return false
		}
	}
	return true
}

func (f *MemoryFilter) Add(url string) {
	offs := positions([]byte(url), f.m, f.k)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, off := range offs {
		bitSet(f.bits, off)
	}
}

func (f *MemoryFilter) AddIfFresh(url string) bool {
	if f.Test(url) {
		return false
	}
	f.Add(url)
	return true
}

func bitGet(b []byte, off uint64) bool {
	i := off / 8
	if int(i) >= len(b) {
		return false
	}
	return b[i]&(1<<(off%8)) != 0
}

func bitSet(b []byte, off uint64) {
	i := off / 8
	if int(i) >= len(b) {
		return
	}
	b[i] |= 1 << (off % 8)
}
