// web ui服务器
package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// web UI html和js文件所在位置
const webUIDir = "webui"

var uiSrv *http.Server

// web UI服务器
func webUI(dir string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in webUI(), the error is:", err)
			lPrintErr("web UI服务器发生错误，尝试重启web UI服务器")
			time.Sleep(2 * time.Second)
			go webUI(dir)
		}
	}()

	lPrintln("启动web UI服务器，现在可以通过 " + address(config.WebPort+10) + " 来访问UI界面")

	uiSrv = &http.Server{
		Addr:         ":" + itoa(config.WebPort+10),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      http.FileServer(http.Dir(dir)),
	}
	err := uiSrv.ListenAndServe()
	if err != http.ErrServerClosed {
		lPrintln(err)
		panic(err)
	}
}

// 启动web UI
func startUI() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in startWebUI(), the error is:", err)
			lPrintErr("web UI启动出现错误，请重启本程序")
		}
	}()
	defer func() {
		*isWebUI = false
	}()

	if !*isWebAPI {
		lPrintln("启动web API服务器")
		startWebAPI()
	}

	dir := filepath.Join(exeDir, webUIDir)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		lPrintErr(dir + " 不存在，停止运行web UI服务器")
		return
	}
	checkErr(err)
	if !info.IsDir() {
		lPrintErr(dir + " 必须是目录，停止运行web UI服务器")
		return
	}
	htmlFile := filepath.Join(dir, "index.html")
	info, err = os.Stat(htmlFile)
	if os.IsNotExist(err) {
		lPrintErr(htmlFile + " 不存在，停止运行web UI服务器")
		return
	}
	checkErr(err)
	if info.IsDir() {
		lPrintErr(htmlFile + " 不能是目录，停止运行web UI服务器")
		return
	}

	webUI(dir)
}

// 启动web UI server
func startWebUI() bool {
	if *isWebUI {
		lPrintWarn("已经启动过web UI服务器")
	} else {
		*isWebUI = true
		go startUI()
	}
	return true
}

// 停止web UI server
func stopWebUI() bool {
	if *isWebUI {
		*isWebUI = false
		lPrintln("停止web UI服务器")
		ctx, cancel := context.WithCancel(mainCtx)
		defer cancel()
		if err := uiSrv.Shutdown(ctx); err != nil {
			lPrintErr("web UI服务器关闭错误：", err)
			lPrintWarn("强行关闭web UI服务器")
			cancel()
		}
	} else {
		lPrintWarn("没有启动web UI服务器")
	}
	return true
}
