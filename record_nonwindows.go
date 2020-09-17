// +build !windows

package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
)

// 查看并获取FFmpeg的位置
func getFFmpeg() (ffmpegFile string) {
	ffmpegFile = "ffmpeg"
	// linux和macOS下确认有没有安装FFmpeg
	if _, err := exec.LookPath(ffmpegFile); err != nil {
		lPrintErr("系统没有安装FFmpeg")
		return ""
	}
	return ffmpegFile
}

// 转换文件名和限制文件名长度
func transFilename(filename string) string {
	// 转换文件名不允许的特殊字符
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	filename = re.ReplaceAllString(filename, "-")
	// linux和macOS下限制文件名长度
	if len(filename) >= 250 {
		filename = filename[:250]
	}
	return filepath.Join(exeDir, filename)
}

// Windows下启用GUI时隐藏FFmpeg的cmd窗口
func hideCmdWindow(cmd *exec.Cmd) {
}
