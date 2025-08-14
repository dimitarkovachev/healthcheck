package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"healthcheck/pkg/config"
	"healthcheck/pkg/scraper"

	"github.com/sirupsen/logrus"
)

// Manager orchestrates healthcheck scrapers and handles ping functionality
type Manager struct {
	config     *config.Config
	factory    *scraper.Factory
	logger     *logrus.Logger
	scrapers   []scraper.Scraper
	httpClient *http.Client
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewManager creates a new healthcheck manager
func NewManager(cfg *config.Config, logger *logrus.Logger) *Manager {
	return &Manager{
		config:  cfg,
		factory: scraper.NewFactory(logger),
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Initialize sets up all scrapers based on configuration
func (m *Manager) Initialize() error {
	m.logger.Info("Initializing healthcheck manager")

	for _, scraperConfig := range m.config.Scrapers {
		scraper, err := m.factory.CreateScraper(scraperConfig)
		if err != nil {
			return fmt.Errorf("failed to create scraper %s: %w", scraperConfig.Type, err)
		}

		m.scrapers = append(m.scrapers, scraper)
		m.logger.WithFields(logrus.Fields{
			"type":       scraper.Type(),
			"scrape_url": scraperConfig.ScrapeURL,
			"ping_url":   scraperConfig.PingURL,
		}).Info("Created scraper")
	}

	m.logger.WithField("scraper_count", len(m.scrapers)).Info("Healthcheck manager initialized")
	return nil
}

// Start begins the healthcheck loop
func (m *Manager) Start() {
	m.logger.Info("Starting healthcheck manager")

	// Start healthcheck loop
	m.wg.Add(1)
	go m.healthcheckLoop()

	m.logger.Info("Healthcheck manager started")
}

// Stop gracefully stops the healthcheck manager
func (m *Manager) Stop() {
	m.logger.Info("Stopping healthcheck manager")
	close(m.stopChan)
	m.wg.Wait()
	m.logger.Info("Healthcheck manager stopped")
}

// healthcheckLoop runs the main healthcheck loop with individual intervals
func (m *Manager) healthcheckLoop() {
	defer m.wg.Done()

	// Create individual tickers for each scraper
	scraperTickers := make(map[scraper.Scraper]*time.Ticker)
	defer func() {
		for _, ticker := range scraperTickers {
			ticker.Stop()
		}
	}()

	// Start individual tickers for each scraper
	for _, s := range m.scrapers {
		interval := s.GetScrapeInterval()
		if interval <= 0 {
			interval = 30 // Default to 30 seconds if not specified
		}

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		scraperTickers[s] = ticker

		// Run initial healthcheck for this scraper
		go m.runSingleHealthcheck(s)

		// Start the ticker loop for this scraper
		go func(scraper scraper.Scraper, ticker *time.Ticker) {
			for {
				select {
				case <-ticker.C:
					m.runSingleHealthcheck(scraper)
				case <-m.stopChan: // Single stop chan all scrapers goroutines?
					return
				}
			}
		}(s, ticker)
	}

	// Wait for stop signal
	<-m.stopChan
}

// runSingleHealthcheck runs a healthcheck for a single scraper
func (m *Manager) runSingleHealthcheck(s scraper.Scraper) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := s.Scrape(ctx)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"scraper_type": s.Type(),
			"error":        err.Error(),
		}).Error("Healthcheck failed with error")
		return
	}

	m.logger.WithFields(logrus.Fields{
		"scraper_type": s.Type(),
		"healthy":      result.Healthy,
		"message":      result.Message,
		"timestamp":    result.Timestamp,
	}).Info("Healthcheck completed")

	// If healthy, ping the success URL
	if result.Healthy {
		m.pingSuccessURL(s.GetPingURL())
	}
}

// pingSuccessURL sends a GET request to the success URL
func (m *Manager) pingSuccessURL(url string) {
	if url == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"url":   url,
			"error": err.Error(),
		}).Error("Failed to create ping request")
		return
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"url":   url,
			"error": err.Error(),
		}).Error("Failed to ping success URL")
		return
	}
	defer resp.Body.Close()

	m.logger.WithFields(logrus.Fields{
		"url":         url,
		"status_code": resp.StatusCode,
	}).Info("Successfully pinged success URL")
}
