# gomc

`gomc` 是一个使用 Go 语言重写 Minecraft Java Edition 1.6.4（MCP 8.11 反编译源码）的工程。

目标不是“灵感复刻”，而是**按原版行为做翻译式重写**：

- 以 `mcp811/src/minecraft/net/minecraft/src` 为行为真值来源
- 优先保证机制、时序、数据格式与原版一致
- 仅在不改变行为的前提下使用 Go 惯用写法

## 当前状态（持续推进中）

项目已经可以启动客户端并进行离线游玩，同时具备较完整的 1.6.4 协议、基础世界、GUI 与渲染链路。当前仍在持续做 1:1 细节对齐（渲染、交互、物理、GUI 细节、边界行为）。

## 已实现（摘要）

以下为当前主要完成模块的概览（非完整清单）：

- `pkg/util`：`JavaRandom`（对齐 `java.util.Random`）
- `pkg/nbt`：NBT 读写（含压缩流）
- `pkg/network/protocol`：1.6.4 协议主干包、登录/游玩核心包、实体/区块/背包相关包
- `pkg/network/crypt`：RSA + AES/CFB8 登录加密链路
- `pkg/network/client`：客户端会话、区块缓存、实体跟踪、背包快照、事件流
- `pkg/network/server`：离线本地服务器、登录流程、移动/交互、基础命令、世界时间推进
- `pkg/world/chunk`：区块结构、NibbleArray、光照/高度基础路径
- `pkg/world/storage`：Region/Anvil 读写与运行时保存
- `pkg/tick`：20 TPS 调度骨架
- `pkg/render/gui`：主菜单、选项菜单、HUD、热键、物品栏/聊天界面基础、第一人称手持物渲染、部分实体与方块渲染
- `cmd/client`：GUI 客户端与离线模式启动
- `cmd/server`：可独立启动的服务端入口

## 环境要求

- Go 1.22+（建议）
- 启用 CGO
- OpenGL 2.1 可用显卡驱动
- Windows（MinGW-w64 GCC）或 Linux（GCC + OpenGL/GLFW 运行库）

Windows（PowerShell）示例：

```powershell
$env:CGO_ENABLED = "1"
go version
gcc --version
```

## 构建与运行

在仓库根目录（`gomc/`）执行：

```powershell
go build ./...
```

启动离线可玩客户端（会自动拉起本地 world server）：

```powershell
go run ./cmd/client -offline -username Steve
```

单独启动服务端：

```powershell
go run ./cmd/server -listen 127.0.0.1:25565
```

客户端连接指定服务器：

```powershell
go run ./cmd/client -addr 127.0.0.1:25565 -username Steve
```

## 工程结构

```text
gomc/
  cmd/
    client/    # GUI 客户端入口
    server/    # 服务端入口
  pkg/
    nbt/
    network/
      protocol/
      client/
      server/
      crypt/
    world/
      block/
      chunk/
      gen/
      storage/
    tick/
    util/
    render/gui/
    audio/
  assets/
```

## 对齐原则

- 这是“翻译工程”，不是重新设计
- 遇到机制不确定时，以 MCP 1.6.4 Java 源码为准
- 保留原版关键常量、tick 顺序与边界行为
- 对非显而易见翻译添加 Java 类/方法注释引用

## 测试与验证

常用检查命令：

```powershell
go test ./pkg/render/gui
go test ./pkg/network/client
go test ./...
go build ./...
```

建议在关键模块上同时做：

- 固定种子结果对比
- 数据包行为对比
- 边界行为回归测试（红石/碰撞/背包交互等）

## 已知差距（持续补齐）

- 仍有部分 GUI 子页面与交互细节未 1:1 完成
- 仍有渲染表现细节在持续对齐原版（含部分模型/透明/特效边界）
- 生物 AI、完整世界生成与更多系统行为仍在继续翻译

## 参考来源

- MCP 8.11 反编译源码：`mcp811/src/minecraft/net/minecraft/src`
- 本工程所有对齐实现以 1.6.4 Java 行为为基准

---

模块路径：`module github.com/lulaide/gomc`
