package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

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
