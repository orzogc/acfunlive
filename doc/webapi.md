# acfunlive web API

web API 默认本地端口为 51880

`http://localhost:51880/listlive` 列出正在直播的主播

`http://localhost:51880/listrecord` 列出正在下载的直播视频

`http://localhost:51880/listdanmu` 列出正在下载的直播弹幕

`http://localhost:51880/liststreamer` 列出设置了开播提醒或自动下载直播的主播

`http://localhost:51880/startmirai` 利用 Mirai 发送直播通知到指定 QQ 或 QQ 群

`http://localhost:51880/addnotifyon/23682490` 通知 uid 为 23682490 的主播的开播

`http://localhost:51880/delnotifyon/23682490` 取消通知 uid 为 23682490 的主播的开播

`http://localhost:51880/addnotifyoff/23682490` 通知 uid 为 23682490 的主播的下播

`http://localhost:51880/delnotifyoff/23682490` 取消通知 uid 为 23682490 的主播的下播

`http://localhost:51880/addnotifyrecord/23682490` 通知 uid 为 23682490 的主播的直播视频下载

`http://localhost:51880/delnotifyrecord/23682490` 取消通知 uid 为 23682490 的主播的直播视频下载

`http://localhost:51880/addnotifydanmu/23682490` 通知 uid 为 23682490 的主播的直播弹幕下载

`http://localhost:51880/delnotifydanmu/23682490` 取消通知 uid 为 23682490 的主播的直播弹幕下载

`http://localhost:51880/addrecord/23682490` uid 为 23682490 的主播直播时自动下载其直播视频

`http://localhost:51880/delrecord/23682490` 取消自动下载 uid 为 23682490 的主播的直播视频

`http://localhost:51880/adddanmu/23682490` uid 为 23682490 的主播直播时自动下载其直播弹幕

`http://localhost:51880/deldanmu/23682490` 取消自动下载 uid 为 23682490 的主播的直播弹幕

`http://localhost:51880/addkeeponline/23682490` uid 为 23682490 的主播直播时在其直播间里挂机

`http://localhost:51880/delkeeponline/23682490` 取消设置在 uid 为 23682490 的主播直播时在其直播间里挂机

`http://localhost:51880/delconfig/23682490` 删除 uid 为 23682490 的主播的所有设置

`http://localhost:51880/getdlurl/23682490` 查看 uid 为 23682490 的主播是否在直播，并输出其直播源

`http://localhost:51880/addqq/23682490/12345` 将 uid 为 23682490 的主播的开播提醒发送到 QQ12345，需要 QQ 机器人已经添加该 QQ 为好友

`http://localhost:51880/delqq/23682490/12345` 取消将 uid 为 23682490 的主播的开播提醒发送到 QQ12345

`http://localhost:51880/cancelqq/23682490` 取消将 uid 为 23682490 的主播的开播提醒发送到任何 QQ

`http://localhost:51880/addqqgroup/23682490/12345` 将 uid 为 23682490 的主播的开播提醒发送到 QQ 群 12345，需要 QQ 机器人已经加入该 QQ 群，最好是管理员，会@全体成员

`http://localhost:51880/delqqgroup/23682490/12345` 取消将 uid 为 23682490 的主播的开播提醒发送到 QQ 群 12345

`http://localhost:51880/cancelqqgroup/23682490` 取消将 uid 为 23682490 的主播的开播提醒发送到任何 QQ 群

`http://localhost:51880/startrecord/23682490` 临时下载 uid 为 23682490 的主播的直播视频

`http://localhost:51880/stoprecord/23682490` 取消下载 uid 为 23682490 的主播的直播视频

`http://localhost:51880/startdanmu/23682490` 临时下载 uid 为 23682490 的主播的直播弹幕

`http://localhost:51880/stopdanmu/23682490` 取消下载 uid 为 23682490 的主播的直播弹幕

`http://localhost:51880/startrecdan/23682490` 临时下载 uid 为 23682490 的主播的直播视频和弹幕

`http://localhost:51880/stoprecdan/23682490` 取消下载 uid 为 23682490 的主播的直播视频和弹幕

`http://localhost:51880/log` 查看 log

`http://localhost:51880/quit` 退出 acfunlive 运行

`http://localhost:51880/help` 显示帮助信息
