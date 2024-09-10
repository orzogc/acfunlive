# 命令行使用方法
命令行模式需要运行`acfunlive -listen`，下载的视频和弹幕默认保存在本程序所在文件夹内

`acfunlive -listen` 运行程序监听，监听过程中可以输入命令修改设置（运行`help`查看详细命令说明）

`acfunlive -listen -config configDir -record recordDir` 运行程序监听，读取`configDir`里的配置文件，并将录播和弹幕文件保存在`recordDir`

`acfunlive -webui` 启动 web UI 服务器，可以通过`http://localhost:51890`访问 web UI 界面

`acfunlive -webapi` 运行监听程序并启动 web API 服务器，可以通过`http://localhost:51880`来查看状态和发送命令

`acfunlive -mirai` 利用 Mirai 发送直播通知到指定 QQ 或 QQ 群

`acfunlive -listlive` 列出正在直播的主播

`acfunlive -addnotify 23682490` 通知 uid 为 23682490 的主播的开播

`acfunlive -delnotify 23682490` 取消通知 uid 为 23682490 的主播的开播

`acfunlive -addrecord 23682490` uid 为 23682490 的主播直播时自动下载其直播视频

`acfunlive -delrecord 23682490` 取消自动下载 uid 为 23682490 的主播的直播视频

`acfunlive -adddanmu 23682490` uid 为 23682490 的主播直播时自动下载其直播弹幕

`acfunlive -deldanmu 23682490` 取消自动下载 uid 为 23682490 的主播的直播弹幕

`acfunlive -getdlurl 23682490` 查看 uid 为 23682490 的主播是否在直播，输出其直播源

`acfunlive -startrecord 23682490` 临时下载 uid 为 23682490 的主播的直播视频

`acfunlive -startdanmu 23682490` 临时下载 uid 为 23682490 的主播的直播弹幕

`acfunlive -startrecdan 23682490` 临时下载 uid 为 23682490 的主播的直播视频和弹幕

运行`acfunlive -h`查看详细设置说明
