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

// 设置修改标记，map[uint]bool
var modify = sync.Map{}

// 主播的设置数据
type streamer struct {
	// uid
	UID uint
	// 主播名字
	ID string
	// 是否开播提醒
	Notify bool
	// 是否自动下载直播
	Record bool
}

// 存放主播的设置数据
var streamers struct {
	mu sync.Mutex
	// 现在的主播的设置数据
	current []streamer
	// 旧的主播的设置数据
	old []streamer
}

// 查看设置文件是否存在
func isConfigFileExist() bool {
	info, err := os.Stat(configFileLocation)
	if os.IsNotExist(err) {
		return false
	}
	if info.IsDir() {
		timePrintln(configFile + "不能是目录")
		os.Exit(1)
	}
	return true
}

// 读取设置文件
func loadConfig() {
	if isConfigFileExist() {
		data, err := ioutil.ReadFile(configFileLocation)
		checkErr(err)

		if json.Valid(data) {
			err = json.Unmarshal(data, &streamers.current)
			checkErr(err)
		} else {
			timePrintln("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
		}
	}
}

// 保存设置文件
func saveConfig() {
	streamers.mu.Lock()
	data, err := json.MarshalIndent(streamers.current, "", "    ")
	checkErr(err)

	err = ioutil.WriteFile(configFileLocation, data, 0644)
	checkErr(err)
	streamers.mu.Unlock()
}

// 设置里删除指定uid的主播
func deleteStreamer(uid uint) {
	for i, s := range streamers.current {
		if s.UID == uid {
			streamers.current = append(streamers.current[:i], streamers.current[i+1:]...)
			logger.Println("删除" + s.ID + "的设置数据")
		}
	}
}

// 循环判断设置文件是否被修改，是的话重新设置
func cycleConfig(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			timePrintln("Recovering from panic in cycleConfig(), the error is:", err)
			timePrintln("循环读取设置文件" + configFile + "时出错，尝试重启循环读取设置文件")
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
				timePrintln("设置文件" + configFile + "被修改，重新读取设置")
				modTime = info.ModTime()
				loadConfig()

				for _, s := range streamers.current {
					for i, olds := range streamers.old {
						if s.UID == olds.UID {
							if s != olds {
								// olds的设置被修改
								timePrintln(s.longID() + "的设置被修改，重新设置")
								restart := controlMsg{s: s, c: startCycle}
								ch, _ := chMap.Load(s.UID)
								modify.Store(s.UID, true)
								ch.(chan controlMsg) <- restart
							}
							break
						} else {
							if i == len(streamers.old)-1 {
								// s为新增的主播
								timePrintln("新增" + s.longID() + "的设置")
								start := controlMsg{s: s, c: startCycle}
								ch, _ := chMap.Load(0)
								modify.Store(s.UID, true)
								ch.(chan controlMsg) <- start
							}
						}
					}
				}

				for _, olds := range streamers.old {
					for i, s := range streamers.current {
						if s.UID == olds.UID {
							break
						} else {
							if i == len(streamers.current)-1 {
								// olds为被删除的主播
								timePrintln(olds.longID() + "的设置被删除")
								stop := controlMsg{s: olds, c: stopCycle}
								ch, _ := chMap.Load(olds.UID)
								ch.(chan controlMsg) <- stop
							}
						}
					}
				}
			}
			streamers.old = append([]streamer(nil), streamers.current...)
			streamers.mu.Unlock()

			// 每分钟循环一次
			time.Sleep(time.Minute)
		}
	}
}
