// Package node 的消息处理模块
// 负责节点之间的网络通信，包括消息的发送、接收、序列化和反序列化
package node

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"Raft/core"
	"Raft/logger"
	"Raft/network"
)

var (
	// conns2Node 全局连接映射表，用于管理到其他节点的 TCP 连接
	// 避免每次发送消息都重新建立连接，提高性能
	conns2Node *network.ConnectionsMap
	// listenConn 监听器，用于接收来自其他节点的连接
	listenConn net.Listener
)

// init 初始化全局变量
func init() {
	conns2Node = network.NewConnectionsMap()
}

// MessageHandler 消息处理器
// 负责处理节点之间的网络通信，包括：
//   - 建立和维护到其他节点的连接
//   - 监听来自其他节点的连接
//   - 消息的序列化（编码）和反序列化（解码）
//   - 消息的发送和接收
type MessageHandler struct {
	exitChan chan struct{}  // 退出通道，用于优雅关闭
	node_ref *Node          // 指向所属节点的引用
	log      *logger.Logger // 日志记录器
}

// NewMessageHandler 创建新的消息处理器实例
// 返回:
//   - *MessageHandler: 初始化完成的消息处理器
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		exitChan: make(chan struct{}, 1), // 创建退出通道
	}
}

// Start 启动消息处理器
// 功能：
//  1. 保存节点引用和日志记录器
//  2. 在后台 goroutine 中启动监听服务，等待其他节点的连接
//
// 参数:
//   - node: 所属节点实例
//   - wg: WaitGroup，用于同步等待监听服务启动完成
func (hub *MessageHandler) Start(node *Node, wg *sync.WaitGroup) {
	if node != nil {
		hub.node_ref = node
		hub.log = node.logger
		wg.Add(1)
		// 在后台启动监听服务
		go hub.listen(hub.node_ref.Address, wg)
	}
}

// --------------------------------------------------------
// Basic Communication Principles Implementation (like Dial & Listen)
// --------------------------------------------------------

// Dial 建立到指定地址的 TCP 连接
// 功能：
//  1. 尝试连接目标地址，超时时间为 5 秒
//  2. 如果第一次连接失败，等待 100ms 后重试一次（处理节点启动时序问题）
//
// 参数:
//   - addr: 目标节点的网络地址（格式：IP:Port）
//
// 返回:
//   - net.Conn: TCP 连接对象
//   - error: 连接错误，如果连接成功则为 nil
func (hub *MessageHandler) Dial(addr string) (net.Conn, error) {
	// 设置连接超时时间为 5 秒
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		hub.log.Debug(fmt.Sprintf("DialTCPError: target_addr=%s, err=%v", addr, err))
		// 如果第一次连接失败，等待 100ms 后重试（可能目标节点还在启动中）
		time.Sleep(100 * time.Millisecond)
		hub.log.Debug(fmt.Sprintf("Try dial again... target_addr=%s", addr))
		conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			hub.log.Debug(fmt.Sprintf("DialTCPError: target_addr=%s, err=%v", addr, err))
			return nil, err
		} else {
			hub.log.Debug(fmt.Sprintf("dial success. target_addr=%s", addr))
		}
	}
	return conn, nil
}

// packMsg 将消息打包成网络传输格式
// 功能：
//  1. 使用 gob 编码将消息序列化为字节流
//  2. 在消息前添加 4 字节的长度前缀（大端序），用于解决 TCP 粘包问题
//
// 参数:
//   - msgType: 消息类型（如 "MsgRequestVote"）
//   - data: 消息数据（已序列化的字节流）
//
// 返回:
//   - []byte: 打包后的消息字节流（格式：4字节长度 + 消息内容）
func (hub *MessageHandler) packMsg(msgType string, data []byte) []byte {
	// 构造消息结构
	msg := &core.Message{
		MsgType: msgType,
		Data:    data,
	}

	// 使用 gob 编码器将消息序列化为字节流
	var buf bytes.Buffer
	msgEnc := gob.NewEncoder(&buf)
	err := msgEnc.Encode(msg)
	if err != nil {
		hub.log.Error(fmt.Sprintf("gobEncodeErr: err=%v, msg=%v", err, msg))
	}

	msgBytes := buf.Bytes()

	// 在消息前添加 4 字节的长度前缀（大端序），用于解决 TCP 粘包问题
	// 接收方可以先读取 4 字节获取消息长度，再读取完整消息
	networkBuf := make([]byte, 4+len(msgBytes))
	binary.BigEndian.PutUint32(networkBuf[:4], uint32(len(msgBytes)))
	copy(networkBuf[4:], msgBytes)

	return networkBuf
}

