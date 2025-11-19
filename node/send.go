// Package node 的消息发送模块
// 负责构造和发送 Raft 协议中的各种消息类型
package node

import (
	"Raft/config"
	"Raft/core"
	"fmt"
)

// SendRequestVote 向所有其他节点发送投票请求消息
// 功能：
//  1. 遍历所有节点地址
//  2. 跳过自己，向其他每个节点发送投票请求
//  3. 投票请求包含当前任期号和节点地址信息
//
// 使用场景：
//
//	当节点想要成为 Leader 时，会调用此函数发起选举
func (n *Node) SendRequestVote() {
	n.logger.Info(fmt.Sprintf("Node %d start view change", n.Id))

	// 遍历所有节点地址
	for id, addr := range config.NodeAddrs {
		// 跳过自己，不向自己发送投票请求
		if id == n.Id {
			continue
		}
		// 构造投票请求数据
		requestVoteData := core.RequestVoteData{
			Term: n.Term,    // 当前任期号
			From: n.Address, // 发送方地址（自己）
			To:   addr,      // 接收方地址（目标节点）
		}
		// 通过消息处理器发送投票请求
		n.messageHandler.Send(core.MsgRequestVote, addr, requestVoteData, nil)
		n.logger.Info(fmt.Sprintf("Send Request Vote Message to %s", addr))
	}
}

// SendRequestVoteResponse 发送投票响应消息
// 功能：
//  1. 构造投票响应数据（同意投票）
//  2. 向请求投票的节点发送响应
//
// 参数:
//   - data: 收到的投票请求数据，用于获取发送方地址
//
// 使用场景：
//
//	当节点收到其他节点的投票请求，且满足投票条件时调用
func (n *Node) SendRequestVoteResponse(data core.RequestVoteData) {
	// 构造投票响应数据，同意投票
	response := core.RequestVoteResponseData{
		From:        n.Address, // 发送方地址（自己）
		To:          data.From, // 接收方地址（请求投票的节点）
		Term:        data.Term, // 任期号（与请求中的任期号一致）
		VoteGranted: true,      // 同意投票
	}
	// 通过消息处理器发送投票响应
	n.messageHandler.Send(core.MsgRequestVoteResponse, data.From, response, nil)
	n.logger.Info(fmt.Sprintf("Send Request Vote Response Message to %s", data.From))
}

// SendAppendEntries 向所有节点发送追加日志条目消息
// 功能：
//  1. 构造追加日志条目数据（包含新的任期号、投票数和 Leader 信息）
//  2. 向所有节点（包括自己）发送消息，通知它们新的 Leader 已产生
//
// 使用场景：
//
//	当节点获得超过半数投票成为 Leader 后调用，通知所有节点选举结果
func (n *Node) SendAppendEntries() {
	// 构造追加日志条目数据
	appendEntriesData := core.AppendEntriesData{
		Term:          n.Term + 1,          // 新任期号（当前任期 + 1）
		VoteNumber:    n.GetCurrentVotes(), // 获得的投票数
		CurrentLeader: n.Address,           // 当前 Leader 地址（自己）
	}
	// 向所有节点发送追加日志条目消息
	for _, addr := range config.NodeAddrs {
		appendEntriesData.To = addr // 设置目标节点地址
		n.messageHandler.Send(core.MsgAppendEntries, addr, appendEntriesData, nil)
		n.logger.Info(fmt.Sprintf("Send Append Entries Message to %s", addr))
	}
}
