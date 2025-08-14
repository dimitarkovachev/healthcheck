package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type HealthcheckScraper struct {
	Type                  string `json:"healthcheck-scraper-type"`
	ScrapeURL             string `json:"scrape_url"`
	PingURL               string `json:"ping_url"`
	ScrapeIntervalSeconds int    `json:"scrape_interval_seconds"`
}

type Config struct {
	Scrapers []HealthcheckScraper `mapstructure:"scrapers"`
}

func NewConfig(logger *logrus.Logger) (*Config, error) {
	config := &Config{}

	// Check if HEALTHCHECK_SCRAPERS environment variable is set
	if scrapersJSON := os.Getenv("HEALTHCHECK_SCRAPERS"); scrapersJSON != "" {
		// Parse the JSON array from environment variable
		if err := json.Unmarshal([]byte(scrapersJSON), &config.Scrapers); err != nil {
			return nil, fmt.Errorf("failed to parse HEALTHCHECK_SCRAPERS JSON: %w", err)
		}
	}

	logger.WithField("config", fmt.Sprintf("%+v", config)).Info("Loaded configuration")

	return config, nil
}
