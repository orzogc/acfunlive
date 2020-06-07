// 设置相关
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

const configFile = "config.json"

var configFileLocation string

// 主播的设置数据
type streamer struct {
	UID      uint
	ID       string
	Notify   bool
	Record   bool
	Restream bool
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
		log.Fatalln(configFile + "不能是目录")
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
			log.Println("设置文件" + configFile + "的内容不符合json格式，请检查其内容")
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
			fmt.Println("删除" + s.ID + "的设置数据")
		}
	}
}

// 设置将下载的直播流同时推向本地端口
func addRestream(uid uint) {
	isExist := false
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			isExist = true
			if s.Record {
				if s.Restream {
					fmt.Println("已经设置过将" + s.ID + "的直播流同时推向本地端口")
				} else {
					streamers[i].Restream = true
					fmt.Println("成功设置将" + s.ID + "的直播流同时推向本地端口")
				}
			} else {
				fmt.Println("请先设置自动下载" + s.ID + "的直播")
			}
		}
	}
	sMutex.Unlock()

	if !isExist {
		fmt.Println("请先设置自动下载该主播的直播")
	}

	saveConfig()
}

// 取消将下载的直播流同时推向本地端口
func delRestream(uid uint) {
	sMutex.Lock()
	for i, s := range streamers {
		if s.UID == uid {
			streamers[i].Restream = false
			fmt.Println("成功取消将" + s.ID + "的直播流同时推向本地端口")
		}
	}
	sMutex.Unlock()

	saveConfig()
}

// 循环判断设置文件是否被修改，是的话重新设置
func cycleConfig(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovering from panic in cycleConfig(), the error is:", err)
			log.Println("循环读取设置文件" + configFile + "时出错，请重新运行本程序")
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
				logPrintln("设置文件" + configFile + "被修改，重新读取设置")
				modTime = info.ModTime()
				loadConfig()

				for _, s := range streamers {
					for i, olds := range oldStreamers {
						if s.UID == olds.UID {
							if s != olds {
								// olds的设置被改变
								logPrintln(s.longID() + "的设置被修改，重新设置")
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
								logPrintln("新增" + s.longID() + "的设置")
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
								logPrintln(olds.longID() + "的设置被删除")
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
