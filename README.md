# Jiwei Logagent

用于将各个服务的日志聚合到日志处理系统的客户端


## 安装与使用

1. 新建一个目录，将可执行文件添加到目录中(以 下面以 `/var/logagent/` 为例)。
2. 在目录下添加log目录和配置文件  `logagent.conf`, 确认配置文件的信息。
3. 对可执行文件添加权限 `chmod +x logagent`。
4. 启动 `nohup /var/logagent/logagent >> /var/logagent/log/agent.log 2>&1 &`
5. 观察 `./log/agent.log` 查看是否正常运行。
6. 注意目录权限

## 基础配置
```conf
[app]
logagent_id=节点名(需要先去 `ccenter` 注册)

# Kafka 配置
[kafka]
address=localhost:9091(kafka队列配置)
queue_size=1000(队列数量, 预留)

# Etcd 配置
[etcd]
address=localhost:23790 (ETCD Address)
```

## 功能
1. 两种格式的日志数据收集方式
2. 引入agent的上下文
3. 监控etcd中的配置，不重启服务就可以变更配置
4. Collector中错误的熔断处理，不影响其他的Collector
5. 守护进程的退出机制
6. 通过offset的重新加载实现断点续传
7. 消息合并发送

## TODO
1. 应使用更加完善的日志记录机制，目前只能输出到STDOUT
2. Agent上下文不应该放在结构体中,下一个版本重构
