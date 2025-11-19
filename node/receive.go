// Package node 的消息接收处理模块
// 负责处理接收到的各种 Raft 协议消息，实现 Raft 算法的核心逻辑
package node

import (
	"Raft/config"
	"Raft/core"
	"fmt"
)

// HandleRequestVoteMessage 处理接收到的投票请求消息
// 功能：
//  1. 检查请求的任期号是否小于当前任期（如果是则拒绝）
//  2. 检查当前任期内是否已投票（如果已投票则拒绝）
//  3. 如果满足条件，则同意投票并发送响应
//
// 参数:
//   - data: 投票请求数据
//
// Raft 协议规则：
//   - 每个节点在每个任期内只能投票给一个候选者
//   - 只能投票给任期号大于等于当前任期的候选者
func (n *Node) HandleRequestVoteMessage(data core.RequestVoteData) {
	// 如果请求的任期号小于当前任期，拒绝投票
	if data.Term < n.Term {
		return
	}

	// 如果当前任期内已投票，拒绝投票（防止重复投票）
	if n.GetHasVoted() {
		return
	}
	n.logger.Info(fmt.Sprintf("Handle Request Vote Message: from %s to %s, term %d", data.From, data.To, data.Term))

	// 标记已投票，防止重复投票
	n.SetHasVoted(true)
	// 发送投票响应，同意投票
	n.SendRequestVoteResponse(data)
}

// HandleRequestVoteResponseMessage 处理接收到的投票响应消息
// 功能：
//  1. 检查响应的任期号是否小于当前任期（如果是则忽略）
//  2. 增加获得的投票数
//  3. 如果获得的投票数超过半数，则成为 Leader 并发送 AppendEntries 消息
//
// 参数:
//   - data: 投票响应数据
//
// Raft 协议规则：
//   - 候选者需要获得超过半数的投票才能成为 Leader
//   - 成为 Leader 后需要向所有节点发送 AppendEntries 消息
func (n *Node) HandleRequestVoteResponseMessage(data core.RequestVoteResponseData) {
	// 如果响应的任期号小于当前任期，忽略该响应
	if data.Term < n.Term {
		return
	}
	n.logger.Info(fmt.Sprintf("Handle Request Vote Response Message: from %s to %s, term %d, vote granted %t", data.From, data.To, data.Term, data.VoteGranted))

	// 增加获得的投票数
	n.AddCurrentVotes()
	// 检查是否获得超过半数的投票
	if n.GetCurrentVotes() > len(config.NodeAddrs)/2 {
		n.logger.Info(fmt.Sprintf("Node %d become leader", n.Id))
		// 成为 Leader 后，向所有节点发送 AppendEntries 消息
		n.SendAppendEntries()
	}
}

// HandleAppendEntriesMessage 处理接收到的追加日志条目消息
// 功能：
//  1. 检查消息的任期号是否小于当前任期（如果是则忽略）
//  2. 更新当前任期号
//  3. 记录当前 Leader 信息
//  4. 关闭节点（表示选举过程结束，已有 Leader）
//
// 参数:
//   - data: 追加日志条目数据
//
// 使用场景：
//
//	当节点收到来自 Leader 的 AppendEntries 消息时，表示选举已完成，有节点成为了 Leader
//	此时节点应该停止选举过程，接受 Leader 的领导
func (n *Node) HandleAppendEntriesMessage(data core.AppendEntriesData) {
	// 如果消息的任期号小于当前任期，忽略该消息
	if data.Term < n.Term {
		return
	}
	n.logger.Info(fmt.Sprintf("Handle Append Entries Message: from %s to %s, term %d, vote number %d", data.CurrentLeader, data.To, data.Term, data.VoteNumber))

	// 更新当前任期号
	n.SetTerm(data.Term)
	n.logger.Info(fmt.Sprintf("current leader: %s", data.CurrentLeader))
	n.logger.Info("Raft ended!")

	// 关闭节点，停止选举过程
	n.Close()
}
