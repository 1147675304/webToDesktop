// tools/desktop/pkg/server.go
// 本地 HTTP 服务器 + 智能代理处理器
package pkg

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// ———— HTTP 服务器 ————

// accessTokenKey 是访问令牌的查询参数名和请求头名
const accessTokenKey = "_wtd_"

// generateAccessToken 生成 16 字节随机令牌，hex 编码为 32 字符字符串。
func generateAccessToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成访问令牌失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func StartServer(staticFS fs.FS, remoteURL string, proxyPrefixes []string, store *Store, signHeader string) (string, string, *http.Server, error) {
	token, err := generateAccessToken()
	if err != nil {
		return "", "", nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", nil, fmt.Errorf("无法监听端口: %w", err)
	}

	// 从监听地址中提取端口号，用于构造实例隔离的 Cookie 名称
	_, port, _ := net.SplitHostPort(listener.Addr().String())

	mux := http.NewServeMux()
	mux.Handle("/", NewProxyHandler(staticFS, remoteURL, proxyPrefixes, store, signHeader, token, port))

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP 服务错误: %v\n", err)
		}
	}()

	addr := fmt.Sprintf("http://%s", listener.Addr().String())
	return addr, token, server, nil
}

func ShutdownServer(server *http.Server, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("服务关闭错误: %v\n", err)
	}
}

// ———— 代理处理器 ————

type proxyHandler struct {
	staticFS      fs.FS
	remoteURL     *url.URL
	proxyPrefixes []string
	store         *Store
	reverseProxy  *httputil.ReverseProxy
	signHeader    string // 桌面端签名请求头名称（来自 .env.production 的 VITE_DESKTOP_SIGN_HEADER）
	accessToken   string // 随机访问令牌，防止浏览器访问
	cookieName    string // 实例隔离的 Cookie 名称（含端口号），避免多实例 Cookie 冲突
}

