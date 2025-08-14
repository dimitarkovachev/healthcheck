package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"healthcheck/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, cfg, manager.config)
	assert.Equal(t, logger, manager.logger)
	assert.NotNil(t, manager.factory)
	assert.NotNil(t, manager.httpClient)
	assert.NotNil(t, manager.stopChan)
}

func TestManager_Initialize_Success(t *testing.T) {
	cfg := &config.Config{
		Scrapers: []config.HealthcheckScraper{
			{
				Type:                  "cloudflared-tunnel-connector",
				ScrapeURL:             "http://localhost:8080/ready",
				PingURL:               "http://localhost:8081/ping",
				ScrapeIntervalSeconds: 120,
			},
		},
	}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	err := manager.Initialize()

	assert.NoError(t, err)
	assert.Len(t, manager.scrapers, 1)
}

func TestManager_Initialize_UnknownScraperType(t *testing.T) {
	cfg := &config.Config{
		Scrapers: []config.HealthcheckScraper{
			{
				Type:      "unknown-scraper-type",
				ScrapeURL: "http://localhost:8080/ready",
				PingURL:   "http://localhost:8081/ping",
			},
		},
	}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	err := manager.Initialize()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown scraper type: unknown-scraper-type")
}

func TestManager_StartAndStop(t *testing.T) {
	cfg := &config.Config{
		Scrapers: []config.HealthcheckScraper{
			{
				Type:                  "cloudflared-tunnel-connector",
				ScrapeURL:             "http://localhost:8080/ready",
				PingURL:               "http://localhost:8081/ping",
				ScrapeIntervalSeconds: 120,
			},
		},
	}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	err := manager.Initialize()
	require.NoError(t, err)

	// Start the manager
	manager.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the manager
	manager.Stop()

	// The manager should have stopped gracefully
	// We can't easily test the internal state, but we can verify it doesn't panic
}

func TestManager_PingSuccessURL(t *testing.T) {
	// Create a test server to receive the ping
	pingReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pingReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Scrapers: []config.HealthcheckScraper{
			{
				Type:                  "cloudflared-tunnel-connector",
				ScrapeURL:             "http://localhost:8080/ready",
				PingURL:               server.URL,
				ScrapeIntervalSeconds: 120,
			},
		},
	}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	// Test ping with valid URL
	manager.pingSuccessURL(server.URL)

	// Give the HTTP client time to make the request
	time.Sleep(100 * time.Millisecond)

	assert.True(t, pingReceived, "Ping URL should have been called")
}

func TestManager_PingSuccessURL_EmptyURL(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	// Test ping with empty URL (should not panic)
	manager.pingSuccessURL("")
	// If we reach here without panic, the test passes
}

func TestManager_PingSuccessURL_InvalidURL(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	manager := NewManager(cfg, logger)

	// Test ping with invalid URL (should not panic)
	manager.pingSuccessURL("http://invalid-url-that-does-not-exist:99999")
	// If we reach here without panic, the test passes
}
