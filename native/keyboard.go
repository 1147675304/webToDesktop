// native/keyboard.go
// 跨平台按键注册中心 — 管理应用自定义快捷键，避免与系统快捷键冲突
//
// 架构：
//   - Go 层维护 shortcuts map（线程安全），供 bridge 查询/管理
//   - 每次 Go 注册表变更后，同步到 C 层静态数组（syncBlockedKeysToC）
//   - 平台钩子回调直接检查 C 层数组，不回调 Go（避免 CGO 导出运行时问题）
//   - 前端通过 bridge 的 registerShortcut/unregisterShortcut 管理

//go:build windows || linux

package native

import "sync"

var (
	mu          sync.RWMutex
	shortcuts   = make(map[string]bool)   // key: "Alt+Tab", value: true=消费
	keyMappings = make(map[string]string) // key: "Super_L", value: "Win"
)

// RegisterShortcut 注册一个应用快捷键。
// 当此组合键被按下时，由应用消费，不传递到系统。
// keyDesc 格式："Alt+Tab", "Ctrl+Shift+S", "Super_L", "Super_R"
// 修饰符顺序: Super > Ctrl > Alt > Shift, 如 "Ctrl+Alt+Delete"
func RegisterShortcut(keyDesc string) {
	mu.Lock()
	shortcuts[keyDesc] = true
	keys := copyKeysLocked()
	mu.Unlock()
	syncBlockedKeysToC(keys)
}

// UnregisterShortcut 注销一个应用快捷键。
func UnregisterShortcut(keyDesc string) {
	mu.Lock()
	delete(shortcuts, keyDesc)
	keys := copyKeysLocked()
	mu.Unlock()
	syncBlockedKeysToC(keys)
}

// ClearShortcuts 清空所有已注册的应用快捷键。
func ClearShortcuts() {
	mu.Lock()
	shortcuts = make(map[string]bool)
	mu.Unlock()
	syncBlockedKeysToC(nil)
}

// IsShortcutRegistered 查询快捷键是否已注册。
func IsShortcutRegistered(keyDesc string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return shortcuts[keyDesc]
}

// ListShortcuts 返回所有已注册的快捷键列表。
func ListShortcuts() []string {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]string, 0, len(shortcuts))
	for k := range shortcuts {
		result = append(result, k)
	}
	return result
}

// RegisterShortcuts 批量注册快捷键。
func RegisterShortcuts(keys []string) {
	mu.Lock()
	for _, k := range keys {
		shortcuts[k] = true
	}
	allKeys := copyKeysLocked()
	mu.Unlock()
	syncBlockedKeysToC(allKeys)
}

// InitDefaultBlockedShortcuts 初始化默认需要拦截的系统快捷键。
// 在键盘快捷键拦截启用时调用。
func InitDefaultBlockedShortcuts() {
	mu.Lock()
	defaults := []string{
		"Ctrl+S", // 禁止保存页面
	}
	for _, k := range defaults {
		shortcuts[k] = true
	}
	keys := copyKeysLocked()
	mu.Unlock()
	syncBlockedKeysToC(keys)
}

// ResetShortcuts 重置快捷键列表为默认值。
func ResetShortcuts() {
	mu.Lock()
	newMap := make(map[string]bool)
	defaults := []string{
		"Ctrl+S",
	}
	for _, k := range defaults {
		newMap[k] = true
	}
	shortcuts = newMap
	keys := copyKeysLocked()
	mu.Unlock()
	syncBlockedKeysToC(keys)
}

// ———— 按键映射管理 ————

// SetKeyMapping 设置按键映射。
// 当 mappedName 为空时，取消该按键的映射。
// 映射通过 RegisterShortcut/UnregisterShortcut 实现（加入组合快捷键拦截列表）。
// 注意：被映射的按键会以原始键名（如 "Super_L"）触发 keyboard-shortcut 事件，
// 前端需根据映射名自行转换。
func SetKeyMapping(keyName, mappedName string) {
	mu.Lock()
	if mappedName == "" {
		delete(keyMappings, keyName)
		mu.Unlock()
		UnregisterShortcut(keyName)
	} else {
		keyMappings[keyName] = mappedName
		mu.Unlock()
		RegisterShortcut(keyName)
	}
}

// SetKeyMappings 批量设置按键映射。
func SetKeyMappings(mappings map[string]string) {
	mu.Lock()
	var toReg, toUnreg []string
	for k, v := range mappings {
		if v == "" {
			delete(keyMappings, k)
			toUnreg = append(toUnreg, k)
		} else {
			keyMappings[k] = v
			toReg = append(toReg, k)
		}
	}
	mu.Unlock()
	for _, k := range toUnreg {
		UnregisterShortcut(k)
	}
	RegisterShortcuts(toReg)
}

// ClearKeyMappings 清空所有按键映射。
func ClearKeyMappings() {
	mu.Lock()
	keys := make([]string, 0, len(keyMappings))
	for k := range keyMappings {
		keys = append(keys, k)
	}
	keyMappings = make(map[string]string)
	mu.Unlock()
	for _, k := range keys {
		UnregisterShortcut(k)
	}
}

// ListKeyMappings 返回所有按键映射的副本。
func ListKeyMappings() map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]string, len(keyMappings))
	for k, v := range keyMappings {
		result[k] = v
	}
	return result
}

// GetKeyMapping 查询指定按键的映射名，未映射时返回空字符串。
func GetKeyMapping(keyName string) string {
	mu.RLock()
	defer mu.RUnlock()
	return keyMappings[keyName]
}

// InitKeyMappings 从配置初始化按键映射。
// 会覆盖当前所有映射。
func InitKeyMappings(mappings map[string]string) {
	if len(mappings) == 0 {
		return
	}
	mu.Lock()
	keyMappings = make(map[string]string, len(mappings))
	keys := make([]string, 0, len(mappings))
	for k, v := range mappings {
		if v != "" {
			keyMappings[k] = v
			keys = append(keys, k)
		}
	}
	mu.Unlock()
	RegisterShortcuts(keys)
}

// copyKeysLocked 在已锁定 mu 的前提下复制所有键。
func copyKeysLocked() []string {
	result := make([]string, 0, len(shortcuts))
	for k := range shortcuts {
		result = append(result, k)
	}
	return result
}

// syncBlockedKeysToC 在各平台文件中实现（windows.go / linux.go / other.go）。
// 将 Go 注册表中的快捷键同步到 C 层静态数组，keys 为 nil 时清空。
