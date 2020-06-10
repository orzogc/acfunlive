# acfunlive
AcFun直播通知和下载助手（命令行版本）

### 运行依赖
- ffmpeg（下载直播需要，不下载不需要，Windows需要将ffmpeg.exe放在本程序所在文件夹内）

### 使用方法
`acfunlive -listen` 运行监听程序，监听过程中可以输入命令修改设置（运行`help`查看详细命令说明）

`acfunlive -adduid 23682490` 通知uid为23682490的用户的直播

`acfunlive -deluid 23682490` 取消通知uid为23682490的用户的直播

`acfunlive -addrecuid 23682490` uid为23682490的用户直播时下载其直播

`acfunlive -delrecuid 23682490` 取消下载uid为23682490的用户的直播

`acfunlive -getdlurl 23682490` 查看uid为23682490的用户是否在直播，获取其直播源

运行`acfunlive -h`查看更多设置参数
