// 弹幕下载相关
package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/orzogc/acfundanmu"
)

// AcFun帐号的cookies
var acfunCookies acfundanmu.Cookies

// 不同的视频分辨率对应的弹幕字幕设置
var subConfigs = map[int]acfundanmu.SubConfig{
	0:    {PlayResX: 720, PlayResY: 1280, FontSize: 60}, // 这是手机直播，有一些是540X960
	540:  {PlayResX: 960, PlayResY: 540, FontSize: 30},
	720:  {PlayResX: 1280, PlayResY: 720, FontSize: 40},
	1080: {PlayResX: 1920, PlayResY: 1080, FontSize: 60},
}

// 下载直播弹幕
func (s streamer) getDanmu(ctx context.Context, info liveInfo) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getDanmu(), the error is:", err)
			lPrintErr("下载" + s.longID() + "的直播弹幕发生错误，如要重启下载，请运行 startdanmu " + s.itoa())
			desktopNotify("下载" + s.Name + "的直播弹幕发生错误")
			s.sendMirai("下载" + s.longID() + "的直播弹幕发生错误，如要重启下载，请运行 startdanmu " + s.itoa())
		}
	}()

	if s.KeepOnline {
		if len(acfunCookies) != 0 {
			lPrintf("开始在%s的直播间挂机", s.longID())
		} else {
			lPrintErrf("没有登陆AcFun帐号，取消在%s的直播间挂机", s.longID())
			if !s.Danmu {
				return
			}
		}
	}

	if s.Danmu {
		lPrintln("开始下载" + s.longID() + "的直播弹幕")
		lPrintln("本次下载的ass文件保存在" + info.assFile)
		if *isListen {
			lPrintln("如果想提前结束下载" + s.longID() + "的直播弹幕，运行 stopdanmu " + s.itoa())
		}
		if s.Notify.NotifyDanmu {
			if !s.Record {
				desktopNotify("开始下载" + s.Name + "的直播弹幕")
				s.sendMirai("开始下载" + s.longID() + "的直播弹幕：" + s.getTitle())
			}
		}
	}

	var cookies acfundanmu.Cookies
	if s.KeepOnline {
		cookies = acfunCookies
	}
	ac, err := acfundanmu.NewAcFunLive(acfundanmu.SetLiverUID(int64(s.UID)), acfundanmu.SetCookies(cookies))
	checkErr(err)
	_ = ac.StartDanmu(ctx, false)
	if s.Danmu {
		ac.WriteASS(ctx, info.cfg, info.assFile, true)
		defer s.moveFile(info.assFile)
	} else if s.KeepOnline {
		for {
			if danmu := ac.GetDanmu(); danmu == nil {
				break
			}
		}
	} else {
		lPrintErr("s.Danmu或s.KeepOnline必须为true")
		return
	}

	time.Sleep(5 * time.Second)

Outer:
	for {
		select {
		case <-ctx.Done():
			break Outer
		default:
			// 因意外结束弹幕下载时重启下载
			// 应付AcFun API可能出现的bug
			if s.isLiveOnByPage() {
				if newLiveID := getLiveID(s.UID); newLiveID == info.LiveID {
					lPrintWarn("因意外结束下载" + s.longID() + "的直播弹幕，尝试重启下载")
					ac, err := acfundanmu.NewAcFunLive(acfundanmu.SetLiverUID(int64(s.UID)), acfundanmu.SetCookies(cookies))
					checkErr(err)
					_ = ac.StartDanmu(ctx, false)
					if s.Danmu {
						ac.WriteASS(ctx, info.cfg, info.assFile, false)
					} else if s.KeepOnline {
						for {
							if danmu := ac.GetDanmu(); danmu == nil {
								break
							}
						}
					}
					time.Sleep(10 * time.Second)
				} else {
					break Outer
				}
			} else {
				break Outer
			}
		}
	}

	if s.KeepOnline {
		lPrintf("停止在%s的直播间挂机", s.longID())
	}
	if s.Danmu {
		lPrintln(s.longID() + "的直播弹幕下载已经结束")
		if s.Notify.NotifyDanmu {
			if !s.Record {
				desktopNotify(s.Name + "的直播弹幕下载已经结束")
				s.sendMirai(s.longID() + "的直播弹幕下载已经结束")
			}
		}
	}
}

