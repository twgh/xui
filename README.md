# xui — shadcn 风格的 xcgui 自绘组件

为 [github.com/twgh/xcgui](https://github.com/twgh/xcgui)(炫彩界面库)提供一套 **shadcn-ui 风格**的自绘组件:主题令牌、变体系统、copy-not-install 分发。

```
┌─────────────────────────────────────────────┐
│  你的应用                                    │
│  ┌───────────────────────────────────────┐  │
│  │ ui/  ← 组件源码归你所有, 随意修改       │  │
│  │   button.go  checkbox.go  input.go …  │  │
│  │   theme.go   ← 你的主题, 改这里换肤    │  │
│  └───────────────────────────────────────┘  │
│              ↑ 依赖              ↑ 拷贝      │
│  github.com/twgh/xui (运行时)   xui CLI     │
│              ↑                              │
│  github.com/twgh/xcgui (行为层)             │
└─────────────────────────────────────────────┘
```

## 快速开始

```bash
# 在你的项目根目录执行
go run github.com/twgh/xui/cmd/xui@latest init
# 仓库未发布时可用本地路径:
# go run D:\path\to\xui\cmd\xui init
```

`init` 会创建 `ui/` 目录:写入 `theme.go`(你的主题文件)并拷入全部基础组件。

```go
package main

import (
    "github.com/twgh/xcgui/app"
    "github.com/twgh/xcgui/window"
    "github.com/twgh/xcgui/xcc"
    ui "yourmodule/ui"
)

func main() {
    app.InitOrExit()
    a := app.New(true) // Direct2D 渲染
    a.EnableAutoDPI(true).EnableDPI(true)

    w := window.New(0, 0, 600, 400, "我的应用", 0, xcc.Window_Style_Default)
    w.SetBorderSize(1, 32, 1, 1)
    xui.RegisterWindow(w.Handle)
    xui.BindWindowBackground(w.Handle) // 窗口底色随主题

    ui.NewButton(24, 48, 120, 0, "确定", w.Handle)
    ui.NewCheckbox(24, 92, 160, 0, "记住我", w.Handle)
    ui.NewInput(24, 128, 200, 0, w.Handle, ui.WithPlaceholder("用户名"))

    w.Show(true)
    a.Run()
    a.Exit()
}
```

## 换肤

组件在**绘制时**实时读取主题令牌(等价于 CSS 变量),切换主题零成本:

```go
xui.SetTheme(xui.Dark())          // 一行全局换肤
```

主题文件 `ui/theme.go` 默认是**内置主题的副本**(逐字段改即可换肤),也可以照 shadcn 的
`:root` / `.dark` 自己写一份 —— 令牌与 CSS 变量一一对应(字段表见 docs/DESIGN.md 3.1)。

## 组件

| 组件 | 说明 |
|---|---|
| Button | 6 变体(Primary/Secondary/Outline/Ghost/Destructive/Link)× 8 尺寸档(xs/sm/default/lg + 4 个方形图标档),hover/按下/禁用/焦点环;`WithIcon` 做图标按钮 |
| Checkbox / Radio | 自绘指示器 + 原生状态管理 |
| Input | 原生 Edit(光标/输入法/选择)+ 自绘壳体与焦点环,占位符随主题 |
| Switch / Progress / Slider | 开关 / 进度条 / 滑块;滑块支持自定义值域与步进(`WithSliderRange` / `WithSliderStep`) |
| Tabs | 标签页(下划线指示条随活动页滑动),支持纵向堆叠(`WithTabsVertical`) |
| ComboBox | 下拉框(列表由独立透明弹出窗口承载);触发区支持 sm 尺寸(`WithComboSize`) |
| Menu | 下拉菜单(原生菜单承载行为 + 自绘皮肤):条目 / 勾选 / 单选组 / 子菜单 / 快捷键 / 禁用 / 破坏性 |
| Tooltip | 提示气泡(独立透明弹出窗口,不抢焦点) |
| Dialog | 模态对话框:全窗遮罩 + 居中面板,标题/描述 + 任意内容 + 页脚按钮;ESC、点击遮罩、右上角 X 关闭,Tab 在对话框内循环 |
| Table | 数据表格(原生 `List` 载体 + 全自绘皮肤):表头/行边框/悬浮/选中/主题化细滚动条;列宽支持"占满剩余宽度" |
| Tree | 树形视图(原生 `Tree` 载体 + 全自绘皮肤):缩进参考线/旋转 chevron/行悬浮选中;叶节点留空槽位文字全树对齐,点击 chevron 展开收起 |
| Pagination | 分页(纯自绘单元素,鼠标驱动):上一页/下一页 + 页码 + 省略号折叠(>7 页),内容居中、当前页为描边态;`SetTotal/SetPageSize/SetPage/OnChange` |
| Label / Card / Badge / Separator | 静态展示组件:Badge 6 变体(含 ghost/link,悬停生效)、Card 支持 `sm` 尺寸与说明/页脚、Separator 支持垂直 |

每个组件默认带齐 4 态:正常 / hover / 按下 / 禁用,外加键盘焦点环。

hover 带过渡动画(默认 **150ms + cubic-bezier(0.4,0,0.2,1)**,与 tailwind / shadcn 的 `transition-*`
默认值一致;表格行/树行/菜单条目这类逐项高亮同样有过渡)。过渡由运行时管理(临时开一个 16ms 的
定时器,跑完自动关,静止时零开销;实际帧率受下面的 `SetPaintFrequency` 节流),组件侧只写一行:
`hv := xui.Hover(hEle, st.Hover)` + `xui.Fade(静止色, 悬停色, hv)`。整体时长/开关可调:

```go
xui.SetHoverDuration(200 * time.Millisecond) // 改时长
xui.SetHoverDuration(0)                      // 关掉过渡(立即到位)
```

想让过渡更顺(约 60fps)只需在 app 初始化里放开 xcgui 默认的 30ms 重绘节流:

```go
a := app.New(true)
a.EnableAutoDPI(true).EnableDPI(true)
a.SetPaintFrequency(10)   // 实测帧率 30.9 → 59.0 fps(见 docs/DESIGN.md 第八节 46)
```

对话框与提示气泡自带进出场动画(dialog 淡入淡出 100ms、tooltip 淡入+8px 滑入 150ms, 对齐
shadcn/tw-animate-css 的默认值)。自己的弹层要用同款动画时:

```go
// 整组元素淡入/淡出(进度 0→1 进场, 1→0 退场; apply 每帧收到当前进度)
xui.Presence(hWnd, 0, visible, xui.PopupDuration,
    func(v float64) { xui.SetElementsAlpha(v, panel, btn1, btn2) },
    func(v float64) { /* 退场到位后在这里隐藏 */ })

// 异型/阴影弹层小窗: 淡入 + 从 (x, y+dy) 滑到 (x, y)
xui.Presence(hWnd, 0, true, xui.PopupDuration,
    xui.SlideWindowAlpha(hPopup, x, y, 0, dy, 255), nil)
```

## 图标

运行时自带一份预处理好的 lucide 图标资源(2130 个,当前版本见 `icons.Version`),
描边用直通 D2D 几何填充出图(自带抗锯齿)。**每个图标一个常量**,名字是 lucide 名的驼峰形式:

```go
import "github.com/twgh/xui/icons"

p.Icon(icons.IconEye, &rc, xui.Current().Foreground, 2)   // 常量: 编译期内联, 零体积成本
p.Icon(icons.IconChartScatter, &rc, color, 2)             // 含"填充点"的图标也照渲染

// 名字来自用户输入/配置时按名字取 —— 会把整张资源表链进二进制(实测 +426KB):
p.Icon(icons.Path("eye"), &rc, color, 2)
icons.Has("star"); icons.Names(); icons.Count()           // 查询
```

也可以直接把任意 lucide path 数据贴进 `Painter.Icon`(24 网格,`strokeWidth` 用 24 网格单位,
lucide 默认 2)。组件内部用到的图标同样取 `icons.IconXxx`:实测"只用常量"的程序体积增量
**0 字节**(2130 条数据里只进用到的那条),所以组件、业务代码都推荐用常量。资源与常量由
`scripts/gen_icons.py` 一起生成(默认取最新版, `--version` 可钉版),转换规则与踩坑见
docs/DESIGN.md 第八节 44。

demo 的第 6 个页签「图标」可以搜索/翻页浏览整份资源:格子里只画图标(不显示名字),
**悬浮出提示气泡显示图标名, 点击把图标名复制到剪贴板**。

## CLI

```bash
xui init [ui目录]      # 初始化(默认目录 ui; 已有 theme.go 则跳过)
xui add button card    # 追加拷贝指定组件(自动带上依赖)
xui add --all          # 拷贝全部组件
xui update             # 更新已装组件(省略组件名 = ui/ 里已装的全部)
xui update button      # 只更新指定组件(含依赖)
xui list               # 列出可用组件
```

选项:`--force` 覆盖本地已修改的文件(默认会警告并跳过)。

组件源码拷进项目后归你所有;`add` / `update` 都不会覆盖你本地改过的文件,确认要丢弃本地改动时加 `--force`。

**数据源**:默认从仓库 `main` 分支现拉组件(组件永远最新,CLI 本体锁版本用
`go run github.com/twgh/xui/cmd/xui@v1.2.3`)。默认源**自动选路**:

```
直连 https://raw.githubusercontent.com/twgh/xui/main
  → Gitee 镜像 https://gitee.com/twgh/xui/raw/main     (国内稳定)
  → 加速代理 ghfast / gh-proxy                          (兜底)
```

每个候选的探测就是拉一次 `registry.json`(6s 超时),第一个通的用它跑完本次命令;全都不通才降级到
CLI 内嵌源码。选路过程逐行打在 stderr 上, 想看清卡在哪一环直接看输出。

环境变量 `XUI_REGISTRY` 覆盖数据源(**写什么就用什么,不再套代理**):

```bash
XUI_REGISTRY=D:\path\to\xui xui add dialog                    # 本地目录(离线)
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/v1.2.3 xui add dialog   # 锁组件版本
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/main xui add dialog     # 只走直连(跳过镜像与代理)
XUI_REGISTRY=https://gitee.com/twgh/xui/raw/main xui add dialog                # 只走 Gitee 镜像
```

显式指定 `XUI_REGISTRY` 时拉取失败会直接报错(便于排查),不静默降级。

## 设计要点(为什么这样做)

- **copy-not-install**:组件源码进你的仓库,可改可删,不受上游版本绑架——组件是起点,不是黑盒。
- **行为/视觉分层**:xcgui 原生控件当作"无头"层(事件/状态/焦点/输入法),`AddEvent_Paint` + `drawx` 只负责外观。
- **绘制时取令牌**:组件永不缓存颜色,这是零成本换肤的前提;需要同步给原生 API 的颜色(Edit 文字色、窗口底色)通过 `xui.OnThemeChange` 订阅。
- **组件文件薄**:绘制细节下沉到运行时助手(`xui.Painter`);连 hover 过渡这种"要定时器/要生命周期"的脏活也归运行时(`xui.Hover`/`HoverItem`/`Fade`),拷贝文件主要是"变体→令牌映射 + 几笔绘制",降低升级冲突面。
- **交互语义对齐 shadcn 源码**:hover 透明度、focus-visible(键盘才显示焦点框)、disabled opacity-50、按压下沉等均逐条对照本地参考实现的组件源码实现。

## 已知约束

- 仅 Windows;渲染模式固定按 Direct2D(`app.New(true)`)设计。
- 组件坐标使用逻辑像素,自绘内容按窗口 DPI 缩放(`EnableAutoDPI` + `xui.Painter.Scale`)。
- 主题切换须在 UI 线程调用 `xui.SetTheme`。

详见 [DESIGN.md](docs/DESIGN.md)。

## 参考实现(本地 shadcn)

`D:\GOProject\src\ui\shadcn` 部署了一份可运行的 shadcn-ui(nova 预设),用于对照组件源码与视觉效果:

```bash
cd D:\GOProject\src\ui\shadcn
pnpm dev        # http://localhost:5173
```

- 组件源码:`src/components/ui/*.tsx`(语义的唯一权威来源)
- 主题令牌:`src/index.css` 的 `:root` / `.dark`
- `src/App.tsx` 是与 `demo/` 同构的对照页,改完两遍跑一遍即可比对

## 项目文档

| 文档 | 内容 |
|---|---|
| README.md | 本文:安装 / 用法 / 组件一览 |
| [docs/DESIGN.md](docs/DESIGN.md) | 知识库:架构原理、主题系统、**xcgui 踩坑全集(第八节)**、已验证 API 清单 |
| [docs/HANDOFF.md](docs/HANDOFF.md) | 开发交接:当前进度 / 待办 / 仓库结构 / 常用命令 / 视觉验证工作流 |
| [AGENTS.md](AGENTS.md) | AI 会话入口硬规则与文档导航 |

> 改任何组件的交互前,先看 DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」,再对照本地 shadcn 源码。
