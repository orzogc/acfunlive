//go:build !tray

// 系统托盘
package main

// 初始化 tray
func initTray() {
	*isNoGUI = true
}

// 运行 systray
func runTray() {}

// 退出 systray
func quitTray() {}

// 启动 systray
//func trayOnReady() {}

// 退出 systray
//func trayOnExit() {}
