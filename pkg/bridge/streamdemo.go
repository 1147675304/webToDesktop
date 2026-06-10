// pkg/bridge/streamdemo.go
// Go → JS 流式推送演示：逐行读取 README.md 每秒推送一行到前端
//
// JS 调用:
//
//	window.__lhpanda__('startReadmeStream')   // 开始逐行推送
//	window.__lhpanda__('stopReadmeStream')    // 提前停止
//
// 前端监听 stream-data 事件即可逐行展示。
package bridge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HandleStartReadmeStream 启动 README.md 逐行推送。
//
// 行为:
//  1. 查找工作目录下的 README.md
//  2. 创建 "readme-stream" 数据流
//  3. 启动 goroutine，每秒推送一行到前端
//  4. 文件结束时发送 stream-end 事件
//
// JS 调用: window.__lhpanda__('startReadmeStream')
// 返回: {listening: true, topic: "readme-stream", totalLines: 42}
func (b *Bridge) HandleStartReadmeStream(params map[string]interface{}) (interface{}, error) {
	topic := "readme-stream"

	// 先停止旧流（如果存在）
	StopStream(topic)

	// 查找 README.md
	readmePath := findReadme()
	if readmePath == "" {
		return nil, fmt.Errorf("未找到 README.md 文件")
	}

	// 预扫描总行数（用于前端进度条）
	totalLines := countLines(readmePath)

	// 创建数据流
	ch := NewStream(b, topic, 8)

	// 逐行推送 goroutine
	go func() {
		defer close(ch)

		file, err := os.Open(readmePath)
		if err != nil {
			ch <- map[string]interface{}{
				"type": "error",
				"msg":  fmt.Sprintf("无法打开文件: %v", err),
			}
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			select {
			case ch <- map[string]interface{}{
				"type":      "line",
				"line":      lineNo,
				"total":     totalLines,
				"text":      scanner.Text(),
				"timestamp": time.Now().UnixMilli(),
			}:
				// 每秒推送一行
				time.Sleep(1 * time.Second)
			case <-time.After(30 * time.Second):
				// 超时保护：30 秒无消费者则退出
				return
			}
		}

		// 文件读取完毕，推送结束标记
		if err := scanner.Err(); err != nil {
			ch <- map[string]interface{}{
				"type": "error",
				"msg":  fmt.Sprintf("读取错误: %v", err),
			}
		} else {
			ch <- map[string]interface{}{
				"type": "done",
				"line": totalLines,
				"msg":  "README.md 推送完毕",
			}
		}
	}()

	return map[string]interface{}{
		"listening":  true,
		"topic":      topic,
		"totalLines": totalLines,
		"file":       filepath.Base(readmePath),
	}, nil
}

// HandleStopReadmeStream 提前停止 README.md 逐行推送。
//
// JS 调用: window.__lhpanda__('stopReadmeStream')
// 返回: {stopped: true, topic: "readme-stream"}
func (b *Bridge) HandleStopReadmeStream(params map[string]interface{}) (interface{}, error) {
	stopped := StopStream("readme-stream")
	return map[string]interface{}{
		"stopped": stopped,
		"topic":   "readme-stream",
	}, nil
}

// ———— 辅助函数 ————

// findReadme 在工作目录或可执行文件所在目录查找 README.md。
func findReadme() string {
	// 候选路径
	candidates := []string{
		"README.md",
		"../README.md",
		"../../README.md",
	}

	// 同时查找可执行文件同目录
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