// 退出直播弹幕下载相关操作
func (s streamer) quitDanmu(liveID string) {
	lInfoMap.Lock()
	defer lInfoMap.Unlock()
	if info, ok := lInfoMap.info[liveID]; ok {
		if s.Danmu {
			info.isDanmu = false
		}
		if s.KeepOnline {
			info.isKeepOnline = false
		}
		lInfoMap.info[info.LiveID] = info
	}
}

// 初始化弹幕下载
func (s streamer) initDanmu(ctx context.Context, liveID, filename string) {
	dctx, dcancel := context.WithCancel(ctx)
	defer dcancel()
	info, ok := getLiveInfo(liveID)
	if ok {
		if info.isDanmu && !info.isKeepOnline {
			if s.Danmu && !s.KeepOnline {
				lPrintWarn("已经在下载" + s.longID() + "的直播弹幕，如要重启下载，请先运行 stopdanmu " + s.itoa())
				return
			}
			s.Danmu = false
		} else if !info.isDanmu && info.isKeepOnline {
			if !s.Danmu && s.KeepOnline {
				lPrintWarn("已经在" + s.longID() + "的直播间挂机")
				return
			}
			s.KeepOnline = false
		} else if info.isDanmu && info.isKeepOnline {
			lPrintWarn("已经在下载" + s.longID() + "的直播弹幕和在其直播间挂机，如要重启下载，请先运行 stopdanmu " + s.itoa())
			return
		}
	} else {
		// 获取直播源和对应的弹幕设置
		var err error
		info, err = s.getLiveInfo()
		if err != nil {
			lPrintErr(err)
			msg := "无法获取" + s.longID() + "的直播源，退出下载直播弹幕，请确定主播正在直播，如要重启下载直播弹幕，请运行 startdanmu " + s.itoa()
			lPrintErr(msg)
			if s.Notify.NotifyDanmu {
				desktopNotify("无法获取" + s.Name + "的直播源，退出下载直播弹幕")
				s.sendMirai(msg)
			}
			return
		}
	}

	if s.Danmu {
		info.isDanmu = true
		info.danmuCancel = dcancel
	}
	if s.KeepOnline {
		info.isKeepOnline = true
		info.onlineCancel = dcancel
	}

	assFile := transFilename(filename)
	if assFile == "" {
		return
	}
	info.assFile = assFile + ".ass"
	info.cfg.Title = filepath.Base(assFile)
	info.cfg.StartTime = time.Now().UnixNano()
	setLiveInfo(info)
	defer s.quitDanmu(info.LiveID)

	s.getDanmu(dctx, info)
}

// 临时下载指定主播的直播弹幕
func startDanmu(uid int) bool {
	s, ok := getStreamer(uid)
	if !ok {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}
		s = streamer{UID: uid, Name: name}
	}
	s.Notify.NotifyDanmu = true
	s.Danmu = true

	liveID := getLiveID(uid)
	if liveID == "" {
		lPrintErr(s.longID() + "不在直播，取消下载直播弹幕")
		return false
	}
	if isDanmu(liveID) {
		lPrintWarn("已经在下载" + s.longID() + "的直播弹幕，如要重启下载，请先运行 stopdanmu " + s.itoa())
		return false
	}

	filename := getTime() + " " + s.Name + " " + s.getTitle()

	// 查看程序是否处于监听状态
	if *isListen {
		// goroutine是为了快速返回
		go s.initDanmu(mainCtx, liveID, filename)
	} else {
		// 程序只在单独下载一个直播弹幕，不用goroutine，防止程序提前结束运行
		s.initDanmu(mainCtx, liveID, filename)
	}
	return true
}

// 停止下载指定主播的直播弹幕
func stopDanmu(uid int) bool {
	infoList, ok := getLiveInfoByUID(uid)
	if !ok {
		lPrintWarn("没有在下载uid为" + itoa(uid) + "的主播的直播弹幕")
		return true
	}

	for _, info := range infoList {
		if info.isDanmu {
			lPrintf("开始停止下载%s的liveID为%s直播弹幕", longID(uid), info.LiveID)
			info.danmuCancel()
		}
	}

	return true
}

// 临时下载指定主播的直播视频和弹幕
func startRecDan(uid int) bool {
	return startRec(uid, true)
}

// 取消下载指定主播的直播视频和弹幕
func stopRecDan(uid int) bool {
	return stopRec(uid) && stopDanmu(uid)
}
