package scrappers

import (
	"fmt"
)

// Product represents a standardized product structure for all scrapers
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

// Scraper interface that all site scrapers must implement
type Scraper interface {
	// Scrape extracts product information from the given URL
	Scrape(url string) (*Product, error)

	// GetSiteName returns the name/identifier of the site
	GetSiteName() string

	// IsValidURL checks if the URL belongs to this scraper's site
	IsValidURL(url string) bool
}

// SiteInfo contains information about a supported site
type SiteInfo struct {
	Name        string `json:"name"`
	Identifier  string `json:"identifier"`
	BaseURL     string `json:"base_url"`
	Description string `json:"description"`
}

// scraperRegistry holds all registered scrapers
var scraperRegistry = make(map[string]Scraper)

// RegisterScraper registers a new scraper for a site
func RegisterScraper(identifier string, scraper Scraper) {
	scraperRegistry[identifier] = scraper
}

// GetScraper returns a scraper for the given site identifier
func GetScraper(siteIdentifier string) (Scraper, error) {
	scraper, exists := scraperRegistry[siteIdentifier]
	if !exists {
		return nil, fmt.Errorf("no scraper found for site: %s", siteIdentifier)
	}
	return scraper, nil
}

// GetAvailableSites returns information about all supported sites
func GetAvailableSites() []SiteInfo {
	var sites []SiteInfo

	for identifier, scraper := range scraperRegistry {
		sites = append(sites, SiteInfo{
			Name:        scraper.GetSiteName(),
			Identifier:  identifier,
			BaseURL:     getBaseURL(identifier),
			Description: fmt.Sprintf("Scraper for %s", scraper.GetSiteName()),
		})
	}

	return sites
}

// getBaseURL returns the base URL for known sites
func getBaseURL(identifier string) string {
	baseURLs := map[string]string{
		"kontakt": "https://kontakt.az",
		"irshad":  "https://irshad.az",
		"optimal": "https://optimal.az",
	}

	if url, exists := baseURLs[identifier]; exists {
		return url
	}
	return ""
}
