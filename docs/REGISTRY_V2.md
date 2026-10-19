# CLI 远程 registry(v2)实施方案

> 状态:**已实现并离线验收通过(2026-10-11)**。仅剩「仓库公开后」两步真实验收(见五,需要 GitHub 可达)。
> 本文保留为实施记录 + 验收依据;关联:HANDOFF「P3 剩余」;DESIGN 第五节(CLI 与 registry);cmd/xui/main.go。

## 一、结论先行(决策已定,勿再开讨论)

- **远程源跟 `main` 分支**(shadcn 同款:组件永远最新);`update` 的哈希警告兜底用户本地改动。
- **版本拉取不需要专门机制,换基址即得**:shadcn 的两层版本能力对我们都有对应——
  - CLI 本体锁版本 = `go run github.com/twgh/xui/cmd/xui@v1.2.3`(Go 模块版本,内嵌 registry 随版本走,与 `npx shadcn@2.3.0` 同构);
  - 组件锁版本 = `XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/v1.2.3`(raw URL 对 tag 引用同样有效;默认不设 = 跟 main)。
  - **不做 `--registry` 旗标**(已定):环境变量已覆盖全部场景(一次性覆盖用内联写法 `XUI_REGISTRY=... xui add dialog`,与旗标等价);现有 CLI 是纯位置参数零 flag 解析,加旗标得不偿失。
- **v1 内嵌源保留为离线回退**:现有 `moduleRoot()`(runtime.Caller 定位模块缓存里的源码树)逻辑原样保留,远程拉取失败时降级并提示。
- **环境变量 `XUI_REGISTRY` 覆盖数据源**(本地路径或任意 URL):离线开发/验证全靠它,**不依赖仓库先公开**。
- 不做缓存/ETag:registry.json 只有几 KB,每次命令现拉,简单可靠。

## 二、现状(v1,已提交 0fc099e 之前)

`cmd/xui/main.go` 单文件:

| 函数 | 职责 | v2 改动 |
|---|---|---|
| `moduleRoot()` | runtime.Caller 定位模块根(模块缓存里也有源码) | 保留,作离线回退 |
| `loadRegistry()` | 读模块根下 registry.json | 拆成 `loadRegistryRemote()` 优先、`loadRegistry()` 兜底 |
| `add(names, uiDir)` | 依赖解析(递归 deps)+ 按清单拷文件 + 内容一致跳过/已改动跳过 | 文件读取从"本地路径"改为"数据源读取";其余逻辑不动 |
| `initProject(dir)` | 建 ui/ + theme.go 模板 + DESIGN.md + 拷基础组件 | theme.go **仍从内嵌模板出**(不远程拉),DESIGN.md 从模块根 `docs/` 读,其余走 add |
| `list()` | 打印清单 | 数据源同 loadRegistry |
| `update` | **未实现** | 新命令,见下 |

`registry.json` 条目(仓库根,shadcn 同构):`{"components": {"dialog": {"files": ["dialog.go"], "deps": ["button","card"]}}}`。

## 三、v2 设计

### 1. 数据源解析(核心新逻辑)

```
默认源:  https://raw.githubusercontent.com/twgh/xui/main
XUI_REGISTRY 环境变量:
  - 空        → 默认源(远程),失败降级内嵌
  - 本地路径  → 直接读该目录(目录下须有 registry.json + components/),完全离线
  - http(s):// 前缀 → 以该值为基址拉取(测试任意仓库/本地 server)
```

- registry URL = `<基址>/registry.json`;组件文件 URL = `<基址>/components/<file>`。
- 请求要设 `User-Agent` 与超时(建议 10s,`http.Client` 全局一个);GitHub raw 对 main 分支无 CDN 陈旧问题(直接透传)。
- 解析失败/网络失败 → **打印一行原因,降级内嵌源继续跑**,除非用户显式指定了 `XUI_REGISTRY`(此时报错退出,静默降级会让调试变痛苦)。

