// tools/desktop/pkg/bridge/credentials.go
// Go ↔ JS 桥接 — 用户凭证加密管理
//
// 安全设计原则:
//   - 密码使用 AES-256-GCM 加密存储于本地文件 (credentials.dat)
//   - 密码永不出现在前端 — getCredentials 仅返回用户名列表，不含密码
//   - 代理层通过 X-Credential-Username 请求头识别用户，自动替换 __DESKTOP_PWD__ 占位符
//   - 加密密钥由 config.yaml 的 security.aes_key 经 SHA256 派生
//
// 存储位置: ~/.config/<app-name>/credentials.dat
//
// JS 调用示例:
//
//	// 保存
//	window.__lhpanda__('saveCredentials', {username: 'admin', password: '123456'})
//	// 查询所有
//	window.__lhpanda__('getCredentials')
//	// 查询指定用户是否存在
//	window.__lhpanda__('getCredentials', {username: 'admin'})
//	// 删除
//	window.__lhpanda__('deleteCredentials', {username: 'admin'})
//	// 清除全部
//	window.__lhpanda__('clearCredentials')

//go:build !minimal && !nocredentials

package bridge

import (
	"fmt"

	"github.com/lhpanda/webtodesktop/pkg"
)

// handleSaveCredentials 加密保存用户凭证（新增或更新，按用户名去重）。
//
// JS 调用: window.__lhpanda__('saveCredentials', {username: "admin", password: "123456"})
// 返回:    {saved: true}
//
// 如果用户名已存在则更新密码，否则新增记录。
// 密码经 AES-256-GCM 加密后写入磁盘，前端无法逆向读取。
func (b *Bridge) HandleSaveCredentials(params map[string]interface{}) (interface{}, error) {
	username, _ := params["username"].(string)
	password, _ := params["password"].(string)
	if username == "" || password == "" {
		return nil, fmt.Errorf("用户名和密码不能为空")
	}
	cred := &pkg.CredentialData{Username: username, Password: password}
	if err := b.store.SaveCredentials(cred); err != nil {
		return nil, fmt.Errorf("保存凭证失败: %w", err)
	}
	return map[string]interface{}{"saved": true}, nil
}

// handleGetCredentials 读取已保存的凭证。
//
// JS 调用:
//
//	// 查询所有已保存用户（仅返回用户名，不含密码）
//	window.__lhpanda__('getCredentials')
//	→ {found: true, credentials: [{username: "admin"}, {username: "user1"}]}
//
//	// 查询指定用户是否存在
//	window.__lhpanda__('getCredentials', {username: "admin"})
//	→ {found: true, username: "admin"}
//
// ★ 安全设计: 密码永不出现在返回值中，前端通过 X-Credential-Username 请求头告知代理层使用哪个账号，
// 代理层在服务端将请求体中的 __DESKTOP_PWD__ 占位符替换为真实密码后转发。
func (b *Bridge) HandleGetCredentials(params map[string]interface{}) (interface{}, error) {
	username, hasUsername := params["username"].(string)

	// 查询单个用户是否存在
	if hasUsername && username != "" {
		cred, err := b.store.GetCredentials(username)
		if err != nil {
			return nil, fmt.Errorf("读取凭证失败: %w", err)
		}
		if cred == nil {
			return map[string]interface{}{"found": false}, nil
		}
		// 注意: 不返回密码
		return map[string]interface{}{"found": true, "username": cred.Username}, nil
	}

	// 返回所有已保存用户列表（仅用户名）
	allCreds, err := b.store.GetAllCredentials()
	if err != nil {
		return nil, fmt.Errorf("读取凭证列表失败: %w", err)
	}
	type credItem struct {
		Username string `json:"username"`
	}
	items := make([]credItem, 0, len(allCreds))
	for _, c := range allCreds {
		items = append(items, credItem{Username: c.Username})
	}
	return map[string]interface{}{"found": len(items) > 0, "credentials": items}, nil
}

// handleDeleteCredentials 删除指定用户的凭证。
//
// JS 调用: window.__lhpanda__('deleteCredentials', {username: "admin"})
// 返回:    {deleted: true}
//
// 用户不存在时不报错，直接返回成功。
func (b *Bridge) HandleDeleteCredentials(params map[string]interface{}) (interface{}, error) {
	username, _ := params["username"].(string)
	if username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}
	if err := b.store.DeleteCredentials(username); err != nil {
		return nil, fmt.Errorf("删除凭证失败: %w", err)
	}
	return map[string]interface{}{"deleted": true}, nil
}

// handleClearCredentials 清除所有已保存的凭证。
//
// JS 调用: window.__lhpanda__('clearCredentials')
// 返回:    {cleared: true}
//
// 直接删除 credentials.dat 文件，下次保存时自动重建。
func (b *Bridge) HandleClearCredentials(params map[string]interface{}) (interface{}, error) {
	if err := b.store.ClearCredentials(); err != nil {
		return nil, fmt.Errorf("清除凭证失败: %w", err)
	}
	return map[string]interface{}{"cleared": true}, nil
}