// accessTokenQuery 从请求中提取访问令牌（优先级: 查询参数 > Cookie > 请求头）。
// ★ 查询参数优先于 Cookie，确保初始导航不会被其他实例的 Cookie 干扰。
// 返回令牌值，未找到则返回空字符串。
func (h *proxyHandler) accessTokenQuery(r *http.Request) string {
	// 1. URL 查询参数（初始导航时使用，优先级最高，避免多实例 Cookie 污染）
	if token := r.URL.Query().Get(accessTokenKey); token != "" {
		return token
	}
	// 2. Cookie（所有同域请求自动携带，使用实例隔离的 cookieName）
	if c, err := r.Cookie(h.cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	// 3. 自定义请求头（XHR/fetch 由前端 JS 手动添加）
	return r.Header.Get("X-Desktop-Signature")
}

// verifyAccessToken 校验请求是否携带有效的访问令牌。
// WebView 的初始导航通过 URL 查询参数携带令牌，服务器设置 Cookie，
// 此后该域的所有请求自动携带 Cookie，无需前端额外处理。
func (h *proxyHandler) verifyAccessToken(r *http.Request) bool {
	return h.accessTokenQuery(r) == h.accessToken
}

// setAccessTokenCookie 当请求通过查询参数验证令牌后，设置 Cookie 供后续请求自动携带。
// 使用实例隔离的 cookieName（含端口后缀），避免多实例 Cookie 互相覆盖。
func (h *proxyHandler) setAccessTokenCookie(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get(accessTokenKey)
	if token == h.accessToken {
		http.SetCookie(w, &http.Cookie{
			Name:     h.cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func NewProxyHandler(staticFS fs.FS, remoteURL string, proxyPrefixes []string, store *Store, signHeader string, accessToken string, port string) http.Handler {
	target, _ := url.Parse(remoteURL)
	// 默认头名称
	if signHeader == "" {
		signHeader = "X-Desktop-Signature"
	}
	h := &proxyHandler{
		staticFS:      staticFS,
		remoteURL:     target,
		proxyPrefixes: proxyPrefixes,
		store:         store,
		signHeader:    strings.TrimSpace(signHeader),
		accessToken:   accessToken,
		cookieName:    accessTokenKey + port, // 实例隔离: _wtd_<port>
	}
	if target != nil {
		h.reverseProxy = httputil.NewSingleHostReverseProxy(target)
	}
	return h
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ★ 访问令牌校验：所有请求必须携带有效令牌
	if !h.verifyAccessToken(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 如果令牌来自 URL 查询参数（初始导航），设置 Cookie 供后续请求自动携带
	// 这样 JS/CSS 等静态资源加载也能通过 Cookie 认证
	hasTokenInURL := r.URL.Query().Get(accessTokenKey) != ""
	if hasTokenInURL {
		h.setAccessTokenCookie(w, r)
	}

	// 从 URL 中移除令牌查询参数（避免污染路径匹配和 SPA 路由）
	q := r.URL.Query()
	q.Del(accessTokenKey)
	r.URL.RawQuery = q.Encode()

	// 1. 代理前缀匹配
	for _, prefix := range h.proxyPrefixes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			h.handleProxy(w, r)
			return
		}
	}

	// 2. /storage/ 特殊前缀
	if strings.HasPrefix(r.URL.Path, "/storage/") {
		h.handleProxy(w, r)
		return
	}

	// 3. 本地静态资源（支持 .gz 压缩文件）
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	f, err := h.staticFS.Open(path)
	isGzip := false
	if err != nil {
		// 原始文件不存在，尝试 .gz 压缩版本
		f, err = h.staticFS.Open(path + ".gz")
		isGzip = true
	}
	if err == nil {
		defer f.Close()
		if isGzip {
			w.Header().Set("Content-Encoding", "gzip")
		}
		ext := filepath.Ext(path)
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		if info, _ := f.Stat(); info != nil {
			http.ServeContent(w, r, path, info.ModTime(), f.(io.ReadSeeker))
		} else {
			io.Copy(w, f)
		}
		return
	}

	// 4. SPA fallback（也支持 .gz 版本）
	indexFile, err := h.staticFS.Open("index.html")
	indexGzip := false
	if err != nil {
		indexFile, err = h.staticFS.Open("index.html.gz")
		indexGzip = true
	}
	if err != nil {
		http.Error(w, "Not Found", 404)
		return
	}
	defer indexFile.Close()
	if indexGzip {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if info, _ := indexFile.Stat(); info != nil {
		http.ServeContent(w, r, "index.html", info.ModTime(), indexFile.(io.ReadSeeker))
	} else {
		io.Copy(w, indexFile)
	}
}

func (h *proxyHandler) handleProxy(w http.ResponseWriter, r *http.Request) {
	if h.reverseProxy == nil {
		http.Error(w, "远程服务器未配置", 502)
		return
	}

	// 移除访问令牌请求头和 Cookie，避免泄露到远程服务器
	r.Header.Del("X-WTD-Token")
	r.Header.Del("Cookie")

	// 桌面端签名：使用配置的请求头名，值为 AES 加密的密钥+日期
	if h.signHeader != "" && h.store != nil {
		sig := buildSignature()
		if sig != "" {
			r.Header.Set(h.signHeader, sig)
		}
	}

	// 凭证透明注入：从请求头获取用户名，查询真实密码
	if username := r.Header.Get("X-Credential-Username"); username != "" && h.store != nil {
		if cred, _ := h.store.GetCredentials(username); cred != nil {
			// 读取并替换请求体中的 __DESKTOP_PWD__
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				r.Body.Close()
				bodyStr := string(bodyBytes)
				hasPlaceholder := strings.Contains(bodyStr, "__DESKTOP_PWD__")
				if hasPlaceholder {
					bodyStr = strings.ReplaceAll(bodyStr, "__DESKTOP_PWD__", cred.Password)
				}
				r.Body = io.NopCloser(strings.NewReader(bodyStr))
				r.ContentLength = int64(len(bodyStr))
			}
		}
	}

	// ★ 修复 Host 头：ReverseProxy 只修改了 req.URL.Host 但未修改 req.Host，
	// HTTP 客户端优先使用 req.Host（仍为原始请求的 127.0.0.1:port），
	// 导致远程服务器因虚拟主机不匹配返回 404。
	r.Host = h.remoteURL.Host

	h.reverseProxy.ServeHTTP(w, r)
}

// buildSignature 使用 AES-256-GCM 加密 "密钥+日期" 生成桌面端签名。
// 签名 = Base64(AES-GCM({key: aes_key, date: "20060102"}))，每次请求重新生成。
func buildSignature() string {
	payload := map[string]string{
		"key":  AppCfg.Security.AESKey,
		"date": time.Now().Format("20060102"),
	}
	plaintext, _ := json.Marshal(payload)
	aesKey := sha256.Sum256([]byte(AppCfg.Security.AESKey))
	ciphertext, err := Encrypt(plaintext, aesKey[:])
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(ciphertext)
}
