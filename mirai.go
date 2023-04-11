// mirai QQ通知
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/mattn/go-colorable"
)

const qqCaptchaImage = "qqcaptcha.jpg"
const qqDeviceFile = "qqdevice.json"
const qqQRCodeImage = "qqqrcode.png"
const qqSessionTokenFile = "qqsession.token"

var (
	isMirai     *bool // 是否通过Mirai连接QQ
	miraiClient *client.QQClient
	token       []byte
)

// Mirai相关设置数据
type miraiData struct {
	AdminQQ       int64   `json:"adminQQ"`       // 管理者的QQ，通过这个QQ发送命令
	BotQQ         int64   `json:"botQQ"`         // bot的QQ号
	BotQQPassword string  `json:"botQQPassword"` // bot的QQ密码
	SendQQ        []int64 `json:"sendQQ"`        // 默认给这些QQ号发送消息，会被live.json里的设置覆盖
	SendQQGroup   []int64 `json:"sendQQGroup"`   // 默认给这些QQ群发送消息，会被live.json里的设置覆盖
}

// 在终端打印二维码
func printQRCode(imgData []byte) {
	const (
		black = "\033[48;5;0m  \033[0m"
		white = "\033[48;5;7m  \033[0m"
	)
	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		log.Panic(err)
	}
	data := img.(*image.Gray).Pix
	bound := img.Bounds().Max.X
	buf := make([]byte, 0, (bound*4+1)*(bound))
	i := 0
	for y := 0; y < bound; y++ {
		i = y * bound
		for x := 0; x < bound; x++ {
			if data[i] != 255 {
				buf = append(buf, white...)
			} else {
				buf = append(buf, black...)
			}
			i++
		}
		buf = append(buf, '\n')
	}
	_, _ = colorable.NewColorableStdout().Write(buf)
}

// 使用二维码登陆QQ
func qrCodeLogin() (e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("qrCodeLogin() error: %v", err)
		}
	}()

	resp, err := miraiClient.FetchQRCodeCustomSize(1, 2, 1)
	checkErr(err)
	qrCodeImage := filepath.Join(*configDir, qqQRCodeImage)
	err = os.WriteFile(qrCodeImage, resp.ImageData, 0600)
	checkErr(err)
	defer func() {
		_ = os.Remove(qrCodeImage)
	}()
	lPrintf("请使用手机QQ登陆帐号%d，然后扫描图片 %s 或者下面的二维码：", config.Mirai.BotQQ, qrCodeImage)
	time.Sleep(time.Second)
	printQRCode(resp.ImageData)
	s, err := miraiClient.QueryQRCodeStatus(resp.Sig)
	checkErr(err)
	prevState := s.State

	for {
		time.Sleep(time.Second)
		s, _ = miraiClient.QueryQRCodeStatus(resp.Sig)
		if s == nil {
			continue
		}
		if prevState == s.State {
			continue
		}
		prevState = s.State
		switch s.State {
		case client.QRCodeCanceled:
			lPrintErr("扫码登陆被用户取消，如要重新登陆QQ，请重启本程序")
			panic("扫码登陆被用户取消")
		case client.QRCodeTimeout:
			lPrintErr("二维码过期，请重启本程序以重新登陆QQ")
			panic("二维码过期")
		case client.QRCodeWaitingForConfirm:
			lPrintln("扫码成功, 请在手机端确认登录")
		case client.QRCodeConfirmed:
			resp, err := miraiClient.QRCodeLogin(s.LoginInfo)
			checkErr(err)
			err = handleLoginResp(resp)
			checkErr(err)
			return nil
		case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
		default:
			lPrintWarnf("未知的扫码状态：%v", s.State)
		}
	}
}

