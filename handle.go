// 命令处理相关
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// 错误命令信息
const handleErrMsg = "请输入正确的命令，输入 help 查看全部命令的解释"

// 帮助信息
const helpMsg = `listlive：列出正在直播的主播
listrecord：列出正在下载的直播视频
listdanmu：列出正在下载的直播弹幕
startwebapi：启动 web API 服务器
stopwebapi：停止 web API 服务器
startwebui：启动 web UI 服务器，需要 web API 服务器运行，如果 web API 服务器没启动会启动 web API 服务器
stopwebui：停止 web UI 服务器
startmirai：利用 Mirai 发送直播通知到指定 QQ 或 QQ 群
addnotifyon uid：订阅指定主播的开播提醒，uid 在主播的网页版个人主页查看
delnotifyon uid：取消订阅指定主播的开播提醒
addnotifyoff uid：订阅指定主播的下播提醒
delnotifyoff uid：取消订阅指定主播的下播提醒
addnotifyrecord uid：通知指定主播的直播视频下载
delnotifyrecord uid：取消通知指定主播的直播视频下载
addnotifydanmu uid：通知指定主播的直播弹幕下载
delnotifydanmu uid：取消通知指定主播的直播弹幕下载
addrecord uid：自动下载指定主播的直播视频
delrecord uid：取消自动下载指定主播的直播视频
adddanmu uid：自动下载指定主播的直播弹幕
deldanmu uid：取消自动下载指定主播的直播弹幕
addkeeponline uid：指定主播直播时在其直播间挂机
delkeeponline uid：取消在指定主播直播时在其直播间挂机
delconfig uid：删除指定主播的所有设置
getdlurl uid：查看指定主播是否在直播，如在直播输出其直播源地址
addqq uid QQ 号：设置将指定主播的开播提醒发送到指定 QQ 号，需要 QQ 机器人已经添加该 QQ 为好友
delqq uid QQ 号：取消设置将指定主播的开播提醒发送到指定 QQ 号
cancelqq uid：取消设置将指定主播的开播提醒发送到任何 QQ
addqqgroup uid QQ 群号：设置将指定主播的开播提醒发送到指定 QQ 群号，需要 QQ 机器人已经加入该 QQ 群，最好是管理员，会@全体成员
delqqgroup uid QQ 群号：取消设置将指定主播的开播提醒发送到指定 QQ 群号
cancelqqgroup uid：取消设置将指定主播的开播提醒发送到任何 QQ 群
startrecord uid：临时下载指定主播的直播视频，如果没有设置自动下载该主播的直播视频，这次为一次性的下载
stoprecord uid：正在下载指定主播的直播视频时取消下载
startdanmu uid：临时下载指定主播的直播弹幕，如果没有设置自动下载该主播的直播弹幕，这次为一次性的下载
stopdanmu uid：正在下载指定主播的直播弹幕时取消下载
startrecdan uid：临时下载指定主播的直播视频和弹幕），如果没有设置自动下载该主播的直播视频和弹幕，这次为一次性的下载
stoprecdan uid：正在下载指定主播的直播视频和弹幕时取消下载
quit：退出本程序，退出需要等待半分钟左右
help：输出本帮助信息`

var boolDispatch = map[string]func() bool{
	//"startweb":   startWebAPI,
	"stopwebapi": stopWebAPI,
	//"startwebui": startWebUI,
	"stopwebui": stopWebUI,
	//"startmirai": startMirai,
}

var uidBoolDispatch = map[string]func(int) bool{
	"delconfig":     deleteStreamer,
	"stoprecord":    stopRec,
	"startdanmu":    startDanmu,
	"stopdanmu":     stopDanmu,
	"startrecdan":   startRecDan,
	"stoprecdan":    stopRecDan,
	"cancelqq":      cancelQQNotify,
	"cancelqqgroup": cancelQQGroup,
}

var listDispatch = map[string]func() []streaming{
	"listlive":   listLive,
	"listrecord": listRecord,
	"listdanmu":  listDanmu,
}

var qqDispatch = map[string]func(int, int64) bool{
	"addqq":      addQQNotify,
	"delqq":      delQQNotify,
	"addqqgroup": addQQGroup,
	"delqqgroup": delQQGroup,
}

// 处理单个命令
func handleCmd(cmd string) string {
	if d, ok := listDispatch[cmd]; ok {
		data, err := json.MarshalIndent(d(), "", "    ")
		checkErr(err)
		return string(data)
	}

	if d, ok := boolDispatch[cmd]; ok {
		return boolStr(d())
	}

	switch cmd {
	case "liststreamer":
		data, err := json.MarshalIndent(getStreamers(), "", "    ")
		checkErr(err)
		return string(data)
	case "quit":
		quitRun()
		return "true"
	default:
		lPrintErr("错误的命令：" + cmd)
		printErr()
		return ""
	}
}

// 处理 "命令 UID"
func handleCmdUID(cmd string, uid int) string {
	if d, ok := uidBoolDispatch[cmd]; ok {
		return boolStr(d(uid))
	}

	// 保持兼容
	if cmd == "addnotify" || cmd == "delnotify" {
		cmd = cmd + "on"
	}
	s, ok := getStreamer(uid)
	if !ok {
		name := getName(uid)
		if name == "" {
			lPrintErrf("不存在 uid 为%d的用户", uid)
			return ""
		}
		s = streamer{UID: uid, Name: name}
	}
	if strings.HasPrefix(cmd, "add") {
		if s.setBoolConfig(cmd[3:], true) {
			return boolStr(true)
		}
		lPrintErr("错误的命令："+cmd, uid)
		printErr()
		return ""
	}
	if strings.HasPrefix(cmd, "del") {
		if s.setBoolConfig(cmd[3:], false) {
			return boolStr(true)
		}
		lPrintErr("错误的命令："+cmd, uid)
		printErr()
		return ""
	}

	switch cmd {
	case "startrecord":
		return boolStr(startRec(uid, false))
	case "getdlurl":
		hlsURL, flvURL := printStreamURL(uid)
		data, err := json.MarshalIndent([]string{hlsURL, flvURL}, "", "    ")
		checkErr(err)
		return string(data)
	default:
		lPrintErr("错误的命令："+cmd, uid)
		printErr()
		return ""
	}
}

// 处理 QQ 命令
func handleCmdQQ(cmd string, uid int, qq int64) string {
	if d, ok := qqDispatch[cmd]; ok {
		return boolStr(d(uid, qq))
	}

	lPrintErr("错误的命令："+cmd, uid, qq)
	printErr()
	return ""
}

// 打印错误命令信息
func printErr() {
	lPrintWarn(handleErrMsg)
}

// 处理所有命令
func handleAllCmd(text string) string {
	cmd := strings.Fields(text)
	switch len(cmd) {
	case 1:
		switch cmd[0] {
		case "help":
			if *isNoGUI {
				fmt.Println(helpMsg)
			}
			return helpMsg
		default:
			return handleCmd(cmd[0])
		}
	case 2:
		if uid, err := strconv.ParseUint(cmd[1], 10, 64); err != nil {
			printErr()
		} else {
			return handleCmdUID(cmd[0], int(uid))
		}
	case 3:
		uid, err1 := strconv.ParseUint(cmd[1], 10, 64)
		qq, err2 := strconv.ParseUint(cmd[2], 10, 64)
		if err1 != nil || err2 != nil {
			printErr()
		} else {
			return handleCmdQQ(cmd[0], int(uid), int64(qq))
		}
	default:
		printErr()
	}

	return ""
}
