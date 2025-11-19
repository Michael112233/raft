// Package config 定义了系统的配置信息
// 包括节点地址映射等
package config

// NodeAddrs 节点 ID 到网络地址的映射表
// key: 节点 ID (int64)
// value: 节点网络地址 (格式: IP:Port)
var NodeAddrs map[int64]string

// init 初始化配置
// 设置四个节点的网络地址（本地回环地址，不同端口）
func init() {
	NodeAddrs = make(map[int64]string)
	// 节点 0 的地址
	NodeAddrs[0] = "127.0.0.1:38000"
	// 节点 1 的地址
	NodeAddrs[1] = "127.0.0.1:38100"
	// 节点 2 的地址
	NodeAddrs[2] = "127.0.0.1:38200"
	// 节点 3 的地址
	NodeAddrs[3] = "127.0.0.1:38300"
}