// 处理登陆返回
func handleLoginResp(resp *client.LoginResponse) (e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("handleLoginResp() error: %v", err)
		}
	}()

	for {
		if !resp.Success {
			switch resp.Error {
			case client.SliderNeededError:
				lPrintWarn("登录需要滑条验证码，请参考文档 https://github.com/Mrs4s/go-cqhttp/blob/master/docs/slider.md 抓包获取 Ticket")
				lPrintWarnf("请用浏览器打开 %s 并获取Ticket", resp.VerifyUrl)
				lPrintln("请输入Ticket，按回车提交：")
				console := bufio.NewReader(os.Stdin)
				ticket, err := console.ReadString('\n')
				checkErr(err)
				resp, err = miraiClient.SubmitTicket(strings.ReplaceAll(ticket, "\n", ""))
				checkErr(err)
				continue
			case client.NeedCaptcha:
				imageFile := filepath.Join(*configDir, qqCaptchaImage)
				err := os.WriteFile(imageFile, resp.CaptchaImage, 0644)
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
				lPrintWarnf("QQ账号已开启设备锁, 向手机 %s 发送短信验证码", resp.SMSPhone)
				if !miraiClient.RequestSMS() {
					lPrintWarn("发送短信验证码失败，可能是请求过于频繁")
					return fmt.Errorf("使用短信验证码登陆失败")
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
				lPrintf("1. 向手机 %s 发送短信验证码", resp.SMSPhone)
				lPrintln("2. 使用手机QQ扫码验证")
				lPrintln("请输入1或2，按回车提交：")
				console := bufio.NewReader(os.Stdin)
				text, err := console.ReadString('\n')
				checkErr(err)
				if strings.Contains(text, "1") {
					if !miraiClient.RequestSMS() {
						lPrintWarn("发送短信验证码失败，可能是请求过于频繁")
						return fmt.Errorf("使用短信验证码登陆失败")
					}
					lPrintln("请输入短信验证码，按回车提交：")
					captcha, err := console.ReadString('\n')
					checkErr(err)
					resp, err = miraiClient.SubmitSMS(strings.ReplaceAll(strings.ReplaceAll(captcha, "\n", ""), "\r", ""))
					checkErr(err)
					continue
				}
				lPrintWarnf("请前往 %s 验证并重启本程序", resp.VerifyUrl)
				return fmt.Errorf("请重启本程序再次登陆QQ")
			case client.UnsafeDeviceError:
				lPrintWarnf("QQ账号已开启设备锁，请前往 %s 验证并重启本程序", resp.VerifyUrl)
				return fmt.Errorf("请重启本程序再次登陆QQ")
			case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
				msg := resp.ErrorMessage
				lPrintErrf("QQ登陆失败，code：%v，错误信息：%s", resp.Code, msg)
				if resp.Code == 235 {
					lPrintf("请删除 %s 后重试", filepath.Join(*configDir, qqDeviceFile))
				}

				return fmt.Errorf("登陆QQ失败")
			default:
				lPrintErrf("QQ登陆出现未处理的错误，响应为：%+v", resp)
				return fmt.Errorf("登陆QQ失败")
			}
		} else {
			break
		}
	}

	return nil
}

