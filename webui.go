package main

import (
	"net/http"
	"path/filepath"
	"time"
)

// web UI html和js文件所在位置
const webUIDir = "webui"

var uiSrv *http.Server

// web UI服务器
func webUI() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in webUI(), the error is:", err)
			lPrintErr("web UI服务器发生错误，尝试重启web UI服务器")
			time.Sleep(2 * time.Second)
			go webUI()
		}
	}()

	uiSrv = &http.Server{
		Addr:         ":" + itoa(config.WebPort+10),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      http.FileServer(http.Dir(filepath.Join(exeDir, webUIDir))),
	}
	err := uiSrv.ListenAndServe()
	if err != http.ErrServerClosed {
		lPrintln(err)
		panic(err)
	}
}
