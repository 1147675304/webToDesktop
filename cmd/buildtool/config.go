// 配置解析：config.yaml 读取、项目查找、.env 读取
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BuildConfig 对应 config.yaml 结构
type BuildConfig struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`
	Projects []struct {
		Name         string `yaml:"name"`
		Description  string `yaml:"description"`
		VueDir       string `yaml:"vue_dir"`
		ExternalHTML bool   `yaml:"external_html"`
	} `yaml:"projects"`
}

func loadConfig() (*BuildConfig, error) {
	data, err := os.ReadFile(configYAML)
	if err != nil {
		return nil, fmt.Errorf("无法读取 config.yaml: %w", err)
	}
	var cfg BuildConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 config.yaml 失败: %w", err)
	}
	return &cfg, nil
}

func findProject(name string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	for _, p := range cfg.Projects {
		if p.Name == name {
			return p.VueDir, nil
		}
	}
	return "", fmt.Errorf("在 config.yaml 中找不到项目 '%s'", name)
}

func getProjectConfigByIndex(idx int) (*struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	VueDir       string `yaml:"vue_dir"`
	ExternalHTML bool   `yaml:"external_html"`
}, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= len(cfg.Projects) {
		return nil, fmt.Errorf("项目索引 %d 超出范围", idx)
	}
	return &cfg.Projects[idx], nil
}

// isExternalHTML 从 config.yaml 查询指定项目是否启用外挂 HTML 模式。
func isExternalHTML(project string) bool {
	cfg, err := loadConfig()
	if err != nil {
		return false
	}
	for _, p := range cfg.Projects {
		if p.Name == project {
			return p.ExternalHTML
		}
	}
	return false
}

// readEnv 读取 .env.production，补充缺失项
func readEnv(envFile string) (remoteURL, proxyPrefixes, signHeader string, err error) {
	if !fileExists(envFile) {
		// 纯前端项目（无 package.json）用默认值，不报错
		dir := filepath.Dir(envFile)
		if !fileExists(filepath.Join(dir, "package.json")) {
			return "about:blank", "", "", nil
		}
		return "", "", "", fmt.Errorf("找不到 %s", envFile)
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		return "", "", "", err
	}

	envMap := parseEnvFile(string(data))

	remoteURL = envMap["VITE_REMOTE_API_URL"]
	if remoteURL == "" {
		warnf("╔══════════════════════════════════════════════╗")
		warnf("║  检测到缺失配置项: VITE_REMOTE_API_URL        ║")
		warnf("║  已自动添加默认值，请修改为实际远程地址:       ║")
		warnf("║  → %s", envFile)
		warnf("╚══════════════════════════════════════════════╝")
		appendEnv(envFile, "\n# ★ 远程 API 服务器地址（由构建工具读取）\n", "VITE_REMOTE_API_URL=https://your-api-server.com\n")
		return "", "", "", fmt.Errorf("请修改 %s 中的 VITE_REMOTE_API_URL 后重新构建", envFile)
	}

	proxyPrefixes = envMap["VITE_PROXY_PREFIXES"]
	if proxyPrefixes == "" {
		warnf("╔══════════════════════════════════════════════╗")
		warnf("║  检测到缺失配置项: VITE_PROXY_PREFIXES        ║")
		warnf("║  已自动添加默认值 (/api/, /storage/)         ║")
		warnf("║  → %s", envFile)
		warnf("╚══════════════════════════════════════════════╝")
		appendEnv(envFile, "\n# ★ 代理到远程的路径前缀，逗号分隔（由构建工具读取）\n", "VITE_PROXY_PREFIXES=/api/,/storage/\n")
		proxyPrefixes = "/api/,/storage/"
	}

	signHeader = envMap["VITE_DESKTOP_SIGN_HEADER"]
	if signHeader == "" {
		signHeader = "X-Desktop-Signature"
	}

	return remoteURL, proxyPrefixes, signHeader, nil
}

func parseEnvFile(content string) map[string]string {
	envMap := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}

func appendEnv(path, prefix, entry string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(prefix)
	f.WriteString(entry)
}

// extractEnvValue 从 env 内容中提取指定 key 的值
func extractEnvValue(content, key string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}
