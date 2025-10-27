/**
 * Web Scraper API Client
 * A JavaScript client for the Web Scraper REST API
 */

class WebScraperClient {
    constructor(baseUrl = 'http://localhost:8080/api/v1') {
        this.baseUrl = baseUrl;
    }

    /**
     * Check if the API server is healthy
     */
    async healthCheck() {
        const response = await fetch(`${this.baseUrl}/health`);
        if (!response.ok) {
            throw new Error(`Health check failed: ${response.status}`);
        }
        return response.json();
    }

    /**
     * Get list of supported sites
     */
    async getSupportedSites() {
        const response = await fetch(`${this.baseUrl}/sites`);
        if (!response.ok) {
            throw new Error(`Failed to get sites: ${response.status}`);
        }
        return response.json();
    }

    /**
     * Scrape product information from a site
     */
    async scrapeProduct(site, uri) {
        const params = new URLSearchParams({
            site: site,
            uri: uri
        });
        
        const response = await fetch(`${this.baseUrl}/scrape?${params}`);
        if (!response.ok) {
            const error = await response.json();
            throw new Error(`Scraping failed: ${error.message || response.status}`);
        }
        return response.json();
    }

    /**
     * Get only the current price of a product
     */
    async getCurrentPrice(site, uri) {
        try {
            const product = await this.scrapeProduct(site, uri);
            return product.current_price;
        } catch (error) {
            console.error('Failed to get current price:', error);
            return null;
        }
    }
}

// Example usage
async function main() {
    const client = new WebScraperClient();

    try {
        // Check API health
        console.log('=== API Health Check ===');
        const health = await client.healthCheck();
        console.log(`Status: ${health.status}`);
        console.log(`Version: ${health.version}`);
        console.log();

        // Get supported sites
        console.log('=== Supported Sites ===');
        const sites = await client.getSupportedSites();
        sites.supported_sites.forEach(site => {
            console.log(`- ${site.name} (${site.identifier})`);
        });
        console.log();

        // Scrape a product
        console.log('=== Scraping Product ===');
        const kontaktUrl = 'https://kontakt.az/iphone-13-128-gb-midnight';
        const product = await client.scrapeProduct('kontakt', kontaktUrl);
        
        console.log(`Product: ${product.name}`);
        console.log(`Price: ${product.current_price}`);
        console.log(`Brand: ${product.brand || 'N/A'}`);
        console.log(`Scraped at: ${product.scraped_at}`);
        console.log();

        // Get just the price
        console.log('=== Current Price Only ===');
        const price = await client.getCurrentPrice('kontakt', kontaktUrl);
        console.log(`Current Price: ${price}`);

    } catch (error) {
        console.error('Error:', error.message);
    }
}

// For Node.js usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = WebScraperClient;
}

// For browser usage or direct execution
if (typeof window !== 'undefined') {
    window.WebScraperClient = WebScraperClient;
}

// Run example if this file is executed directly
if (typeof require !== 'undefined' && require.main === module) {
    main();
}