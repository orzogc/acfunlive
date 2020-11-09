// web服务相关
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// web服务帮助信息
const webHelp = `/listlive ：列出正在直播的主播
/listrecord ：列出正在下载的直播视频
/listdanmu：列出正在下载的直播弹幕
/startwebui：启动web UI服务器
/stopwebui：停止web UI服务器
/liststreamer：列出设置了开播提醒或自动下载直播的主播
/startmirai：利用Mirai发送直播通知到指定QQ或QQ群
/addnotifyon/uid ：订阅指定主播的开播提醒，uid在主播的网页版个人主页查看
/delnotifyon/uid ：取消订阅指定主播的开播提醒
/addnotifyoff/uid ：订阅指定主播的下播提醒
/delnotifyoff/uid ：取消订阅指定主播的下播提醒
/addnotifyrecord/uid ：通知指定主播的直播视频下载
/delnotifyrecord/uid ：取消通知指定主播的直播视频下载
/addnotifydanmu/uid ：通知指定主播的直播弹幕下载
/delnotifydanmu/uid ：取消通知指定主播的直播弹幕下载
/addrecord/uid ：自动下载指定主播的直播视频
/delrecord/uid ：取消自动下载指定主播的直播视频
/adddanmu/uid ：自动下载指定主播的直播弹幕
/deldanmu/uid ：取消自动下载指定主播的直播弹幕
/addkeeponline/uid ：指定主播直播时在其直播间挂机
/delkeeponline/uid ：取消在指定主播直播时在其直播间挂机
/delconfig/uid：删除指定主播的所有设置
/getdlurl/uid ：查看指定主播是否在直播，如在直播输出其直播源地址
/addqq/uid/QQ号：设置将指定主播的开播提醒发送到指定QQ号
/delqq/uid：取消设置将指定主播的开播提醒发送到QQ
/addqqgroup/uid/QQ群号：设置将指定主播的开播提醒发送到指定QQ群号
/delqqgroup/uid：取消设置将指定主播的开播提醒发送到QQ群号
/startrecord/uid ：临时下载指定主播的直播视频，如果没有设置自动下载该主播的直播视频，这次为一次性的下载
/stoprecord/uid ：正在下载指定主播的直播视频时取消下载
/startdanmu/uid：临时下载指定主播的直播弹幕，如果没有设置自动下载该主播的直播弹幕，这次为一次性的下载
/stopdanmu/uid：正在下载指定主播的直播弹幕时取消下载
/startrecdan/uid：临时下载指定主播的直播视频和弹幕，如果没有设置自动下载该主播的直播视频和弹幕，这次为一次性的下载
/stoprecdan/uid：正在下载指定主播的直播视频和弹幕时取消下载
/log ：查看log
/quit ：退出本程序，退出需要等待半分钟左右
/help ：本帮助信息`

var apiSrv *http.Server

// 返回localhost地址和端口
func address(port int) string {
	return "http://localhost:" + itoa(port)
}

// 处理 "/cmd"
func cmdHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := vars["cmd"]
	w.Header().Set("Content-Type", "application/json")
	if s := handleCmd(cmd); s != "" {
		fmt.Fprint(w, s)
	} else {
		fmt.Fprint(w, "null")
	}
}

// 处理 "/cmd/uid"
func cmdUIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := vars["cmd"]
	uid, err := atoi(vars["uid"])
	checkErr(err)
	w.Header().Set("Content-Type", "application/json")
	if s := handleCmdUID(cmd, uid); s != "" {
		fmt.Fprint(w, s)
	} else {
		fmt.Fprint(w, "null")
	}
}

// 处理 "/cmd/uid/qq"
func cmdQQHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmd := vars["cmd"]
	uid, err := strconv.ParseUint(vars["uid"], 10, 64)
	checkErr(err)
	qq, err := strconv.ParseUint(vars["qq"], 10, 64)
	checkErr(err)
	w.Header().Set("Content-Type", "application/json")
	if s := handleCmdQQ(cmd, int(uid), int64(qq)); s != "" {
		fmt.Fprint(w, s)
	} else {
		fmt.Fprint(w, "null")
	}
}

// 显示favicon
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, logoFile)
}

// 打印日志
func logHandler(w http.ResponseWriter, r *http.Request) {
	logString.Lock()
	defer logString.Unlock()
	fmt.Fprint(w, logString.str.String())
}

// 打印帮助
func helpHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webHelp)
}

// 打印web请求
func printRequestURI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *isNoGUI {
			log.Println("处理web请求：" + r.RequestURI)
		}
		next.ServeHTTP(w, r)
	})
}

// web API服务器
func webAPI() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in webAPI(), the error is:", err)
			lPrintErr("web API服务器发生错误，尝试重启web API服务器")
			time.Sleep(2 * time.Second)
			go webAPI()
		}
	}()

	lPrintln("启动web API服务器，现在可以通过 " + address(config.WebPort) + " 来查看状态和发送命令")

	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", faviconHandler)
	r.HandleFunc("/log", logHandler)
	r.HandleFunc("/help", helpHandler)
	r.HandleFunc("/", helpHandler)
	r.HandleFunc("/{cmd}", cmdHandler)
	r.HandleFunc("/{cmd}/{uid:[1-9][0-9]*}", cmdUIDHandler)
	r.HandleFunc("/{cmd}/{uid:[1-9][0-9]*}/{qq:[1-9][0-9]*}", cmdQQHandler)
	r.Use(printRequestURI)

	// 跨域处理
	handler := cors.Default().Handler(r)

	apiSrv = &http.Server{
		Addr:         ":" + itoa(config.WebPort),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      handler,
	}

	err := apiSrv.ListenAndServe()
	if err != http.ErrServerClosed {
		lPrintln(err)
		panic(err)
	}
}

// 启动web API server
func startWebAPI() bool {
	if *isWebAPI {
		lPrintWarn("已经启动过web API服务器")
	} else {
		*isWebAPI = true
		go webAPI()
	}
	return true
}

// 停止web API server
func stopWebAPI() bool {
	if *isWebAPI {
		*isWebAPI = false
		lPrintln("停止web API服务器")
		ctx, cancel := context.WithCancel(mainCtx)
		defer cancel()
		if err := apiSrv.Shutdown(ctx); err != nil {
			lPrintErr("web API服务器关闭错误：", err)
			lPrintWarn("强行关闭web API服务器")
			cancel()
		}
	} else {
		lPrintWarn("没有启动web API服务器")
	}
	return true
}
