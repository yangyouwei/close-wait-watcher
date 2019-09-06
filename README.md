使用

    COMMAND -conf /conf/dir/watcher.conf

不指定配置文件，默认寻找当前目录下的watcher.conf 文件

配置文件说明

    [main]
    //执行命令的时间间隔
    interval = 5
    //colose_wait 阈值
    close_wait_max = 0
    //重启服务命令，重启服务的输出如果有标准或错误输出会在日志输出。
    service_command = service restart ss
    //service 监听端口。判断服务是否运行。端口不存在则重启服务
    service_port = 88
    //日志开关
    log = true
    //日志位置
    log_file = ./watcher.log
