// 通知相关
package main

import (
	"github.com/gen2brain/beeep"
)

const logoFile = "acfunlogo.ico"

var logoFileLocation string

// 添加订阅指定uid的直播提醒
func addNotify(uid int) bool {
	isExist := false
	streamers.mu.Lock()
	if s, ok := streamers.crt[uid]; ok {
		isExist = true
		if s.Notify {
			lPrintln("已经订阅过" + s.Name + "的开播提醒")
		} else {
			s.Notify = true
			sets(s)
			lPrintln("成功订阅" + s.Name + "的开播提醒")
		}
	}
	streamers.mu.Unlock()

	if !isExist {
		name := getName(uid)
		if name == "" {
			lPrintln("不存在uid为" + itoa(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, Name: name, Notify: true, Record: false}
		streamers.mu.Lock()
		sets(newStreamer)
		streamers.mu.Unlock()
		lPrintln("成功订阅" + name + "的开播提醒")
	}

	saveConfig()
	return true
}

// 取消订阅指定uid的直播提醒
func delNotify(uid int) bool {
	streamers.mu.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Record {
			s.Notify = false
			sets(s)
		} else {
			deleteStreamer(uid)
		}
		lPrintln("成功取消订阅" + s.Name + "的开播提醒")
	} else {
		lPrintln("没有订阅过uid为" + itoa(uid) + "的主播的开播提醒")
	}
	streamers.mu.Unlock()

	saveConfig()
	return true
}

// 桌面通知
func desktopNotify(notifyWords string) {
	beeep.Alert("AcFun直播通知", notifyWords, logoFileLocation)
}
