// Package network 提供了网络连接管理的功能
// 包括线程安全的连接映射表，用于管理到其他节点的 TCP 连接
package network

import (
	"net"
	"sync"
)

// ConnectionsMap 线程安全的连接映射表
// 用于存储和管理到其他节点的 TCP 连接，避免每次发送消息都重新建立连接
// 使用读写锁保证并发安全
type ConnectionsMap struct {
	sync.RWMutex                    // 读写锁，保证并发安全
	Connections map[string]net.Conn // 地址到连接的映射表
}

// NewConnectionsMap 创建新的连接映射表实例
// 返回:
//   - *ConnectionsMap: 初始化完成的连接映射表
func NewConnectionsMap() *ConnectionsMap {
	return &ConnectionsMap{
		Connections: make(map[string]net.Conn),
	}
}

// Add 向映射表中添加连接（线程安全）
// 参数:
//   - key: 节点地址（格式: IP:Port）
//   - conn: TCP 连接对象
func (cm *ConnectionsMap) Add(key string, conn net.Conn) {
	cm.Lock()
	cm.Connections[key] = conn
	cm.Unlock()
}

// Get 从映射表中获取连接（线程安全）
// 参数:
//   - key: 节点地址（格式: IP:Port）
// 返回:
//   - net.Conn: TCP 连接对象，如果不存在则为 nil
//   - bool: 连接是否存在
func (cm *ConnectionsMap) Get(key string) (net.Conn, bool) {
	cm.RLock()
	conn, ok := cm.Connections[key]
	cm.RUnlock()
	return conn, ok
}

// Remove 从映射表中移除连接（线程安全）
// 参数:
//   - key: 节点地址（格式: IP:Port）
func (cm *ConnectionsMap) Remove(key string) {
	cm.Lock()
	delete(cm.Connections, key)
	cm.Unlock()
}
