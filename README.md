# go_sign 小红书签名服务

## 项目简介

本项目基于 Go + Playwright 实现小红书签名 HTTP 服务，支持自动注入 stealth.min.js，提供 /sign 接口用于签名。

## 主要依赖
- [gin](https://github.com/gin-gonic/gin)：高性能 HTTP Web 框架
- [playwright-go](https://github.com/mxschmitt/playwright-go)：浏览器自动化

## 目录结构
```
internal/xhs/sign.go   # 核心签名逻辑
internal/xhs/http.go   # HTTP 路由注册
main.go                # 程序入口
```

## 配置说明
- `stealth.min.js` 路径通过 --stealth 参数指定，默认为当前目录下。
- HTTP 监听地址通过 --addr 参数指定，默认为 :5005。

## 启动方法
```sh
go mod tidy
go run main.go --stealth=/path/to/stealth.min.js --addr=:5005
```

## API 示例
POST /sign
```
{
  "uri": "/api/some/path",
  "data": {"key": "value"},
  "a1": "xxx",
  "web_session": "yyy"
}
```
返回：
```
{
  "x-s": "...",
  "x-t": "..."
}
```

## 注意事项
- 需提前下载好 stealth.min.js 并指定路径。
- 生产环境请注意安全与资源管理。 