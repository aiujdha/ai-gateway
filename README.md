# AI Gateway

基于 Go 标准库的统一模型调用服务，包含浏览器操作页、本地模型适配器、主备渠道切换、流式输出及调用记录。

## 启动

安装 Go 1.22+，在仓库根目录运行：

```bash
go run ./cmd/gateway
```

打开 http://127.0.0.1:8080 。输入文字并发送请求；勾选“模拟主渠道故障”可观察请求切换至 backup，以及同一 Request ID 下的两次渠道记录。服务默认只监听本机，可用 GATEWAY_ADDR 修改地址。

## 接口

| 接口 | 用途 |
| --- | --- |
| GET /v1/models | 查看本地模型 local-chat |
| POST /v1/chat/completions | 普通 JSON 或 SSE 响应 |
| GET /api/status | 当前本地额度与最近 100 条渠道执行记录 |
| GET /health | 服务就绪检查 |

调用 /v1/ 和 /api/ 接口需要请求头 `Authorization: Bearer local-gateway-key`。这是公开的本机体验密钥，不是可用于公网服务的认证配置。

```bash
curl http://127.0.0.1:8080/v1/chat/completions -H "Authorization: Bearer local-gateway-key" -H "Content-Type: application/json" -d '{"model":"local-chat","messages":[{"role":"user","content":"hello"}],"stream":false}'
```

请求头 `X-Local-Fail-Primary: true` 可触发主渠道失败；`X-Local-Fail-All: true` 可检查所有渠道失败时不扣额度的行为。返回头包括 X-Request-ID、X-Trace-ID、X-Channel-ID。

## 模块

- internal/relay：Provider 协议与本地确定性文本适配器。
- internal/server：HTTP 接入、主备调用、内存额度及请求追踪，页面随 Go 二进制嵌入。
- internal/routing、access、billing、task、trace：保留扩展接口与类型。

本地适配器不会调用真实模型。响应字段中的 Token 数按字符数近似，仅用于演示用量流程，不代表模型实际 Token 或费用。服务重启后额度和记录清空。当前仅实现 local-chat 的文本调用子集；权重调度、真实模型协议转换、异步媒体任务、数据库与 OSS 尚未接入。

## 检查

```bash
go test ./...
```

架构主题参考 [New API](https://github.com/QuantumNous/new-api)；本仓库是独立实现，不包含其企业定制代码。采用 [MIT License](LICENSE)。
