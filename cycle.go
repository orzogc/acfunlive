// 循环相关
package main

import (
	"time"
)

// 处理管道信号
func (s streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		lPrintln("重启监听" + s.longID() + "的直播状态")
		ch, _ := chMap.Load(0)
		ch.(chan controlMsg) <- msg
	case stopCycle:
		lPrintln("删除" + s.longID())
		chMap.Delete(s.UID)
	case quit:
		//timePrintln("正在退出" + s.longID() + "的循环")
	default:
		lPrintln("未知的controlMsg：", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in cycle(), the error is:", err)
			lPrintln(s.longID() + "的循环处理发生错误，尝试重启循环")

			restart := controlMsg{s: s, c: startCycle}
			ch, _ := chMap.Load(0)
			ch.(chan controlMsg) <- restart
		}
	}()

	ch, _ := chMap.Load(s.UID)

	// 设置文件里有该主播，但是不通知不下载
	if !(s.Notify || s.Record) {
		for {
			msg := <-ch.(chan controlMsg)
			s.handleMsg(msg)
			return
		}
	}

	lPrintln("开始监听" + s.longID() + "的直播状态")

	isLive := false
	for {
		select {
		case msg := <-ch.(chan controlMsg):
			s.handleMsg(msg)
			return
		default:
			if s.isLiveOn() {
				if !isLive {
					isLive = true
					title := s.getTitle()
					lPrintln(s.longID() + "正在直播：" + title)
					lPrintln(s.ID + "的直播观看地址：" + livePage + s.uidStr())

					if s.Notify {
						desktopNotify(s.ID + "正在直播：" + title)
					}
					if s.Record {
						// 直播短时间内重启的情况下，通常上一次的直播下载的退出会比较慢
						rec, ok := recordMap.Load(s.UID)
						if ok {
							// 如果设置被修改，不重启已有的下载
							modified, _ := modify.Load(s.UID)
							if !modified.(bool) {
								go s.recordLive()
								rec.(record).ch <- stopRecord
								danglingRec.mu.Lock()
								danglingRec.records = append(danglingRec.records, rec.(record))
								danglingRec.mu.Unlock()
								/*
									io.WriteString(rec.stdin, "q")
									time.Sleep(20 * time.Second)
									rec.cancel()
								*/
							}
						} else {
							// 没有下载时就直接启动下载
							go s.recordLive()
						}
					} else {
						lPrintln("如果要临时下载" + s.ID + "的直播，可以运行startrecord " + s.uidStr())
					}
				}
			} else {
				if isLive {
					isLive = false
					lPrintln(s.longID() + "已经下播")
					if s.Notify {
						desktopNotify(s.ID + "已经下播")
					}
					if s.Record {
						rec, ok := recordMap.Load(s.UID)
						if ok {
							rec.(record).ch <- liveOff
						}
					}

				}
			}

			modified, _ := modify.Load(s.UID)
			if modified.(bool) {
				modify.Store(s.UID, false)
			}
		}

		time.Sleep(time.Second)
	}
}

// 完成对cycle()的初始化
func (s streamer) initCycle() {
	controlCh := make(chan controlMsg, 20)
	chMap.Store(s.UID, controlCh)
	// 初始化modify，因为sync.Map不会自动初始化
	modify.Store(s.UID, false)
	s.cycle()
}
