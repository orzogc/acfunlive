// 循环相关
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const livePage = "https://live.acfun.cn/live/"

// 处理管道信号
func (s streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		logPrintln("重启监听" + s.longID() + "的直播状态")
		chMutex.Lock()
		ch := chMap[0]
		chMutex.Unlock()
		ch <- msg
	case stopCycle:
		logPrintln("删除" + s.longID())
		chMutex.Lock()
		delete(chMap, s.UID)
		chMutex.Unlock()
	case quit:
		logPrintln("正在退出" + s.longID() + "的循环")
	default:
		log.Println("未知的controlMsg：", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in cycle(), the error is:", err)
			log.Println(s.longID() + "的循环处理发生错误，尝试重启循环")

			restart := controlMsg{s: s, c: startCycle}
			chMutex.Lock()
			ch := chMap[0]
			chMutex.Unlock()
			ch <- restart
		}
	}()

	chMutex.Lock()
	ch := chMap[s.UID]
	chMutex.Unlock()

	// 设置文件里有该主播，但是不通知不下载
	if !(s.Notify || s.Record) {
		for {
			msg := <-ch
			s.handleMsg(msg)
			return
		}
	}

	logPrintln("开始监听" + s.longID() + "的直播状态")

	isLive := false
	for {
		select {
		case msg := <-ch:
			s.handleMsg(msg)
			return
		default:
			if s.isLiveOn() {
				if !isLive {
					isLive = true
					title := s.getTitle()
					logPrintln(s.longID() + "正在直播：\n" + title)
					fmt.Println(s.ID + "的直播观看地址：\n" + livePage + s.uidStr())
					/*
						hlsURL, flvURL := s.getStreamURL()
						if flvURL == "" {
							log.Println("无法获取" + s.ID + "的直播源，尝试重启循环")
							restart := controlMsg{s: s, c: startCycle}
							chMutex.Lock()
							ch := chMap[0]
							chMutex.Unlock()
							ch <- restart
							return
						}
						fmt.Println(s.ID + "直播源的hls和flv地址分别是：\n" + hlsURL + "\n" + flvURL)
					*/

					if s.Notify {
						desktopNotify(s.ID + "正在直播")
					}
					if s.Record {
						// 查看下载是否已经启动
						recMutex.Lock()
						_, ok := recordMap[s.UID]
						if !ok {
							// 没有下载时启动下载直播源，有下载时recordLive()会自行处理
							go s.recordLive()
						}
						recMutex.Unlock()
					} else {
						fmt.Println("如果要临时下载" + s.ID + "的直播，可以运行startrecord " + s.uidStr())
					}
				}
			} else {
				if isLive {
					logPrintln(s.longID() + "已经下播")
					if s.Notify {
						desktopNotify(s.ID + "已经下播")
					}
				}
				isLive = false
			}

			// 大约每二十几秒获取一次主播的直播状态
			rand.Seed(time.Now().UnixNano())
			min := 20
			max := 30
			duration := rand.Intn(max-min) + min
			time.Sleep(time.Duration(duration) * time.Second)
		}
	}
}

// 完成对cycle()的初始化
func (s streamer) initCycle() {
	controlCh := make(chan controlMsg, 20)
	chMutex.Lock()
	chMap[s.UID] = controlCh
	chMutex.Unlock()
	s.cycle()
}
