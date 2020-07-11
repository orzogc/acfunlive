# acfunlive
AcFun直播桌面通知和下载助手（命令行版本）

* [acfunlive](#acfunlive)
    * [运行依赖](#运行依赖)
    * [编译安装](#编译安装)
    * [配置文件详解](#配置文件详解)
      * [live\.json](#livejson)
      * [config\.json](#configjson)
    * [命令行使用方法](#命令行使用方法)
    * [web服务使用方法](#web服务使用方法)
    * [酷Q使用方法](#酷Q使用方法)

### 运行依赖
* ffmpeg（下载直播视频需要，不下载不需要，Windows需要将ffmpeg.exe放在本程序所在文件夹内）

### 编译安装
`go get -u github.com/orzogc/acfunlive`

### 配置文件详解
#### live.json
live.json的内容可以手动修改，本程序会自动读取更改后的设置，无需重新启动本程序
```
{
    "UID": 23682490,    // 主播的uid
    "Name": "AC娘本体", // 主播的昵称
    "Notify": {
            "NotifyOn": true,      // 主播开播通知
            "NotifyOff": false,    // 主播下播通知（需自行修改设置）
            "NotifyRecord": true,  // 下载主播直播相关的通知
            "NotifyDanmu": false   // 下载主播直播弹幕相关的通知（需自行修改设置）
        },
    "Record": true,       // 是否下载直播视频
    "Danmu": true,        // 是否下载直播弹幕
    "SendQQ": 12345,      // 发送开播提醒到该QQ（需要QQ机器人添加该QQ为好友）
    "SendQQGroup": 123456 // 发送开播提醒到该QQ群（需要QQ机器人在该群，最好是管理员）
}
```
#### config.json
config.json的内容手动修改后需要重新启动本程序以生效
```
{
    "Source": "flv",  // 直播源，有hls和flv两种，默认是flv
    "Output": "mp4",  // 下载的直播视频的格式，必须是有效的视频格式后缀名
    "WebPort": 51880, // web服务的本地端口
    "Coolq": {
        "CqhttpWSAddr": "ws://localhost:6700", // CQHTTP的WebSocket地址和端口
        "AdminQQ": 12345,                      // 用来发送命令控制本程序的管理者QQ
        "AccessToken": "",                     // CQHTTP的access_token，可以为空
        "Secret": ""                           // CQHTTP的secret，可以为空
    }
}
```

### 命令行使用方法
桌面通知和自动下载直播需要运行`acfunlive -listen`，下载的视频和弹幕默认保存在本程序所在文件夹内

`acfunlive -listen` 运行监听程序，监听过程中可以输入命令修改设置（运行`help`查看详细命令说明）

`acfunlive -listen -web` 运行监听程序并启动web服务，可以通过`http://localhost:51880`来查看状态和发送命令

`acfunlive -listen -coolq` 使用酷Q发送直播通知到指定QQ或QQ群，需要事先设置并启动酷Q

`acfunlive -listlive` 列出正在直播的主播

`acfunlive -addnotify 23682490` 通知uid为23682490的主播的直播

`acfunlive -delnotify 23682490` 取消通知uid为23682490的主播的直播

`acfunlive -addrecord 23682490` uid为23682490的主播直播时自动下载其直播视频

`acfunlive -delrecord 23682490` 取消自动下载uid为23682490的主播的直播视频

`acfunlive -adddanmu 23682490` uid为23682490的主播直播时自动下载其直播弹幕

`acfunlive -deldanmu 23682490` 取消自动下载uid为23682490的主播的直播弹幕

`acfunlive -getdlurl 23682490` 查看uid为23682490的主播是否在直播，输出其直播源

`acfunlive -startrecord 23682490` 临时下载uid为23682490的主播的直播视频

`acfunlive -startdanmu 23682490` 临时下载uid为23682490的主播的直播弹幕

`acfunlive -startrecdan 23682490` 临时下载uid为23682490的主播的直播视频和弹幕

运行`acfunlive -h`查看详细设置说明

### web服务使用方法
web服务默认本地端口为51880

`http://localhost:51880/listlive` 列出正在直播的主播

`http://localhost:51880/listrecord` 列出正在下载的直播视频

`http://localhost:51880/listdanmu` 列出正在下载的直播弹幕

`http://localhost:51880/liststreamer` 列出设置了开播提醒或自动下载直播的主播

`http://localhost:51880/startcoolq` 使用酷Q发送直播通知到指定QQ或QQ群，需要事先设置并启动酷Q

`http://localhost:51880/addnotify/23682490` 通知uid为23682490的主播的直播

`http://localhost:51880/delnotify/23682490` 取消通知uid为23682490的主播的直播

`http://localhost:51880/addrecord/23682490` uid为23682490的主播直播时自动下载其直播视频

`http://localhost:51880/delrecord/23682490` 取消自动下载uid为23682490的主播的直播视频

`http://localhost:51880/adddanmu/23682490` uid为23682490的主播直播时自动下载其直播弹幕

`http://localhost:51880/deldanmu/23682490` 取消自动下载uid为23682490的主播的直播弹幕

`http://localhost:51880/getdlurl/23682490` 查看uid为23682490的主播是否在直播，并输出其直播源

`http://localhost:51880/addqq/23682490/12345` 将uid为23682490的主播的开播提醒发送到QQ12345，需要QQ机器人已经添加其为好友

`http://localhost:51880/delqq/23682490` 取消将uid为23682490的主播的开播提醒发送到QQ

`http://localhost:51880/addqqgroup/23682490/12345` 将uid为23682490的主播的开播提醒发送到QQ群12345，需要QQ机器人已经加入该群

`http://localhost:51880/delqqgroup/23682490` 取消将uid为23682490的主播的开播提醒发送到QQ群

`http://localhost:51880/startrecord/23682490` 临时下载uid为23682490的主播的直播视频

`http://localhost:51880/stoprecord/23682490` 取消下载uid为23682490的主播的直播视频

`http://localhost:51880/startdanmu/23682490` 临时下载uid为23682490的主播的直播弹幕

`http://localhost:51880/stopdanmu/23682490` 取消下载uid为23682490的主播的直播弹幕

`http://localhost:51880/startrecdan/23682490` 临时下载uid为23682490的主播的直播视频和弹幕

`http://localhost:51880/stoprecdan/23682490` 取消下载uid为23682490的主播的直播视频和弹幕

`http://localhost:51880/log` 查看log

`http://localhost:51880/quit` 退出本程序

`http://localhost:51880/help` 显示帮助信息

### 酷Q使用方法
本程序使用 [酷Q](https://cqp.cc/) 和 [CQHTTP](https://github.com/richardchien/coolq-http-api) 作为WebSocket服务端来发送QQ消息，请事先设置好酷Q和CQHTTP插件并启动酷Q，具体可以看 [CQHTTP的文档](https://richardchien.gitee.io/coolq-http-api/docs/) 。

CQHTTP插件必须启用WebSocket服务端，也就是其配置里的use_ws必须为true。

本程序酷Q相关配置参考 [config\.json配置](#configjson) 。

目前群通知@全体成员 需要酷Q Pro。

config.json里的AdminQQ为自己的QQ号时，添加QQ机器人为好友或者将QQ机器人加进QQ群后，可以发送命令给机器人控制本程序（在QQ群里需要@机器人的昵称），发送help查看具体命令。
