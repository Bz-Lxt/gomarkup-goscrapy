package grpcctl

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"goscrapy/internal/logger"
	"goscrapy/internal/metrics"
	"goscrapy/internal/timeutil"
	"goscrapy/internal/worker"
)

type Client struct {
	addr     string
	workerID string
	col      *metrics.Collector
	runner   *worker.Runner
	interval time.Duration
}

func NewClient(addr, workerID string, col *metrics.Collector, runner *worker.Runner) *Client {
	return &Client{addr: addr, workerID: workerID, col: col, runner: runner, interval: 2 * time.Second}
}

func (c *Client) Run(ctx context.Context) {
	lg := logger.Named("grpc-client").With(zap.String("worker", c.workerID))
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.session(ctx); err != nil && ctx.Err() == nil {
			lg.Warn("control stream ended", zap.Error(err))
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (c *Client) session(ctx context.Context) error {
	conn, err := grpc.NewClient(c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: 15 * time.Second, Timeout: 3 * time.Second, PermitWithoutStream: true}),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := NewControlPlaneClient(conn)
	stream, err := cli.Connect(ctx)
	if err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() {
		for {
			cmd, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			c.apply(cmd)
			_ = stream.Send(&WorkerMessage{
				WorkerId: c.workerID,
				Payload:  &WorkerMessage_Ack{Ack: &CommandAck{CommandId: cmd.GetCommandId(), Ok: true}},
			})
		}
	}()
	tick := time.NewTicker(c.interval)
	defer tick.Stop()
	_ = stream.Send(c.heartbeat())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err == io.EOF {
				return nil
			}
			return err
		case <-tick.C:
			if err := stream.Send(c.metricsMsg()); err != nil {
				return err
			}
		}
	}
}

func (c *Client) heartbeat() *WorkerMessage {
	return &WorkerMessage{
		WorkerId: c.workerID,
		Payload: &WorkerMessage_Heartbeat{Heartbeat: &Heartbeat{
			Status: "online",
			TsUnix: timeutil.Unix(timeutil.Now()),
			Role:   "worker",
		}},
	}
}

func (c *Client) metricsMsg() *WorkerMessage {
	snap := metrics.Snapshot{}
	if c.col != nil {
		snap = c.col.Snapshot()
	}
	return &WorkerMessage{
		WorkerId: c.workerID,
		Payload: &WorkerMessage_Metrics{Metrics: &MetricsReport{
			Cpu:         snap.CPU,
			MemoryMb:    snap.MemoryMB,
			PagesPerMin: snap.PagesPerMin,
			FailRate:    snap.FailRate,
			Status:      "online",
		}},
	}
}

func (c *Client) apply(cmd *MasterCommand) {
	if cmd == nil || c.runner == nil {
		return
	}
	if sd := cmd.GetShutdown(); sd != nil {
		c.runner.Shutdown()
	}
	if rl := cmd.GetRateLimit(); rl != nil {
		c.runner.ApplyRateLimit(rl.GetDomain(), rl.GetQps())
		logger.Named("grpc-client").Info("rate limit command",
			zap.String("domain", rl.GetDomain()), zap.Float64("qps", rl.GetQps()))
	}
	if rr := cmd.GetRuleReload(); rr != nil {
		logger.Named("grpc-client").Info("rule reload",
			zap.Int64("rule", rr.GetRuleId()), zap.Int64("ver", rr.GetVersion()))
	}
}
