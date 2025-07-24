// 调试脚本：跟踪Wave Terminal工作区数据链路
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type WorkspaceData struct {
	OID  string                 `json:"oid"`
	Data map[string]interface{} `json:"data"`
}

func main() {
	fmt.Println("🔍 Wave Terminal工作区数据链路调试工具")
	fmt.Println(strings.Repeat("=", 50))

	// 1. 检查数据库文件
	checkDatabases()

	// 2. 检查API响应
	checkAPI()

	// 3. 对比结果
	fmt.Println("\n📊 调试总结:")
	fmt.Println("请查看上述输出，找出数据在哪一层被过滤或丢失")
}

func checkDatabases() {
	fmt.Println("\n1️⃣ 检查数据库文件:")
	
	dbPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/waveterm/db/waveterm.db"),
		filepath.Join(os.Getenv("HOME"), "Library/Application Support/waveterm/db/waveterm.db"),
	}

	for i, dbPath := range dbPaths {
		fmt.Printf("\n数据库 %d: %s\n", i+1, dbPath)
		
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("❌ 文件不存在\n")
			continue
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Printf("❌ 无法打开数据库: %v\n", err)
			continue
		}
		defer db.Close()

		// 查询工作区数量
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM db_workspace").Scan(&count)
		if err != nil {
			fmt.Printf("❌ 查询失败: %v\n", err)
			continue
		}
		
		fmt.Printf("📊 工作区总数: %d\n", count)

		// 查询详细信息
		rows, err := db.Query("SELECT oid, data FROM db_workspace ORDER BY oid")
		if err != nil {
			fmt.Printf("❌ 查询详细信息失败: %v\n", err)
			continue
		}
		defer rows.Close()

		fmt.Printf("📋 工作区列表:\n")
		for rows.Next() {
			var oid, dataJSON string
			err := rows.Scan(&oid, &dataJSON)
			if err != nil {
				fmt.Printf("❌ 读取行失败: %v\n", err)
				continue
			}

			var data map[string]interface{}
			err = json.Unmarshal([]byte(dataJSON), &data)
			if err != nil {
				fmt.Printf("❌ 解析JSON失败: %v\n", err)
				continue
			}

			name, _ := data["name"].(string)
			if name == "" {
				name = "<无名称>"
			}

			fmt.Printf("  • %s (ID: %s)\n", name, oid)
			fmt.Printf("    Data: %s\n", dataJSON)
		}
	}
}

func checkAPI() {
	fmt.Println("\n2️⃣ 检查API响应:")
	
	resp, err := http.Get("http://localhost:61269/api/v1/widgets/workspaces")
	if err != nil {
		fmt.Printf("❌ API请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ API返回状态码: %d\n", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Printf("❌ 解析API响应失败: %v\n", err)
		return
	}

	fmt.Printf("📡 API响应: %s\n", jsonPretty(result))
	
	if workspaces, ok := result["workspaces"].([]interface{}); ok {
		fmt.Printf("📊 API返回的工作区数量: %d\n", len(workspaces))
		
		fmt.Printf("📋 API工作区列表:\n")
		for i, ws := range workspaces {
			if wsMap, ok := ws.(map[string]interface{}); ok {
				name, _ := wsMap["name"].(string)
				id, _ := wsMap["workspace_id"].(string)
				fmt.Printf("  %d. %s (ID: %s)\n", i+1, name, id)
			}
		}
	}
}

func jsonPretty(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("JSON序列化失败: %v", err)
	}
	return string(b)
}