package scraper

import (
	"testing"

	"healthcheck/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)

	assert.NotNil(t, factory)
	assert.Equal(t, logger, factory.logger)
}

func TestFactory_CreateScraper_CloudflaredTunnel(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)

	scraperConfig := config.HealthcheckScraper{
		Type:                  "cloudflared-tunnel-connector",
		ScrapeURL:             "http://localhost:8080/ready",
		PingURL:               "http://localhost:8081/ping",
		ScrapeIntervalSeconds: 120,
	}

	scraper, err := factory.CreateScraper(scraperConfig)

	assert.NoError(t, err)
	assert.NotNil(t, scraper)
	assert.Equal(t, "cloudflared-tunnel-connector", scraper.Type())
	assert.Equal(t, "http://localhost:8081/ping", scraper.GetPingURL())
	assert.Equal(t, 120, scraper.GetScrapeInterval())
}

func TestFactory_CreateScraper_UnknownType(t *testing.T) {
	logger := logrus.New()
	factory := NewFactory(logger)

	scraperConfig := config.HealthcheckScraper{
		Type:      "unknown-scraper-type",
		ScrapeURL: "http://localhost:8080/ready",
		PingURL:   "http://localhost:8081/ping",
	}

	scraper, err := factory.CreateScraper(scraperConfig)

	assert.Error(t, err)
	assert.Nil(t, scraper)
	assert.Contains(t, err.Error(), "unknown scraper type: unknown-scraper-type")
}
