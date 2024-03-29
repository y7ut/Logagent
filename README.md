# Jiwei Logagent

用于将各个服务的日志聚合到日志处理系统的客户端

## Change Log

### 0.1.0

1. 两种格式的日志数据收集方式
2. 热重载
3. 错误的熔断处理
4. 优雅的退出机制
5. 断点续传
6. 消息合并发送

### 1.0.0

1. 新增程序的启动项，可以自定义配置文件位置
2. 支持 `--version` 查询软件版本。

### 1.0.1

1. 修复对时间类型日志文件格式的支持

### 1.0.2

1. 可以设置`Kafka Queue`的长度啦

### 2.0.0

1. 更新了对topic模块的支持

### 2.1.0

1. 修改节点发现逻辑

### 2.1.1

1. 优化退出的信号监听

### 2.2.0

1. 调整程序执行时协程的生命周期

### 3.0.0

1. 添加cobra的壳, 修复一些问题

## 安装与使用

1. go build -o bifrost
2. ./bifrost

## 基础配置

```conf
[app]
logagent_id=节点名(需要先去 `ccenter` 注册)

# Kafka 配置
[kafka]
address=localhost:9091(kafka队列配置)
queue_size=1000

# Etcd 配置
[etcd]
address=localhost:23790 (ETCD Address)
```
