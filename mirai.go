// mirai QQ通知
package main

import (
	"bufio"
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
	AdminQQ       int64   `json:"adminQQ"`       // 管理者的QQ，通过这个QQ发送命令
	BotQQ         int64   `json:"botQQ"`         // bot的QQ号
	BotQQPassword string  `json:"botQQPassword"` // bot的QQ密码
	SendQQ        []int64 `json:"sendQQ"`        // 默认给这些QQ号发送消息，会被live.json里的设置覆盖
	SendQQGroup   []int64 `json:"sendQQGroup"`   // 默认给这些QQ群发送消息，会被live.json里的设置覆盖
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
func initMirai() (result bool) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in initMirai(), the error is:", err)
			lPrintErr("初始化Mirai出现错误，停止启动Mirai")
			if miraiClient != nil {
				miraiClient.Disconnect()
			}
			miraiClient = nil
			*isMirai = false
			result = false
		}
	}()

	miraiClient = client.NewClient(config.Mirai.BotQQ, config.Mirai.BotQQPassword)

	/*
		miraiClient.OnLog(func(c *client.QQClient, e *client.LogEvent) {
			switch e.Type {
			case "INFO":
				lPrintln("Mirai INFO: " + e.Message)
			case "ERROR":
				lPrintErr("Mirai ERROR: " + e.Message)
			case "DEBUG":
				lPrintln("Mirai DEBUG: " + e.Message)
			}
		})
	*/

	resp, err := miraiClient.Login()
	checkErr(err)

	for {
		if !resp.Success {
			switch resp.Error {
			case client.SliderNeededError:
				if client.SystemDeviceInfo.Protocol == client.AndroidPhone {
					lPrintWarn("Android Phone强制要求暂不支持的滑条验证码, 请开启设备锁或切换到Watch协议验证通过后再使用。")
					miraiClient.Disconnect()
					return false
				}
				miraiClient.AllowSlider = false
				miraiClient.Disconnect()
				resp, err = miraiClient.Login()
				checkErr(err)
				continue
			case client.NeedCaptcha:
				imageFile := filepath.Join(*configDir, qqCaptchaImage)
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
			case client.SMSNeededError:
				lPrintWarnf("QQ账号已开启设备锁, 向手机 %v 发送短信验证码", resp.SMSPhone)
				if !miraiClient.RequestSMS() {
					lPrintWarn("发送短信验证码失败，可能是请求过于频繁")
					miraiClient.Disconnect()
					return false
				}
				lPrintln("请输入短信验证码，按回车提交：")
				console := bufio.NewReader(os.Stdin)
				captcha, err := console.ReadString('\n')
				checkErr(err)
				resp, err = miraiClient.SubmitSMS(strings.ReplaceAll(strings.ReplaceAll(captcha, "\n", ""), "\r", ""))
				checkErr(err)
				continue
			case client.SMSOrVerifyNeededError:
				lPrintWarn("QQ账号已开启设备锁，请选择验证方式：")
				lPrintf("1. 向手机 %v 发送短信验证码", resp.SMSPhone)
				lPrintln("2. 使用手机QQ扫码验证")
				lPrintln("请输入1或2，按回车提交：")
				console := bufio.NewReader(os.Stdin)
				text, err := console.ReadString('\n')
				checkErr(err)
				if strings.Contains(text, "1") {
					if !miraiClient.RequestSMS() {
						lPrintWarn("发送短信验证码失败，可能是请求过于频繁")
						miraiClient.Disconnect()
						return false
					}
					lPrintln("请输入短信验证码，按回车提交：")
					captcha, err := console.ReadString('\n')
					checkErr(err)
					resp, err = miraiClient.SubmitSMS(strings.ReplaceAll(strings.ReplaceAll(captcha, "\n", ""), "\r", ""))
					checkErr(err)
					continue
				}
				lPrintWarnf("请前往 %s 验证并重启本程序", resp.VerifyUrl)
				miraiClient.Disconnect()
				return false
			case client.UnsafeDeviceError:
				lPrintWarnf("QQ账号已开启设备锁，请前往 %s 验证并重启本程序", resp.VerifyUrl)
				miraiClient.Disconnect()
				return false
			case client.OtherLoginError, client.UnknownLoginError:
				lPrintErrf("QQ登陆失败：%s", resp.ErrorMessage)
				miraiClient.Disconnect()
				return false
			default:
				lPrintErrf("QQ登陆出现未处理的错误，响应为：%+v", resp)
				miraiClient.Disconnect()
				return false
			}
		} else {
			break
		}
	}

	lPrintf("QQ登陆 %s（%d） 成功", miraiClient.Nickname, miraiClient.Uin)
	lPrintln("开始加载QQ好友列表")
	err = miraiClient.ReloadFriendList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.FriendList), "个QQ好友")
	lPrintln("开始加载QQ群列表")
	err = miraiClient.ReloadGroupList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.GroupList), "个QQ群")

	miraiClient.OnDisconnected(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		if miraiClient != nil {
			if miraiClient.Online {
				lPrintWarn("QQ帐号已登陆，无需重连")
				return
			}

			lPrintWarn("QQ Bot已离线，尝试重连")
			time.Sleep(10 * time.Second)
			resp, err := miraiClient.Login()
			if err != nil {
				lPrintErrf("QQ帐号重连失败，请重启本程序：%v", err)
				return
			}

			if !resp.Success {
				switch resp.Error {
				case client.NeedCaptcha:
					lPrintErr("QQ帐号重连失败：需要验证码，请重启本程序")
				case client.UnsafeDeviceError:
					lPrintErr("QQ帐号重连失败：设备锁")
					lPrintWarnf("QQ账号已开启设备锁，请前往 %s 验证并重启本程序", resp.VerifyUrl)
				default:
					lPrintErrf("QQ重连失败，请重启本程序，响应为：%+v", resp)
				}
			} else {
				lPrintln("QQ帐号重连成功")
			}
		} else {
			lPrintErr("miraiClient不能为nil")
		}
	})

	if config.Mirai.AdminQQ > 0 {
		miraiClient.OnPrivateMessage(privateMsgEvent)
		//miraiClient.OnTempMessage(tempMsgEvent)
		miraiClient.OnGroupMessage(groupMsgEvent)
	}

	return true
}

