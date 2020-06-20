// web服务相关
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// web服务帮助信息
const webHelp = `/listlive ：列出正在直播的主播
/listrecord ：列出正在下载的直播视频
/listdanmu：列出正在下载的直播弹幕
/liststreamer：列出设置了开播提醒或自动下载直播的主播
/addnotify/数字 ：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
/delnotify/数字 ：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
/addrecord/数字 ：自动下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看）
/delrecord/数字 ：取消自动下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看）
/adddanmu/数字 ：自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
/deldanmu/数字 ：取消自动下载指定主播的直播弹幕，需要主播的uid（在主播的网页版个人主页查看）
/getdlurl/数字 ：查看指定主播是否在直播，如在直播输出其直播源地址，数字为主播的uid（在主播的网页版个人主页查看）
/startrecord/数字 ：临时下载指定主播的直播视频，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播视频，这次为一次性的下载
/stoprecord/数字 ：正在下载指定主播的直播视频时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
/startdanmu/数字：临时下载指定主播的直播弹幕，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播弹幕，这次为一次性的下载
/stopdanmu/数字：正在下载指定主播的直播弹幕时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
/log ：查看log
/quit ：退出本程序，退出需要等待半分钟左右
/help ：本帮助信息`

// web服务本地默认端口
const port = ":51880"

var listDispatch = map[string]func() []streaming{
	"listlive":   listLive,
	"listrecord": listRecord,
	"listdanmu":  listDanmu,
}

// 储存日志
var webLog strings.Builder

var srv *http.Server

// 处理dispatch
func handleDispatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := atoi(vars["uid"])
	checkErr(err)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, dispatch[mux.CurrentRoute(r).GetName()](uid))
}

// 处理listDispatch
func handleListDispatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	j := json.NewEncoder(w)
	j.SetIndent("", "    ")
	err := j.Encode(listDispatch[mux.CurrentRoute(r).GetName()]())
	checkErr(err)
}

// 列出直播的下载源
func handleStreamURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := atoi(vars["uid"])
	checkErr(err)
	hlsURL, flvURL := printStreamURL(uid)
	w.Header().Set("Content-Type", "application/json")
	j := json.NewEncoder(w)
	j.SetIndent("", "    ")
	err = j.Encode([]string{hlsURL, flvURL})
	checkErr(err)
}

// 列出设置里的主播
func handleListStreamer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	j := json.NewEncoder(w)
	j.SetIndent("", "    ")
	err := j.Encode(getStreamers())
	checkErr(err)
}

// 打印日志
func handleLog(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webLog.String())
}

// 退出程序
func handleQuit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, true)
	quitRun()
}

// 打印帮助
func handleHelp(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webHelp)
}

// web服务
func httpServer() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in httpServer(), the error is:", err)
			lPrintln("web服务发生错误，尝试重启web服务")
			time.Sleep(2 * time.Second)
			go httpServer()
		}
	}()

	r := mux.NewRouter()
	for str := range dispatch {
		r.HandleFunc(fmt.Sprintf("/%s/{uid:[1-9][0-9]*}", str), handleDispatch).Name(str)
	}
	for str := range listDispatch {
		r.HandleFunc(fmt.Sprintf("/%s", str), handleListDispatch).Name(str)
	}
	r.HandleFunc("/getdlurl/{uid:[1-9][0-9]*}", handleStreamURL)
	r.HandleFunc("/liststreamer", handleListStreamer)
	r.HandleFunc("/log", handleLog)
	r.HandleFunc("/quit", handleQuit)
	r.HandleFunc("/help", handleHelp)
	r.HandleFunc("/", handleHelp)

	// 跨域处理
	handler := cors.Default().Handler(r)

	srv = &http.Server{
		Addr:         port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      handler,
	}

	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		lPrintln(err)
		panic(err)
	}
}
