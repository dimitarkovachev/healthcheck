package scraper

import (
	"context"
	"time"
)

// Scraper defines the interface for healthcheck scrapers
type Scraper interface {
	// Type returns the type identifier for this scraper
	Type() string

	// Scrape performs the healthcheck and returns the result
	Scrape(ctx context.Context) (*ScrapeResult, error)

	// GetPingURL returns the URL to ping on successful healthcheck
	GetPingURL() string

	// GetScrapeInterval returns the scrape interval in seconds
	GetScrapeInterval() int
}

// ScrapeResult represents the result of a healthcheck scrape
type ScrapeResult struct {
	Healthy   bool
	Message   string
	Timestamp time.Time
	Details   map[string]interface{}
}
