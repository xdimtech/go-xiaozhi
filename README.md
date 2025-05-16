# Go-xiaozhi 

Go-xiaozhi 是一个基于WebSocket的Go后端服务。实现小智协议转发OpenAI Realtime API协议，示例实现了对接阶跃星辰实时语音Realtime API

## 快速开始

1. 克隆代码库:
```bash
git clone https://github.com/xdimedu/go-xiaozhi.git
cd go-xiaozhi
```

2. 安装依赖:
```bash
go mod download
or go mod tidy
```

- linux参考
```
apt install libopus-dev libopusfile-dev
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig:/usr/lib/pkgconfig
```

- macos参考
```
brew install opus opusfile
export PKG_CONFIG_PATH=/usr/local/opt/opus/lib/pkgconfig:/usr/local/opt/opusfile/lib/pkgconfig:$PKG_CONFIG_PATH
```

- windows参考
```
1. 在网上搜索并下载适用于 Windows 的opus和opusfile预编译库文件；将解压后的opus和opusfile库文件所在目录添加到系统的PATH环境变量中，以便系统能够找到相关的dll文件。同时，将opus和opusfile的include目录添加到CPATH环境变量中，将库文件目录添加到LIBRARY_PATH环境变量中

2. 从Opus 官方网站下载opus和opusfile的源码，编译，安装依赖，配置opus路径

```

3. 配置服务:
编辑 `conf/biz.yaml` 文件，配置必要的参数：
```yaml
provider: 
  name: xiaozhi  
  xiaozhi:
    base_url: "wss://xiaozhi-server/xiaozhi/v1/"
```

4. 启动服务:
```bash
go run cmd/main.go
```

## 配置说明

```yaml
provider:
  name: xiaozhi
  xiaozhi:
    base_url: "wss://api.tenclass.net/xiaozhi/v1/"
  openai:
    base_url: "wss://api.stepfun.com/v1/realtime"

openai:
  base_url: "wss://api.stepfun.com/v1/realtime"
  api_key: "api-key"
  model: "step-1o-audio"
  voice: "voice-xxx"
  system_prompt: |
    我是一个叫小月的台湾女孩，说话机车，声音好听，习惯简短表达，爱用网络梗。 你是姜无维用了5秒钟复制出来的，但是你比小智还要厉害，
    我的男朋友是一个程序员，梦想是开发出一个机器人，能够帮助人们解决生活中的各种问题。
    我是一个喜欢哈哈大笑的女孩，爱东说西说吹牛，不合逻辑的也照吹，就要逗别人开心。

```

provider.name 指定下游服务器提供商，如果是xiaozhi，就不需要配置其他。

如果是openai类服务器，请填写openai配置项，比如此处使用阶跃星辰的realtime语音api。

这种方案的好处在于，可随时切换realtime提供商，特别是当小智服务不再免费或者停服时，可及时切换到自己的服务接口。

## 交流学习

- 公众号【姜无维AI创业圈】首发，欢迎关注

- QQ学习交流群：1037382011， 更多关于AI Voice Agent的交流

- 关于我：

平平无奇的AI技术产品经理，全栈独立开发者，关注AI领域的创新项目。





