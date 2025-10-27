import requests
import json
from typing import Optional, Dict, Any

class WebScraperClient:
    """Client for the Web Scraper REST API."""
    
    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url
    
    def health_check(self) -> Dict[str, Any]:
        """Check if the API server is healthy."""
        response = requests.get(f"{self.base_url}/health")
        response.raise_for_status()
        return response.json()
    
    def get_supported_sites(self) -> Dict[str, Any]:
        """Get list of supported sites."""
        response = requests.get(f"{self.base_url}/sites")
        response.raise_for_status()
        return response.json()
    
    def scrape_product(self, site: str, uri: str) -> Dict[str, Any]:
        """Scrape product information from a site."""
        params = {
            "site": site,
            "uri": uri
        }
        response = requests.get(f"{self.base_url}/scrape", params=params)
        response.raise_for_status()
        return response.json()
    
    def get_current_price(self, site: str, uri: str) -> Optional[str]:
        """Get only the current price of a product."""
        try:
            product = self.scrape_product(site, uri)
            return product.get("current_price")
        except requests.RequestException:
            return None

def main():
    """Example usage of the WebScraperClient."""
    client = WebScraperClient()
    
    try:
        # Check API health
        print("=== API Health Check ===")
        health = client.health_check()
        print(f"Status: {health['status']}")
        print(f"Version: {health['version']}")
        print()
        
        # Get supported sites
        print("=== Supported Sites ===")
        sites = client.get_supported_sites()
        for site in sites["supported_sites"]:
            print(f"- {site['name']} ({site['identifier']})")
        print()
        
        # Scrape a product
        print("=== Scraping Product ===")
        kontakt_url = "https://kontakt.az/iphone-13-128-gb-midnight"
        product = client.scrape_product("kontakt", kontakt_url)
        
        print(f"Product: {product['name']}")
        print(f"Price: {product['current_price']}")
        print(f"Brand: {product.get('brand', 'N/A')}")
        print(f"Scraped at: {product['scraped_at']}")
        print()
        
        # Get just the price
        print("=== Current Price Only ===")
        price = client.get_current_price("kontakt", kontakt_url)
        print(f"Current Price: {price}")
        
    except requests.RequestException as e:
        print(f"API Error: {e}")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()