// 保存会话缓存
func saveToken(file string) error {
	if miraiClient != nil {
		token = miraiClient.GenToken()
		return os.WriteFile(file, token, 0600)
	}

	return fmt.Errorf("没有登陆bot QQ")
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
				miraiClient.Release()
			}
			miraiClient = nil
			*isMirai = false
			result = false
		}
	}()

	miraiClient = client.NewClient(config.Mirai.BotQQ, config.Mirai.BotQQPassword)

	device := new(client.DeviceInfo)
	deviceFile := filepath.Join(*configDir, qqDeviceFile)
	if _, err := os.Stat(deviceFile); err == nil {
		data, err := os.ReadFile(deviceFile)
		checkErr(err)
		err = device.ReadJson(data)
		checkErr(err)
	} else if os.IsNotExist(err) {
		device = client.GenRandomDevice()
		err := os.WriteFile(deviceFile, device.ToJson(), 0644)
		checkErr(err)
	} else {
		panic(fmt.Sprintf("读取QQ设备文件失败：%v", err))
	}
	miraiClient.UseDevice(device)

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

	isTokenLogin := false
	SessionTokenFile := filepath.Join(*configDir, qqSessionTokenFile)
	if _, err := os.Stat(SessionTokenFile); err == nil {
		data, err := os.ReadFile(SessionTokenFile)
		if err == nil {
			reader := binary.NewReader(data)
			uin := reader.ReadInt64()
			if uin != config.Mirai.BotQQ {
				lPrintWarnf("bot的QQ号%d和会话缓存文件里的QQ号%d不一致", config.Mirai.BotQQ, uin)
				lPrintWarnf("取消登陆QQ，如要登陆QQ，请删除会话缓存文件 %s 或者修改配置文件 %s 里的botQQ", SessionTokenFile, configFileLocation)
				return false
			}

			if err = miraiClient.TokenLogin(data); err != nil {
				_ = os.Remove(SessionTokenFile)
				lPrintErrf("恢复会话失败: %v , 尝试使用正常流程登录", err)
				time.Sleep(time.Second)
				miraiClient.Disconnect()
				miraiClient.Release()
				miraiClient = client.NewClient(config.Mirai.BotQQ, config.Mirai.BotQQPassword)
				miraiClient.UseDevice(device)
			} else {
				isTokenLogin = true
			}
		} else {
			lPrintErrf("读取会话缓存文件 %s 失败：%v", SessionTokenFile, err)
		}
	}

	if !isTokenLogin {
		if device.Protocol == 2 {
			err := qrCodeLogin()
			checkErr(err)
		} else {
			resp, err := miraiClient.Login()
			checkErr(err)
			err = handleLoginResp(resp)
			checkErr(err)
		}
	}

	err := saveToken(SessionTokenFile)
	checkErr(err)
	miraiClient.AllowSlider = true
	lPrintf("QQ登陆 %s（%d） 成功", miraiClient.Nickname, miraiClient.Uin)
	lPrintln("开始加载QQ好友列表")
	err = miraiClient.ReloadFriendList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.FriendList), "个QQ好友")
	lPrintln("开始加载QQ群列表")
	err = miraiClient.ReloadGroupList()
	checkErr(err)
	lPrintln("共加载", len(miraiClient.GroupList), "个QQ群")

	var reLoginLock sync.Mutex
	miraiClient.DisconnectedEvent.Subscribe(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		if miraiClient != nil {
			reLoginLock.Lock()
			defer reLoginLock.Unlock()

			if miraiClient.Online.Load() {
				lPrintWarn("QQ帐号已登陆，无需重连")
				return
			}

			lPrintWarn("QQ Bot已离线，尝试重连")
			time.Sleep(10 * time.Second)

			err := miraiClient.TokenLogin(token)
			if err == nil {
				err = saveToken(SessionTokenFile)
				if err != nil {
					lPrintErrf("无法保存会话缓存文件 %s ：%v", SessionTokenFile, err)
				}
				return
			}

			lPrintWarnf("快速重连失败：%v", err)
			if device.Protocol == 2 {
				lPrintErrf("扫码登录无法重连，请重启本程序")
				return
			}

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
				err = saveToken(SessionTokenFile)
				if err != nil {
					lPrintErrf("无法保存会话缓存文件 %s ：%v", SessionTokenFile, err)
				}
			}
		} else {
			lPrintErr("miraiClient不能为nil")
		}
	})

	if config.Mirai.AdminQQ > 0 {
		miraiClient.PrivateMessageEvent.Subscribe(privateMsgEvent)
		miraiClient.GroupMessageEvent.Subscribe(groupMsgEvent)
	}

	return true
}

// 处理私聊消息事件
func privateMsgEvent(c *client.QQClient, m *message.PrivateMessage) {
	handlePrivateMsg(m.Sender.Uin, m.Elements)
}

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
	if qq <= 0 {
		lPrintErrf("QQ号 %d 小于等于0，取消发送消息", qq)
		return
	}
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
	if qqGroup <= 0 {
		lPrintErrf("QQ群号 %d 小于等于0，取消发送消息", qqGroup)
		return
	}
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
	if qqGroup <= 0 {
		lPrintErrf("QQ群号 %d 小于等于0，取消发送消息", qqGroup)
		return
	}
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
func (s *streamer) sendMirai(text string, isSendGroup bool) {
	defer func() {
		if err := recover(); err != nil {
			lPrintErr("Recovering from panic in sendMirai(), the error is:", err)
			lPrintErrf("发送%s的消息（%s）到指定的QQ/QQ群时发生错误，取消发送", s.longID(), text)
		}
	}()

	if *isMirai && miraiClient != nil {
		//text = strings.ReplaceAll(text, "（", "(")
		//text = strings.ReplaceAll(text, "）", ")")

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

		if isSendGroup {
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
			lPrintWarnf("不存在uid为%d的用户", uid)
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
		lPrintWarnf("没有设置过uid为%d的主播的QQ提醒", uid)
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
		lPrintWarnf("没有设置过uid为%d的主播的QQ提醒", uid)
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
			lPrintWarnf("不存在uid为%d的用户", uid)
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
		lPrintWarnf("没有设置过uid为%d的主播的QQ群提醒", uid)
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
		lPrintWarnf("没有设置过uid为%d的主播的QQ群提醒", uid)
	}

	saveLiveConfig()
	return true
}
