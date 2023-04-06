// 部分数据和函数定义
package main

import (
	"context"
	"fmt"
	"io"
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
	stopRecord
	quit
)

// 主播的管道信息
type controlMsg struct {
	s      streamer
	c      control
	liveID string
}

// 主播信息
type streamerInfo struct {
	//streamer
	ch     chan controlMsg // 控制信息
	modify bool
}

// 直播信息
type liveInfo struct {
	streamInfo
	uid          int                // 主播的uid
	streamURL    string             // 直播源链接
	isRecording  bool               // 是否正在下载直播
	isDanmu      bool               // 是否正在下载直播弹幕
	isKeepOnline bool               // 是否正在直播间挂机
	recordCh     chan control       // 控制录播的管道
	ffmpegStdin  io.WriteCloser     // ffmpeg的stdin
	recordCancel context.CancelFunc // 用来强行停止ffmpeg运行
	danmuCancel  context.CancelFunc // 用来停止下载弹幕
	onlineCancel context.CancelFunc // 用来停止直播间挂机
	recordFile   string             // 录播文件路径
	assFile      string             // 弹幕文件路径
}

// 直播源信息
type streamInfo struct {
	acfundanmu.StreamInfo        // 直播源信息
	hlsURL                string // hls直播源
	flvURL                string // flv直播源
	cfg                   acfundanmu.SubConfig
}

// streamerInfo的map
var sInfoMap struct {
	sync.Mutex
	info map[int]*streamerInfo
}

// liveInfo的map
var lInfoMap struct {
	sync.RWMutex
	info map[string]liveInfo
}

// 储存日志
var logString struct {
	sync.Mutex
	str strings.Builder
}

// AcFun帐号的cookies
var acfunCookies struct {
	sync.RWMutex
	time    time.Time
	cookies acfundanmu.Cookies
}

// 设备ID
var deviceID string

var (
	exeDir    string                                  // 运行程序所在文件夹
	mainCh    chan controlMsg                         // main()的管道
	mainCtx   context.Context                         // main()的ctx
	isListen  *bool                                   // 程序是否处于监听状态
	isWebAPI  *bool                                   // 程序是否启动web API服务器
	isWebUI   *bool                                   // 程序是否启动web UI服务器
	configDir *string                                 // 设置文件所在文件夹
	recordDir *string                                 // 下载录播和弹幕时保存的文件夹
	isNoGUI   = new(bool)                             // 程序是否启动GUI界面
	logger    = log.New(os.Stdout, "", log.LstdFlags) // 可以同步输出的logger
	atoi      = strconv.Atoi                          // 将字符串转换为int
	boolStr   = strconv.FormatBool                    // 将bool类型转换为字符串
)

// 检查错误
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// 尝试运行，三次出错后结束运行
func runThrice(f func() error) error {
	var err error
	for retry := 0; retry < 3; retry++ {
		if err = f(); err != nil {
			log.Printf("%v", err)
		} else {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("运行三次都出现错误：%v", err)
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
func lPrintln(msg ...any) {
	if *isNoGUI {
		logger.Println(msg...)
	}
	// 同时输出日志到web服务
	logString.Lock()
	defer logString.Unlock()
	// 防止内存泄漏
	if logString.str.Len() > 1000000 {
		logString.str.Reset()
	}
	fmt.Fprint(&logString.str, getLogTime()+" ")
	fmt.Fprintln(&logString.str, msg...)
}

// 打印警告信息
func lPrintWarn(msg ...any) {
	w := []any{"[WARN]"}
	msg = append(w, msg...)
	lPrintln(msg...)
}

// 打印错误信息
func lPrintErr(msg ...any) {
	e := []any{"[ERROR]"}
	msg = append(e, msg...)
	lPrintln(msg...)
}

// 打印带时间戳的log信息（格式化字符串）
func lPrintf(format string, a ...any) {
	lPrintln(fmt.Sprintf(format, a...))
}

// 打印警告信息（格式化字符串）
func lPrintWarnf(format string, a ...any) {
	lPrintWarn(fmt.Sprintf(format, a...))
}

// 打印错误信息（格式化字符串）
func lPrintErrf(format string, a ...any) {
	lPrintErr(fmt.Sprintf(format, a...))
}

// 返回ID（UID）形式的字符串
func (s *streamer) longID() string {
	return fmt.Sprintf("%s（%d）", s.Name, s.UID)
}

// 返回ID（UID）形式的字符串
func longID(uid int) string {
	return fmt.Sprintf("%s（%d）", getName(uid), uid)
}

// 根据uid获取liveInfo
func getLiveInfoByUID(uid int) (infoList []liveInfo, ok bool) {
	lInfoMap.RLock()
	defer lInfoMap.RUnlock()
	for _, info := range lInfoMap.info {
		if info.uid == uid {
			infoList = append(infoList, info)
			ok = true
		}
	}
	return infoList, ok
}

// 根据liveID获取liveInfo
func getLiveInfo(liveID string) (liveInfo, bool) {
	lInfoMap.RLock()
	defer lInfoMap.RUnlock()
	if info, ok := lInfoMap.info[liveID]; ok {
		return info, true
	}
	return liveInfo{}, false
}

// 将info放进lInfoMap里
func setLiveInfo(info liveInfo) {
	lInfoMap.Lock()
	defer lInfoMap.Unlock()
	lInfoMap.info[info.LiveID] = info
}

// 根据liveID查询是否正在下载直播视频
func isRecording(liveID string) bool {
	if info, ok := getLiveInfo(liveID); ok {
		return info.isRecording
	}
	return false
}

// 根据liveID查询是否正在下载直播弹幕
func isDanmu(liveID string) bool {
	if info, ok := getLiveInfo(liveID); ok {
		return info.isDanmu
	}
	return false
}

// 是否登陆AcFun帐号
func is_login_acfun() bool {
	acfunCookies.RLock()
	defer acfunCookies.RUnlock()
	return len(acfunCookies.cookies) != 0
}

// 登陆AcFun帐号
func acfun_login() error {
	if config.Acfun.Account != "" && config.Acfun.Password != "" {
		acfunCookies.Lock()
		defer acfunCookies.Unlock()
		cookies, err := acfundanmu.Login(config.Acfun.Account, config.Acfun.Password)
		if err != nil {
			return err
		}
		acfunCookies.cookies = cookies
		acfunCookies.time = time.Now()
		return nil
	}
	return fmt.Errorf("没有设置AcFun帐号或密码")
}

// 返回AcFun帐号的cookies
func acfun_cookies() acfundanmu.Cookies {
	if config.Acfun.Account != "" && config.Acfun.Password != "" && is_login_acfun() {
		acfunCookies.RLock()
		cookies_time := time.Since(acfunCookies.time)
		acfunCookies.RUnlock()
		// 20天后重新登陆A站帐号
		if cookies_time > 480*time.Hour {
			err := acfun_login()
			if err != nil {
				lPrintErrf("重新登陆AcFun帐号时出现错误：%v", err)
			} else {
				lPrintln("重新登陆AcFun帐号成功")
			}
		}
		acfunCookies.RLock()
		defer acfunCookies.RUnlock()
		return append(acfundanmu.Cookies{}, acfunCookies.cookies...)
	}
	return nil
}
