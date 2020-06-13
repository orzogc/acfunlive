package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const webHelp = `/listlive ：列出正在直播的主播
/listrecord ：列出正在下载的直播
/addnotify/数字 ：订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
/delnotify/数字 ：取消订阅指定主播的开播提醒，数字为主播的uid（在主播的网页版个人主页查看）
/addrecord/数字 ：自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
/delrecord/数字 ：取消自动下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看）
/getdlurl/数字 ：查看指定主播是否在直播，如在直播输出其直播源地址，数字为主播的uid（在主播的网页版个人主页查看）
/startrecord/数字 ：临时下载指定主播的直播，数字为主播的uid（在主播的网页版个人主页查看），如果没有设置自动下载该主播的直播，这次为一次性的下载
/stoprecord/数字 ：正在下载指定主播的直播时取消下载，数字为主播的uid（在主播的网页版个人主页查看）
/log ：查看log
/quit ：退出本程序，退出需要等待半分钟左右
/help ：本帮助信息`

const port = ":51880"

var dispatch = map[string]func(uint) bool{
	"addnotify":   addNotify,
	"delnotify":   delNotify,
	"addrecord":   addRecord,
	"delrecord":   delRecord,
	"startrecord": startRec,
	"stoprecord":  stopRec,
}

var webLog strings.Builder

func handleDispatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := strconv.Atoi(vars["uid"])
	checkErr(err)
	fmt.Fprint(w, dispatch[mux.CurrentRoute(r).GetName()](uint(uid)))
}

func handleStreamURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid, err := strconv.Atoi(vars["uid"])
	checkErr(err)
	hlsURL, flvURL := printStreamURL(uint(uid))
	fmt.Fprint(w, hlsURL+"\n"+flvURL)
}

func handleListLive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, listLive())
}

func handleListRecord(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, listRecord())
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webLog.String())
}

func handleQuit(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "quit")
	quitRun()
}

func handleHelp(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, webHelp)
}

func server() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in server(), the error is:", err)
			lPrintln("web服务器发生错误，尝试重启web服务器")
			time.Sleep(2 * time.Second)
			go server()
		}
	}()

	r := mux.NewRouter()
	s := r.Methods("GET").Subrouter()
	for str := range dispatch {
		s.HandleFunc(fmt.Sprintf("/%s/{uid:[1-9][0-9]*}", str), handleDispatch).Name(str)
	}
	s.HandleFunc("/getdlurl/{uid:[1-9][0-9]*}", handleStreamURL)
	s.HandleFunc("/listlive", handleListLive)
	s.HandleFunc("/listrecord", handleListRecord)
	s.HandleFunc("/log", handleLog)
	s.HandleFunc("/quit", handleQuit)
	s.HandleFunc("/help", handleHelp)
	s.HandleFunc("/", handleHelp)

	log.Fatal(http.ListenAndServe(port, s))
}
