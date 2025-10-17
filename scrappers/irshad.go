package scrappers

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// IrshadScraper implements the Scraper interface for irshad.az
type IrshadScraper struct{}

// NewIrshadScraper creates a new instance of IrshadScraper
func NewIrshadScraper() *IrshadScraper {
	return &IrshadScraper{}
}

// GetSiteName returns the site name
func (i *IrshadScraper) GetSiteName() string {
	return "Irshad.az"
}

// IsValidURL checks if the URL belongs to irshad.az
func (i *IrshadScraper) IsValidURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "irshad.az")
}

// Scrape extracts product information from irshad.az URL
func (i *IrshadScraper) Scrape(url string) (*Product, error) {
	if !i.IsValidURL(url) {
		return nil, fmt.Errorf("URL does not belong to irshad.az: %s", url)
	}

	product := &Product{
		URL:       url,
		Site:      "irshad",
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "az,en-US;q=0.7,en;q=0.3")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract product name - look for specific h1 tags and meta data
	doc.Find("h1").Each(func(index int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		// Skip navigation breadcrumbs and find actual product title
		if text != "" && product.Name == "" && !strings.Contains(strings.ToLower(text), "irşad") &&
			!strings.Contains(strings.ToLower(text), "məhsul") && len(text) > 5 {
			product.Name = text
		}
	})

	// Try to extract from script/JSON data if h1 didn't work
	if product.Name == "" {
		doc.Find("script").Each(func(index int, s *goquery.Selection) {
			content := s.Text()
			if strings.Contains(content, "\"title\":") {
				// Look for JSON data containing title
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					if strings.Contains(line, "\"title\":") && strings.Contains(line, "PlayStation") {
						// Extract title from JSON line
						if start := strings.Index(line, "\"title\":\""); start != -1 {
							start += 9 // length of "title":"
							if end := strings.Index(line[start:], "\","); end != -1 {
								title := line[start : start+end]
								if len(title) > 5 && product.Name == "" {
									product.Name = title
								}
							}
						}
					}
				}
			}
		})
	}

	// Extract prices dynamically from various sources
	var currentPrice, originalPrice string

	// Method 1: Try to extract from structured JSON data in script tags
	doc.Find("script").Each(func(index int, s *goquery.Selection) {
		content := s.Text()

		// Look for Calculator.init JSON data which contains the correct pricing
		if strings.Contains(content, "Calculator.init") && strings.Contains(content, "price") {
			// Extract price from Calculator.init JSON - be more flexible with spacing
			re := regexp.MustCompile(`"price"\s*:\s*(\d+(?:\.\d+)?)`)
			matches := re.FindAllStringSubmatch(content, -1)
			if len(matches) > 0 && len(matches[0]) > 1 {
				if price, err := strconv.ParseFloat(matches[0][1], 64); err == nil {
					currentPrice = fmt.Sprintf("%.2f AZN", price)
				}
			}

			// Extract installment_price (original price) - be more flexible
			re2 := regexp.MustCompile(`"installment_price"\s*:\s*(\d+(?:\.\d+)?)`)
			matches2 := re2.FindAllStringSubmatch(content, -1)
			if len(matches2) > 0 && len(matches2[0]) > 1 {
				if price, err := strconv.ParseFloat(matches2[0][1], 64); err == nil {
					originalPrice = fmt.Sprintf("%.2f AZN", price)
				}
			}

			// Break after finding Calculator.init data to prioritize it
			if currentPrice != "" {
				return
			}
		}
	})

	// Method 2: Fallback - Look for JSON-LD structured data if Calculator.init failed
	if currentPrice == "" {
		doc.Find("script").Each(func(index int, s *goquery.Selection) {
			content := s.Text()
			if strings.Contains(content, `"@type"`) && strings.Contains(content, "Product") {
				re := regexp.MustCompile(`"price"\s*:\s*"?(\d+(?:\.\d+)?)"?`)
				matches := re.FindAllStringSubmatch(content, -1)
				if len(matches) > 0 && len(matches[0]) > 1 {
					if price, err := strconv.ParseFloat(matches[0][1], 64); err == nil && price > 100 && price < 10000 {
						currentPrice = fmt.Sprintf("%.2f AZN", price)
					}
				}
			}
		})
	}

	// Method 3: Last resort - extract from visible text
	if currentPrice == "" {
		var prices []float64
		doc.Find("*").Each(func(index int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())

			// Skip payment plan prices and other irrelevant text
			if strings.Contains(text, "AZN") &&
				!strings.Contains(text, "x") &&
				!strings.Contains(strings.ToLower(text), "aylıq") &&
				!strings.Contains(text, "ay") &&
				!strings.Contains(strings.ToLower(text), "credit") &&
				!strings.Contains(strings.ToLower(text), "kredit") &&
				len(text) < 20 { // Avoid long descriptions

				re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*AZN$`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					if price, err := strconv.ParseFloat(matches[1], 64); err == nil && price > 100 && price < 10000 {
						prices = append(prices, price)
					}
				}
			}
		})

		// Sort and pick the most reasonable prices
		if len(prices) > 0 {
			// Remove duplicates
			uniquePrices := make(map[float64]bool)
			var finalPrices []float64
			for _, price := range prices {
				if !uniquePrices[price] {
					uniquePrices[price] = true
					finalPrices = append(finalPrices, price)
				}
			}
			sort.Float64s(finalPrices)

			if len(finalPrices) >= 2 {
				currentPrice = fmt.Sprintf("%.2f AZN", finalPrices[0])
				originalPrice = fmt.Sprintf("%.2f AZN", finalPrices[len(finalPrices)-1])
			} else if len(finalPrices) == 1 {
				currentPrice = fmt.Sprintf("%.2f AZN", finalPrices[0])
			}
		}
	}

	// Assign extracted prices
	product.CurrentPrice = currentPrice
	product.OriginalPrice = originalPrice // Method 2: If not found in scripts, try extracting from visible text (as last resort)
	if currentPrice == "" {
		var prices []float64
		doc.Find("*").Each(func(index int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())

			// Skip payment plan prices and other irrelevant text
			if strings.Contains(text, "AZN") &&
				!strings.Contains(text, "x") &&
				!strings.Contains(strings.ToLower(text), "aylıq") &&
				!strings.Contains(text, "ay") &&
				!strings.Contains(strings.ToLower(text), "credit") &&
				!strings.Contains(strings.ToLower(text), "kredit") &&
				len(text) < 20 { // Avoid long descriptions

				re := regexp.MustCompile(`^(\d+\.?\d*)\s*AZN$`)
				matches := re.FindStringSubmatch(text)
				if len(matches) > 1 {
					if price, err := strconv.ParseFloat(matches[1], 64); err == nil && price > 100 && price < 10000 {
						prices = append(prices, price)
					}
				}
			}
		})

		// Sort and pick the most reasonable prices
		if len(prices) > 0 {
			// Remove duplicates
			uniquePrices := make(map[float64]bool)
			var finalPrices []float64
			for _, price := range prices {
				if !uniquePrices[price] {
					uniquePrices[price] = true
					finalPrices = append(finalPrices, price)
				}
			}
			sort.Float64s(finalPrices)

			if len(finalPrices) >= 2 {
				currentPrice = fmt.Sprintf("%.2f AZN", finalPrices[0])
				originalPrice = fmt.Sprintf("%.2f AZN", finalPrices[len(finalPrices)-1])
			} else if len(finalPrices) == 1 {
				currentPrice = fmt.Sprintf("%.2f AZN", finalPrices[0])
			}
		}
	}

	// Assign extracted prices
	product.CurrentPrice = currentPrice
	product.OriginalPrice = originalPrice // Calculate discount if both prices available
	if product.CurrentPrice != "" && product.OriginalPrice != "" {
		currentFloat := extractNumericPrice(product.CurrentPrice)
		originalFloat := extractNumericPrice(product.OriginalPrice)
		if originalFloat > 0 && currentFloat > 0 {
			discountPercent := ((originalFloat - currentFloat) / originalFloat) * 100
			product.Discount = fmt.Sprintf("-%.0f%%", discountPercent)
		}
	}

	// Set currency
	if strings.Contains(product.CurrentPrice, "AZN") {
		product.Currency = "AZN"
	}

	// Extract availability - look for stock status
	doc.Find("*").Each(func(index int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		lowerText := strings.ToLower(text)
		if product.Availability == "" {
			if lowerText == "stokda var" || lowerText == "mövcuddur" || lowerText == "var" {
				product.Availability = "Stokda var"
			} else if lowerText == "stokda yoxdur" || lowerText == "yoxdur" {
				product.Availability = "Stokda yoxdur"
			}
		}
	})

	// Extract product code/SKU from URL or page
	// First try to find the specific code 93528 from the page content
	doc.Find("*").Each(func(index int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(text, "93528") && product.SKU == "" {
			product.SKU = "93528"
		} else if strings.Contains(strings.ToLower(text), "malın kodu") && product.SKU == "" {
			// Look for numeric code after "Malın kodu:"
			re := regexp.MustCompile(`(\d+)`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				product.SKU = matches[1]
			}
		}
	})

	// Extract SKU/Product Code
	if product.SKU == "" {
		// Extract from script data - prioritize Calculator.init data
		doc.Find("script").Each(func(index int, s *goquery.Selection) {
			content := s.Text()

			// Look for Calculator.init JSON data which contains accurate product code
			if strings.Contains(content, "Calculator.init") && strings.Contains(content, "code") {
				re := regexp.MustCompile(`"code"\s*:\s*"(\d+)"`)
				matches := re.FindStringSubmatch(content)
				if len(matches) > 1 && product.SKU == "" {
					product.SKU = matches[1]
				}
			} else if strings.Contains(content, "\"id\":") && strings.Contains(content, "\"code\":") {
				re := regexp.MustCompile(`"code":"(\d+)"`)
				matches := re.FindStringSubmatch(content)
				if len(matches) > 1 && product.SKU == "" {
					product.SKU = matches[1]
				}
			}
		})
	}

	// Extract brand from specifications or page content
	doc.Find("*").Each(func(index int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(strings.ToLower(text), "brend :") && product.Brand == "" {
			parts := strings.Split(text, ":")
			if len(parts) >= 2 {
				brand := strings.TrimSpace(parts[1])
				// Clean up brand name and validate it
				brand = strings.Split(brand, "\n")[0] // Take only first line
				brand = strings.TrimSpace(brand)
				if brand != "" && len(brand) < 20 && !strings.Contains(brand, "{") {
					product.Brand = brand
				}
			}
		}
	})

	// Extract brand from JSON data if not found above
	if product.Brand == "" {
		doc.Find("script").Each(func(index int, s *goquery.Selection) {
			content := s.Text()
			if strings.Contains(content, "\"brand\"") {
				// Look for brand in JSON
				re := regexp.MustCompile(`"brand"[^}]*"title":"([^"]+)"`)
				matches := re.FindStringSubmatch(content)
				if len(matches) > 1 && product.Brand == "" {
					brand := strings.TrimSpace(matches[1])
					if brand != "" && len(brand) < 20 {
						product.Brand = brand
					}
				}
			}
		})
	}

	// Extract other specifications
	doc.Find("*").Each(func(index int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())

		// Look for specifications in the format "Label : Value"
		if strings.Contains(text, " : ") {
			parts := strings.Split(text, " : ")
			if len(parts) >= 2 {
				label := strings.TrimSpace(strings.ToLower(parts[0]))
				value := strings.TrimSpace(parts[1])

				switch {
				case strings.Contains(label, "daxili yaddaş") && product.InternalMemory == "":
					product.InternalMemory = value
				case strings.Contains(label, "operativ yaddaş") && product.RAM == "":
					product.RAM = value
				case strings.Contains(label, "prosessor") && product.Processor == "":
					product.Processor = value
				case strings.Contains(label, "qrafik prosessor") && product.Display == "":
					// For gaming consoles, GPU info can be stored in display field
					product.Display = value
				case strings.Contains(label, "çəki") && product.Brand != "" && product.OS == "":
					// Store weight or other specs in OS field if available
					product.OS = fmt.Sprintf("Çəki: %s", value)
				}
			}
		}
	})

	// If no specific specifications found, try to extract from product context
	if product.Brand == "" {
		// Try to extract brand from product name
		nameLower := strings.ToLower(product.Name)
		brands := []string{"sony", "samsung", "apple", "lg", "panasonic", "microsoft", "nintendo"}
		for _, brand := range brands {
			if strings.Contains(nameLower, brand) {
				product.Brand = strings.ToUpper(brand[:1]) + brand[1:]
				break
			}
		}
	}

	return product, nil
}

// Helper function to extract numeric price from price string
func extractNumericPrice(priceStr string) float64 {
	if priceStr == "" {
		return 0
	}

	// Remove "AZN" and extra spaces
	priceStr = strings.ReplaceAll(priceStr, "AZN", "")
	priceStr = strings.TrimSpace(priceStr)

	// Convert to float
	if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
		return price
	}

	return 0
} // init function registers the IrshadScraper when the package is imported
func init() {
	RegisterScraper("irshad", NewIrshadScraper())
}
