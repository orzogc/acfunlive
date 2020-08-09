// 酷Q相关
package main

import (
	"fmt"
	"strings"
	"time"

	qqbotapi "github.com/catsworld/qq-bot-api"
)

// 是否连接酷Q
var isCoolq *bool

var bot *qqbotapi.BotAPI = nil

// 酷Q相关设置数据
type coolqData struct {
	CqhttpWSAddr string // CQHTTP的WebSocket地址
	AdminQQ      int64  // 管理者的QQ，通过这个QQ发送命令
	AccessToken  string // CQHTTP的access_token
	Secret       string // CQHTTP的secret
}

// 建立对酷Q的连接
func startCoolq() bool {
	if *isCoolq {
		lPrintWarn("已经建立过对酷Q的连接")
	} else {
		*isCoolq = true
		lPrintln("尝试通过 " + config.Coolq.CqhttpWSAddr + " 连接酷Q")
		initCoolq()
	}
	return true
}

// 设置QQ开播提醒
func addQQNotify(uid int, qq int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Notify.NotifyOn {
			s.SendQQ = int64(qq)
			sets(s)
			lPrintln("成功设置将" + s.Name + "的开播提醒发送到QQ" + itoa(qq))
		} else {
			lPrintWarn("设置QQ的开播提醒需要先订阅" + s.Name + "的开播提醒，请运行addnotify " + s.itoa())
			streamers.Unlock()
			return false
		}
	} else {
		lPrintWarn("设置QQ的开播提醒需要先订阅uid为" + itoa(uid) + "的主播的开播提醒，请运行addnotify " + itoa(uid))
		streamers.Unlock()
		return false
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 取消设置QQ开播提醒
func delQQNotify(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		s.SendQQ = 0
		sets(s)
		lPrintln("成功取消设置" + s.Name + "的QQ开播提醒")
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 设置QQ群开播提醒
func addQQGroup(uid int, qqGroup int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Notify.NotifyOn {
			s.SendQQGroup = int64(qqGroup)
			sets(s)
			lPrintln("成功设置将" + s.Name + "的开播提醒发送到QQ群" + itoa(qqGroup))
		} else {
			lPrintWarn("设置QQ群的开播提醒需要先订阅" + s.Name + "的开播提醒，请运行addnotify " + s.itoa())
			streamers.Unlock()
			return false
		}
	} else {
		lPrintWarn("设置QQ群的开播提醒需要先订阅uid为" + itoa(uid) + "的主播的开播提醒，请运行addnotify " + itoa(uid))
		streamers.Unlock()
		return false
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 取消设置QQ群开播提醒
func delQQGroup(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		s.SendQQGroup = 0
		sets(s)
		lPrintln("成功取消设置" + s.Name + "的QQ群开播提醒")
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ群开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 初始化对酷Q的连接
func initCoolq() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in initCoolq(), the error is:", err)
			lPrintErr("连接酷Q出现问题，请确定已启动酷Q")
			bot = nil
			*isCoolq = false
		}
	}()

	if bot != nil {
		lPrintWarn("已经建立过对酷Q的连接")
		return
	}

	newBot, err := qqbotapi.NewBotAPI(config.Coolq.AccessToken, config.Coolq.CqhttpWSAddr, config.Coolq.Secret)
	checkErr(err)
	bot = newBot
	lPrintln("成功通过 " + config.Coolq.CqhttpWSAddr + " 连接酷Q")

	go getCoolqMsg()
}

// 发送消息到指定的QQ
func coolqSendQQ(qq int64, text string) {
	s := bot.NewMessage(qq, "private").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息到指定的QQ群
func coolqSendQQGroup(qqGroup int64, text string) {
	s := bot.NewMessage(qqGroup, "group").At("all").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息
func (s streamer) sendCoolq(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendCoolq(), the error is:", err)
			lPrintErr("发送" + s.longID() + "的消息（" + text + "）到指定的QQ（" + itoa(int(s.SendQQ)) + "）/QQ群（" + itoa(int(s.SendQQGroup)) + "）时发生错误，取消发送")
		}
	}()

	if *isCoolq && bot != nil {
		if s.SendQQ != 0 {
			coolqSendQQ(s.SendQQ, text)
		}
		if s.SendQQGroup != 0 {
			coolqSendQQGroup(s.SendQQGroup, text)
		}
	}
}

// 获取发送给酷Q机器人的消息
func getCoolqMsg() {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in getCoolqMsg(), the error is:", err)
			lPrintErr("获取发送给酷Q机器人的消息时出现错误，尝试重新获取")
			time.Sleep(2 * time.Second)
			go getCoolqMsg()
		}
	}()

	u := qqbotapi.NewUpdate(0)
	updates, err := bot.GetUpdatesChan(u)
	checkErr(err)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		handleCoolqMsg(update.Message)
	}
}

// 处理并回复QQ消息
func handleCoolqMsg(msg *qqbotapi.Message) {
	if msg.From.ID == config.Coolq.AdminQQ {
		if msg.Chat.Type == "private" {
			lPrintln(fmt.Sprintf("处理来自QQ%d的命令：%s", msg.From.ID, msg.Text))
			if s := handleAllCmd(msg.Text); s != "" {
				bot.SendMessage(msg.Chat.ID, msg.Chat.Type, s)
			} else {
				bot.SendMessage(msg.Chat.ID, msg.Chat.Type, handleErrMsg)
			}
		} else {
			if bot.IsMessageToMe(*msg) {
				i := strings.Index(msg.Text, "]")
				text := msg.Text[i+1:]
				lPrintln(fmt.Sprintf("处理来自QQ群%d里QQ%d的命令：%s", msg.Chat.ID, msg.From.ID, text))
				if s := handleAllCmd(text); s != "" {
					bot.SendMessage(msg.Chat.ID, msg.Chat.Type, s)
				} else {
					bot.SendMessage(msg.Chat.ID, msg.Chat.Type, handleErrMsg)
				}
			}
		}
	}
}
