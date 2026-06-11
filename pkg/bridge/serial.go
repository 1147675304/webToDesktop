// pkg/bridge/serial.go
// 串口双向数据流 — 扫描、打开、读写串口，实时推送数据到 JS
//
// JS 交互:
//
//	window.__lhpanda__('listSerialPorts')                              // 扫描可用串口
//	window.__lhpanda__('openSerialPort', {port, baudRate, ...})       // 打开并开始推送
//	window.__lhpanda__('sendSerialData', {data: "..."})               // 发送数据到串口
//	window.__lhpanda__('closeSerialPort')                              // 关闭串口
//	window.__lhpanda__('getSerialState')                               // 获取串口状态
//
// 前端监听 stream-data（topic: "serial-data"）接收串口数据。

//go:build !minimal && !nostream && !noserial

package bridge

import (
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
)

// ———— 串口控制结构 ————

type serialState struct {
	mu     sync.Mutex
	port   serial.Port // 打开的串口（nil 表示未打开）
	active bool        // 是否正在读取
	cancel chan struct{}
	ch     chan<- interface{} // 流通道（bridge 事件 / 串口数据推送用）
}

var currentSerial = &serialState{}

// ———— JS 可调用的 Bridge 方法 ————

// HandleListSerialPorts 扫描并返回可用串口列表。
//
// JS 调用: window.__lhpanda__('listSerialPorts')
// 返回: {ports: [{name: "/dev/ttyUSB0", vid: "...", pid: "..."}, ...]}
func (b *Bridge) HandleListSerialPorts(params map[string]interface{}) (interface{}, error) {
	names, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("扫描串口失败: %w", err)
	}

	type portInfo struct {
		Name string `json:"name"`
	}
	ports := make([]portInfo, 0, len(names))
	for _, n := range names {
		ports = append(ports, portInfo{Name: n})
	}
	return map[string]interface{}{
		"ports": ports,
	}, nil
}

// HandleOpenSerialPort 打开串口并启动数据流推送。
//
//	JS 调用: window.__lhpanda__('openSerialPort', {
//	    port: "/dev/ttyUSB0",    // 串口名称（必填）
//	    baudRate: 115200,        // 波特率（默认 115200）
//	    dataBits: 8,             // 数据位 5/6/7/8（默认 8）
//	    parity: "none",          // 校验: none/odd/even/mark/space（默认 none）
//	    stopBits: 1,             // 停止位: 1/1.5/2（默认 1）
//	})
//
// 返回: {opened: true, port: "/dev/ttyUSB0", baudRate: 115200, ...}
//
// 打开后串口数据通过 stream-data（topic: "serial-data"）实时推送到 JS。
func (b *Bridge) HandleOpenSerialPort(params map[string]interface{}) (interface{}, error) {
	// 先关闭旧端口
	closeSerialInternal("openSerialPort")

	portName := getString(params, "port")
	if portName == "" {
		return nil, fmt.Errorf("缺少 port 参数")
	}

	baudRate := getInt(params, "baudRate")
	if baudRate <= 0 {
		baudRate = 115200
	}
	dataBits := getInt(params, "dataBits")
	if dataBits == 0 {
		dataBits = 8
	}
	stopBitsVal := getInt(params, "stopBits")
	stopBits := serial.OneStopBit
	switch stopBitsVal {
	case 15, 2:
		stopBits = serial.TwoStopBits
		if stopBitsVal == 15 {
			stopBits = serial.OnePointFiveStopBits
		}
	}

	parityStr := getString(params, "parity")
	parity := serial.NoParity
	switch parityStr {
	case "odd":
		parity = serial.OddParity
	case "even":
		parity = serial.EvenParity
	case "mark":
		parity = serial.MarkParity
	case "space":
		parity = serial.SpaceParity
	}

	// 打开串口
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: dataBits,
		Parity:   parity,
		StopBits: stopBits,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("打开串口失败: %w", err)
	}

	// 设置无超时（持续读取）
	port.SetReadTimeout(serial.NoTimeout)

	// 更新状态
	currentSerial.mu.Lock()
	currentSerial.port = port
	currentSerial.active = true
	currentSerial.cancel = make(chan struct{})
	currentSerial.mu.Unlock()

	// 创建数据流并启动读取 goroutine
	ch := NewStream(b, "serial-data", 256)
	currentSerial.mu.Lock()
	currentSerial.ch = ch
	currentSerial.mu.Unlock()
	go readSerialLoop(ch, port, currentSerial.cancel)

	return map[string]interface{}{
		"opened":   true,
		"port":     portName,
		"baudRate": baudRate,
		"dataBits": dataBits,
		"parity":   parityStr,
		"stopBits": stopBitsVal,
	}, nil
}

// HandleSendSerialData 向已打开的串口发送数据。
//
// JS 调用: window.__lhpanda__('sendSerialData', {data: "hello\r\n"})
// 返回: {sent: true, bytes: 7}
//
// 注意：发送数据由前端本地即刻显示（sent: true 标记），
// Go 端不再回推到 stream channel，避免重复。
func (b *Bridge) HandleSendSerialData(params map[string]interface{}) (interface{}, error) {
	currentSerial.mu.Lock()
	port := currentSerial.port
	active := currentSerial.active
	currentSerial.mu.Unlock()

	if port == nil || !active {
		return nil, fmt.Errorf("串口未打开")
	}

	dataStr := getString(params, "data")
	if dataStr == "" {
		return nil, fmt.Errorf("缺少 data 参数")
	}

	n, err := port.Write([]byte(dataStr))
	if err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	return map[string]interface{}{
		"sent":  true,
		"bytes": n,
	}, nil
}

