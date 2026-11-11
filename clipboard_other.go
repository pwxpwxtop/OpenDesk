//go:build !windows

package main

import "errors"

// 非 Windows 平台暂不支持从剪贴板粘贴本地文件。
func readClipboardFiles() ([]string, error) {
	return nil, errors.New("当前平台不支持从剪贴板粘贴文件")
}
