package renderer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"goscrapy/internal/domtree"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

type Record struct {
	ID        string
	URL       string
	Width     int
	Height    int
	PNG       []byte
	HTML      string
	Tree      *domtree.Tree
	Nodes     []model.SnapshotNode
	Source    string
	CreatedAt time.Time
}

type Store struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]*Record
}

func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &Store{ttl: ttl, data: map[string]*Record{}}
}

func (s *Store) Put(rec *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked()
	s.data[rec.ID] = rec
}

func (s *Store) Get(id string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data[id]
	if !ok {
		return nil, false
	}
	if timeutil.Since(rec.CreatedAt) > s.ttl {
		return nil, false
	}
	return rec, true
}

func (s *Store) gcLocked() {
	now := timeutil.Now()
	for id, rec := range s.data {
		if now.Sub(rec.CreatedAt) > s.ttl {
			delete(s.data, id)
		}
	}
}

type Service struct {
	ws    string
	fetch *fetcher.Client
	store *Store
}

func NewService(rendererWS string, fetch *fetcher.Client) *Service {
	return &Service{ws: rendererWS, fetch: fetch, store: NewStore(30 * time.Minute)}
}

func (s *Service) Capture(ctx context.Context, pageURL string) (*Record, error) {
	if pageURL == "" {
		return nil, fmt.Errorf("url required")
	}
	var cap *Capture
	var err error
	if s.ws != "" {
		cap, err = CaptureCDP(ctx, s.ws, pageURL)
		if err != nil {
			logger.Named("renderer").Warn("cdp snapshot failed, falling back to static html",
				zap.Error(err), zap.String("url", pageURL))
		}
	}
	if cap == nil {
		fallback, fallbackErr := CaptureStatic(ctx, s.fetch, pageURL)
		if fallbackErr != nil {
			return nil, fallbackErr
		}
		cap = fallback
	}
	if err != nil {
		return nil, err
	}
	rec := &Record{
		ID:        "snap_" + uuid.NewString(),
		URL:       cap.URL,
		Width:     cap.Width,
		Height:    cap.Height,
		PNG:       cap.PNG,
		HTML:      cap.HTML,
		Tree:      cap.Tree,
		Nodes:     cap.Nodes,
		Source:    cap.Source,
		CreatedAt: timeutil.Now(),
	}
	s.store.Put(rec)
	return rec, nil
}

func (s *Service) Get(id string) (*Record, bool) {
	return s.store.Get(id)
}

func (s *Service) Store() *Store { return s.store }
