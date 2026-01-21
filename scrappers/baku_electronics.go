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

// BakuElectronicsScraper implements the Scraper interface for bakuelectronics.az
type BakuElectronicsScraper struct{}

// NewBakuElectronicsScraper creates a new instance of BakuElectronicsScraper
func NewBakuElectronicsScraper() *BakuElectronicsScraper {
	return &BakuElectronicsScraper{}
}

// GetSiteName returns the site name
func (b *BakuElectronicsScraper) GetSiteName() string {
	return "Baku Electronics"
}

// IsValidURL checks if the URL belongs to bakuelectronics.az
func (b *BakuElectronicsScraper) IsValidURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "bakuelectronics.az")
}

// Scrape extracts product information from bakuelectronics.az URL
func (b *BakuElectronicsScraper) Scrape(url string) (*Product, error) {
	if !b.IsValidURL(url) {
		return nil, fmt.Errorf("URL does not belong to bakuelectronics.az: %s", url)
	}

	product := &Product{
		URL:       url,
		Site:      "bakuelectronics",
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

	// Extract product name from title tag or product heading
	product.Name = b.extractProductName(doc)

	// Extract SKU from URL or product code
	product.SKU = b.extractSKU(url, doc)

	// Extract prices
	product.CurrentPrice, product.OriginalPrice, product.Discount = b.extractPrices(doc)

	// Set currency
	if strings.Contains(product.CurrentPrice, "₼") {
		product.Currency = "AZN"
	}

	// Extract availability
	product.Availability = b.extractAvailability(doc)

	// Extract rating and review count
	product.Rating, product.ReviewCount = b.extractRatingAndReviews(doc)

	// Extract specifications
	b.extractSpecifications(doc, product)

	return product, nil
}

// extractProductName extracts the product name from various possible locations
func (b *BakuElectronicsScraper) extractProductName(doc *goquery.Document) string {
	// Try different selectors for product name
	selectors := []string{
		"title",
		"h1",
		".ProdDetails_ProductDetailsContentRight h1",
		".product-title",
		".ProductDetailsContentRight h1",
		"[data-testid='product-title']",
	}

	for _, selector := range selectors {
		found := ""
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" && found == "" {
				// Clean up the title text (remove "qiyməti ✅" suffix if present)
				found = strings.Replace(text, " qiyməti ✅", "", 1)
			}
		})
		if found != "" {
			return found
		}
	}

	// Fallback: extract from title tag
	title := doc.Find("title").Text()
	if title != "" {
		// Remove common suffixes from title
		title = strings.Replace(title, " qiyməti ✅", "", 1)
		title = strings.TrimSpace(title)
		return title
	}

	return ""
}

// extractSKU extracts the SKU from URL or document
func (b *BakuElectronicsScraper) extractSKU(url string, doc *goquery.Document) string {
	// Extract SKU from URL (last digits after the last dash)
	re := regexp.MustCompile(`-(\d+)/?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find SKU in the document
	sku := ""
	doc.Find("span, div, p").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "məhsul kodu") || strings.Contains(text, "kod") {
			// Extract number after "kod" or similar
			codeRegex := regexp.MustCompile(`kod[:\s]*(\d+)`)
			matches := codeRegex.FindStringSubmatch(text)
			if len(matches) > 1 && sku == "" {
				sku = matches[1]
			}
		}
	})

	return sku
}

// extractPrices extracts current price, original price, and discount
func (b *BakuElectronicsScraper) extractPrices(doc *goquery.Document) (string, string, string) {
	var currentPrice, originalPrice, discount string

	// Look for price patterns in the text
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())

		// Skip if text is too long (likely not a price)
		if len(text) > 50 {
			return
		}

		// Look for price patterns (number followed by ₼)
		priceRegex := regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*₼`)
		matches := priceRegex.FindAllString(text, -1)

		if len(matches) > 0 {
			// If we find multiple prices, likely original and current
			if len(matches) >= 2 {
				// Usually the first larger price is original, second smaller is current
				price1 := matches[0]
				price2 := matches[1]

				// Extract numeric values to compare
				num1 := b.extractNumericValue(price1)
				num2 := b.extractNumericValue(price2)

				if num1 > num2 {
					if originalPrice == "" {
						originalPrice = price1
					}
					if currentPrice == "" {
						currentPrice = price2
					}
				} else {
					if currentPrice == "" {
						currentPrice = price1
					}
					if originalPrice == "" && num2 > num1 {
						originalPrice = price2
					}
				}
			} else if currentPrice == "" {
				currentPrice = matches[0]
			}
		}

		// Look for discount pattern
		if strings.Contains(text, "-") && strings.Contains(text, "₼") {
			discountRegex := regexp.MustCompile(`-\s*(\d+(?:[.,]\d+)?)\s*₼`)
			discountMatches := discountRegex.FindStringSubmatch(text)
			if len(discountMatches) > 1 && discount == "" {
				discount = "-" + discountMatches[1] + " ₼"
			}
		}
	})

	return currentPrice, originalPrice, discount
}

// extractNumericValue extracts numeric value from price string for comparison
func (b *BakuElectronicsScraper) extractNumericValue(price string) float64 {
	re := regexp.MustCompile(`(\d+(?:[.,]\d+)?)`)
	match := re.FindString(price)
	if match != "" {
		// Replace comma with dot for parsing
		match = strings.Replace(match, ",", ".", 1)
		// Simple conversion (we don't need precise float parsing for comparison)
		var val float64
		fmt.Sscanf(match, "%f", &val)
		return val
	}
	return 0
}

