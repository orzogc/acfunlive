# acfunlive web API
web API默认本地端口为51880

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

`http://localhost:51880/delconfig/23682490` 删除uid为23682490的主播的所有设置

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

`http://localhost:51880/quit` 退出acfunlive运行

`http://localhost:51880/help` 显示帮助信息
