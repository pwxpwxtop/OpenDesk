package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// previewMaxBytes 是文本类文档允许在线查看的上限（1MB）。
const previewMaxBytes int64 = 1 << 20

// sshServiceRef 供 HTTP 预览路由访问当前会话（由 main 注入）。
var sshServiceRef *SSHService

// previewAssetHandler 拦截 /preview 请求，直接从远程读取文件内容（支持 Range，
// 因此视频/音频可以边下边播并拖动进度），其余请求交给静态资源服务。
func previewAssetHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/preview" {
			servePreview(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func servePreview(w http.ResponseWriter, r *http.Request) {
	if sshServiceRef == nil {
		http.Error(w, "服务未就绪", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("session")
	remotePath := r.URL.Query().Get("path")
	if sessionID == "" || remotePath == "" {
		http.Error(w, "缺少 session 或 path 参数", http.StatusBadRequest)
		return
	}

	sess, err := sshServiceRef.get(sessionID)
	if err != nil {
		http.Error(w, "会话不存在或已断开", http.StatusNotFound)
		return
	}
	client, err := sshServiceRef.sftpClient(sess)
	if err != nil {
		http.Error(w, "无法建立 SFTP 连接", http.StatusBadGateway)
		return
	}
	file, err := client.Open(remotePath)
	if err != nil {
		http.Error(w, "无法打开远程文件", http.StatusNotFound)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "目标不是文件", http.StatusBadRequest)
		return
	}

	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(remotePath))); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

// ReadTextFile returns the remote file content for the in-app text viewer.
// Files larger than 1MB are rejected.
func (s *SSHService) ReadTextFile(sessionID string, remotePath string) (string, error) {
	sess, err := s.get(sessionID)
	if err != nil {
		return "", err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return "", err
	}
	file, err := client.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr == nil && info.Size() > previewMaxBytes {
		return "", errors.New("文件超过 1MB，不支持查看，请下载后打开")
	}

	data, err := io.ReadAll(io.LimitReader(file, previewMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	if int64(len(data)) > previewMaxBytes {
		return "", errors.New("文件超过 1MB，不支持查看，请下载后打开")
	}
	return string(data), nil
}

var _ io.ReadSeeker = (*sftp.File)(nil)
