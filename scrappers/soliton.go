package scrappers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// SolitonScraper implements the Scraper interface for soliton.az
type SolitonScraper struct{}

// NewSolitonScraper creates a new instance of SolitonScraper
func NewSolitonScraper() *SolitonScraper {
	return &SolitonScraper{}
}

// GetSiteName returns the site name
func (s *SolitonScraper) GetSiteName() string {
	return "Soliton.az"
}

// IsValidURL checks if the URL belongs to soliton.az
func (s *SolitonScraper) IsValidURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "soliton.az")
}

// Scrape extracts product information from soliton.az URL
func (s *SolitonScraper) Scrape(url string) (*Product, error) {
	if !s.IsValidURL(url) {
		return nil, fmt.Errorf("URL does not belong to soliton.az: %s", url)
	}

	product := &Product{
		URL:       url,
		Site:      "soliton",
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Use chromedp to handle JavaScript-heavy site
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the entire operation
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	if os.Getenv("DEBUG") == "1" {
		fmt.Printf("DEBUG: Starting chromedp for URL: %s\n", url)
	}

	var htmlContent string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second), // Wait for page load
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Additional wait for dynamic content
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		if os.Getenv("DEBUG") == "1" {
			fmt.Printf("DEBUG: chromedp error: %v\n", err)
		}
		return nil, fmt.Errorf("failed to fetch URL with chromedp: %w", err)
	}

	if os.Getenv("DEBUG") == "1" {
		fmt.Printf("DEBUG: Successfully fetched HTML (length: %d bytes)\n", len(htmlContent))
		os.WriteFile("debug.html", []byte(htmlContent), 0644)
		fmt.Println("Debug HTML saved to debug.html")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract product name
	product.Name = s.extractProductName(doc)

	// Extract SKU from URL
	product.SKU = s.extractSKU(url, doc)

	// Extract prices
	product.CurrentPrice, product.OriginalPrice, product.Discount = s.extractPrices(doc)

	// Set currency - Soliton uses "M" for AZN
	if strings.Contains(product.CurrentPrice, "M") {
		product.Currency = "AZN"
	}

	// Extract availability
	product.Availability = s.extractAvailability(doc)

	// Extract rating and review count
	product.Rating, product.ReviewCount = s.extractRatingAndReviews(doc)

	// Extract specifications
	s.extractSpecifications(doc, product)

	return product, nil
}

// extractProductName extracts the product name from various possible locations
func (s *SolitonScraper) extractProductName(doc *goquery.Document) string {
	// Try different selectors for product name
	selectors := []string{
		"h1",
		"title",
		".product-title",
		".product-name",
		"[data-product-name]",
	}

	for _, selector := range selectors {
		found := ""
		doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if text != "" && found == "" {
				found = text
			}
		})
		if found != "" {
			return found
		}
	}

	// Fallback: extract from title tag
	title := doc.Find("title").Text()
	if title != "" {
		return strings.TrimSpace(title)
	}

	return ""
}

