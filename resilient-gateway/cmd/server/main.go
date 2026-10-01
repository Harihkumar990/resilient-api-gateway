package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	
	"resilient-gateway/internal/handler"
	"resilient-gateway/internal/limiter"
	"resilient-gateway/internal/middleware"
	"resilient-gateway/internal/config"

)

func main() {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr : cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB: 0,
	})

	rateLimiter := limiter.New(rdb, cfg.RateLimit, cfg.RateWindow)
	healthHdl := handler.NewHealthHandler(rdb)
	apiHdl := handler.NewAPIHandler()

	mux := http.NewServeMux()

	mux.HandleFunc("/livez", healthHdl.Livez)
	mux.HandleFunc("/readyz", healthHdl.Readyz)
	mux.Handle("/metrics", promhttp.Handler())

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/api/v1/resource", apiHdl.HandleData)

	rateLimitedHandler := middleware.RateLimitMiddleware(rateLimiter)(protectedMux)
	mux.Handle("/api/", rateLimitedHandler)

	rootHandler := middleware.MetricsMiddleware(mux)
	
	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: rootHandler,
		ReadTimeout: cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,

	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting resilient fateway on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("http server failure: %w", err)

		}
	}()
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <- serverErrors:
		log.Fatalf("fatal server error: %v", err)
	case sig := <- shutdown:
		log.Printf("Received termination signal (%s), starting graceful shutdown....", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Forced shutdown due to error: %v", err)
			_ = server.Close()
		}

		if err := rdb.Close(); err != nil {
			log.Printf("Error closing redis connection: %v", err)

		}
		log.Println("Server exited cleanly.")
	}
	

}