// 命令输入相关
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// 帮助信息
const helpMsg = `listlive：列出正在直播的主播
listrecord：列出正在下载的直播视频
listdanmu：列出正在下载的直播弹幕
startweb：启动web服务
stopweb：停止web服务
addnotify 数字：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
delnotify 数字：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
addrecord 数字：自动下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看）
delrecord 数字：取消自动下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看）
adddanmu 数字：自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
deldanmu 数字：取消自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
getdlurl 数字：查看指定主播是否在直播，如在直播输出其直播源地址，数字为主播的uid（在主播的网页版个人主页查看）
startrecord 数字：临时下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播视频，这次为一次性的下载
stoprecord 数字：正在下载指定主播的直播视频时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
startdanmu 数字：临时下载指定主播的直播弹幕，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播弹幕，这次为一次性的下载
stopdanmu 数字：正在下载指定主播的直播弹幕时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
startrecdan 数字：临时下载指定主播的直播视频和弹幕，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播视频和弹幕，这次为一次性的下载
stoprecdan 数字：正在下载指定主播的直播视频和弹幕时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
quit：退出本程序，退出需要等待半分钟左右
help：本帮助信息`

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

// 打印错误命令信息
func printErr() {
	lPrintln("请输入正确的命令，输入 help 查看全部命令的解释")
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
		cmd := strings.Fields(scanner.Text())
		if len(cmd) == 1 {
			switch cmd[0] {
			case "startweb":
				if !*isWebServer {
					*isWebServer = true
					lPrintln("启动web服务，现在可以通过 " + address(config.WebPort) + " 来发送命令")
					go httpServer()
				} else {
					lPrintln("已经启动过web服务")
				}
			case "stopweb":
				if *isWebServer {
					*isWebServer = false
					lPrintln("正在停止web服务")
					srv.Shutdown(context.TODO())
				} else {
					lPrintln("没有启动web服务")
				}
			case "help":
				fmt.Println(helpMsg)
			default:
				handleCmd(cmd[0])
			}
		} else if len(cmd) == 2 {
			uid, err := atoi(cmd[1])
			if err != nil {
				printErr()
			} else {
				handleCmdUID(cmd[0], uid)
			}
		} else {
			printErr()
		}
	}
	if err := scanner.Err(); err != nil {
		lPrintln("Reading standard input err:", err)
	}
}
