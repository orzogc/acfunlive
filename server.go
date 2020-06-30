// web服务相关
package main

import (
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
/startrecdan/数字：临时下载指定主播的直播视频和弹幕，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播视频和弹幕，这次为一次性的下载
/stoprecdan/数字：正在下载指定主播的直播视频和弹幕时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
/log ：查看log
/quit ：退出本程序，退出需要等待半分钟左右
/help ：本帮助信息`

// 储存日志
var webLog strings.Builder

var srv *http.Server

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

// 处理 "/cmd/UID"
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

// 显示favicon
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, logoFile)
}

// 打印日志
func logHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webLog.String())
}

// 打印帮助
func helpHandler(w http.ResponseWriter, r *http.Request) {
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
	r.HandleFunc("/favicon.ico", faviconHandler)
	r.HandleFunc("/log", logHandler)
	r.HandleFunc("/help", helpHandler)
	r.HandleFunc("/", helpHandler)
	r.HandleFunc("/{cmd}", cmdHandler)
	r.HandleFunc("/{cmd}/{uid:[1-9][0-9]*}", cmdUIDHandler)

	// 跨域处理
	handler := cors.Default().Handler(r)

	srv = &http.Server{
		Addr:         ":" + itoa(config.WebPort),
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
