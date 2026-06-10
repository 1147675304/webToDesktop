// tools/desktop/pkg/bridge/storage.go
// Go ↔ JS 桥接 — 通用键值对持久化存储
//
// 提供类似浏览器 localStorage 的键值对存储能力，数据经 AES-256-GCM 加密后持久化到本地磁盘。
// 用于替代浏览器 localStorage/sessionStorage，实现桌面端的持久化数据存储。
//
// 安全设计:
//   - 所有数据使用 AES-256-GCM 加密存储（与凭证存储共用加密密钥）
//   - 存储文件: ~/.config/<app-name>/credentials.dat（与凭证共用文件）
//
// JS 调用示例:
//
//	// 存储
//	window.__lhpanda__('setItem', {key: 'theme', value: 'dark'})
//	// 读取
//	window.__lhpanda__('getItem', {key: 'theme'})
//	// 删除
//	window.__lhpanda__('removeItem', {key: 'theme'})
//	// 获取所有
//	window.__lhpanda__('getAllItems')
//	// 清除全部
//	window.__lhpanda__('clearItems')
package bridge

import (
	"fmt"
)

// handleSetItem 加密保存一个键值对。
//
// JS 调用: window.__lhpanda__('setItem', {key: "theme", value: "dark"})
// 返回:    {saved: true}
//
// 如果 key 已存在则更新 value，不存在则新增。
// 数据经 AES-256-GCM 加密后写入磁盘文件。
func (b *Bridge) HandleSetItem(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	key, _ := params["key"].(string)
	value, _ := params["value"].(string)
	if key == "" {
		return nil, fmt.Errorf("key 不能为空")
	}

	if err := b.store.SetItem(key, value); err != nil {
		return nil, fmt.Errorf("保存键值对失败: %w", err)
	}
	return map[string]interface{}{"saved": true}, nil
}

// handleGetItem 获取指定键的值。
//
// JS 调用: window.__lhpanda__('getItem', {key: "theme"})
// 返回:    {found: true, value: "dark"}  或  {found: false}
//
// 如果 key 不存在返回 {found: false}。
func (b *Bridge) HandleGetItem(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	key, _ := params["key"].(string)
	if key == "" {
		return nil, fmt.Errorf("key 不能为空")
	}

	value, err := b.store.GetItem(key)
	if err != nil {
		return nil, fmt.Errorf("读取键值对失败: %w", err)
	}
	if value == "" {
		return map[string]interface{}{"found": false}, nil
	}
	return map[string]interface{}{"found": true, "value": value}, nil
}

// handleRemoveItem 删除指定键的键值对。
//
// JS 调用: window.__lhpanda__('removeItem', {key: "theme"})
// 返回:    {removed: true}
//
// key 不存在时不报错，直接返回成功。
func (b *Bridge) HandleRemoveItem(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	key, _ := params["key"].(string)
	if key == "" {
		return nil, fmt.Errorf("key 不能为空")
	}

	if err := b.store.RemoveItem(key); err != nil {
		return nil, fmt.Errorf("删除键值对失败: %w", err)
	}
	return map[string]interface{}{"removed": true}, nil
}

// handleClearItems 清除所有键值对。
//
// JS 调用: window.__lhpanda__('clearItems')
// 返回:    {cleared: true}
//
// 清空 KeyValues 映射后写回加密文件。
func (b *Bridge) HandleClearItems(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	if err := b.store.ClearItems(); err != nil {
		return nil, fmt.Errorf("清除键值对失败: %w", err)
	}
	return map[string]interface{}{"cleared": true}, nil
}

// handleGetAllItems 获取所有键值对。
//
// JS 调用: window.__lhpanda__('getAllItems')
// 返回:    {items: {key1: "value1", key2: "value2"}}
//
// 返回完整的键值对映射，前端可据此重建本地状态。
func (b *Bridge) HandleGetAllItems(params map[string]interface{}) (interface{}, error) {
	if b.store == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	items, err := b.store.GetAllItems()
	if err != nil {
		return nil, fmt.Errorf("读取所有键值对失败: %w", err)
	}
	return map[string]interface{}{"items": items}, nil
}
