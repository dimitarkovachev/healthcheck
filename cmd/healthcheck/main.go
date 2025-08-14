package main

import (
	"os"
	"os/signal"
	"syscall"

	"healthcheck/pkg/config"
	"healthcheck/pkg/healthcheck"

	"github.com/sirupsen/logrus"
)

func main() {
	// Setup logging
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.NewConfig(logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Validate configuration
	if len(cfg.Scrapers) == 0 {
		logger.Warn("No scrapers configured - application will exit")
		return
	}

	// Create and initialize healthcheck manager
	manager := healthcheck.NewManager(cfg, logger)
	if err := manager.Initialize(); err != nil {
		logger.WithError(err).Fatal("Failed to initialize healthcheck manager")
	}

	// Start the manager
	manager.Start()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.WithField("signal", sig).Info("Received shutdown signal")

	// Gracefully stop the manager
	manager.Stop()
	logger.Info("Application shutdown complete")
}
