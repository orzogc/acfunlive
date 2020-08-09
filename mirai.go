package main

import (
	"bytes"
	"fmt"
	"image"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	asciiart "github.com/yinghau76/go-ascii-art"
)

var cli *client.QQClient

func initMirai() {
	cli = client.NewClient()
	resp, err := cli.Login()
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

	lPrintln("QQ登陆 " + cli.Nickname + "（" + fmt.Sprint(cli.Uin) + "）" + " 成功")
	lPrintln("开始加载QQ好友列表")
	err = cli.ReloadFriendList()
	checkErr(err)
	lPrintln("共加载", len(cli.FriendList), "个QQ好友")
	lPrintln("开始加载QQ群列表")
	err = cli.ReloadGroupList()
	checkErr(err)
	lPrintln("共加载", len(cli.GroupList), "个QQ群")
}

func miraiSendQQ(qq int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(&message.TextElement{Content: text})
	cli.SendPrivateMessage(qq, msg)
}

func miraiSendQQGroup(qqGroup int64, text string) {
	msg := message.NewSendingMessage()
	msg.Append(message.AtAll())
	msg.Append(&message.TextElement{Content: text})
	cli.SendGroupMessage(qqGroup, msg)
}
