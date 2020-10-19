# acfunlive
AcFun直播通知和下载助手

* [acfunlive](#acfunlive)
    * [依赖](#依赖)
      * [运行依赖](#运行依赖)
      * [编译依赖](#编译依赖)
    * [编译](#编译)
      * [使用GNU Make](#使用gnu-make)
      * [不使用GNU Make](#不使用gnu-make)
    * [配置文件详解](#配置文件详解)
      * [live\.json](#livejson)
      * [config\.json](#configjson)
    * [使用方法](#使用方法)
    * [web API](#web-api)
    * [Mirai使用方法](#mirai使用方法)
    * [酷Q使用方法](#酷q使用方法)

### 依赖
#### 运行依赖
* ffmpeg（下载直播视频需要，不下载不需要，Windows需要将ffmpeg.exe放在本程序所在文件夹内）
* gtk3 和 libappindicator3 （Linux下需要）

#### 编译依赖
* go
* yarn
* gtk3 和 libappindicator3 （Linux下需要）
* GNU Make （Linux下可选）

### 编译
#### 使用GNU Make
```
# 更新repo需另外运行 git submodule update --remote --merge
git clone --recursive https://github.com/orzogc/acfunlive.git
cd acfunlive
# 编译Windows版本运行 make build-windows-gui 或 make build-windows-cli
make
```
编译好的文件在bin文件夹下

#### 不使用GNU Make
```
# 更新repo需另外运行 git submodule update --remote --merge
git clone --recursive https://github.com/orzogc/acfunlive.git
# Windows下编译没有控制台的gui版本需加上 -ldflags -H=windowsgui 参数
go build
cd acfunlive-ui
yarn install
yarn generate
```
在编译好的`acfunlive`或`acfunlive.exe`所在的文件夹下新建webui文件夹，将acfunlive-ui下dist文件夹内的所有文件拷贝到webui文件夹内

### 配置文件详解
#### live.json
`live.json`的内容可以手动修改，本程序会自动读取更改后的设置，无需重新启动本程序
```
{
    "UID": 23682490,    // 主播的uid
    "Name": "AC娘本体", // 主播的昵称
    "Notify": {
        "NotifyOn": true,     // 主播开播通知
        "NotifyOff": false,   // 主播下播通知，需自行手动修改设置
        "NotifyRecord": true, // 下载主播直播相关的通知
        "NotifyDanmu": false  // 下载主播直播弹幕相关的通知，需自行手动修改设置
        },
    "Record": true,     // 是否下载直播视频
    "Danmu": true,      // 是否下载直播弹幕
    "KeepOnline": true, // 是否在该主播的直播间挂机，目前主要用于挂粉丝牌等级
    "Bitrate": 0,       // 设置要下载的直播源的最高码率（Kbps），需自行手动修改设置
    "SendQQ": [         // 发送开播提醒到数组里的所有QQ（需要QQ机器人添加这些QQ为好友）
            12345,
            123456
        ],
    "SendQQGroup": [ // 发送开播提醒到数组里的所有QQ群（需要QQ机器人在这些QQ群里，最好是管理员，会@全体成员）
            1234567
        ]
}
```
Bitrate默认为0，相当于默认下载码率最高的直播源，如果设置为其他数字，则会下载码率小于等于Bitrate的条件下码率最高的直播源。直播源具体的名字和码率的对应看下表：
| 直播源名字 | 高清 | 超清 | 蓝光 4M | 蓝光 5M | 蓝光 6M | 蓝光 7M | 蓝光 8M |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 码率 | 1000/2000 | 2000/3000 | 4000 | 5000 | 6000 | 7000 | 8000 |

#### config.json
`config.json`的内容手动修改后需要重新启动本程序以生效
```
{
    "Source": "flv",  // 直播源，有hls和flv两种，默认是flv
    "Output": "mp4",  // 下载的直播视频的格式，必须是有效的视频格式后缀名
    "WebPort": 51880, // web API的本地端口，使用web UI的话不能修改这个端口
    "Directory": "",  // 直播视频和弹幕下载完毕后会被移动到该文件夹，其值最好是绝对路径
    "Acfun": {
        "UserEmail": "", // AcFun帐号邮箱或手机号，目前只用于直播间挂机，不需要可以为空
        "Password": ""   // AcFun帐号密码
    },
    "Mirai": {
        "AdminQQ": 12345,        // 用来发送命令控制本程序的管理者QQ，可选
        "BotQQ": 123456,         // QQ机器人的QQ号
        "BotQQPassword": "abcde" // QQ机器人QQ号的密码
    },
    "Coolq": {
        "CqhttpWSAddr": "ws://localhost:6700", // CQHTTP的WebSocket地址和端口
        "AdminQQ": 12345,                      // 用来发送命令控制本程序的管理者QQ，可选
        "AccessToken": "",                     // CQHTTP的access_token，可以为空
        "Secret": ""                           // CQHTTP的secret，可以为空
    }
}
```

### 使用方法
Windows的gui版本直接运行即可，程序会出现在系统托盘那里，可以通过`http://localhost:51890`访问web UI界面。

Windows下如果要使用命令行模式，下载cli版本，运行需要加上`-nogui`参数，具体参数看 [cli.md](https://github.com/orzogc/acfunlive/blob/master/doc/cli.md) 。

本程序下载的视频和弹幕默认保存在本程序所在文件夹内。

命令行模式运行时可以输入命令控制本程序，运行时输入help查看具体命令。

### web API
具体看 [webapi.md](https://github.com/orzogc/acfunlive/blob/master/doc/webapi.md)

### Mirai使用方法
**本项目使用 [MiraiGo](https://github.com/Mrs4s/MiraiGo) 。**

命令行模式启动时加上`-mirai`参数，需要在`config.json`里的Mirai对象设置机器人QQ号和密码。

如果由于设备锁无法登陆，请利用日志里的链接验证后重新启动本程序。

`config.json`里Mirai对象的AdminQQ为自己的QQ号时，添加QQ机器人为好友或者将QQ机器人加进QQ群后，可以发送命令给机器人控制本程序（在QQ群里需要@机器人的昵称），发送help查看具体命令。

### 酷Q使用方法
**酷Q官方已经停止维护，本项目也不会有后续维护。**

本程序使用 [酷Q](https://cqp.cc/) 和 [CQHTTP](https://github.com/richardchien/coolq-http-api) 作为WebSocket服务端来发送QQ消息，请事先设置好酷Q和CQHTTP插件并启动酷Q，具体可以看 [CQHTTP的文档](https://richardchien.gitee.io/coolq-http-api/docs/) 。

CQHTTP插件必须启用WebSocket服务端，也就是其配置里的use_ws必须为true。

本程序酷Q相关配置参考 [config\.json配置](#configjson) 。

目前群通知@全体成员 需要酷Q Pro。

`config.json`里Coolq对象的AdminQQ为自己的QQ号时，添加QQ机器人为好友或者将QQ机器人加进QQ群后，可以发送命令给机器人控制本程序（在QQ群里需要@机器人的昵称），发送help查看具体命令。
