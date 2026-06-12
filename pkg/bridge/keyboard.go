// tools/desktop/pkg/bridge/keyboard.go
// Go ↔ JS 桥接 — 快捷键注册与管理
//
// 前端可以动态注册/注销快捷键，避免与系统快捷键冲突。
// 所有按键拦截在 native/keyboard.go 的注册中心中管理。
//
// JS 调用示例:
//
//	// 注册快捷键：这些组合键将被应用消费，不传递到系统
//	window.__lhpanda__('registerShortcut', {keys: ['Ctrl+S', 'Ctrl+Shift+F']})
//
//	// 注销快捷键
//	window.__lhpanda__('unregisterShortcut', {keys: ['Ctrl+S']})
//
//	// 查询已注册的快捷键
//	window.__lhpanda__('listShortcuts')
//
//	// 重置为默认值（仅保留系统拦截）
//	window.__lhpanda__('resetShortcuts')
package bridge

import (
	"github.com/lhpanda/webtodesktop/native"
)

// HandleRegisterShortcut 注册一个或多个应用自定义快捷键。
// 注册后该组合键将被应用消费，不会传递到系统。
//
// JS 调用: window.__lhpanda__('registerShortcut', {keys: ['Ctrl+S', 'Alt+Tab']})
// 参数:
//   - keys: string | string[] — 快捷键描述，支持单字符串或数组
//
// 返回: {ok: true, registered: ["Ctrl+S", "Alt+Tab"]}
//
// 快捷键格式: 修饰符+键名，修饰符顺序: Super > Ctrl > Alt > Shift
// 示例: "Ctrl+S", "Alt+Tab", "Ctrl+Shift+F", "Super_L", "Super_R"
func (b *Bridge) HandleRegisterShortcut(params map[string]interface{}) (interface{}, error) {
	keys := extractKeys(params)
	if len(keys) == 0 {
		return map[string]interface{}{"ok": false, "error": "未提供 keys 参数"}, nil
	}
	native.RegisterShortcuts(keys)
	return map[string]interface{}{"ok": true, "registered": keys}, nil
}

// HandleUnregisterShortcut 注销一个或多个应用快捷键。
// 注销后该组合键将恢复为系统默认行为。
//
// JS 调用: window.__lhpanda__('unregisterShortcut', {keys: ['Ctrl+S']})
// 参数:
//   - keys: string | string[] — 要注销的快捷键
//
// 返回: {ok: true}
func (b *Bridge) HandleUnregisterShortcut(params map[string]interface{}) (interface{}, error) {
	keys := extractKeys(params)
	if len(keys) == 0 {
		return map[string]interface{}{"ok": false, "error": "未提供 keys 参数"}, nil
	}
	for _, k := range keys {
		native.UnregisterShortcut(k)
	}
	return map[string]interface{}{"ok": true}, nil
}

// HandleListShortcuts 列出所有当前已注册的快捷键。
//
// JS 调用: window.__lhpanda__('listShortcuts')
// 返回: {ok: true, shortcuts: ["Alt+Tab", "Ctrl+S", ...]}
func (b *Bridge) HandleListShortcuts(params map[string]interface{}) (interface{}, error) {
	list := native.ListShortcuts()
	return map[string]interface{}{"ok": true, "shortcuts": list}, nil
}

// HandleResetShortcuts 重置快捷键列表为默认值（仅保留系统快捷键拦截）。
//
// JS 调用: window.__lhpanda__('resetShortcuts')
// 返回: {ok: true}
func (b *Bridge) HandleResetShortcuts(params map[string]interface{}) (interface{}, error) {
	native.ResetShortcuts()
	return map[string]interface{}{"ok": true}, nil
}

// HandleClearShortcuts 清空所有已注册的快捷键（包括默认拦截）。
//
// JS 调用: window.__lhpanda__('clearShortcuts')
// 返回: {ok: true}
func (b *Bridge) HandleClearShortcuts(params map[string]interface{}) (interface{}, error) {
	native.ClearShortcuts()
	return map[string]interface{}{"ok": true}, nil
}

// HandleSetKeyboardEnabled 启用或禁用键盘快捷键拦截。
// 启用时恢复默认快捷键 + 已注册的快捷键；禁用时清空所有。
//
// JS 调用: window.__lhpanda__('setKeyboardEnabled', {enabled: true})
// 返回: {ok: true, enabled: true}
func (b *Bridge) HandleSetKeyboardEnabled(params map[string]interface{}) (interface{}, error) {
	enabled, _ := params["enabled"].(bool)
	if enabled {
		native.ResetShortcuts()
	} else {
		native.ClearShortcuts()
	}
	return map[string]interface{}{"ok": true, "enabled": enabled}, nil
}

// HandleSetKeyMapping 设置按键映射。
// 映射后的按键被钩子拦截，不会传递到系统，同时以映射名触发 keyboard-shortcut 事件。
//
// JS 调用: window.__lhpanda__('setKeyMapping', {key: "Super_L", mappedName: "Win"})
// 或批量: window.__lhpanda__('setKeyMapping', {mappings: {"Super_L": "Win", "Alt_L": "Alt"}})
// 取消映射: window.__lhpanda__('setKeyMapping', {key: "Super_L", mappedName: ""})
// 返回: {ok: true}
func (b *Bridge) HandleSetKeyMapping(params map[string]interface{}) (interface{}, error) {
	if params == nil {
		return map[string]interface{}{"ok": false, "error": "参数为空"}, nil
	}
	// 批量设置
	if mappingsRaw, ok := params["mappings"]; ok {
		if mappings, ok := mappingsRaw.(map[string]interface{}); ok {
			strMap := make(map[string]string, len(mappings))
			for k, v := range mappings {
				if s, ok := v.(string); ok {
					strMap[k] = s
				}
			}
			native.SetKeyMappings(strMap)
			return map[string]interface{}{"ok": true}, nil
		}
	}
	// 单个设置
	key, _ := params["key"].(string)
	if key == "" {
		return map[string]interface{}{"ok": false, "error": "缺少 key 参数"}, nil
	}
	mappedName, _ := params["mappedName"].(string)
	native.SetKeyMapping(key, mappedName)
	return map[string]interface{}{"ok": true}, nil
}

// HandleListKeyMappings 列出所有按键映射。
//
// JS 调用: window.__lhpanda__('listKeyMappings')
// 返回: {ok: true, mappings: {"Super_L": "Win", "Alt_L": "Alt", ...}}
func (b *Bridge) HandleListKeyMappings(params map[string]interface{}) (interface{}, error) {
	mappings := native.ListKeyMappings()
	return map[string]interface{}{"ok": true, "mappings": mappings}, nil
}

// HandleClearKeyMappings 清空所有按键映射。
//
// JS 调用: window.__lhpanda__('clearKeyMappings')
// 返回: {ok: true}
func (b *Bridge) HandleClearKeyMappings(params map[string]interface{}) (interface{}, error) {
	native.ClearKeyMappings()
	return map[string]interface{}{"ok": true}, nil
}

// extractKeys 从 params 中提取 keys 参数，支持字符串或字符串数组。
func extractKeys(params map[string]interface{}) []string {
	if params == nil {
		return nil
	}
	raw, ok := params["keys"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
}
