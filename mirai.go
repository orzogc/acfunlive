package main

import (
	"bytes"
	"fmt"
	"image"
	"time"

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

// 启动Mirai
func startMirai() bool {
	if *isMirai {
		lPrintWarn("已经启动过Mirai")
	} else {
		*isMirai = true
		lPrintln("尝试利用Mirai登陆bot QQ", config.Mirai.BotQQ)
		if !initMirai() {
			lPrintErr("启动Mirai失败，请重新启动Mirai")
			*isMirai = false
			return false
		}
	}
	return true
}

// 初始化Mirai
func initMirai() bool {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in initMirai(), the error is:", err)
			lPrintErr("初始化Mirai出现错误，停止启动Mirai")
			miraiClient = nil
			*isMirai = false
		}
	}()

	miraiClient = client.NewClient(config.Mirai.BotQQ, config.Mirai.BotQQPassword)
	resp, err := miraiClient.Login()
	checkErr(err)

	if !resp.Success {
		switch resp.Error {
		case client.NeedCaptcha:
			img, format, err := image.Decode(bytes.NewReader(resp.CaptchaImage))
			checkErr(err)
			lPrintln("QQ image format:", format)
			lPrintln("验证码图片：\n" + asciiart.New("image", img).Art)
			lPrintWarn("目前暂不支持验证码登陆")
			return false
		case client.UnsafeDeviceError:
			lPrintWarn("账号已开启设备锁，请前往 " + resp.VerifyUrl + " 验证并重启Mirai")
			return false
		case client.OtherLoginError, client.UnknownLoginError:
			lPrintErr("登陆失败：" + resp.ErrorMessage)
			return false
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

	miraiClient.OnDisconnected(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		lPrintWarn("Bot已离线，尝试重连")
		time.Sleep(10 * time.Second)
		resp, err := miraiClient.Login()
		checkErr(err)

		if !resp.Success {
			switch resp.Error {
			case client.NeedCaptcha:
				lPrintErr("重连失败：需要验证码")
			case client.UnsafeDeviceError:
				lPrintErr("重连失败：设备锁")
			case client.OtherLoginError, client.UnknownLoginError:
				lPrintErr("重连失败：" + resp.ErrorMessage)
			}
		}
	})

	if config.Mirai.AdminQQ > 0 {
		miraiClient.OnPrivateMessage(privateMsgEvent)
		miraiClient.OnTempMessage(tempMsgEvent)
	}

	return true
}

// 处理私人信息事件
func privateMsgEvent(c *client.QQClient, m *message.PrivateMessage) {
	handleMiraiMsg(m.Elements, m.Sender.Uin)
}

// 处理临时信息事件
func tempMsgEvent(c *client.QQClient, m *message.TempMessage) {
	handleMiraiMsg(m.Elements, m.Sender.Uin)
}

// 处理QQ bot接受到的信息
func handleMiraiMsg(Elements []message.IMessageElement, qq int64) {
	if qq == config.Mirai.AdminQQ {
		for _, ele := range Elements {
			if e, ok := ele.(*message.TextElement); ok {
				text := e.Content
				lPrintln(fmt.Sprintf("处理来自 QQ %d 的命令：%s", qq, text))
				if s := handleAllCmd(text); s != "" {
					miraiSendQQ(qq, s)
				} else {
					miraiSendQQ(qq, handleErrMsg)
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

// 发送消息
func (s streamer) sendMirai(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendMirai(), the error is:", err)
			lPrintErr(fmt.Sprintf("发送%s的消息（%s）到指定的QQ（%d）/QQ群（%d）时发生错误，取消发送", s.longID(), text, s.SendQQ, s.SendQQGroup))
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
