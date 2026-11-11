package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/crypto/ssh"
)

// SSHConfig describes how to reach a remote Linux host.
type SSHConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	PrivateKeyPath string `json:"privateKeyPath"`
	Passphrase     string `json:"passphrase"`
}

// SSHOutput is the payload of the "ssh:output" event.
// Data is base64 encoded PTY output so that arbitrary bytes survive JSON.
type SSHOutput struct {
	SessionID string `json:"sessionId"`
	Data      string `json:"data"`
}

// SSHClosed is the payload of the "ssh:closed" event.
type SSHClosed struct {
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason"`
}

// UploadProgress reports the transfer state of a single file to the UI.
// While bytes are still moving Done is false and Transferred grows; once the
// file is written Done is true (or Error explains the failure).
type UploadProgress struct {
	SessionID   string `json:"sessionId"`
	RemoteDir   string `json:"remoteDir"`
	Name        string `json:"name"`
	RemotePath  string `json:"remotePath"`
	Size        int64  `json:"size"`
	Transferred int64  `json:"transferred"`
	Done        bool   `json:"done"`
	Error       string `json:"error"`
}

// progressReader reports how many bytes have been read so far, throttled to
// avoid flooding the event bus with updates.
type progressReader struct {
	reader io.Reader
	sent   int64
	last   time.Time
	report func(sent int64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.sent += int64(n)
		if p.report != nil && time.Since(p.last) >= 120*time.Millisecond {
			p.last = time.Now()
			p.report(p.sent)
		}
	}
	return n, err
}

// UploadItem describes one top-level item copied to the remote host, so the
// frontend can draw it as a desktop icon.
type UploadItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// UploadSummary is the payload of the "ssh:upload:summary" event.
type UploadSummary struct {
	SessionID string       `json:"sessionId"`
	RemoteDir string       `json:"remoteDir"`
	Total     int          `json:"total"`
	Uploaded  int          `json:"uploaded"`
	Failed    int          `json:"failed"`
	Items     []UploadItem `json:"items"`
}

// FileEntry is a directory entry returned by the SFTP file manager.
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	IsLink  bool   `json:"isLink"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"modTime"`
}

