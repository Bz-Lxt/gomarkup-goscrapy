package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"goscrapy/internal/api"
	"goscrapy/internal/auth"
	"goscrapy/internal/bloom"
	"goscrapy/internal/config"
	"goscrapy/internal/fetcher"
	"goscrapy/internal/grpcctl"
	"goscrapy/internal/logger"
	"goscrapy/internal/parser"
	"goscrapy/internal/proxy"
	"goscrapy/internal/queue"
	"goscrapy/internal/redisx"
	"goscrapy/internal/renderer"
	"goscrapy/internal/scheduler"
	"goscrapy/internal/seed"
	"goscrapy/internal/store"
	"goscrapy/internal/ws"
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
	if err := store.Migrate(db, cfg.MigrationsDir); err != nil {
		lg.Fatal("migrate", zap.Error(err))
	}
	repos := store.NewRepos(db)
	if err := seed.Ensure(ctx, repos, cfg.MockTargetURL); err != nil {
		lg.Fatal("seed", zap.Error(err))
	}

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
	hub := ws.NewHub()
	elector := scheduler.NewElector(rdb, cfg.InstanceID, cfg.ElectionLockTTL)
	loop := scheduler.NewLoop(elector, q, bf, repos, hub, cfg.ReclaimInterval)
	go loop.Run(ctx)

	grpcSrv := grpcctl.NewServer(repos)
	go func() {
		if err := grpcctl.ListenAndServe(ctx, cfg.GRPCAddr, grpcSrv); err != nil {
			lg.Error("grpc serve", zap.Error(err))
		}
	}()

	deps := &api.Deps{
		Cfg:     cfg,
		Auth:    auth.New(cfg.JWTSecret, cfg.JWTExpire),
		Repos:   repos,
		Redis:   rdb,
		Queue:   q,
		Bloom:   bf,
		Proxy:   pool,
		Fetch:   fetch,
		Engine:  parser.NewEngine(),
		Snap:    renderer.NewService(cfg.RendererWS, fetch),
		Hub:     hub,
		Elector: elector,
	}
	engine := api.NewRouter(deps)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: engine, ReadHeaderTimeout: 8 * time.Second}
	go func() {
		lg.Info("master http listening", zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lg.Fatal("http", zap.Error(err))
		}
	}()

	<-ctx.Done()
	elector.Resign(context.Background())
	shCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(shCtx)
}
