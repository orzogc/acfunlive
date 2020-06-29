// 酷Q相关
package main

import qqbotapi "github.com/catsworld/qq-bot-api"

var isCoolq *bool

var bot *qqbotapi.BotAPI

// 酷Q相关设置数据
type coolqData struct {
	CqhttpPort    int    // CQHTTP的端口
	CqhttpPostURL string // CQHTTP的post_url
	AccessToken   string // CQHTTP的access_token
	Secret        string // CQHTTP的secret
}

func initCoolq() {
	defer func() {
		if err := recover(); err != nil {
			lPrintln("Recovering from panic in initCoolq(), the error is:", err)
			lPrintln("连接酷Q出现问题，请确定已启动酷Q")
			bot = nil
		}
	}()

	newBot, err := qqbotapi.NewBotAPI(config.Coolq.AccessToken, address(config.Coolq.CqhttpPort), config.Coolq.Secret)
	checkErr(err)
	bot = newBot
}

func sendQQ(qq int64, text string) {
	s := bot.NewMessage(qq, "private").Text(text).Send()
	checkErr(s.Err)
}

func sendQQGroup(qqGroup int64, text string) {
	s := bot.NewMessage(qqGroup, "group").At("all").Text(text).Send()
	checkErr(s.Err)
}

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