// DiskUsage is one mounted filesystem.
type DiskUsage struct {
	Filesystem string  `json:"filesystem"`
	Mount      string  `json:"mount"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	Percent    float64 `json:"percent"`
}

// ProcessInfo is one row of the remote process list.
type ProcessInfo struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	Command string  `json:"command"`
}

// Metrics is the snapshot rendered by the monitor app.
type Metrics struct {
	Hostname   string        `json:"hostname"`
	Kernel     string        `json:"kernel"`
	Uptime     string        `json:"uptime"`
	LoadAvg    string        `json:"loadAvg"`
	CPUPercent float64       `json:"cpuPercent"`
	CPUCores   int           `json:"cpuCores"`
	MemTotal   uint64        `json:"memTotal"`
	MemUsed    uint64        `json:"memUsed"`
	SwapTotal  uint64        `json:"swapTotal"`
	SwapUsed   uint64        `json:"swapUsed"`
	Disks      []DiskUsage   `json:"disks"`
	Processes  []ProcessInfo `json:"processes"`
	Timestamp  int64         `json:"timestamp"`
}

type cpuSample struct {
	idle  uint64
	total uint64
	at    time.Time
}

type sshSession struct {
	id      string
	cfg     SSHConfig
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser

	sftpMu sync.Mutex
	sftp   *sftp.Client

	cpuMu   sync.Mutex
	lastCPU cpuSample

	closeOnce sync.Once
	closed    chan struct{}
}

// SSHService exposes SSH / SFTP / monitoring operations to the frontend.
type SSHService struct {
	mu       sync.RWMutex
	sessions map[string]*sshSession

	winMu   sync.Mutex
	windows map[string]map[uint]struct{}
}

func newSSHService() *SSHService {
	return &SSHService{
		sessions: make(map[string]*sshSession),
		windows:  make(map[string]map[uint]struct{}),
	}
}

func newSessionID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Connect opens an interactive PTY shell and returns the session id.
func (s *SSHService) Connect(cfg SSHConfig) (string, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return "", errors.New("主机地址不能为空")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return "", errors.New("用户名不能为空")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}

	clientCfg, err := buildClientConfig(cfg)
	if err != nil {
		return "", err
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	client, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		return "", fmt.Errorf("连接失败: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return "", fmt.Errorf("创建会话失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 30, 100, modes); err != nil {
		session.Close()
		client.Close()
		return "", fmt.Errorf("申请终端失败: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return "", err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return "", err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return "", err
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return "", fmt.Errorf("启动 shell 失败: %w", err)
	}

	id, err := newSessionID()
	if err != nil {
		session.Close()
		client.Close()
		return "", err
	}

	sess := &sshSession{
		id:      id,
		cfg:     cfg,
		client:  client,
		session: session,
		stdin:   stdin,
		closed:  make(chan struct{}),
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	go s.pump(id, stdout)
	go s.pump(id, stderr)
	go s.keepAlive(sess)
	go func() {
		_ = session.Wait()
		s.cleanup(sess, "连接已断开")
	}()

	return id, nil
}

func buildClientConfig(cfg SSHConfig) (*ssh.ClientConfig, error) {
	var auths []ssh.AuthMethod

	if cfg.PrivateKeyPath != "" {
		key, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取私钥失败: %w", err)
		}
		var signer ssh.Signer
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		auths = append(auths, ssh.Password(cfg.Password))
	}
	auths = append(auths, ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, len(questions))
		for i := range answers {
			answers[i] = cfg.Password
		}
		return answers, nil
	}))

	if len(auths) == 0 {
		return nil, errors.New("请填写密码或选择私钥")
	}

	return &ssh.ClientConfig{
		User: cfg.User,
		Auth: auths,
		// TODO: 后续可接入 known_hosts 校验，这里为了便于连接任意测试机暂时跳过。
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}, nil
}

func (s *SSHService) pump(sessionID string, r io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			application.Get().Event.Emit("ssh:output", SSHOutput{
				SessionID: sessionID,
				Data:      base64.StdEncoding.EncodeToString(buf[:n]),
			})
		}
		if err != nil {
			return
		}
	}
}

func (s *SSHService) keepAlive(sess *sshSession) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-sess.closed:
			return
		case <-ticker.C:
			if _, _, err := sess.client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				return
			}
		}
	}
}

func (s *SSHService) cleanup(sess *sshSession, reason string) {
	sess.closeOnce.Do(func() {
		close(sess.closed)
		s.mu.Lock()
		delete(s.sessions, sess.id)
		s.mu.Unlock()

		sess.sftpMu.Lock()
		if sess.sftp != nil {
			_ = sess.sftp.Close()
			sess.sftp = nil
		}
		sess.sftpMu.Unlock()

		_ = sess.session.Close()
		_ = sess.client.Close()

		application.Get().Event.Emit("ssh:closed", SSHClosed{SessionID: sess.id, Reason: reason})
	})
}

func (s *SSHService) get(sessionID string) (*sshSession, error) {
	s.mu.RLock()
	sess, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("会话不存在或已断开")
	}
	return sess, nil
}

// Write sends keyboard input to the remote PTY.
func (s *SSHService) Write(sessionID string, data string) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(sess.stdin, data)
	return err
}

// Resize updates the remote PTY window size.
func (s *SSHService) Resize(sessionID string, cols int, rows int) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return err
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return sess.session.WindowChange(rows, cols)
}

// Disconnect closes a session.
func (s *SSHService) Disconnect(sessionID string) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return nil
	}
	s.cleanup(sess, "已主动断开")
	return nil
}

// sessionWindowSpec describes the size of a session window per app kind.
type sessionWindowSpec struct {
	width  int
	height int
	minW   int
	minH   int
}

var sessionWindowSpecs = map[string]sessionWindowSpec{
	"desktop": {width: 1280, height: 800, minW: 960, minH: 620},
}

// OpenSessionWindow opens a native window bound to the given session.
// kind is currently "desktop": a Windows-like desktop for the remote host.
func (s *SSHService) OpenSessionWindow(sessionID string, kind string, title string, host string) error {
	if _, err := s.get(sessionID); err != nil {
		return err
	}
	spec, ok := sessionWindowSpecs[kind]
	if !ok {
		return fmt.Errorf("未知窗口类型: %s", kind)
	}
	if title == "" {
		title = "OpenDesk"
	}

	query := url.Values{}
	query.Set("host", host)
	target := fmt.Sprintf("/#/%s/%s?%s", kind, url.PathEscape(sessionID), query.Encode())

	app := application.Get()
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            title,
		Width:            spec.width,
		Height:           spec.height,
		MinWidth:         spec.minW,
		MinHeight:        spec.minH,
		BackgroundColour: application.NewRGB(18, 22, 30),
		URL:              target,
		EnableFileDrop:   true,
	})

	// 从系统资源管理器拖入文件/文件夹时，上传到投放目标所在目录
	// （data-drop-dir 由前端标注，缺省为远程用户主目录）。
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
		files := e.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		remoteDir := ""
		if target := e.Context().DropTargetDetails(); target != nil {
			remoteDir = target.Attributes["data-drop-dir"]
		}
		go s.uploadLocalFiles(sessionID, remoteDir, files)
	})

	s.trackWindow(sessionID, win)
	win.Focus()
	return nil
}

// trackWindow disconnects the session once the last window using it is closed.
func (s *SSHService) trackWindow(sessionID string, win *application.WebviewWindow) {
	windowID := win.ID()

	s.winMu.Lock()
	if s.windows[sessionID] == nil {
		s.windows[sessionID] = make(map[uint]struct{})
	}
	s.windows[sessionID][windowID] = struct{}{}
	s.winMu.Unlock()

	win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.winMu.Lock()
		delete(s.windows[sessionID], windowID)
		remaining := len(s.windows[sessionID])
		if remaining == 0 {
			delete(s.windows, sessionID)
		}
		s.winMu.Unlock()

		if remaining == 0 {
			_ = s.Disconnect(sessionID)
		}
	})
}

func (s *SSHService) sftpClient(sess *sshSession) (*sftp.Client, error) {
	sess.sftpMu.Lock()
	defer sess.sftpMu.Unlock()
	if sess.sftp != nil {
		return sess.sftp, nil
	}
	client, err := sftp.NewClient(sess.client)
	if err != nil {
		return nil, fmt.Errorf("SFTP 不可用: %w", err)
	}
	sess.sftp = client
	return client, nil
}

// HomeDir returns the remote user's home directory.
func (s *SSHService) HomeDir(sessionID string) (string, error) {
	sess, err := s.get(sessionID)
	if err != nil {
		return "", err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return "", err
	}
	wd, err := client.Getwd()
	if err != nil {
		return "/", nil
	}
	return wd, nil
}

// ListDir lists a remote directory through SFTP.
func (s *SSHService) ListDir(sessionID string, dir string) ([]FileEntry, error) {
	sess, err := s.get(sessionID)
	if err != nil {
		return nil, err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return nil, err
	}
	if dir == "" {
		dir = "/"
	}
	entries, err := client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		full := path.Join(dir, entry.Name())
		isDir := entry.IsDir()
		isLink := entry.Mode()&os.ModeSymlink != 0
		if isLink {
			if target, statErr := client.Stat(full); statErr == nil {
				isDir = target.IsDir()
			}
		}
		result = append(result, FileEntry{
			Name:    entry.Name(),
			Path:    full,
			IsDir:   isDir,
			IsLink:  isLink,
			Size:    entry.Size(),
			Mode:    entry.Mode().String(),
			ModTime: entry.ModTime().Unix(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

// PickAndUpload opens a native picker (files and/or folders) and uploads the
// selection into remoteDir in the background. Progress is reported through the
// "ssh:upload" and "ssh:upload:summary" events so the UI never blocks.
func (s *SSHService) PickAndUpload(sessionID string, remoteDir string) error {
	if _, err := s.get(sessionID); err != nil {
		return err
	}
	paths, err := application.Get().Dialog.OpenFile().
		SetTitle("选择要上传到远程桌面的文件").
		CanChooseFiles(true).
		CanChooseDirectories(true).
		PromptForMultipleSelection()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	go s.uploadLocalFiles(sessionID, remoteDir, paths)
	return nil
}

// PasteLocalFiles uploads whatever files/folders the Windows clipboard holds.
// This powers Ctrl+C in Explorer followed by Ctrl+V inside OpenDesk.
func (s *SSHService) PasteLocalFiles(sessionID string, remoteDir string) error {
	if _, err := s.get(sessionID); err != nil {
		return err
	}
	files, err := readClipboardFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("剪贴板里没有可复制的文件")
	}
	go s.uploadLocalFiles(sessionID, remoteDir, files)
	return nil
}

func emitUpload(p UploadProgress) {
	application.Get().Event.Emit("ssh:upload", p)
}

// uploadLocalFiles copies local files (folders are walked recursively) to the
// remote host. An empty remoteDir means the remote user's home directory.
func (s *SSHService) uploadLocalFiles(sessionID string, remoteDir string, paths []string) {
	sess, err := s.get(sessionID)
	if err != nil {
		return
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		application.Get().Event.Emit("ssh:upload:summary", UploadSummary{
			SessionID: sessionID,
			RemoteDir: remoteDir,
			Total:     len(paths),
			Failed:    len(paths),
		})
		return
	}

	if remoteDir == "" || remoteDir == "desktop" {
		home := "/"
		if wd, wdErr := client.Getwd(); wdErr == nil && wd != "" {
			home = wd
		}
		if remoteDir == "desktop" {
			// 直接拖到桌面壁纸时，落到远程用户的 ~/Desktop。
			remoteDir = path.Join(home, "Desktop")
			_ = client.MkdirAll(remoteDir)
		} else {
			remoteDir = home
		}
	}

	summary := UploadSummary{SessionID: sessionID, RemoteDir: remoteDir, Items: []UploadItem{}}
	for _, localPath := range paths {
		info, statErr := os.Stat(localPath)
		if statErr != nil {
			summary.Total++
			summary.Failed++
			emitUpload(UploadProgress{
				SessionID: sessionID,
				RemoteDir: remoteDir,
				Name:      filepath.Base(localPath),
				Error:     statErr.Error(),
			})
			continue
		}

		if info.IsDir() {
			uploaded, failed := s.uploadTree(client, sessionID, remoteDir, localPath)
			summary.Total += uploaded + failed
			summary.Uploaded += uploaded
			summary.Failed += failed
			if uploaded > 0 {
				summary.Items = append(summary.Items, UploadItem{
					Name:  filepath.Base(localPath),
					Path:  path.Join(remoteDir, filepath.ToSlash(filepath.Base(localPath))),
					IsDir: true,
				})
			}
			continue
		}

		summary.Total++
		remotePath := path.Join(remoteDir, filepath.Base(localPath))
		if err := s.uploadFileTo(client, sessionID, remoteDir, localPath, remotePath, info.Size()); err != nil {
			summary.Failed++
			emitUpload(UploadProgress{
				SessionID: sessionID,
				RemoteDir: remoteDir,
				Name:      filepath.Base(localPath),
				Error:     err.Error(),
			})
			continue
		}
		summary.Uploaded++
		summary.Items = append(summary.Items, UploadItem{
			Name: filepath.Base(localPath),
			Path: remotePath,
		})
	}

	application.Get().Event.Emit("ssh:upload:summary", summary)
}

func (s *SSHService) uploadFileTo(client *sftp.Client, sessionID, remoteDir, localPath, remotePath string, size int64) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
		return fmt.Errorf("创建远端目录失败: %w", err)
	}
	dst, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远端文件失败: %w", err)
	}

	name := filepath.Base(localPath)
	reader := &progressReader{
		reader: src,
		last:   time.Now(),
		report: func(sent int64) {
			emitUpload(UploadProgress{
				SessionID:   sessionID,
				RemoteDir:   remoteDir,
				Name:        name,
				RemotePath:  remotePath,
				Size:        size,
				Transferred: sent,
			})
		},
	}

	_, copyErr := io.Copy(dst, reader)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = client.Remove(remotePath)
		return fmt.Errorf("上传失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = client.Remove(remotePath)
		return fmt.Errorf("写入失败: %w", closeErr)
	}
	_ = client.Chmod(remotePath, 0o644)

	emitUpload(UploadProgress{
		SessionID:   sessionID,
		RemoteDir:   remoteDir,
		Name:        name,
		RemotePath:  remotePath,
		Size:        size,
		Transferred: size,
		Done:        true,
	})
	return nil
}

// uploadTree mirrors a local directory tree into remoteDir and returns how many
// files succeeded / failed.
func (s *SSHService) uploadTree(client *sftp.Client, sessionID, remoteDir, localRoot string) (uploaded int, failed int) {
	base := filepath.Base(localRoot)
	_ = filepath.Walk(localRoot, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			failed++
			return nil
		}
		rel, relErr := filepath.Rel(localRoot, current)
		if relErr != nil {
			failed++
			return nil
		}

		target := path.Join(remoteDir, filepath.ToSlash(base))
		if rel != "." {
			target = path.Join(remoteDir, filepath.ToSlash(base), filepath.ToSlash(rel))
		}

		if info.IsDir() {
			if err := client.MkdirAll(target); err != nil {
				failed++
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		if err := s.uploadFileTo(client, sessionID, remoteDir, current, target, info.Size()); err != nil {
			failed++
			emitUpload(UploadProgress{
				SessionID: sessionID,
				RemoteDir: remoteDir,
				Name:      filepath.ToSlash(rel),
				Error:     err.Error(),
			})
			return nil
		}
		uploaded++
		return nil
	})
	return uploaded, failed
}

// Download opens a local save dialog and downloads remotePath to the chosen location.
func (s *SSHService) Download(sessionID string, remotePath string) (string, error) {
	sess, err := s.get(sessionID)
	if err != nil {
		return "", err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return "", err
	}

	localPath, err := application.Get().Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:    "保存到本地",
		Filename: path.Base(remotePath),
	}).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if localPath == "" {
		return "", nil
	}

	src, err := client.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("打开远端文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	return localPath, nil
}

// Remove deletes a file or directory (recursively) on the remote host.
func (s *SSHService) Remove(sessionID string, remotePath string, isDir bool) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return err
	}
	if isDir {
		return client.RemoveAll(remotePath)
	}
	return client.Remove(remotePath)
}

// Rename moves or renames a remote file/directory.
func (s *SSHService) Rename(sessionID string, oldPath string, newPath string) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return err
	}
	return client.Rename(oldPath, newPath)
}

// Mkdir creates a remote directory.
func (s *SSHService) Mkdir(sessionID string, remotePath string) error {
	sess, err := s.get(sessionID)
	if err != nil {
		return err
	}
	client, err := s.sftpClient(sess)
	if err != nil {
		return err
	}
	return client.MkdirAll(remotePath)
}

func (s *SSHService) run(sess *sshSession, command string) (string, error) {
	out, err := sess.client.NewSession()
	if err != nil {
		return "", err
	}
	defer out.Close()
	data, err := out.CombinedOutput(command)
	return string(data), err
}

func parseSections(raw string) map[string]string {
	sections := make(map[string]string)
	current := ""
	var buf strings.Builder
	flush := func() {
		if current != "" {
			sections[current] = buf.String()
		}
		buf.Reset()
	}
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "@@") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "@@"))
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
	return sections
}

func parseCPUStat(line string) cpuSample {
	fields := strings.Fields(line)
	sample := cpuSample{at: time.Now()}
	if len(fields) < 5 || fields[0] != "cpu" {
		return sample
	}
	var vals []uint64
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			continue
		}
		vals = append(vals, v)
	}
	if len(vals) < 4 {
		return sample
	}
	idle := vals[3]
	if len(vals) > 4 {
		idle += vals[4]
	}
	var total uint64
	for _, v := range vals {
		total += v
	}
	sample.idle = idle
	sample.total = total
	return sample
}

func parseMeminfo(raw string) (total uint64, available uint64, swapTotal uint64, swapUsed uint64) {
	values := make(map[string]uint64)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[key] = v * 1024
	}
	total = values["MemTotal"]
	available = values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	swapTotal = values["SwapTotal"]
	swapFree := values["SwapFree"]
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	return
}

func parseDisks(raw string) []DiskUsage {
	disks := make([]DiskUsage, 0, 4)
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		total, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)
		avail, _ := strconv.ParseUint(fields[3], 10, 64)
		if total == 0 {
			continue
		}
		disks = append(disks, DiskUsage{
			Filesystem: fields[0],
			Mount:      fields[5],
			Total:      total * 1024,
			Used:       used * 1024,
			Available:  avail * 1024,
			Percent:    float64(used) / float64(total) * 100,
		})
	}
	return disks
}

func parseProcesses(raw string) []ProcessInfo {
	procs := make([]ProcessInfo, 0, 16)
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		procs = append(procs, ProcessInfo{
			PID:     pid,
			User:    fields[1],
			CPU:     cpu,
			Mem:     mem,
			Command: strings.Join(fields[4:], " "),
		})
	}
	return procs
}

func formatUptime(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d 天 %d 小时 %d 分", days, hours, minutes)
	}
	return fmt.Sprintf("%d 小时 %d 分", hours, minutes)
}

// Metrics collects a system snapshot from the remote host.
func (s *SSHService) Metrics(sessionID string) (*Metrics, error) {
	sess, err := s.get(sessionID)
	if err != nil {
		return nil, err
	}

	const script = `
echo @@HOST
hostname 2>/dev/null
uname -sr 2>/dev/null
echo @@UPTIME
cat /proc/uptime 2>/dev/null
echo @@LOAD
cat /proc/loadavg 2>/dev/null
echo @@CPU
head -n 1 /proc/stat 2>/dev/null
echo @@CORES
nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null
echo @@MEM
cat /proc/meminfo 2>/dev/null
echo @@DISK
df -kP -x tmpfs -x devtmpfs 2>/dev/null
echo @@PROC
ps -eo pid,user,pcpu,pmem,args --sort=-pcpu 2>/dev/null | head -n 16
`
	raw, err := s.run(sess, script)
	if err != nil && raw == "" {
		return nil, err
	}
	sections := parseSections(raw)

	m := &Metrics{Timestamp: time.Now().Unix()}

	hostLines := strings.Split(strings.TrimSpace(sections["HOST"]), "\n")
	if len(hostLines) > 0 {
		m.Hostname = strings.TrimSpace(hostLines[0])
	}
	if len(hostLines) > 1 {
		m.Kernel = strings.TrimSpace(hostLines[1])
	}

	if up := strings.Fields(sections["UPTIME"]); len(up) > 0 {
		if secs, err := strconv.ParseFloat(up[0], 64); err == nil {
			m.Uptime = formatUptime(secs)
		}
	}

	if load := strings.Fields(sections["LOAD"]); len(load) >= 3 {
		m.LoadAvg = strings.Join(load[:3], " / ")
	}

	sample := parseCPUStat(strings.TrimSpace(sections["CPU"]))
	sess.cpuMu.Lock()
	prev := sess.lastCPU
	sess.lastCPU = sample
	sess.cpuMu.Unlock()

	if !prev.at.IsZero() && sample.total > prev.total {
		deltaTotal := float64(sample.total - prev.total)
		deltaIdle := float64(sample.idle - prev.idle)
		if deltaTotal > 0 {
			m.CPUPercent = (deltaTotal - deltaIdle) / deltaTotal * 100
		}
	}

	coresRaw := strings.TrimSpace(sections["CORES"])
	if coresRaw != "" {
		if cores, err := strconv.Atoi(strings.Fields(coresRaw)[0]); err == nil {
			m.CPUCores = cores
		}
	}

	memTotal, memAvail, swapTotal, swapUsed := parseMeminfo(sections["MEM"])
	m.MemTotal = memTotal
	m.SwapTotal = swapTotal
	m.SwapUsed = swapUsed
	if memTotal > memAvail {
		m.MemUsed = memTotal - memAvail
	}

	m.Disks = parseDisks(sections["DISK"])
	m.Processes = parseProcesses(sections["PROC"])
	return m, nil
}
