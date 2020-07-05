// 通知相关
package main

import (
	"github.com/gen2brain/beeep"
)

// logo文件名字
const logoFile = "acfunlogo.ico"

// logo文件位置
var logoFileLocation string

// 添加订阅指定uid的直播提醒
func addNotify(uid int) bool {
	isExist := false
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		isExist = true
		if s.Notify {
			lPrintWarn("已经订阅过" + s.Name + "的开播提醒")
		} else {
			s.Notify = true
			sets(s)
			lPrintln("成功订阅" + s.Name + "的开播提醒")
		}
	}
	streamers.Unlock()

	if !isExist {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, Name: name, Notify: true}
		streamers.Lock()
		sets(newStreamer)
		streamers.Unlock()
		lPrintln("成功订阅" + name + "的开播提醒")
	}

	saveLiveConfig()
	return true
}

// 取消订阅指定uid的直播提醒
func delNotify(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Record || s.Danmu {
			s.Notify = false
			sets(s)
		} else {
			deleteStreamer(uid)
		}
		lPrintln("成功取消订阅" + s.Name + "的开播提醒")
	} else {
		lPrintWarn("没有订阅过uid为" + itoa(uid) + "的主播的开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 桌面通知
func desktopNotify(text string) {
	beeep.Alert("AcFun直播通知", text, logoFileLocation)
}