// HandleCloseSerialPort 关闭串口并停止数据推送。
//
// JS 调用: window.__lhpanda__('closeSerialPort')
// 返回: {closed: true}
func (b *Bridge) HandleCloseSerialPort(params map[string]interface{}) (interface{}, error) {
	closed := closeSerialInternal("closeSerialPort")
	return map[string]interface{}{"closed": closed}, nil
}

// HandleGetSerialState 获取当前串口状态。
//
// JS 调用: window.__lhpanda__('getSerialState')
// 返回: {opened: true/false, port: "...", baudRate: 115200}
func (b *Bridge) HandleGetSerialState(params map[string]interface{}) (interface{}, error) {
	currentSerial.mu.Lock()
	active := currentSerial.active
	currentSerial.mu.Unlock()

	if !active {
		return map[string]interface{}{"opened": false}, nil
	}
	return map[string]interface{}{"opened": true}, nil
}

// HandleSetDtr 设置 DTR（Data Terminal Ready）信号。
//
// JS 调用: window.__lhpanda__('setDtr', {on: true})
// 返回: {dtr: true}
func (b *Bridge) HandleSetDtr(params map[string]interface{}) (interface{}, error) {
	currentSerial.mu.Lock()
	port := currentSerial.port
	currentSerial.mu.Unlock()
	if port == nil {
		return nil, fmt.Errorf("串口未打开")
	}
	on := getBool(params, "on")
	if err := port.SetDTR(on); err != nil {
		return nil, fmt.Errorf("设置 DTR 失败: %w", err)
	}
	return map[string]interface{}{"dtr": on}, nil
}

// HandleSetRts 设置 RTS（Request To Send）信号。
//
// JS 调用: window.__lhpanda__('setRts', {on: true})
// 返回: {rts: true}
func (b *Bridge) HandleSetRts(params map[string]interface{}) (interface{}, error) {
	currentSerial.mu.Lock()
	port := currentSerial.port
	currentSerial.mu.Unlock()
	if port == nil {
		return nil, fmt.Errorf("串口未打开")
	}
	on := getBool(params, "on")
	if err := port.SetRTS(on); err != nil {
		return nil, fmt.Errorf("设置 RTS 失败: %w", err)
	}
	return map[string]interface{}{"rts": on}, nil
}

// HandleGetModemStatus 获取 Modem 状态位（CTS/DSR/DCD/RI）。
//
// JS 调用: window.__lhpanda__('getModemStatus')
// 返回: {cts: true, dsr: false, dcd: true, ri: false}
func (b *Bridge) HandleGetModemStatus(params map[string]interface{}) (interface{}, error) {
	currentSerial.mu.Lock()
	port := currentSerial.port
	currentSerial.mu.Unlock()
	if port == nil {
		return nil, fmt.Errorf("串口未打开")
	}
	status, err := port.GetModemStatusBits()
	if err != nil {
		return nil, fmt.Errorf("获取状态位失败: %w", err)
	}
	return map[string]interface{}{
		"cts": status.CTS,
		"dsr": status.DSR,
		"dcd": status.DCD,
		"ri":  status.RI,
	}, nil
}

// ———— 串口读取循环 ————

// readSerialLoop 持续从串口读取数据并推送到 JS。
//
// 在独立 goroutine 中运行，通过 cancel channel 控制退出。
// 读取到的数据作为 byte slice 推送到 stream channel，消费者会自动通过 pushToJS 转发。
func readSerialLoop(ch chan<- interface{}, port serial.Port, cancel chan struct{}) {
	defer close(ch)
	defer func() {
		StopStream("serial-data")
		currentSerial.mu.Lock()
		if currentSerial.port == port {
			port.Close()
			currentSerial.port = nil
			currentSerial.active = false
		}
		currentSerial.mu.Unlock()
	}()

	buf := make([]byte, 256)
	for {
		select {
		case <-cancel:
			return
		default:
		}

		// 设置短超时以便响应取消信号
		port.SetReadTimeout(100 * time.Millisecond)
		n, err := port.Read(buf)
		if err != nil {
			// 检查是否被取消
			select {
			case <-cancel:
				return
			default:
			}
			// 其他读取错误，推送错误消息后继续
			select {
			case ch <- map[string]interface{}{
				"type": "error",
				"msg":  fmt.Sprintf("读取错误: %v", err),
			}:
			case <-cancel:
				return
			}
			continue
		}

		if n > 0 {
			// 推送接收到的数据（作为文本 + 十六进制）
			hexStr := fmt.Sprintf("%X", buf[:n])
			select {
			case ch <- map[string]interface{}{
				"type":      "data",
				"text":      string(buf[:n]),
				"hex":       hexStr,
				"bytes":     n,
				"timestamp": time.Now().UnixMilli(),
			}:
			case <-cancel:
				return
			}
		}
	}
}

// ———— 内部辅助 ————

func closeSerialInternal(reason string) bool {
	currentSerial.mu.Lock()
	defer currentSerial.mu.Unlock()

	if currentSerial.port == nil {
		return false
	}

	// 通知读取循环退出
	if currentSerial.cancel != nil {
		close(currentSerial.cancel)
		currentSerial.cancel = nil
	}

	currentSerial.port.Close()
	currentSerial.port = nil
	currentSerial.active = false
	currentSerial.ch = nil
	StopStream("serial-data")
	return true
}
