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
const helpMsg = `adduid 数字：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
deluid 数字：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
addrecuid 数字：自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
delrecuid 数字：取消自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
addrstuid 数字：下载指定主播的直播同时将直播流推向本地UDP端口，节省边下载边观看同一直播的流量，但播放器的播放画面可能有点卡顿，数字为主播的uid（在主播的网页版个人主页查看），需要事先设置自动下载指定主播的直播
delrstuid 数字：取消下载指定主播的直播同时将直播流推向本地端口，数字为主播的uid（在主播的网页版个人主页查看）
startrecord 数字：临时下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
startrecrst 数字：临时下载指定主播的直播并将直播流推向本地UDP端口，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
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
			log.Println("输入处理发生错误")
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := strings.Fields(scanner.Text())
		if len(cmd) == 1 {
			switch cmd[0] {
			case "help":
				fmt.Println(helpMsg)
			case "quit":
				fmt.Println("正在准备退出，请等待...")
				chMutex.Lock()
				ch := chMap[0]
				chMutex.Unlock()
				q := controlMsg{c: quit}
				ch <- q
				return
			default:
				printErr()
			}
		} else if len(cmd) == 2 {
			switch cmd[0] {
			case "adduid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addNotify(uint(uid))
			case "deluid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delNotify(uint(uid))
			case "addrecuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addRecord(uint(uid))
			case "delrecuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delRecord(uint(uid))
			case "addrstuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addRestream(uint(uid))
			case "delrstuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delRestream(uint(uid))
			case "startrecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				startRec(uint(uid), false)
			case "startrecrst":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				startRec(uint(uid), true)
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
