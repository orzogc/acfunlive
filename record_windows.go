//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"unicode/utf8"
)

// 查看并获取 FFmpeg 的位置
func getFFmpeg() (ffmpegFile string) {
	// windows 下 ffmpeg.exe 需要和本程序 exe 放在同一文件夹下
	ffmpegFile = filepath.Join(exeDir, "ffmpeg.exe")
	if _, err := os.Stat(ffmpegFile); os.IsNotExist(err) {
		lPrintErr("ffmpeg.exe 需要和本程序放在同一文件夹下")
		return ""
	}
	return ffmpegFile
}

// 转换文件名和限制文件名长度，添加程序所在文件夹的路径
func transFilename(filename string) string {
	// 转换文件名不允许的特殊字符
	re := regexp.MustCompile(`[<>:"/\\|?*\r\n]`)
	filename = re.ReplaceAllString(filename, "-")
	outFilename := filepath.Join(*recordDir, filename)
	// windows 下全路径文件名不能过长
	if utf8.RuneCountInString(outFilename) > 255 {
		lPrintErr("全路径文件名太长，取消下载")
		desktopNotify("全路径文件名太长，取消下载")
		return ""
	}
	return outFilename
}

// Windows 下启用 GUI 时隐藏 FFmpeg 的 cmd 窗口
func hideCmdWindow(cmd *exec.Cmd) {
	if !*isNoGUI {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}
