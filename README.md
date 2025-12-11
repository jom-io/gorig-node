# gorig-node

`gorig-node` makes your [gorig](https://github.com/jom-io/gorig) service act as a hub-aware microservice node: it registers handlers to gorig-hub and can expose an HTTP ingress for direct calls. One process can publish multiple services; each service can have its own host binding. `_cmd` is just a demo; integrate into your own gorig service.

## Quick start (English)
1) Add dependency: `go get github.com/jom-io/gorig-node@latest`
2) Configure hub/node addresses (config or code):
```go
gncfg.UseConfig(gncfg.GlobalConfig{
    HubAddr:  "127.0.0.1:5806", // required
    NodeAddr: "127.0.0.1:5807", // optional, auto-detected if empty
})
```
3) In your gorig service `main`, register once before startup:
```go
func main() {
    gnnode.RegServer()
    bootstrap.StartUp()
}
```
4) Register handlers (auto method name via `Reg`, or explicit via `RegName`); set `Host` per service when needed:
```go
register.Server("UserSample").
    Reg(func(ctx context.Context, req loginReq) (loginResp, error) { /* ... */ }).
    Create()

register.Server("OrderSample").
    Host("10.0.0.5:8081").
    Reg(func(ctx context.Context, req orderReq) (orderResp, error) { /* ... */ }).
    Create()
```
5) Run your gorig service. For quick demo: `go run ./_cmd`.  
6) Optional HTTP ingress for debugging: `POST /{service}/{method}` on the node address.

## 快速上手（中文）
1) 引用依赖：`go get github.com/jom-io/gorig-node@latest`
2) 配置地址（配置文件或代码）：
```go
gncfg.UseConfig(gncfg.GlobalConfig{
    HubAddr:  "127.0.0.1:5806", // 必填
    NodeAddr: "127.0.0.1:5807", // 选填，留空自动探测
})
```
3) 在 gorig 服务入口调用一次：
```go
func main() {
    gnnode.RegServer()
    bootstrap.StartUp()
}
```
4) 注册接口：`Reg` 自动取函数名，`RegName` 自定义方法名；需要时用 `Host(...)` 为单个服务指定地址：
```go
register.Server("UserSample").
    Reg(func(ctx context.Context, req loginReq) (loginResp, error) { /* ... */ }).
    Create()

register.Server("OrderSample").
    Host("10.0.0.5:8081").
    Reg(func(ctx context.Context, req orderReq) (orderResp, error) { /* ... */ }).
    Create()
```
5) 像平常一样启动 gorig；体验示例可运行 `go run ./_cmd`。  
6) 调试可直连节点：`POST /{service}/{method}`。  
