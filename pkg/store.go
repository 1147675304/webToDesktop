// tools/desktop/pkg/store.go
// 本地加密持久化存储：AES-256-GCM + 配置文件指定密钥
package pkg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// CredentialData 凭证数据
type CredentialData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type storePayload struct {
	Credentials []CredentialData  `json:"credentials,omitempty"`
	KeyValues   map[string]string `json:"key_values,omitempty"`
}

// Store 本地加密持久化存储
type Store struct {
	mu               sync.Mutex
	filePath         string
	windowConfigPath string
	key              []byte
}

// NewStore 创建存储实例。
// projectName 用于隔离不同前端项目的持久化数据。
func NewStore(projectName string) (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户配置目录失败: %w", err)
	}

	appDir := AppCfg.App.Name
	if appDir == "" {
		appDir = "webtodesktop"
	}
	// 按项目名称隔离持久化数据
	if projectName != "" {
		appDir = appDir + "/" + projectName
	}
	dir := filepath.Join(configDir, appDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	aesKey := sha256.Sum256([]byte(AppCfg.Security.AESKey))

	return &Store{
		filePath:         filepath.Join(dir, "credentials.dat"),
		windowConfigPath: filepath.Join(dir, "window_config.json"),
		key:              aesKey[:],
	}, nil
}

// SaveCredentials 加密保存凭证
func (s *Store) SaveCredentials(cred *CredentialData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return err
	}
	found := false
	for i, c := range payload.Credentials {
		if c.Username == cred.Username {
			payload.Credentials[i].Password = cred.Password
			found = true
			break
		}
	}
	if !found {
		payload.Credentials = append(payload.Credentials, *cred)
	}
	return s.writeEncrypted(payload)
}

// GetAllCredentials 获取所有凭证
func (s *Store) GetAllCredentials() ([]CredentialData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.readEncrypted()
	if err != nil {
		return nil, err
	}
	if payload.Credentials == nil {
		return []CredentialData{}, nil
	}
	return payload.Credentials, nil
}

// GetCredentials 获取指定用户凭证
func (s *Store) GetCredentials(username string) (*CredentialData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.readEncrypted()
	if err != nil {
		return nil, err
	}
	for _, c := range payload.Credentials {
		if c.Username == username {
			return &CredentialData{Username: c.Username, Password: c.Password}, nil
		}
	}
	return nil, nil
}

// DeleteCredentials 删除指定用户凭证
func (s *Store) DeleteCredentials(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := s.readEncrypted()
	if err != nil {
		return err
	}
	for i, c := range payload.Credentials {
		if c.Username == username {
			payload.Credentials = append(payload.Credentials[:i], payload.Credentials[i+1:]...)
			return s.writeEncrypted(payload)
		}
	}
	return nil
}

// ClearCredentials 清除所有凭证
func (s *Store) ClearCredentials() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清除凭证失败: %w", err)
	}
	return nil
}

// SetItem 设置键值对（加密存储）
func (s *Store) SetItem(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return err
	}
	if payload.KeyValues == nil {
		payload.KeyValues = make(map[string]string)
	}
	payload.KeyValues[key] = value
	return s.writeEncrypted(payload)
}

// GetItem 获取指定键的值
func (s *Store) GetItem(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return "", err
	}
	if payload.KeyValues == nil {
		return "", nil
	}
	return payload.KeyValues[key], nil
}

// RemoveItem 删除指定键
func (s *Store) RemoveItem(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return err
	}
	if payload.KeyValues != nil {
		delete(payload.KeyValues, key)
		return s.writeEncrypted(payload)
	}
	return nil
}

// ClearItems 清除所有键值对
func (s *Store) ClearItems() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return err
	}
	payload.KeyValues = nil
	return s.writeEncrypted(payload)
}

// GetAllItems 获取所有键值对
func (s *Store) GetAllItems() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readEncrypted()
	if err != nil {
		return nil, err
	}
	if payload.KeyValues == nil {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(payload.KeyValues))
	for k, v := range payload.KeyValues {
		result[k] = v
	}
	return result, nil
}

// SaveWindowConfig 保存窗口配置
func (s *Store) SaveWindowConfig(cfg *WindowConfigData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化窗口配置失败: %w", err)
	}
	return os.WriteFile(s.windowConfigPath, data, 0600)
}

// LoadWindowConfig 加载窗口配置
func (s *Store) LoadWindowConfig() (*WindowConfigData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.windowConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取窗口配置失败: %w", err)
	}
	var cfg WindowConfigData
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析窗口配置失败: %w", err)
	}
	return &cfg, nil
}

// ———— 加密工具 ————

func (s *Store) writeEncrypted(payload *storePayload) error {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}
	ciphertext, err := encrypt(plaintext, s.key)
	if err != nil {
		return fmt.Errorf("加密数据失败: %w", err)
	}
	return os.WriteFile(s.filePath, ciphertext, 0600)
}

func (s *Store) readEncrypted() (*storePayload, error) {
	ciphertext, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &storePayload{}, nil
		}
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	plaintext, err := decrypt(ciphertext, s.key)
	if err != nil {
		return &storePayload{}, nil
	}
	var payload storePayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return &storePayload{}, nil
	}
	return &payload, nil
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("生成随机数失败: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Encrypt 公开的加密函数（供 proxy.go 签名使用）
func Encrypt(plaintext, key []byte) ([]byte, error) {
	return encrypt(plaintext, key)
}
