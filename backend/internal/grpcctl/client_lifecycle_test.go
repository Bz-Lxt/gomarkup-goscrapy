package grpcctl

import (
	"context"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type rejectingControlPlane struct {
	UnimplementedControlPlaneServer
	sessions atomic.Int32
}

func (s *rejectingControlPlane) Connect(stream grpc.BidiStreamingServer[WorkerMessage, MasterCommand]) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	s.sessions.Add(1)
	return status.Error(codes.Unavailable, "control plane restarting")
}

type trackingListener struct {
	net.Listener
	mu     sync.Mutex
	active int
}

func (l *trackingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.mu.Lock()
	l.active++
	l.mu.Unlock()
	return &trackedConn{Conn: c, closed: func() {
		l.mu.Lock()
		l.active--
		l.mu.Unlock()
	}}, nil
}

func (l *trackingListener) activeConnections() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.active
}

type trackedConn struct {
	net.Conn
	once   sync.Once
	closed func()
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.closed)
	return err
}

func TestClientRunReleasesFailedSessions(t *testing.T) {
	f, err := os.CreateTemp("/tmp", "goscrapy-control-*.sock")
	if err != nil {
		t.Fatal(err)
	}
	socket := f.Name()
	_ = f.Close()
	_ = os.Remove(socket)
	t.Cleanup(func() { _ = os.Remove(socket) })
	base, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	lis := &trackingListener{Listener: base}
	server := grpc.NewServer()
	control := &rejectingControlPlane{}
	RegisterControlPlaneServer(server, control)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	client := NewClient("unix://"+socket, "worker-lifecycle", nil, nil)
	done := make(chan struct{})
	go func() {
		client.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for control.sessions.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := control.sessions.Load(); got < 2 {
		cancel()
		<-done
		t.Fatalf("expected at least two control sessions, got %d", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("client did not stop after cancellation")
	}

	deadline = time.Now().Add(time.Second)
	for lis.activeConnections() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := lis.activeConnections(); got != 0 {
		t.Fatalf("control client stopped with %d transport connections still open", got)
	}
}
