// 下载直播相关
package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"
)

// record用来传递下载信息
type record struct {
	stdin  io.WriteCloser
	cancel context.CancelFunc
	ch     chan control
}

// 下载信息的map，map[uint]record
var recordMap = sync.Map{}

// 存放某些没在recordMap的下载
var danglingRec struct {
	mu      sync.Mutex
	records []record
}

// 设置自动下载指定主播的直播
func addRecord(uid uint) bool {
	isExist := false
	streamers.mu.Lock()
	for i, s := range streamers.current {
		if s.UID == uid {
			isExist = true
			if s.Record {
				lPrintln("已经设置过自动下载" + s.ID + "的直播")
			} else {
				streamers.current[i].Record = true
				lPrintln("成功设置自动下载" + s.ID + "的直播")
			}
		}
	}
	streamers.mu.Unlock()

	if !isExist {
		id := getID(uid)
		if id == "" {
			lPrintln("不存在uid为" + uidStr(uid) + "的用户")
			return false
		}

		newStreamer := streamer{UID: uid, ID: id, Notify: false, Record: true}
		streamers.mu.Lock()
		streamers.current = append(streamers.current, newStreamer)
		streamers.mu.Unlock()
		lPrintln("成功设置自动下载" + id + "的直播")
	}

	saveConfig()
	return true
}

// 取消自动下载指定主播的直播
func delRecord(uid uint) bool {
	streamers.mu.Lock()
	for i, s := range streamers.current {
		if s.UID == uid {
			if s.Notify {
				streamers.current[i].Record = false
			} else {
				deleteStreamer(uid)
			}
			lPrintln("成功取消自动下载" + s.ID + "的直播")
		}
	}
	streamers.mu.Unlock()

	saveConfig()
	return true
}

// 临时下载指定主播的直播
func startRec(uid uint) bool {
	id := getID(uid)
	if id == "" {
		lPrintln("不存在uid为" + uidStr(uid) + "的用户")
		return false
	}
	s := streamer{UID: uid, ID: id}

	_, ok := recordMap.Load(s.UID)
	if ok {
		lPrintln("已经在下载" + s.longID() + "的直播，如要重启下载，请先运行stoprecord " + s.uidStr())
		return false
	}

	if !s.isLiveOn() {
		lPrintln(s.longID() + "不在直播，取消下载")
		return false
	}

	// 查看程序是否处于监听状态
	if *isListen {
		go s.recordLive()
	} else {
		// 程序只在单独下载一个直播，不用goroutine，防止程序提前结束运行
		s.recordLive()
	}
	return true
}

// 停止下载指定主播的直播
func stopRec(uid uint) bool {
	r, ok := recordMap.Load(uid)
	if ok {
		rec := r.(record)
		lPrintln("开始结束uid为" + uidStr(uid) + "的主播的下载")
		rec.ch <- stopRecord
		io.WriteString(rec.stdin, "q")
		// 等待20秒强关下载
		time.Sleep(20 * time.Second)
		rec.cancel()
		// 需要删除recordMap里相应的key
		recordMap.Delete(uid)
	} else {
		lPrintln("没有在下载uid为" + uidStr(uid) + "的主播的直播")
	}

	return true
}

// 下载主播的直播
func (s streamer) recordLive() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in recordLive(), the error is:", err)
			lPrintln("下载" + s.longID() + "的直播发生错误，如要重启下载，请运行startrecord " + s.uidStr())
			desktopNotify("下载" + s.ID + "的直播发生错误")
			time.Sleep(2 * time.Second)
			recordMap.Delete(s.UID)
		}
	}()

	// 下载hls直播源，想下载flv直播源的话可手动更改此处
	liveURL, _ := s.getStreamURL()
	if liveURL == "" {
		lPrintln("无法获取" + s.longID() + "的直播源，退出下载，如要重启下载，请运行startrecord " + s.uidStr())
		desktopNotify("无法获取" + s.ID + "的直播源，退出下载")
		return
	}

	ffmpegFile := "ffmpeg"
	// windows下ffmpeg.exe需要和本程序exe放在同一文件夹下
	if runtime.GOOS == "windows" {
		ffmpegFile = filepath.Join(exeDir, "ffmpeg.exe")
	}

	title := s.getTitle()
	recordTime := getTime()
	filename := recordTime + " " + s.ID + " " + title
	// 转换文件名不允许的特殊字符
	var re *regexp.Regexp
	if runtime.GOOS == "linux" {
		re = regexp.MustCompile(`[/]`)
	}
	if runtime.GOOS == "windows" {
		re = regexp.MustCompile(`[<>:"/\\|?*]`)
	}
	filename = re.ReplaceAllString(filename, "-")
	// linux和macOS下限制文件名长度
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if len(filename) > 250 {
			filename = filename[:250]
		}
	}
	// 想要输出其他视频格式可以修改这里的mp4
	outFile := filepath.Join(exeDir, filename+".mp4")
	// windows下全路径文件名不能过长
	if runtime.GOOS == "windows" {
		if utf8.RuneCountInString(outFile) > 259 {
			lPrintln("全路径文件名太长，取消下载")
			return
		}
	}

	lPrintln("开始下载" + s.longID() + "的直播")
	lPrintln("本次下载的文件保存在" + outFile)
	if *isListen {
		lPrintln("如果想提前结束下载，运行stoprecord " + s.uidStr())
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
	recordMap.Store(s.UID, rec)

	if !(*isListen) {
		// 程序单独下载一个直播时可以按q键退出（ffmpeg的特性）
		cmd.Stdin = os.Stdin
		lPrintln("按q键退出下载")
	}

	err = cmd.Run()
	if err != nil {
		lPrintln("下载"+s.longID()+"的直播出现错误，尝试重启下载：", err)
	}

	if s.isLiveOn() {
		select {
		case msg := <-ch:
			switch msg {
			// 收到下播的信号
			case liveOff:
			// 收到停止下载的信号
			case stopRecord:
			default:
				lPrintln("未知的controlMsg：", msg)
			}
		default:
			// 程序处于监听状态时重启下载，否则不重启
			if *isListen {
				// 由于某种原因导致下载意外结束
				lPrintln("因意外结束下载" + s.longID() + "的直播，尝试重启下载")
				// 延迟两秒，防止意外情况下刷屏
				time.Sleep(2 * time.Second)
				go s.recordLive()
			}
		}
	} else {
		recordMap.Delete(s.UID)
	}

	lPrintln(s.longID() + "的直播下载已经结束")
	desktopNotify(s.ID + "的直播下载已经结束")
}
