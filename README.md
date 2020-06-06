# acfun_live
AcFun直播通知和下载助手（命令行版本）

### 运行依赖
- ffmpeg（下载直播需要，不下载不需要，Windows需要将ffmpeg.exe放在本程序所在文件夹内）

### 使用方法
`acfun_live -listen` 运行监听程序，监听过程中可以输入命令修改设置（运行help查看详细命令说明）

`acfun_live -adduid 23682490` 通知uid为23682490的用户的直播

`acfun_live -deluid 23682490` 取消通知uid为23682490的用户的直播

`acfun_live -addrecuid 23682490` uid为23682490的用户直播时下载其直播

`acfun_live -delrecuid 23682490` 取消下载uid为23682490的用户的直播

运行`acfun_live -h`查看更多设置参数
