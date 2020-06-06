// AcFun直播通知和下载助手
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	//stopRecord
	quit
)

// 管道信息
type controlMsg struct {
	s streamer
	c control
}

// 帮助信息
const helpMsg = `adduid 数字：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
deluid 数字：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
addrecuid 数字：自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
delrecuid 数字：取消自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
addrstuid 数字：下载指定主播的直播同时将直播流推向本地UDP端口，节省边下载边观看同一直播的流量，但播放器的播放画面可能有点卡顿，数字为主播的uid（在主播的网页版个人主页查看），需要事先设置自动下载指定主播的直播
delrstuid 数字：取消下载指定主播的直播同时将直播流推向本地端口，数字为主播的uid（在主播的网页版个人主页查看）
startrecord 数字：临时下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
startrecrst 数字：临时下载指定主播的直播并将直播流推向本地UDP端口，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
stoprecord 数字：正在下载指定主播的直播时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
quit：退出本程序，退出需要等待半分钟
help：本帮助信息`

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

// 处理管道信号
func (s streamer) handleMsg(msg controlMsg) {
	switch msg.c {
	case startCycle:
		logPrintln("重启监听" + s.ID + "（" + s.uidStr() + "）" + "的直播状态")
		chMutex.Lock()
		ch := chMap[0]
		chMutex.Unlock()
		ch <- msg
	case stopCycle:
		logPrintln("删除" + s.ID)
		chMutex.Lock()
		delete(chMap, s.UID)
		chMutex.Unlock()
	case quit:
		logPrintln("正在退出" + s.ID + "（" + s.uidStr() + "）" + "的循环")
	default:
		log.Println("未知controlMsg：", msg)
	}
}

// 循环获取指定主播的直播状态，通知开播和自动下载直播
func (s streamer) cycle() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in cycle(), the error is:", err)
			log.Println(s.ID + "（" + s.uidStr() + "）" + "的循环处理发生错误")
			restart := controlMsg{s: s, c: startCycle}
			chMutex.Lock()
			ch := chMap[0]
			chMutex.Unlock()
			ch <- restart
		}
	}()

	const livePage = "https://live.acfun.cn/live/"

	chMutex.Lock()
	ch := chMap[s.UID]
	chMutex.Unlock()

	// 设置文件里有该主播，但是不通知不下载
	if !(s.Notify || s.Record) {
		for {
			msg := <-ch
			s.handleMsg(msg)
			return
		}
	}

	logPrintln("开始监听" + s.ID + "（" + s.uidStr() + "）" + "的直播状态")

	isLive := false
	for {
		select {
		case msg := <-ch:
			s.handleMsg(msg)
			return
		default:
			if s.isLiveOn() {
				if !isLive {
					isLive = true
					title := s.getTitle()
					logPrintln(s.ID + "（" + s.uidStr() + "）" + "正在直播：")
					fmt.Println(title)
					hlsURL, flvURL := s.getStreamURL()
					if hlsURL == "" {
						log.Println("无法获取" + s.ID + "的直播源，尝试重启循环")
						restart := controlMsg{s: s, c: startCycle}
						chMutex.Lock()
						ch := chMap[0]
						chMutex.Unlock()
						ch <- restart
						return
					}
					fmt.Println(s.ID + "的直播观看地址：")
					fmt.Println(livePage + s.uidStr())
					fmt.Println(s.ID + "直播源的hls和flv地址分别是：")
					fmt.Println(hlsURL)
					fmt.Println(flvURL)

					if s.Notify {
						desktopNotify(s.ID + "正在直播")
					}
					if s.Record {
						// 下载hls直播源，想下载flv直播源的话可手动更改此处
						go s.recordLive(hlsURL)
					}
				}
			} else {
				if isLive {
					logPrintln(s.ID + "（" + s.uidStr() + "）" + "已经下播")
					if s.Notify {
						desktopNotify(s.ID + "已经下播")
					}
				}
				isLive = false
			}

			// 大约每二十几秒获取一次主播的直播状态
			rand.Seed(time.Now().UnixNano())
			min := 20
			max := 30
			duration := rand.Intn(max-min+1) + min
			time.Sleep(time.Duration(duration) * time.Second)
		}
	}
}

// 完成对cycle()的初始化
func (s streamer) initCycle() {
	controlCh := make(chan controlMsg, 20)
	chMutex.Lock()
	chMap[s.UID] = controlCh
	chMutex.Unlock()
	s.cycle()
}

// 打印错误命令信息
func printErr() {
	fmt.Println("请输入正确的命令，输入help查看全部命令的解释")
}

// 处理输入
func handleInput() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in handleInput(), the error is:", err)
			log.Println("输入处理发生错误")
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := strings.Fields(scanner.Text())
		if len(cmd) == 1 {
			switch cmd[0] {
			case "help":
				fmt.Println(helpMsg)
			case "quit":
				fmt.Println("正在准备退出，请等待...")
				chMutex.Lock()
				ch := chMap[0]
				chMutex.Unlock()
				q := controlMsg{c: quit}
				ch <- q
				return
			default:
				printErr()
			}
		} else if len(cmd) == 2 {
			switch cmd[0] {
			case "adduid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addNotify(uint(uid))
			case "deluid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delNotify(uint(uid))
			case "addrecuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addRecord(uint(uid))
			case "delrecuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delRecord(uint(uid))
			case "addrstuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				addRestream(uint(uid))
			case "delrstuid":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				delRestream(uint(uid))
			case "startrecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				startRec(uint(uid), false)
			case "startrecrst":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				startRec(uint(uid), true)
			case "stoprecord":
				uid, err := strconv.ParseUint(cmd[1], 10, 64)
				if err != nil {
					printErr()
					break
				}
				stopRec(uint(uid))
			default:
				printErr()
			}
		} else {
			printErr()
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("Reading standard input err:", err)
	}
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
				cancel()
				chMutex.Lock()
				for _, ch := range chMap {
					ch <- msg
				}
				chMutex.Unlock()
				recMutex.Lock()
				for _, rec := range recordMap {
					stdin := rec.stdin
					io.WriteString(stdin, "q")
				}
				recMutex.Unlock()
				time.Sleep(30 * time.Second)
				logPrintln("本程序结束运行")
				return
			default:
				log.Println("未知controlMsg：", msg)
			}
		}
	}
}
