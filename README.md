# acfunlive

AcFun 直播通知和下载助手

- [acfunlive](#acfunlive)
  - [依赖](#依赖)
    - [运行依赖](#运行依赖)
    - [编译依赖](#编译依赖)
  - [编译](#编译)
    - [使用 GNU Make](#使用-gnu-make)
    - [不使用 GNU Make](#不使用-gnu-make)
  - [配置文件详解](#配置文件详解)
    - [live.json](#livejson)
    - [config.json](#configjson)
  - [使用方法](#使用方法)
  - [web API](#web-api)
  - [Mirai 使用方法](#mirai-使用方法)
  - [Docker](#docker)

### 依赖

#### 运行依赖

- ffmpeg（下载直播视频需要，不下载不需要，Windows 需要将 ffmpeg.exe 放在本程序所在文件夹内）
- gtk3 和 libayatana-appindicator3（Linux 下运行 GUI 版本需要）

#### 编译依赖

- go
- yarn
- gtk3 和 libayatana-appindicator3（Linux 下编译 GUI 版本需要）
- GNU Make（Linux 下可选）

### 编译

#### 使用 GNU Make

```
# 更新repo需另外运行 git submodule update --remote --merge
git clone --recursive https://github.com/orzogc/acfunlive.git
cd acfunlive
# 编译GUI版本运行 make build-gui ，编译Windows版本运行 make build-windows-gui 或 make build-windows-cli
make
```

编译好的文件在 bin 文件夹下

#### 不使用 GNU Make

```
# 更新repo需另外运行 git submodule update --remote --merge
git clone --recursive https://github.com/orzogc/acfunlive.git
cd acfunlive
# Linux下编译GUI版本需加上 -tags tray 参数，Windows下编译没有控制台的GUI版本需加上 -tags tray -ldflags -H=windowsgui 参数
go build
# 如果不需要webui可以不运行下面的命令
cd acfunlive-ui
yarn install
yarn generate
```

在编译好的`acfunlive`或`acfunlive.exe`所在的文件夹下新建 webui 文件夹，将 acfunlive-ui 下 dist 文件夹内的所有文件复制到 webui 文件夹内

### 配置文件详解

可以先运行一次本程序以生成配置文件。

配置文件`config.json`和`live.json`默认保存在本程序所在文件夹内，运行时可用参数`-config`指定配置文件所在文件夹。

#### live.json

`live.json`的内容可以手动修改，本程序会自动读取更改后的设置，无需重新启动本程序

```
[
    {
        "uid": 23682490,    // 主播的uid
        "name": "AC娘本体", // 主播的昵称
        "notify": {
            "notifyOn": true,     // 主播开播通知
            "notifyOff": false,   // 主播下播通知
            "notifyRecord": true, // 下载主播直播相关的通知
            "notifyDanmu": false  // 下载主播直播弹幕相关的通知
            },
        "record": true,     // 是否下载直播视频
        "danmu": true,      // 是否下载直播弹幕
        "keepOnline": true, // 是否在该主播的直播间挂机，目前主要用于挂粉丝牌等级
        "bitrate": 0,       // 设置要下载的直播源的最高码率（Kbps），需自行手动修改设置
        "directory": "",    // 直播视频和弹幕下载结束后会被移动到该文件夹，其值最好是绝对路径，会覆盖config.json里的设置，需自行手动修改设置
        "sendQQ": [         // 发送开播提醒和录播相关消息到数组里的所有QQ（需要QQ机器人添加这些QQ为好友），会覆盖config.json里的设置，QQ号小于等于0会取消通知QQ
            12345,
            123456
        ],
        "sendQQGroup": [ // 发送开播提醒到数组里的所有QQ群（需要QQ机器人在这些QQ群里，最好是管理员，会@全体成员），会覆盖config.json里的设置，QQ群号小于等于0会取消通知QQ群
            1234567
        ]
    }
]
```

`bitrate`默认为 0，相当于默认下载码率最高的直播源，如果设置为其他数字，则会下载码率小于等于`bitrate`的条件下码率最高的直播源。直播源具体的名字和码率的对应看下表：
| 直播源名字 | 高清 | 超清 | 蓝光 4M | 蓝光 5M | 蓝光 6M | 蓝光 7M | 蓝光 8M |
| ---------- | --------- | --------- | ------- | ------- | ------- | ------- | ------- |
| 码率 | 1000/2000 | 2000/3000 | 4000 | 5000 | 6000 | 7000 | 8000 |

#### config.json

`config.json`的内容手动修改后需要重新启动本程序以生效

```
{
    "source": "flv",  // 直播源，有hls和flv两种，默认是flv
    "output": "mp4",  // 下载的直播视频的格式，必须是有效的视频格式后缀名
    "webPort": 51880, // web API的本地端口，使用web UI的话不能修改这个端口
    "directory": "",  // 直播视频和弹幕下载结束后会被移动到该文件夹，其值最好是绝对路径，会被live.json里的设置覆盖
    "acfun": {
        "cookies": "", // AcFun帐号的cookies，形式为`key=value`，多个key和value使用`;`分隔；可以在AcFun网页按`F12`将网络请求里的cookies直接复制到这里，注意网页的cookies只有30天的有效期；该值不为空时会忽略下面的`account`和`password`，目前只用于直播间挂机，不需要可以为空
        "account": "", // AcFun帐号邮箱或手机号，目前只用于直播间挂机，不需要可以为空
        "password": "" // AcFun帐号密码
    },
    "autoKeepOnline": true, // 是否自动在有守护徽章的直播间挂机，需要设置AcFun帐号和密码
    "mirai": {
        "adminQQ": 12345,        // 用来发送命令控制本程序的管理者QQ，可选
        "botQQ": 123456,         // QQ机器人的QQ号
        "botQQPassword": "abcde" // QQ机器人QQ号的密码
        "sendQQ": [              // 发送开播提醒和录播相关消息到数组里的所有QQ（需要QQ机器人添加这些QQ为好友），会被live.json里的设置覆盖
            12345,
            123456
        ],
        "sendQQGroup": [        // 发送开播提醒到数组里的所有QQ群（需要QQ机器人在这些QQ群里，最好是管理员，会@全体成员），会被live.json里的设置覆盖
            1234567
        ]
    }
}
```

### 使用方法

Windows 的 GUI 版本直接运行即可，程序会出现在系统托盘那里，可以通过`http://localhost:51890`访问 web UI 界面。

Windows 下如果要使用命令行模式，下载 CLI 版本，具体参数看 [cli.md](https://github.com/orzogc/acfunlive/blob/master/doc/cli.md) 。

本程序下载的直播视频和弹幕默认保存在本程序所在文件夹内，运行时可用参数`-record`指定下载录播和弹幕的文件夹。

命令行模式运行时可以输入命令控制本程序，运行时输入`help`查看具体命令，输入`quit`退出程序。

### web API

具体看 [webapi.md](https://github.com/orzogc/acfunlive/blob/master/doc/webapi.md)

### Mirai 使用方法

**本项目使用 [MiraiGo](https://github.com/Mrs4s/MiraiGo) 。**

命令行模式启动时加上`-mirai`参数，需要在`config.json`里的`mirai`对象设置机器人 QQ 号和密码。

如果由于设备锁无法登陆，请利用日志里的链接验证后重新启动本程序。

`config.json`里`mirai`对象的`adminQQ`为自己的 QQ 号时，添加 QQ 机器人为好友或者将 QQ 机器人加进 QQ 群后，可以发送命令给机器人控制本程序（在 QQ 群里需要@机器人的昵称），发送`help`查看具体命令。

如果实在无法登陆 QQ，修改配置文件所在文件夹里的`qqdevice.json`，将`protocol`改为 2 可以使用手机 QQ 扫码登陆。

如果在一台电脑/服务器能登陆 QQ，而另外一台电脑/服务器登陆失败，可以将登陆成功的`qqdevice.json`和`qqsession.token`复制到登陆失败的电脑/服务器配置文件所在文件夹里，再启动本程序试试。

### Docker

```
git clone --recursive https://github.com/orzogc/acfunlive.git
cd acfunlive
docker build -t acfunlive .
# configDir是配置文件所在文件夹，recordDir是录播和弹幕下载所在文件夹，-webui可以换成其他参数
docker run -i -v configDir:/acfunlive/config -v recordDir:/acfunlive/record -p 51880:51880 -p 51890:51890 -u `id -u`:`id -g` acfunlive:latest -webui
```