// listen 启动 TCP 监听服务，等待其他节点的连接
// 功能：
//  1. 在指定地址上启动 TCP 监听
//  2. 循环接受来自其他节点的连接
//  3. 为每个连接启动独立的 goroutine 处理消息
//
// 参数:
//   - addr: 监听地址（格式：IP:Port）
//   - wg: WaitGroup，用于通知调用者监听服务已启动
func (hub *MessageHandler) listen(addr string, wg *sync.WaitGroup) {
	defer wg.Done() // 函数退出时通知 WaitGroup
	// 在指定地址上启动 TCP 监听
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		hub.log.Error(fmt.Sprintf("Error setting up listener: err=%v", err))
		return
	}
	hub.log.Info(fmt.Sprintf("start listening on %s", addr))
	listenConn = ln
	defer ln.Close() // 函数退出时关闭监听器

	// 循环接受连接
	for {
		// 注释掉的代码：可以设置超时，超过时间限制没有收到新连接则退出
		// ln.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
		// 接受来自其他节点的连接
		conn, err := ln.Accept()
		if err != nil {
			hub.log.Debug(fmt.Sprintf("Error accepting connection: err=%v", err))
			return
		}
		// 为每个连接启动独立的 goroutine 处理消息（支持并发处理多个连接）
		go hub.handleConnection(conn, ln)
	}
}

// unpackMsg 将网络传输格式的消息解包
// 功能：
//  1. 使用 gob 解码器将字节流反序列化为消息结构
//
// 参数:
//   - packedMsg: 打包后的消息字节流（不包含长度前缀，只有消息内容）
//
// 返回:
//   - *core.Message: 解包后的消息结构
func (hub *MessageHandler) unpackMsg(packedMsg []byte) *core.Message {
	var networkBuf bytes.Buffer
	networkBuf.Write(packedMsg)
	// 使用 gob 解码器反序列化消息
	msgDec := gob.NewDecoder(&networkBuf)

	var msg core.Message
	err := msgDec.Decode(&msg)
	if err != nil {
		hub.log.Error(fmt.Sprintf("unpackMsgErr: err=%v, msgBytes=%v", err, packedMsg))
	}

	return &msg
}

// Send 发送消息的统一入口
// 根据消息类型调用相应的发送函数
// 参数:
//   - msgType: 消息类型（如 "MsgRequestVote"）
//   - ip: 目标节点地址（当前未使用，消息数据中已包含目标地址）
//   - msg: 消息数据（具体类型根据 msgType 而定）
//   - callback: 回调函数（当前未使用）
func (hub *MessageHandler) Send(msgType string, ip string, msg interface{}, callback func(...interface{})) {
	switch msgType {
	case core.MsgRequestVote:
		// 发送投票请求消息
		if data, ok := msg.(core.RequestVoteData); ok {
			hub.sendRequestVote(data)
		}
	case core.MsgRequestVoteResponse:
		// 发送投票响应消息
		if data, ok := msg.(core.RequestVoteResponseData); ok {
			hub.sendRequestVoteResponse(data)
		}
	case core.MsgAppendEntries:
		// 发送追加日志条目消息
		if data, ok := msg.(core.AppendEntriesData); ok {
			hub.sendAppendEntries(data)
		}
	default:
		hub.log.Error("Unknown message type received. msgType=" + msgType)
	}
}

