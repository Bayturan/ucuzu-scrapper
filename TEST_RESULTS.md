# Web Scraper Test Results

## Date: October 17, 2025

### ✅ KONTAKT.AZ Scraper (with Chromedp)

#### Test 1: iPhone 13 (With Discount)
- **URL**: https://kontakt.az/iphone-13-128-gb-midnight
- **Status**: ✅ SUCCESS
- **Current Price**: 1.149,99 ₼
- **Original Price**: 1.379,99 ₼
- **Discount**: -230₼
- **Response Time**: ~24 seconds

#### Test 2: PlayStation 5 (No Discount)
- **URL**: https://kontakt.az/playstation-5-slim-1-tb-abunelik
- **Status**: ✅ SUCCESS
- **Current Price**: 1.439,99 ₼
- **Original Price**: 1.439,99 ₼ (same as current - no discount)
- **Discount**: N/A
- **Response Time**: ~23 seconds

#### Test 3: Alzzoni Divan (With Discount)
- **URL**: https://kontakt.az/alzzoni-frost-divan-face-4
- **Status**: ✅ SUCCESS
- **Current Price**: 449,99 ₼
- **Original Price**: 849,99 ₼
- **Discount**: -400₼
- **Response Time**: ~25 seconds

#### Test 4: Samsung Galaxy S25 (With Discount)
- **URL**: https://kontakt.az/samsung-galaxy-s25-sm-s931b-12-128-gb-navy
- **Status**: ✅ SUCCESS
- **Current Price**: 1.599,99 ₼
- **Original Price**: 1.899,99 ₼
- **Discount**: -300₼
- **Response Time**: ~26 seconds

### ✅ IRSHAD.AZ Scraper

#### Test 5: iPhone 13 (With Discount)
- **URL**: https://irshad.az/az/mehsullar/iphone-13-128-gb-starlight-861668
- **Status**: ✅ SUCCESS
- **Current Price**: 1229.99 AZN
- **Original Price**: 1349.99 AZN
- **Brand**: Apple
- **Response Time**: <1 second

---

## Summary

### Kontakt.az Scraper
- **Technology**: Chromedp (headless browser automation)
- **Why**: Bypasses Cloudflare protection
- **Performance**: 20-30 seconds per request
- **Accuracy**: 100% - correctly extracts current/original prices and discounts
- **Handles**:
  - ✅ Products with discounts
  - ✅ Products without discounts
  - ✅ Dynamic JavaScript-rendered content
  - ✅ Cloudflare protection

### Irshad.az Scraper
- **Technology**: Standard HTTP + goquery
- **Performance**: <1 second per request
- **Accuracy**: 100% - correctly extracts prices from Calculator.init JSON
- **Handles**:
  - ✅ Products with discounts
  - ✅ Dynamic JavaScript data extraction
  - ✅ Multiple price formats

### Recommendations
1. **Caching**: Implement Redis/in-memory cache for Kontakt scraper (20-30 min TTL)
2. **Queue System**: Use background workers for Kontakt scraping to avoid timeout issues
3. **Rate Limiting**: Implement rate limiting to avoid overloading target sites
4. **Monitoring**: Add logging and alerts for failed scrapes
5. **API Alternative**: Consider contacting Kontakt.az for official API access

### Known Limitations
- **Kontakt.az**: Slow due to Cloudflare bypass requirement (20-30 seconds)
- **Irshad.az**: Fast but depends on their Calculator.init JSON structure

### API Endpoints
- `GET /api/v1/scrape?site=kontakt&uri=<url>` - Scrape Kontakt.az
- `GET /api/v1/scrape?site=irshad&uri=<url>` - Scrape Irshad.az
- `GET /api/v1/health` - Health check
- `GET /api/v1/sites` - List available scrapers
