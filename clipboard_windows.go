//go:build windows

package main

import (
	"errors"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WebView2 无法把资源管理器里 Ctrl+C 复制的文件列表暴露给网页，
// 这里直接读 Windows 剪贴板的 CF_HDROP（文件列表）。
const cfHDrop = 15

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	shell32               = windows.NewLazySystemDLL("shell32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procIsClipboardFormat = user32.NewProc("IsClipboardFormatAvailable")
	procDragQueryFileW    = shell32.NewProc("DragQueryFileW")
)

// readClipboardFiles returns the local files/folders currently in the Windows
// clipboard. An empty slice means the clipboard holds no files.
func readClipboardFiles() ([]string, error) {
	if available, _, _ := procIsClipboardFormat.Call(cfHDrop); available == 0 {
		return nil, nil
	}

	var opened uintptr
	for attempt := 0; attempt < 5; attempt++ {
		opened, _, _ = procOpenClipboard.Call(0)
		if opened != 0 {
			break
		}
		// 剪贴板可能被其他程序短暂占用，稍等重试。
		time.Sleep(40 * time.Millisecond)
	}
	if opened == 0 {
		return nil, errors.New("剪贴板正被其他程序占用，请稍后重试")
	}
	defer procCloseClipboard.Call()

	handle, _, _ := procGetClipboardData.Call(cfHDrop)
	if handle == 0 {
		return nil, nil
	}

	count, _, _ := procDragQueryFileW.Call(handle, 0xFFFFFFFF, 0, 0)
	if count == 0 {
		return nil, nil
	}

	files := make([]string, 0, count)
	for index := uintptr(0); index < count; index++ {
		length, _, _ := procDragQueryFileW.Call(handle, index, 0, 0)
		if length == 0 {
			continue
		}
		buf := make([]uint16, length+1)
		procDragQueryFileW.Call(handle, index, uintptr(unsafe.Pointer(&buf[0])), length+1)
		files = append(files, windows.UTF16ToString(buf))
	}
	return files, nil
}
