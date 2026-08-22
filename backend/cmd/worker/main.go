package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"goscrapy/internal/bloom"
	"goscrapy/internal/config"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/grpcctl"
	"goscrapy/internal/logger"
	"goscrapy/internal/metrics"
	"goscrapy/internal/proxy"
	"goscrapy/internal/queue"
	"goscrapy/internal/ratelimit"
	"goscrapy/internal/redisx"
	"goscrapy/internal/store"
	"goscrapy/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Exit(1)
	}
	lg := logger.Init(cfg.LogLevel)
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		lg.Fatal("postgres", zap.Error(err))
	}
	defer db.Close()
	repos := store.NewRepos(db)

	rdb, err := redisx.Open(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		lg.Fatal("redis", zap.Error(err))
	}
	defer rdb.Close()

	q := queue.New(rdb, cfg.LeaseTTL)
	if err := q.LoadScripts(ctx); err != nil {
		lg.Fatal("queue scripts", zap.Error(err))
	}
	bf := bloom.New(rdb, cfg.BloomM, cfg.BloomK)
	pool := proxy.New(cfg.ProxyPoolMode, cfg.ProxyList)
	pool.BindRedis(rdb)
	fetch := fetcher.New(pool, 15*time.Second)
	limit := ratelimit.NewLimiter(2)
	adapt := ratelimit.NewAdaptive(limit)
	col := metrics.NewCollector()
	run := worker.New(cfg.WorkerID, cfg.WorkerConcurrency, q, bf, repos, fetch, limit, adapt, col)

	cli := grpcctl.NewClient(cfg.MasterGRPC, cfg.WorkerID, col, run)
	go cli.Run(ctx)

	lg.Info("worker started", zap.String("id", cfg.WorkerID), zap.Int("concurrency", cfg.WorkerConcurrency))
	go run.Run(ctx)
	<-ctx.Done()
	run.Shutdown()
	time.Sleep(200 * time.Millisecond)
}
