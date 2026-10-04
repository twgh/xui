# HANDOFF — 交接文档

> 新会话/新人接手从这里开始。读完本文件 → 按需查 AGENTS.md(硬规则)→ DESIGN.md(架构与踩坑)→ README.md(用法)。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现的源码为唯一权威。

## 当前状态(2026-10-04)

- ✅ **P0 运行时**完成:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter 绘制助手、字体缓存、focus-visible 焦点来源跟踪
- ✅ **P1 组件**完成并视觉验收:Button(5 变体×3 尺寸)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**完成并视觉验收:Switch(2 尺寸)、Progress、Slider、Tabs(default/line 两变体)、Tooltip、ComboBox、图标系统(矢量复刻 lucide path,非 SVG 渲染——原因见 DESIGN.md 第八节 17)
- ✅ **Slider 渲染缺陷彻底修复**(2026-10-04,computer-use 真实鼠标逐场景验证):①悬浮冒灰方块/②拖动后冒白方块(根因:原生 SliderBar 的滑块按钮 BkInfo 在 Stay/Down/Focus 态的残留绘制, 全套压制无效 → **弃用原生载体, 改纯自绘 Element + 自实现拖拽/点击跳转/键盘**,见 DESIGN.md 第八节 25);③两端 ring 被裁剪(几何重排,见第八节 26);④hover 离开/点击后灰环残留(根因:透明元素重绘不擦底, onPaint 开头先铺宿主底色, 见第八节 28)。期间误判过 `FillRoundRect` 半径语义(x2 导致全组件过圆, 已回退, 结论见第八节 27)
- ✅ **CLI** 完成:`xui init / add / list`(registry.json 驱动,组件源码拷入用户项目)
- ✅ **demo + 用户项目 E2E**:拷贝出的组件在独立用户模块编译/运行/渲染全部正常
- ✅ **demo 布局化**(2026-10-04):卡片内容改用 `widget.NewLayoutEle` 流式布局(vStack 列布局 / hRow 行布局 / vCol Weight 等分列),不再手算 y。LayoutEle 的真实语义(默认横向流、Fill 独占整行、间距 API 分工等)全部探针实测,已记入 DESIGN.md 第八节 29
- ✅ **参考实现补齐**(2026-10-04):switch/progress/slider/tabs/tooltip/**select** 的 radix-nova 源码已从 registry 拉取落盘到 `shadcn/src/components/ui/`(select 里的 `IconPlaceholder` 换成了直接导入 lucide-react 图标,本项目没有 icon-placeholder),App.tsx 加了 Controls 对照卡(与 demo 卡 3 同构:开关/滑动条/进度/标签页/下拉框,line 变体 + Tooltip 包住保存按钮)。`pnpm build` 通过 + 浏览器渲染验证。tsconfig.app.json 加了 `ignoreDeprecations: "6.0"`(新 TS 的 baseUrl 弃用报错)
- ⬜ P3:Dialog、原生菜单皮肤、Table/List/Tree、hover 过渡动画、CLI 远程 registry
- ⬜ 发布 GitHub(发布后 `go run github.com/twgh/xui/cmd/xui@latest init` 即可用;CLI 已按此设计)

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color/theme/themes_builtin/manager/
│                                 draw(Painter)/focus(焦点来源)/font/state
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│   ├── button.go  checkbox.go  radio.go  input.go
│   ├── label.go   separator.go(含 Badge)  card.go
├── cmd\xui\                      CLI(init/add/list)
├── demo\                         演示应用(也是全组件编译测试目标)
├── registry.json                 组件清单(files + deps)
├── DESIGN.md                     架构设计 + xcgui 踩坑清单(第八节必读)
├── README.md                     用法
└── AGENTS.md                     agent 会话自动加载的入口规则

D:\GOProject\src\ui\shadcn\       ← 本地部署的 shadcn-ui 参考实现(语义权威)
├── src\components\ui\*.tsx       组件语义的唯一来源(hover/focus-visible/disabled 等)
├── src\index.css                 主题令牌(:root/.dark, nova 预设 neutral 色板)
└── src\App.tsx                   与 demo 同构的对照页

D:\GOProject\src\tools\llama-launcher\  ← 同技术栈的成品项目(可参考)
└── internal\ui\builder.go        LayoutEle 布局盒子的成熟用法(lkSize 尺寸描述 /
                                  Box/Card 工厂 / widget 自带 LayoutItem_* 包装),
                                  其布局语义结论与本项目 DESIGN.md 第八节 29 一致
```

## 常用命令

```bash
# xui demo(全部组件的编译+视觉验证目标)
cd D:\GOProject\src\ui\xui
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe

# shadcn 参考实现 dev server(可能已在运行: netstat -ano | findstr 5173)
cd D:\GOProject\src\ui\shadcn && pnpm dev    # http://localhost:5173

# CLI 本地测试(仓库未发布, 远程 go run 会失败)
go run ./cmd/xui list / init <目录> / add <组件...>
```

## 关键设计决策(一句话版, 详见 DESIGN.md)

1. **copy-not-install**:组件源码由 CLI 拷进用户 `ui/` 目录,归用户;运行时(theme/icons/helpers)走 go.mod
2. **绘制时取令牌**:组件永不缓存颜色,paint 回调里 `xui.Current()` 现取 → 切主题 = 换指针 + Redraw
3. **行为/视觉分层**:xcgui 原生控件 = 无头行为层,`AddEvent_Paint` + `drawx` 只管视觉
4. **交互语义逐条对齐 shadcn 源码**:改动交互前先读 `shadcn/src/components/ui/*.tsx`
5. **组件文件写薄**:绘制细节下沉运行时助手,降低拷贝模式下的升级冲突面

## xcgui 踩坑清单(浓缩版, 全文见 DESIGN.md 第八节)

- **绘制坐标是逻辑像素**(DPI=144 实测):`XEle_GetClientRect` 返回未缩放矩形,框架渲染时变换;自绘**不要**再按 DPI 预缩放(双重缩放 → 指示器撑满被裁)
- **元素绘制裁剪在自身客户区内**:外扩式焦点环不可见;贴边描边的外半侧被裁 → FocusFrame 用"边缘填充 + 内缩填充"两层结构
- **cy=0 的控件高度就是 0**,不渲染 → 组件构造函数都补了默认高度
- **窗口背景**:必须 `XBkM_Clear` + `XBkM_AddFill(Window_State_Flag_Leave)`,用 Nothing(0) 状态注册不生效
- **标题文字默认白色**(为蓝色标题栏设计),主题化窗口需 `SetTitleColor` 跟随主题
- **事件一律 AddEvent\***(2000 回调上限)、**不碰背景管理器**(累积上限 20,窗口背景的 Clear+AddFill 除外)
- **弹层真透明**:`SetTransparentType(Window_Transparent_Shaped)` + `SetTransparentAlpha(255)` 后,窗口形状 = 自身自绘内容中不透明像素的并集 → 圆角外/箭头两侧真透明。**此时不能再给窗口挂背景填充**(挂了整窗变不透明方块,旧 Tooltip 就是这么退化的)。详见 DESIGN.md 第八节 24;背景对象圆角用法也在那一条
- **`Window_Style_Nothing`/`Center` 的窗口自身不绘制任何东西**(只画子元素),别拿它当普通宿主窗口
- **渲染固定 Direct2D**(`app.New(true)`)
- **LayoutEle 默认是"横向流 + 自动换行", 不是竖排**:纵向堆叠必须 `EnableHorizon(false)`;列布局主轴间距用 `SetSpace`, 行布局换行行距用 `SetSpaceRow`(只设 SetSpace 纵向间隙恒 0);`Fill` 独占一整行, 等分用 `Weight`;项级 `LayoutItem_SetAlign` 无效, 用布局级 `SetAlignH/V`(详见 DESIGN.md 第八节 29)
- 标题栏无图标无三键的窗口样式组合见 demo/main.go;ESC 关闭用 `XWnd_CloseWindow`

## 与 shadcn 对齐的语义(改交互前先对照源码)

| 语义 | shadcn 源码依据 |
|---|---|
| hover:bg-primary/80(实心)/80 透明度 | button.tsx variant.default |
| focus-visible 键盘才显示焦点框,ring-3 ring-ring/50,边框变 Ring 色 | button/input/checkbox 的 focus-visible:* |
| 文本输入框点击聚焦也显示环(浏览器对文本控件恒匹配 focus-visible) | input.tsx |
| disabled:opacity-50 原色半透明 | 各组件 disabled:* |
| active:translate-y-px 按下整体下沉 1px(含文字→文字手动绘制) | button.tsx 基类 |
| destructive 软色调(bg-destructive/10 + 红字) | button.tsx / badge.tsx |
| radio 选中 = bg-primary 实心圆 + size-2 白点(bg-primary-foreground) | radio-group.tsx |
| checkbox size-4 rounded-[4px],选中 bg-primary 白勾 | checkbox.tsx |
| card 描边 = ring-foreground/10(非 Border 令牌), rounded-xl | card.tsx |
| badge h-5 胶囊(rounded-4xl) | badge.tsx |
| switch 轨道 32x18(选中 bg-primary/未选 bg-input),滑块 16 圆、两侧各留 1px;dark 下选中滑块用 primary-foreground、未选用 foreground | switch.tsx(radix-nova) |
| switch 元素比轨道每边大 3px:给 focus-visible 的 ring-3 留绘制空间,同时外扩点击热区(≈ after:-inset-*) | switch.tsx |
| progress = h-1 rounded-full bg-muted 轨道 + bg-primary 指示器(宽度按百分比) | progress.tsx |
| slider = **纯自绘**(普通 Element, 不用原生 SliderBar——其滑块按钮 BkInfo 残留绘制压不住, 见 DESIGN.md 第八节 25):轨道 h-1 bg-muted + 已完成区间 bg-primary + 滑块 size-3 border-ring bg-white(字面量 white),hover/拖动中/focus-visible 显示 ring-3 | slider.tsx |
| slider 元素比可见轨道左右各宽 9px(滑块半径 6 + ring 3, 给两端 ring 留绘制空间, 见 DESIGN.md 第八节 26);滑块中心在轨道两端之间移动, 位置由 pos 自算 | slider.tsx(radix 滑块中心贴轨道端点) |
| slider 交互全自实现:点击跳转 + LButtonDown 起 `XEle_SetCapture` 拖拽 + MouseStay/MouseLeave 自跟踪 hover(拖完移开光环即消失, 与 shadcn 一致)+ KeyDown 左右 ±5%/Home/End | — |
| tabs 列表 rounded-lg p-[3px] h-8 bg-muted,标签等宽 rounded-md,未选 text-foreground/60、选中 bg-background;line 变体透明列表 + 底边 h-0.5 下划线;标签 text-sm **font-medium(500)≈中文常规粗细,勿用 Bold(700)**(实测对比) | tabs.tsx |
| tooltip = bg-foreground rounded-md px-3 py-1.5 text-xs text-background + size-2.5 旋转方块箭头,延迟 0;窗口为异型透明,气泡外真透明 | tooltip.tsx |
| select 触发区 h-8 rounded-lg border-input **bg-transparent(放卡上 = Card 色)** + 右侧 chevron-down 用 **text-muted-foreground 全色**(非前景半透明);触发区对滚轮无响应(`EnableEvent_XE_MOUSEWHEEL(false)`);弹层条目 = p-1 容器 + 条目 h-7(28) rounded-md hover:bg-accent + 选中项右侧 Check 图标 + 打开时自动高亮选中项;内容高度自适应(≤10 项不出滚动条) | select.tsx(已落盘) + combobox.go(弹层接管路线见 DESIGN 第八节 30) |
| tooltip 绑定元素自身有弹层(如 ComboBox)时, 弹层打开要主动调 `HideTooltip(hEle)`——弹层窗口接管鼠标后触发元素收不到 MouseLeave | tooltip.go / combobox.go |
| **disabled:opacity-50 是组级透明**:文字/勾/滑块等"内容"先按原色合成再整体压暗,核心像素保持原色(实测浏览器禁用实心钮白字核心=255,给文字加 50% alpha 只有 194);只有壳体/轨道变淡。radio 的 Label 不跟随禁用 | button/checkbox/radio/tabs/switch/slider.tsx |
| 令牌表 nova 预设 neutral 色板(oklch 已换算 hex) | index.css :root/.dark |

## 视觉验证工作流(本项目惯例)

按模型能力分两条路径,像素级事实(颜色值/位置/尺寸/占比)在两条路径里都**用脚本测量、不靠目测**——"看起来的颜色"经常是错的,本项目多次靠像素测量纠偏:

**主模型有视觉**:

1. 构建 + 运行 demo → computer-use 截图(返回图片直接看)
2. `crop`(vision-skills 工具)或 computer-use zoom 放大细部目检
3. 精确数值用 PIL 脚本 `scripts/dominant_colors.py` 测量(如圆盘/白点直径占比、hover 前后色值)

**无视觉能力的模型**:

- vision CLI 在 `C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`:glance 看图/ground 定位/crop 放大
- 或主模型自己看完,把坐标与结论转述给子智能体

**通用**:

- 疑难渲染问题写探针程序(.verify/probeN)二分定位,完事删掉
- 浏览器对照:shadcn 参考页 + xui demo 同屏对比
- 同一张截图里量两边(ours vs ref)是最可靠的对照法

## 已知待办 / 注意事项

- 切到暗色主题后标题栏会出现原生三键(light 下没有), 疑似 SetTheme 触发标题栏样式重算, 与布局化无关, 待查
- shadcn dev server 由会话后台启动,机器重启后需按上文命令重启
- Button Link 变体(下划线)未实现;text 手绘后 Input 仍是原生文字;hover 过渡动画未做(ani 包可用)
- 用户项目须 `app.New(true)`(D2D)+ `EnableAutoDPI`,demo 是标准样板