### 2. `update` 命令语义

```
xui update              # 更新 ui/ 里已装的全部组件(按 registry 依赖闭包)
xui update <组件...>    # 只更新指定组件(含其依赖)
xui add --all           # = init 的组件部分(全量 add)
```

逐文件规则(与现有 add 的内容对比同款,只是分支动作不同):

- 用户目录无此文件 → 创建(等价 add);
- 内容与远程一致 → `已一致,跳过`;
- 内容与远程不一致但与"上次安装的版本"一致 → 直接覆盖;
- 都不一致(用户改过) → **警告并跳过**,列出文件名与提示(`--force` 覆盖);
- "上次安装的版本" v2 简化:**不存 manifest**,用两段对比实现不了就先做成"不一致即警告跳过 + `--force`",manifest 留待后续(勿过度设计,先跑通)。

### 3. 边界与不变式

- **只拷 `registry.json` 里 `files` 列的文件**,不做目录遍历(icons 子包、theme.go、demo 均不进远程拷贝范围;icons 走 `go get github.com/twgh/xui` 的运行时依赖,用户项目 import 即得)。
- 依赖解析闭包现有 `resolve()` 递归直接复用,远程/内嵌只是数据来源不同。
- 写文件失败、未知组件等 fatal 路径保持现有风格(`xui: ...` 到 stderr)。
- 输出风格:中文、逐文件一行,与现状一致。

## 四、实施步骤(建议顺序)

1. `loadRegistry` 拆分:新增 `fetch(url) ([]byte, error)` + `registrySource()`(解析 `XUI_REGISTRY` → 返回 registry 与组件文件的统一读取函数 `readComponent(file) ([]byte, error)`;本地路径与远程 URL 两个实现)。
2. `add/list` 接数据源(本地路径实现先通,`XUI_REGISTRY=<仓库根>` 指向本仓库即可测,因为目录布局与远程一致)。
3. 本地 HTTP server 验证远程路径:`python -m http.server 8000`(在仓库根起),`XUI_REGISTRY=http://127.0.0.1:8000` —— 验证 URL 拼接、404 处理、超时降级。
4. 实现 `update` + `--all`(复用 add 的闭包与对比逻辑)。
5. 文档同步:README 用法段加 `update/--all` 与 `XUI_REGISTRY`;DESIGN 第五节"实现现状"段更新;HANDOFF 勾掉待办。
6. 全部构建通过:`go build ./...`(注意 cmd/xui 是独立 main 包,别破坏现有 `go run ./cmd/xui list`)。

## 五、验证清单

- [x] `XUI_REGISTRY=<本仓库根> go run ./cmd/xui list` → 列表来自本地 registry.json,并打印数据源行
- [x] `XUI_REGISTRY=http://127.0.0.1:8123 xui add dialog`(临时目录)→ 拷出 dialog.go + button.go + label.go(dialog 的 deps)
- [x] 默认源(未设 XUI_REGISTRY)→ 按 直连 → Gitee 镜像 → `ghfast` → `gh-proxy` 依次尝试(每个失败打一行 stderr),全不通才**降级内嵌源**,命令仍成功
- [x] `XUI_REGISTRY=http://127.0.0.1:9999`(不可达)与 `…/notexist`(HTTP 404)→ 均报错退出(显式指定不降级)
- [x] `update`:改过 ui/button.go → 警告跳过;删掉 ui/label.go → 重新创建;`--force` → 覆盖;指定组件只更新该闭包;空目录 → 提示"没有已安装的组件"
- [x] `add --all` → 18 个文件(19 个组件条目, separator/badge 共享 separator.go)
- [x] **逐组件单独安装 + 编译**(19 个组件各建一个临时项目 `go build ./ui`):全部通过
- [x] `init` → theme.go + DESIGN.md + 基础组件,临时项目 `go build ./ui` 通过(DESIGN.md 与源文件 MD5 一致;重复 init 跳过已存在)
- [x] **仓库公开后**(唯一依赖 GitHub 的两步,尚未做):
  - [ ] 不设 XUI_REGISTRY 直接 `add` → 走 raw.githubusercontent 真链路
  - [ ] `go run github.com/twgh/xui/cmd/xui@latest init`(新 proxy 会话下;若 proxy 未收录,`GOPROXY=direct go run ...` 先验,并提醒等 proxy 刷新)

