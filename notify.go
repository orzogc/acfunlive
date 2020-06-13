// 通知相关
package main

import (
	"github.com/gen2brain/beeep"
)

const logoFile = "acfunlogo.ico"

var logoFileLocation string

// 添加订阅指定uid的直播提醒
func addNotify(uid uint) bool {
	isExist := false
	streamers.mu.Lock()
	for i, s := range streamers.current {
		if s.UID == uid {
			isExist = true
			if s.Notify {
				timePrintln("已经订阅过" + s.ID + "的开播提醒")
			} else {
				streamers.current[i].Notify = true
				timePrintln("成功订阅" + s.ID + "的开播提醒")
			}
		}
	}
	streamers.mu.Unlock()

	if !isExist {
		id := getID(uid)
		if id == "" {
			timePrintln("不存在uid为" + uidStr(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, ID: id, Notify: true, Record: false}
		streamers.mu.Lock()
		streamers.current = append(streamers.current, newStreamer)
		streamers.mu.Unlock()
		timePrintln("成功订阅" + id + "的开播提醒")
	}

	saveConfig()
	return true
}

// 取消订阅指定uid的直播提醒
func delNotify(uid uint) bool {
	streamers.mu.Lock()
	for i, s := range streamers.current {
		if s.UID == uid {
			if s.Record {
				streamers.current[i].Notify = false
			} else {
				deleteStreamer(uid)
			}
			timePrintln("成功取消订阅" + s.ID + "的开播提醒")
		}
	}
	streamers.mu.Unlock()

	saveConfig()
	return true
}

// 桌面通知
func desktopNotify(notifyWords string) {
	beeep.Alert("AcFun直播通知", notifyWords, logoFileLocation)
}
