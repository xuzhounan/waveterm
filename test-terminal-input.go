package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/blockcontroller"
	"github.com/wavetermdev/waveterm/pkg/filestore"
	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wstore"
	"time"
)

func main() {
	// 设置环境变量
	dataDir := "/tmp/waveterm-mcp"
	os.Setenv("WAVETERM_DATA_HOME", dataDir)
	os.Setenv("WAVETERM_CONFIG_HOME", dataDir)
	
	log.Printf("Using Wave data directory: %s", dataDir)
	
	// 初始化数据库
	err := wavebase.CacheAndRemoveEnvVars()
	if err != nil {
		log.Fatalf("error caching env vars: %v", err)
	}
	
	err = filestore.InitFilestore()
	if err != nil {
		log.Fatalf("error initializing filestore: %v", err)
	}
	
	err = wstore.InitWStore()
	if err != nil {
		log.Fatalf("error initializing wstore: %v", err)
	}
	
	log.Println("Connected to Wave Terminal database!")
	
	ctx := context.Background()
	service := &widgetapiservice.WidgetAPIService{}
	
	// 获取测试工作区
	fmt.Println("\n=== Getting test workspace ===")
	workspaceResp, err := service.GetWorkspaceByName(ctx, "waveterm")
	if err != nil || !workspaceResp.Success {
		log.Fatalf("Error getting workspace: %v", err)
	}
	
	workspaceId := workspaceResp.Workspace.WorkspaceId
	fmt.Printf("Found workspace: %s (ID: %s)\n", workspaceResp.Workspace.Name, workspaceId)
	
	// 创建一个terminal widget
	fmt.Println("\n=== Creating terminal widget ===")
	createReq := widgetapiservice.CreateWidgetAPIRequest{
		WorkspaceId: workspaceId,
		WidgetType:  "terminal",
		Title:       "Test Terminal for Input",
		Meta: map[string]any{
			"cwd": "/tmp",
		},
	}
	
	createResp, err := service.CreateWidget(ctx, createReq)
	if err != nil || !createResp.Success {
		log.Fatalf("Error creating terminal: %v", err)
	}
	
	blockId := createResp.BlockId
	tabId := createResp.Widget.TabId
	fmt.Printf("Created terminal: %s (Block ID: %s, Tab ID: %s)\n", createReq.Title, blockId, tabId)
	
	// 手动启动block controller
	fmt.Println("\n=== Starting block controller ===")
	err = blockcontroller.ResyncController(ctx, tabId, blockId, nil, false)
	if err != nil {
		log.Printf("Error starting controller: %v", err)
	} else {
		fmt.Println("Block controller started successfully")
	}
	
	// 等待controller初始化
	fmt.Println("\n=== Waiting for controller to initialize ===")
	time.Sleep(2 * time.Second)
	
	// 检查controller状态
	bc := blockcontroller.GetBlockController(blockId)
	if bc != nil {
		status := bc.GetRuntimeStatus()
		fmt.Printf("Controller status: %s\n", status.ShellProcStatus)
	} else {
		fmt.Println("No controller found")
	}
	
	// 测试发送输入
	fmt.Println("\n=== Testing terminal input ===")
	
	// Test 1: 发送文本输入
	fmt.Println("\n1. Sending text input:")
	inputReq := widgetapiservice.SendBlockInputAPIRequest{
		BlockId:   blockId,
		InputData: "echo 'Hello from MCP Agent!'\n",
		InputType: "text",
	}
	
	inputResp, err := service.SendBlockInput(ctx, inputReq)
	if err != nil {
		log.Printf("Error sending input: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(inputResp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
	}
	
	// Test 2: 发送另一个命令
	fmt.Println("\n2. Sending another command:")
	inputReq2 := widgetapiservice.SendBlockInputAPIRequest{
		BlockId:   blockId,
		InputData: "pwd\n",
		InputType: "text",
	}
	
	inputResp2, err := service.SendBlockInput(ctx, inputReq2)
	if err != nil {
		log.Printf("Error sending input: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(inputResp2, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
	}
	
	// Test 3: 获取terminal输出内容
	fmt.Println("\n3. Getting terminal output:")
	contentResp, err := service.GetBlockContent(ctx, blockId, "", 0, 0)
	if err != nil {
		log.Printf("Error getting content: %v", err)
	} else {
		fmt.Printf("Terminal content (%d bytes):\n", contentResp.Size)
		fmt.Printf("```\n%s\n```\n", contentResp.Content)
	}
	
	// Test 4: 获取terminal状态
	fmt.Println("\n4. Getting terminal status:")
	statusResp, err := service.GetBlockStatus(ctx, blockId)
	if err != nil {
		log.Printf("Error getting status: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(statusResp, "", "  ")
		fmt.Printf("Status: %s\n", jsonData)
	}
	
	fmt.Println("\n=== Terminal input test completed ===")
}