// 循环相关
package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"
)

// 处理管道信号
func (s streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		logPrintln("重启监听" + s.ID + "（" + s.uidStr() + "）" + "的直播状态")
		chMutex.Lock()
		ch := chMap[0]
		chMutex.Unlock()
		ch <- msg
	case stopCycle:
		logPrintln("删除" + s.ID)
		chMutex.Lock()
		delete(chMap, s.UID)
		chMutex.Unlock()
	case quit:
		logPrintln("正在退出" + s.ID + "（" + s.uidStr() + "）" + "的循环")
	default:
		log.Println("未知controlMsg：", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in cycle(), the error is:", err)
			log.Println(s.ID + "（" + s.uidStr() + "）" + "的循环处理发生错误")

			if s.Record {
				recMutex.Lock()
				rec, ok := recordMap[s.UID]
				recMutex.Unlock()
				if ok {
					fmt.Println("开始结束下载" + s.ID + "的直播")
					rec.ch <- stopRecord
					io.WriteString(rec.stdin, "q")
					time.Sleep(20 * time.Second)
					rec.cancel()
				}
			}

			restart := controlMsg{s: s, c: startCycle}
			chMutex.Lock()
			ch := chMap[0]
			chMutex.Unlock()
			ch <- restart
		}
	}()

	const livePage = "https://live.acfun.cn/live/"

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

	logPrintln("开始监听" + s.ID + "（" + s.uidStr() + "）" + "的直播状态")

	recCh := make(chan control, 5)

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
					logPrintln(s.ID + "（" + s.uidStr() + "）" + "正在直播：\n" + title)
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
					fmt.Println(s.ID + "的直播观看地址：\n" + livePage + s.uidStr())
					fmt.Println(s.ID + "直播源的hls和flv地址分别是：\n" + hlsURL + "\n" + flvURL)

					if s.Notify {
						desktopNotify(s.ID + "正在直播")
					}
					if s.Record {
						// 下载直播源
						go s.recordLive(recCh)
					}
				}
			} else {
				if isLive {
					logPrintln(s.ID + "（" + s.uidStr() + "）" + "已经下播")
					if s.Notify {
						desktopNotify(s.ID + "已经下播")
					}

					recCh <- liveOff
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
