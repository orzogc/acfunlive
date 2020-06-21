// AcFun直播通知和下载助手
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// 运行程序所在文件夹
var exeDir string

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
	recording   bool               // 是否正在下载
	modify      bool               // 是否被修改设置
	danmuCancel context.CancelFunc // 用来停止下载弹幕
}

// sMsg的map
var msgMap struct {
	mu  sync.Mutex
	msg map[int]sMsg
}

// 程序是否处于监听状态
var isListen *bool

// 程序是否启动web服务
var isWebServer *bool

// 可以同步输出的logger
var logger = log.New(os.Stdout, "", log.LstdFlags)

// 检查错误
func checkErr(err error) {
	if err != nil {
		lPrintln(err)
		panic(err)
	}
}

// 获取时间
func getTime() string {
	t := time.Now()
	timeStr := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	return timeStr
}

// 打印带时间戳的log信息
func lPrintln(msg ...interface{}) {
	logger.Println(msg...)
	// 同时输出日志到web服务
	fmt.Fprintln(&webLog, msg...)
}

// 将int转换为字符串
var itoa = strconv.Itoa

// 将字符串转换为int
var atoi = strconv.Atoi

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
	streamers.mu.Lock()
	defer streamers.mu.Unlock()
	msgMap.mu.Lock()
	defer msgMap.mu.Unlock()
	_, oks := streamers.crt[uid]
	m, okm := msgMap.msg[uid]
	// 删除临时下载的msg
	if !oks && okm && !m.recording && (m.danmuCancel == nil) {
		delete(msgMap.msg, uid)
	}
}

