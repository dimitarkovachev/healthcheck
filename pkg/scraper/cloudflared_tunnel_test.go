package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCloudflaredTunnelScraper(t *testing.T) {
	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper("http://localhost:8080/ready", "http://localhost:8081/ping", 120, logger)

	assert.Equal(t, "cloudflared-tunnel-connector", scraper.Type())
	assert.Equal(t, "http://localhost:8081/ping", scraper.GetPingURL())
	assert.Equal(t, 120, scraper.GetScrapeInterval())
	assert.NotNil(t, scraper.client)
}

func TestNewCloudflaredTunnelScraper_DefaultInterval(t *testing.T) {
	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper("http://localhost:8080/ready", "http://localhost:8081/ping", 0, logger)

	assert.Equal(t, 30, scraper.GetScrapeInterval()) // Should default to 30 seconds
}

func TestNewCloudflaredTunnelScraper_NegativeInterval(t *testing.T) {
	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper("http://localhost:8080/ready", "http://localhost:8081/ping", -10, logger)

	assert.Equal(t, 30, scraper.GetScrapeInterval()) // Should default to 30 seconds
}

func TestCloudflaredTunnelScraper_Scrape_Success(t *testing.T) {
	// Create a test server that returns a healthy response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"readyConnections":4,"connectorId":"test-id"}`))
	}))
	defer server.Close()

	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper(server.URL, "http://localhost:8081/ping", 30, logger)

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.True(t, result.Healthy)
	assert.Contains(t, result.Message, "Tunnel healthy with 4 ready connections")
	assert.Equal(t, 200, result.Details["status"])
	assert.Equal(t, 4, result.Details["readyConnections"])
	assert.Equal(t, "test-id", result.Details["connectorId"])
}

func TestCloudflaredTunnelScraper_Scrape_Unhealthy_ZeroConnections(t *testing.T) {
	// Create a test server that returns unhealthy response (0 connections)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"readyConnections":0,"connectorId":"test-id"}`))
	}))
	defer server.Close()

	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper(server.URL, "http://localhost:8081/ping", 30, logger)

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "Tunnel unhealthy: status=200, readyConnections=0")
}

func TestCloudflaredTunnelScraper_Scrape_Unhealthy_Non200Status(t *testing.T) {
	// Create a test server that returns non-200 status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper(server.URL, "http://localhost:8081/ping", 30, logger)

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "HTTP status 500")
}

func TestCloudflaredTunnelScraper_Scrape_ConnectionError(t *testing.T) {
	logger := logrus.New()
	// Use a non-existent URL to simulate connection error
	scraper := NewCloudflaredTunnelScraper("http://localhost:99999/ready", "http://localhost:8081/ping", 30, logger)

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "Failed to connect to")
}

func TestCloudflaredTunnelScraper_Scrape_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper(server.URL, "http://localhost:8081/ping", 30, logger)

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "Failed to parse response")
}

func TestCloudflaredTunnelScraper_Scrape_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":200,"readyConnections":4,"connectorId":"test-id"}`))
	}))
	defer server.Close()

	logger := logrus.New()
	scraper := NewCloudflaredTunnelScraper(server.URL, "http://localhost:8081/ping", 30, logger)
	// Set a very short timeout to trigger timeout error
	scraper.client.Timeout = 50 * time.Millisecond

	ctx := context.Background()
	result, err := scraper.Scrape(ctx)

	require.NoError(t, err)
	assert.False(t, result.Healthy)
	assert.Contains(t, result.Message, "Failed to connect to")
}
