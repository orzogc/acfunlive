// 循环相关
package main

import (
	"io"
	"time"
)

const livePage = "https://live.acfun.cn/live/"

// 处理管道信号
func (s streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		timePrintln("重启监听" + s.longID() + "的直播状态")
		chMutex.Lock()
		ch := chMap[0]
		chMutex.Unlock()
		ch <- msg
	case stopCycle:
		timePrintln("删除" + s.longID())
		chMutex.Lock()
		delete(chMap, s.UID)
		chMutex.Unlock()
	case quit:
		timePrintln("正在退出" + s.longID() + "的循环")
	default:
		timePrintln("未知的controlMsg：", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle() {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in cycle(), the error is:", err)
			timePrintln(s.longID() + "的循环处理发生错误，尝试重启循环")

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

	timePrintln("开始监听" + s.longID() + "的直播状态")

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
					timePrintln(s.longID() + "正在直播：" + title)
					logger.Println(s.ID + "的直播观看地址：" + livePage + s.uidStr())

					if s.Notify {
						desktopNotify(s.ID + "正在直播：" + title)
					}
					if s.Record {
						// 直播短时间内重启的情况下，通常上一次的直播下载的退出会比较慢
						recMutex.Lock()
						rec, ok := recordMap[s.UID]
						recMutex.Unlock()
						if ok {
							// 如果设置被修改，不重启已有的下载
							moMutex.Lock()
							modified := modify[s.UID]
							moMutex.Unlock()
							if !modified {
								go s.recordLive()
								rec.ch <- stopRecord
								io.WriteString(rec.stdin, "q")
								time.Sleep(20 * time.Second)
								rec.cancel()
							}
						} else {
							// 没有下载时就直接启动下载
							go s.recordLive()
						}
					} else {
						logger.Println("如果要临时下载" + s.ID + "的直播，可以运行startrecord " + s.uidStr())
					}
				}
			} else {
				if isLive {
					timePrintln(s.longID() + "已经下播")
					if s.Notify {
						desktopNotify(s.ID + "已经下播")
					}
					if s.Record {
						recMutex.Lock()
						rec, ok := recordMap[s.UID]
						recMutex.Unlock()
						if ok {
							rec.ch <- liveOff
						}
					}

				}
				isLive = false
			}

			moMutex.Lock()
			if modify[s.UID] {
				modify[s.UID] = false
			}
			moMutex.Unlock()
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
