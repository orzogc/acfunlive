# 命令行使用方法
命令行模式需要运行`acfunlive -nogui -listen`，下载的视频和弹幕默认保存在本程序所在文件夹内

`acfunlive -nogui -webapi -webui` 启动web UI服务器，可以通过`http://localhost:51890`访问web UI界面

`acfunlive -nogui -listen` 运行监听程序，监听过程中可以输入命令修改设置（运行`help`查看详细命令说明）

`acfunlive -nogui -listen -webapi` 运行监听程序并启动web API服务器，可以通过`http://localhost:51880`来查看状态和发送命令

`acfunlive -nogui -listen -mirai` 利用Mirai发送直播通知到指定QQ或QQ群

`acfunlive -nogui -listen -coolq` 使用酷Q发送直播通知到指定QQ或QQ群，需要事先设置并启动酷Q

`acfunlive -nogui -listlive` 列出正在直播的主播

`acfunlive -nogui -addnotify 23682490` 通知uid为23682490的主播的直播

`acfunlive -nogui -delnotify 23682490` 取消通知uid为23682490的主播的直播

`acfunlive -nogui -addrecord 23682490` uid为23682490的主播直播时自动下载其直播视频

`acfunlive -nogui -delrecord 23682490` 取消自动下载uid为23682490的主播的直播视频

`acfunlive -nogui -adddanmu 23682490` uid为23682490的主播直播时自动下载其直播弹幕

`acfunlive -nogui -deldanmu 23682490` 取消自动下载uid为23682490的主播的直播弹幕

`acfunlive -nogui -getdlurl 23682490` 查看uid为23682490的主播是否在直播，输出其直播源

`acfunlive -nogui -startrecord 23682490` 临时下载uid为23682490的主播的直播视频

`acfunlive -nogui -startdanmu 23682490` 临时下载uid为23682490的主播的直播弹幕

`acfunlive -nogui -startrecdan 23682490` 临时下载uid为23682490的主播的直播视频和弹幕

运行`acfunlive -h`查看详细设置说明
