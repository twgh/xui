# xui — shadcn 风格的 xcgui 自绘组件

<p>
    <a href="https://github.com/twgh/xcgui"><img src="https://img.shields.io/badge/xcgui≥1.4.3-blue" alt="release"></a>
   <a href="https://golang.org"> <img src="https://img.shields.io/badge/golang-≥1.18-blue" alt="golang"></a>
    <a href="https://pkg.go.dev/github.com/twgh/xui"><img src="https://img.shields.io/badge/go.dev-reference-brightgreen" alt="GoDoc"></a>
    <a href="https://github.com/twgh/xui/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-brightgreen" alt="License"></a>
</p>

## 介绍

为 [github.com/twgh/xcgui](https://github.com/twgh/xcgui) (炫彩界面库)提供一套 **shadcn-ui 风格**的自绘组件: 主题令牌、变体系统、copy-not-install 分发。

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

### 示例图

![xui](https://s41.ax1x.com/2026/11/01/pngSYCR.png)

## 快速开始

### 直接运行示例(想先看效果)

不想先建项目时, 可以直接运行仓库里的 demo(全部组件 + 主题切换):

```bash
go run github.com/twgh/xui/demo@latest
```

### 新建项目(推荐)

先装一次 CLI (之后直接敲 `xui`; 安装位置与 PATH 见下方 [CLI](#cli) 章节):

```bash
go install github.com/twgh/xui/cmd/xui@latest
xui init -n myapp
cd myapp
go run .
```

`-n` 会创建一个默认的项目, 目录已存在且非空时需加 `-f/--force`。

想一直用最新的 CLI 可用以下命令:

```bash
go run github.com/twgh/xui/cmd/xui@latest init -n myapp
```

### 已有项目

在项目根目录执行。组件目录默认 ui/, 想换名字: xui init components (或 -p components)

```bash
xui init
```

### 示例源码

```go
package main

import (
    "github.com/twgh/xcgui/app"
    ui "yourmodule/ui" // 换成你的模块路径
)

func main() {
    app.InitOrExit()
    a := app.New(true)                    // Direct2D 渲染
    a.EnableAutoDPI(true).EnableDPI(true) // 跟随系统 DPI
    a.SetPaintFrequency(10)               // hover 过渡 ~60fps

    // 窗口: 圆角阴影 + 自绘标题栏(最小化/最大化/关闭) + 主题化底色, 全部默认配好
    w := ui.NewAppWindow("我的应用", 640, 480)
    content := w.Content()
    content.SetPadding(24, 20, 24, 24) // 内容区留白

    ui.NewButton(0, 0, 0, 0, "确定", content.Handle)
    ui.NewCheckbox(0, 0, 0, 0, "记住我", content.Handle)
    ui.NewInput(0, 0, 0, 0, content.Handle, ui.WithPlaceholder("用户名"))

    w.Show(true)
    a.Run()
    a.Exit()
}
```

标题栏要加自己的按钮(如切换主题)用 `w.AddCaptionButton(icons.IconMoon, func() { ... })`,
排在最小化按钮左边; 内容组件纵向排列, 间距由内容容器的 `SetSpace` 控制。

## 换肤

组件在**绘制时**实时读取主题令牌(等价于 CSS 变量), 切换主题零成本:

```go
xui.SetTheme(xui.Dark())          // 一行全局换肤
```

主题文件 `ui/theme.go` 默认是**内置主题的副本**(逐字段改即可换肤), 也可以照 shadcn 的
`:root` / `.dark` 自己写一份 —— 令牌与 CSS 变量一一对应(字段表见 DESIGN.md 3.1)。

想运行时换"主题色"时, 克隆当前主题并只替换主色组即可(不必重建窗口, 下一帧全窗口自动用新色):

```go
t := *xui.Light()                 // 暗色用 *xui.Dark()
t.Primary = xui.Hex("#2563eb")
t.PrimaryForeground = xui.Hex("#fafafa")
t.Ring = t.Primary                // 焦点环
t.SidebarPrimary = t.Primary      // 侧边栏主色(如用 Sidebar)
t.SidebarPrimaryForeground = t.PrimaryForeground
xui.SetTheme(&t)
```

demo 的标题栏调色板与「基础1 → 主题色」卡片演示了这套做法(`go run github.com/twgh/xui/demo@latest` 可直接跑)。

## 组件

| 组件 | 说明 |
|---|---|
| Button | 6 变体(default/secondary/outline/ghost/destructive/link)× 8 尺寸档;图标按钮 / 图标+文字 / 内容对齐 / 胶囊 / 加载态;`cx` 传 0 按文字自动量宽 |
| ButtonGroup | 按钮组:横向/纵向、首尾外角 + 相邻边直角、宽高按内容自撑;可混放 Button / Input / ComboBox |
| Alert | 提示条:图标 + 标题 + 描述 + 右上动作槽;两变体(default/destructive),高度按内容自撑、文字自动折行 |
| Empty | 空状态容器:图标底板 + 标题 + 描述 + 动作区;三个槽位 `Header()` / `Content()` / `Body()`,不画边框 |
| Field | 表单字段布局:Label / Title / Description / Content / 校验错误区,垂直或水平取向;横向时内容默认 240 宽(嵌套布局只认 Fixed 宽) |
| Item | 列表项:媒体 + 内容 + 动作区 + 头/尾槽位,三变体两尺寸;配 ItemGroup / ItemSeparator / ItemTitle / ItemDescription |
| Breadcrumb | 面包屑:链接(hover 过渡 + `OnClick`)/ 当前页 / 分隔符 / 省略号 |
| Kbd | 键帽:宽度按文字自适应;`NewKbdGroup` 排组合键 |
| Skeleton | 骨架屏:主题圆角 + 2s 脉冲;`WithSkeletonCircle` 做头像占位 |
| Spinner | 加载指示器:图标 1s/圈旋转,尺寸/配色可调 |
| Toggle / ToggleGroup | 双状态按钮与按钮组:default / outline 两变体、三档尺寸;鼠标/空格/回车切换;单选组默认互斥且可全部取消,`spacing=0` 做成分段控件 |
| NativeSelect | 原生下拉框:`<option value>` 字符串语义,最省事的选择框;富条目(图标/禁用/分组)用 Select |
| Select | 自绘下拉(radix item-aligned 弹层):条目可带图标/禁用/分组;弹层盖住触发区、选中项正对触发区,键盘 ↑↓/Home/End/Enter/ESC |
| InputOTP | 验证码输入:逐格输入、打进已填格替换并前进,Ctrl+V 分发粘贴;可限字符模式与分隔位,全满触发 `OnComplete` |
| Accordion | 手风琴:single 单开互斥(可全收)或多开;200ms 高度过渡。**注意**内容里的子项要 `xui.ItemFill` 声明宽度 |
| Collapsible | 可折叠面板:任意控件绑作触发区,内容放进 `ContentStack()`;收起后不占布局 |
| Popover | 弹出卡片:点击触发、点外部/ESC 关闭;内容里嵌了会自己弹窗的控件(如 ColorPicker)时要 `WithPopoverKeepOnBlur(true)`,否则本卡片会先失焦关掉 |
| HoverCard | 悬停卡片:悬停 700ms 打开、移出 300ms 关闭,可交互且不抢焦点;适合用户资料预览 |
| Avatar / AvatarGroup | 圆形头像:图片居中裁圆、无图回落首字母、右下徽标;组做负间距叠放 |
| Sheet | 侧边面板:标题/描述/内容/页脚与 Dialog 同一套;ESC / 点遮罩 / X 关闭,Tab 焦点陷阱,从贴边侧滑入,四边可选 |
| Drawer | 底部抽屉:顶部抓手按住下拖过面板高 1/3 关闭、否则回弹;方向可改 top/left/right |
| Dialog | 模态对话框:全窗遮罩 + 居中面板,标题/描述折行 + 任意内容 + 页脚按钮;`WithDialogScroll` 内容区滚动而头部/页脚固定,`BindClose` 把任意控件变成关闭钮 |
| AlertDialog | 警示对话框:必须明确选择(无 X、点遮罩不关,ESC 关,焦点落在 Cancel);可与 Dialog 共用遮罩/模糊/焦点陷阱,支持再叠一层 |
| Menu | 下拉菜单 / 右键菜单(原生菜单承载行为 + 自绘皮肤):条目/勾选/单选/子菜单/快捷键/禁用/破坏性/前置图标;`NewContextMenu(元素, build)` 右键触发,每次重建 |
| Menubar | 菜单栏:条目 API 与 Menu 共用;打开任一菜单后鼠标移到其它触发区即切换,全套键盘导航,子菜单悬停展开 |
| NavigationMenu | 导航菜单:所有菜单共用一个视口,悬停开、切换时视口随内容收放、新页面从切换方向滑入;内容里可 `LinkIn` 进 `xui.HStack` 做多列网格 |
| Sidebar | 侧边栏:返回"面板 + 主内容区",分组/子菜单/徽章/动作钮/折叠(offcanvas/icon/none, Ctrl+B)/左右贴边;三种外观 sidebar/floating/inset |
| Command | 命令面板:输入即过滤、↑↓ 循环跳过禁用、Enter/ESC;可放进 `ui.NewDialog` 组合成 ⌘K 弹窗 |
| Combobox | "输入即过滤"的建议列表(= Popover + Command):单选选中即关,`WithComboboxMultiple` 多选保持打开;条目模型与 Select 共用 |
| ComboBox | **注意**: 这是原生 ComboBox 载体(对应 shadcn 的 `select`),不是上面那个 Combobox;要"输入即过滤"用 Combobox |
| Input | 输入框:原生 Edit(光标/输入法/选择)+ 自绘壳体与焦点环;校验态、占位符、前缀/后缀/清除/密码切换/计数,以及步进器 `WithStepper`(−/+ 与 ↑/↓,失焦自动夹取吸附) |
| Textarea | 文本域:Input 的多行预设 + **随内容自动长高**;要固定高度/滚动条用 `Input` + `WithMultiline` |
| InputGroup | 输入组:输入框与附加件共用一层边框/焦点环,内层是"裸输入"(不画壳体);四向 addon 可放文本/图标/按钮/Kbd/下拉触发,一个 addon 放多个控件只整体外扩一次;`block-*` addon 存在时自动切成三段式 |
| Checkbox / Radio | 自绘指示器 + 原生状态管理:Checkbox 三态(不确定态画减号、点击落到选中),两者都有校验态、空格切换与键盘焦点环;"整块区域可点"时用 `Checkbox.Toggle()` / `Radio.Select()`(走与点击指示器相同的路径,`SetSelected` 是不触发回调的静默版),放进非 Card 底色容器时用 `SetBackdrop` 跟随宿主底色 |
| RadioGroup | 单选组:`AddItem(value, text)` 或 `Attach(radio, value)` 收编放在 Field 等子容器里的条目(跨父级互斥);↑↓←→ 循环移动并选中、Home/End 首末 |
| Switch / Progress / Slider | 开关 / 进度条 / 滑块:都有禁用与校验态;Progress 可 indeterminate 短条往返与 `ui.NewProgressField`(标题 + 百分比 + 条);Slider 支持值域/步进、范围与多滑块、垂直、反向 |
| Calendar | 月历(全自绘):6×7 网格、范围选择(两次点击落区间)、多月并排、ISO 周数列、禁用与预订划线、年月可下拉、页脚/预设;尺寸有 helper 算宽 |
| DatePicker | 日期框:outline 触发区 + 弹层内放 Calendar,下方放不下自动翻到上方;范围模式并排两个月、预设按钮、禁用日期、可输入形态(键入 `2026-10-22` / `今天` / `3天后`) |
| TimePicker | 时间框:弹层内"时/分"两列滚动列表,选择即生效、弹层保持打开 |
| ColorPicker | 颜色选择器:触发区(色块 + `#rrggbb` + chevron)+ 弹层(SV 方块 / 色相条 / 预设色 / Hex 输入);颜色以 `#rrggbb` 进出 |
| Table | 数据表格(原生 `List` 载体 + 自绘皮肤):表头/行边框/悬浮/选中/主题化细滚动条;列宽可"占满剩余宽度" |
| DataTable | 在 Table 之上加排序 / 筛选 / 分页 / 勾选 / 列显隐 / 双击原地编辑;表头菜单排序(Shift+点多列排序)、表头可拖拽换列顺序、分面筛选、跨页全选(Shift+点表头复选框);列的行为由 `DataColumn` 的字段描述(排序/对齐/富渲染/隐藏/筛选项/只读) |
| Tree | 树形视图(原生 `Tree` 载体 + 自绘皮肤):缩进参考线 / 旋转 chevron / 行悬浮选中;叶节点留空槽位使文字全树对齐 |
| Chart | 图表(纯自绘):柱/折/面积/饼/圆环/径向/雷达,堆叠、平滑、数据点、图例临时隐藏、悬停指示线与提示卡、Y 轴刻度按数据自动取整齐值、点击数据点回调 |
| Pagination | 分页:上/下一页 + 页码 + 省略号折叠(>7 页),当前页描边态 |
| Carousel | 轮播:整页滑动(300ms、循环首尾相接)、←/→ 键盘翻页、鼠标拖拽松手吸附;上/下张是内容外侧 48px 的圆形描边按钮,到边界禁用;幻灯片是纵向流容器,内容超出由视口裁剪;可自动播放与指定初始页 |
| Resizable | 可调整分栏:面板按百分比分主轴空间、窗口缩放保持比例,拖动分隔条伸缩两侧(相邻到下限后挤压更远的);方向键每次 5%、Home/End 到边界、双击分隔条复位,分隔条可 Tab 聚焦 |
| ScrollArea | 滚动容器(原生 `LayoutFrame` 载体):内容比容器高时滚动、内容大小按子项自动撑开、滚动条自动显隐;`SetContentFill()` 切"铺满视口 + 面板吃剩余高度"形态 |
| AspectRatio | 固定宽高比容器:高度 = 宽度 / ratio,宽度变化时自动重算并逐级重排 |
| Tabs | 标签页(default / line 两变体):行内图标、单标签禁用、value 语义、`BindPanel` 绑面板;支持纵向堆叠与自动换行;键盘 = 整块一个 Tab 站点,←/→/↑/↓ 移动、Home/End 首末、Enter/空格切换 |
| Tooltip | 提示气泡(独立透明弹出窗口,不抢焦点):四方向 + 对齐/间距、显示延迟与跳过延迟、键盘焦点也触发、内容可带快捷键块 |
| Toast | 通知(窗口内覆盖元素):默认右下角堆叠,四角与上下居中可选;4 种语义类型 + 动作按钮 + 可选关闭钮,自动关闭、悬停暂停、滑入淡入 |
| ScreenToast | 屏幕通知:独立置顶小窗,固定在**屏幕**右下角,应用最小化/隐藏也能看到、不抢焦点;卡片绘制与自动关闭与 Toast 同款 |
| ScreenPopup | 屏幕弹窗:固定在屏幕工作区右下角的**常驻**面板 —— 不是通知(默认不自动关闭、不堆叠),内容任意 xui 控件、高度自撑;默认不抢焦点,`WithScreenPopupFocusable(true)` 改为点击后键盘可用 |
| Tray | 系统托盘:左键还原/前台化绑定窗口(可覆盖),右键弹 themed Menu,资源图标可直接用 `.syso` 里的 HICON;监听 `TaskbarCreated`,Explorer 重启后自动恢复图标。⚠️ 同一进程只应创建一个托盘图标(xcgui 的托盘 API 是全局状态) |
| AppWindow | 主窗口封装:圆角阴影 + 自绘标题栏(最小化/最大化/关闭、双击最大化/还原、最大化图标自动切换)+ 主题化底色;`NewAppWindow("标题", 640, 480)` 后往 `Content()` 放内容,`AddCaptionButton` 加标题栏按钮 |
| Label / Card / Badge / Separator | 静态展示组件:Badge 6 变体(含 ghost/link)、行内图标、加载指示器、自定义配色与点击回调,`cx` 传 0 按内容量宽、`SetText` 后自动重量;Card 有 `sm` 尺寸与卡片头(标题 + 说明 + 右上动作区)/ 说明 / 页脚;Separator 支持垂直 |

交互类组件默认带齐 4 态: 正常 / hover / 按下 / 禁用, 外加键盘焦点环(静态展示与容器类组件没有按下态)。

hover 带过渡动画(默认 **150ms + cubic-bezier(0.4,0,0.2,1)**, 与 tailwind / shadcn 的 `transition-*`
默认值一致; 表格行/树行/菜单条目同样有), 过渡由运行时管理(静止时零开销)。两件事你自己要做:

```go
a := app.New(true)
a.EnableAutoDPI(true).EnableDPI(true)
a.SetPaintFrequency(10)                  // 放开 xcgui 默认的 30ms 重绘节流: 实测 30.9 → 59.0 fps
xui.SetHoverDuration(200 * time.Millisecond) // 改过渡时长;传 0 = 立即到位(关掉过渡)
```

对话框与提示气泡自带进出场动画(dialog 淡入淡出 100ms、tooltip 淡入 + 8px 滑入 150ms)。自己的弹层
要用同款动画就交给运行时的 `xui.Presence`(不要自己起定时器), 用法与坑见 DESIGN.md。

## 图标

运行时自带一份预处理好的 lucide 图标资源(2130 个, 当前版本见 `icons.Version`),
描边用直通 D2D 几何填充出图(自带抗锯齿)。**每个图标一个常量**, 名字是 lucide 名的驼峰形式:

```go
import "github.com/twgh/xui/icons"

p.Icon(icons.IconEye, &rc, xui.Current().Foreground, 2)   // 常量: 编译期内联, 零体积成本
p.Icon(icons.IconChartScatter, &rc, color, 2)             // 含"填充点"的图标也照渲染

// 名字来自用户输入/配置时按名字取 —— 会把整张资源表链进二进制(实测 +426KB):
p.Icon(icons.Path("eye"), &rc, color, 2)
icons.Has("star"); icons.Names(); icons.Count()           // 查询
```

也可以直接把 path 数据贴进 `Painter.Icon`(24 网格,`strokeWidth` 用 24 网格单位,lucide 默认 2);
**手里是整段 SVG**(从 heroicons / feather / iconfont 复制来的 `<svg viewBox="…"><path d="…"/></svg>`)
也交给**同一个 `Painter.Icon`** —— 它会自动识别(看首字符是不是 `<`), 提取 path、按 viewBox 归一到
24 网格、按 fill/stroke 判定填充:

```go
p.Icon(svgText, &rc, color, 0)                // 整段 SVG: 0 = 用 SVG 自己的 stroke-width(heroicons 1.5)
paths, sw, err := xui.ParseIconSVG(svgText)   // 想先校验 / 自己缓存归一化结果
```

组件内部用到的图标同样取 `icons.IconXxx`: 实测"只用常量"的程序体积增量 **0 字节**(2130 条数据里
只进用到的那条), 所以组件、业务代码都推荐用常量。

[Lucide 官网](https://lucide.nodejs.cn/) 可以搜索/浏览整份资源。

## CLI

开关清单与改动判定以 `xui --help` 为准(它随 CLI 版本走); 下面只列影响用法的几个, `-j` 的字段表只在本文档里。

### 安装 / 升级

```bash
go install github.com/twgh/xui/cmd/xui@latest
```

二进制在 `$(go env GOPATH)/bin` (Windows 默认 `%USERPROFILE%\go\bin\xui.exe`), 该目录不在 `PATH` 时把它加进去。

升级重跑同一条命令, 或者用 CLI 自己: `xui self-update` (加 `--check` 只比较当前版本与 `@latest`)。`@latest` 偶尔被
模块代理缓存拖旧, 想立刻用某个版本就直接写 tag (`xui self-update v0.8.2`, 也支持 `main` / commit 短哈希)。

### 命令

```bash
xui init -n myapp       # 新建完整项目(自动 go mod tidy; go.mod + main.go + README + AGENTS.md + 组件目录(默认 ui/) + winres/)
xui init -n myapp -p components # 同上, 但组件目录叫 components/(import 路径与下发文档都跟着这个名字)
xui init -n myapp --dry-run # 预览整个生成过程: 不建目录、不写文件、不联网
xui init -n myapp --force # 目录已存在且非空时放开这道闸门(你的改动仍然保留; 要丢弃再加 -o)
xui init -n .           # 在当前空目录直接生成项目(项目名取目录名)
xui init [组件目录]     # 已有项目: 只写组件目录(theme.go + 基础组件 + DESIGN.md + AGENTS.md; 已存在则跳过)
xui add button card     # 追加拷贝指定组件(自动带上依赖)
xui add --all           # 拷贝全部组件
xui add card --dry-run  # 只显示将创建/覆盖/跳过的文件, 不写本地
xui add card --view     # 打印组件源码(可加 =card.go 或 =card 只看某个文件/某个组件), 不写本地
xui add card --diff     # 与本地文件做逐行 diff(过滤值同上), 不写本地
xui update              # 升级已装组件(省略组件名 = 组件目录里已装的全部, 外加已在的两份文档; 不接 --all)
xui update button       # 只升级指定组件(含依赖; 没装过的只提示, 不会替你装 —— 那是 add)
xui update -o           # 连本地改过的文件一起覆盖(改动会丢, 先备份)
xui list                # 列出可用组件
xui search button       # 搜索: 位置参数就是搜索词(等价于 -q button)
xui view button         # 打印组件源码
xui info                # 显示 CLI / 项目环境信息(组件目录不是 ui 时: xui info -p <目录>)
xui version             # 只打印版本(同 -v / --version; 离线可用, 不拉数据源)
xui self-update [版本]  # 升级 CLI 自己(默认 @latest; 也可 tag / main / commit 短哈希)
xui self-update --check # 只比较当前与 @latest, 不安装(不接受版本参数)
```

组件名统一是 shadcn 文档的 kebab-case(`navigation-menu` / `date-picker` / `scroll-area`), 但不区分大小写、也可省略连字符与下划线: `xui add navigationmenu`、`xui add datepicker` 与 `xui add navigation-menu`、`xui add date-picker` 等价(只有唯一的归一化结果才接受, 不会按前缀猜; 输错时会提示最接近的组件名)。

选项:

| 选项 | 用于 | 作用 |
|---|---|---|
| `-n, --name <项目名>` | init | 新建完整项目(`-n .` = 铺在当前空目录) |
| `-f, --force` | init -n | 放开"目标目录已存在且非空"这道闸门; **不**隐含丢弃你的改动 |
| `-o, --overwrite` | init / add / update | 丢弃本地改动(默认保留, 见下方判定); 管模板文件与组件文件 |
| `-p, --path <目录>` | init / add / update / info | 组件目录, 默认 `ui`。`init -n` 用它在新项目里定名字(import 路径与下发文档跟着走); 只许项目内的相对目录(绝对路径或 `..` 跳出会被拒) |
| `--no-install` | init / add / update | 组件文件照写, 但不联网动依赖(`init -n` 还跳过 `go mod tidy` 与 `xc` 安装) |
| `--dry-run` | init / add / update | 只预览, **没有任何副作用**: 不建目录、不写文件、不写清单、不联网、不生成 `.syso` |
| `-a, --all` | add | 全部组件(`update` 不接: 省略组件名就已经是"已装的全部", 要连没装的一起装用 `add --all`) |
| `--view[=组件或文件]` / `--diff[=组件或文件]` | add / update | 打印源码 / 与本地对比, 不写本地; 过滤值给组件名(`card`)或文件基名(`card.go`)都行, **谁都没命中就按失败退出(码 1)** —— 别把它当成"没有差异" |
| `[词]` / `-q, --query` / `-l, --limit` / `-o, --offset` | list / search | 搜索词(位置参数与 `-q` 等价)、限制数量、跳过前 n 个 |
| `-j, --json` | list / search / info / view / `--view` / `--diff` / `self-update --check` | 结果出结构化 JSON(字段表见下) |
| `--strict` | add / update | 有文件被跳过时用退出码 2 失败(默认只汇总一句就正常退出) |
| `-c, --cwd <目录>` | 全部 | 先切到该目录, 后续相对路径都以它为准 |
| `-s, --silent` | 全部 | **只压进度输出**: 命令结果(`list` 的表、`view`/`--diff` 的正文、`info`、`version`、`--check` 的对比)照常, 警告与错误照常;`xui list -s` 就是一份没有表头、可直接接管道的清单 |
| `-v, --version` / `-h, --help` | 全部 | 版本 / 帮助(`-h` 任何位置都生效) |

三条约束:

- **带值的选项不给空值** —— `-p ""` 与 `-p=` 一样当场报错(空目录会把组件拷进当前目录根, 空的 `--view=` 会从"只看一个文件"变成"全部打印")。
- **`--view` / `--diff` 不与 `--dry-run` / `-o` / `--no-install` / `--strict` 同用** —— 它们本来就不写文件, 一起给会直接报错; `-v` 是版本号, 看源码写 `--view`; `self-update --check` 不接受版本参数。
- 短选项也接受 `-x=值`, 与长选项一致: `-p=src`、`-l=5`、`-o=2`。

**退出码**: `0` = 跑通(有跳过时 stderr 给一句"共 N 个文件被跳过"的汇总); `1` = 命令失败(源拉不到、权限、参数错、`--view`/`--diff` 的过滤值没命中); `2` = 加了 `--strict` 且有文件被跳过。CI 想"没更新干净就失败"就跑 `xui update --strict`(配 `--dry-run` 可先空跑一遍)。

**结构化输出(`-j`)的字段**:

| 命令 | JSON 形状(字段) |
|---|---|
| `list` / `search` | `source` `cli` `runtime`(数据源声明的配套运行时) `query` `offset` `limit` `total`(匹配数) `shown`(本次显示数) `components[]{name files[] deps[]}` |
| `info` | `cli` `cwd` `go` `componentsDir` `componentsInitialized` `goFiles` `source` `updateCheckAt` `updateCheckLatest` |
| `view` / `--view` | `source` `filter` `matched` `files[]{name component content}` |
| `--diff` | `dir` `filter` `matched` `files[]{path file state diff}`, `state` = `missing` / `upgrade` / `edited` / `unknown` |
| `self-update --check` | `current` `latest` `localBuild` `upgradable` `command`(下一步该跑什么, 已最新时为空串) |

空的 `files` / `components` / `deps` 一律是 `[]` 而不是 `null`; 过滤值没命中时靠 `matched=false` 区分(这时 `files` 也是空, 但不是"没有差异")。

组件源码拷进项目后归你所有; `add` / `update` / `init` 都不会覆盖你本地改过的文件, 确认要丢弃本地改动时加 `-o/--overwrite`。

**更新时的判定**: CLI 在组件目录写一份 `.xui-manifest.json`, 记下每个文件安装那一刻的 sha256(`init -n` 生成的 `go.mod` / `main.go` / `README.md` / `AGENTS.md` / `.gitignore` 也一起记)。之后逐文件比对 —— 组件文件与模板文件**共用这一套判定**:

| 本地文件 | `xui update` 的动作 |
|---|---|
| 不存在 | 创建 |
| 与新版完全一致 | 跳过(“已一致”) |
| 内容不同,但该文件的清单哈希与本地一致(装完之后没动过) | **直接替换**(“更新: 版本升级”;加 `-o` 也是这句) |
| 内容不同,且清单哈希对不上(你改过) | 警告并跳过,要 `-o` 才覆盖(“本地修改已丢弃”) |
| 内容不同,但清单里没有这个文件(老项目 / 手动删了清单) | 警告并跳过,要 `-o` 才覆盖(只说“无安装清单基线, 无法判断是否改过”, 不断言你改过) |

也就是说: 自己没动过的文件会跟着版本正常升级, 只有你手改过的那些才需要 `-o`; `--diff` 的每个文件头也标的是同一个来源判定。清单缺失(老项目 / 手动删了它)时退回保守模式 —— 一律"无法判断来源"并跳过, 加 `-o` 之后清单会重新生成。

**运行时版本**: 组件源码跟 `main`(永远最新), 而运行时 `github.com/twgh/xui` 是 Go 模块依赖、**只能按 tag 取**; `registry.json` 的 `runtime` 字段声明与这批组件**配套**的运行时版本(每次发布都抬成刚发的 tag)。`add` / `update` / `init` 装完组件会顺手把项目 `go.mod` 的该依赖抬到它(等价 `go get github.com/twgh/xui@<tag>`, `go.sum` 一起更新), 升级完直接 `go build` 就行, 不用你再 `go get -u`。

这五种情况不动你的依赖: `--dry-run` 与 `--no-install` 只打印命令; `go.mod` 里把该模块 `replace` 到本地工作副本时以你的 `replace` 为准; 项目里找不到 `go.mod` 时只提示; 找到的那份 `go.mod` 既在当前目录之外、又没要求 `github.com/twgh/xui` 时也只提示。想自己锁版本: 先 `go get github.com/twgh/xui@<你选的 tag>` 再跑 `xui update`。

**数据源**: 默认从仓库 `main` 分支现拉组件(组件永远最新); CLI 本体的版本由 `go install` / `go run` 时的 `@latest` 或 tag 决定, 组件版本用下面的 `XUI_REGISTRY` 锁。默认源**自动选路**, 第一个通的用它跑完本次命令, 全都不通才降级到 CLI 内嵌源码; 拷到一半某个文件拉不到(镜像限速、临时挂起)会自动换下一个源接着拷, 不会留下半套组件目录。选路过程逐行打在 stderr 上。

```
直连 https://raw.githubusercontent.com/twgh/xui/main
  → Gitee 镜像 https://gitee.com/twgh/xui/raw/main        (国内稳定)
  → GitCode 镜像 https://api.gitcode.com/api/v5/repos/twgh/xui/contents (国内备用)
  → 加速代理 ghfast / gh-proxy                             (兜底)
```

环境变量 `XUI_REGISTRY` 覆盖数据源(**写什么就用什么,不再套代理**):

```bash
XUI_REGISTRY=D:\path\to\xui xui add dialog                    # 本地目录(离线)
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/v0.8.2 xui add dialog   # 锁组件版本(换成实际 tag)
XUI_REGISTRY=https://raw.githubusercontent.com/twgh/xui/main xui add dialog     # 只走直连(跳过镜像与代理)
XUI_REGISTRY=https://gitee.com/twgh/xui/raw/main xui add dialog                # 只走 Gitee 镜像
XUI_REGISTRY=https://api.gitcode.com/api/v5/repos/twgh/xui/contents xui add dialog  # 只走 GitCode 镜像
```

显式指定 `XUI_REGISTRY` 时拉取失败会直接报错(便于排查), 不静默降级。

## 已知约束

- 渲染固定按 Direct2D 设计(`app.New(true)`)。
- 组件坐标使用逻辑像素(`XEle_GetClientRect` 返回未缩放矩形), 框架渲染时统一按窗口 DPI 变换;
  自绘**不要**再乘 DPI(开 `EnableAutoDPI` + `EnableDPI` 即可)。
- **本模块的 `go` 指令是 1.18**: 即使你用 Go 1.22+ 的工具链, `for range` 的循环变量也按**旧语义共享** ——
  在循环里给控件注册回调(或起 goroutine)时, 循环变量必须先拷一份, 否则所有回调都取到最后一次的值:

  ```go
  for _, item := range items {
      item := item // ← 必须有(本模块 go 指令 < 1.22)
      btn.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int { use(item); return 0 })
  }
  ```

详见 [DESIGN.md](docs/DESIGN.md)。

## 项目文档

交互语义与视觉逐条对照 [shadcn-ui](https://github.com/shadcn-ui/ui) 官方源码实现; 汇总在 DESIGN.md 第四节末。

| 文档 | 内容 |
|---|---|
| README.md | 本文: 安装 / 用法 / 组件一览 |
| [docs/DESIGN.md](docs/DESIGN.md) | **自定义指南(随 `xui init` 安装到组件目录, 默认 `ui/DESIGN.md`)**: 架构原理、主题系统、组件解剖、与 shadcn 的语义对照、**xcgui 踩坑全集(第八节)**、本库依赖的 xcgui API 清单(第九节) |
| [docs/HANDOFF.md](docs/HANDOFF.md) | 仓库维护者用: 当前进度 / 待办 / 仓库结构 / 常用命令 / 视觉验证工作流 |
| [AGENTS.md](AGENTS.md) | 仓库维护者用: AI 会话入口硬规则与文档导航 |

组件自己的完整选项与方法清单在 [pkg.go.dev](https://pkg.go.dev/github.com/twgh/xui) 和拷进你项目的源码注释里(源码就在组件目录, 每个 `With*` 都带说明)。

## License

[MIT](LICENSE)
