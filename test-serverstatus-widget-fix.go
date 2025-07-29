package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type CreateWidgetRequest struct {
	WorkspaceId string `json:"workspace_id"`
	WidgetType  string `json:"widget_type"`
	Title       string `json:"title,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

func main() {
	// 测试创建serverstatus widget
	request := CreateWidgetRequest{
		WorkspaceId: "39720a34-6d5b-477c-bc5f-4ac6f8eb1abf", // waveterm workspace
		WidgetType:  "serverstatus",
		Title:       "Server Status Monitor",
		Icon:        "server",
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	// 发送创建widget请求
	resp, err := http.Post("http://localhost:60289/api/v1/widgets", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	// 如果创建成功，测试服务器状态功能
	if resp.StatusCode == 200 {
		fmt.Println("\n=== Testing server status endpoints ===")
		time.Sleep(2 * time.Second)

		// 测试MCP服务器状态
		testEndpoint("http://localhost:60289/api/v1/widgets/mcp/status", "MCP Server Status")
		
		// 测试持久化服务器状态
		testEndpoint("http://localhost:60289/api/v1/widgets/persistent-server/status", "Persistent Server Status")
	}
}

func testEndpoint(url, name string) {
	fmt.Printf("\n--- Testing %s ---\n", name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
}