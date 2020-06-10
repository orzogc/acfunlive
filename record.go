// 下载直播相关
package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// record用来传递下载信息
type record struct {
	stdin  io.WriteCloser
	cancel context.CancelFunc
	ch     chan control
	//isLiveOff bool
}

// recordMap的锁
var recMutex sync.Mutex
var recordMap = make(map[uint]record)

// 设置自动下载指定主播的直播
func addRecord(uid uint) {
	isExist := false
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			isExist = true
			if s.Record {
				logger.Println("已经设置过自动下载" + s.ID + "的直播")
			} else {
				streamers[i].Record = true
				logger.Println("成功设置自动下载" + s.ID + "的直播")
			}
		}
	}
	sMutex.Unlock()

	if !isExist {
		id := getID(uid)
		if id == "" {
			logger.Println("不存在这个用户")
			return
		}

		newStreamer := streamer{UID: uid, ID: id, Notify: false, Record: true}
		sMutex.Lock()
		streamers = append(streamers, newStreamer)
		sMutex.Unlock()
		logger.Println("成功设置自动下载" + id + "的直播")
	}

	saveConfig()
}

// 取消自动下载指定主播的直播
func delRecord(uid uint) {
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			if s.Notify {
				streamers[i].Record = false
			} else {
				deleteStreamer(uid)
			}
			logger.Println("成功取消自动下载" + s.ID + "的直播")
		}
	}
	sMutex.Unlock()

	saveConfig()
}

// 临时下载指定主播的直播
func startRec(uid uint) {
	s := streamer{UID: uid, ID: getID(uid)}

	recMutex.Lock()
	_, ok := recordMap[s.UID]
	recMutex.Unlock()
	if ok {
		logger.Println("已经在下载" + s.longID() + "的直播，如要重启下载，请先运行stoprecord " + s.uidStr())
		return
	}

	if !s.isLiveOn() {
		logger.Println(s.longID() + "不在直播，取消下载")
		return
	}

	go s.recordLive()
}

// 停止下载指定主播的直播
func stopRec(uid uint) {
	recMutex.Lock()
	rec, ok := recordMap[uid]
	recMutex.Unlock()
	if ok {
		logger.Println("开始结束该主播的下载")
		rec.ch <- stopRecord
		io.WriteString(rec.stdin, "q")
		time.Sleep(20 * time.Second)
		rec.cancel()
	} else {
		logger.Println("没有在下载该主播的直播")
	}
}

// 下载主播的直播
func (s streamer) recordLive() {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in recordLive(), the error is:", err)
			timePrintln("下载" + s.longID() + "的直播发生错误，如要重启下载，请运行startrecord " + s.uidStr())
			desktopNotify("下载" + s.ID + "的直播发生错误")
			recMutex.Lock()
			delete(recordMap, s.UID)
			recMutex.Unlock()
		}
	}()

	// 只能有一个在下载
	for {
		recMutex.Lock()
		_, ok := recordMap[s.UID]
		recMutex.Unlock()
		if !ok {
			break
		}
		time.Sleep(time.Second)
	}

	// 下载hls直播源，想下载flv直播源的话可手动更改此处
	liveURL, _ := s.getStreamURL()
	if liveURL == "" {
		timePrintln("无法获取" + s.longID() + "的直播源，退出下载，如要重启下载，请运行startrecord " + s.uidStr())
		desktopNotify("无法获取" + s.ID + "的直播源，退出下载")
		return
	}

	ffmpegFile := "ffmpeg"
	// windows下ffmpeg.exe需要和本程序exe放在同一文件夹下
	if runtime.GOOS == "windows" {
		ffmpegFile = filepath.Join(exeDir, "ffmpeg.exe")
	}

	title := s.getTitle()
	timePrintln("开始下载" + s.longID() + "的直播")
	recordTime := getTime()
	outFile := filepath.Join(exeDir, recordTime+" "+s.ID+" "+title+".mp4")
	logger.Println("本次下载的文件保存在" + outFile)
	if *isListen {
		logger.Println("如果想提前结束下载，运行stoprecord " + s.uidStr())
	}
	desktopNotify("开始下载" + s.ID + "的直播")

	// 运行ffmpeg下载直播
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegFile,
		"-timeout", "10000000",
		"-i", liveURL,
		"-c", "copy", outFile)

	stdin, err := cmd.StdinPipe()
	checkErr(err)
	defer stdin.Close()
	ch := make(chan control, 20)
	rec := record{stdin: stdin, cancel: cancel, ch: ch}
	recMutex.Lock()
	recordMap[s.UID] = rec
	recMutex.Unlock()

	if !*isListen {
		cmd.Stdin = os.Stdin
		logger.Println("按q退出下载")
	}

	err = cmd.Run()
	if err != nil {
		timePrintln("下载"+s.longID()+"的直播出现错误，尝试重启下载：", err)
	}

	recMutex.Lock()
	if s.isLiveOn() {
		select {
		case msg := <-ch:
			switch msg {
			// 收到退出信号
			case stopRecord:
			default:
				timePrintln("未知的controlMsg：", msg)
			}
		default:
			// 由于某种原因导致下载意外结束
			timePrintln("因意外结束下载" + s.longID() + "的直播，尝试重启下载")
			go s.recordLive()
		}
	}
	delete(recordMap, s.UID)
	recMutex.Unlock()

	timePrintln(s.longID() + "的直播下载已经结束")
	desktopNotify(s.ID + "的直播下载已经结束")
}
