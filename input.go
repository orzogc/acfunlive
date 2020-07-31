// 命令输入相关
package main

import (
	"bufio"
	"encoding/json"
	"os"
)

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
		if m.isRecording {
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
	mainCh <- q
}

// 处理输入
func handleInput() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in handleInput(), the error is:", err)
			lPrintErr("输入处理发生错误，尝试重启输入处理")
			go handleInput()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		handleAllCmd(scanner.Text())
	}
	err := scanner.Err()
	checkErr(err)
}
