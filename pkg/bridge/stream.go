// pkg/bridge/stream.go
// Go → JS 流式数据推送
//
// 通过 Go channel + wv.Eval() 实现 Go 端主动向 JS 推送实时数据。
// 前端通过监听 window 上的 CustomEvent 接收数据，无需轮询。
//
// 架构:
//
//	Go goroutine  ──channel──▶  消费者 goroutine  ──wv.Eval()──▶  JS CustomEvent
//
// JS 调用示例:
//
//	// 启动监听
//	window.__lhpanda__('listenStream', {topic: 'logs'})
//	// 监听事件
//	window.addEventListener('stream-data', (e) => console.log(e.detail))
//	// 停止监听
//	window.__lhpanda__('stopStream', {topic: 'logs'})
//
// 安全性:
//   - 所有 JS 评估通过 wv.Dispatch 在 UI 线程执行
//   - JSON 序列化处理特殊字符，防止注入

//go:build !minimal && !nostream

package bridge

import (
	"encoding/json"
	"fmt"
	"sync"
)

// streamEntry 表示一个活跃的数据流。
type streamEntry struct {
	ch     chan interface{} // 数据通道
	cancel chan struct{}    // 取消信号
}

// streamRegistry 管理所有活跃的数据流。
// key 为 topic 名称，由 JS 调用时指定。
var streamRegistry = struct {
	sync.Mutex
	streams map[string]*streamEntry
}{streams: make(map[string]*streamEntry)}

// ———— 公开 API：业务方可调用来创建数据源 ————

// NewStream 创建一个具名数据通道并返回发送端。
//
// 业务方调用此方法获取 channel，向其中发送数据即可自动推送到 JS。
// 当 JS 调用 stopStream 或通道关闭时，关联的 goroutine 自动清理。
//
// 示例:
//
//	ch := bridge.NewStream(b, "sensor-data")
//	go func() {
//	    for _, val := range sensor.Readings() {
//	        ch <- val
//	    }
//	    close(ch)
//	}()
func NewStream(b *Bridge, topic string, bufferSize int) chan<- interface{} {
	entry := &streamEntry{
		ch:     make(chan interface{}, bufferSize),
		cancel: make(chan struct{}),
	}

	streamRegistry.Lock()
	streamRegistry.streams[topic] = entry
	streamRegistry.Unlock()

	// 消费者 goroutine：读取 channel → 推送到 JS
	go func() {
		defer func() {
			streamRegistry.Lock()
			delete(streamRegistry.streams, topic)
			streamRegistry.Unlock()
		}()
		for {
			select {
			case data, ok := <-entry.ch:
				if !ok {
					// channel 已关闭，发送结束事件
					b.dispatchEvent("stream-end", map[string]interface{}{"topic": topic})
					return
				}
				b.pushToJS(topic, data)
			case <-entry.cancel:
				return
			}
		}
	}()

	return entry.ch
}

// StopStream 停止一个活跃的数据流。
//
// 业务方也可调用来主动终止流。
func StopStream(topic string) bool {
	streamRegistry.Lock()
	entry, ok := streamRegistry.streams[topic]
	if ok {
		close(entry.cancel)
		delete(streamRegistry.streams, topic)
	}
	streamRegistry.Unlock()
	return ok
}

// ———— Bridge 方法（自动注册为 JS 可调用） ————

