// 命令处理相关
package main

import (
	"encoding/json"
	"strconv"
)

var boolDispatch = map[string]func(int) bool{
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
}

var listDispatch = map[string]func() []streaming{
	"listlive":   listLive,
	"listrecord": listRecord,
	"listdanmu":  listDanmu,
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
	if d, ok := boolDispatch[cmd]; ok {
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
