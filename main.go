// AcFun直播通知和下载助手
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"
	"github.com/orzogc/acfundanmu"
)

// 命令行参数处理
func argsHandle() {
	const usageStr = "本程序用于AcFun主播的开播提醒和自动下载直播"

	shortHelp := flag.Bool("h", false, "输出本帮助信息")
	longHelp := flag.Bool("help", false, "输出本帮助信息")
	isListen = flag.Bool("listen", false, "监听主播的直播状态，自动通知主播的直播状态或下载主播的直播，运行过程中如需更改设置又不想退出本程序，可以直接输入相应命令或手动修改设置文件"+liveFile)
	isWebAPI = flag.Bool("webapi", false, "启动web API服务器，可以通过 "+address(config.WebPort)+" 来查看状态和发送命令，需要listen参数")
	isWebUI = flag.Bool("webui", false, "启动web UI服务器，可以通过 "+address(config.WebPort+10)+" 访问web UI界面，需要webapi参数")
	isNoGUI = flag.Bool("nogui", false, "不使用GUI界面")
	isMirai = flag.Bool("mirai", false, "利用Mirai发送直播通知到指定QQ或QQ群，需要listen参数")
	isCoolq = flag.Bool("coolq", false, "使用酷Q发送直播通知到指定QQ或QQ群，需要事先设置并启动酷Q，需要listen参数")
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
	startRecDanmu := flag.Uint("startrecdan", 0, "临时下载指定主播的直播视频和弹幕，需要主播的uid（在主播的网页版个人主页查看）")
	flag.Parse()

	initialize()

	if flag.NArg() != 0 {
		lPrintErr("请输入正确的参数")
		if *isNoGUI {
			fmt.Println(usageStr)
			flag.PrintDefaults()
		}
	} else {
		if *shortHelp || *longHelp {
			if *isNoGUI {
				fmt.Println(usageStr)
				flag.PrintDefaults()
			}
		}
		if !*isNoGUI {
			*isListen = true
			*isWebAPI = true
			*isWebUI = true
		}
		if *isWebUI {
			if !*isWebAPI {
				lPrintErr("webui参数需要和webapi参数一起运行")
				os.Exit(1)
			}
		}
		if *isWebAPI || *isCoolq {
			if !*isListen {
				lPrintErr("webapi和coolq参数需要和listen参数一起运行")
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
			startRec(int(*startRecord), false)
		}
		if *startDlDanmu != 0 {
			startDanmu(int(*startDlDanmu))
		}
		if *startRecDanmu != 0 {
			startRecDan(int(*startRecDanmu))
		}
	}
}

// 检查config.json里的配置
func checkConfig() {
	if config.Source != "hls" && config.Source != "flv" {
		lPrintErr(configFile + "里的Source必须是hls或flv")
		os.Exit(1)
	}
	if config.WebPort < 1024 || config.WebPort > 65525 {
		lPrintErr(configFile + "里的WebPort必须大于1023且少于65526")
		os.Exit(1)
	}
	if config.Directory != "" {
		info, err := os.Stat(config.Directory)
		checkErr(err)
		if !info.IsDir() {
			lPrintErr(configFile + "里的Directory必须是存在的文件夹")
			os.Exit(1)
		}
	}
	if config.Mirai.AdminQQ < 0 || config.Mirai.BotQQ < 0 || config.Coolq.AdminQQ < 0 {
		lPrintErr(configFile + "里的QQ号必须大于等于0")
		os.Exit(1)
	}
}

// 程序初始化
func initialize() {
	// 避免 initialization loop
	boolDispatch["startwebapi"] = startWebAPI
	boolDispatch["startwebui"] = startWebUI
	boolDispatch["startmirai"] = startMirai
	boolDispatch["startcoolq"] = startCoolq

	exePath, err := os.Executable()
	checkErr(err)
	exeDir = filepath.Dir(exePath)
	logoFileLocation = filepath.Join(exeDir, logoFile)
	liveFileLocation = filepath.Join(exeDir, liveFile)
	configFileLocation = filepath.Join(exeDir, configFile)

	if _, err := os.Stat(logoFileLocation); os.IsNotExist(err) {
		lPrintln("下载AcFun的logo")
		fetchAcLogo()
	}

	if !isConfigFileExist(liveFile) {
		err = ioutil.WriteFile(liveFileLocation, []byte("[]"), 0644)
		checkErr(err)
		lPrintln("创建设置文件" + liveFile)
	}
	if !isConfigFileExist(configFile) {
		data, err := json.MarshalIndent(config, "", "    ")
		checkErr(err)
		err = ioutil.WriteFile(configFileLocation, data, 0644)
		checkErr(err)
		lPrintln("创建设置文件" + configFile)
	}

	msgMap.msg = make(map[int]*sMsg)
	streamers.crt = make(map[int]streamer)
	streamers.old = make(map[int]streamer)
	loadLiveConfig()
	loadConfig()
	checkConfig()

	for uid, s := range streamers.crt {
		streamers.old[uid] = s
	}

	getDidCookie()
	fetchAllRooms()
	liveRooms.rooms = liveRooms.newRooms

	if config.Acfun.UserEmail != "" && config.Acfun.Password != "" {
		acfunCookies, err = acfundanmu.Login(config.Acfun.UserEmail, config.Acfun.Password)
		if err != nil {
			lPrintErrf("登陆AcFun帐号时出现错误，取消登陆：%v", err)
			acfunCookies = nil
		} else if len(acfunCookies) != 0 {
			lPrintln("登陆AcFun帐号成功")
		}
	}
}

func main() {
	argsHandle()

	if *isListen {
		if len(streamers.crt) == 0 {
			if *isNoGUI {
				lPrintWarn("请订阅指定主播的开播提醒或自动下载，输入 help 查看帮助")
			} else {
				lPrintWarn("请在web UI界面订阅指定主播的开播提醒或自动下载")
			}
		}

		lPrintln("本程序开始监听主播的直播状态")

		mainCh = make(chan controlMsg, 20)

		if *isMirai {
			lPrintln("尝试利用Mirai登陆bot QQ", config.Mirai.BotQQ)
			if config.Mirai.BotQQ <= 0 || config.Mirai.BotQQPassword == "" {
				lPrintErr("请先在" + configFile + "里设置好Mirai相关配置")
			} else if !initMirai() {
				lPrintErr("启动Mirai失败，请重新启动本程序")
				*isMirai = false
			}
		}

		if *isCoolq {
			lPrintln("尝试通过 " + config.Coolq.CqhttpWSAddr + " 连接酷Q")
			initCoolq()
		}

		for _, s := range streamers.crt {
			go s.cycle()
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mainCtx = ctx

		go cycleConfig(ctx)

		// 启动GUI时不需要处理命令输入
		if *isNoGUI {
			lPrintln("现在可以输入命令控制本程序，输入 help 查看全部命令的解释")
			go handleInput()
		}

		if *isWebAPI {
			go webAPI()
		}

		if *isWebUI {
			go startUI()
		}

		go cycleFetch(ctx)

		if !*isNoGUI {
			go systray.Run(trayOnReady, trayOnExit)
		}

		for {
			select {
			case msg := <-mainCh:
				switch msg.c {
				case startCycle:
					go msg.s.cycle()
				case quit:
					// 退出systray
					if !*isNoGUI {
						systray.Quit()
					}
					// 停止web UI服务器
					if *isWebUI {
						stopWebUI()
					}
					// 停止web API服务器
					if *isWebAPI {
						stopWebAPI()
					}
					// 结束所有mainCtx的子ctx
					cancel()
					// 结束cycle()
					lPrintln("正在退出各主播的循环")
					msgMap.Lock()
					for _, m := range msgMap.msg {
						// 退出各主播的循环
						if m.ch != nil {
							m.ch <- msg
						}
						// 结束下载直播视频
						if m.isRecording {
							m.rec.ch <- stopRecord
							io.WriteString(m.rec.stdin, "q")
						}
						// 结束下载弹幕
						if m.danmuCancel != nil {
							m.danmuCancel()
						}
					}
					msgMap.Unlock()
					danglingRec.Lock()
					for _, rec := range danglingRec.records {
						rec.ch <- stopRecord
						io.WriteString(rec.stdin, "q")
					}
					danglingRec.Unlock()
					// 等待20秒，等待其他goroutine结束
					time.Sleep(20 * time.Second)
					lPrintln("本程序结束运行")
					return
				default:
					lPrintErr("未知controlMsg：", msg)
				}
			}
		}
	}
}
