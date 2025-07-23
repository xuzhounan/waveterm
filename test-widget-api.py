#!/usr/bin/env python3
"""
Wave Terminal Widget API 测试脚本
测试新开发的Widget API功能
"""

import json
import requests
import time
import sys
from typing import Dict, Any, Optional

class WidgetAPITester:
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.api_base = f"{base_url}/api/v1/widgets"
        self.session = requests.Session()
        
    def test_connection(self) -> bool:
        """测试与服务器的连接"""
        try:
            response = self.session.get(f"{self.base_url}/wave/service", timeout=5)
            print(f"✓ 服务器连接正常 (状态码: {response.status_code})")
            return True
        except requests.exceptions.RequestException as e:
            print(f"✗ 服务器连接失败: {e}")
            return False
    
    def test_list_widget_types(self) -> Dict[str, Any]:
        """测试获取widget类型列表"""
        print("\n🧪 测试: 获取widget类型列表")
        try:
            response = self.session.get(self.api_base)
            print(f"   状态码: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"   ✓ 成功获取widget类型")
                print(f"   支持的类型: {list(data.get('widget_types', {}).keys())}")
                return data
            else:
                print(f"   ✗ 请求失败: {response.text}")
                return {}
        except Exception as e:
            print(f"   ✗ 异常: {e}")
            return {}
    
    def test_list_workspaces(self) -> Dict[str, Any]:
        """测试获取工作空间列表"""
        print("\n🧪 测试: 获取工作空间列表")
        try:
            response = self.session.get(f"{self.api_base}/workspaces")
            print(f"   状态码: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"   ✓ 成功获取工作空间列表")
                workspaces = data.get('workspaces', [])
                print(f"   工作空间数量: {len(workspaces)}")
                for ws in workspaces[:3]:  # 只显示前3个
                    print(f"   - {ws.get('name', 'Unnamed')} ({ws.get('workspace_id', 'No ID')})")
                return data
            else:
                print(f"   ✗ 请求失败: {response.text}")
                return {}
        except Exception as e:
            print(f"   ✗ 异常: {e}")
            return {}
    
    def test_get_workspace_widgets(self, workspace_id: str) -> Dict[str, Any]:
        """测试获取工作空间widget配置"""
        print(f"\n🧪 测试: 获取工作空间widget配置 (workspace_id: {workspace_id})")
        try:
            response = self.session.get(f"{self.api_base}/workspace/{workspace_id}")
            print(f"   状态码: {response.status_code}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"   ✓ 成功获取工作空间widget配置")
                widgets = data.get('widgets', {})
                print(f"   可用widget数量: {len(widgets)}")
                for widget_key in list(widgets.keys())[:3]:  # 只显示前3个
                    widget = widgets[widget_key]
                    print(f"   - {widget.get('label', 'Unnamed')} ({widget_key})")
                return data
            else:
                print(f"   ✗ 请求失败: {response.text}")
                return {}
        except Exception as e:
            print(f"   ✗ 异常: {e}")
            return {}
    
    def test_create_widget(self, workspace_id: str, widget_type: str = "terminal") -> Dict[str, Any]:
        """测试创建widget"""
        print(f"\n🧪 测试: 创建{widget_type} widget")
        
        # 构建请求数据
        widget_data = {
            "workspace_id": workspace_id,
            "widget_type": widget_type,
            "title": f"Test {widget_type.title()} Widget",
        }
        
        # 根据不同类型添加特定配置
        if widget_type == "terminal":
            widget_data["meta"] = {"cwd": "/tmp"}
        elif widget_type == "web":
            widget_data["meta"] = {"url": "https://www.waveterm.dev"}
        elif widget_type == "files":
            widget_data["meta"] = {"file": "~"}
        
        try:
            response = self.session.post(
                self.api_base,
                json=widget_data,
                headers={"Content-Type": "application/json"}
            )
            print(f"   请求数据: {json.dumps(widget_data, indent=2)}")
            print(f"   状态码: {response.status_code}")
            
            if response.status_code == 201:
                data = response.json()
                print(f"   ✓ Widget创建成功")
                print(f"   Block ID: {data.get('block_id', 'N/A')}")
                print(f"   消息: {data.get('message', 'N/A')}")
                return data
            else:
                print(f"   ✗ Widget创建失败: {response.text}")
                return {}
                
        except Exception as e:
            print(f"   ✗ 异常: {e}")
            return {}
    
    def test_error_handling(self) -> None:
        """测试错误处理"""
        print(f"\n🧪 测试: 错误处理")
        
        # 测试1: 无效的workspace_id
        print("   测试1: 无效的workspace_id")
        try:
            response = self.session.post(
                self.api_base,
                json={"workspace_id": "invalid-workspace", "widget_type": "terminal"},
                headers={"Content-Type": "application/json"}
            )
            print(f"   状态码: {response.status_code}")
            if response.status_code == 400 or response.status_code == 404:
                print("   ✓ 正确处理无效workspace_id")
            else:
                print(f"   ✗ 期望400/404状态码，实际: {response.status_code}")
        except Exception as e:
            print(f"   ✗ 异常: {e}")
        
        # 测试2: 缺少必需参数
        print("   测试2: 缺少必需参数")
        try:
            response = self.session.post(
                self.api_base,
                json={"workspace_id": "test"},  # 缺少widget_type
                headers={"Content-Type": "application/json"}
            )
            print(f"   状态码: {response.status_code}")
            if response.status_code == 400:
                print("   ✓ 正确处理缺少参数")
            else:
                print(f"   ✗ 期望400状态码，实际: {response.status_code}")
        except Exception as e:
            print(f"   ✗ 异常: {e}")
        
        # 测试3: 无效的widget类型
        print("   测试3: 无效的widget类型")
        try:
            response = self.session.post(
                self.api_base,
                json={"workspace_id": "test-workspace", "widget_type": "invalid-type"},
                headers={"Content-Type": "application/json"}
            )
            print(f"   状态码: {response.status_code}")
            if response.status_code == 400:
                print("   ✓ 正确处理无效widget类型")
            else:
                print(f"   ✗ 期望400状态码，实际: {response.status_code}")
            
            # 检查错误响应格式
            try:
                error_data = response.json()
                if 'success' in error_data and error_data['success'] == False:
                    print("   ✓ 错误响应格式正确")
                else:
                    print("   ✗ 错误响应格式不正确")
            except:
                print("   ✗ 无法解析错误响应JSON")
                
        except Exception as e:
            print(f"   ✗ 异常: {e}")
    
    def run_comprehensive_test(self) -> None:
        """运行完整的API测试"""
        print("🚀 开始Wave Terminal Widget API 完整测试")
        print("=" * 60)
        
        # 测试连接
        if not self.test_connection():
            print("\n❌ 服务器连接失败，跳过其他测试")
            return
        
        # 测试API端点
        widget_types = self.test_list_widget_types()
        workspaces = self.test_list_workspaces()
        
        # 如果有工作空间，测试详细功能
        if workspaces.get('workspaces'):
            first_workspace = workspaces['workspaces'][0]
            workspace_id = first_workspace.get('workspace_id')
            
            if workspace_id:
                # 测试获取工作空间配置
                self.test_get_workspace_widgets(workspace_id)
                
                # 测试创建不同类型的widget
                if widget_types.get('widget_types'):
                    for widget_type in ['terminal', 'web', 'files', 'ai']:
                        if widget_type in widget_types['widget_types']:
                            self.test_create_widget(workspace_id, widget_type)
                            time.sleep(1)  # 避免请求过于频繁
        
        # 测试错误处理
        self.test_error_handling()
        
        print("\n" + "=" * 60)
        print("✅ Widget API 测试完成")

def main():
    """主函数"""
    import argparse
    parser = argparse.ArgumentParser(description='Wave Terminal Widget API 测试脚本')
    parser.add_argument('--url', default='http://localhost:8080', 
                       help='Wave Terminal服务器URL (默认: http://localhost:8080)')
    parser.add_argument('--test', choices=['connection', 'types', 'workspaces', 'create', 'all'], 
                       default='all', help='指定要运行的测试')
    
    args = parser.parse_args()
    
    tester = WidgetAPITester(args.url)
    
    if args.test == 'connection':
        tester.test_connection()
    elif args.test == 'types':
        tester.test_list_widget_types()
    elif args.test == 'workspaces':
        tester.test_list_workspaces()
    elif args.test == 'create':
        workspaces = tester.test_list_workspaces()
        if workspaces.get('workspaces'):
            workspace_id = workspaces['workspaces'][0].get('workspace_id')
            if workspace_id:
                tester.test_create_widget(workspace_id)
    else:
        tester.run_comprehensive_test()

if __name__ == "__main__":
    main()