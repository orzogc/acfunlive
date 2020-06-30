// 酷Q相关
package main

import (
	qqbotapi "github.com/catsworld/qq-bot-api"
)

// 是否连接酷Q
var isCoolq *bool

var bot *qqbotapi.BotAPI

// 酷Q相关设置数据
type coolqData struct {
	CqhttpWSAddr string // CQHTTP的WebSocket地址
	AdminQQ      int    // 管理者的QQ，通过这个QQ发送命令
	AccessToken  string // CQHTTP的access_token
	Secret       string // CQHTTP的secret
}

// 初始化酷Q
func initCoolq() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in initCoolq(), the error is:", err)
			lPrintln("连接酷Q出现问题，请确定已启动酷Q")
			bot = nil
		}
	}()

	newBot, err := qqbotapi.NewBotAPI(config.Coolq.AccessToken, config.Coolq.CqhttpWSAddr, config.Coolq.Secret)
	checkErr(err)
	bot = newBot

	u := qqbotapi.NewUpdate(0)
	u.PreloadUserInfo = true
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		//fmt.Printf("%s: %s", update.Message.From.String(), update.Message.Text)

		bot.SendMessage(update.Message.Chat.ID, update.Message.Chat.Type, update.Message.Text)
	}
}

// 发送消息到指定的QQ
func sendQQ(qq int64, text string) {
	s := bot.NewMessage(qq, "private").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息到指定的QQ群
func sendQQGroup(qqGroup int64, text string) {
	s := bot.NewMessage(qqGroup, "group").At("all").Text(text).Send()
	checkErr(s.Err)
}

// 发送消息
func (s streamer) sendCoolq(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in sendCoolq(), the error is:", err)
			lPrintln("发送" + s.longID() + "的消息到指定的QQ/QQ群时发生错误")
		}
	}()

	if *isCoolq && bot != nil {
		if s.SendQQ != 0 {
			sendQQ(s.SendQQ, text)
		}
		if s.SendQQGroup != 0 {
			sendQQGroup(s.SendQQGroup, text)
		}
	}
}
