package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Config struct {
	FetchURL      string        `yaml:"fetch_url"`
	ScrapeInterval time.Duration `yaml:"scrape_interval"`
	ListenAddress string        `yaml:"listen_address"`
}

var (
	oblocUtilization = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "obloc_utilization_percent",
		Help: "The current O'Bloc utilization",
	})

	scrapeDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "obloc_scrape_duration_seconds",
		Help:    "Time taken to scrape O'Bloc utilization data",
		Buckets: prometheus.DefBuckets,
	})

	scrapeErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "obloc_scrape_errors_total",
		Help: "Total number of errors while scraping O'Bloc utilization",
	})

	config = Config{
		FetchURL:      "https://obloc.ch/_cmsbox_backends_/obloc/guestcounter/",
		ScrapeInterval: 300 * time.Second,
		ListenAddress: ":8081",
	}

	logger *zap.Logger
)

func fetchUtilization() (int, error) {
	start := time.Now()
	defer func() {
		scrapeDuration.Observe(time.Since(start).Seconds())
	}()

	resp, err := http.Get(config.FetchURL)
	if err != nil {
		scrapeErrors.Inc()
		return 0, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		scrapeErrors.Inc()
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		scrapeErrors.Inc()
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	i, err := strconv.Atoi(strings.ReplaceAll(string(body), "\"", ""))
	if err != nil {
		scrapeErrors.Inc()
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return i, nil
}

func recordMetrics(ctx context.Context) {
	logger.Info("Starting metrics collection")
	ticker := time.NewTicker(config.ScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping metrics collection")
			return
		case <-ticker.C:
			utilization, err := fetchUtilization()
			if err != nil {
				logger.Error("Failed to fetch utilization", zap.Error(err))
				continue
			}
			logger.Info("Successfully fetched utilization", zap.Int("value", utilization))
			oblocUtilization.Set(float64(utilization))
		}
	}
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration from environment
	if val, present := os.LookupEnv("FETCH_URL"); present {
		config.FetchURL = val
	}
	if val, present := os.LookupEnv("SCRAPE_INTERVAL"); present {
		if d, err := time.ParseDuration(val); err == nil {
			config.ScrapeInterval = d
		}
	}
	if val, present := os.LookupEnv("LISTEN_ADDRESS"); present {
		config.ListenAddress = val
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics collection
	go recordMetrics(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    config.ListenAddress,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", zap.String("address", config.ListenAddress))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", zap.Error(err))
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server...")
	
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
	
	logger.Info("Server stopped")
}
