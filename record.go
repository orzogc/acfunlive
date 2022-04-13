// 直播下载相关
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const ffmpegNotExist = "没有找到FFmpeg，停止下载直播视频"

// 临时下载指定主播的直播视频
func startRec(uid int, danmu bool) bool {
	var name string
	s, ok := getStreamer(uid)
	if !ok {
		name = getName(uid)
		if name == "" {
			lPrintWarnf("不存在uid为%d的用户", uid)
			return false
		}
		s = streamer{UID: uid, Name: name}
	}
	s.Notify.NotifyRecord = true
	s.Record = true
	s.Danmu = danmu

	liveID := getLiveID(uid)
	if liveID == "" {
		lPrintErr(s.longID() + "不在直播，取消下载直播视频")
		return false
	}
	if isRecording(liveID) {
		lPrintWarnf("已经在下载%s的直播视频，如要重启下载，请先运行 stoprecord %d", s.longID(), s.UID)
		return false
	}

	if ffmpegFile := getFFmpeg(); ffmpegFile == "" {
		desktopNotify(ffmpegNotExist)
		s.sendMirai(ffmpegNotExist, false)
		return false
	}

	// 查看程序是否处于监听状态
	if *isListen {
		// goroutine是为了快速返回
		go s.recordLive(danmu)
	} else {
		// 程序只在单独下载一个直播视频，不用goroutine，防止程序提前结束运行
		s.recordLive(danmu)
	}
	return true
}

// 停止下载指定主播的直播视频
func stopRec(uid int) bool {
	infoList, ok := getLiveInfoByUID(uid)
	if !ok {
		lPrintWarnf("没有在下载uid为%d的主播的直播视频", uid)
		return true
	}

	for _, info := range infoList {
		if info.isRecording {
			lPrintf("开始停止下载%s的liveID为%s直播视频", longID(uid), info.LiveID)
			info.recordCh <- stopRecord
			_, _ = io.WriteString(info.ffmpegStdin, "q")
			// 等待20秒强制停止下载，goroutine是为了防止锁住时间过长
			go func(cancel context.CancelFunc) {
				time.Sleep(20 * time.Second)
				cancel()
			}(info.recordCancel)
		}
	}

	return true
}

// 退出直播视频下载相关操作
func quitRec(liveID string) {
	lInfoMap.Lock()
	defer lInfoMap.Unlock()
	if info, ok := lInfoMap.info[liveID]; ok {
		if info.isRecording {
			info.isRecording = false
			lInfoMap.info[info.LiveID] = info
		}
	}
}

// 下载主播的直播视频
func (s streamer) recordLive(danmu bool) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in recordLive(), the error is:", err)
			msg := "下载%s的直播视频发生错误，如要重启下载，请运行 startrecord %d 或 startrecdan %d"
			lPrintErrf(msg, s.longID(), s.UID, s.UID)
			desktopNotify("下载" + s.Name + "的直播视频发生错误")
			s.sendMirai(fmt.Sprintf(msg, s.Name, s.UID, s.UID), false)
		}
	}()

	ffmpegFile := getFFmpeg()
	if ffmpegFile == "" {
		desktopNotify(ffmpegNotExist)
		s.sendMirai(ffmpegNotExist, false)
		return
	}

	// 获取直播源
	info, err := s.getLiveInfo()
	if err != nil {
		lPrintErr(err)
		msg := "无法获取%s的直播源，退出下载直播视频，请确定主播正在直播，如要重启下载，请运行 startrecord %d 或 startrecdan %d"
		lPrintErrf(msg, s.longID(), s.UID, s.UID)
		if s.Notify.NotifyRecord {
			desktopNotify("无法获取" + s.Name + "的直播源，退出下载直播视频")
			s.sendMirai(fmt.Sprintf(msg, s.Name, s.UID, s.UID), false)
		}
		return
	}

	if existInfo, ok := getLiveInfo(info.LiveID); ok {
		if existInfo.isRecording {
			lPrintWarnf("已经在下载%s的直播视频，如要重启下载，请先运行 stoprecord %d", s.longID(), s.UID)
			return
		}
		url := info.streamURL
		info = existInfo
		info.streamURL = url
	}

	title := s.getTitle()
	filename := getTime() + " " + s.Name + " " + title
	recordFile := transFilename(filename)
	if recordFile == "" {
		return
	}
	// 想要输出其他视频格式可以修改config.json里的Output
	recordFile = recordFile + "." + config.Output
	info.recordFile = recordFile

	lPrintln("开始下载" + s.longID() + "的直播视频")
	lPrintln("本次下载的视频文件保存在" + recordFile)
	if *isListen {
		lPrintf("如果想提前结束下载%s的直播视频，运行 stoprecord %d", s.longID(), s.UID)
	}
	if s.Notify.NotifyRecord {
		if danmu {
			desktopNotify("开始下载" + s.Name + "的直播视频和弹幕")
			s.sendMirai(fmt.Sprintf("开始下载%s的直播视频和弹幕：%s，观看地址：%s", s.Name, title, s.getURL()), false)
		} else {
			desktopNotify("开始下载" + s.Name + "的直播视频")
			s.sendMirai(fmt.Sprintf("开始下载%s的直播视频：%s，观看地址：%s", s.Name, title, s.getURL()), false)
		}
	}

	// 运行ffmpeg下载直播视频，不用mainCtx是为了能正常退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegFile,
		"-rw_timeout", "20000000",
		"-timeout", "20000000",
		"-i", info.streamURL,
		"-c", "copy", recordFile)
	hideCmdWindow(cmd)

	stdin, err := cmd.StdinPipe()
	checkErr(err)
	defer stdin.Close()
	info.recordCh = make(chan control, 20)
	info.ffmpegStdin = stdin
	info.recordCancel = cancel
	info.isRecording = true
	setLiveInfo(info)
	// 只运行一次
	var once sync.Once
	q := func() {
		quitRec(info.LiveID)
	}
	defer once.Do(q)

	if !*isListen {
		// 程序单独下载一个直播视频时可以按q键退出（ffmpeg的特性）
		cmd.Stdin = os.Stdin
		lPrintln("按q键退出下载直播视频")
	}

	// 下载弹幕
	if danmu {
		go s.initDanmu(ctx, info.LiveID, filename)
	}

	err = cmd.Run()
	if err != nil {
		lPrintErrf("下载%s的直播视频出现错误，尝试重启下载：%v", s.longID(), err)
	}
	defer s.moveFile(recordFile)

	// 取消弹幕下载
	cancel()
	time.Sleep(10 * time.Second)

	if s.isLiveOnByPage() {
		select {
		case <-info.recordCh:
		default:
			if newLiveID := getLiveID(s.UID); newLiveID == info.LiveID && *isListen {
				// 程序处于监听状态时重启下载，否则不重启
				lPrintWarn("因意外结束下载" + s.longID() + "的直播视频，尝试重启下载")
				once.Do(q)
				go s.recordLive(danmu)
			}
		}
	}

	lPrintln(s.longID() + "的直播视频下载已经结束")
	if s.Notify.NotifyRecord {
		if danmu {
			desktopNotify(s.Name + "的直播视频和弹幕下载已经结束")
			s.sendMirai(s.Name+"的直播视频和弹幕下载已经结束", false)
		} else {
			desktopNotify(s.Name + "的直播视频下载已经结束")
			s.sendMirai(s.Name+"的直播视频下载已经结束", false)
		}
	}
}