// HandleListenStream 启动一个流式数据监听。
//
// JS 调用: window.__lhpanda__('listenStream', {topic: 'my-topic'})
//
// 该方法的典型使用场景是由 JS 发起监听请求，
// Go 端已有或新建数据源，数据通过 CustomEvent 推送到 JS。
//
// 注意：此方法本身不创建数据源；数据源需由业务代码通过 NewStream 预先创建，
// 或在此 handler 中根据 topic 动态创建。
//
// 返回: {listening: true, topic: 'my-topic'}
func (b *Bridge) HandleListenStream(params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("缺少 topic 参数")
	}

	// 如果尚未注册，创建一个空流（业务方可后续通过 NewStream 获取 channel）
	streamRegistry.Lock()
	_, exists := streamRegistry.streams[topic]
	if !exists {
		entry := &streamEntry{
			ch:     make(chan interface{}, 64),
			cancel: make(chan struct{}),
		}
		streamRegistry.streams[topic] = entry

		// 启动消费者 goroutine
		go func() {
			defer func() {
				streamRegistry.Lock()
				delete(streamRegistry.streams, topic)
				streamRegistry.Unlock()
			}()
			for {
				select {
				case data, ok := <-entry.ch:
					if !ok {
						b.dispatchEvent("stream-end", map[string]interface{}{"topic": topic})
						return
					}
					b.pushToJS(topic, data)
				case <-entry.cancel:
					return
				}
			}
		}()
	}
	streamRegistry.Unlock()

	return map[string]interface{}{
		"listening": true,
		"topic":     topic,
	}, nil
}

// HandleStopStream 停止一个流式数据推送。
//
// JS 调用: window.__lhpanda__('stopStream', {topic: 'my-topic'})
// 返回: {stopped: true, topic: 'my-topic'}
func (b *Bridge) HandleStopStream(params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("缺少 topic 参数")
	}

	stopped := StopStream(topic)
	return map[string]interface{}{
		"stopped": stopped,
		"topic":   topic,
	}, nil
}

// HandleSendToStream 向指定数据流发送一条数据（用于 JS 侧向流写入数据）。
//
// JS 调用: window.__lhpanda__('sendToStream', {topic: 'my-topic', data: {...}})
// 返回: {sent: true, topic: 'my-topic'}
func (b *Bridge) HandleSendToStream(params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("缺少 topic 参数")
	}

	data, hasData := params["data"]
	if !hasData {
		return nil, fmt.Errorf("缺少 data 参数")
	}

	streamRegistry.Lock()
	entry, ok := streamRegistry.streams[topic]
	streamRegistry.Unlock()

	if !ok {
		return nil, fmt.Errorf("流不存在: %s", topic)
	}

	select {
	case entry.ch <- data:
		return map[string]interface{}{"sent": true, "topic": topic}, nil
	default:
		return nil, fmt.Errorf("流已满: %s", topic)
	}
}

// HandleListStreams 列出所有活跃的数据流。
//
// JS 调用: window.__lhpanda__('listStreams')
// 返回: {streams: ['topic1', 'topic2']}
func (b *Bridge) HandleListStreams(params map[string]interface{}) (interface{}, error) {
	streamRegistry.Lock()
	defer streamRegistry.Unlock()

	topics := make([]string, 0, len(streamRegistry.streams))
	for topic := range streamRegistry.streams {
		topics = append(topics, topic)
	}
	return map[string]interface{}{"streams": topics}, nil
}

// ———— 内部方法 ————

// pushToJS 将数据通过 CustomEvent 推送到 JS 端。
//
// 在 UI 线程执行 wv.Eval()，以 window.dispatchEvent(new CustomEvent(...)) 形式传递数据。
// 数据经 JSON 序列化，特殊字符安全转义。
func (b *Bridge) pushToJS(topic string, data interface{}) {
	if b.wv == nil {
		return
	}

	payload := map[string]interface{}{
		"topic": topic,
		"data":  data,
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// 构建安全的 JS 调用字符串
	js := fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent('stream-data',{detail:%s}))`,
		string(jsonBytes),
	)

	b.wv.Dispatch(func() {
		b.wv.Eval(js)
	})
}

// dispatchEvent 向 JS 端发送一个具名事件（不含 topic 数据）。
func (b *Bridge) dispatchEvent(eventName string, detail map[string]interface{}) {
	if b.wv == nil {
		return
	}
	jsonBytes, err := json.Marshal(detail)
	if err != nil {
		return
	}
	js := fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent(%q,{detail:%s}))`,
		eventName, string(jsonBytes),
	)
	b.wv.Dispatch(func() {
		b.wv.Eval(js)
	})
}
