// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"

	"github.com/wavetermdev/waveterm/pkg/wavebase"
)

// CORSHandler is a specialized CORS handler for the Wave Terminal API
type CORSHandler struct {
	handler http.Handler
}

// NewCORSHandler creates a new CORS handler wrapper
func NewCORSHandler(handler http.Handler) *CORSHandler {
	return &CORSHandler{handler: handler}
}

// ServeHTTP handles CORS for all requests
func (c *CORSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	
	// Set CORS headers based on origin and development mode
	c.setCORSHeaders(w, r, origin)
	
	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Continue with the next handler
	c.handler.ServeHTTP(w, r)
}

// setCORSHeaders sets appropriate CORS headers based on the request
func (c *CORSHandler) setCORSHeaders(w http.ResponseWriter, r *http.Request, origin string) {
	// Always allow credentials for API requests
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Set allowed methods
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
	
	// Set allowed headers
	allowedHeaders := []string{
		"Accept",
		"Authorization",
		"Content-Type",
		"X-CSRF-Token",
		"X-AuthKey",
		"X-Requested-With",
		"Cache-Control",
	}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
	
	// Set exposed headers
	exposedHeaders := []string{
		"Content-Type",
		"Content-Length",
		"X-ZoneFileInfo",
		"Cache-Control",
		"Last-Modified",
	}
	w.Header().Set("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
	
	// Set Max-Age for preflight caching
	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
	
	// Determine and set the appropriate origin
	c.setOriginHeader(w, origin)
}

// setOriginHeader sets the Access-Control-Allow-Origin header based on the request origin
func (c *CORSHandler) setOriginHeader(w http.ResponseWriter, origin string) {
	// Always allow these development origins
	developmentOrigins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
	}
	
	// In development mode, be more permissive
	if wavebase.IsDevMode() {
		// Check if origin is in our allowed development origins
		for _, allowedOrigin := range developmentOrigins {
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				return
			}
		}
		
		// For development, also allow localhost and 127.0.0.1 on any port
		if strings.HasPrefix(origin, "http://localhost:") || 
		   strings.HasPrefix(origin, "http://127.0.0.1:") ||
		   strings.HasPrefix(origin, "https://localhost:") ||
		   strings.HasPrefix(origin, "https://127.0.0.1:") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			return
		}
		
		// For development mode, default to wildcard if no specific origin
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			// In dev mode, trust the origin if it's from localhost/127.0.0.1
			if isLocalOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}
	} else {
		// Production mode: more restrictive
		// Only allow specific known origins or wildcard for non-credentialed requests
		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			// In production, you might want to maintain a whitelist of allowed origins
			// For now, we'll be permissive but this should be configured for production
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}
}

// isLocalOrigin checks if the origin is from localhost or 127.0.0.1
func isLocalOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost") ||
		   strings.HasPrefix(origin, "https://localhost") ||
		   strings.HasPrefix(origin, "http://127.0.0.1") ||
		   strings.HasPrefix(origin, "https://127.0.0.1")
}

// corsPreflightHandler handles OPTIONS requests for CORS preflight
func corsPreflightHandler(w http.ResponseWriter, r *http.Request) {
	corsHandler := NewCORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This won't be called for OPTIONS requests
	}))
	corsHandler.ServeHTTP(w, r)
}