// 处理私聊消息事件
func privateMsgEvent(c *client.QQClient, m *message.PrivateMessage) {
	handlePrivateMsg(m.Sender.Uin, m.Elements)
}

// 处理临时会话消息事件
//func tempMsgEvent(c *client.QQClient, m *message.TempMessage) {
//	handlePrivateMsg(m.Sender.Uin, m.Elements)
//}

// 处理QQ bot接收到的私聊或临时会话消息
func handlePrivateMsg(qq int64, Elements []message.IMessageElement) {
	if qq == config.Mirai.AdminQQ {
		for _, ele := range Elements {
			if e, ok := ele.(*message.TextElement); ok {
				text := e.Content
				lPrintf("处理来自 QQ %d 的命令：%s", qq, text)
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
			lPrintf("处理来自QQ群 %d 里QQ %d 的命令：%s", m.GroupCode, m.Sender.Uin, cmd)
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
	if miraiClient != nil {
		lPrintln("给QQ", qq, "发送消息")
		if result := miraiClient.SendPrivateMessage(qq, msg); result == nil || result.Id <= 0 {
			lPrintErr("给QQ", qq, "发送消息失败")
		}
	} else {
		lPrintErr("miraiClient不能为nil")
	}
}

// 发送消息到指定的QQ群
func miraiSendQQGroup(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.NewText(text))
	if miraiClient != nil {
		lPrintln("给QQ群", qqGroup, "发送消息")
		if result := miraiClient.SendGroupMessage(qqGroup, msg); result == nil || result.Id <= 0 {
			lPrintErr("给QQ群", qqGroup, "发送消息失败")
		}
	} else {
		lPrintErr("miraiClient不能为nil")
	}
}

// 发送消息到指定的QQ群，并@全体成员
func miraiSendQQGroupAtAll(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.AtAll())
	msg.Append(message.NewText(text))
	if miraiClient != nil {
		lPrintln("给QQ群", qqGroup, "发送消息")
		if result := miraiClient.SendGroupMessage(qqGroup, msg); result == nil || result.Id <= 0 {
			lPrintErr("给QQ群", qqGroup, "发送@全体成员的消息失败，尝试发送普通群消息")
			miraiSendQQGroup(qqGroup, text)
		}
	} else {
		lPrintErr("miraiClient不能为nil")
	}
}

// 发送消息
func (s *streamer) sendMirai(text string) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendMirai(), the error is:", err)
			lPrintErrf("发送%s的消息（%s）到指定的QQ/QQ群时发生错误，取消发送", s.longID(), text)
		}
	}()

	if *isMirai && miraiClient != nil {
		sendQQ := config.Mirai.SendQQ
		if len(s.SendQQ) != 0 {
			sendQQ = s.SendQQ
		}
		for _, qq := range sendQQ {
			if qq > 0 {
				if miraiClient.FindFriend(qq) == nil {
					lPrintErrf("QQ号 %d 不是QQ机器人的好友，取消发送消息", qq)
					continue
				}
				miraiSendQQ(qq, text)
			} else {
				lPrintErrf("QQ号 %d 小于等于0，取消发送消息", qq)
			}
		}

		sendQQGroup := config.Mirai.SendQQGroup
		if len(s.SendQQGroup) != 0 {
			sendQQGroup = s.SendQQGroup
		}
		for _, qqGroup := range sendQQGroup {
			if qqGroup > 0 {
				if groupInfo := miraiClient.FindGroup(qqGroup); groupInfo == nil {
					lPrintErrf("QQ机器人未加入QQ群 %d，取消发送消息", qqGroup)
					continue
				} else {
					info := groupInfo.FindMember(config.Mirai.BotQQ)
					if info.Permission == client.Member {
						miraiSendQQGroup(qqGroup, text)
					} else {
						miraiSendQQGroupAtAll(qqGroup, text)
					}
				}
			} else {
				lPrintErrf("QQ群号 %d 小于等于0，取消发送消息", qqGroup)
			}
		}
	}
}