// handleConnection 处理来自其他节点的连接
// 功能：
//  1. 读取消息长度前缀（4 字节）
//  2. 根据长度读取完整消息
//  3. 解包消息并根据消息类型分发到相应的处理函数
//
// 参数:
//   - conn: TCP 连接对象
//   - ln: 监听器（当前未使用）
func (hub *MessageHandler) handleConnection(conn net.Conn, ln net.Listener) {
	defer conn.Close() // 函数退出时关闭连接
	for {
		// 第一步：读取消息长度前缀（4 字节，大端序）
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(conn, lenBuf)
		if err != nil {
			if err.Error() == "EOF" {
				// 发送端主动关闭连接，正常退出
				return
			}
			hub.log.Debug(fmt.Sprintf("Error reading from connection: err=%v", err))
			return
		}
		// 解析消息长度
		length := int(binary.BigEndian.Uint32(lenBuf))
		// 第二步：根据长度读取完整消息内容
		packedMsg := make([]byte, length)
		_, err = io.ReadFull(conn, packedMsg)
		if err != nil {
			hub.log.Error(fmt.Sprintf("Error reading from connection: err=%v", err))
		}

		// 第三步：解包消息并根据类型分发处理
		msg := hub.unpackMsg(packedMsg)
		switch msg.MsgType {
		case core.MsgRequestVote:
			// 处理投票请求消息
			hub.handleRequestVoteMessage(msg.Data)
		case core.MsgRequestVoteResponse:
			// 处理投票响应消息
			hub.handleRequestVoteResponseMessage(msg.Data)
		case core.MsgAppendEntries:
			// 处理追加日志条目消息
			hub.handleAppendEntriesMessage(msg.Data)
		default:
			hub.log.Error(fmt.Sprintf("Unknown message type received: msgType=%s", msg.MsgType))
		}
	}
}

// --------------------------------------------------------
// Message Handler Implementation Send
// --------------------------------------------------------

// sendRequestVote 发送投票请求消息
// 功能：
//  1. 序列化投票请求数据
//  2. 打包消息（添加长度前缀）
//  3. 获取或建立到目标节点的连接
//  4. 发送消息
//
// 参数:
//   - data: 投票请求数据
func (hub *MessageHandler) sendRequestVote(data core.RequestVoteData) {
	// 序列化投票请求数据
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("gobEncodeErr. Send Request Vote Message. caller: %s targetAddr: %s", data.From, data.To))
	}

	// 打包消息（添加长度前缀）
	msg_bytes := hub.packMsg(core.MsgRequestVote, buf.Bytes())

	// 获取或建立到目标节点的连接
	addr := data.To
	conn, ok := conns2Node.Get(addr)
	if !ok || conn == nil {
		// 连接不存在，建立新连接
		conn, err = hub.Dial(addr)
		if err != nil || conn == nil {
			hub.log.Error(fmt.Sprintf("Dial Error. Send Request Vote Message. caller: %s targetAddr: %s", data.From, addr))
			return
		}
		// 将新连接添加到连接映射表
		conns2Node.Add(addr, conn)
	}
	// 使用带缓冲的写入器发送消息
	writer := bufio.NewWriter(conn)
	if _, err := writer.Write(msg_bytes); err != nil {
		hub.log.Error(fmt.Sprintf("Write Error. Send Request Vote Message. caller: %s targetAddr: %s, err=%v", data.From, addr, err))
		return
	}
	// 刷新缓冲区，确保消息立即发送
	if err := writer.Flush(); err != nil {
		hub.log.Error(fmt.Sprintf("Flush Error. Send Request Vote Message. caller: %s targetAddr: %s, err=%v", data.From, addr, err))
		return
	}
}

// sendRequestVoteResponse 发送投票响应消息
// 功能：
//  1. 序列化投票响应数据
//  2. 打包消息（添加长度前缀）
//  3. 获取或建立到目标节点的连接
//  4. 发送消息
//
// 参数:
//   - data: 投票响应数据
func (hub *MessageHandler) sendRequestVoteResponse(data core.RequestVoteResponseData) {
	// 序列化投票响应数据
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("gobEncodeErr. Send Request Vote Message. caller: %s targetAddr: %s", data.From, data.To))
	}

	// 打包消息（添加长度前缀）
	msg_bytes := hub.packMsg(core.MsgRequestVoteResponse, buf.Bytes())

	// 获取或建立到目标节点的连接
	addr := data.To
	conn, ok := conns2Node.Get(addr)
	if !ok || conn == nil {
		// 连接不存在，建立新连接
		conn, err = hub.Dial(addr)
		if err != nil || conn == nil {
			hub.log.Error(fmt.Sprintf("Dial Error. Send Request Vote Response Message. caller: %s targetAddr: %s", data.From, addr))
			return
		}
		// 将新连接添加到连接映射表
		conns2Node.Add(addr, conn)
	}
	// 使用带缓冲的写入器发送消息
	writer := bufio.NewWriter(conn)
	if _, err := writer.Write(msg_bytes); err != nil {
		hub.log.Error(fmt.Sprintf("Write Error. Send Request Vote Response Message. caller: %s targetAddr: %s, err=%v", data.From, addr, err))
		return
	}
	// 刷新缓冲区，确保消息立即发送
	if err := writer.Flush(); err != nil {
		hub.log.Error(fmt.Sprintf("Flush Error. Send Request Vote Response Message. caller: %s targetAddr: %s, err=%v", data.From, addr, err))
		return
	}
}

