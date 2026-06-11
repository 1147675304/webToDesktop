// pkg/bridge/serial_bridge.go
// 串口⇆网络 双向透传桥接
//
// 支持三种网络模式，将网络数据与串口数据双向透明转发，
// 桥接数据通过 stream-data 事件推送到前端终端显示。
//
// 模式:
//   TCP Server  — 监听端口，接受客户端连接后桥接
//   TCP Client  — 主动连接远端，建立后桥接
//   UDP         — 绑定本地端口，收发 UDP 数据报
//
// JS 调用:
//   window.__lhpanda__('startSerialTcpServer', {port: 8888})
//   window.__lhpanda__('startSerialTcpClient', {host: "192.168.1.1", port: 8888})
//   window.__lhpanda__('startSerialUdp', {bindPort: 9999, remoteHost: "...", remotePort: 9999})
//   window.__lhpanda__('stopSerialBridge')
//   window.__lhpanda__('getSerialBridgeState')
//
// 桥接数据在终端显示格式:
//   [SER→NET] ...  绿色 — 串口收到数据转发到网络
//   [NET→SER] ...  蓝色 — 网络收到数据转发到串口
//
// 事件:
//   type="bridge", event="connected"     — 连接/绑定成功
//   type="bridge", event="disconnected"  — 连接断开

//go:build !minimal && !nostream && !noserial

package bridge

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// serialBridge 串口⇆网络桥接的运行时状态。
// 同一时间只允许一个活跃桥接实例。
type serialBridge struct {
	active   bool          // 桥接是否活跃
	mode     string        // "tcp-server" | "tcp-client" | "udp"
	cancel   chan struct{} // 关闭信号，close 后所有 goroutine 退出
	listener net.Listener  // TCP Server 模式的监听器
	conn     net.Conn      // 当前网络连接（TCP 或 UDP）
}

// 当前活跃的桥接实例（包级单例）
var (
	currentBridge   *serialBridge
	currentBridgeMu sync.Mutex
)

// ==============================================
// TCP Server — 监听端口，接受客户端后双向透传
// ==============================================

// HandleStartSerialTcpServer 启动 TCP Server 桥接模式。
//
// 在指定端口监听，接受一个客户端连接后，
// 串口数据→客户端，客户端数据→串口，双向透明转发。
// 客户端断开后可重新连接。
//
// JS: window.__lhpanda__('startSerialTcpServer', {port: 8888})
func (b *Bridge) HandleStartSerialTcpServer(params map[string]interface{}) (interface{}, error) {
	// 先停止旧的桥接
	stopSerialBridgeInternal()

	port := getInt(params, "port")
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("端口号范围 1-65535")
	}

	// 启动 TCP 监听
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("TCP 监听失败: %w", err)
	}

	// 获取当前串口和流通道
	currentSerial.mu.Lock()
	sp := currentSerial.port
	sa := currentSerial.active
	ch := currentSerial.ch
	currentSerial.mu.Unlock()

	if sp == nil || !sa {
		listener.Close()
		return nil, fmt.Errorf("请先打开串口")
	}

	// 注册桥接实例
	cb := &serialBridge{active: true, mode: "tcp-server", cancel: make(chan struct{}), listener: listener}
	currentBridgeMu.Lock()
	currentBridge = cb
	currentBridgeMu.Unlock()

	// 后台接受连接
	go func() {
		defer listener.Close()
		for {
			// 检查取消信号
			select {
			case <-cb.cancel:
				return
			default:
			}
			// 带超时的 Accept（避免阻塞无法退出）
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			conn, err := listener.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			cb.conn = conn
			// 推送连接事件到前端
			pushBridgeEvent(ch, "connected", fmt.Sprintf("客户端已连接: %s", conn.RemoteAddr()))
			// 双向透传
			runBidirectional(ch, sp, conn, cb.cancel)
			conn.Close()
			cb.conn = nil
			pushBridgeEvent(ch, "disconnected", "客户端已断开")
		}
	}()

	return map[string]interface{}{"listening": true, "port": port, "mode": "tcp-server"}, nil
}

// ==============================================
// TCP Client — 主动连接远端，双向透传
// ==============================================

