# gorig-node

gorig-node lets a [gorig](https://github.com/jom-io/gorig) backend run as a service node for microservice expansion: it advertises your handlers to the gorig hub and can optionally expose an HTTP ingress for direct/manual calls. One process can publish multiple services, each with its own `Host` for precise placement. `_cmd` is a demo; real projects integrate this inside their gorig service.

## Use it (English)
1. Configure hub/node addresses via config (`gn.hub.addr`, `gn.node.addr`) or code:
   ```go
   gncfg.UseConfig(gncfg.GlobalConfig{
       HubAddr:  "127.0.0.1:5806", // required
       NodeAddr: "127.0.0.1:5807", // optional, auto-detected if empty
   })
   ```
2. In your gorig service, hook once before gorig boots:
   ```go
   func main() {
       gnnode.RegServer()
       bootstrap.StartUp() // from gorig
   }
   ```
3. Register handlers; `RegName` sets a method name, `Reg` auto-infers from the func name. Publish multiple services and tune their locations with `Host(...)` when needed:
   ```go
   register.Server("UserSample").
       Reg(func(ctx context.Context, req loginReq) (loginResp, error) { /* ... */ }).
       Create()

   register.Server("OrderSample").
       Host("10.0.0.5:8081").
       Reg(func(ctx context.Context, req orderReq) (orderResp, error) { /* ... */ }).
       Create()
   ```
4. Run your gorig service as usual. For a quick demo, run: `go run ./_cmd`.
5. Optional: call APIs directly for debugging at `POST /{service}/{method}` on the node address.

## 简体中文
gorig-node 让 [gorig](https://github.com/jom-io/gorig) 框架的后端项目晋级为可对外扩展的微服务节点：把注册的函数上报到 hub，并可选暴露 HTTP 入口便于直连或手工调用。一个进程可承载多个服务，并可为每个服务单独配置 `Host` 控制落点。`_cmd` 仅作示例，实际请在自己的 gorig 项目中集成。

1. 通过配置（`gn.hub.addr`、`gn.node.addr`）或代码设置地址，`node` 为空会自动探测：同上代码示例。  
2. 在 gorig 服务的启动入口调用一次 `gnnode.RegServer()`，然后走 `bootstrap.StartUp()`。  
3. 用 `register.Server(...).RegName(...)` 或更简的 `Reg(...)`（自动推断方法名）注册业务函数，再 `Create()`；可并行注册多个服务，必要时对某个服务调用 `Host(...)` 指定地址。  
4. 像平常一样启动 gorig 服务；快速体验可 `go run ./_cmd`。  
5. 调试时可直接访问节点地址的 `POST /{service}/{method}`。  
