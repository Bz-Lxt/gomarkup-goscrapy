package grpcctl_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/metadata"

	"goscrapy/internal/grpcctl"
	"goscrapy/internal/store"
)

func TestControlStreamCancellationStopsHeartbeatWrite(t *testing.T) {
	state := &blockingDBState{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	db := sql.OpenDB(blockingConnector{state: state})
	t.Cleanup(func() {
		state.unblock()
		_ = db.Close()
	})

	srv := grpcctl.NewServer(store.NewRepos(sqlx.NewDb(db, "blocking")))
	ctx, cancel := context.WithCancel(context.Background())
	stream := &heartbeatStream{
		ctx: ctx,
		messages: []*grpcctl.WorkerMessage{{
			WorkerId: "worker-cancelled",
			Payload: &grpcctl.WorkerMessage_Heartbeat{Heartbeat: &grpcctl.Heartbeat{
				Status: "online",
				Role:   "worker",
			}},
		}},
	}
	done := make(chan error, 1)
	go func() {
		done <- srv.Connect(stream)
	}()

	select {
	case <-state.started:
	case <-time.After(time.Second):
		t.Fatal("heartbeat write did not start")
	}
	cancel()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Connect returned %v after stream cancellation", err)
		}
	case <-time.After(300 * time.Millisecond):
		state.unblock()
		<-done
		t.Fatal("Connect did not stop while a heartbeat write was blocked")
	}
}

type blockingDBState struct {
	started     chan struct{}
	release     chan struct{}
	startedOnce sync.Once
	releaseOnce sync.Once
}

func (s *blockingDBState) unblock() {
	s.releaseOnce.Do(func() { close(s.release) })
}

type blockingConnector struct {
	state *blockingDBState
}

func (c blockingConnector) Connect(context.Context) (driver.Conn, error) {
	return &blockingConn{state: c.state}, nil
}

func (blockingConnector) Driver() driver.Driver { return blockingDriver{} }

type blockingDriver struct{}

func (blockingDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("blocking driver requires a connector")
}

type blockingConn struct {
	state *blockingDBState
}

func (c *blockingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *blockingConn) Close() error { return nil }

func (c *blockingConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c *blockingConn) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	c.state.startedOnce.Do(func() { close(c.state.started) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.state.release:
		return driver.RowsAffected(1), nil
	}
}

type heartbeatStream struct {
	ctx      context.Context
	messages []*grpcctl.WorkerMessage
	mu       sync.Mutex
}

func (s *heartbeatStream) Recv() (*grpcctl.WorkerMessage, error) {
	s.mu.Lock()
	if len(s.messages) > 0 {
		msg := s.messages[0]
		s.messages = s.messages[1:]
		s.mu.Unlock()
		return msg, nil
	}
	s.mu.Unlock()
	<-s.ctx.Done()
	return nil, s.ctx.Err()
}

func (*heartbeatStream) Send(*grpcctl.MasterCommand) error { return nil }

func (*heartbeatStream) SetHeader(metadata.MD) error  { return nil }
func (*heartbeatStream) SendHeader(metadata.MD) error { return nil }
func (*heartbeatStream) SetTrailer(metadata.MD)       {}
func (s *heartbeatStream) Context() context.Context   { return s.ctx }
func (*heartbeatStream) SendMsg(any) error            { return nil }
func (*heartbeatStream) RecvMsg(any) error            { return io.EOF }
