//go:build tray

// 系统托盘
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

// 初始化 tray
func initTray() {
	*isNoGUI = false
}

// 运行 systray
func runTray() {
	go systray.Run(trayOnReady, trayOnExit)
}

// 退出 systray
func quitTray() {
	systray.Quit()
}

// 启动 systray
func trayOnReady() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in trayOnReady(), the error is:", err)
			lPrintErr("systray 发生错误，请重启本程序")
		}
	}()

	lPrintln("启动 systray")

	icon, err := ioutil.ReadFile(logoFileLocation)
	checkErr(err)
	systray.SetTemplateIcon(icon, icon)
	systray.SetTitle("AcFun 直播助手")
	systray.SetTooltip("AcFun 直播助手")

	openWebUI := systray.AddMenuItem("打开 web 界面", "打开 web 界面")
	quit := systray.AddMenuItem("退出", "退出 acfunlive")

	for {
		select {
		case <-openWebUI.ClickedCh:
			lPrintln("通过 systray 打开 web 界面")
			err := open.Run(fmt.Sprintf("http://localhost:%d", config.WebPort+10))
			checkErr(err)
		case <-quit.ClickedCh:
			quitRun()
			return
		}
	}
}

// 退出 systray
func trayOnExit() {
	lPrintln("退出 systray")
}
