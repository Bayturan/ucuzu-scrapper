package scrappers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// KontaktScraper implements the Scraper interface for kontakt.az
type KontaktScraper struct{}

// NewKontaktScraper creates a new instance of KontaktScraper
func NewKontaktScraper() *KontaktScraper {
	return &KontaktScraper{}
}

// GetSiteName returns the site name
func (k *KontaktScraper) GetSiteName() string {
	return "Kontakt.az"
}

// IsValidURL checks if the URL belongs to kontakt.az
func (k *KontaktScraper) IsValidURL(url string) bool {
	return strings.Contains(strings.ToLower(url), "kontakt.az")
}

// Scrape extracts product information from kontakt.az URL
func (k *KontaktScraper) Scrape(url string) (*Product, error) {
	if !k.IsValidURL(url) {
		return nil, fmt.Errorf("URL does not belong to kontakt.az: %s", url)
	}

	product := &Product{
		URL:       url,
		Site:      "kontakt",
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Use chromedp to bypass Cloudflare and execute JavaScript
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
		chromedp.Sleep(3*time.Second), // Wait for page load and Cloudflare
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
	}

	if os.Getenv("DEBUG") == "1" {
		os.WriteFile("debug.html", []byte(htmlContent), 0644)
		fmt.Println("Debug HTML saved to debug.html")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract product name
	doc.Find("h1.page-title span, h1.page-title, span.base").Each(func(i int, s *goquery.Selection) {
		if product.Name == "" {
			product.Name = strings.TrimSpace(s.Text())
		}
	})

	// Extract SKU
	doc.Find("div.product.attribute.sku div.value").Each(func(i int, s *goquery.Selection) {
		product.SKU = strings.TrimSpace(s.Text())
	})

	// Extract prices - Kontakt.az specific structure
	// Current (discounted) price is in: prodCart__prices > strong > span (the first direct child span)
	// Original price is in: span[data-price-type="finalPrice"]

	// First, try to get the original price from data-price-type attribute
	doc.Find("span[data-price-type='finalPrice']").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if os.Getenv("DEBUG") == "1" {
			fmt.Printf("DEBUG: Found data-price-type=finalPrice text: '%s'\n", text)
		}
		if product.OriginalPrice == "" && strings.Contains(text, "₼") {
			product.OriginalPrice = text
		}
	})

	// Then get current price from the strong > span structure
	// We need to be careful to get only the direct child span, not nested ones
	doc.Find("div.prodCart__prices strong").Each(func(i int, s *goquery.Selection) {
		// Get the first direct child span of strong
		s.Find("span").First().Each(func(j int, span *goquery.Selection) {
			text := strings.TrimSpace(span.Text())
			if os.Getenv("DEBUG") == "1" {
				fmt.Printf("DEBUG: Found strong > span text: '%s'\n", text)
			}
			if product.CurrentPrice == "" && strings.Contains(text, "₼") {
				product.CurrentPrice = text
			}
		})
	})

	// If current price wasn't found, use the original price as fallback
	if product.CurrentPrice == "" && product.OriginalPrice != "" {
		product.CurrentPrice = product.OriginalPrice
		product.OriginalPrice = "" // Clear original since there's no discount
	}

	// Extract discount
	doc.Find("span i, div.label-discount span.cash").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if os.Getenv("DEBUG") == "1" {
			fmt.Printf("DEBUG: Found discount text: '%s'\n", text)
		}
		if product.Discount == "" && strings.Contains(text, "₼") && strings.Contains(text, "-") {
			// Extract just the "-XXX₼" part
			parts := strings.Split(text, " ")
			if len(parts) > 0 && strings.Contains(parts[0], "-") {
				product.Discount = parts[0]
			}
		}
	})

	// Set currency
	if strings.Contains(product.CurrentPrice, "₼") {
		product.Currency = "AZN"
	}

	// Extract availability
	doc.Find("div.stock span, div.stock.available span, div.stock").Each(func(i int, s *goquery.Selection) {
		if product.Availability == "" {
			text := strings.TrimSpace(s.Text())
			lowerText := strings.ToLower(text)
			if text != "" && (strings.Contains(lowerText, "mövcud") ||
				strings.Contains(lowerText, "available") ||
				strings.Contains(lowerText, "stock")) {
				product.Availability = text
			}
		}
	})

	// Extract rating
	doc.Find("div.rating-summary span.rating-result span, span[itemprop='ratingValue']").Each(func(i int, s *goquery.Selection) {
		if product.Rating == "" {
			product.Rating = strings.TrimSpace(s.Text())
		}
	})

	// Extract review count
	doc.Find("div.reviews-actions a.action.view, span[itemprop='reviewCount']").Each(func(i int, s *goquery.Selection) {
		if product.ReviewCount == "" {
			product.ReviewCount = strings.TrimSpace(s.Text())
		}
	})

	// Extract specifications from div.har__znach structure (kontakt.az specific)
	doc.Find("div.har__row").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("div.har__title").Text())
		value := strings.TrimSpace(s.Find("div.har__znach").Text())
		if label == "" || value == "" {
			return
		}
		labelLower := strings.ToLower(label)
		switch {
		case strings.Contains(labelLower, "brend"):
			product.Brand = value
		case strings.Contains(labelLower, "daxili yaddaş"):
			product.InternalMemory = value
		case strings.Contains(labelLower, "operativ yaddaş"):
			product.RAM = value
		case strings.Contains(labelLower, "əsas kamera"):
			product.MainCamera = value
		case strings.Contains(labelLower, "ön kamera"):
			product.FrontCamera = value
		case strings.Contains(labelLower, "prosessorun adı") || strings.Contains(labelLower, "prosessorun növü"):
			if product.Processor == "" {
				product.Processor = value
			} else if !strings.Contains(product.Processor, value) {
				product.Processor = product.Processor + " " + value
			}
		case strings.Contains(labelLower, "əməliyyat sistemi"):
			product.OS = value
		case strings.Contains(labelLower, "displey növü"):
			product.Display = value
		}
	})

	// Also try table format for specifications (fallback)
	doc.Find("table.data.table.additional-attributes tr, table.additional-attributes tr").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("th").Text())
		value := strings.TrimSpace(s.Find("td").Text())
		if label == "" || value == "" {
			return
		}
		labelLower := strings.ToLower(label)
		switch {
		case strings.Contains(labelLower, "brend") && product.Brand == "":
			product.Brand = value
		case strings.Contains(labelLower, "daxili yaddaş") && product.InternalMemory == "":
			product.InternalMemory = value
		case strings.Contains(labelLower, "operativ yaddaş") && product.RAM == "":
			product.RAM = value
		case strings.Contains(labelLower, "əsas kamera") && product.MainCamera == "":
			product.MainCamera = value
		case strings.Contains(labelLower, "ön kamera") && product.FrontCamera == "":
			product.FrontCamera = value
		case (strings.Contains(labelLower, "prosessorun adı") || strings.Contains(labelLower, "prosessorun növü")) && product.Processor == "":
			product.Processor = value
		case strings.Contains(labelLower, "əməliyyat sistemi") && product.OS == "":
			product.OS = value
		case strings.Contains(labelLower, "displey növü") && product.Display == "":
			product.Display = value
		}
	})

	return product, nil
}

// init function registers the KontaktScraper when the package is imported
func init() {
	RegisterScraper("kontakt", NewKontaktScraper())
}
