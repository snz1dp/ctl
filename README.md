# Snz1DPCtl

> 这里懒得啰嗦，具体看这👉：<https://docs.dingtalk.com/i/nodes/1DKw2zgV2vkxoOzgf1rDyzwwVB5r9YAn>

## 1、编译指南

### 1.1、环境要求

- go 1.25+
- go rice
- automake
- git

### 1.2、配置GoProxy

因为众所周知周知的原因需要配置代理才能愉快的使用go模块，请在`~/.bash_profile`文件中加入以下配置：

```bash
#=============================Go代理配置==============================#
# 启用 Go Modules 功能
export GO111MODULE=on
# 配置 GOPROXY 环境变量
export GOPROXY=https://goproxy.io
```

### 1.2、下载依赖

使用以下命令一键下载依赖：

```bash
make depends
```

***1. 下载rice工具***

```bash
go get github.com/GeertJohan/go.rice/rice
```

***2. 初始化子模块***

```bash
git submodule init
git submodule update --init --recursive
```

***3. 下载依赖库***

```bash
go get
```

### 1.3、编译目标

```bash
make build
```

执行上述命令后会生成三个平台的目标文件：

- snz1dpctl-linux-amd64 Linux
- snz1dpctl-darwin-amd64 MacOS
- snz1dpctl-windows-amd64.exe Windows

> 目前目标输出都为64位版本。

## 2、准备Node

### 2.1、Linux amd64

```bash
docker pull --platform linux/amd64 bitnami/git:2.41.0-debian-11-r11
docker run --platform linux/amd64 --rm -ti -v ./files/nodejs:/nodejs bitnami/git:2.41.0-debian-11-r11 bash
```

```docker
apt-get update
apt-get install -y wget
cd /nodejs
wget https://nodejs.org/dist/v16.20.2/node-v16.20.2-linux-x64.tar.gz
tar xzvf node-v16.20.2-linux-x64.tar.gz && rm -rf node-v16.20.2-linux-x64.tar.gz
mv node-v16.20.2-linux-x64 node-v16.20.2-linux-amd64
cd node-v16.20.2-linux-amd64
export PATH=$PATH:$PWD/bin
./bin/npm install -g nrm
cd ..
tar zcvf node-v16.20.2-linux-amd64.tgz node-v16.20.2-linux-amd64/
rm -rf node-v16.20.2-linux-amd64/
```

### 2.2、Linux arm64

```bash
docker pull --platform linux/arm64 bitnami/git:2.41.0-debian-11-r11
docker run --platform linux/arm64 --rm -ti -v ./files/nodejs:/nodejs bitnami/git:2.41.0-debian-11-r11 bash
```

```docker
apt-get update
apt-get install -y wget
cd /nodejs
wget https://nodejs.org/dist/v16.20.2/node-v16.20.2-linux-arm64.tar.gz
tar xzvf node-v16.20.2-linux-arm64.tar.gz && rm -rf node-v16.20.2-linux-arm64.tar.gz
cd node-v16.20.2-linux-arm64
export PATH=$PATH:$PWD/bin
./bin/npm install -g nrm
cd ..
tar zcvf node-v16.20.2-linux-arm64.tgz node-v16.20.2-linux-arm64/
rm -rf node-v16.20.2-linux-arm64/
```

### 2.3、Windows amd64

### 2.4、MacOS amd64

### 2.4、MacOS arm64

## 3、参考资料

1. <https://pkg.go.dev/>
