domeos/agent
===

更新说明：

2016.06.03

镜像地址：pub.domeos.org/domeos/agent:2.5

1 升级cadvisor，以支持新版docker，实测docker1.11.1下可正常上报数据

2 容器运行agent时，修正dockerConnector导致宿主机docker exec进程残留问题

3 基础镜像修改为docker:1.8，并修正时区，减少了容器启动的挂载目录

4 重新编译dockerConnector与falcon-agent，去掉cgo依赖(对cadvisor和prometheus源码稍作了修改，不影响agent使用)

5 修正容器内存limit为0时内存使用率NaN问题

6 修正连接多个transfer时crash问题

## Notice

domeos/agent模块是以open-falcon原生agent模块为基础，为适应DomeOS监控报警需求而设计修改的，包名已修改为github.com/domeos/agent，与原生open-falcon的agent主要区别为：

- 在配置模板中对很大部分监控项设置了忽略，仅保留一些系统基础监控项
- 嵌入了cadvisor代码，实现容器监控，其tag为id={容器64位id}
- 镜像中集成了DomeOS WebSSH客户端，默认监听2222端口上的SSH请求，配合WebSSH Server实现Web端容器登录

domeos/agent模块需在集群中所有node节点部署。若使用DomeOS给定的脚本添加主机，且指定--start-agent参数为true，默认将自动启动agent容器；否则需要手动启动agent容器。

## Installation

这一安装方式将仅安装监控agent组件。

```bash
# set $GOPATH and $GOROOT
mkdir -p $GOPATH/src/github.com/domeos
cd $GOPATH/src/github.com/domeos
git clone https://github.com/domeos/agent.git
cd agent
go get ./...
./control build
./control start

# goto http://localhost:1988
```

## Configuration

- heartbeat: 心跳服务器地址
- transfer: transfer地址，可以配置多个
- ignore: 忽略(不上报)的监控项

## Run In Docker Container

注意这里已经将dockerConnector集成进agent镜像中，可以通过DomeOS WebSSH Server登录进本机容器内部。

首先构建domeos/agent镜像(tag以latest为例)：

```bash
sudo docker build -t="domeos/agent:latest" ./docker/
```

启动docker容器：(注意要挂出系统和docker相应目录，以在容器内部获取宿主机监控信息以及获取容器监控)

```bash
sudo docker run -d --restart=always \
	-p 2222:2222 \
	-p <_agent_http_port>:1988 \
	-e HOSTNAME="\"<_hostname>\"" \
	-e TRANSFER_ADDR="[<_transfer_addr>]" \
	-e TRANSFER_INTERVAL="<_interval>" \
	-e HEARTBEAT_ENABLED="true" \
	-e HEARTBEAT_ADDR="\"<_heartbeat_addr>\"" \
	-v /:/rootfs:ro \
	-v /var/run:/var/run:rw \
	-v /sys:/sys:ro \
	-v <_docker_graph_path>:<_docker_graph_path>:ro \
	--name agent \
	domeos/agent:latest
```

参数说明：

- _agent_http_port: agent服务http端口，主要用于状态检测、调试等。
- _hostname: 监控系统中主机的endpoint名，需与kubernetes中添加node时配置的hostname相同。DomeOS添加主机脚本中使用主机运行hostname命令的执行结果。
- _transfer_addr: transfer的rpc地址，可以配置多个。注意每个IP:Port需加双引号，多个IP:Port之间用逗号分隔。DomeOS添加主机脚本中使用DomeOS全局配置中的transfer配置。
- _interval: 监控数据上报时间间隔，单位为秒(s)。DomeOS中支持的最小上报时间间隔为10s。DomeOS添加主机脚本中默认设置为10s。
- _heartbeat_addr: heartbeat server的rpc地址，IP:Port形式。DomeOS添加主机脚本中使用DomeOS全局配置中的hbs配置。
- _docker_graph_path: docker启动时配置的graph目录路径(-g参数)。若没有配置，默认为/var/lib/docker目录。

样例：

```bash
sudo docker run -d --restart=always \
	-p 2222:2222 \
	-p 1988:1988 \
	-e HOSTNAME="\"bx-42-198\"" \
	-e TRANSFER_ADDR="[\"10.16.42.198:8433\",\"10.16.42.199:8433\"]" \
	-e TRANSFER_INTERVAL="10" \
	-e HEARTBEAT_ENABLED="true" \
	-e HEARTBEAT_ADDR="\"10.16.42.199:6030\"" \
	-v /:/rootfs:ro \
	-v /var/run:/var/run:rw \
	-v /sys:/sys:ro \
	-v /var/lib/docker:/var/lib/docker:ro \
	--name agent \
	domeos/agent:latest
```

验证：

通过curl -s localhost:<_agent_http_port>/health命令查看运行状态，若运行正常将返回ok。


DomeOS仓库中domeos/agent对应版本： pub.domeos.org/domeos/agent:2.5