// HandleStartSerialTcpClient 启动 TCP Client 桥接模式。
//
// 主动连接远端 TCP 服务器，连接成功后串口⇆网络双向透传。
// 连接断开后桥接自动停止。
//
// JS: window.__lhpanda__('startSerialTcpClient', {host: "192.168.1.1", port: 8888})
func (b *Bridge) HandleStartSerialTcpClient(params map[string]interface{}) (interface{}, error) {
	stopSerialBridgeInternal()

	host := getString(params, "host")
	if host == "" {
		return nil, fmt.Errorf("缺少 host 参数")
	}
	port := getInt(params, "port")
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("端口号范围 1-65535")
	}

	currentSerial.mu.Lock()
	sp := currentSerial.port
	sa := currentSerial.active
	ch := currentSerial.ch
	currentSerial.mu.Unlock()

	if sp == nil || !sa {
		return nil, fmt.Errorf("请先打开串口")
	}

	// 连接远端（5 秒超时）
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("TCP 连接失败: %w", err)
	}

	cb := &serialBridge{active: true, mode: "tcp-client", cancel: make(chan struct{}), conn: conn}
	currentBridgeMu.Lock()
	currentBridge = cb
	currentBridgeMu.Unlock()

	pushBridgeEvent(ch, "connected", fmt.Sprintf("已连接到 %s:%d", host, port))
	go func() {
		runBidirectional(ch, sp, conn, cb.cancel)
		conn.Close()
		pushBridgeEvent(ch, "disconnected", "连接已断开")
	}()

	return map[string]interface{}{"connected": true, "host": host, "port": port, "mode": "tcp-client"}, nil
}

// ==============================================
// UDP — 绑定本地端口，收发数据报
// ==============================================

// HandleStartSerialUdp 启动 UDP 桥接模式。
//
// 绑定本地 UDP 端口，可指定固定远端地址或自动学习来源地址。
// 串口数据→UDP 远端，UDP 数据→串口。
//
// JS: window.__lhpanda__('startSerialUdp', {bindPort: 9999, remoteHost: "192.168.1.1", remotePort: 9999})
func (b *Bridge) HandleStartSerialUdp(params map[string]interface{}) (interface{}, error) {
	stopSerialBridgeInternal()

	bindPort := getInt(params, "bindPort")
	if bindPort <= 0 {
		return nil, fmt.Errorf("缺少 bindPort 参数")
	}
	remoteHost := getString(params, "remoteHost")
	remotePort := getInt(params, "remotePort")

	currentSerial.mu.Lock()
	sp := currentSerial.port
	sa := currentSerial.active
	ch := currentSerial.ch
	currentSerial.mu.Unlock()

	if sp == nil || !sa {
		return nil, fmt.Errorf("请先打开串口")
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: bindPort})
	if err != nil {
		return nil, fmt.Errorf("UDP 绑定失败: %w", err)
	}

	cb := &serialBridge{active: true, mode: "udp", cancel: make(chan struct{}), conn: conn}
	currentBridgeMu.Lock()
	currentBridge = cb
	currentBridgeMu.Unlock()

	pushBridgeEvent(ch, "connected", fmt.Sprintf("UDP 已绑定 :%d", bindPort))

	go func() {
		defer conn.Close()
		buf := make([]byte, 4096)
		var remoteAddr *net.UDPAddr
		if remoteHost != "" && remotePort > 0 {
			remoteAddr = &net.UDPAddr{IP: net.ParseIP(remoteHost), Port: remotePort}
		}

		// 串口→UDP: 读取串口数据，转发到远端地址
		go func() {
			rb := make([]byte, 256)
			for {
				select {
				case <-cb.cancel:
					return
				default:
				}
				sp.SetReadTimeout(100 * time.Millisecond)
				n, err := sp.Read(rb)
				if err != nil || n == 0 {
					continue
				}
				if remoteAddr == nil {
					continue // 尚无远端地址，跳过
				}
				conn.WriteToUDP(rb[:n], remoteAddr)
				pushBridgedData(ch, rb[:n], "SER→NET", true)
			}
		}()

		// UDP→串口: 接收 UDP 数据，写入串口
		for {
			select {
			case <-cb.cancel:
				return
			default:
			}
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			if remoteAddr == nil {
				remoteAddr = addr // 自动学习来源地址
			}
			sp.Write(buf[:n])
			pushBridgedData(ch, buf[:n], "NET→SER", false)
		}
	}()

	result := map[string]interface{}{"bound": true, "bindPort": bindPort, "mode": "udp"}
	if remoteHost != "" {
		result["remoteHost"] = remoteHost
		result["remotePort"] = remotePort
	}
	return result, nil
}

