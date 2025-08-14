# Healthcheck

A Go-based healthcheck application that scrapes healthcheck endpoints and pings healthcehck monitors on successful healthchecks. The application is designed to be deployed in a container.

## Features

- **Modular Scraper Architecture**: Easy to add new healthcheck types
- **Configurable via Environment Variables**: Simple configuration without config files
- **Automatic Health Monitoring**: Runs healthchecks every 30 seconds
- **Success Notifications**: Pings configured URLs when healthchecks pass
- **Graceful Shutdown**: Handles SIGINT and SIGTERM signals properly
- **Comprehensive Logging**: JSON-formatted logs for easy parsing

## Supported Scraper Types

### Cloudflared Tunnel Connector

Monitors Cloudflare tunnels by checking the `/ready` endpoint. This endpoint is exposed when you run cloudflared with the `--metrics` flag.

**Health Criteria:**
- HTTP status must be 200
- `readyConnections` must be greater than 0

**Configuration:**
```json
{
  "healthcheck-scraper-type": "cloudflared-tunnel-connector",
  "scrape_url": "http://localhost:8080/ready",
  "scrape_interval_seconds": 120,
  "ping_url": "http://your-monitoring-service.com/health"
}
```

## Configuration

The application is configured entirely through environment variables. All configuration keys are prefixed with `HEALTHCHECK_`.

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `HEALTHCHECK_SCRAPERS` | JSON array of scraper configurations | `[]` | See configuration examples below |

### Configuration Examples

#### Single Scraper
```bash
export HEALTHCHECK_SCRAPERS='[{"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8080/ready","scrape_interval_seconds":120,"ping_url":"http://your-monitoring-service.com/health"}]'
```

#### Multiple Scrapers
```bash
export HEALTHCHECK_SCRAPERS='[
  {"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8080/ready","scrape_interval_seconds":120,"ping_url":"http://monitoring1.com/health"},
  {"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8081/ready","scrape_interval_seconds":60,"ping_url":"http://monitoring2.com/health"}
]'
```

## Cloudflared Tunnel Setup

To use the cloudflared tunnel connector scraper, you need to enable the metrics server on your cloudflared instance:

```bash
# For locally-managed tunnels
cloudflared tunnel --metrics 127.0.0.1:8080 run my-tunnel

# For remotely-managed tunnels, add to your config.yml
metrics: 127.0.0.1:8080
```

The `/ready` endpoint will be available at `http://127.0.0.1:8080/ready` and returns:
```json
{
  "status": 200,
  "readyConnections": 4,
  "connectorId": "8e7ba03c-19c9-4fef-89f8-f054c3485b56"
}
```

## Building and Running

### Local Development

```bash
# Download dependencies
go mod tidy

# Run tests
go test ./...

# Build the application
go build -o healthcheck ./cmd/healthcheck

# Run with configuration
export HEALTHCHECK_SCRAPERS='[{"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8080/ready","ping_url":"http://localhost:8081/ping"}]'
./healthcheck
```

### Docker

```bash
# Build the image
docker build -t healthcheck .

# Run with configuration
docker run -e HEALTHCHECK_SCRAPERS='[{"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://localhost:8080/ready","ping_url":"http://your-monitoring-service.com/health"}]' healthcheck
```

### Docker Compose

```yaml
version: '3.8'
services:
  healthcheck:
    build: .
    environment:
      - HEALTHCHECK_SCRAPERS=[{"healthcheck-scraper-type":"cloudflared-tunnel-connector","scrape_url":"http://host.docker.internal:8080/ready","scrape_interval_seconds":120,"ping_url":"http://your-monitoring-service.com/health"}]
    restart: unless-stopped
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Main Entry    │───▶│  Config Loader   │───▶│ Healthcheck     │
│   Point         │    │                  │    │ Manager         │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
                                              ┌──────────────────┐
                                              │   Scraper        │
                                              │   Factory        │
                                              └──────────────────┘
                                                       │
                                                       ▼
                                              ┌──────────────────┐
                                              │   Scrapers       │
                                              │   (Cloudflared,  │
                                              │    etc.)         │
                                              └──────────────────┘
```

## Project Structure

```
healthcheck/
├── cmd/
│   └── healthcheck/
│       └── main.go              # Application entry point
├── pkg/
│   ├── config/
│   │   ├── config.go            # Configuration management
│   │   └── config_test.go       # Configuration tests
│   ├── scraper/
│   │   ├── scraper.go           # Scraper interface
│   │   ├── factory.go           # Scraper factory
│   │   ├── cloudflared_tunnel.go # Cloudflared tunnel scraper
│   │   └── *_test.go            # Scraper tests
│   └── healthcheck/
│       ├── manager.go            # Healthcheck orchestration
│       └── manager_test.go      # Manager tests
├── Dockerfile                    # Container build instructions
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
└── README.md                    # This file
```

## Adding New Scraper Types

To add a new scraper type:

1. Implement the `Scraper` interface in a new file under `pkg/scraper/`
2. Add the new type to the factory in `pkg/scraper/factory.go`
3. Add tests for the new scraper
4. Update this README with configuration examples

Example scraper implementation:
```go
type MyCustomScraper struct {
    scrapeURL            string
    pingURL              string
    scrapeIntervalSeconds int
    logger               *logrus.Logger
}

func (m *MyCustomScraper) Type() string {
    return "my-custom-scraper"
}

func (m *MyCustomScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
    // Implement your healthcheck logic here
}

func (m *MyCustomScraper) GetPingURL() string {
    return m.pingURL
}

func (m *MyCustomScraper) GetScrapeInterval() int {
    return m.scrapeIntervalSeconds
}
```

## Logging

The application uses structured logging with JSON format. Log levels can be controlled via the `LOG_LEVEL` environment variable.

Example log output:
```json
{"level":"info","msg":"Healthcheck manager started","time":"2024-01-15T10:30:00Z"}
{"level":"info","msg":"Healthcheck completed","scraper_type":"cloudflared-tunnel-connector","healthy":true,"message":"Tunnel healthy with 4 ready connections","time":"2024-01-15T10:30:30Z"}
```

## Healthcheck Frequency

Each scraper can have its own configurable scrape interval via the `scrape_interval_seconds` field. If not specified or set to 0 or negative values, the default interval of 30 seconds is used.

**Example intervals:**
- `30` - Check every 30 seconds (default)
- `60` - Check every minute
- `120` - Check every 2 minutes
- `300` - Check every 5 minutes

**Note:** Each scraper runs independently with its own timer, so you can have different intervals for different services.

## Error Handling

- **Connection Failures**: Scrapers return unhealthy status when they can't connect
- **Invalid Responses**: Non-200 HTTP status codes or malformed JSON result in unhealthy status
- **Timeout Handling**: All HTTP requests have configurable timeouts
- **Graceful Degradation**: Individual scraper failures don't stop the entire system

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
