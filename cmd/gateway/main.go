package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"middleware-gateway/middleware/ratelimit"
	"middleware-gateway/middleware/ratelimit/domain"
	"middleware-gateway/middleware/ratelimit/infra"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	target, err := url.Parse(cfg.upstreamURL)
	if err != nil {
		log.Fatalf("invalid UPSTREAM_URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	store := infra.NewStore(cfg.rateRPS, cfg.rateBurst)

	var statsStore domain.StatsStore
	if cfg.rateStatsEnabled {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.rateStatsRedisAddr,
			Password: cfg.rateStatsRedisPassword,
			DB:       cfg.rateStatsRedisDB,
		})
		defer func() { _ = rdb.Close() }()

		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := rdb.Ping(pingCtx).Result()
		cancel()
		if err != nil {
			log.Fatalf("redis stats ping error: %v", err)
		}

		statsStore = infra.NewRedisStatsStore(
			rdb,
			infra.WithStatsPrefix(cfg.rateStatsPrefix),
			infra.WithStatsTTL(cfg.rateStatsTTL),
			infra.WithStatsBucket(cfg.rateStatsBucket),
			infra.WithStatsTrackKeys(cfg.rateStatsTrackKeys),
		)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	store.StartJanitor(ctx)

	h := http.Handler(proxy)
	h = ratelimit.ConcurrencyMiddleware(ratelimit.ConcurrencyOptions{
		Max:            cfg.concurrencyMax,
		RejectStatus:   http.StatusServiceUnavailable,
		AcquireTimeout: cfg.concurrencyTimeout,
	})(h)
	if cfg.rateEnabled {
		h = ratelimit.Middleware(ratelimit.Options{
			Store:               store,
			Stats:               statsStore,
			KeyHeader:           cfg.rateKeyHeader,
			TrustXForwardedFor:  cfg.trustXFF,
			RejectStatus:        http.StatusTooManyRequests,
			RetryAfter:          cfg.retryAfter,
			AddRateLimitHeaders: cfg.addHeaders,
		})(h)
	}

	srv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("gateway listening on %s -> %s", cfg.listenAddr, target)
	log.Printf("rate: enabled=%v rps=%.3f burst=%d keyHeader=%q trustXFF=%v", cfg.rateEnabled, cfg.rateRPS, cfg.rateBurst, cfg.rateKeyHeader, cfg.trustXFF)
	log.Printf("rate-stats: enabled=%v redisAddr=%q bucket=%q ttl=%s trackKeys=%v", cfg.rateStatsEnabled, cfg.rateStatsRedisAddr, cfg.rateStatsBucket, cfg.rateStatsTTL, cfg.rateStatsTrackKeys)
	log.Printf("concurrency: max=%d acquireTimeout=%s", cfg.concurrencyMax, cfg.concurrencyTimeout)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

type config struct {
	listenAddr         string
	upstreamURL        string
	rateEnabled        bool
	rateRPS            float64
	rateBurst          int
	rateKeyHeader      string
	trustXFF           bool
	retryAfter         time.Duration
	addHeaders         bool
	concurrencyMax     int
	concurrencyTimeout time.Duration

	rateStatsEnabled       bool
	rateStatsRedisAddr     string
	rateStatsRedisPassword string
	rateStatsRedisDB       int
	rateStatsPrefix        string
	rateStatsTTL           time.Duration
	rateStatsBucket        string
	rateStatsTrackKeys     bool
}

func readConfig() (config, error) {
	cfg := config{}
	cfg.listenAddr = getenvDefault("LISTEN_ADDR", ":8080")
	cfg.upstreamURL = stringsRequired("UPSTREAM_URL")
	cfg.rateEnabled = getenvBoolDefault("RATE_ENABLED", true)
	cfg.rateRPS = getenvFloatDefault("RATE_RPS", 10)
	// IMPORTANTE: o "burst" permite uma rajada inicial de requisições.
	// Com RPS muito baixo (ex: 0.02), o padrão 20 pode dar a impressão de que
	// o limiter não está funcionando, porque as primeiras ~20 passam.
	if burst, ok := getenvInt("RATE_BURST"); ok {
		cfg.rateBurst = burst
	} else {
		cfg.rateBurst = 20
		if getenvIsSet("RATE_RPS") && cfg.rateRPS > 0 && cfg.rateRPS < 1 {
			cfg.rateBurst = 1
		}
	}
	cfg.rateKeyHeader = os.Getenv("RATE_KEY_HEADER")
	cfg.trustXFF = getenvBoolDefault("TRUST_XFF", false)
	cfg.retryAfter = getenvDurationDefault("RETRY_AFTER", 1*time.Second)
	cfg.addHeaders = getenvBoolDefault("ADD_RATELIMIT_HEADERS", false)
	cfg.concurrencyMax = getenvIntDefault("CONCURRENCY_MAX", 100)
	cfg.concurrencyTimeout = getenvDurationDefault("CONCURRENCY_TIMEOUT", 0)

	cfg.rateStatsEnabled = getenvBoolDefault("RATE_STATS_ENABLED", false)
	cfg.rateStatsRedisAddr = getenvDefault("RATE_STATS_REDIS_ADDR", "")
	cfg.rateStatsRedisPassword = os.Getenv("RATE_STATS_REDIS_PASSWORD")
	cfg.rateStatsRedisDB = getenvIntDefault("RATE_STATS_REDIS_DB", 0)
	cfg.rateStatsPrefix = getenvDefault("RATE_STATS_PREFIX", "ratelimit:stats")
	cfg.rateStatsTTL = getenvDurationDefault("RATE_STATS_TTL", 24*time.Hour)
	cfg.rateStatsBucket = getenvDefault("RATE_STATS_BUCKET", "minute")
	cfg.rateStatsTrackKeys = getenvBoolDefault("RATE_STATS_TRACK_KEYS", false)

	if cfg.rateStatsEnabled && strings.TrimSpace(cfg.rateStatsRedisAddr) == "" {
		return config{}, errors.New("RATE_STATS_REDIS_ADDR is required when RATE_STATS_ENABLED=true")
	}

	if cfg.upstreamURL == "" {
		return config{}, errors.New("UPSTREAM_URL is required")
	}
	if cfg.rateRPS <= 0 {
		return config{}, errors.New("RATE_RPS must be > 0")
	}
	if cfg.rateBurst <= 0 {
		return config{}, errors.New("RATE_BURST must be > 0")
	}
	if cfg.concurrencyMax < 0 {
		return config{}, errors.New("CONCURRENCY_MAX must be >= 0")
	}
	return cfg, nil
}
func stringsRequired(k string) string { return os.Getenv(k) }

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvIntDefault(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getenvInt(k string) (int, bool) {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return i, true
}

func getenvIsSet(k string) bool {
	v, ok := os.LookupEnv(k)
	return ok && v != ""
}

func getenvFloatDefault(k string, def float64) float64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func getenvBoolDefault(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvDurationDefault(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