// ==============================================
// 控制方法
// ==============================================

// HandleStopSerialBridge 停止当前串口桥接。
func (b *Bridge) HandleStopSerialBridge(params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"stopped": stopSerialBridgeInternal()}, nil
}

// HandleGetSerialBridgeState 获取桥接状态。
// 返回: {active: bool, mode: "tcp-server"|"tcp-client"|"udp"}
func (b *Bridge) HandleGetSerialBridgeState(params map[string]interface{}) (interface{}, error) {
	currentBridgeMu.Lock()
	cb := currentBridge
	currentBridgeMu.Unlock()
	if cb == nil {
		return map[string]interface{}{"active": false}, nil
	}
	return map[string]interface{}{"active": true, "mode": cb.mode}, nil
}

// ==============================================
// 内部实现
// ==============================================

// stopSerialBridgeInternal 停止并清理当前桥接实例。
// 关闭 cancel channel 通知所有 goroutine 退出，关闭网络连接。
func stopSerialBridgeInternal() bool {
	currentBridgeMu.Lock()
	defer currentBridgeMu.Unlock()
	if currentBridge == nil {
		return false
	}
	close(currentBridge.cancel)
	if currentBridge.listener != nil {
		currentBridge.listener.Close()
	}
	if currentBridge.conn != nil {
		currentBridge.conn.Close()
	}
	currentBridge = nil
	return true
}

// runBidirectional TCP 双向透传核心: 串口⇆网络。
// 启动两个 goroutine:
//   - 串口→网络: 持续读取串口，写入 TCP 连接
//   - 网络→串口: 持续读取 TCP 连接，写入串口
//
// 任一方向出错或 cancel 时退出。
func runBidirectional(ch chan<- interface{}, sp io.ReadWriter, conn net.Conn, cancel chan struct{}) {
	done := make(chan struct{}, 2)

	// 串口→网络
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 256)
		for {
			select {
			case <-cancel:
				return
			default:
			}
			// 设置读超时以响应取消信号
			if p, ok := sp.(interface{ SetReadTimeout(time.Duration) error }); ok {
				p.SetReadTimeout(100 * time.Millisecond)
			}
			n, err := sp.Read(buf)
			if err != nil || n == 0 {
				continue
			}
			// 写入网络（3 秒写超时）
			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
			if _, err := conn.Write(buf[:n]); err != nil {
				return
			}
			// 推送桥接数据到前端终端
			pushBridgedData(ch, buf[:n], "SER→NET", true)
		}
	}()

	// 网络→串口
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 4096)
		for {
			select {
			case <-cancel:
				return
			default:
			}
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			if n > 0 {
				sp.Write(buf[:n])
				pushBridgedData(ch, buf[:n], "NET→SER", false)
			}
		}
	}()

	// 等待任一方向结束
	<-done
}

// pushBridgeEvent 推送桥接事件到前端（连接/断开等）。
func pushBridgeEvent(ch chan<- interface{}, evtType, msg string) {
	select {
	case ch <- map[string]interface{}{
		"type": "bridge", "event": evtType, "msg": msg, "timestamp": time.Now().UnixMilli(),
	}:
	default:
	}
}

// pushBridgedData 推送桥接数据到前端终端。
// direction 示例: "SER→NET"（串口到网络）, "NET→SER"（网络到串口）
func pushBridgedData(ch chan<- interface{}, data []byte, direction string, sent bool) {
	hexStr := fmt.Sprintf("%X", data)
	select {
	case ch <- map[string]interface{}{
		"type":      "data",
		"text":      fmt.Sprintf("[%s] %s", direction, string(data)),
		"hex":       hexStr,
		"bytes":     len(data),
		"timestamp": time.Now().UnixMilli(),
		"sent":      sent,
		"bridge":    true,
	}:
	default:
	}
}
