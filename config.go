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

// 主播的设置数据
type streamer struct {
	UID    uint
	ID     string
	Notify bool
	Record bool
}

// streamers的锁
var sMutex sync.Mutex
var streamers []streamer
var oldStreamers []streamer

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
			err = json.Unmarshal(data, &streamers)
			checkErr(err)
		} else {
			timePrintln("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
		}
	}
}

// 保存设置文件
func saveConfig() {
	sMutex.Lock()
	data, err := json.MarshalIndent(streamers, "", "    ")
	checkErr(err)

	err = ioutil.WriteFile(configFileLocation, data, 0644)
	checkErr(err)
	sMutex.Unlock()
}

// 设置里删除指定uid的主播
func deleteStreamer(uid uint) {
	for i, s := range streamers {
		if s.UID == uid {
			streamers = append(streamers[:i], streamers[i+1:]...)
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

			sMutex.Lock()
			if info.ModTime().After(modTime) {
				timePrintln("设置文件" + configFile + "被修改，重新读取设置")
				modTime = info.ModTime()
				loadConfig()

				for _, s := range streamers {
					for i, olds := range oldStreamers {
						if s.UID == olds.UID {
							if s != olds {
								// olds的设置被改变
								timePrintln(s.longID() + "的设置被修改，重新设置")
								restart := controlMsg{s: s, c: startCycle}
								chMutex.Lock()
								ch := chMap[s.UID]
								chMutex.Unlock()
								ch <- restart
							}
							break
						} else {
							if i == len(oldStreamers)-1 {
								// s为新增的主播
								timePrintln("新增" + s.longID() + "的设置")
								start := controlMsg{s: s, c: startCycle}
								chMutex.Lock()
								ch := chMap[0]
								chMutex.Unlock()
								ch <- start
							}
						}
					}
				}

				for _, olds := range oldStreamers {
					for i, s := range streamers {
						if s.UID == olds.UID {
							break
						} else {
							if i == len(streamers)-1 {
								// olds为被删除的主播
								timePrintln(olds.longID() + "的设置被删除")
								stop := controlMsg{s: olds, c: stopCycle}
								chMutex.Lock()
								ch := chMap[olds.UID]
								chMutex.Unlock()
								ch <- stop
							}
						}
					}
				}
			}
			oldStreamers = append([]streamer(nil), streamers...)
			sMutex.Unlock()

			// 每分钟循环一次
			time.Sleep(time.Minute)
		}
	}
}