// 命令行参数处理
func argsHandle() {
	const usageStr = "本程序用于AcFun主播的开播提醒和自动下载直播"

	shortHelp := flag.Bool("h", false, "输出本帮助信息")
	longHelp := flag.Bool("help", false, "输出本帮助信息")
	isListen = flag.Bool("listen", false, "监听主播的直播状态，自动通知主播的直播状态或下载主播的直播，运行过程中如需更改设置又不想退出本程序，可以直接输入相应命令或手动修改设置文件"+configFile)
	isWebServer = flag.Bool("web", false, "启动web服务，可以通过 http://localhost"+port+" 来查看状态和发送命令，需要listen参数")
	isListLive := flag.Bool("listlive", false, "列出正在直播的主播")
	addNotifyUID := flag.Uint("addnotify", 0, "订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	delNotifyUID := flag.Uint("delnotify", 0, "取消订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	addRecordUID := flag.Uint("addrecord", 0, "自动下载指定主播的直播视频，需要主播的uid（在主播的网页版个人主页查看）")
	delRecordUID := flag.Uint("delrecord", 0, "取消自动下载指定主播的直播视频，需要主播的uid（在主播的网页版个人主页查看）")
	addDanmuUID := flag.Uint("adddanmu", 0, "自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）")
	delDanmuUID := flag.Uint("deldanmu", 0, "取消自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）")
	getStreamURL := flag.Uint("getdlurl", 0, "查看指定主播是否在直播，如在直播输出其直播源地址，需要主播的uid（在主播的网页版个人主页查看）")
	startRecord := flag.Uint("startrecord", 0, "临时下载指定主播的直播视频，需要主播的uid（在主播的网页版个人主页查看）")
	startDlDanmu := flag.Uint("startdanmu", 0, "临时下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）")
	flag.Parse()

	if flag.NArg() != 0 || flag.NFlag() == 0 {
		fmt.Println("请输入正确的参数")
		fmt.Println(usageStr)
		flag.PrintDefaults()
	} else {
		if *shortHelp || *longHelp {
			fmt.Println(usageStr)
			flag.PrintDefaults()
		}
		if *isWebServer {
			if *isListen != true {
				fmt.Println("web参数需要和listen参数一起运行")
				os.Exit(1)
			}
		}
		if *isListLive {
			listLive()
		}
		if *addNotifyUID != 0 {
			addNotify(int(*addNotifyUID))
		}
		if *delNotifyUID != 0 {
			delNotify(int(*delNotifyUID))
		}
		if *addRecordUID != 0 {
			addRecord(int(*addRecordUID))
		}
		if *delRecordUID != 0 {
			delRecord(int(*delRecordUID))
		}
		if *addDanmuUID != 0 {
			addDanmu(int(*addDanmuUID))
		}
		if *delDanmuUID != 0 {
			delDanmu(int(*delDanmuUID))
		}
		if *getStreamURL != 0 {
			printStreamURL(int(*getStreamURL))
		}
		if *startRecord != 0 {
			startRec(int(*startRecord))
		}
		if *startDlDanmu != 0 {
			startDanmu(int(*startDlDanmu))
		}
	}
}

// 程序初始化
func initialize() {
	exePath, err := os.Executable()
	checkErr(err)
	exeDir = filepath.Dir(exePath)
	logoFileLocation = filepath.Join(exeDir, logoFile)
	configFileLocation = filepath.Join(exeDir, configFile)

	_, err = os.Stat(logoFileLocation)
	if os.IsNotExist(err) {
		lPrintln("下载AcFun的logo")
		fetchAcLogo()
	}

	if !isConfigFileExist() {
		newConfigFile, err := os.Create(configFileLocation)
		checkErr(err)
		defer newConfigFile.Close()
		_, err = newConfigFile.WriteString("[]")
		checkErr(err)
		lPrintln("创建设置文件" + configFile)
	}
	msgMap.msg = make(map[int]sMsg)
	streamers.crt = make(map[int]streamer)
	streamers.old = make(map[int]streamer)
	loadConfig()

	for uid, s := range streamers.crt {
		streamers.old[uid] = s
	}

	fetchAllRooms()
}

func main() {
	initialize()

	argsHandle()

	if *isListen {
		if len(streamers.crt) == 0 {
			lPrintln("请订阅指定主播的开播提醒或自动下载，运行acfun_live -h查看帮助")
			return
		}

		lPrintln("本程序开始监听主播的直播状态")

		mainCh := make(chan controlMsg, 20)
		msgMap.msg[0] = sMsg{ch: mainCh}

		for _, s := range streamers.crt {
			go s.cycle()
		}

		ctx, configCancel := context.WithCancel(context.Background())
		defer configCancel()
		go cycleConfig(ctx)

		lPrintln("现在可以输入命令修改设置，输入help查看全部命令的解释")
		go handleInput()

		if *isWebServer {
			lPrintln("启动web服务，现在可以通过 http://localhost" + port + " 来查看状态和发送命令")
			go httpServer()
		}

		ctx, fetchCancel := context.WithCancel(context.Background())
		defer fetchCancel()
		go cycleFetch(ctx)

		for {
			select {
			case msg := <-mainCh:
				switch msg.c {
				case startCycle:
					go msg.s.cycle()
				case quit:
					// 结束cycleConfig()
					configCancel()
					// 结束cycleFetch()
					fetchCancel()
					// 结束cycle()
					lPrintln("正在退出各主播的循环")
					msgMap.mu.Lock()
					for _, m := range msgMap.msg {
						// 退出各主播的循环
						if m.ch != nil {
							m.ch <- msg
						}
						// 结束下载直播视频
						if m.recording {
							m.rec.ch <- stopRecord
							io.WriteString(m.rec.stdin, "q")
						}
						// 结束下载弹幕
						if m.danmuCancel != nil {
							m.danmuCancel()
						}
					}
					msgMap.mu.Unlock()
					danglingRec.mu.Lock()
					for _, rec := range danglingRec.records {
						rec.ch <- stopRecord
						io.WriteString(rec.stdin, "q")
					}
					danglingRec.mu.Unlock()
					// 停止web服务
					if *isWebServer {
						lPrintln("正在停止web服务")
						srv.Shutdown(context.TODO())
					}
					// 等待20秒，等待其他goroutine结束
					time.Sleep(20 * time.Second)
					lPrintln("本程序结束运行")
					return
				default:
					lPrintln("未知controlMsg：", msg)
				}
			}
		}
	}
}
