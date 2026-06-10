// pkg/bridge/streamdemo.go
// Go → JS 流式推送演示：逐行读取 README.md 推送前端
//
// JS 交互式控制:
//
//	window.__lhpanda__('startReadmeStream', {startLine: 10, speed: 500})
//	window.__lhpanda__('pauseReadmeStream')
//	window.__lhpanda__('resumeReadmeStream')
//	window.__lhpanda__('seekReadmeStream', {line: 5})
//	window.__lhpanda__('getReadmeState')
//	window.__lhpanda__('stopReadmeStream')
//
// 前端监听 stream-data 事件获取每行数据。
package bridge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// readmeCtrl 控制 README 流的运行状态。
// handler 方法通过修改此结构体控制 goroutine 行为。
type readmeCtrl struct {
	mu       sync.Mutex
	paused   bool          // 是否暂停
	seekLine int           // 跳转到指定行（>0 时触发跳转）
	cancel   chan struct{} // 停止信号
}

// globalReadmeCtrl 当前活跃的 README 流控制（同一时间只有一个）。
var globalReadmeCtrl *readmeCtrl
var globalReadmeMu sync.Mutex

func (c *readmeCtrl) isPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func (c *readmeCtrl) setPaused(v bool) {
	c.mu.Lock()
	c.paused = v
	c.mu.Unlock()
}

func (c *readmeCtrl) getAndResetSeek() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.seekLine
	c.seekLine = 0
	return v
}

func (c *readmeCtrl) setSeek(line int) {
	c.mu.Lock()
	c.seekLine = line
	c.mu.Unlock()
}

// ———— JS 可调用的 Bridge 方法 ————

// HandleStartReadmeStream 启动 README.md 逐行推送。
//
// JS 调用: window.__lhpanda__('startReadmeStream', {startLine: 3, speed: 800})
//
// 参数:
//
//	startLine  int  从第几行开始推送（默认 1）
//	speed      int  行间间隔毫秒数（默认 1000，最小 50）
//
// 返回: {listening: true, topic: "readme-stream", totalLines: 42, startLine: 3}
func (b *Bridge) HandleStartReadmeStream(params map[string]interface{}) (interface{}, error) {
	topic := "readme-stream"

	// 先停止旧流
	stopReadmeInternal()

	// 解析参数
	startLine := 1
	if v, ok := params["startLine"].(float64); ok && v >= 1 {
		startLine = int(v)
	}
	speed := 1000 // 默认每秒一行
	if v, ok := params["speed"].(float64); ok && v >= 50 {
		speed = int(v)
	}

	// 查找 README.md
	readmePath := findReadme()
	if readmePath == "" {
		return nil, fmt.Errorf("未找到 README.md 文件")
	}

	totalLines := countLines(readmePath)
	if startLine > totalLines {
		return nil, fmt.Errorf("起始行 %d 超出总行数 %d", startLine, totalLines)
	}

	// 创建控制结构
	ctrl := &readmeCtrl{
		cancel: make(chan struct{}),
	}
	globalReadmeMu.Lock()
	globalReadmeCtrl = ctrl
	globalReadmeMu.Unlock()

	// 创建数据流
	ch := NewStream(b, topic, 8)

	// 启动逐行推送 goroutine
	go runReadmeStream(ch, ctrl, readmePath, startLine, totalLines, speed)

	return map[string]interface{}{
		"listening":  true,
		"topic":      topic,
		"totalLines": totalLines,
		"startLine":  startLine,
		"speed":      speed,
		"file":       filepath.Base(readmePath),
	}, nil
}

// HandlePauseReadmeStream 暂停 README 逐行推送。
//
// JS 调用: window.__lhpanda__('pauseReadmeStream')
// 返回: {paused: true, topic: "readme-stream"}
func (b *Bridge) HandlePauseReadmeStream(params map[string]interface{}) (interface{}, error) {
	ctrl := getReadmeCtrl()
	if ctrl == nil {
		return nil, fmt.Errorf("没有正在运行的 README 流")
	}
	ctrl.setPaused(true)
	return map[string]interface{}{
		"paused": true,
		"topic":  "readme-stream",
	}, nil
}

// HandleResumeReadmeStream 恢复 README 逐行推送。
//
// JS 调用: window.__lhpanda__('resumeReadmeStream')
// 返回: {resumed: true, topic: "readme-stream"}
func (b *Bridge) HandleResumeReadmeStream(params map[string]interface{}) (interface{}, error) {
	ctrl := getReadmeCtrl()
	if ctrl == nil {
		return nil, fmt.Errorf("没有正在运行的 README 流")
	}
	ctrl.setPaused(false)
	return map[string]interface{}{
		"resumed": true,
		"topic":   "readme-stream",
	}, nil
}

// HandleSeekReadmeStream 跳转到指定行并从该行继续推送。
//
// JS 调用: window.__lhpanda__('seekReadmeStream', {line: 10})
// 返回: {seek: true, line: 10, topic: "readme-stream"}
//
// 注意: 跳转会重新打开文件，在流中插入一条 type: "seek" 的消息。
func (b *Bridge) HandleSeekReadmeStream(params map[string]interface{}) (interface{}, error) {
	ctrl := getReadmeCtrl()
	if ctrl == nil {
		return nil, fmt.Errorf("没有正在运行的 README 流")
	}
	line, ok := params["line"].(float64)
	if !ok || line < 1 {
		return nil, fmt.Errorf("需要有效的 line 参数（>=1）")
	}
	ctrl.setSeek(int(line))
	return map[string]interface{}{
		"seek":  true,
		"line":  int(line),
		"topic": "readme-stream",
	}, nil
}