// extractSKU extracts the SKU from URL or document
func (s *SolitonScraper) extractSKU(url string, doc *goquery.Document) string {
	// Extract SKU from URL (timestamp-based ID pattern in Soliton URLs)
	re := regexp.MustCompile(`/(\d{17})-`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find product ID in the document
	sku := ""
	doc.Find("script, meta, [data-product-id], [item_id]").Each(func(i int, sel *goquery.Selection) {
		// Check attributes
		if id, exists := sel.Attr("item_id"); exists && sku == "" {
			sku = id
			return
		}
		if id, exists := sel.Attr("data-product-id"); exists && sku == "" {
			sku = id
			return
		}

		// Check script content for product ID
		text := sel.Text()
		if strings.Contains(text, "item_id") {
			idRegex := regexp.MustCompile(`item_id[=:]\s*['"]?(\d+)['"]?`)
			matches := idRegex.FindStringSubmatch(text)
			if len(matches) > 1 && sku == "" {
				sku = matches[1]
			}
		}
	})

	return sku
}

// extractPrices extracts current price, original price, and discount
func (s *SolitonScraper) extractPrices(doc *goquery.Document) (string, string, string) {
	var currentPrice, originalPrice, discount string

	// Look for price patterns specific to Soliton (using "M" as currency)
	doc.Find("*").Each(func(i int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())

		// Skip if text is too long (likely not a price)
		if len(text) > 100 {
			return
		}

		// Look for price patterns (number followed by M)
		priceRegex := regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*M`)
		matches := priceRegex.FindAllString(text, -1)

		if len(matches) > 0 {
			// If we find multiple prices in same element, likely original and current
			if len(matches) >= 2 {
				// Usually first is original, second is current (discounted)
				price1 := matches[0]
				price2 := matches[1]

				// Extract numeric values to compare
				num1 := s.extractNumericValue(price1)
				num2 := s.extractNumericValue(price2)

				if num1 > num2 {
					if originalPrice == "" {
						originalPrice = price1
					}
					if currentPrice == "" {
						currentPrice = price2
					}
				} else if num2 > num1 {
					if originalPrice == "" {
						originalPrice = price2
					}
					if currentPrice == "" {
						currentPrice = price1
					}
				}
			} else if currentPrice == "" {
				currentPrice = matches[0]
			}
		}
	})

	// Calculate discount if both prices are available
	if originalPrice != "" && currentPrice != "" {
		origVal := s.extractNumericValue(originalPrice)
		currVal := s.extractNumericValue(currentPrice)
		if origVal > currVal {
			discountVal := origVal - currVal
			discount = fmt.Sprintf("%.2f M", discountVal)
		}
	}

	return currentPrice, originalPrice, discount
}

// extractNumericValue extracts numeric value from price string for comparison
func (s *SolitonScraper) extractNumericValue(price string) float64 {
	re := regexp.MustCompile(`(\d+(?:[.,]\d+)?)`)
	match := re.FindString(price)
	if match != "" {
		// Replace comma with dot for parsing
		match = strings.Replace(match, ",", ".", 1)
		var val float64
		fmt.Sscanf(match, "%f", &val)
		return val
	}
	return 0
}

// extractAvailability extracts availability status
func (s *SolitonScraper) extractAvailability(doc *goquery.Document) string {
	// Look for buy buttons or availability indicators
	foundBuyButton := false

	doc.Find("a, button").Each(func(i int, sel *goquery.Selection) {
		text := strings.ToLower(sel.Text())
		href, _ := sel.Attr("href")

		// Check for buy links or buttons
		if (strings.Contains(text, "nağd") || strings.Contains(text, "hissə") ||
			strings.Contains(text, "buy") || strings.Contains(href, "buy.php")) && !foundBuyButton {
			foundBuyButton = true
		}
	})

	// Look for specific availability text
	availabilityKeywords := []string{
		"mövcud",
		"yoxdur",
		"stokda",
		"çatdırılma",
		"available",
		"stock",
	}

	for _, keyword := range availabilityKeywords {
		found := ""
		doc.Find("*").Each(func(i int, sel *goquery.Selection) {
			text := strings.ToLower(strings.TrimSpace(sel.Text()))
			if strings.Contains(text, keyword) && len(text) < 100 && found == "" {
				found = sel.Text()
			}
		})
		if found != "" {
			return found
		}
	}

	if foundBuyButton {
		return "Mövcuddur"
	}

	return "Məlumat yoxdur"
}

// extractRatingAndReviews extracts rating and review count
func (s *SolitonScraper) extractRatingAndReviews(doc *goquery.Document) (string, string) {
	var rating, reviewCount string

	// Look for rating and review patterns
	doc.Find("*").Each(func(i int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())

		// Look for rating patterns (like "4.7")
		ratingRegex := regexp.MustCompile(`([0-5](?:[.,]\d)?)\s*(?:\/\s*5|★|stars?)?`)
		matches := ratingRegex.FindStringSubmatch(text)
		if len(matches) > 1 && rating == "" {
			// Validate it's actually a rating (reasonable range)
			val := s.extractNumericValue(matches[1])
			if val > 0 && val <= 5 {
				rating = matches[1]
			}
		}

		// Look for review count patterns (like "121 səs")
		reviewRegex := regexp.MustCompile(`(\d+)\s*(?:səs|rəy|review|отзыв)`)
		reviewMatches := reviewRegex.FindStringSubmatch(text)
		if len(reviewMatches) > 1 && reviewCount == "" {
			reviewCount = reviewMatches[1]
		}
	})

	return rating, reviewCount
}

// extractSpecifications extracts technical specifications from tables
func (s *SolitonScraper) extractSpecifications(doc *goquery.Document, product *Product) {
	// Look for specification tables
	doc.Find("table, .specs, .specifications").Each(func(i int, table *goquery.Selection) {
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td, th")
			if cells.Length() >= 2 {
				label := strings.TrimSpace(cells.Eq(0).Text())
				value := strings.TrimSpace(cells.Eq(1).Text())

				if label != "" && value != "" {
					s.assignSpecification(label, value, product)
				}
			}
		})
	})

	// Also check for specifications in div structures
	doc.Find("div, p, span").Each(func(i int, sel *goquery.Selection) {
		text := sel.Text()
		if len(text) < 200 && strings.Contains(text, ":") {
			parts := strings.Split(text, ":")
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if len(label) < 50 && len(value) < 100 {
					s.assignSpecification(label, value, product)
				}
			}
		}
	})

	// Extract brand from product name if not found in specs
	if product.Brand == "" {
		product.Brand = s.extractBrandFromName(product.Name)
	}
}

// assignSpecification assigns specification value to appropriate product field
func (s *SolitonScraper) assignSpecification(label, value string, product *Product) {
	labelLower := strings.ToLower(label)
	
	switch {
	case strings.Contains(labelLower, "brand") || strings.Contains(labelLower, "brend") || 
		 strings.Contains(labelLower, "marka"):
		if product.Brand == "" {
			product.Brand = value
		}
	case strings.Contains(labelLower, "növ") && !strings.Contains(strings.ToLower(value), "kondisioner"):
		// Only use "növ" as brand if it's not describing the product type
		if product.Brand == "" {
			product.Brand = value
		}
	case strings.Contains(labelLower, "yaddaş") && strings.Contains(labelLower, "daxili"):
		if product.InternalMemory == "" {
			product.InternalMemory = value
		}
	case strings.Contains(labelLower, "ram") || (strings.Contains(labelLower, "yaddaş") && strings.Contains(labelLower, "operativ")):
		if product.RAM == "" {
			product.RAM = value
		}
	case strings.Contains(labelLower, "kamera") && (strings.Contains(labelLower, "əsas") || strings.Contains(labelLower, "arxa")):
		if product.MainCamera == "" {
			product.MainCamera = value
		}
	case strings.Contains(labelLower, "kamera") && (strings.Contains(labelLower, "ön") || strings.Contains(labelLower, "front")):
		if product.FrontCamera == "" {
			product.FrontCamera = value
		}
	case strings.Contains(labelLower, "prosessor") || strings.Contains(labelLower, "cpu") || strings.Contains(labelLower, "çip"):
		if product.Processor == "" {
			product.Processor = value
		}
	case strings.Contains(labelLower, "əməliyyat") && strings.Contains(labelLower, "sistem"):
		if product.OS == "" {
			product.OS = value
		}
	case strings.Contains(labelLower, "ekran") || strings.Contains(labelLower, "displey") || strings.Contains(labelLower, "display"):
		if product.Display == "" {
			product.Display = value
		}
	case strings.Contains(labelLower, "sahə") || strings.Contains(labelLower, "m2"):
		// For appliances like air conditioners, store coverage area
		if product.Display == "" { // Reuse display field for coverage area
			product.Display = value
		}
	case strings.Contains(labelLower, "soyutma") || strings.Contains(labelLower, "isitmə") || 
		 strings.Contains(labelLower, "btu") || strings.Contains(labelLower, "cooling") || strings.Contains(labelLower, "heating"):
		// Store power/BTU info in processor field for appliances
		if product.Processor == "" {
			product.Processor = value
		} else if !strings.Contains(product.Processor, value) {
			product.Processor = product.Processor + " / " + value
		}
	case strings.Contains(labelLower, "rəng") || strings.Contains(labelLower, "color"):
		// Store color info if no other suitable field
		if product.OS == "" {
			product.OS = value
		}
	case strings.Contains(labelLower, "çəki") || strings.Contains(labelLower, "weight"):
		// Store weight info in RAM field for appliances (since RAM not applicable)
		if product.RAM == "" {
			product.RAM = value
		}
	case strings.Contains(labelLower, "ölçü") || strings.Contains(labelLower, "size") || strings.Contains(labelLower, "dimension"):
		// Store dimensions in internal memory field for appliances
		if product.InternalMemory == "" {
			product.InternalMemory = value
		}
	}
}// extractBrandFromName extracts brand name from product name
func (s *SolitonScraper) extractBrandFromName(name string) string {
	// Common brands found on Soliton
	brands := []string{
		"Samsung", "LG", "Riffel", "Bosch", "Electrolux", "Beko", "Arçelik",
		"Whirlpool", "Haier", "Midea", "TCL", "Panasonic", "Sony", "Philips",
		"Xiaomi", "Apple", "Huawei", "Asus", "Acer", "HP", "Dell", "Lenovo",
	}

	nameLower := strings.ToLower(name)
	for _, brand := range brands {
		if strings.Contains(nameLower, strings.ToLower(brand)) {
			return brand
		}
	}

	// Try to extract first word as brand if it looks like a brand name
	words := strings.Fields(name)
	if len(words) > 0 {
		firstWord := words[0]
		// If first word is likely a brand (starts with capital, reasonable length)
		if len(firstWord) >= 2 && len(firstWord) <= 15 {
			return firstWord
		}
	}

	return ""
}

// init function registers the SolitonScraper when the package is imported
func init() {
	RegisterScraper("soliton", NewSolitonScraper())
}
