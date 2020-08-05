package main

import (
	"io/ioutil"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

func trayOnReady() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in trayOnReady(), the error is:", err)
			lPrintErr("systray发生错误，请重启本程序")
		}
	}()

	icon, err := ioutil.ReadFile(logoFileLocation)
	checkErr(err)
	systray.SetTemplateIcon(icon, icon)
	systray.SetTitle("AcFun直播助手")
	systray.SetTooltip("AcFun直播助手")

	openWebUI := systray.AddMenuItem("打开web界面", "打开web界面")
	quit := systray.AddMenuItem("退出", "退出acfunlive")

	for {
		select {
		case <-openWebUI.ClickedCh:
			lPrintln("通过systray打开web界面")
			err := open.Run("http://localhost:" + itoa(config.WebPort+10))
			checkErr(err)
		case <-quit.ClickedCh:
			quitRun()
			return
		}
	}
}

func trayOnExit() {
	lPrintln("退出systray")
}
