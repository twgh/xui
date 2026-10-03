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

主题文件 `ui/theme.go` 里是一组与 shadcn CSS 变量一一对应的令牌,
把任何 shadcn 社区主题的 `:root { --background: … }` 照抄过来即可。

## 组件

| 组件 | 说明 |
|---|---|
| Button | 5 变体(Primary/Secondary/Outline/Ghost/Destructive)× 3 尺寸,hover/按下/禁用/焦点环 |
| Checkbox / Radio | 自绘指示器 + 原生状态管理 |
| Input | 原生 Edit(光标/输入法/选择)+ 自绘壳体与焦点环,占位符随主题 |
| Label / Card / Badge / Separator | 静态展示组件 |

每个组件默认带齐 4 态:正常 / hover / 按下 / 禁用,外加键盘焦点环。

## CLI

```bash
xui init [ui目录]     # 初始化(默认目录 ui)
xui add button card   # 追加拷贝指定组件(自动带上依赖)
xui list              # 列出可用组件
```

组件源码拷进项目后归你所有;`add` 不会覆盖你本地改过的文件。
升级组件:删除对应文件后重新 `add`。

## 设计要点(为什么这样做)

- **copy-not-install**:组件源码进你的仓库,可改可删,不受上游版本绑架——组件是起点,不是黑盒。
- **行为/视觉分层**:xcgui 原生控件当作"无头"层(事件/状态/焦点/输入法),`AddEvent_Paint` + `drawx` 只负责外观。
- **绘制时取令牌**:组件永不缓存颜色,这是零成本换肤的前提;需要同步给原生 API 的颜色(Edit 文字色、窗口底色)通过 `xui.OnThemeChange` 订阅。
- **组件文件薄**:绘制细节下沉到运行时助手(`xui.Painter`),拷贝文件主要是"变体→令牌映射 + 几笔绘制",降低升级冲突面。
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
