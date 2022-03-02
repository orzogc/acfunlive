// 通知相关
package main

import (
	"github.com/gen2brain/beeep"
)

// logo文件名字
const logoFile = "acfunlogo.ico"

// logo文件位置
var logoFileLocation string

type notify struct {
	NotifyOn     bool `json:"notifyOn"`     // 通知开播
	NotifyOff    bool `json:"notifyOff"`    // 通知下播
	NotifyRecord bool `json:"notifyRecord"` // 通知下载直播视频相关
	NotifyDanmu  bool `json:"notifyDanmu"`  // 通知下载直播弹幕相关
}

// 桌面通知
func desktopNotify(text string) {
	beeep.Alert("AcFun直播通知", text, logoFileLocation)
}
