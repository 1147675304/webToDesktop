// tools/desktop/pkg/server.go
// 本地 HTTP 服务器 + 智能代理处理器
package pkg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
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

func StartServer(staticFS fs.FS, remoteURL string, proxyPrefixes []string, store *Store, signHeader string) (string, *http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("无法监听端口: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", NewProxyHandler(staticFS, remoteURL, proxyPrefixes, store, signHeader))

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

	return fmt.Sprintf("http://%s", listener.Addr().String()), server, nil
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
}

func NewProxyHandler(staticFS fs.FS, remoteURL string, proxyPrefixes []string, store *Store, signHeader string) http.Handler {
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
	}
	if target != nil {
		h.reverseProxy = httputil.NewSingleHostReverseProxy(target)
	}
	return h
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
				if strings.Contains(bodyStr, "__DESKTOP_PWD__") {
					bodyStr = strings.ReplaceAll(bodyStr, "__DESKTOP_PWD__", cred.Password)
					r.Body = io.NopCloser(strings.NewReader(bodyStr))
					r.ContentLength = int64(len(bodyStr))
				} else {
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					r.ContentLength = int64(len(bodyBytes))
				}
			}
		}
	}

	log.Printf("[proxy] %s %s → %s%s", r.Method, r.URL.Path, h.remoteURL.Host, r.URL.Path)
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
