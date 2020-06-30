// 命令处理相关
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const handleErrMsg = "请输入正确的命令，输入 help 查看全部命令的解释"

var boolDispatch = map[string]func() bool{
	//"startweb":   startWeb,
	"stopweb": stopWeb,
	//"startcoolq": startCoolq,
}

var uidBoolDispatch = map[string]func(int) bool{
	"addnotify":   addNotify,
	"delnotify":   delNotify,
	"addrecord":   addRecord,
	"delrecord":   delRecord,
	"adddanmu":    addDanmu,
	"deldanmu":    delDanmu,
	"stoprecord":  stopRec,
	"startdanmu":  startDanmu,
	"stopdanmu":   stopDanmu,
	"startrecdan": startRecDan,
	"stoprecdan":  stopRecDan,
	"delqq":       delQQNotify,
	"delqqgroup":  delQQGroup,
}

var listDispatch = map[string]func() []streaming{
	"listlive":   listLive,
	"listrecord": listRecord,
	"listdanmu":  listDanmu,
}

var qqDispatch = map[string]func(int, int) bool{
	"addqq":      addQQNotify,
	"addqqgroup": addQQGroup,
}

// 将bool类型转换为字符串
var boolStr = strconv.FormatBool

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
		lPrintln("错误的命令：" + cmd)
		printErr()
		return ""
	}
}

// 处理 "命令 UID"
func handleCmdUID(cmd string, uid int) string {
	if d, ok := uidBoolDispatch[cmd]; ok {
		return boolStr(d(uid))
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
		lPrintln("错误的命令："+cmd, uid)
		printErr()
		return ""
	}
}

func handleCmdQQ(cmd string, uid int, qq int) string {
	if d, ok := qqDispatch[cmd]; ok {
		return boolStr(d(uid, qq))
	}

	lPrintln("错误的命令："+cmd, uid, qq)
	printErr()
	return ""
}

// 打印错误命令信息
func printErr() {
	lPrintln(handleErrMsg)
}

// 处理所有命令
func handleAllCmd(text string) string {
	cmd := strings.Fields(text)
	switch len(cmd) {
	case 1:
		switch cmd[0] {
		case "help":
			fmt.Println(helpMsg)
			return helpMsg
		default:
			return handleCmd(cmd[0])
		}
	case 2:
		uid, err := strconv.ParseUint(cmd[1], 10, 64)
		if err != nil {
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
			return handleCmdQQ(cmd[0], int(uid), int(qq))
		}
	default:
		printErr()
	}

	return ""
}
