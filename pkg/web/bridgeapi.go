// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Bridge API for receiving events from remote servers
// Handles cross-server event synchronization

package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wavetermdev/waveterm/pkg/wps"
)

// BridgeAPIHandler handles incoming bridge events from remote servers
func BridgeAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var bridgeEvent wps.BridgeEvent
	if err := json.NewDecoder(r.Body).Decode(&bridgeEvent); err != nil {
		log.Printf("BridgeAPI: Failed to decode event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the event
	if bridgeEvent.Event.Event == "" {
		log.Printf("BridgeAPI: Received event with empty event type")
		http.Error(w, "Event type is required", http.StatusBadRequest)
		return
	}

	// Check if event is too old (prevent replay attacks and stale events)
	eventAge := time.Now().Unix() - bridgeEvent.Timestamp
	if eventAge > 300 { // 5 minutes
		log.Printf("BridgeAPI: Ignoring stale event (age: %d seconds)", eventAge)
		http.Error(w, "Event too old", http.StatusBadRequest)
		return
	}

	// Log the received event for debugging
	log.Printf("BridgeAPI: Received event %s from source %s (age: %ds)", 
		bridgeEvent.Event.Event, bridgeEvent.SourceID, eventAge)

	// Publish the event to local broker
	// Note: We don't use PublishWithBridge here to avoid infinite loops
	wps.Broker.Publish(bridgeEvent.Event)

	// Return success response
	response := map[string]interface{}{
		"success":   true,
		"processed": time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// BridgeStatusHandler returns the status of the event bridge
func BridgeStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"enabled":      wps.Bridge.IsEnabled(),
		"remote_urls":  wps.Bridge.GetRemoteURLs(),
		"timestamp":    time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// BridgeConfigHandler handles bridge configuration (enable/disable, add/remove remotes)
func BridgeConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleBridgeConfig(w, r)
	case http.MethodGet:
		BridgeStatusHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

type BridgeConfigRequest struct {
	Action string `json:"action"` // "enable", "disable", "add_remote", "remove_remote"
	URL    string `json:"url,omitempty"`    // For add_remote/remove_remote actions
}

func handleBridgeConfig(w http.ResponseWriter, r *http.Request) {
	var req BridgeConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "enable":
		wps.Bridge.SetEnabled(true)
	case "disable":
		wps.Bridge.SetEnabled(false)
	case "add_remote":
		if req.URL == "" {
			http.Error(w, "URL is required for add_remote action", http.StatusBadRequest)
			return
		}
		wps.Bridge.AddRemoteServer(req.URL)
	case "remove_remote":
		if req.URL == "" {
			http.Error(w, "URL is required for remove_remote action", http.StatusBadRequest)
			return
		}
		wps.Bridge.RemoveRemoteServer(req.URL)
	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", req.Action), http.StatusBadRequest)
		return
	}

	// Return updated status
	BridgeStatusHandler(w, r)
}