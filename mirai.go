package main

import (
	"bytes"
	"fmt"
	"image"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	asciiart "github.com/yinghau76/go-ascii-art"
)

// 是否通过Mirai连接QQ
var isMirai *bool

var miraiClient *client.QQClient = nil

// Mirai相关设置数据
type miraiData struct {
	AdminQQ       int64  // 管理者的QQ，通过这个QQ发送命令
	BotQQ         int64  // bot的QQ号
	BotQQPassword string // bot的QQ密码
}

func initMirai() {
	miraiClient = client.NewClient(config.Mirai.BotQQ, config.Mirai.BotQQPassword)
	resp, err := miraiClient.Login()
	checkErr(err)

	if !resp.Success {
		switch resp.Error {
		case client.NeedCaptcha:
			img, format, err := image.Decode(bytes.NewReader(resp.CaptchaImage))
			checkErr(err)
			lPrintln("qq image format:", format)
			lPrintln("验证码图片：\n" + asciiart.New("image", img).Art)
			lPrintln("目前暂不支持验证码登陆")
			return
		case client.UnsafeDeviceError:
			lPrintWarn("账号已开启设备锁，请前往 " + resp.VerifyUrl + " 验证并重启Mirai")
			return
		case client.OtherLoginError, client.UnknownLoginError:
			lPrintErr("登陆失败：" + resp.ErrorMessage)
			return
		}
	}

	lPrintln(fmt.Sprintf("QQ登陆 %s（%d） 成功", miraiClient.Nickname, miraiClient.Uin))
	lPrintln("开始加载QQ好友列表")
	err = miraiClient.ReloadFriendList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.FriendList), "个QQ好友")
	lPrintln("开始加载QQ群列表")
	err = miraiClient.ReloadGroupList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.GroupList), "个QQ群")

	miraiClient.OnPrivateMessage(privateMsgEvent)
}

func privateMsgEvent(c *client.QQClient, m *message.PrivateMessage) {
	if m.Sender.Uin == config.Mirai.AdminQQ {
		for _, e := range m.Elements {
			switch e.Type() {
			case message.Text:
				text := e.(*message.TextElement).Content
				lPrintln(fmt.Sprintf("处理来自QQ%d的命令：%s", m.Sender.Uin, text))
				if s := handleAllCmd(text); text != "" {
					miraiSendQQ(m.Sender.Uin, s)
				} else {
					miraiSendQQ(m.Sender.Uin, handleErrMsg)
				}
			}
		}
	}
}

// 发送消息到指定的QQ
func miraiSendQQ(qq int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(&message.TextElement{Content: text})
	miraiClient.SendPrivateMessage(qq, msg)
}

// 发送消息到指定的QQ群
func miraiSendQQGroup(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.AtAll())
	msg.Append(&message.TextElement{Content: text})
	miraiClient.SendGroupMessage(qqGroup, msg)
}

func (s streamer) sendMirai(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendMirai(), the error is:", err)
			lPrintErr("发送" + s.longID() + "的消息（" + text + "）到指定的QQ（" + itoa(int(s.SendQQ)) + "）/QQ群（" + itoa(int(s.SendQQGroup)) + "）时发生错误，取消发送")
		}
	}()

	if *isMirai && miraiClient != nil {
		if s.SendQQ > 0 {
			miraiSendQQ(s.SendQQ, text)
		}
		if s.SendQQGroup > 0 {
			miraiSendQQGroup(s.SendQQGroup, text)
		}
	}
}
