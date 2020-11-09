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

// record用来传递下载信息
/*
type record struct {
	stdin  io.WriteCloser     // ffmpeg的stdin
	cancel context.CancelFunc // 用来强行停止ffmpeg运行
	ch     chan control       // 下载goroutine的管道
}
*/

// 存放某些没在recordMap的下载
/*
var danglingRec struct {
	sync.Mutex // records的锁
	records    []record
}
*/

const ffmpegNotExist = "没有找到FFmpeg，停止下载直播视频"

// 设置自动下载指定主播的直播视频
func addRecord(uid int) bool {
	isExist := false
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		isExist = true
		if s.Record {
			lPrintWarn("已经设置过自动下载" + s.longID() + "的直播视频")
		} else {
			s.Record = true
			s.Notify.NotifyRecord = true
			sets(s)
			lPrintln("成功设置自动下载" + s.longID() + "的直播视频")
		}
	}
	streamers.Unlock()

	if !isExist {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, Name: name, Record: true, Notify: notify{NotifyRecord: true}}
		streamers.Lock()
		sets(newStreamer)
		streamers.Unlock()
		lPrintln("成功设置自动下载" + newStreamer.longID() + "的直播视频")
	}

	saveLiveConfig()
	return true
}

// 取消自动下载指定主播的直播视频
func delRecord(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		s.Record = false
		s.Notify.NotifyRecord = false
		sets(s)
		lPrintln("成功取消自动下载" + s.longID() + "的直播视频")
	} else {
		lPrintWarn("没有设置过自动下载uid为" + itoa(uid) + "的主播的直播视频")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 临时下载指定主播的直播视频
func startRec(uid int, danmu bool) bool {
	var name string
	streamers.Lock()
	s, ok := streamers.crt[uid]
	streamers.Unlock()
	if !ok {
		name = getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}
		s = streamer{UID: uid, Name: name}
	}
	s.Notify.NotifyRecord = true
	s.Record = true
	s.Danmu = danmu

	if isLive, room, err := tryFetchLiveInfo(s.UID); err != nil {
		lPrintErr(err)
		return false
	} else if !isLive {
		lPrintWarn(s.longID() + "不在直播，取消下载直播视频")
		return false
	} else if isRecording(room.liveID) {
		lPrintWarn("已经在下载" + s.longID() + "的直播视频，如要重启下载，请先运行 stoprecord " + s.itoa())
		return false
	}

	if ffmpegFile := getFFmpeg(); ffmpegFile == "" {
		desktopNotify(ffmpegNotExist)
		s.sendMirai(ffmpegNotExist)
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
		lPrintWarn("没有在下载uid为" + itoa(uid) + "的主播的直播视频")
		return true
	}

	for _, info := range infoList {
		if info.isRecording {
			lPrintf("开始停止下载%s的liveID为%s直播视频", longID(uid), info.LiveID)
			info.recordCh <- stopRecord
			io.WriteString(info.ffmpegStdin, "q")
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
			lPrintErr("下载" + s.longID() + "的直播视频发生错误，如要重启下载，请运行 startrecord " + s.itoa() + " 或 startrecdan " + s.itoa())
			desktopNotify("下载" + s.Name + "的直播视频发生错误")
			s.sendMirai("下载" + s.longID() + "的直播视频发生错误，如要重启下载，请运行 startrecord " + s.itoa() + " 或 startrecdan " + s.itoa())
		}
	}()

	ffmpegFile := getFFmpeg()
	if ffmpegFile == "" {
		desktopNotify(ffmpegNotExist)
		s.sendMirai(ffmpegNotExist)
		return
	}

	// 获取直播源
	info, err := s.getLiveInfo()
	if err != nil {
		lPrintErr(err)
		msg := fmt.Sprintf("无法获取%s的直播源，退出下载直播视频，请确定主播正在直播，如要重启下载，请运行 startrecord %s 或 startrecdan %s", s.longID(), s.itoa(), s.itoa())
		lPrintErr(msg)
		if s.Notify.NotifyRecord {
			desktopNotify("无法获取" + s.Name + "的直播源，退出下载直播视频")
			s.sendMirai(msg)
		}
		return
	}

	if existInfo, ok := getLiveInfo(info.LiveID); ok {
		if existInfo.isRecording {
			lPrintWarn("已经在下载" + s.longID() + "的直播视频，如要重启下载，请先运行 stoprecord " + s.itoa())
			return
		}
		url := info.streamURL
		info = existInfo
		info.streamURL = url
	}
	var once sync.Once
	q := func() {
		quitRec(info.LiveID)
	}
	defer once.Do(q)

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
		lPrintln("如果想提前结束下载" + s.longID() + "的直播视频，运行 stoprecord " + s.itoa())
	}
	if s.Notify.NotifyRecord {
		if danmu {
			desktopNotify("开始下载" + s.Name + "的直播视频和弹幕")
			s.sendMirai("开始下载" + s.longID() + "的直播视频和弹幕：" + title)
		} else {
			desktopNotify("开始下载" + s.Name + "的直播视频")
			s.sendMirai("开始下载" + s.longID() + "的直播视频：" + title)
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
		lPrintErr("下载"+s.longID()+"的直播视频出现错误，尝试重启下载：", err)
	}

	time.Sleep(10 * time.Second)

	if s.isLiveOnByPage() {
		select {
		case <-info.recordCh:
		default:
			if _, room, err := tryFetchLiveInfo(s.UID); err != nil {
				lPrintErr(err)
				return
			} else if room.liveID == info.LiveID && *isListen {
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
			s.sendMirai(s.longID() + "的直播视频和弹幕下载已经结束")
		} else {
			desktopNotify(s.Name + "的直播视频下载已经结束")
			s.sendMirai(s.longID() + "的直播视频下载已经结束")
		}
	}

	s.moveFile(recordFile)
}
