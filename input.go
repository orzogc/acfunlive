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
listrecord：列出正在下载的直播
startweb：启动web服务
stopweb：停止web服务
addnotify 数字：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
delnotify 数字：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
addrecord 数字：自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
delrecord 数字：取消自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
adddanmu 数字：自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
deldanmu 数字：取消自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
getdlurl 数字：查看指定主播是否在直播，如在直播输出其直播源地址，数字为主播的uid（在主播的网页版个人主页查看）
startrecord 数字：临时下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
stoprecord 数字：正在下载指定主播的直播时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
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
	streamers.mu.Lock()
	defer streamers.mu.Unlock()
	for _, s := range streamers.crt {
		if s.isLiveOn() {
			lPrintln(s.longID() + "：" + s.getTitle() + " " + s.getURL())
			streamings = append(streamings, streaming(s))
		}
	}

	return streamings
}

// 列出正在下载的直播
func listRecord() (recordings []streaming) {
	lPrintln("正在下载的直播：")
	msgMap.mu.Lock()
	defer msgMap.mu.Unlock()
	for uid, m := range msgMap.msg {
		if m.recording {
			s := streamer{UID: uid, Name: getName(uid)}
			lPrintln(s.longID() + "：" + s.getTitle() + " " + s.getURL())
			recordings = append(recordings, streaming(s))
		}
	}

	return recordings
}

// 通知main()退出程序
func quitRun() {
	lPrintln("正在准备退出，请等待...")
	q := controlMsg{c: quit}
	msgMap.mu.Lock()
	defer msgMap.mu.Unlock()
	msgMap.msg[0].ch <- q
}

// 打印错误命令信息
func printErr() {
	lPrintln("请输入正确的命令，输入help查看全部命令的解释")
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
			case "listlive":
				listLive()
			case "listrecord":
				listRecord()
			case "startweb":
				if !*isWebServer {
					*isWebServer = true
					lPrintln("启动web服务，现在可以通过 http://localhost" + port + " 来发送命令")
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
			case "quit":
				quitRun()
				return
			case "help":
				fmt.Println(helpMsg)
			default:
				printErr()
			}
		} else if len(cmd) == 2 {
			switch cmd[0] {
			case "addnotify":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				addNotify(uid)
			case "delnotify":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				delNotify(uid)
			case "addrecord":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				addRecord(uid)
			case "delrecord":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				delRecord(uid)
			case "adddanmu":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				addDanmu(uid)
			case "deldanmu":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				delDanmu(uid)
			case "getdlurl":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				printStreamURL(uid)
			case "startrecord":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				startRec(uid)
			case "stoprecord":
				uid, err := atoi(cmd[1])
				if err != nil {
					printErr()
					break
				}
				stopRec(uid)
			default:
				printErr()
			}
		} else {
			printErr()
		}
	}
	if err := scanner.Err(); err != nil {
		lPrintln("Reading standard input err:", err)
	}
}
