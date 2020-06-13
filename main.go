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

// 每个streamer的控制管道的map，map[uint]chan controlMsg
var chMap = sync.Map{}

type control int

// 控制信息
const (
	startCycle control = iota
	stopCycle
	liveOff
	stopRecord
	quit
)

// 管道信息
type controlMsg struct {
	s streamer
	c control
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
/*
func timePrintln(logs ...interface{}) {
	//logger.Print(getTime() + " ")
	logs = append([]interface{}{getTime()}, logs...)
	logger.Println(logs...)
}
*/

func lPrintln(msg ...interface{}) {
	logger.Println(msg...)
	fmt.Fprintln(&webLog, msg...)
}

// 将UID转换成字符串
func (s streamer) uidStr() string {
	return strconv.Itoa(int(s.UID))
}

func uidStr(uid uint) string {
	return strconv.Itoa(int(uid))
}

// 返回ID（UID）形式的字符串
func (s streamer) longID() string {
	return s.ID + "（" + s.uidStr() + "）"
}

// 获取sync.Map的长度
func length(sm *sync.Map) int {
	count := 0
	sm.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// 打印sync.Map的内容
func mapPrintln(sm *sync.Map) {
	sm.Range(func(key, value interface{}) bool {
		fmt.Println(key, value)
		return true
	})
}

// 命令行参数处理
func argsHandle() {
	const usageStr = "本程序用于AcFun主播的开播提醒和自动下载直播"

	shortHelp := flag.Bool("h", false, "输出本帮助信息")
	longHelp := flag.Bool("help", false, "输出本帮助信息")
	isListen = flag.Bool("listen", false, "监听主播的直播状态，自动通知主播的直播状态或下载主播的直播，运行过程中如需更改设置又不想退出本程序，可以直接输入相应命令或手动修改设置文件"+configFile)
	isWebServer = flag.Bool("weblisten", false, "监听主播的直播状态，自动通知主播的直播状态或下载主播的直播，可以通过http://localhost"+port+"来发送命令")
	isListLive := flag.Bool("listlive", false, "列出正在直播的主播")
	addNotifyUID := flag.Uint("addnotify", 0, "订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	delNotifyUID := flag.Uint("delnotify", 0, "取消订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	addRecordUID := flag.Uint("addrecord", 0, "自动下载指定主播的直播，需要主播的uid（在主播的网页版个人主页查看）")
	delRecordUID := flag.Uint("delrecord", 0, "取消自动下载指定主播的直播，需要主播的uid（在主播的网页版个人主页查看）")
	getStreamURL := flag.Uint("getdlurl", 0, "查看指定主播是否在直播，如在直播输出其直播源地址，需要主播的uid（在主播的网页版个人主页查看）")
	startRecord := flag.Uint("startrecord", 0, "临时下载指定主播的直播，需要主播的uid（在主播的网页版个人主页查看）")
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
			*isListen = true
		}
		if *isListLive {
			listLive()
		}
		if *addNotifyUID != 0 {
			addNotify(*addNotifyUID)
		}
		if *delNotifyUID != 0 {
			delNotify(*delNotifyUID)
		}
		if *addRecordUID != 0 {
			addRecord(*addRecordUID)
		}
		if *delRecordUID != 0 {
			delRecord(*delRecordUID)
		}
		if *getStreamURL != 0 {
			printStreamURL(*getStreamURL)
		}
		if *startRecord != 0 {
			startRec(*startRecord)
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
	loadConfig()
	streamers.old = append([]streamer(nil), streamers.current...)

	fetchAllRooms()
}

func main() {
	initialize()

	argsHandle()

	if *isListen {
		if len(streamers.current) == 0 {
			lPrintln("请订阅指定主播的开播提醒或自动下载，运行acfun_live -h查看帮助")
			return
		}

		lPrintln("本程序开始监听主播的直播状态")

		mainCh := make(chan controlMsg, 20)
		chMap.Store(0, mainCh)

		for _, s := range streamers.current {
			go s.initCycle()
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go cycleConfig(ctx)

		lPrintln("现在可以输入命令修改设置，输入help查看全部命令的解释")
		go handleInput()

		if *isWebServer {
			lPrintln("启动web服务，现在可以通过 http://localhost" + port + " 来发送命令")
			go httpServer()
		}

		for {
			select {
			case msg := <-mainCh:
				switch msg.c {
				case startCycle:
					go msg.s.initCycle()
				case quit:
					// 结束cycleConfig()
					cancel()
					// 结束cycle()
					lPrintln("正在退出各主播的循环")
					chMap.Range(func(key, value interface{}) bool {
						value.(chan controlMsg) <- msg
						return true
					})
					// 结束下载直播
					recordMap.Range(func(key, value interface{}) bool {
						rec := value.(record)
						rec.ch <- stopRecord
						io.WriteString(rec.stdin, "q")
						return true
					})
					danglingRec.mu.Lock()
					for _, rec := range danglingRec.records {
						rec.ch <- stopRecord
						io.WriteString(rec.stdin, "q")
					}
					danglingRec.mu.Unlock()
					// 关闭web服务
					if *isWebServer {
						lPrintln("正在关闭web服务")
						srv.Shutdown(context.TODO())
					}
					// 等待20秒，等待其他goroutine结束
					time.Sleep(20 * time.Second)
					lPrintln("本程序结束运行")
					return
				default:
					lPrintln("未知controlMsg：", msg)
				}
			default:
			}

			fetchAllRooms()

			// 每20秒循环一次
			time.Sleep(20 * time.Second)
		}
	}
}
