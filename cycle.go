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
		msgMap.Lock()
		defer msgMap.Unlock()
		msgMap.msg[0].ch <- msg
	case stopCycle:
		lPrintln("退出" + s.longID() + "的循环")
		deleteMsg(s.UID)
	case quit:
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
			msgMap.Lock()
			msgMap.msg[0].ch <- restart
			msgMap.Unlock()
		}
	}()

	ch := make(chan controlMsg, 20)
	msgMap.Lock()
	if m, ok := msgMap.msg[s.UID]; ok {
		m.ch = ch
	} else {
		msgMap.msg[s.UID] = &sMsg{ch: ch}
	}
	msgMap.Unlock()

	// 设置文件里有该主播，但是不通知不下载
	if !(s.Notify || s.Record || s.Danmu) {
		for {
			msg := <-ch
			s.handleMsg(msg)
			return
		}
	}

	lPrintln("开始监听" + s.longID() + "的直播状态")

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
					lPrintln(s.longID() + "正在直播：" + title)
					lPrintln(s.Name + "的直播观看地址：" + s.getURL())

					if s.Notify {
						desktopNotify(s.Name + "正在直播：" + title)
						s.sendCoolq(s.Name + "正在直播：" + title)
					}
					if s.Record {
						msgMap.Lock()
						// 直播短时间内重启的情况下，通常上一次的直播视频下载的退出会比较慢
						if m := msgMap.msg[s.UID]; m.recording {
							// 如果设置被修改，不重启已有的下载
							if !m.modify {
								m.rec.ch <- stopRecord
								danglingRec.Lock()
								danglingRec.records = append(danglingRec.records, m.rec)
								danglingRec.Unlock()
								go s.recordLive(getFFmpeg(), s.Danmu)
							}
						} else {
							// 没有下载时就直接启动下载
							go s.recordLive(getFFmpeg(), s.Danmu)
						}
						msgMap.Unlock()
					} else {
						lPrintln("如果要临时下载" + s.Name + "的直播视频，可以运行 startrecord " + s.itoa() + " 或 startrecdan " + s.itoa())
						// 不下载直播视频时下载弹幕
						if s.Danmu {
							startDanmu(s.UID)
						}
					}
				}
			} else {
				if isLive {
					// 主播没下播但是liveRoom里没有该主播导致重新显示开播（可能是A站的bug）
					if _, flvURL, _ := s.getStreamURL(); flvURL == "" {
						isLive = false
						lPrintln(s.longID() + "已经下播")
						if s.Notify {
							desktopNotify(s.Name + "已经下播")
							s.sendCoolq(s.Name + "已经下播")
						}
						if s.Record {
							msgMap.Lock()
							if m := msgMap.msg[s.UID]; m.recording {
								m.rec.ch <- liveOff
							}
							msgMap.Unlock()
						}
					}
				}
			}

			msgMap.Lock()
			if m := msgMap.msg[s.UID]; m.modify {
				m.modify = false
			}
			msgMap.Unlock()
		}

		time.Sleep(time.Second)
	}
}
