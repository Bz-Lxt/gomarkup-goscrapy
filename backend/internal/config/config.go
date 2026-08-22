package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Role              string
	HTTPAddr          string
	GRPCAddr          string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RendererWS        string
	MockTargetURL     string
	JWTSecret         string
	JWTExpire         time.Duration
	LogLevel          string
	ProxyPoolMode     string
	ProxyList         []string
	LLMEnabled        bool
	ElectionLockTTL   time.Duration
	WorkerID          string
	MasterGRPC        string
	WorkerConcurrency int
	LeaseTTL          time.Duration
	MigrationsDir     string
	BloomM            uint64
	BloomK            int
	MetricsInterval   time.Duration
	ReclaimInterval   time.Duration
	WSPushInterval    time.Duration
	InstanceID        string
}

func Load() (*Config, error) {
	cfg := &Config{
		Role:              env("APP_ROLE", "master"),
		HTTPAddr:          env("HTTP_ADDR", ":8080"),
		GRPCAddr:          env("GRPC_ADDR", ":9090"),
		DatabaseURL:       env("DATABASE_URL", "postgres://goscrapy:goscrapy@127.0.0.1:27334/goscrapy?sslmode=disable"),
		RedisAddr:         env("REDIS_ADDR", "127.0.0.1:27335"),
		RedisPassword:     env("REDIS_PASSWORD", ""),
		RedisDB:           envInt("REDIS_DB", 0),
		RendererWS:        env("RENDERER_WS", "ws://127.0.0.1:27337"),
		MockTargetURL:     strings.TrimRight(env("MOCK_TARGET_URL", "http://mock-target"), "/"),
		JWTSecret:         env("JWT_SECRET", "goscrapy-dev-secret"),
		JWTExpire:         envDuration("JWT_EXPIRE", 24*time.Hour),
		LogLevel:          env("LOG_LEVEL", "info"),
		ProxyPoolMode:     strings.ToLower(env("PROXY_POOL_MODE", "mock")),
		ProxyList:         splitCSV(env("PROXY_LIST", "")),
		LLMEnabled:        envBool("LLM_ENABLED", false),
		ElectionLockTTL:   envDuration("ELECTION_LOCK_TTL", 8*time.Second),
		WorkerID:          env("WORKER_ID", "worker-local"),
		MasterGRPC:        env("MASTER_GRPC", "127.0.0.1:27333"),
		WorkerConcurrency: envInt("WORKER_CONCURRENCY", 4),
		LeaseTTL:          envDuration("LEASE_TTL", 30*time.Second),
		MigrationsDir:     env("MIGRATIONS_DIR", "migrations"),
		BloomM:            envUint64("BLOOM_M", 95850583),
		BloomK:            envInt("BLOOM_K", 7),
		MetricsInterval:   envDuration("METRICS_INTERVAL", 2*time.Second),
		ReclaimInterval:   envDuration("RECLAIM_INTERVAL", 2*time.Second),
		WSPushInterval:    envDuration("WS_PUSH_INTERVAL", 3*time.Second),
		InstanceID:        env("INSTANCE_ID", ""),
	}
	if cfg.InstanceID == "" {
		host, _ := os.Hostname()
		cfg.InstanceID = fmt.Sprintf("%s-%d", host, os.Getpid())
	}
	if cfg.WorkerConcurrency < 1 {
		cfg.WorkerConcurrency = 1
	}
	if cfg.LeaseTTL > 30*time.Second {
		cfg.LeaseTTL = 30 * time.Second
	}
	if cfg.LeaseTTL < time.Second {
		cfg.LeaseTTL = time.Second
	}
	if cfg.BloomK < 1 {
		cfg.BloomK = 7
	}
	if cfg.BloomM < 1024 {
		cfg.BloomM = 95850583
	}
	if cfg.ProxyPoolMode != "real" {
		cfg.ProxyPoolMode = "mock"
	}
	if cfg.Role != "worker" && cfg.Role != "master" {
		cfg.Role = "master"
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envUint64(key string, def uint64) uint64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		if n, e2 := strconv.Atoi(v); e2 == nil {
			return time.Duration(n) * time.Second
		}
		return def
	}
	return d
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
