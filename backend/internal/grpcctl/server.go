package grpcctl

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/store"
)

type Server struct {
	UnimplementedControlPlaneServer
	repos *store.Repos
	mu    sync.Mutex
	subs  map[string]chan *MasterCommand
}

func NewServer(repos *store.Repos) *Server {
	return &Server{repos: repos, subs: map[string]chan *MasterCommand{}}
}

func (s *Server) Connect(stream grpc.BidiStreamingServer[WorkerMessage, MasterCommand]) error {
	lg := logger.Named("grpc")
	ctx := stream.Context()
	var workerID string
	out := make(chan *MasterCommand, 8)
	defer func() {
		s.mu.Lock()
		if workerID != "" {
			delete(s.subs, workerID)
		}
		s.mu.Unlock()
	}()
	errCh := make(chan error, 2)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case cmd, ok := <-out:
				if !ok {
					return
				}
				if err := stream.Send(cmd); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if msg.GetWorkerId() != "" {
			if workerID == "" {
				workerID = msg.GetWorkerId()
				s.mu.Lock()
				s.subs[workerID] = out
				s.mu.Unlock()
				lg.Info("worker connected", zap.String("id", workerID))
			}
		}
		s.handle(ctx, msg)
		select {
		case err := <-errCh:
			return err
		default:
		}
	}
}

func (s *Server) handle(ctx context.Context, msg *WorkerMessage) {
	if s.repos == nil || s.repos.Nodes == nil || msg == nil {
		return
	}
	if msg.GetWorkerId() == "" {
		return
	}
	n := &model.WorkerNode{
		ID:     msg.GetWorkerId(),
		Role:   "worker",
		Status: model.NodeOnline,
	}
	if hb := msg.GetHeartbeat(); hb != nil {
		if hb.GetStatus() != "" {
			n.Status = hb.GetStatus()
		}
		if hb.GetRole() != "" {
			n.Role = hb.GetRole()
		}
	}
	if m := msg.GetMetrics(); m != nil {
		n.CPU = m.GetCpu()
		n.MemoryMB = m.GetMemoryMb()
		n.PagesPerMin = m.GetPagesPerMin()
		n.FailRate = m.GetFailRate()
		if m.GetStatus() != "" {
			n.Status = m.GetStatus()
		}
		if err := s.repos.Nodes.Upsert(ctx, n); err != nil {
			logger.Named("grpc").Debug("upsert node", zap.Error(err))
		}
		return
	}
	if msg.GetHeartbeat() == nil {
		return
	}
	if err := s.repos.Nodes.Touch(ctx, n); err != nil {
		logger.Named("grpc").Debug("touch node", zap.Error(err))
	}
}

func (s *Server) Broadcast(cmd *MasterCommand) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subs {
		select {
		case ch <- cmd:
		default:
		}
	}
}

func (s *Server) SendTo(workerID string, cmd *MasterCommand) {
	s.mu.Lock()
	ch := s.subs[workerID]
	s.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- cmd:
	default:
	}
}

func ListenAndServe(ctx context.Context, addr string, srv *Server) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	g := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    20 * time.Second,
		Timeout: 5 * time.Second,
	}))
	RegisterControlPlaneServer(g, srv)
	go func() {
		<-ctx.Done()
		g.GracefulStop()
	}()
	logger.Named("grpc").Info("control plane listening", zap.String("addr", addr))
	return g.Serve(lis)
}
