#!/bin/bash

# 在四个独立终端窗口中运行四个 Raft 节点的脚本

# 检查是否已经编译
if [ ! -f "./raft" ]; then
    echo "正在编译程序..."
    go build -o raft main.go
    if [ $? -ne 0 ]; then
        echo "编译失败！"
        exit 1
    fi
fi

# 清理 logs 目录中的所有日志文件
echo "清理日志文件..."
rm -f logs/*.log
mkdir -p logs

# 获取当前目录的绝对路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "正在启动四个节点，每个节点将在独立的终端窗口中运行..."

# 在 macOS 上打开四个新的 Terminal 窗口
osascript <<EOF
tell application "Terminal"
    -- 打开第一个节点 (Node 0)
    do script "cd '$SCRIPT_DIR' && ./raft -node-id 0"
    set custom title of front window to "Raft Node 0"
    
    -- 打开第二个节点 (Node 1)
    do script "cd '$SCRIPT_DIR' && ./raft -node-id 1"
    set custom title of front window to "Raft Node 1"
    
    -- 打开第三个节点 (Node 2)
    do script "cd '$SCRIPT_DIR' && ./raft -node-id 2"
    set custom title of front window to "Raft Node 2"
    
    -- 打开第四个节点 (Node 3)
    do script "cd '$SCRIPT_DIR' && ./raft -node-id 3"
    set custom title of front window to "Raft Node 3"
end tell
EOF

echo "四个节点已在独立的终端窗口中启动！"
echo "每个窗口的标题已设置为对应的节点 ID"