#!/bin/bash

# Web Scraper API Test Script
# This script demonstrates how to use the web scraper REST API

echo "=== Web Scraper API Test ==="
echo

# Base URL for the API
API_BASE="http://localhost:8080/api/v1"

# Test health endpoint
echo "1. Testing health endpoint..."
curl -s "${API_BASE}/health" | python3 -m json.tool 2>/dev/null || curl -s "${API_BASE}/health"
echo -e "\n"

# Test sites endpoint
echo "2. Testing sites endpoint..."
curl -s "${API_BASE}/sites" | python3 -m json.tool 2>/dev/null || curl -s "${API_BASE}/sites"
echo -e "\n"

# Test scraping with kontakt.az
echo "3. Testing scraping from kontakt.az..."
KONTAKT_URL="https://kontakt.az/iphone-13-128-gb-midnight"
curl -s "${API_BASE}/scrape?site=kontakt&uri=${KONTAKT_URL}" | python3 -m json.tool 2>/dev/null || curl -s "${API_BASE}/scrape?site=kontakt&uri=${KONTAKT_URL}"
echo -e "\n"

# Test error handling - unsupported site
echo "4. Testing error handling (unsupported site)..."
curl -s "${API_BASE}/scrape?site=unknownsite&uri=https://example.com"
echo -e "\n"

# Test error handling - missing parameters
echo "5. Testing error handling (missing parameters)..."
curl -s "${API_BASE}/scrape?site=kontakt"
echo -e "\n"

echo "=== Test Complete ==="