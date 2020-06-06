// 通知相关
package main

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

const logoFile = "acfun_logo.ico"

var logoFileLocation string

// 添加订阅指定uid的直播提醒
func addNotify(uid uint) {
	isExist := false
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			isExist = true
			if s.Notify {
				fmt.Println("已经订阅过" + s.ID + "的开播提醒")
			} else {
				streamers[i].Notify = true
				fmt.Println("成功订阅" + s.ID + "的开播提醒")
			}
		}
	}
	sMutex.Unlock()

	if !isExist {
		id := getID(uid)
		if id == "" {
			fmt.Println("不存在这个用户")
			return
		}

		newStreamer := streamer{UID: uid, ID: id, Notify: true, Record: false, Restream: false}
		sMutex.Lock()
		streamers = append(streamers, newStreamer)
		sMutex.Unlock()
		fmt.Println("成功订阅" + id + "的开播提醒")
	}

	saveConfig()
}

// 取消订阅指定uid的直播提醒
func delNotify(uid uint) {
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			if s.Record {
				streamers[i].Notify = false
			} else {
				deleteStreamer(uid)
			}
			fmt.Println("成功取消订阅" + s.ID + "的开播提醒")
		}
	}
	sMutex.Unlock()

	saveConfig()
}

// 桌面通知
func desktopNotify(notifyWords string) {
	beeep.Alert("AcFun直播通知", notifyWords, logoFileLocation)
}