// extractAvailability extracts availability status
func (b *BakuElectronicsScraper) extractAvailability(doc *goquery.Document) string {
	// Look for availability indicators
	keywords := []string{
		"mövcud",
		"yoxdur",
		"stok",
		"available",
		"stock",
		"in stock",
		"out of stock",
	}

	for _, keyword := range keywords {
		doc.Find("*").Each(func(i int, s *goquery.Selection) {
			text := strings.ToLower(strings.TrimSpace(s.Text()))
			if strings.Contains(text, keyword) && len(text) < 100 {
				return
			}
		})
	}

	// Check for add to cart button as availability indicator
	addToCartFound := false
	doc.Find("button, a").Each(func(i int, s *goquery.Selection) {
		text := strings.ToLower(s.Text())
		if strings.Contains(text, "səbətə") || strings.Contains(text, "cart") || strings.Contains(text, "al") {
			addToCartFound = true
			return
		}
	})

	if addToCartFound {
		return "Mövcuddur"
	}

	return "Məlumat yoxdur"
}

// extractRatingAndReviews extracts rating and review count
func (b *BakuElectronicsScraper) extractRatingAndReviews(doc *goquery.Document) (string, string) {
	var rating, reviewCount string

	// Look for rating patterns
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())

		// Look for star rating patterns (like "4.5" or "★★★★☆")
		ratingRegex := regexp.MustCompile(`([0-5](?:[.,]\d)?)\s*(?:\/\s*5|★|stars?)`)
		matches := ratingRegex.FindStringSubmatch(text)
		if len(matches) > 1 && rating == "" {
			rating = matches[1]
		}

		// Look for review count patterns
		reviewRegex := regexp.MustCompile(`(\d+)\s*(?:rəy|review|отзыв)`)
		reviewMatches := reviewRegex.FindStringSubmatch(text)
		if len(reviewMatches) > 1 && reviewCount == "" {
			reviewCount = reviewMatches[1]
		}
	})

	return rating, reviewCount
}

// extractSpecifications extracts technical specifications
func (b *BakuElectronicsScraper) extractSpecifications(doc *goquery.Document, product *Product) {
	// Look for specification sections
	doc.Find("table, .specs, .specifications, .features, .xususiyyetler").Each(func(i int, table *goquery.Selection) {
		table.Find("tr, .spec-row, .feature-row").Each(func(j int, row *goquery.Selection) {
			var label, value string

			// Try different structures
			cells := row.Find("td, .spec-label, .spec-value, th")
			if cells.Length() >= 2 {
				label = strings.TrimSpace(cells.Eq(0).Text())
				value = strings.TrimSpace(cells.Eq(1).Text())
			} else {
				// Try label: value pattern
				text := row.Text()
				if strings.Contains(text, ":") {
					parts := strings.Split(text, ":")
					if len(parts) >= 2 {
						label = strings.TrimSpace(parts[0])
						value = strings.TrimSpace(parts[1])
					}
				}
			}

			if label != "" && value != "" {
				b.assignSpecification(label, value, product)
			}
		})
	})

	// Also check for inline specifications in the product description
	doc.Find("p, div, span").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if len(text) < 200 && strings.Contains(text, ":") { // Reasonable length for specs
			parts := strings.Split(text, ":")
			if len(parts) == 2 {
				label := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if len(label) < 50 && len(value) < 100 { // Reasonable spec length
					b.assignSpecification(label, value, product)
				}
			}
		}
	})

	// Extract brand from product name if not found in specs
	if product.Brand == "" {
		product.Brand = b.extractBrandFromName(product.Name)
	}
}

// assignSpecification assigns specification value to appropriate product field
func (b *BakuElectronicsScraper) assignSpecification(label, value string, product *Product) {
	labelLower := strings.ToLower(label)

	switch {
	case strings.Contains(labelLower, "brand") || strings.Contains(labelLower, "brend") || strings.Contains(labelLower, "marka"):
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
		} else if !strings.Contains(product.Processor, value) {
			product.Processor = product.Processor + " " + value
		}
	case strings.Contains(labelLower, "əməliyyat") && strings.Contains(labelLower, "sistem"):
		if product.OS == "" {
			product.OS = value
		}
	case strings.Contains(labelLower, "ekran") || strings.Contains(labelLower, "displey") || strings.Contains(labelLower, "display"):
		if product.Display == "" {
			product.Display = value
		}
	}
}

// extractBrandFromName extracts brand name from product name
func (b *BakuElectronicsScraper) extractBrandFromName(name string) string {
	// Common smartphone brands
	brands := []string{
		"Apple", "Samsung", "Xiaomi", "Huawei", "OnePlus", "Google", "Sony",
		"LG", "Motorola", "Nokia", "Oppo", "Vivo", "Realme", "Honor", "Poco",
		"Tecno", "Infinix", "Nothing", "Asus", "ZTE", "TCL", "Alcatel",
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

// init function registers the BakuElectronicsScraper when the package is imported
func init() {
	RegisterScraper("bakuelectronics", NewBakuElectronicsScraper())
}
