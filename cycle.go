// 循环相关
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// 处理管道信号
func (s *streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		lPrintln("重启监听" + s.longID() + "的直播状态")
		mainCh <- msg
	case stopCycle:
		lPrintln("退出" + s.longID() + "的循环")
		sInfoMap.Lock()
		if _, ok := sInfoMap.info[s.UID]; ok {
			delete(sInfoMap.info, s.UID)
		} else {
			lPrintErr("sInfoMap没有%s的key", s.longID())
		}
		sInfoMap.Unlock()
	case quit:
	default:
		lPrintErrf("未知的 controlMsg：%+v", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle(liveID string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in cycle(), the error is:", err)
			lPrintErr(s.longID() + "的循环处理发生错误，尝试重启循环")

			restart := controlMsg{s: s, c: startCycle}
			mainCh <- restart
		}
	}()

	ch := make(chan controlMsg, 20)
	var modify bool
	sInfoMap.Lock()
	if m, ok := sInfoMap.info[s.UID]; ok {
		m.ch = ch
		modify = m.modify
		m.modify = false
	} else {
		sInfoMap.info[s.UID] = &streamerInfo{ch: ch}
	}
	sInfoMap.Unlock()

	// 设置文件里有该主播，但是不通知不下载
	if !(s.Notify.NotifyOn || s.Notify.NotifyOff || s.Notify.NotifyRecord || s.Notify.NotifyDanmu || s.Record || s.Danmu || s.KeepOnline) {
		for {
			msg := <-ch
			s.handleMsg(msg)
			return
		}
	}

	lPrintln("开始监听" + s.longID() + "的直播状态")

	var isLive bool
	for {
		select {
		case msg := <-ch:
			msg.liveID = liveID
			s.handleMsg(msg)
			return
		default:
			if s.isLiveOn() {
				isLive = true
				newLiveID := s.getLiveID()
				if newLiveID == "" {
					lPrintErrf("无法获取%s的liveID", s.longID())
					time.Sleep(time.Second)
					continue
				}

				if liveID != newLiveID || modify {
					liveID = newLiveID
					modify = false
					title := s.getTitle()
					lPrintln(s.longID() + "正在直播：" + title)
					lPrintln(s.Name + "的直播观看地址：" + s.getURL())

					if s.Notify.NotifyOn {
						desktopNotify(s.Name + "正在直播：" + title)
						s.sendMirai(fmt.Sprintf("%s正在直播：%s，观看地址：%s", s.Name, title, s.getURL()), true)
					}

					info, _ := getLiveInfo(liveID)

					// 优先级：录播 > 弹幕/挂机
					if s.Record && !info.isRecording {
						go s.recordLive(s.Danmu || s.KeepOnline)
					} else {
						lPrintf("如果要临时下载%s的直播视频，可以运行 startrecord %d 或 startrecdan %d", s.Name, s.UID, s.UID)
						// 不下载直播视频时下载弹幕
						if (s.Danmu && !info.isDanmu) || (s.KeepOnline && !info.isKeepOnline) {
							filename := getTime() + " " + s.Name + " " + title
							go s.initDanmu(mainCtx, liveID, filename)
						}
					}
				}
			} else {
				// 应付 AcFun API 可能出现的 bug：主播没下播但 API 显示下播
				if isLive && !s.isLiveOnByPage() {
					isLive = false
					lPrintln(s.longID() + "已经下播")
					if s.Notify.NotifyOff {
						msg := s.Name + "已经下播"
						desktopNotify(msg)
						s.sendMirai(msg, true)
					}
				}
			}
		}

		time.Sleep(time.Second)
	}
}

// 循环检测删除 lInfoMap.info 里没有下载视频和弹幕以及不在挂机的 key
func cycleDelKey(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			lInfoMap.Lock()
			for liveID, info := range lInfoMap.info {
				if !(info.isRecording || info.isDanmu || info.isKeepOnline) {
					delete(lInfoMap.info, liveID)
				}
			}
			lInfoMap.Unlock()

			// 每分钟循环一次
			time.Sleep(time.Minute)
		}
	}
}

// 循环获取 AcFun 直播间数据
func cycleFetch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ok := fetchAllRooms(); ok {
				if len(liveRooms.newRooms) == 0 {
					lPrintWarn("没有人在直播")
				}

				liveRooms.Lock()
				if config.AutoKeepOnline && is_login_acfun() && needMdealInfo.Load() {
					for uid, room := range liveRooms.newRooms {
						// 这样可以防止请求过多，但是要下一场直播才会自动挂牌子
						if _, ok := liveRooms.rooms[uid]; !ok {
							go func(uid int, name string) {
								r := rand.New(rand.NewSource(time.Now().UnixNano()))
								n := r.Intn(10000)
								time.Sleep(time.Duration(n) * time.Millisecond)

								var isChanged bool
								streamers.Lock()
								if s, ok := streamers.crt[uid]; ok {
									if !s.KeepOnline {
										hasMedal, err := fetchMedalInfo(uid)
										if err != nil {
											lPrintErr("%+v", err)
										} else if hasMedal {
											s.KeepOnline = true
											streamers.crt[s.UID] = s
											isChanged = true
										}
									}
								} else {
									hasMedal, err := fetchMedalInfo(uid)
									if err != nil {
										lPrintErr("%+v", err)
									} else if hasMedal {
										s := streamer{
											UID:        uid,
											Name:       name,
											KeepOnline: true,
										}
										streamers.crt[s.UID] = s
										isChanged = true
									}
								}
								streamers.Unlock()

								if isChanged {
									saveLiveConfig()
								}
							}(uid, room.name)
						}
					}
				}

				for uid, room := range liveRooms.rooms {
					delete(liveRooms.rooms, uid)
					liveRoomPool.Put(room)
				}
				liveRooms.rooms = liveRooms.newRooms
				liveRooms.Unlock()
			}

			// 每 60 秒循环一次
			time.Sleep(60 * time.Second)
		}
	}
}

// 循环获取登陆帐号拥有的徽章列表
func cycleGetMedals(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErrf("Recovering from panic in cycleGetMedals(), the error is: %v", err)
			lPrintErr("自动挂机出现错误，取消自动挂机")
		}
	}()

	if !is_login_acfun() {
		lPrintErr("没有登陆 AcFun 帐号，取消自动挂机")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			list, err := fetchMedalList()
			if err == nil {
				length := len(list)

				var isChanged bool
				streamers.Lock()
				for _, m := range list {
					if s, ok := streamers.crt[int(m.uid)]; ok {
						if !s.KeepOnline {
							s.KeepOnline = true
							streamers.crt[s.UID] = s
							isChanged = true
						}
					} else {
						s := streamer{
							UID:        int(m.uid),
							Name:       m.name,
							KeepOnline: true,
						}
						streamers.crt[s.UID] = s
						isChanged = true
					}
					medalInfoPool.Put(m)
				}
				streamers.Unlock()

				if isChanged {
					saveLiveConfig()
				}

				// 守护徽章列表最多只有 500 个
				if length >= 500 {
					_ = needMdealInfo.CompareAndSwap(false, true)
					return
				}
			} else {
				lPrintErrf("%+v", err)
			}

			// 每分钟循环一次
			time.Sleep(time.Minute)
		}
	}
}
