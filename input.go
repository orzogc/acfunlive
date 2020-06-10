// 命令输入相关
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// 帮助信息
const helpMsg = `listlive：列出正在直播的主播
listrecord：列出正在下载的直播
addnotify 数字：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
delnotify 数字：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
addrecord 数字：自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
delrecord 数字：取消自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
getdlurl 数字：查看指定主播是否在直播，如在直播输出其直播源地址，数字为主播的uid（在主播的网页版个人主页查看）
startrecord 数字：临时下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
stoprecord 数字：正在下载指定主播的直播时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
quit：退出本程序，退出需要等待半分钟
help：本帮助信息`

// 打印错误命令信息
func printErr() {
	fmt.Println("请输入正确的命令，输入help查看全部命令的解释")
}

// 处理输入
func handleInput() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in handleInput(), the error is:", err)
			log.Println("输入处理发生错误，尝试重启输入处理")
			go handleInput()
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := strings.Fields(scanner.Text())
		if len(cmd) == 1 {
			switch cmd[0] {
			case "listlive":
				fmt.Println("正在直播的主播：")
				for _, s := range streamers {
					if s.isLiveOn() {
						fmt.Println(s.longID() + "：" + livePage + s.uidStr())
					}
				}
			case "listrecord":
				fmt.Println("正在下载的直播：")
				for uid := range recordMap {
					s := streamer{UID: uid, ID: getID(uid)}
					fmt.Println(s.longID() + "：" + s.getTitle())
				}
			case "quit":
				fmt.Println("正在准备退出，请等待...")
				chMutex.Lock()
				ch := chMap[0]
				chMutex.Unlock()
				q := controlMsg{c: quit}
				ch <- q
				return
			case "help":
				fmt.Println(helpMsg)
			default:
				printErr()
			}
		} else if len(cmd) == 2 {
			switch cmd[0] {
			case "addnotify":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addNotify(uint(uid))
			case "delnotify":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delNotify(uint(uid))
			case "addrecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addRecord(uint(uid))
			case "delrecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delRecord(uint(uid))
			case "getdlurl":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				printStreamURL(uint(uid))
			case "startrecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				startRec(uint(uid))
			case "stoprecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				stopRec(uint(uid))
			default:
				printErr()
			}
		} else {
			printErr()
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("Reading standard input err:", err)
	}
}
