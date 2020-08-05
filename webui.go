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

func startWebUI() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in startWebUI(), the error is:", err)
			lPrintErr("web UI启动出现错误，请重启本程序")
		}
	}()

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

func stopWebUI() {
	lPrintln("停止web API服务器")
	ctx, cancel := context.WithCancel(mainCtx)
	defer cancel()
	if err := uiSrv.Shutdown(ctx); err != nil {
		lPrintErr("web UI服务器关闭错误：", err)
		lPrintWarn("强行关闭web UI服务器")
		cancel()
	}
}
