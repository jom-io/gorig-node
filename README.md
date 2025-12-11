# gorig-node

gorig-node is a lightweight Go runtime (part of the [gorig](https://github.com/jom-io/gorig) ecosystem) that registers local business functions to a gorig hub and exposes an optional HTTP ingress for direct invocation and debugging. It builds API schemas from your function signatures, reports them to the hub with heartbeats, and lets you poke the handlers over HTTP without a generated SDK.

## Features
- Reflective API schema generation (args, returns, JSON tags) for hub reporting and SDK generation
- Automatic service registration plus periodic heartbeats to the configured gorig hub
- Drop-in hook for gorig projects: call `gnnode.RegServer()` and let gorig boot via `bootstrap.StartUp()`
- HTTP ingress (`/{service}/{method}`) for manual tests and local integration checks
- Sample services under `_cmd` for standalone demo; real projects embed the hook inside an existing gorig service
- Minimal configuration via `gncfg` or `gn.*` config keys (`gn.hub.addr`, `gn.node.addr`)

## Quick start
1. Prerequisite: Go 1.23+.
2. Configure hub/node addresses. You can either rely on the `gorig` config system (`gn.hub.addr`, optional `gn.node.addr`), or set them programmatically:
   ```go
   import "gorig-node/gncfg"

   func init() {
       gncfg.UseConfig(gncfg.GlobalConfig{
           HubAddr:  "127.0.0.1:5806", // required
           NodeAddr: "127.0.0.1:5807", // optional, auto-detected if empty
       })
   }
   ```
   If `NodeAddr` is empty, the node auto-detects a private IPv4 and appends the default port `:5807`.
3. Wire into your gorig service. In a gorig project (see [gorig repo](https://github.com/jom-io/gorig)), call `gnnode.RegServer()` once; gorig will invoke `bootstrap.StartUp()` as part of its lifecycle. For a standalone demo, `_cmd/main.go` shows the pattern:
   ```go
   package main

   import (
       "github.com/jom-io/gorig/bootstrap"
       "gorig-node/gnnode"
       _ "gorig-node/gnnode" // optional side-effect import if needed
   )

   func main() {
       gnnode.RegServer()
       bootstrap.StartUp() // provided by gorig
   }
   ```
4. Register your services (examples from `_cmd/sample.go`):
   ```go
   register.Server("UserSample").
       Env("dev").
       Version("v1.0.0").
       RegName("Login", func(ctx context.Context, req loginReq) (loginResp, error) {
           return loginResp{
               Token:   "token-for-" + req.Username,
               Profile: profileReq{UID: req.Username + "_uid"},
           }, nil
       }).
       Create()
   ```
   - `Server(name)` groups related APIs.
   - `RegName(method, fn, ...)` inspects the handler signature, validates it (ctx optional as first arg, error optional as last return), and records schemas.
   - For the shortest form, let the method name auto-infer from the function name:
     ```go
     _ = register.Server("UserSample").
         Reg(func(ctx context.Context, req loginReq) (loginResp, error) {
             return loginResp{
                 Token:   "token-for-" + req.Username,
                 Profile: profileReq{UID: req.Username + "_uid"},
             }, nil
         }).
         Create()
     ```
   - `Create()` finalizes registration for that service.
5. Run locally:
   ```bash
   go run ./_cmd
   ```
   Startup registers all created services to the hub and begins heartbeats; HTTP ingress listens on the node port (default `:5807`).

## HTTP ingress (for debugging)
Each registered method is available at `POST /{service}/{method}` on the node. The body wraps arguments by index:
```json
{
  "args": {
    "arg0": {
      "username": "alice",
      "password": "secret"
    }
  }
}
```
Response shape:
```json
{
  "resp": {
    "resp0": {
      "token": "token-for-alice",
      "profile": { "uid": "alice_uid" }
    }
  },
  "error": ""
}
```
Business errors are returned as the `error` string; HTTP status remains 200 for successful dispatch.

## Development
- Run tests: `go test ./...` (the register flow test needs a private IPv4 to simulate hub callbacks).
- Services send heartbeats every minute; call `register.Stop()` in your own shutdown hooks if you embed the library elsewhere.

## 中文简介
gorig-node 是一个用于将本地业务函数注册到 gorig hub 的轻量级 Go 运行时，同时提供可选的 HTTP 入口用于本地调试和直连调用，属于 [gorig](https://github.com/jom-io/gorig) 生态的一部分。它会从函数签名自动生成 API 描述，上报 hub 并定期心跳，还可以在本地通过 HTTP 直接触发。

- 反射生成参数/返回/JSON 标签的 API schema，便于 hub 记录和 SDK 生成  
- 自动上报注册信息并定时心跳  
- 在现有 gorig 服务里调用 `gnnode.RegServer()` 挂钩，`bootstrap.StartUp()` 由 gorig 启动  
- HTTP 入口 `/{service}/{method}` 便于手工联调  
- `_cmd` 目录仅作演示，真实项目请在自身 gorig 服务中集成  
- 通过 `gncfg` 或配置键 `gn.hub.addr`、`gn.node.addr` 即可设置；`gn.node.addr` 为空时会自动探测局域网 IP 并使用 `:5807` 端口  

快速开始：设置 hub/node 地址，在 gorig 项目中调用 `gnnode.RegServer()`，使用 `register.Server(...).RegName(...).Create()` 或最简 `Reg(...)`（自动推断方法名）注册业务函数，然后按 gorig 的方式启动服务；本仓库的 `_cmd` 可 `go run ./_cmd` 做本地示例。入参需要按 `arg0/arg1` 包裹成 JSON，返回体中 `error` 为业务错误信息。
