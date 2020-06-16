# acfunlive
AcFun直播桌面通知和下载助手（命令行版本）

### 运行依赖
- ffmpeg（下载直播需要，不下载不需要，Windows需要将ffmpeg.exe放在本程序所在文件夹内）

### 使用方法
桌面通知和自动下载直播需要运行`acfunlive -listen`

`acfunlive -listen` 运行监听程序，监听过程中可以输入命令修改设置（运行`help`查看详细命令说明）

`acfunlive -weblisten` 运行监听程序并启动web服务，可以通过`http://localhost:51880`来查看状态和发送命令

`acfunlive -listlive` 列出正在直播的主播

`acfunlive -addnotify 23682490` 通知uid为23682490的用户的直播

`acfunlive -delnotify 23682490` 取消通知uid为23682490的用户的直播

`acfunlive -addrecord 23682490` uid为23682490的用户直播时自动下载其直播

`acfunlive -delrecord 23682490` 取消自动下载uid为23682490的用户的直播

`acfunlive -getdlurl 23682490` 查看uid为23682490的用户是否在直播，输出其直播源

`acfunlive -startrecord 23682490` 临时下载uid为23682490的用户的直播

运行`acfunlive -h`查看详细设置说明

### web服务使用方法
web服务默认本地端口为51880

`http://localhost:51880/listlive` 列出正在直播的主播

`http://localhost:51880/listrecord` 列出正在下载的直播

`http://localhost:51880/liststreamer` 列出设置了开播提醒或自动下载直播的主播

`http://localhost:51880/addnotify/23682490` 通知uid为23682490的用户的直播

`http://localhost:51880/delnotify/23682490` 取消通知uid为23682490的用户的直播

`http://localhost:51880/addrecord/23682490` uid为23682490的用户直播时自动下载其直播

`http://localhost:51880/delrecord/23682490` 取消自动下载uid为23682490的用户的直播

`http://localhost:51880/getdlurl/23682490` 查看uid为23682490的用户是否在直播，输出其直播源

`http://localhost:51880/startrecord/23682490` 临时下载uid为23682490的用户的直播

`http://localhost:51880/stoprecord/23682490` 取消下载uid为23682490的用户的直播

`http://localhost:51880/log` 查看log

`http://localhost:51880/quit` 退出本程序

`http://localhost:51880/help` 显示帮助信息
