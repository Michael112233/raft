// Package core 定义了 Raft 协议中使用的消息类型常量
package core

const (
	// MsgRequestVote 投票请求消息类型
	// 候选者向其他节点发送，请求投票支持
	MsgRequestVote string = "MsgRequestVote"

	// MsgRequestVoteResponse 投票响应消息类型
	// 节点对投票请求的响应
	MsgRequestVoteResponse string = "MsgRequestVoteResponse"

	// MsgAppendEntries 追加日志条目消息类型
	// Leader 向所有节点发送，用于通知新的 Leader 已产生和同步日志
	MsgAppendEntries string = "MsgAppendEntries"
)