package scraper

import (
	"fmt"

	"healthcheck/pkg/config"

	"github.com/sirupsen/logrus"
)

// Factory creates scrapers based on configuration
type Factory struct {
	logger *logrus.Logger
}

// NewFactory creates a new scraper factory
func NewFactory(logger *logrus.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateScraper creates a scraper based on the configuration
func (f *Factory) CreateScraper(scraperConfig config.HealthcheckScraper) (Scraper, error) {
	switch scraperConfig.Type {
	case "cloudflared-tunnel-connector":
		return NewCloudflaredTunnelScraper(scraperConfig.ScrapeURL, scraperConfig.PingURL, scraperConfig.ScrapeIntervalSeconds, f.logger), nil
	default:
		return nil, fmt.Errorf("unknown scraper type: %s", scraperConfig.Type)
	}
}
