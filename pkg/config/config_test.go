package config

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	logger := logrus.New()

	// Clear any existing environment variables
	os.Unsetenv("HEALTHCHECK_SCRAPERS")

	config, err := NewConfig(logger)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Scrapers)
}

func TestNewConfig_WithEnvironmentVariables(t *testing.T) {
	logger := logrus.New()

	// Set environment variables
	os.Setenv("HEALTHCHECK_SCRAPERS", `[{"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8080/ready","ping_url":"http://localhost:8081/ping","scrape_interval_seconds":120}]`)
	defer os.Unsetenv("HEALTHCHECK_SCRAPERS")

	config, err := NewConfig(logger)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Scrapers, 1)
	assert.Equal(t, "cloudflared-tunnel-connector", config.Scrapers[0].Type)
	assert.Equal(t, "http://localhost:8080/ready", config.Scrapers[0].ScrapeURL)
	assert.Equal(t, "http://localhost:8081/ping", config.Scrapers[0].PingURL)
	assert.Equal(t, 120, config.Scrapers[0].ScrapeIntervalSeconds)
}

func TestNewConfig_InvalidJSON(t *testing.T) {
	logger := logrus.New()

	// Set invalid JSON in environment variable
	os.Setenv("HEALTHCHECK_SCRAPERS", `invalid json`)
	defer os.Unsetenv("HEALTHCHECK_SCRAPERS")

	config, err := NewConfig(logger)

	// This should fail due to invalid JSON
	assert.Error(t, err)
	assert.Nil(t, config)
}
