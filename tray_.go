// +build !tray

// 系统托盘
package main

// 初始化tray
func initTray() {
	*isNoGUI = true
}

// 运行systray
func runTray() {}

// 退出systray
func quitTray() {}

// 启动systray
func trayOnReady() {}

// 退出systray
func trayOnExit() {}
