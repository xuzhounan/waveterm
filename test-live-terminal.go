package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	baseURL := "http://127.0.0.1:56521"
	blockId := "0d1ce86b-5989-48e8-a995-2d073f5e0949"
	
	fmt.Println("🎯 测试实际运行的Wave Terminal输入功能")
	
	// 测试1: 检查block状态
	fmt.Println("\n📊 检查block状态:")
	statusURL := fmt.Sprintf("%s/api/v1/widgets/block/status/%s", baseURL, blockId)
	resp, err := http.Get(statusURL)
	if err != nil {
		log.Fatalf("状态检查失败: %v", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("状态响应: %s\n", body)
	
	// 等待terminal完全初始化
	fmt.Println("\n⏳ 等待5秒让terminal完全初始化...")
	time.Sleep(5 * time.Second)
	
	// 测试2: 发送输入
	fmt.Println("\n⌨️ 发送输入到terminal:")
	inputURL := fmt.Sprintf("%s/api/v1/widgets/block/%s/input", baseURL, blockId)
	
	inputData := map[string]interface{}{
		"input_data": "echo 'Hello from Live Wave Terminal!'\n",
		"input_type": "text",
	}
	
	jsonData, _ := json.Marshal(inputData)
	fmt.Printf("发送数据: %s\n", jsonData)
	
	resp2, err := http.Post(inputURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("输入发送失败: %v", err)
		return
	}
	defer resp2.Body.Close()
	
	body2, _ := io.ReadAll(resp2.Body)
	fmt.Printf("状态码: %d\n", resp2.StatusCode)
	fmt.Printf("输入响应: %s\n", body2)
	
	if resp2.StatusCode == 404 {
		fmt.Println("❌ 输入API不存在 - 可能运行的是旧版本的Wave Terminal")
		fmt.Println("💡 需要确保运行的是包含新输入功能的开发版本")
		return
	}
	
	// 等待命令执行
	fmt.Println("\n⏳ 等待3秒让命令执行...")
	time.Sleep(3 * time.Second)
	
	// 测试3: 读取输出
	fmt.Println("\n📄 读取terminal输出:")
	contentURL := fmt.Sprintf("%s/api/v1/widgets/block/content/%s", baseURL, blockId)
	resp3, err := http.Get(contentURL)
	if err != nil {
		log.Printf("读取输出失败: %v", err)
		return
	}
	defer resp3.Body.Close()
	
	body3, _ := io.ReadAll(resp3.Body)
	fmt.Printf("输出响应: %s\n", body3)
}