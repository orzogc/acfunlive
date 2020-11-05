package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/orzogc/acfundanmu"
)

type control int

// 控制信息
const (
	startCycle control = iota
	stopCycle
	liveOff
	stopRecord
	quit
)

// 主播的管道信息
type controlMsg struct {
	s streamer
	c control
}

// 主播的信息结构
type sMsg struct {
	ch          chan controlMsg    // 控制信息
	rec         record             // 下载信息
	isRecording bool               // 是否正在下载
	modify      bool               // 是否被修改设置
	danmuCancel context.CancelFunc // 用来停止下载弹幕
}

type streamInfo struct {
	acfundanmu.StreamInfo        // 直播源信息
	hlsURL                string // hls直播源
	flvURL                string // flv直播源
	cfg                   acfundanmu.SubConfig
}

// sMsg的map
var msgMap struct {
	sync.Mutex
	msg map[int]*sMsg
}

// 储存日志
var logString struct {
	sync.Mutex
	str strings.Builder
}

var (
	exeDir   string                                  // 运行程序所在文件夹
	mainCh   chan controlMsg                         // main()的管道
	mainCtx  context.Context                         // main()的ctx
	isListen *bool                                   // 程序是否处于监听状态
	isWebAPI *bool                                   // 程序是否启动web API服务器
	isWebUI  *bool                                   // 程序是否启动web UI服务器
	isNoGUI  = new(bool)                             // 程序是否启动GUI界面
	logger   = log.New(os.Stdout, "", log.LstdFlags) // 可以同步输出的logger
	itoa     = strconv.Itoa                          // 将int转换为字符串
	atoi     = strconv.Atoi                          // 将字符串转换为int
)

// 检查错误
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// 尝试运行，三次出错后结束运行
func run(f func() error) error {
	for retry := 0; retry < 3; retry++ {
		if err := f(); err != nil {
			log.Printf("%v", err)
		} else {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("运行三次都出现错误，停止运行")
}

// 获取时间
func getTime() string {
	t := time.Now()
	return fmt.Sprintf("%d-%02d-%02d %02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

// 获取时间，按照log的时间格式
func getLogTime() string {
	t := time.Now()
	return fmt.Sprintf("%d/%02d/%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

// 打印带时间戳的log信息
func lPrintln(msg ...interface{}) {
	if *isNoGUI {
		logger.Println(msg...)
	}
	// 同时输出日志到web服务
	logString.Lock()
	defer logString.Unlock()
	fmt.Fprint(&logString.str, getLogTime()+" ")
	fmt.Fprintln(&logString.str, msg...)
}

// 打印警告信息
func lPrintWarn(msg ...interface{}) {
	w := []interface{}{"[WARN]"}
	msg = append(w, msg...)
	lPrintln(msg...)
}

// 打印错误信息
func lPrintErr(msg ...interface{}) {
	e := []interface{}{"[ERROR]"}
	msg = append(e, msg...)
	lPrintln(msg...)
}

// 打印带时间戳的log信息（格式化字符串）
func lPrintf(format string, a ...interface{}) {
	lPrintln(fmt.Sprintf(format, a...))
}

// 打印警告信息（格式化字符串）
func lPrintWarnf(format string, a ...interface{}) {
	lPrintWarn(fmt.Sprintf(format, a...))
}

// 打印错误信息（格式化字符串）
func lPrintErrf(format string, a ...interface{}) {
	lPrintErr(fmt.Sprintf(format, a...))
}

// 将UID转换成字符串
func (s streamer) itoa() string {
	return itoa(s.UID)
}

// 返回ID（UID）形式的字符串
func (s streamer) longID() string {
	return s.Name + "（" + s.itoa() + "）"
}

// 尝试删除msgMap.msg里的键
func deleteMsg(uid int) {
	streamers.Lock()
	defer streamers.Unlock()
	msgMap.Lock()
	defer msgMap.Unlock()
	_, oks := streamers.crt[uid]
	m, okm := msgMap.msg[uid]
	// 删除临时下载的msg
	if !oks && okm && !m.isRecording && (m.danmuCancel == nil) {
		delete(msgMap.msg, uid)
	}
}
