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
	"sync"
	"time"
)

// 运行程序所在文件夹
var exeDir string

// chMap的锁
var chMutex sync.Mutex

// 每个streamer的控制管道
var chMap = make(map[uint]chan controlMsg)

type control int

// 控制信息
const (
	startCycle control = iota
	stopCycle
	//startRecord
	stopRecord
	//liveOff
	quit
)

// 管道信息
type controlMsg struct {
	s streamer
	c control
}

// 检查错误
func checkErr(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

// 获取时间
func getTime() string {
	t := time.Now()
	timeStr := fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	return timeStr
}

// 打印log信息
func logPrintln(log string) {
	timeStr := getTime()
	fmt.Println(timeStr + " " + log)
}

func (s streamer) uidStr() string {
	return fmt.Sprint(s.UID)
}

// 命令行参数处理
func argsHandle() bool {
	const usageStr = "本程序用于AcFun主播的开播提醒和自动下载直播"

	shortHelp := flag.Bool("h", false, "输出本帮助信息")
	longHelp := flag.Bool("help", false, "输出本帮助信息")
	addUID := flag.Uint("adduid", 0, "订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	delUID := flag.Uint("deluid", 0, "取消订阅指定主播的开播提醒，需要主播的uid（在主播的网页版个人主页查看）")
	addRecordUID := flag.Uint("addrecuid", 0, "自动下载指定主播的直播，需要主播的uid（在主播的网页版个人主页查看）")
	delRecordUID := flag.Uint("delrecuid", 0, "取消自动下载指定主播的直播，需要主播的uid（在主播的网页版个人主页查看）")
	addrestreamUID := flag.Uint("addrstuid", 0, "下载指定主播的直播同时将直播流推向本地UDP端口，节省边下载边观看同一直播的流量，但播放器的播放画面可能有点卡顿，需要主播的uid（在主播的网页版个人主页查看），需要事先设置自动下载指定主播的直播")
	delrestreamUID := flag.Uint("delrstuid", 0, "取消下载指定主播的直播同时将直播流推向本地端口，需要主播的uid（在主播的网页版个人主页查看）")
	isListen := flag.Bool("listen", false, "监听主播的直播状态，自动通知主播的直播状态或下载主播的直播，运行过程中如需更改设置又不想退出本程序，可以直接输入相应命令或手动修改设置文件"+configFile)
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
		if *addUID != 0 {
			addNotify(*addUID)
		}
		if *delUID != 0 {
			delNotify(*delUID)
		}
		if *addRecordUID != 0 {
			addRecord(*addRecordUID)
		}
		if *delRecordUID != 0 {
			delRecord(*delRecordUID)
		}
		if *addrestreamUID != 0 {
			addRestream(*addrestreamUID)
		}
		if *delrestreamUID != 0 {
			delRestream(*delrestreamUID)
		}
	}

	return *isListen
}

// 程序初始化
func initialize() {
	exePath, err := os.Executable()
	checkErr(err)
	exeDir = filepath.Dir(exePath)
	logoFileLocation = filepath.Join(exeDir, configFile)
	configFileLocation = filepath.Join(exeDir, configFile)

	_, err = os.Stat(logoFileLocation)
	if os.IsNotExist(err) {
		logPrintln("下载AcFun的logo")
		fetchAcLogo()
	}

	if !isConfigFileExist() {
		newConfigFile, err := os.Create(configFileLocation)
		checkErr(err)
		defer newConfigFile.Close()
		logPrintln("创建设置文件" + configFile)
	}
	loadConfig()
	oldStreamers = append([]streamer(nil), streamers...)
}

func main() {
	initialize()

	if argsHandle() {
		if len(streamers) == 0 {
			fmt.Println("请订阅指定主播的开播提醒，运行acfun_live -h查看帮助")
			return
		}

		logPrintln("本程序开始监听主播的直播状态")

		mainCh := make(chan controlMsg, 20)
		chMap[0] = mainCh

		for _, s := range streamers {
			go s.initCycle()
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go cycleConfig(ctx)

		fmt.Println("现在可以输入命令修改设置，输入help查看全部命令的解释")
		go handleInput()

		for {
			msg := <-mainCh
			switch msg.c {
			case startCycle:
				go msg.s.initCycle()
			case quit:
				// 结束cycleConfig()
				cancel()
				// 结束cycle()
				chMutex.Lock()
				for _, ch := range chMap {
					ch <- msg
				}
				chMutex.Unlock()
				// 结束下载直播
				recMutex.Lock()
				for _, rec := range recordMap {
					rec.ch <- stopRecord
					io.WriteString(rec.stdin, "q")
				}
				recMutex.Unlock()
				// 等待30秒，等待其他goroutine结束
				time.Sleep(30 * time.Second)
				logPrintln("本程序结束运行")
				return
			default:
				log.Println("未知controlMsg：", msg)
			}
		}
	}
}
