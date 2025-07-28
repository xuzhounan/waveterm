// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Event Bridge for cross-server event synchronization
// Allows MCP API servers to sync events with main Wave Terminal instances

package wps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// EventBridge manages cross-server event synchronization
type EventBridge struct {
	Lock       *sync.RWMutex
	RemoteURLs []string              // URLs of remote servers to notify
	Client     *http.Client          // HTTP client for sending events
	Enable     bool                  // Whether bridge is enabled
	Timeout    time.Duration         // Request timeout
}

// BridgeEvent represents an event to be sent across servers
type BridgeEvent struct {
	Event      WaveEvent `json:"event"`
	SourceID   string    `json:"source_id"`   // ID of the source server
	Timestamp  int64     `json:"timestamp"`   // Unix timestamp
}

// Global event bridge instance
var Bridge = &EventBridge{
	Lock:       &sync.RWMutex{},
	RemoteURLs: []string{},
	Client: &http.Client{
		Timeout: 5 * time.Second,
	},
	Enable:  false,
	Timeout: 5 * time.Second,
}

// AddRemoteServer adds a remote server URL to sync events with
func (eb *EventBridge) AddRemoteServer(url string) {
	eb.Lock.Lock()
	defer eb.Lock.Unlock()
	
	// Check if URL already exists
	for _, existing := range eb.RemoteURLs {
		if existing == url {
			return
		}
	}
	
	eb.RemoteURLs = append(eb.RemoteURLs, url)
	log.Printf("EventBridge: Added remote server %s", url)
}

// RemoveRemoteServer removes a remote server URL
func (eb *EventBridge) RemoveRemoteServer(url string) {
	eb.Lock.Lock()
	defer eb.Lock.Unlock()
	
	for i, existing := range eb.RemoteURLs {
		if existing == url {
			eb.RemoteURLs = append(eb.RemoteURLs[:i], eb.RemoteURLs[i+1:]...)
			log.Printf("EventBridge: Removed remote server %s", url)
			return
		}
	}
}

// SetEnabled enables or disables the event bridge
func (eb *EventBridge) SetEnabled(enabled bool) {
	eb.Lock.Lock()
	defer eb.Lock.Unlock()
	eb.Enable = enabled
	log.Printf("EventBridge: %s", map[bool]string{true: "Enabled", false: "Disabled"}[enabled])
}

// IsEnabled returns whether the bridge is enabled
func (eb *EventBridge) IsEnabled() bool {
	eb.Lock.RLock()
	defer eb.Lock.RUnlock()
	return eb.Enable
}

// GetRemoteURLs returns a copy of remote server URLs
func (eb *EventBridge) GetRemoteURLs() []string {
	eb.Lock.RLock()
	defer eb.Lock.RUnlock()
	urls := make([]string, len(eb.RemoteURLs))
	copy(urls, eb.RemoteURLs)
	return urls
}

// ForwardEvent forwards an event to all registered remote servers
func (eb *EventBridge) ForwardEvent(event WaveEvent, sourceID string) {
	if !eb.IsEnabled() {
		return
	}
	
	urls := eb.GetRemoteURLs()
	if len(urls) == 0 {
		return
	}
	
	bridgeEvent := BridgeEvent{
		Event:     event,
		SourceID:  sourceID,
		Timestamp: time.Now().Unix(),
	}
	
	// Forward to all remote servers asynchronously
	for _, url := range urls {
		go eb.sendEventToRemote(url, bridgeEvent)
	}
}

// sendEventToRemote sends an event to a specific remote server
func (eb *EventBridge) sendEventToRemote(url string, bridgeEvent BridgeEvent) {
	jsonData, err := json.Marshal(bridgeEvent)
	if err != nil {
		log.Printf("EventBridge: Failed to marshal event for %s: %v", url, err)
		return
	}
	
	endpoint := fmt.Sprintf("%s/api/v1/bridge/event", url)
	
	ctx, cancel := context.WithTimeout(context.Background(), eb.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("EventBridge: Failed to create request for %s: %v", url, err)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := eb.Client.Do(req)
	if err != nil {
		log.Printf("EventBridge: Failed to send event to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("EventBridge: Remote server %s returned status %d", url, resp.StatusCode)
		return
	}
	
	// Optionally log successful forwards (disabled by default to reduce noise)
	// log.Printf("EventBridge: Successfully forwarded event %s to %s", bridgeEvent.Event.Event, url)
}

// Enhanced Broker Publish method with event bridging
func (b *BrokerType) PublishWithBridge(event WaveEvent, sourceID string) {
	// First publish locally
	b.Publish(event)
	
	// Then forward to remote servers if bridge is enabled
	Bridge.ForwardEvent(event, sourceID)
}

// Initialize bridge with environment variables or config
func InitEventBridge() {
	// This can be called during server startup to configure the bridge
	// Environment variables or config file can specify remote servers
	log.Printf("EventBridge: Initialized")
}