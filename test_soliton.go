package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"web-scrappers/scrappers"
)

// TestSoliton tests the Soliton scraper
// To run this test, comment out the main function in main.go and uncomment the main function below
func TestSoliton() {
	// Test URL for Riffel ASW-JF 120/121 air conditioner
	testURL := "https://soliton.az/az/meiset-texnikasi/kondisionerler/20240709030951581-riffel-asw-jf-120-121.html"
	
	// Get the scraper
	scraper, err := scrappers.GetScraper("soliton")
	if err != nil {
		log.Fatalf("Failed to get scraper: %v", err)
	}

	// Test if URL is valid
	if !scraper.IsValidURL(testURL) {
		log.Fatalf("URL is not valid for this scraper: %s", testURL)
	}

	fmt.Printf("Testing Soliton scraper with URL: %s\n", testURL)
	fmt.Printf("Scraper: %s\n", scraper.GetSiteName())
	
	// Set debug mode
	os.Setenv("DEBUG", "1")
	
	// Scrape the product
	product, err := scraper.Scrape(testURL)
	if err != nil {
		log.Fatalf("Failed to scrape product: %v", err)
	}

	// Print the results in JSON format
	jsonData, err := json.MarshalIndent(product, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("\nScraped Product Data:")
	fmt.Println(string(jsonData))
}

// Uncomment this to test the scraper directly
// func main() {
// 	TestSoliton()
// }