// sendAppendEntries 发送追加日志条目消息
// 功能：
//  1. 序列化追加日志条目数据
//  2. 打包消息（添加长度前缀）
//  3. 获取或建立到目标节点的连接
//  4. 发送消息
//
// 参数:
//   - data: 追加日志条目数据
func (hub *MessageHandler) sendAppendEntries(data core.AppendEntriesData) {
	// 序列化追加日志条目数据
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("gobEncodeErr. Send Append Entries Message. caller: %s targetAddr: %s", data.CurrentLeader, data.To))
	}

	// 打包消息（添加长度前缀）
	msg_bytes := hub.packMsg(core.MsgAppendEntries, buf.Bytes())

	// 获取或建立到目标节点的连接
	addr := data.To
	conn, ok := conns2Node.Get(addr)
	if !ok || conn == nil {
		// 连接不存在，建立新连接
		conn, err = hub.Dial(addr)
		if err != nil || conn == nil {
			hub.log.Error(fmt.Sprintf("Dial Error. Send Append Entries Message. caller: %s targetAddr: %s", data.CurrentLeader, addr))
			return
		}
		// 将新连接添加到连接映射表
		conns2Node.Add(addr, conn)
	}
	// 使用带缓冲的写入器发送消息
	writer := bufio.NewWriter(conn)
	if _, err := writer.Write(msg_bytes); err != nil {
		hub.log.Error(fmt.Sprintf("Write Error. Send Append Entries Message. caller: %s targetAddr: %s, err=%v", data.CurrentLeader, addr, err))
		return
	}
	// 刷新缓冲区，确保消息立即发送
	if err := writer.Flush(); err != nil {
		hub.log.Error(fmt.Sprintf("Flush Error. Send Append Entries Message. caller: %s targetAddr: %s, err=%v", data.CurrentLeader, addr, err))
		return
	}
}

// --------------------------------------------------------
// Message Handler Implementation Receive
// --------------------------------------------------------

// handleRequestVoteMessage 处理接收到的投票请求消息
// 功能：
//  1. 反序列化投票请求数据
//  2. 将消息转发给节点处理
//
// 参数:
//   - dataBytes: 消息数据的字节流
func (hub *MessageHandler) handleRequestVoteMessage(dataBytes []byte) {
	// 反序列化投票请求数据
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.RequestVoteData
	err := dataDec.Decode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("handleRequestVoteMessageErr: err=%v, dataBytes=%v", err, dataBytes))
	}
	// 将消息转发给节点处理
	hub.node_ref.HandleRequestVoteMessage(data)
}

// handleAppendEntriesMessage 处理接收到的追加日志条目消息
// 功能：
//  1. 反序列化追加日志条目数据
//  2. 将消息转发给节点处理
//
// 参数:
//   - dataBytes: 消息数据的字节流
func (hub *MessageHandler) handleAppendEntriesMessage(dataBytes []byte) {
	// 反序列化追加日志条目数据
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.AppendEntriesData
	err := dataDec.Decode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("handleAppendEntriesMessageErr: err=%v, dataBytes=%v", err, dataBytes))
	}
	// 将消息转发给节点处理
	hub.node_ref.HandleAppendEntriesMessage(data)
}

// handleRequestVoteResponseMessage 处理接收到的投票响应消息
// 功能：
//  1. 反序列化投票响应数据
//  2. 将消息转发给节点处理
//
// 参数:
//   - dataBytes: 消息数据的字节流
func (hub *MessageHandler) handleRequestVoteResponseMessage(dataBytes []byte) {
	// 反序列化投票响应数据
	var buf bytes.Buffer
	buf.Write(dataBytes)
	dataDec := gob.NewDecoder(&buf)

	var data core.RequestVoteResponseData
	err := dataDec.Decode(&data)
	if err != nil {
		hub.log.Error(fmt.Sprintf("handleRequestVoteResponseMessageErr: err=%v, dataBytes=%v", err, dataBytes))
	}
	// 将消息转发给节点处理
	hub.node_ref.HandleRequestVoteResponseMessage(data)
}
