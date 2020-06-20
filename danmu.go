// 弹幕下载相关
package main

import "github.com/orzogc/acfundanmu"

var subConfigs = map[int]acfundanmu.SubConfig{
	540:  {PlayResX: 960, PlayResY: 540, FontSize: 30},
	720:  {PlayResX: 1280, PlayResY: 720, FontSize: 40},
	1080: {PlayResX: 1920, PlayResY: 1080, FontSize: 60},
}

// 设置自动下载指定主播的直播弹幕
func addDanmu(uid int) bool {
	isExist := false
	streamers.mu.Lock()
	if s, ok := streamers.crt[uid]; ok {
		isExist = true
		if s.Danmu {
			lPrintln("已经设置过自动下载" + s.Name + "的直播弹幕")
		} else {
			s.Danmu = true
			sets(s)
			lPrintln("成功设置自动下载" + s.Name + "的直播弹幕")
		}
	}
	streamers.mu.Unlock()

	if !isExist {
		name := getName(uid)
		if name == "" {
			lPrintln("不存在uid为" + itoa(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, Name: name, Notify: false, Record: false, Danmu: true}
		streamers.mu.Lock()
		sets(newStreamer)
		streamers.mu.Unlock()
		lPrintln("成功设置自动下载" + name + "的直播弹幕")
	}

	saveConfig()
	return true
}

// 取消自动下载指定主播的直播弹幕
func delDanmu(uid int) bool {
	streamers.mu.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Notify || s.Record {
			s.Danmu = false
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
