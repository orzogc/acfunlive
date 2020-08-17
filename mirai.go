package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
)

const qqCaptchaImage = "qqcaptcha.jpg"

var (
	isMirai     *bool // 是否通过Mirai连接QQ
	miraiClient *client.QQClient
)

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
		if config.Mirai.BotQQ <= 0 || config.Mirai.BotQQPassword == "" {
			lPrintErr("请先在" + configFile + "里设置好Mirai相关配置")
			return false
		}
		*isMirai = true
		lPrintln("尝试利用Mirai登陆bot QQ", config.Mirai.BotQQ)
		if !initMirai() {
			lPrintErr("启动Mirai失败，请重新启动本程序")
			miraiClient = nil
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

	for {
		if !resp.Success {
			switch resp.Error {
			case client.NeedCaptcha:
				imageFile := filepath.Join(exeDir, qqCaptchaImage)
				err = ioutil.WriteFile(imageFile, resp.CaptchaImage, 0644)
				checkErr(err)
				lPrintln("QQ验证码图片保存在：" + imageFile)
				lPrintln("请输入验证码，按回车提交：")
				console := bufio.NewReader(os.Stdin)
				captcha, err := console.ReadString('\n')
				checkErr(err)
				resp, err = miraiClient.SubmitCaptcha(strings.ReplaceAll(captcha, "\n", ""), resp.CaptchaSign)
				checkErr(err)
				continue
			case client.UnsafeDeviceError:
				lPrintWarn("QQ账号已开启设备锁，请前往 " + resp.VerifyUrl + " 验证并重启本程序")
				return false
			case client.OtherLoginError, client.UnknownLoginError:
				lPrintErr("QQ登陆失败：" + resp.ErrorMessage)
				return false
			}
		}
		break
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
		lPrintWarn("QQ Bot已离线，尝试重连")
		time.Sleep(10 * time.Second)
		resp, err := miraiClient.Login()
		checkErr(err)

		if !resp.Success {
			switch resp.Error {
			case client.NeedCaptcha:
				lPrintErr("QQ重连失败：需要验证码，请重启本程序")
			case client.UnsafeDeviceError:
				lPrintErr("QQ重连失败：设备锁")
				lPrintWarn("QQ账号已开启设备锁，请前往 " + resp.VerifyUrl + " 验证并重启本程序")
			case client.OtherLoginError, client.UnknownLoginError:
				lPrintErr("QQ重连失败：" + resp.ErrorMessage + "，请重启本程序")
			}
		}
	})

	if config.Mirai.AdminQQ > 0 {
		miraiClient.OnPrivateMessage(privateMsgEvent)
		miraiClient.OnTempMessage(tempMsgEvent)
		miraiClient.OnGroupMessage(groupMsgEvent)
	}

	return true
}

// 处理私聊消息事件
func privateMsgEvent(c *client.QQClient, m *message.PrivateMessage) {
	handlePrivateMsg(m.Sender.Uin, m.Elements)
}

// 处理临时会话消息事件
func tempMsgEvent(c *client.QQClient, m *message.TempMessage) {
	handlePrivateMsg(m.Sender.Uin, m.Elements)
}

// 处理QQ bot接收到的私聊或临时会话消息
func handlePrivateMsg(qq int64, Elements []message.IMessageElement) {
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

// 处理群消息事件
func groupMsgEvent(c *client.QQClient, m *message.GroupMessage) {
	if m.Sender.Uin == config.Mirai.AdminQQ {
		var isAt bool
		var text []string
		for _, ele := range m.Elements {
			switch e := ele.(type) {
			case *message.AtElement:
				if e.Target == config.Mirai.BotQQ {
					isAt = true
				}
			case *message.TextElement:
				text = append(text, e.Content)
			}
		}
		if isAt {
			cmd := strings.Join(text, "")
			lPrintln(fmt.Sprintf("处理来自QQ群 %d 里QQ %d 的命令：%s", m.GroupCode, m.Sender.Uin, cmd))
			if s := handleAllCmd(cmd); s != "" {
				miraiSendQQGroup(m.GroupCode, s)
			} else {
				miraiSendQQGroup(m.GroupCode, handleErrMsg)
			}
		}
	}
}

// 发送消息到指定的QQ
func miraiSendQQ(qq int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(text))
	if result := miraiClient.SendPrivateMessage(qq, msg); result == nil {
		lPrintErr("给QQ", qq, "的消息发送失败")
	}
}

// 发送消息到指定的QQ群
func miraiSendQQGroup(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(text))
	if result := miraiClient.SendGroupMessage(qqGroup, msg); result == nil {
		lPrintErr("给QQ群", qqGroup, "的消息发送失败")
	}
}

// 发送消息到指定的QQ群，并@全体成员
func miraiSendQQGroupAtAll(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.AtAll())
	msg.Append(message.NewText(text))
	if result := miraiClient.SendGroupMessage(qqGroup, msg); result == nil {
		lPrintErr("给QQ群", qqGroup, "的消息发送失败")
	}
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
			miraiSendQQGroupAtAll(s.SendQQGroup, text)
		}
	}
}