// HandleGetReadmeState 获取当前推送状态。
//
// JS 调用: window.__lhpanda__('getReadmeState')
// 返回: {hasStream: true, paused: false}
func (b *Bridge) HandleGetReadmeState(params map[string]interface{}) (interface{}, error) {
	ctrl := getReadmeCtrl()
	if ctrl == nil {
		return map[string]interface{}{"hasStream": false}, nil
	}
	return map[string]interface{}{
		"hasStream": true,
		"paused":    ctrl.isPaused(),
		"topic":     "readme-stream",
	}, nil
}

// HandleStopReadmeStream 停止 README 逐行推送。
//
// JS 调用: window.__lhpanda__('stopReadmeStream')
// 返回: {stopped: true, topic: "readme-stream"}
func (b *Bridge) HandleStopReadmeStream(params map[string]interface{}) (interface{}, error) {
	stopped := stopReadmeInternal()
	return map[string]interface{}{
		"stopped": stopped,
		"topic":   "readme-stream",
	}, nil
}

// ———— 核心推送逻辑 ————

// runReadmeStream 逐行读取文件并通过 channel 推送到 JS。
//
// 参数:
//   - ch: 数据通道，写入的数据会被 pushToJS 转发
//   - ctrl: 运行时控制（暂停/跳转/取消）
//   - path: README.md 文件路径
//   - startLine: 起始行号
//   - totalLines: 总行数
//   - speedMs: 行间间隔毫秒数
func runReadmeStream(ch chan<- interface{}, ctrl *readmeCtrl, path string, startLine, totalLines, speedMs int) {
	defer close(ch)
	defer func() {
		globalReadmeMu.Lock()
		if globalReadmeCtrl == ctrl {
			globalReadmeCtrl = nil
		}
		globalReadmeMu.Unlock()
	}()

	file, err := os.Open(path)
	if err != nil {
		ch <- map[string]interface{}{
			"type": "error", "msg": fmt.Sprintf("无法打开文件: %v", err),
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0

	for scanner.Scan() {
		lineNo++

		// 跳过 startLine 之前的行
		if lineNo < startLine {
			continue
		}

		// —— 处理暂停 ——
		for ctrl.isPaused() {
			select {
			case <-ctrl.cancel:
				return
			case <-time.After(200 * time.Millisecond):
			}
		}

		// —— 处理跳转请求 ——
		if seekLine := ctrl.getAndResetSeek(); seekLine > 0 {
			if seekLine > totalLines {
				seekLine = totalLines
			}
			// 重新打开文件跳到指定行
			file.Close()
			newFile, err := os.Open(path)
			if err != nil {
				ch <- map[string]interface{}{"type": "error", "msg": "跳转失败: 无法重新打开文件"}
				return
			}
			file = newFile
			scanner = bufio.NewScanner(file)
			lineNo = 0
			for scanner.Scan() {
				lineNo++
				if lineNo >= seekLine {
					break
				}
			}
			// 发送跳转通知到前端
			ch <- map[string]interface{}{
				"type": "seek", "line": seekLine,
				"msg": fmt.Sprintf("已跳转到第 %d 行", seekLine),
			}
			continue
		}

		// —— 推送数据行 ——
		select {
		case ch <- map[string]interface{}{
			"type":      "line",
			"line":      lineNo,
			"total":     totalLines,
			"text":      scanner.Text(),
			"timestamp": time.Now().UnixMilli(),
		}:
			// 等待间隔，可被取消打断
			select {
			case <-time.After(time.Duration(speedMs) * time.Millisecond):
			case <-ctrl.cancel:
				return
			}
		case <-ctrl.cancel:
			return
		}
	}

	// 文件读取完毕
	if err := scanner.Err(); err != nil {
		ch <- map[string]interface{}{"type": "error", "msg": fmt.Sprintf("读取错误: %v", err)}
	} else {
		ch <- map[string]interface{}{"type": "done", "line": totalLines, "msg": "README.md 推送完毕"}
	}
}

// ———— 内部辅助 ————

func getReadmeCtrl() *readmeCtrl {
	globalReadmeMu.Lock()
	defer globalReadmeMu.Unlock()
	return globalReadmeCtrl
}

func stopReadmeInternal() bool {
	StopStream("readme-stream")
	ctrl := getReadmeCtrl()
	if ctrl != nil {
		close(ctrl.cancel)
		globalReadmeMu.Lock()
		globalReadmeCtrl = nil
		globalReadmeMu.Unlock()
		return true
	}
	return false
}

// findReadme 在工作目录或可执行文件所在目录查找 README.md。
func findReadme() string {
	candidates := []string{
		"README.md",
		"../README.md",
		"../../README.md",
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "README.md"),
			filepath.Join(dir, "..", "README.md"),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// countLines 统计文件行数。
func countLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}
