// 设置相关
package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const configFile = "config.json"

var configFileLocation string

// 设置修改标记，map[int]bool
//var modify = sync.Map{}

// 主播的设置数据
type streamer struct {
	// 主播uid
	UID int
	// 主播名字
	Name string
	// 是否开播提醒
	Notify bool
	// 是否自动下载直播
	Record bool
}

// 存放主播的设置数据
var streamers struct {
	// crt的锁
	mu sync.Mutex
	// 现在的主播的设置数据
	crt map[int]streamer
	// 旧的主播的设置数据
	old map[int]streamer
}

// 获取对应uid的streamer
/*
func gets(uid int) streamer {
	return streamers.crt[uid]
}
*/

// 将s放进streamers里
func sets(s streamer) {
	streamers.crt[s.UID] = s
}

/*
func (s streamer) sets() {
	streamers.crt[s.UID] = s
}
*/

// 查看设置文件是否存在
func isConfigFileExist() bool {
	info, err := os.Stat(configFileLocation)
	if os.IsNotExist(err) {
		return false
	}
	if info.IsDir() {
		lPrintln(configFile + "不能是目录")
		os.Exit(1)
	}
	return true
}

// 读取设置文件
func loadConfig() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in loadConfig(), the error is:", err)
			lPrintln("读取设置文件" + configFile + "时出错，请重启本程序")
		}
	}()

	if isConfigFileExist() {
		data, err := ioutil.ReadFile(configFileLocation)
		checkErr(err)

		if json.Valid(data) {
			var ss []streamer
			err = json.Unmarshal(data, &ss)
			checkErr(err)
			for _, s := range ss {
				sets(s)
			}
		} else {
			lPrintln("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
		}
	}
}

// 保存设置文件
func saveConfig() {
	streamers.mu.Lock()
	defer streamers.mu.Unlock()

	var ss []streamer
	for _, s := range streamers.crt {
		ss = append(ss, s)
	}
	data, err := json.MarshalIndent(ss, "", "    ")
	checkErr(err)

	err = ioutil.WriteFile(configFileLocation, data, 0644)
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
			lPrintln("循环读取设置文件" + configFile + "时出错，尝试重启循环读取设置文件")
			time.Sleep(2 * time.Second)
			go cycleConfig(ctx)
		}
	}()

	modTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			info, err := os.Stat(configFileLocation)
			checkErr(err)

			streamers.mu.Lock()
			if info.ModTime().After(modTime) {
				lPrintln("设置文件" + configFile + "被修改，重新读取设置")
				modTime = info.ModTime()
				loadConfig()

				for uid, s := range streamers.crt {
					if olds, ok := streamers.old[uid]; ok {
						if s != olds {
							// olds的设置被修改
							lPrintln(s.longID() + "的设置被修改，重新设置")
							restart := controlMsg{s: s, c: startCycle}
							msgMap.mu.Lock()
							m := msgMap.msg[s.UID]
							m.modify = true
							msgMap.msg[s.UID] = m
							m.ch <- restart
							msgMap.mu.Unlock()
						}
					} else {
						// s为新增的主播
						lPrintln("新增" + s.longID() + "的设置")
						start := controlMsg{s: s, c: startCycle}
						msgMap.mu.Lock()
						if m, ok := msgMap.msg[s.UID]; ok {
							m.modify = true
							msgMap.msg[s.UID] = m
						} else {
							msgMap.msg[s.UID] = sMsg{modify: true}
						}
						msgMap.msg[0].ch <- start
						msgMap.mu.Unlock()
					}
				}

				for uid, olds := range streamers.old {
					if _, ok := streamers.crt[uid]; !ok {
						// olds为被删除的主播
						lPrintln(olds.longID() + "的设置被删除")
						stop := controlMsg{s: olds, c: stopCycle}
						msgMap.mu.Lock()
						msgMap.msg[olds.UID].ch <- stop
						msgMap.mu.Unlock()
					}
				}
			}
			for uid, s := range streamers.crt {
				streamers.old[uid] = s
			}
			streamers.mu.Unlock()

			// 每半分钟循环一次
			time.Sleep(30 * time.Second)
		}
	}
}
