// 命令输入相关
package main

import (
	"bufio"
	"encoding/json"
	"os"
)

// 帮助信息
const helpMsg = `listlive：列出正在直播的主播
listrecord：列出正在下载的直播视频
listdanmu：列出正在下载的直播弹幕
startweb：启动web服务
stopweb：停止web服务
startcoolq：使用酷Q发送直播通知到指定QQ或QQ群，需要事先设置并启动酷Q
addnotify uid：订阅指定主播的开播提醒，uid在主播的网页版个人主页查看
delnotify uid：取消订阅指定主播的开播提醒
addrecord uid：自动下载指定主播的直播视频
delrecord uid：取消自动下载指定主播的直播视频
adddanmu uid：自动下载指定主播的直播弹幕
deldanmu uid：取消自动下载指定主播的直播弹幕
getdlurl uid：查看指定主播是否在直播，如在直播输出其直播源地址
addqq uid QQ号：设置将指定主播的开播提醒发送到指定QQ号
delqq uid：取消设置将指定主播的开播提醒发送到QQ
addqqgroup uid QQ群号：设置将指定主播的开播提醒发送到指定QQ群号
delqqgroup uid：取消设置将指定主播的开播提醒发送到QQ群
startrecord uid：临时下载指定主播的直播视频，如果没有设置自动下载该主播的直播视频，这次为一次性的下载
stoprecord uid：正在下载指定主播的直播视频时取消下载
startdanmu uid：临时下载指定主播的直播弹幕，如果没有设置自动下载该主播的直播弹幕，这次为一次性的下载
stopdanmu uid：正在下载指定主播的直播弹幕时取消下载
startrecdan uid：临时下载指定主播的直播视频和弹幕），如果没有设置自动下载该主播的直播视频和弹幕，这次为一次性的下载
stoprecdan uid：正在下载指定主播的直播视频和弹幕时取消下载
quit：退出本程序，退出需要等待半分钟左右
help：输出本帮助信息`

// 正在直播的主播
type streaming streamer

// 实现json.Marshaler接口
func (s streaming) MarshalJSON() ([]byte, error) {
	type sJSON struct {
		UID              int
		Name, Title, URL string
	}
	sj := sJSON{UID: s.UID, Name: s.Name, Title: streamer(s).getTitle(), URL: streamer(s).getURL()}
	return json.Marshal(sj)
}

// 列出正在直播的主播
func listLive() (streamings []streaming) {
	lPrintln("正在直播的主播：")
	streamers.Lock()
	defer streamers.Unlock()
	for _, s := range streamers.crt {
		if s.isLiveOn() {
			lPrintln(s.longID() + "：" + s.getTitle() + " " + s.getURL())
			streamings = append(streamings, streaming(s))
		}
	}

	return streamings
}

// 列出正在下载的直播视频
func listRecord() (recordings []streaming) {
	lPrintln("正在下载的直播视频：")
	msgMap.Lock()
	defer msgMap.Unlock()
	for uid, m := range msgMap.msg {
		if m.recording {
			s := streamer{UID: uid, Name: getName(uid)}
			lPrintln(s.longID() + "：" + s.getTitle() + " " + s.getURL())
			recordings = append(recordings, streaming(s))
		}
	}

	return recordings
}

// 列出正在下载的直播弹幕
func listDanmu() (danmu []streaming) {
	lPrintln("正在下载的直播弹幕：")
	msgMap.Lock()
	defer msgMap.Unlock()
	for uid, m := range msgMap.msg {
		if m.danmuCancel != nil {
			s := streamer{UID: uid, Name: getName(uid)}
			lPrintln(s.longID() + "：" + s.getTitle() + " " + s.getURL())
			danmu = append(danmu, streaming(s))
		}
	}

	return danmu
}

// 通知main()退出程序
func quitRun() {
	lPrintln("正在准备退出，请等待...")
	q := controlMsg{c: quit}
	msgMap.Lock()
	defer msgMap.Unlock()
	msgMap.msg[0].ch <- q
}

// 处理输入
func handleInput() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in handleInput(), the error is:", err)
			lPrintln("输入处理发生错误，尝试重启输入处理")
			go handleInput()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		handleAllCmd(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		lPrintln("Reading standard input err:", err)
	}
}