> 本轮顺带修掉的 registry 数据缺陷(组件单独安装编译不过,原 v1 清单就错):
> `dialog` 漏登记 `backdrop.go`(backdrop-blur 助手),`combobox` / `menu` 漏登记对 `tooltip` 的依赖
> (`HideTooltip`)。**每次新增组件/新增跨文件助手,必须跑一遍上面的"逐组件单独安装 + 编译"**。

## 六、已知坑(前人踩过,别再踩)

- **默认源自动选路 = 直连 → Gitee 镜像 → 加速代理**(2026-10-11 定稿; 起初按"先代理后直连"实现过
  一版, 按用户要求改为"直连优先 + Gitee 镜像"):候选依次为
  `https://raw.githubusercontent.com/twgh/xui/main` → `https://gitee.com/twgh/xui/raw/main` →
  `ghfast` → `gh-proxy`; 探测 = 拉一次 `registry.json`(6s 超时, 不额外发可达性请求), 第一个通的
  跑完本次命令, 失败逐行打 stderr 便于定位卡点; 显式 `XUI_REGISTRY` **一律原样使用, 不套镜像/代理**
  (想只走直连就把默认地址显式写上)。实测(本机, 2026-10-11):
  - Gitee raw 布局与 GitHub raw **同构**, 直接可用:`https://gitee.com/twgh/xcgui/raw/main/README.md`
    = **200**(gitee API 查得 xcgui 镜像默认分支是 `main`, 不是 `master`); UA 用 `xui-cli` 也放行,
    无需伪装浏览器。
  - **`gitee.com/twgh/xui` 尚不存在(404)**、GitHub 的 `twgh/xui` 也**未公开**(直连被 reset/超时,
    gh-proxy 回源 404)—— 四条候选现在都拿不到真内容, 靠降级内嵌源跑。**仓库公开 + Gitee 建镜像后
    才能真验收**(链路本身已按序跑通)。注意 gitee 镜像需手动同步(或在 Gitee 建仓库并同步 main)。
  - `ghfast` 时通时断(先 TCP refused, 后 HTTP 404), 仅作兜底。
  - 直连挂起时会吃满 6s 探测超时(实测总耗时 3.5s ~ 8.7s 取决于直连是 RST 还是挂起), 这是"直连优先"
    的固有代价; 想避免就显式 `XUI_REGISTRY` 指向 Gitee 镜像。
  - 同形态的第三方实现可参考会话交接文件 `utils/proxy.go`。
- **`go run @latest` 的 CLI 本体有 proxy 缓存延迟**:组件跟 main 不受影响(走 raw.githubusercontent),但 CLI 代码本身的更新要等 proxy 抓取或打 tag —— 组件更新频繁而 CLI 很少变,这是选择"registry 跟 main、CLI 走版本"的根据。
- `moduleRoot()` 依赖 `runtime.Caller`:**v2 改造时不要动它**,内嵌回退还靠它;注意 `go run pkg@version` 时该路径在模块缓存里同样有效(已验证)。
- 用户 ui 目录名可以不是 `ui`(`init` 的参数),`add` 里 uiDir 一直在传参链上,别写成硬编码。
- theme.go 一旦存在必须跳过(用户改主题),v2 远程化后同理,**不要**把 theme.go 加进任何远程清单。
- `registry.json` 的 `files` 是相对 `components/` 的扁平文件名(无子目录),拼接时直接 `components/<file>`,不要做路径清洗以外的处理。
