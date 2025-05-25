package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// SteamAPIResponse represents the response from Steam API
type SteamAPIResponse struct {
	Result struct {
		NextCode string `json:"nextcode"`
	} `json:"result"`
}

// DemoServiceRequest represents the request to demo service
type DemoServiceRequest struct {
	WebhookURL string `json:"webhook_url"`
}

// DemoServiceResponse represents the response from demo service
type DemoServiceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SteamPoller manages Steam API polling for all users
type SteamPoller struct {
	apiKey        string
	webhookURL    string
	parseURL      string
	stopChan      chan bool
	isRunning     bool
	mutex         sync.RWMutex
	processedCodes map[string]bool // Track processed share codes to avoid duplicates
}

// NewSteamPoller creates a new Steam API poller
func NewSteamPoller() *SteamPoller {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		log.Fatal("STEAM_API_KEY environment variable is required")
	}

	webhookURL := os.Getenv("WEBHOOK_BASE_URL")
	if webhookURL == "" {
		webhookURL = "https://cs-bot.simonfalke.com"
	}

	parseURL := os.Getenv("DEMO_PARSE_BASE_URL")
	if parseURL == "" {
		parseURL = "https://cs-demo-parsing.simonfalke.com"
	}

	return &SteamPoller{
		apiKey:         apiKey,
		webhookURL:     webhookURL,
		parseURL:       parseURL,
		stopChan:       make(chan bool),
		processedCodes: make(map[string]bool),
	}
}

// Start begins the polling process
func (sp *SteamPoller) Start() {
	sp.mutex.Lock()
	if sp.isRunning {
		sp.mutex.Unlock()
		return
	}
	sp.isRunning = true
	sp.mutex.Unlock()

	log.Println("Starting Steam API poller...")
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sp.stopChan:
			log.Println("Steam API poller stopped")
			return
		case <-ticker.C:
			sp.pollAllUsers()
		}
	}
}

// Stop stops the polling process
func (sp *SteamPoller) Stop() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	if !sp.isRunning {
		return
	}
	
	sp.isRunning = false
	close(sp.stopChan)
}

// pollAllUsers polls Steam API for all registered users
func (sp *SteamPoller) pollAllUsers() {
	users, err := getAllUsers()
	if err != nil {
		log.Printf("Error getting users for polling: %v", err)
		return
	}

	if len(users) == 0 {
		return
	}

	log.Printf("Polling Steam API for %d users...", len(users))

	// Track which share codes we've seen in this polling cycle
	currentCodes := make(map[string][]*User)

	for _, user := range users {
		if user.LastShareCode == "" {
			log.Printf("User %s has no last share code, skipping", user.SteamID)
			continue
		}

		nextCode, err := sp.pollUserAPI(user)
		if err != nil {
			log.Printf("Error polling for user %s: %v", user.SteamID, err)
			continue
		}

		if nextCode != "" && nextCode != "n/a" && nextCode != user.LastShareCode {
			log.Printf("New match found for user %s: %s", user.SteamID, nextCode)
			
			// Group users by share code to avoid duplicate downloads
			currentCodes[nextCode] = append(currentCodes[nextCode], user)
		}
	}

	// Process each unique share code
	for shareCode, usersWithCode := range currentCodes {
		sp.processNewMatch(shareCode, usersWithCode)
	}
}

// pollUserAPI polls Steam API for a specific user
func (sp *SteamPoller) pollUserAPI(user *User) (string, error) {
	url := fmt.Sprintf(
		"https://api.steampowered.com/ICSGOPlayers_730/GetNextMatchSharingCode/v1?key=%s&steamid=%s&steamidkey=%s&knowncode=%s",
		sp.apiKey,
		user.SteamID,
		user.AuthCode,
		user.LastShareCode,
	)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Steam API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp SteamAPIResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return apiResp.Result.NextCode, nil
}

// processNewMatch processes a new match found for users
func (sp *SteamPoller) processNewMatch(shareCode string, users []*User) {
	// Check if we've already processed this share code
	sp.mutex.RLock()
	if sp.processedCodes[shareCode] {
		sp.mutex.RUnlock()
		log.Printf("Share code %s already processed, skipping", shareCode)
		return
	}
	sp.mutex.RUnlock()

	// Mark as processed
	sp.mutex.Lock()
	sp.processedCodes[shareCode] = true
	sp.mutex.Unlock()

	log.Printf("Processing new match %s for %d users", shareCode, len(users))

	// Update last share code for all users
	for _, user := range users {
		err := updateUserLastShareCode(user.SteamID, shareCode)
		if err != nil {
			log.Printf("Error updating last share code for user %s: %v", user.SteamID, err)
		}
	}

	// Request demo download (only once per share code)
	err := sp.requestDemoDownload(shareCode)
	if err != nil {
		log.Printf("Error requesting demo download for %s: %v", shareCode, err)
		// Remove from processed codes so we can retry later
		sp.mutex.Lock()
		delete(sp.processedCodes, shareCode)
		sp.mutex.Unlock()
		return
	}

	log.Printf("Successfully requested demo download for %s", shareCode)
}

// requestDemoDownload requests demo download from the demo service
func (sp *SteamPoller) requestDemoDownload(shareCode string) error {
	url := fmt.Sprintf("%s/getDemo/%s", sp.parseURL, shareCode)
	
	requestBody := DemoServiceRequest{
		WebhookURL: fmt.Sprintf("%s/webhooks/demoReady", sp.webhookURL),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("demo service returned status %d: %s", resp.StatusCode, string(body))
	}

	var serviceResp DemoServiceResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, &serviceResp)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !serviceResp.Success {
		return fmt.Errorf("demo service returned error: %s", serviceResp.Message)
	}

	return nil
}

// requestDemoParsing requests demo parsing from the demo service
func (sp *SteamPoller) requestDemoParsing(shareCode string) error {
	url := fmt.Sprintf("%s/parseDemo/%s", sp.parseURL, shareCode)
	
	requestBody := DemoServiceRequest{
		WebhookURL: fmt.Sprintf("%s/webhooks/demoParsed", sp.webhookURL),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("demo service returned status %d: %s", resp.StatusCode, string(body))
	}

	var serviceResp DemoServiceResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, &serviceResp)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !serviceResp.Success {
		return fmt.Errorf("demo service returned error: %s", serviceResp.Message)
	}

	return nil
}

// GetDemoParsingRequest returns the demo parsing request function for webhook use
func (sp *SteamPoller) GetDemoParsingRequest() func(string) error {
	return sp.requestDemoParsing
}

// IsRunning returns whether the poller is currently running
func (sp *SteamPoller) IsRunning() bool {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	return sp.isRunning
}

// CleanupProcessedCodes removes old processed codes to prevent memory leaks
func (sp *SteamPoller) CleanupProcessedCodes() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	// Keep only recent codes (last 1000)
	if len(sp.processedCodes) > 1000 {
		// Simple cleanup: clear all and start fresh
		sp.processedCodes = make(map[string]bool)
		log.Println("Cleaned up processed codes cache")
	}
}