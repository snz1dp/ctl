# Java后端

TODO: 请补充说明

## 开发调试

> 首次使用请运行`snz1dpctl profile login {{ .ServerURL }}`命令登录开发平台。

执行以下命令准备好依赖环境：

```bash
make develop
```

> `windows`下或无安装`make`命令的机器上执行：

```cmd
snz1dpctl make standalone develop
```

然后在`IDE`中启动`{{ .Package }}.Application`类进行调试。

> 命令行运行服务程序：

```bash
snz1dpctl make run
```

> 命令行运行单元测试:

```bash
snz1dpctl make test
```

> 执行以下命令打包发布：

```bash
snz1dpctl make publish
```
