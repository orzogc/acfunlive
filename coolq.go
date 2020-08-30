// 酷Q相关
package main

import (
	"strings"
	"time"

	qqbotapi "github.com/catsworld/qq-bot-api"
)

var (
	isCoolq  *bool // 是否连接酷Q
	coolqBot *qqbotapi.BotAPI
)

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

// 设置将主播的开播提醒发送到指定的QQ
func addQQNotify(uid int, qq int64) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Notify.NotifyOn {
			for _, q := range s.SendQQ {
				if q == qq {
					lPrintf("已经设置过将%s的开播提醒发送到QQ%d", s.longID(), qq)
					streamers.Unlock()
					return true
				}
			}
			s.SendQQ = append(s.SendQQ, qq)
			sets(s)
			lPrintf("成功设置将%s的开播提醒发送到QQ%d", s.longID(), qq)
		} else {
			lPrintWarn("设置QQ的开播提醒需要先订阅" + s.longID() + "的开播提醒，请运行addnotify " + s.itoa())
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

// 取消设置将主播的开播提醒发送到指定的QQ
func delQQNotify(uid int, qq int64) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		for i, q := range s.SendQQ {
			if q == qq {
				s.SendQQ = append(s.SendQQ[:i], s.SendQQ[i+1:]...)
				break
			}
		}
		sets(s)
		lPrintf("成功取消设置将%s的开播提醒发送到QQ%d", s.longID(), qq)
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 取消设置QQ开播提醒
func cancelQQNotify(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		s.SendQQ = []int64{}
		sets(s)
		lPrintln("成功取消设置" + s.longID() + "的QQ开播提醒")
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 设置将主播的开播提醒发送到指定的QQ群
func addQQGroup(uid int, qqGroup int64) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		if s.Notify.NotifyOn {
			for _, q := range s.SendQQGroup {
				if q == qqGroup {
					lPrintf("已经设置过将%s的开播提醒发送到QQ群%d", s.longID(), qqGroup)
					streamers.Unlock()
					return true
				}
			}
			s.SendQQGroup = append(s.SendQQGroup, qqGroup)
			sets(s)
			lPrintf("成功设置将%s的开播提醒发送到QQ群%d", s.longID(), qqGroup)
		} else {
			lPrintWarn("设置QQ群的开播提醒需要先订阅" + s.longID() + "的开播提醒，请运行addnotify " + s.itoa())
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

// 取消设置将主播的开播提醒发送到指定的QQ群
func delQQGroup(uid int, qqGroup int64) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		for i, q := range s.SendQQGroup {
			if q == qqGroup {
				s.SendQQGroup = append(s.SendQQGroup[:i], s.SendQQGroup[i+1:]...)
				break
			}
		}
		sets(s)
		lPrintf("成功取消设置将%s的开播提醒发送到QQ群%d", s.longID(), qqGroup)
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ群开播提醒")
	}
	streamers.Unlock()

	saveLiveConfig()
	return true
}

// 取消设置QQ群开播提醒
func cancelQQGroup(uid int) bool {
	streamers.Lock()
	if s, ok := streamers.crt[uid]; ok {
		s.SendQQGroup = []int64{}
		sets(s)
		lPrintln("成功取消设置" + s.longID() + "的QQ群开播提醒")
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
			coolqBot = nil
			*isCoolq = false
		}
	}()

	if coolqBot != nil {
		lPrintWarn("已经建立过对酷Q的连接")
		return
	}

	newBot, err := qqbotapi.NewBotAPI(config.Coolq.AccessToken, config.Coolq.CqhttpWSAddr, config.Coolq.Secret)
	checkErr(err)
	coolqBot = newBot
	lPrintln("成功通过 " + config.Coolq.CqhttpWSAddr + " 连接酷Q")

	go getCoolqMsg()
}

// 发送消息到指定的QQ
func coolqSendQQ(qq int64, text string) {
	s := coolqBot.NewMessage(qq, "private").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息到指定的QQ群
func coolqSendQQGroup(qqGroup int64, text string) {
	s := coolqBot.NewMessage(qqGroup, "group").At("all").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息
func (s streamer) sendCoolq(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendCoolq(), the error is:", err)
			lPrintErr("发送" + s.longID() + "的消息（" + text + "）到指定的QQ/QQ群时发生错误，取消发送")
		}
	}()

	if *isCoolq && coolqBot != nil {
		for _, qq := range s.SendQQ {
			if qq > 0 {
				coolqSendQQ(qq, text)
			}
		}
		for _, qqGroup := range s.SendQQGroup {
			if qqGroup > 0 {
				coolqSendQQGroup(qqGroup, text)
			}
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
	updates, err := coolqBot.GetUpdatesChan(u)
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
			lPrintf("处理来自QQ%d的命令：%s", msg.From.ID, msg.Text)
			if s := handleAllCmd(msg.Text); s != "" {
				coolqBot.SendMessage(msg.Chat.ID, msg.Chat.Type, s)
			} else {
				coolqBot.SendMessage(msg.Chat.ID, msg.Chat.Type, handleErrMsg)
			}
		} else {
			if coolqBot.IsMessageToMe(*msg) {
				i := strings.Index(msg.Text, "]")
				text := msg.Text[i+1:]
				lPrintf("处理来自QQ群%d里QQ%d的命令：%s", msg.Chat.ID, msg.From.ID, text)
				if s := handleAllCmd(text); s != "" {
					coolqBot.SendMessage(msg.Chat.ID, msg.Chat.Type, s)
				} else {
					coolqBot.SendMessage(msg.Chat.ID, msg.Chat.Type, handleErrMsg)
				}
			}
		}
	}
}
