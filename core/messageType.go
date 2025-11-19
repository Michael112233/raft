// Package core 定义了 Raft 协议中使用的核心消息类型和数据结构
package core

// Message 是网络传输中的通用消息结构
// 包含消息类型和序列化后的数据
type Message struct {
	MsgType string // 消息类型（如 "MsgRequestVote"）
	Data    []byte // 消息数据（已序列化的字节流）
}

// RequestVoteData 投票请求消息的数据结构
// 候选者向其他节点发送，请求投票支持
type RequestVoteData struct {
	Term int64  // 候选者的任期号
	From string // 发送方节点地址
	To   string // 接收方节点地址
}

// RequestVoteResponseData 投票响应消息的数据结构
// 节点对投票请求的响应
type RequestVoteResponseData struct {
	Term        int64  // 任期号（与请求中的任期号一致）
	VoteGranted bool   // 是否同意投票
	From        string // 发送方节点地址（投票的节点）
	To          string // 接收方节点地址（候选者）
}

// AppendEntriesData 追加日志条目消息的数据结构
// Leader 向所有节点发送，用于：
//  1. 通知节点新的 Leader 已产生
//  2. 同步日志条目（在完整实现中）
type AppendEntriesData struct {
	Term          int64  // 新的任期号
	VoteNumber    int    // Leader 获得的投票数
	CurrentLeader string // 当前 Leader 的地址
	To            string // 接收方节点地址
}
