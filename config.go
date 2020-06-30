// 设置相关
package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// 主播设置文件名字
const liveFile = "live.json"

// 设置文件名字
const configFile = "config.json"

// 主播设置文件位置
var liveFileLocation string

// 设置文件位置
var configFileLocation string

// 主播的设置数据
type streamer struct {
	UID         int    // 主播uid
	Name        string // 主播名字
	Notify      bool   // 是否开播提醒
	Record      bool   // 是否自动下载直播视频
	Danmu       bool   // 是否自动下载直播弹幕
	SendQQ      int64  // 给这个QQ号发送消息
	SendQQGroup int64  // 给这个QQ群发送消息
}

// 存放主播的设置数据
var streamers struct {
	sync.Mutex                  // crt的锁
	crt        map[int]streamer // 现在的主播的设置数据
	old        map[int]streamer // 旧的主播的设置数据
}

// 设置数据
type configData struct {
	Source  string    // 直播源，分为hls和flv两种
	Output  string    // 直播下载视频格式的后缀名
	WebPort int       // web服务的本地端口
	Coolq   coolqData // 酷Q相关设置
}

// 默认设置
var config = configData{
	Source:  "hls",
	Output:  "mp4",
	WebPort: 51880,
	Coolq: coolqData{
		CqhttpWSAddr: "ws://localhost:6700",
		AdminQQ:      0,
		AccessToken:  "",
		Secret:       "",
	},
}

// 将s放进streamers里
func sets(s streamer) {
	streamers.crt[s.UID] = s
}

// 将map[int]streamer转换为[]streamer，按照uid大小排序
func getStreamers() []streamer {
	var ss []streamer
	streamers.Lock()
	for _, s := range streamers.crt {
		ss = append(ss, s)
	}
	streamers.Unlock()
	// 按uid大小排序
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].UID < ss[j].UID
	})
	return ss
}

// 查看设置文件是否存在
func isConfigFileExist(filename string) bool {
	fileLocation := filepath.Join(exeDir, filename)
	info, err := os.Stat(fileLocation)
	if os.IsNotExist(err) {
		return false
	}
	if info.IsDir() {
		lPrintln(filename + "不能是目录")
		os.Exit(1)
	}
	return true
}

// 读取live.json
func loadLiveConfig() {
	if isConfigFileExist(liveFile) {
		data, err := ioutil.ReadFile(liveFileLocation)
		checkErr(err)

		if json.Valid(data) {
			var ss []streamer
			err = json.Unmarshal(data, &ss)
			checkErr(err)
			news := make(map[int]streamer)
			for _, s := range ss {
				news[s.UID] = s
			}
			streamers.crt = news
		} else {
			lPrintln("设置文件" + liveFile + "的内容不符合json格式，请检查其内容")
		}
	}
}

// 读取config.json
func loadConfig() {
	if isConfigFileExist(configFile) {
		data, err := ioutil.ReadFile(configFileLocation)
		checkErr(err)

		if json.Valid(data) {
			err = json.Unmarshal(data, &config)
			checkErr(err)
		} else {
			lPrintln("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
		}
	}
}

// 保存live.json
func saveLiveConfig() {
	data, err := json.MarshalIndent(getStreamers(), "", "    ")
	checkErr(err)

	err = ioutil.WriteFile(liveFileLocation, data, 0644)
	checkErr(err)
}

// 设置里删除指定uid的主播
func deleteStreamer(uid int) {
	if s, ok := streamers.crt[uid]; ok {
		delete(streamers.crt, uid)
		lPrintln("删除" + s.Name + "的设置数据")
	}
}

// 循环判断设置文件是否被修改，是的话重新设置
func cycleConfig(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in cycleConfig(), the error is:", err)
			lPrintln("循环读取设置文件" + liveFile + "时出错，请重启本程序")
		}
	}()

	modTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			info, err := os.Stat(liveFileLocation)
			checkErr(err)

			streamers.Lock()
			if info.ModTime().After(modTime) {
				lPrintln("设置文件" + liveFile + "被修改，重新读取设置")
				modTime = info.ModTime()
				loadLiveConfig()

				for uid, s := range streamers.crt {
					if olds, ok := streamers.old[uid]; ok {
						if s != olds {
							// olds的设置被修改
							lPrintln(s.longID() + "的设置被修改，重新设置")
							restart := controlMsg{s: s, c: startCycle}
							msgMap.Lock()
							m := msgMap.msg[s.UID]
							m.modify = true
							m.ch <- restart
							msgMap.Unlock()
						}
					} else {
						// s为新增的主播
						lPrintln("新增" + s.longID() + "的设置")
						start := controlMsg{s: s, c: startCycle}
						msgMap.Lock()
						if m, ok := msgMap.msg[s.UID]; ok {
							m.modify = true
						} else {
							msgMap.msg[s.UID] = &sMsg{modify: true}
						}
						msgMap.msg[0].ch <- start
						msgMap.Unlock()
					}
				}

				for uid, olds := range streamers.old {
					if _, ok := streamers.crt[uid]; !ok {
						// olds为被删除的主播
						lPrintln(olds.longID() + "的设置被删除")
						stop := controlMsg{s: olds, c: stopCycle}
						msgMap.Lock()
						msgMap.msg[olds.UID].ch <- stop
						msgMap.Unlock()
					}
				}

				oldstreamers := make(map[int]streamer)
				for uid, s := range streamers.crt {
					oldstreamers[uid] = s
				}
				streamers.old = oldstreamers
			}
			streamers.Unlock()

			// 每半分钟循环一次
			time.Sleep(30 * time.Second)
		}
	}
}
