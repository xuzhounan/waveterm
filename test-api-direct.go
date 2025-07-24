package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// 直接测试API端点的HTTP调用
func main() {
	log.Println("Testing Widget API HTTP endpoints...")
	
	baseURL := "http://localhost:8090"
	
	// 测试函数
	testEndpoint := func(method, path string) {
		fmt.Printf("\n=== Testing %s %s ===\n", method, path)
		
		fullURL := baseURL + path
		req, err := http.NewRequest(method, fullURL, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making request: %v", err)
			return
		}
		defer resp.Body.Close()
		
		fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
		
		var result interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			log.Printf("Error decoding response: %v", err)
			return
		}
		
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
	}
	
	// Test 1: List widget types
	testEndpoint("GET", "/api/v1/widgets")
	
	// Test 2: List workspaces
	testEndpoint("GET", "/api/v1/widgets/workspaces")
	
	// Test 3: Get workspace by name - "waveterm"
	encodedName := url.QueryEscape("waveterm")
	testEndpoint("GET", "/api/v1/widgets/workspace/name/"+encodedName)
	
	// Test 4: Get workspace by name - "WAVETERM" (case insensitive)
	encodedNameUpper := url.QueryEscape("WAVETERM")
	testEndpoint("GET", "/api/v1/widgets/workspace/name/"+encodedNameUpper)
	
	// Test 5: Get workspace by name - "Default"
	encodedDefault := url.QueryEscape("Default")
	testEndpoint("GET", "/api/v1/widgets/workspace/name/"+encodedDefault)
	
	// Test 6: Get workspace by name - non-existent
	testEndpoint("GET", "/api/v1/widgets/workspace/name/NonExistent")
	
	// Test 7: Get workspace by name - with spaces
	encodedSpaces := url.QueryEscape("My Workspace")
	testEndpoint("GET", "/api/v1/widgets/workspace/name/"+encodedSpaces)
	
	fmt.Println("\n=== All HTTP tests completed ===")
	fmt.Println("Note: If you see connection errors, make sure the Wave Terminal server is running:")
	fmt.Println("  go run cmd/server/main-server.go")
}