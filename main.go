// Package main 是 Raft 共识算法实现的入口程序
// 该程序启动一个 Raft 节点，节点通过命令行参数指定其 ID
package main

import (
	"Raft/node"
	"flag"
	"fmt"
)

// main 是程序的入口函数
// 功能：
//  1. 解析命令行参数获取节点 ID
//  2. 创建并启动 Raft 节点
func main() {
	// 解析命令行参数，获取节点 ID（默认为 0）
	nodeID := flag.Int64("node-id", 0, "Node ID to start")
	flag.Parse()

	// 创建新的 Raft 节点实例
	node := node.NewNode(*nodeID)
	// 启动节点，开始 Raft 协议流程
	node.Start()

	fmt.Println("Node started")
}
