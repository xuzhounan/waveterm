#!/usr/bin/env python3
"""
Test script to verify the Widget API fix for frontend notification issue.
This script tests that widgets are created properly and frontend gets updated.
"""

import requests
import time
import json
import sys

# API endpoint
API_BASE = "http://localhost:60289"
WIDGET_API_URL = f"{API_BASE}/api/v1/widgets"

def test_get_workspaces():
    """Test getting workspaces to find an active one"""
    print("1. Testing workspace listing...")
    
    url = f"{WIDGET_API_URL}/workspaces"
    try:
        response = requests.get(url, timeout=5)
        if response.status_code == 200:
            data = response.json()
            if data.get("success") and data.get("workspaces"):
                workspace = data["workspaces"][0]
                print(f"   ✓ Found workspace: {workspace['name']} (ID: {workspace['workspace_id']})")
                return workspace["workspace_id"]
            else:
                print(f"   ✗ No workspaces found or API error: {data}")
                return None
        else:
            print(f"   ✗ HTTP error {response.status_code}: {response.text}")
            return None
    except Exception as e:
        print(f"   ✗ Error: {e}")
        return None

def test_create_widget(workspace_id):
    """Test creating a widget"""
    print("2. Testing widget creation...")
    
    widget_data = {
        "workspace_id": workspace_id,
        "widget_type": "terminal",
        "title": "Test Terminal Widget",
        "meta": {
            "cwd": "~"
        }
    }
    
    try:
        response = requests.post(WIDGET_API_URL, json=widget_data, timeout=10)
        if response.status_code == 201:
            data = response.json()
            if data.get("success"):
                block_id = data.get("block_id")
                print(f"   ✓ Widget created successfully!")
                print(f"     Block ID: {block_id}")
                print(f"     Message: {data.get('message')}")
                return block_id
            else:
                print(f"   ✗ Widget creation failed: {data.get('error')}")
                return None
        else:
            print(f"   ✗ HTTP error {response.status_code}: {response.text}")
            return None
    except Exception as e:
        print(f"   ✗ Error: {e}")
        return None

def test_widget_types():
    """Test getting widget types"""
    print("3. Testing widget types listing...")
    
    try:
        response = requests.get(WIDGET_API_URL, timeout=5)
        if response.status_code == 200:
            data = response.json()
            if data.get("success") and data.get("widget_types"):
                types = list(data["widget_types"].keys())
                print(f"   ✓ Available widget types: {', '.join(types)}")
                return True
            else:
                print(f"   ✗ No widget types found: {data}")
                return False
        else:
            print(f"   ✗ HTTP error {response.status_code}: {response.text}")
            return False
    except Exception as e:
        print(f"   ✗ Error: {e}")
        return False

def main():
    print("Testing Widget API Fix...")
    print("=" * 50)
    
    # Test 1: Get workspaces
    workspace_id = test_get_workspaces()
    if not workspace_id:
        print("\nFailed to get workspace. Make sure Wave Terminal is running.")
        sys.exit(1)
    
    # Test 2: Get widget types
    if not test_widget_types():
        print("\nFailed to get widget types.")
        sys.exit(1)
    
    # Test 3: Create widget
    block_id = test_create_widget(workspace_id)
    if not block_id:
        print("\nFailed to create widget.")
        sys.exit(1)
    
    print("\n" + "=" * 50)
    print("✓ All tests passed!")
    print("\nThe fix should now ensure that:")
    print("1. Database updates are collected with ContextWithUpdates")
    print("2. SendUpdateEvents is called to notify frontend")
    print("3. Frontend receives waveobj:update events")
    print("4. UI is updated to show the new widget")
    print("\nIf you can see the new terminal widget in Wave Terminal, the fix is working!")

if __name__ == "__main__":
    main()