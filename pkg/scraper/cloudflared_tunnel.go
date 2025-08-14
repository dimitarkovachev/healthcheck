package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// CloudflaredTunnelResponse represents the response from the /ready endpoint
type CloudflaredTunnelResponse struct {
	Status           int    `json:"status"`
	ReadyConnections int    `json:"readyConnections"`
	ConnectorID      string `json:"connectorId"`
}

// CloudflaredTunnelScraper implements the Scraper interface for cloudflared tunnel healthchecks
type CloudflaredTunnelScraper struct {
	scrapeURL             string
	pingURL               string
	scrapeIntervalSeconds int
	logger                *logrus.Logger
	client                *http.Client
}

// NewCloudflaredTunnelScraper creates a new cloudflared tunnel scraper
func NewCloudflaredTunnelScraper(scrapeURL, pingURL string, scrapeIntervalSeconds int, logger *logrus.Logger) *CloudflaredTunnelScraper {
	// Set default interval if not specified
	if scrapeIntervalSeconds <= 0 {
		scrapeIntervalSeconds = 30 // Default to 30 seconds
	}

	return &CloudflaredTunnelScraper{
		scrapeURL:             scrapeURL,
		pingURL:               pingURL,
		scrapeIntervalSeconds: scrapeIntervalSeconds,
		logger:                logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Type returns the scraper type identifier
func (c *CloudflaredTunnelScraper) Type() string {
	return "cloudflared-tunnel-connector"
}

// GetPingURL returns the URL to ping on successful healthcheck
func (c *CloudflaredTunnelScraper) GetPingURL() string {
	return c.pingURL
}

// GetScrapeInterval returns the scrape interval in seconds
func (c *CloudflaredTunnelScraper) GetScrapeInterval() int {
	return c.scrapeIntervalSeconds
}

// Scrape performs the healthcheck by calling the /ready endpoint
func (c *CloudflaredTunnelScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	c.logger.WithField("url", c.scrapeURL).Debug("Starting cloudflared tunnel healthcheck")

	req, err := http.NewRequestWithContext(ctx, "GET", c.scrapeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &ScrapeResult{
			Healthy:   false,
			Message:   fmt.Sprintf("Failed to connect to %s: %v", c.scrapeURL, err),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}, nil
	}
	defer resp.Body.Close()

	// Check if response status is not 200
	if resp.StatusCode != http.StatusOK {
		return &ScrapeResult{
			Healthy:   false,
			Message:   fmt.Sprintf("HTTP status %d from %s", resp.StatusCode, c.scrapeURL),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
			},
		}, nil
	}

	// Parse the response body
	var tunnelResp CloudflaredTunnelResponse
	if err := json.NewDecoder(resp.Body).Decode(&tunnelResp); err != nil {
		return &ScrapeResult{
			Healthy:   false,
			Message:   fmt.Sprintf("Failed to parse response from %s: %v", c.scrapeURL, err),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}, nil
	}

	// Check if the tunnel response indicates unhealthy state
	// Based on the Cloudflare documentation and your curl example:
	// - status should be 200 (already checked above)
	// - readyConnections should be > 0 (0 connections means unhealthy)
	healthy := tunnelResp.Status == 200 && tunnelResp.ReadyConnections > 0

	var message string
	if healthy {
		message = fmt.Sprintf("Tunnel healthy with %d ready connections", tunnelResp.ReadyConnections)
	} else {
		message = fmt.Sprintf("Tunnel unhealthy: status=%d, readyConnections=%d", tunnelResp.Status, tunnelResp.ReadyConnections)
	}

	c.logger.WithFields(logrus.Fields{
		"status":           tunnelResp.Status,
		"readyConnections": tunnelResp.ReadyConnections,
		"connectorId":      tunnelResp.ConnectorID,
		"healthy":          healthy,
	}).Info("Cloudflared tunnel healthcheck completed")

	return &ScrapeResult{
		Healthy:   healthy,
		Message:   message,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"status":           tunnelResp.Status,
			"readyConnections": tunnelResp.ReadyConnections,
			"connectorId":      tunnelResp.ConnectorID,
		},
	}, nil
}
