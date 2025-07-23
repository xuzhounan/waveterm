// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Simple syntax test for Widget API implementation
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
)

func main() {
	ctx := context.Background()
	
	// Test creating the service instance
	service := widgetapiservice.WidgetAPIServiceInstance
	if service == nil {
		log.Fatal("Failed to get WidgetAPIService instance")
	}
	
	// Test struct initialization
	req := widgetapiservice.CreateWidgetAPIRequest{
		WorkspaceId: "test-workspace",
		WidgetType:  "terminal",
		Title:       "Test Terminal",
		Meta: map[string]any{
			"cwd": "/home/test",
		},
	}
	
	fmt.Printf("API request initialized successfully: %+v\n", req)
	
	// Test ListWorkspaces method signature
	_, err := service.ListWorkspaces(ctx)
	if err != nil {
		fmt.Printf("ListWorkspaces test (expected error): %v\n", err)
	}
	
	fmt.Println("Widget API syntax test completed successfully!")
}