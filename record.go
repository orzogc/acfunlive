// 下载直播相关
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var udpPort = 50158

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
				fmt.Println("已经设置过自动下载" + s.ID + "的直播")
			} else {
				streamers[i].Record = true
				fmt.Println("成功设置自动下载" + s.ID + "的直播")
			}
		}
	}
	sMutex.Unlock()

	if !isExist {
		id := getID(uid)
		if id == "" {
			fmt.Println("不存在这个用户")
			return
		}

		newStreamer := streamer{UID: uid, ID: id, Notify: false, Record: true, Restream: false}
		sMutex.Lock()
		streamers = append(streamers, newStreamer)
		sMutex.Unlock()
		fmt.Println("成功设置自动下载" + id + "的直播")
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
				streamers[i].Restream = false
			} else {
				deleteStreamer(uid)
			}
			fmt.Println("成功取消自动下载" + s.ID + "的直播")
		}
	}
	sMutex.Unlock()

	saveConfig()
}

// 临时下载指定主播的直播
func startRec(uid uint, restream bool) {
	s := streamer{UID: uid, ID: getID(uid), Restream: restream}

	recMutex.Lock()
	_, ok := recordMap[s.UID]
	recMutex.Unlock()
	if ok {
		fmt.Println("已经在下载" + s.ID + "（" + s.uidStr() + "）" + "的直播，如要重启下载，请先运行stoprecord " + s.uidStr())
		return
	}

	if !s.isLiveOn() {
		fmt.Println(s.ID + "不在直播，取消下载")
		return
	}

	recCh := make(chan control, 20)
	go s.recordLive(recCh)
}

// 停止下载指定主播的直播
func stopRec(uid uint) {
	recMutex.Lock()
	rec, ok := recordMap[uid]
	recMutex.Unlock()
	if ok {
		fmt.Println("开始结束该主播的下载")
		rec.ch <- stopRecord
		io.WriteString(rec.stdin, "q")
		time.Sleep(20 * time.Second)
		rec.cancel()
	} else {
		fmt.Println("没有在下载该主播的直播")
	}
}

// 下载主播的直播
func (s streamer) recordLive(ch chan control) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in recordLive(), the error is:", err)
			log.Println("下载" + s.ID + "（" + s.uidStr() + "）" + "的直播发生错误，如要重启下载，请运行startrecord " + s.uidStr())
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
		log.Println("无法获取" + s.ID + "（" + s.uidStr() + "）" + "的直播源，退出下载，如要重启下载，请运行startrecord " + s.uidStr())
		desktopNotify("无法获取" + s.ID + "的直播源，退出下载")
		return
	}

	ffmpegFile := "ffmpeg"
	// windows下ffmpeg.exe需要和本程序exe放在同一文件夹下
	if runtime.GOOS == "windows" {
		ffmpegFile = filepath.Join(exeDir, "ffmpeg.exe")
	}

	title := s.getTitle()
	logPrintln("开始下载" + s.ID + "（" + s.uidStr() + "）" + "的直播")
	recordTime := getTime()
	outFile := filepath.Join(exeDir, recordTime+" "+s.ID+" "+title+".mp4")
	fmt.Println("本次下载的文件保存在" + outFile + "\n" + "如果想提前结束下载，运行stoprecord " + s.uidStr())
	desktopNotify("开始下载" + s.ID + "的直播")
	// 运行ffmpeg下载直播
	var cmd *exec.Cmd
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if s.Restream {
		if udpPort > 65535 {
			log.Println("UDP端口不能超过65535，请重新运行本程序")
			desktopNotify("UDP端口不能超过65535，请重新运行本程序")
			return
		}
		udpURL := "udp://@127.0.0.1:" + fmt.Sprint(udpPort)
		udpPort++
		cmd = exec.CommandContext(ctx, ffmpegFile,
			"-timeout", "10000000",
			"-i", liveURL,
			"-c", "copy", outFile,
			"-c", "copy", "-f", "mpegts", udpURL)
		fmt.Println("现在可以利用本地UDP端口观看" + s.ID + "的直播" + "\n" + "播放器的观看地址是：\n" + udpURL)
	} else {
		cmd = exec.CommandContext(ctx, ffmpegFile,
			"-timeout", "10000000",
			"-i", liveURL,
			"-c", "copy", outFile)
	}

	stdin, err := cmd.StdinPipe()
	checkErr(err)
	defer stdin.Close()
	rec := record{stdin: stdin, cancel: cancel, ch: ch}
	recMutex.Lock()
	recordMap[s.UID] = rec
	recMutex.Unlock()

	err = cmd.Run()
	if err != nil {
		log.Println("下载"+s.ID+"（"+s.uidStr()+"）"+"的直播出现错误，尝试重启下载：", err)
	}

	recMutex.Lock()
	if s.isLiveOn() {
		select {
		case msg := <-ch:
			switch msg {
			// 收到退出信号
			case stopRecord:
			}
		default:
			// 由于某种原因导致下载意外结束
			logPrintln("因意外结束下载" + s.ID + "（" + s.uidStr() + "）" + "的直播，尝试重启下载")
			go s.recordLive(ch)
		}
	}
	delete(recordMap, s.UID)
	recMutex.Unlock()

	logPrintln(s.ID + "（" + s.uidStr() + "）" + "的直播下载已经结束")
	desktopNotify(s.ID + "的直播下载已经结束")
}
