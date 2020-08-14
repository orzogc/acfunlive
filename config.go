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

	"github.com/fsnotify/fsnotify"
)

const (
	liveFile   = "live.json"   // 主播设置文件名字
	configFile = "config.json" // 设置文件名字
)

var (
	liveFileLocation   string // 主播设置文件位置
	configFileLocation string // 设置文件位置
)

// 主播的设置数据
type streamer struct {
	UID         int    // 主播uid
	Name        string // 主播名字
	Notify      notify // 开播提醒相关
	Record      bool   // 是否自动下载直播视频
	Danmu       bool   // 是否自动下载直播弹幕
	Bitrate     int    // 下载直播视频的最高码率
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
	Source  string    // 直播源，有hls和flv两种
	Output  string    // 直播下载视频格式的后缀名
	WebPort int       // web API的本地端口
	Mirai   miraiData // Mirai相关设置
	Coolq   coolqData // 酷Q相关设置
}

// 默认设置
var config = configData{
	Source:  "flv",
	Output:  "mp4",
	WebPort: 51880,
	Mirai: miraiData{
		AdminQQ:       0,
		BotQQ:         0,
		BotQQPassword: "",
	},
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
	checkErr(err)
	if info.IsDir() {
		lPrintErr(fileLocation + " 不能是目录")
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
			lPrintErr("设置文件" + liveFile + "的内容不符合json格式，请检查其内容")
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
			lPrintErr("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
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
func deleteStreamer(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		delete(streamers.crt, uid)
		lPrintln("删除" + s.Name + "的设置数据")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 监控config.json是否被修改，是的话重新设置
func cycleConfig(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in cycleConfig(), the error is:", err)
			lPrintErr("监控设置文件" + liveFile + "时出错，请重启本程序")
		}
	}()

	lPrintln("开始监控设置文件" + liveFile)

	watcher, err := fsnotify.NewWatcher()
	checkErr(err)
	defer watcher.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			case event, ok := <-watcher.Events:
				if !ok {
					wg.Done()
					return
				}
				// 很多时候保存文件会分为数段写入，避免读取未完成写入的设置文件
				time.Sleep(100 * time.Millisecond)
			Outer:
				for {
					select {
					case event, ok = <-watcher.Events:
						if !ok {
							wg.Done()
							return
						}
					default:
						if event.Op&fsnotify.Write == fsnotify.Write {
							lPrintln("设置文件" + liveFile + "被修改，重新读取设置")
							loadNewConfig()
						}
						break Outer
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					wg.Done()
					return
				}
				lPrintErr("监控设置文件"+liveFile+"时出现错误：", err)
			}
		}
	}()

	err = watcher.Add(liveFileLocation)
	checkErr(err)
	wg.Wait()
	lPrintln("停止监控设置文件" + liveFile)
}

// 读取修改后的config.json
func loadNewConfig() {
	streamers.Lock()

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
			msgMap.Unlock()
			mainCh <- start
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

	streamers.Unlock()
}