// 设置将主播的相关提醒消息发送到指定的QQ
func addQQNotify(uid int, qq int64) bool {
	s, ok := getStreamer(uid)
	if ok {
		for _, q := range s.SendQQ {
			if q == qq {
				lPrintf("已经设置过将%s的相关提醒消息发送到QQ%d", s.longID(), qq)
				return true
			}
		}
		s.SendQQ = append(s.SendQQ, qq)
	} else {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}
		s = streamer{UID: uid, Name: name, SendQQ: []int64{qq}}
	}
	setStreamer(s)
	lPrintf("成功设置将%s的相关提醒消息发送到QQ%d", s.longID(), qq)

	saveLiveConfig()
	return true
}

// 取消设置将主播的相关提醒消息发送到指定的QQ
func delQQNotify(uid int, qq int64) bool {
	if s, ok := getStreamer(uid); ok {
		var isSet bool
		for i, q := range s.SendQQ {
			if q == qq {
				s.SendQQ = append(s.SendQQ[:i], s.SendQQ[i+1:]...)
				isSet = true
				break
			}
		}
		if isSet {
			setStreamer(s)
			lPrintf("成功取消设置将%s的相关提醒消息发送到QQ%d", s.longID(), qq)
		} else {
			lPrintWarnf("没有设置过将%s的相关提醒消息发送到QQ%d", s.longID(), qq)
		}
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ提醒")
	}

	saveLiveConfig()
	return true
}

// 取消设置QQ提醒
func cancelQQNotify(uid int) bool {
	if s, ok := getStreamer(uid); ok {
		s.SendQQ = []int64{}
		setStreamer(s)
		lPrintln("成功取消设置" + s.longID() + "的QQ提醒")
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ提醒")
	}

	saveLiveConfig()
	return true
}

// 设置将主播的相关提醒消息发送到指定的QQ群
func addQQGroup(uid int, qqGroup int64) bool {
	s, ok := getStreamer(uid)
	if ok {
		for _, q := range s.SendQQGroup {
			if q == qqGroup {
				lPrintf("已经设置过将%s的相关提醒消息发送到QQ群%d", s.longID(), qqGroup)
				return true
			}
		}
		s.SendQQGroup = append(s.SendQQGroup, qqGroup)
	} else {
		name := getName(uid)
		if name == "" {
			lPrintWarn("不存在uid为" + itoa(uid) + "的用户")
			return false
		}
		s = streamer{UID: uid, Name: name, SendQQGroup: []int64{qqGroup}}
	}
	setStreamer(s)
	lPrintf("成功设置将%s的相关提醒消息发送到QQ群%d", s.longID(), qqGroup)

	saveLiveConfig()
	return true
}

// 取消设置将主播的相关提醒消息发送到指定的QQ群
func delQQGroup(uid int, qqGroup int64) bool {
	if s, ok := getStreamer(uid); ok {
		var isSet bool
		for i, q := range s.SendQQGroup {
			if q == qqGroup {
				s.SendQQGroup = append(s.SendQQGroup[:i], s.SendQQGroup[i+1:]...)
				isSet = true
				break
			}
		}
		if isSet {
			setStreamer(s)
			lPrintf("成功取消设置将%s的相关提醒消息发送到QQ群%d", s.longID(), qqGroup)
		} else {
			lPrintWarnf("没有设置过将%s的相关提醒消息发送到QQ群%d", s.longID(), qqGroup)
		}
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ群提醒")
	}

	saveLiveConfig()
	return true
}

// 取消设置QQ群提醒
func cancelQQGroup(uid int) bool {
	if s, ok := getStreamer(uid); ok {
		s.SendQQGroup = []int64{}
		setStreamer(s)
		lPrintln("成功取消设置" + s.longID() + "的QQ群提醒")
	} else {
		lPrintWarn("没有设置过uid为" + itoa(uid) + "的主播的QQ群提醒")
	}

	saveLiveConfig()
	return true
}
