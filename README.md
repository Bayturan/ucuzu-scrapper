# Web Scraper REST API

A REST API for scraping product information from various e-commerce websites.

## Features

- Modular scraper architecture with support for multiple sites
- RESTful API with JSON responses
- Standardized product data structure
- Error handling and validation
- CORS support for web clients
- Request logging middleware

## Supported Sites

- **Kontakt.az** (`kontakt`) - Full implementation with detailed product information
- **Irshad.az** (`irshad`) - Template implementation (requires customization)

## Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the application:
   ```bash
   go build .
   ```
4. Run the server:
   ```bash
   ./web-scrappers
   ```

Or run directly with Go:
```bash
go run .
```

## API Endpoints

### Health Check
```
GET /api/v1/health
```

Returns server health status and version information.

**Response:**
```json
{
  "service": "web-scraper-api",
  "status": "healthy",
  "timestamp": "2025-10-16T16:21:08.450685764Z",
  "version": "1.0.0"
}
```

### List Supported Sites
```
GET /api/v1/sites
```

Returns a list of all supported sites and their information.

**Response:**
```json
{
  "count": 2,
  "supported_sites": [
    {
      "name": "Kontakt.az",
      "identifier": "kontakt",
      "base_url": "https://kontakt.az",
      "description": "Scraper for Kontakt.az"
    },
    {
      "name": "Irshad.az",
      "identifier": "irshad",
      "base_url": "https://irshad.az",
      "description": "Scraper for Irshad.az"
    }
  ]
}
```

### Scrape Product Information
```
GET /api/v1/scrape?site={site}&uri={product_url}
```

Scrapes product information from the specified site and URL.

**Parameters:**
- `site`: Site identifier (e.g., "kontakt", "irshad")
- `uri`: Full URL of the product page

**Example:**
```bash
curl "http://localhost:8080/api/v1/scrape?site=kontakt&uri=https://kontakt.az/iphone-13-128-gb-midnight"
```

**Response:**
```json
{
  "name": "iPhone 13 128 GB Midnight",
  "current_price": "1.379,99 ₼",
  "currency": "AZN",
  "availability": "",
  "review_count": "3 Rəylər",
  "brand": "Apple",
  "internal_memory": "128 GB",
  "ram": "4 GB",
  "main_camera": "Var",
  "front_camera": "12 MP",
  "processor": "Apple Apple A15 Bionic",
  "os": "iOS 15",
  "display": "Super Retina XDR OLED",
  "url": "https://kontakt.az/iphone-13-128-gb-midnight",
  "site": "kontakt",
  "scraped_at": "2025-10-16T16:21:20Z"
}
```

## Error Responses

The API returns structured error responses:

### Missing Parameters
```json
{
  "error": "missing_parameters",
  "message": "Both 'site' and 'uri' parameters are required"
}
```

### Unsupported Site
```json
{
  "error": "unsupported_site",
  "message": "Site 'unknownsite' is not supported. Use /api/v1/sites to see available sites"
}
```

### Scraping Failed
```json
{
  "error": "scraping_failed",
  "message": "Failed to scrape URL: [error details]"
}
```

## Product Data Structure

All scrapers return a standardized product structure:

```go
type Product struct {
    Name           string `json:"name"`
    SKU            string `json:"sku,omitempty"`
    CurrentPrice   string `json:"current_price"`
    OriginalPrice  string `json:"original_price,omitempty"`
    Discount       string `json:"discount,omitempty"`
    Currency       string `json:"currency"`
    Availability   string `json:"availability"`
    Rating         string `json:"rating,omitempty"`
    ReviewCount    string `json:"review_count,omitempty"`
    Brand          string `json:"brand,omitempty"`
    InternalMemory string `json:"internal_memory,omitempty"`
    RAM            string `json:"ram,omitempty"`
    MainCamera     string `json:"main_camera,omitempty"`
    FrontCamera    string `json:"front_camera,omitempty"`
    Processor      string `json:"processor,omitempty"`
    OS             string `json:"os,omitempty"`
    Display        string `json:"display,omitempty"`
    URL            string `json:"url"`
    Site           string `json:"site"`
    ScrapedAt      string `json:"scraped_at"`
}
```

## Adding New Scrapers

To add a new scraper for a different site:

1. Create a new file in the `scrappers/` directory (e.g., `scrappers/newsite.go`)
2. Implement the `Scraper` interface:
   ```go
   type Scraper interface {
       Scrape(url string) (*Product, error)
       GetSiteName() string
       IsValidURL(url string) bool
   }
   ```
3. Register the scraper in the `init()` function:
   ```go
   func init() {
       RegisterScraper("newsite", NewNewSiteScraper())
   }
   ```
4. Update the `getBaseURL()` function in `registry.go` if needed

## Development

### Running in Debug Mode
Set the `DEBUG` environment variable to enable debug features:
```bash
DEBUG=1 go run .
```

This will save the HTML content to `debug.html` for analysis.

### Project Structure
```
.
├── main.go                 # REST API server
├── go.mod                  # Go module dependencies
├── scrappers/
│   ├── registry.go         # Scraper interface and registry
│   ├── kontakt.go          # Kontakt.az scraper
│   └── irshad.go           # Irshad.az scraper (template)
└── README.md               # This file
```

## License

This project is for educational purposes. Please respect the terms of service of the websites you scrape.