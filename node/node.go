// Package node 实现了 Raft 共识算法的节点核心逻辑
// 包括节点的生命周期管理、投票机制、任期管理等
package node

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"Raft/config"
	"Raft/logger"
)

// Node 表示一个 Raft 节点
// 每个节点维护自己的状态，包括：
//   - Id: 节点唯一标识符
//   - Address: 节点的网络地址
//   - Term: 当前任期号（Raft 协议中的核心概念）
//   - logger: 日志记录器，用于记录节点运行日志
//   - messageHandler: 消息处理器，负责与其他节点通信
//   - hasVoted: 当前任期内是否已投票（防止重复投票）
//   - currentVotes: 当前获得的投票数（用于判断是否成为 Leader）
type Node struct {
	Id      int64      // 节点 ID
	Address string     // 节点网络地址
	Term    int64      // 当前任期号
	TermMu  sync.Mutex // 保护 Term 的互斥锁

	logger         *logger.Logger  // 日志记录器
	messageHandler *MessageHandler // 消息处理器

	mu           sync.Mutex // 保护 hasVoted 和 Term 的互斥锁
	hasVoted     bool       // 当前任期内是否已投票
	voteMu       sync.Mutex // 保护 currentVotes 的互斥锁
	currentVotes int        // 当前获得的投票数
}

// NewNode 创建一个新的 Raft 节点实例
// 参数:
//   - id: 节点 ID
//
// 返回:
//   - *Node: 初始化完成的节点实例
func NewNode(id int64) *Node {
	// 为节点创建专用的日志记录器
	logger := logger.NewLogger(id, "node")
	return &Node{
		Id:             id,
		Address:        config.NodeAddrs[id], // 从配置中获取节点地址
		Term:           0,
		TermMu:         sync.Mutex{}, // 初始任期为 0
		logger:         logger,
		currentVotes:   0,                   // 初始投票数为 0
		messageHandler: NewMessageHandler(), // 创建消息处理器
	}
}

// Start 启动节点，开始 Raft 协议流程
// 功能：
//  1. 启动消息处理器，开始监听网络连接
//  2. 在后台启动 Raft 协议流程（发起选举等）
func (n *Node) Start() {
	n.logger.Info(fmt.Sprintf("Node %d started", n.Id))
	// 使用 WaitGroup 等待消息处理器启动完成
	wg := &sync.WaitGroup{}
	// 启动消息处理器，开始监听网络连接
	n.messageHandler.Start(n, wg)
	// 在后台 goroutine 中启动 Raft 协议流程
	go n.startRaft()
	// 主线程等待 15 秒（给节点足够时间完成选举等操作）
	time.Sleep(15 * time.Second)
}

// startRaft 启动 Raft 协议的核心流程
// 功能：
//  1. 随机延迟一段时间（0-1000ms），避免所有节点同时发起选举
//  2. 发起选举请求，向其他节点请求投票
func (n *Node) startRaft() {
	n.logger.Info(fmt.Sprintf("Node %d started raft", n.Id))
	// 随机延迟 0-1000ms，避免所有节点同时发起选举（防止选举冲突）
	sleepDuration := time.Duration(rand.Intn(1000)) * time.Millisecond
	time.Sleep(sleepDuration)

	// 向其他节点发送投票请求，开始选举流程
	n.SendRequestVote()
}

// SetHasVoted 设置节点的投票状态（线程安全）
// 参数:
//   - hasVoted: 是否已投票
func (n *Node) SetHasVoted(hasVoted bool) {
	n.mu.Lock() // 锁定互斥锁，防止多个线程同时设置投票状态
	defer n.mu.Unlock()
	n.hasVoted = hasVoted
}

// GetHasVoted 获取节点的投票状态（线程安全）
// 返回:
//   - bool: 当前任期内是否已投票
func (n *Node) GetHasVoted() bool {
	n.mu.Lock() // 锁定互斥锁，防止多个线程同时获取投票状态
	defer n.mu.Unlock()
	return n.hasVoted
}

// AddCurrentVotes 增加当前获得的投票数（线程安全）
// 当收到其他节点的投票响应时调用
func (n *Node) AddCurrentVotes() {
	n.voteMu.Lock() // 锁定互斥锁，防止多个线程同时增加投票数
	defer n.voteMu.Unlock()
	n.currentVotes++
}

// GetCurrentVotes 获取当前获得的投票数（线程安全）
// 返回:
//   - int: 当前获得的投票数
func (n *Node) GetCurrentVotes() int {
	n.voteMu.Lock() // 锁定互斥锁，防止多个线程同时获取投票数
	defer n.voteMu.Unlock()
	return n.currentVotes
}

// SetTerm 设置节点的当前任期（线程安全）
// 参数:
//   - term: 新的任期号
func (n *Node) SetTerm(term int64) {
	n.TermMu.Lock() // 锁定互斥锁，防止多个线程同时设置任期号
	defer n.TermMu.Unlock()
	n.Term = term
}

// Close 关闭节点，停止消息处理器
// 当节点收到 AppendEntries 消息（表示已有 Leader）时调用
func (n *Node) Close() {
	// 向消息处理器的退出通道发送信号
	n.messageHandler.exitChan <- struct{}{}
	n.logger.Info(fmt.Sprintf("Node %d closed", n.Id))
}
