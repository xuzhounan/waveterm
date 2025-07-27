#!/bin/bash

# 清理错误创建的$HOME目录结构

echo "正在清理错误创建的目录结构..."

# 移到项目目录
cd "/Users/xzn/Desktop/code-project/waveterm"

# 检查并显示要删除的目录
if [ -d "\$HOME" ]; then
    echo "发现错误目录: \$HOME"
    echo "目录内容:"
    find "\$HOME" -type f | head -10
    
    echo ""
    echo "删除错误目录..."
    rm -rf "\$HOME"
    
    if [ $? -eq 0 ]; then
        echo "✅ 成功删除错误目录"
    else
        echo "❌ 删除失败"
    fi
else
    echo "✅ 未发现错误目录"
fi

echo "清理完成"