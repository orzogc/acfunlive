//go:build !windows

package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
)

// 查看并获取 FFmpeg 的位置
func getFFmpeg() (ffmpegFile string) {
	ffmpegFile = "ffmpeg"
	// linux 和 macOS 下确认有没有安装 FFmpeg
	if _, err := exec.LookPath(ffmpegFile); err != nil {
		lPrintErr("系统没有安装 FFmpeg")
		return ""
	}
	return ffmpegFile
}

// 转换文件名和限制文件名长度，添加程序所在文件夹的路径
func transFilename(filename string) string {
	// 转换文件名不允许的特殊字符
	re := regexp.MustCompile(`[<>:"/\\|?*\r\n]`)
	filename = re.ReplaceAllString(filename, "-")
	// linux 和 macOS 下限制文件名长度
	if len(filename) >= 250 {
		filename = filename[:250]
	}
	return filepath.Join(*recordDir, filename)
}

// Windows 下启用 GUI 时隐藏 FFmpeg 的 cmd 窗口
func hideCmdWindow(cmd *exec.Cmd) {
}
