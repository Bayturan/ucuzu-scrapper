package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"web-scrappers/scrappers"

	"github.com/gorilla/mux"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type ScrapRequest struct {
	Site string `json:"site"`
	URI  string `json:"uri"`
}

func main() {
	r := mux.NewRouter()

	// API endpoints
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/scrape", handleScrape).Methods("GET")
	api.HandleFunc("/health", handleHealth).Methods("GET")
	api.HandleFunc("/sites", handleListSites).Methods("GET")

	// Middleware
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)

	fmt.Println("Web Scraper API Server starting on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET /api/v1/scrape?site=kontakt&uri=<product-url>")
	fmt.Println("  GET /api/v1/health")
	fmt.Println("  GET /api/v1/sites")

	log.Fatal(http.ListenAndServe(":8080", r))
}

func handleScrape(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Query().Get("site")
	uri := r.URL.Query().Get("uri")

	if site == "" || uri == "" {
		http.Error(w, `{"error":"missing_parameters","message":"Both 'site' and 'uri' parameters are required"}`, http.StatusBadRequest)
		return
	}

	scraper, err := scrappers.GetScraper(site)
	if err != nil {
		errorResp := ErrorResponse{
			Error:   "unsupported_site",
			Message: fmt.Sprintf("Site '%s' is not supported. Use /api/v1/sites to see available sites", site),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	product, err := scraper.Scrape(uri)
	if err != nil {
		errorResp := ErrorResponse{
			Error:   "scraping_failed",
			Message: fmt.Sprintf("Failed to scrape URL: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"service":   "web-scraper-api",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func handleListSites(w http.ResponseWriter, r *http.Request) {
	sites := scrappers.GetAvailableSites()
	response := map[string]interface{}{
		"supported_sites": sites,
		"count":           len(sites),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s - %v", r.Method, r.RequestURI, r.RemoteAddr, time.Since(start))
	})
}
