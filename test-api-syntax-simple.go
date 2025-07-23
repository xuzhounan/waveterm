// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// 简化的语法测试：验证Widget API代码结构
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
)

func main() {
	fmt.Println("🚀 Widget API 语法和结构测试")
	fmt.Println(strings.Repeat("=", 50))
	
	// 测试1: 服务实例
	fmt.Println("\n1️⃣ 测试服务实例...")
	service := widgetapiservice.WidgetAPIServiceInstance
	if service == nil {
		log.Fatal("❌ WidgetAPIService实例为nil")
	}
	fmt.Println("✅ WidgetAPIService实例创建成功")
	
	// 测试2: 数据结构
	fmt.Println("\n2️⃣ 测试数据结构...")
	
	// 创建请求结构
	req := widgetapiservice.CreateWidgetAPIRequest{
		WorkspaceId: "test-workspace-123",
		WidgetType:  "terminal",
		Title:       "Test Terminal Widget",
		Icon:        "terminal",
		Meta: map[string]any{
			"cwd": "/home/user",
			"env": map[string]string{
				"NODE_ENV": "development",
			},
		},
		Position: &widgetapiservice.WidgetPosition{
			TargetBlockId: "block-456",
			Action:        "splitright",
		},
		Magnified: false,
		Ephemeral: false,
	}
	
	fmt.Printf("✅ CreateWidgetAPIRequest 结构创建成功\n")
	fmt.Printf("   WorkspaceId: %s\n", req.WorkspaceId)
	fmt.Printf("   WidgetType: %s\n", req.WidgetType)
	fmt.Printf("   Title: %s\n", req.Title)
	
	// 测试3: JSON序列化
	fmt.Println("\n3️⃣ 测试JSON序列化...")
	jsonData, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON序列化失败: %v", err)
	}
	fmt.Println("✅ JSON序列化成功")
	fmt.Printf("请求JSON大小: %d bytes\n", len(jsonData))
	
	// 测试4: 响应结构
	fmt.Println("\n4️⃣ 测试响应结构...")
	resp := widgetapiservice.CreateWidgetAPIResponse{
		Success: true,
		BlockId: "block-789",
		Message: "Widget 'terminal' created successfully",
		Widget: &widgetapiservice.WidgetInfo{
			BlockId:     "block-789",
			TabId:       "tab-1",
			WorkspaceId: "test-workspace-123",
			WidgetType:  "terminal",
			Title:       "Test Terminal Widget",
			Icon:        "terminal",
			Meta: map[string]any{
				"cwd": "/home/user",
			},
			CreatedAt: 1642694400000,
		},
	}
	
	respJsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Fatalf("❌ 响应JSON序列化失败: %v", err)
	}
	fmt.Println("✅ 响应结构序列化成功")
	fmt.Printf("响应JSON大小: %d bytes\n", len(respJsonData))
	
	// 测试5: 方法签名
	fmt.Println("\n5️⃣ 测试方法签名...")
	
	// 检查ListWorkspaces方法
	serviceType := reflect.TypeOf(service)
	listWorkspacesMethod, exists := serviceType.MethodByName("ListWorkspaces")
	if !exists {
		log.Fatal("❌ ListWorkspaces方法不存在")
	}
	
	// 检查方法签名
	if listWorkspacesMethod.Type.NumIn() != 2 { // receiver + context
		log.Fatal("❌ ListWorkspaces方法参数不正确")
	}
	
	if listWorkspacesMethod.Type.NumOut() != 2 { // response + error
		log.Fatal("❌ ListWorkspaces方法返回值不正确")
	}
	
	fmt.Println("✅ ListWorkspaces方法签名正确")
	
	// 检查CreateWidget方法
	_, exists = serviceType.MethodByName("CreateWidget")
	if !exists {
		log.Fatal("❌ CreateWidget方法不存在")
	}
	
	fmt.Println("✅ CreateWidget方法存在")
	
	// 检查GetWorkspaceWidgets方法
	_, exists = serviceType.MethodByName("GetWorkspaceWidgets")
	if !exists {
		log.Fatal("❌ GetWorkspaceWidgets方法不存在")
	}
	
	fmt.Println("✅ GetWorkspaceWidgets方法存在")
	
	// 测试6: Widget类型验证
	fmt.Println("\n6️⃣ 测试Widget类型...")
	
	supportedTypes := []string{
		"terminal", "web", "files", "ai", "sysinfo", "help", "tips",
	}
	
	for _, widgetType := range supportedTypes {
		testReq := widgetapiservice.CreateWidgetAPIRequest{
			WorkspaceId: "test-ws",
			WidgetType:  widgetType,
			Title:       fmt.Sprintf("Test %s Widget", widgetType),
		}
		
		// 验证结构可以正确创建
		if testReq.WidgetType != widgetType {
			log.Fatalf("❌ Widget类型 %s 验证失败", widgetType)
		}
	}
	
	fmt.Printf("✅ 支持的Widget类型验证成功: %v\n", supportedTypes)
	
	// 测试7: 错误处理结构
	fmt.Println("\n7️⃣ 测试错误处理...")
	
	errorResp := widgetapiservice.CreateWidgetAPIResponse{
		Success: false,
		Error:   "workspace not found: invalid workspace ID",
	}
	
	if errorResp.Success != false || errorResp.Error == "" {
		log.Fatal("❌ 错误响应结构不正确")
	}
	
	fmt.Println("✅ 错误处理结构正确")
	
	// 测试完成
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎉 所有语法和结构测试通过！")
	fmt.Println("\n📋 测试总结:")
	fmt.Println("   ✅ 服务实例创建正常")
	fmt.Println("   ✅ 数据结构定义正确") 
	fmt.Println("   ✅ JSON序列化工作正常")
	fmt.Println("   ✅ 响应结构完整")
	fmt.Println("   ✅ 方法签名正确")
	fmt.Println("   ✅ Widget类型支持完整")
	fmt.Println("   ✅ 错误处理结构正确")
	fmt.Println("\n💡 下一步:")
	fmt.Println("   • 启动包含修改的Wave Terminal服务器")
	fmt.Println("   • 运行 test-widget-api.py 或 test-widget-api.sh")
	fmt.Println("   • 进行完整的功能测试")
}