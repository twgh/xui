# HANDOFF — 交接文档

> 新会话/新人接手从这里开始。读完本文件 → 按需查 AGENTS.md(硬规则)→ DESIGN.md(架构与踩坑)→ README.md(用法)。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现的源码为唯一权威。

## 当前状态(2026-10-06)

- ✅ **P0 运行时**完成:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter 绘制助手、字体缓存、focus-visible 焦点来源跟踪
- ✅ **P1 组件**完成并视觉验收:Button(5 变体×3 尺寸)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**完成并视觉验收:Switch(2 尺寸)、Progress、Slider、Tabs(default/line 两变体)、Tooltip、ComboBox、图标系统(矢量复刻 lucide path,非 SVG 渲染——原因见 DESIGN.md 第八节 17;抗锯齿出图见第八节 43)
- ✅ **P3 首项 Dialog 完成**(详见下方 Dialog 条目)
- ✅ **Slider 渲染缺陷彻底修复**(2026-10-04,computer-use 真实鼠标逐场景验证):①悬浮冒灰方块/②拖动后冒白方块(根因:原生 SliderBar 的滑块按钮 BkInfo 在 Stay/Down/Focus 态的残留绘制, 全套压制无效 → **弃用原生载体, 改纯自绘 Element + 自实现拖拽/点击跳转/键盘**,见 DESIGN.md 第八节 25);③两端 ring 被裁剪(几何重排,见第八节 26);④hover 离开/点击后灰环残留(根因:透明元素重绘不擦底, onPaint 开头先铺宿主底色, 见第八节 28)。期间误判过 `FillRoundRect` 半径语义(x2 导致全组件过圆, 已回退, 结论见第八节 27)
- ✅ **CLI** 完成:`xui init / add / list`(registry.json 驱动,组件源码拷入用户项目)
- ✅ **demo + 用户项目 E2E**:拷贝出的组件在独立用户模块编译/运行/渲染全部正常
- ✅ **demo 布局化**(2026-10-04):卡片内容改用 `widget.NewLayoutEle` 流式布局(vStack 列布局 / hRow 行布局 / vCol Weight 等分列),不再手算 y。LayoutEle 的真实语义(默认横向流、Fill 独占整行、间距 API 分工等)全部探针实测,已记入 DESIGN.md 第八节 29
- ✅ **参考实现补齐**(2026-10-04):switch/progress/slider/tabs/tooltip/**select** 的 radix-nova 源码已从 registry 拉取落盘到 `shadcn/src/components/ui/`(select 里的 `IconPlaceholder` 换成了直接导入 lucide-react 图标,本项目没有 icon-placeholder),App.tsx 加了 Controls 对照卡(与 demo 卡 3 同构:开关/滑动条/进度/标签页/下拉框,line 变体 + Tooltip 包住保存按钮)。`pnpm build` 通过 + 浏览器渲染验证。tsconfig.app.json 加了 `ignoreDeprecations: "6.0"`(新 TS 的 baseUrl 弃用报错)
- ✅ **table 参考实现补齐**(2026-10-05):`npx shadcn@latest add table` 落盘 `shadcn/src/components/ui/table.tsx`, App.tsx 加了"数据表格"对照卡(与 demo 数据页同构:12 行、容器高 180 出现滚动条、第 2 行 `data-state="selected"`)。顺带修掉 `dropdown-menu.tsx` 里残留的 `IconPlaceholder` 导入(同样换成 lucide-react `Check`/`ChevronRight`),`pnpm build` 恢复通过
- ✅ **ComboBox 弹层对齐 shadcn**(2026-10-04,computer-use 截图 + PIL 像素实测):①弹层边框只剩左边一条(根因:BkM border 贴边画被 SetWindowRgn 裁掉三面 + ListBox 自身 paint 事件在该弹层窗口上不渲染输出,内缩 10px 的 6px 红环都画不出来)→ ring 改画在 `XE_PAINT_END` 并内缩 1px,四边完整;②弹层定位改为 **item-aligned**(盖住触发元素、选中项正对触发位置,对齐 radix SelectContent 默认),`wapi.SetWindowPos` 物理像素重定位 + 屏幕边界截断滚动。结论记入 DESIGN.md 第八节 30
- ✅ **Dialog 完成并视觉/交互验收**(2026-10-04,computer-use 真实鼠标键盘 + PIL 像素实测):路线选**窗口内覆盖元素**(不是 ModalWindow/DoModal):全窗自绘遮罩(bg-black/10, 实测 255→229 精确 0.9 合成)+ 居中面板(bg-popover/rounded-xl/ring-foreground/10),标题(text-base)/描述(text-sm muted)/任意内容容器/页脚(-mx-4 -mb-4 贴边、border-t、bg-muted/50、只有下两角圆角)+ 右上角幽灵图标关闭钮。交互:ESC / 点遮罩 / 关闭钮关闭,打开时焦点进第一个可聚焦控件,Tab 在对话框内循环(Shift+Tab 反向),点面板内不关闭;关闭后整窗重绘擦掉遮罩。Button 新增 `WithIcon`(lucide 图标替代文字, 16px 居中)供关闭钮复用;Label 现在 `EnableSwitchFocus(false)` + 灭原生边框。坑与探针结论见 DESIGN.md 第八节 33,对照卡已加到 shadcn App.tsx(dialog.tsx 落盘)
- ✅ **Dialog/文字视觉对齐打磨**(2026-10-04,与用户逐轮截图对比):灭掉内容容器点击后出现的原生蓝焦点框/深色边框;行盒按 leading-none 收紧并给文本元素上下外扩(避免元素裁字);页脚高度全量计入(面板不再矮一截);页脚按钮量宽 ×2 校正(`GetTextShowRect` 在 144 DPI 下约为实绘宽度的一半),按钮宽度与 shadcn 的 48 逻辑像素完全一致;字重收敛到常规(中文 500≈常规);Tooltip 量宽字体与绘制对齐 + 定位去掉重复的客户区偏移(箭头贴住元素上边)。结论全部记入 DESIGN.md 第八节 34
- ✅ **Menu(原生菜单皮肤)完成并视觉/交互验收**(2026-10-05, computer-use + zoom):路线 = 原生 Menu 承载行为 + 窗口级 XWM_MENU_DRAW_BACKGROUND/DRAWITEM 自绘皮肤(每窗口注册一次, 绘制实时取令牌)。条目/勾选/单选(组内互斥)/分组标签/分隔栏/快捷键("	" 借原生算宽)/子菜单/禁用/破坏性全对齐 dropdown-menu.tsx;弹窗圆角 SetWindowRound、元素锚点定位(sideOffset=4, 翻转+夹紧)。探针结论记 DESIGN.md 第八节 36;**注意全屏缩略截图里暗色弹层会假显亮底, 弹层验收必须 zoom**。demo 卡 5 新增
- ✅ **Table(List)完成并视觉/交互验收**(2026-10-05, `components/table.go`):路线 = **原生 `List` 载体 + 外观全自绘**(原生 List 的灰表头/横竖网格线/系统滚动条/白底全部压掉)。为什么不是原生 `Table`:`widget.Table` 是 Shape 系的"单元格画布"(合并单元格、逐格画线、无滚动/无虚拟行), 与 shadcn 的可滚动行列表语义不符, 故 P3 里的 Table/List 两项合并为同一个组件(registry 名 `table`)。对齐 shadcn table.tsx:表头 h-10(40)/px-2 + 1px 底边, 数据行 36(p-2 + text-sm), 行 `border-b`(**最后一行不画** = `[&_tr:last-child]:border-0`), 行悬浮 `hover:bg-muted/50`、选中 `bg-muted`;容器底色取 Card(同 ComboBox 的"组件都放卡片上"约定, 可用 `WithTableOnBackground` 改为 Background)。API:`NewTable(x,y,cx,cy,parent,cols)` + `AddRow/SetCell/SetRow/CellText/Rows/Clear/Select/Selected/OnSelect/Refresh/FitColumns`,列宽 `Width<=0` 表示占满剩余宽度(滚动条显隐翻转时自动重算)。像素实测(亮/暗双主题):行底 `#f5f5f5`→暗 `#262626`, 悬浮 `#fafafa`→暗 `#1f1f1f`, 行边框 `#e5e5e5`→暗白 10% 合成, 行高 54 物理/36 逻辑、表头 60/40, 自适应列 = `cx − 4 − 12`(滚动条让步)。踩坑全部记入 DESIGN.md 第八节 40、41(demo 那个"表格整块消失"是 LayoutEle **纵向溢出换列**, 见 41)
- ✅ **Tree(树形视图)完成并视觉/交互验收**(2026-10-06, `components/tree.go`):路线 = **原生 `Tree` 载体 + 外观全自绘**(展开/收起、点击 chevron 展开、选中、滚动全原生;原生项背景/连接线/分割线/展开按钮 "+/-" 图形全部压掉)。shadcn 无官方 tree 组件,语义按社区通行实现并复用本项目已验收令牌:行高 32(h-8)、视觉矩形左右内缩 4、圆角 6、悬浮/选中 bg-accent(同 select/menu 条目)、每级缩进 16 + 祖先 1px 参考线、chevron 加大居中于点击热区(收起朝右/展开朝下)、叶节点留空槽位文字全树对齐、条目文字 text-sm。自定义项模板(行高 32 + 16px 按钮槽)+ `XE_TREE_DRAWITEM` 自绘;换肤走 `RefreshData()`。像素实测(亮/暗):行高 48 物理/32 逻辑、每级缩进 24/16、选中/hover `#f5f5f5`→暗 `#262626`、普通行透出 Card `#171717`(暗)、文字 `#fafafa`(暗)。交互 computer-use 真实点击验证:chevron 展开/收起、行选中、hover、运行时 Dark/Light 切换整树换肤。踩坑记 DESIGN.md 第八节 42;demo 新增"结构"页(隐藏页创建时选中态丢失已在 demo 规避);shadcn App.tsx 加了同构"树形视图"对照卡(手写 TreeRow,非官方组件)
- ⬜ P3 剩余:hover 过渡动画、CLI 远程 registry

- ⬜ 发布 GitHub(发布后 `go run github.com/twgh/xui/cmd/xui@latest init` 即可用;CLI 已按此设计)

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color/theme/themes_builtin/manager/
│                                 draw(Painter)/icons(图标几何+直通D2D 出图)/focus/font/state
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│   ├── button.go  checkbox.go  radio.go  input.go  switch.go  progress.go
│   ├── slider.go  tabs.go  combobox.go  tooltip.go  dialog.go  menu.go
│   ├── label.go   separator.go(含 Badge)  card.go  table.go  tree.go
├── icons\                        图标资源子包: lucide 预处理数据(2121 个) + Path/Has/Names
├── scripts\gen_icons.py          lucide-static SVG → icons\lucide_gen.go(重新生成时才需要跑)
├── cmd\xui\                      CLI(init/add/list)
├── icons_test.go                 解析器语法 + 资源完整性测试(go test ./...)
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
├── internal\ui\caption.go        菜单美化, 标题上自定义按钮
└── internal\ui\builder.go        List美化,LayoutEle 布局盒子的成熟用法(lkSize 尺寸描述 /
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

# 暗色对照验收: 环境变量决定启动主题, 不用改代码
#     XUI_THEME=dark  启动 demo 即暗色(cmd: set XUI_THEME=dark && demo\demo.exe)
#     运行中仍可用右上角按钮切换(参考页 shadcn 同样支持按 d 键切换)
# 直接启动到指定页签(验收某一页时省得点): XUI_HOME_PAGE=0..5 = 表单/控件/反馈/数据/结构/图标
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
- **自绘标题栏窗口样式**: `Caption|Border|Drag_Window`(无默认图标/标题/三键) + 30px BorderSize, 展示元素要 `EnableMouseThrough(true)` 才不挡拖动, 见 DESIGN 第八节 38; demo 是标准样板
- **渲染固定 Direct2D**(`app.New(true)`)
- **解析 lucide path 时 arc 的标志位是单字符**:`a.25.25 0 00.407.195` 是 flag=0 / flag=0 / x=.407 / y=.195, 按"数字"贪婪切会读成 `00.407` → 参数不足 → **后面整段图形被静默丢掉**(图标只画一半)。解析器已按 SVG 语法逐字符读标志(见 DESIGN.md 第八节 44);自定义图标走 `icons.Path(...)` 或直接贴 path 都行
- **直通 D2D 绘制(第三方 COM 接口)必须先 `XDraw_ConvRect` 把客户区坐标换算成画布对齐坐标**:炫彩只在刷新自己的 `XDraw_*` 命令时才施加元素变换/裁剪, 直通调用拿到的是**窗口坐标系**(不换算就会画到元素外面, 症状可能是"图标莫名消失");另外 `XDraw_GetD2dRenderTarget`/`XC_GetD2dFactory` 交出来的是**真 COM 指针**, 可纯 Go 直通调用(ABI: 禁 float 形参、8 字节结构体按值打包);注意 `ID2D1SimplifiedGeometrySink` 除 `Close` 外全是 void —— 返回值是寄存器垃圾, 当成 HRESULT 查会静默回落。详见 DESIGN.md 第八节 43
- **LayoutEle 默认是"横向流 + 自动换行", 不是竖排**:纵向堆叠必须 `EnableHorizon(false)`;列布局主轴间距用 `SetSpace`, 行布局换行行距用 `SetSpaceRow`(只设 SetSpace 纵向间隙恒 0);`Fill` 独占一整行, 等分用 `Weight`;项级 `LayoutItem_SetAlign` 无效, 用布局级 `SetAlignH/V`(详见 DESIGN.md 第八节 29)
- 顶层 Tabs 分页: 页面容器 Show(false)→Show(true) 后其内部布局元素不会自动重排(空白壳), **对显示的页面调 `AdjustLayout()`** 再 Redraw(demo buildPage);ESC 关闭用 `XWnd_CloseWindow`
- **LayoutEle 纵向(列)布局也会"溢出换列"**: 主轴上剩余高度放不下下一项时, 该项会被甩到下一列(`x += 容器宽度, y = 0`)→ 表现为控件整块消失。列布局一律 `EnableAutoWrap(false)`, 并保证容器高度足够(见 DESIGN 第八节 41)
- 原生 List 作表格载体: 内容区**左上各内缩 2 逻辑像素**、**表头单元格比数据单元格靠左 2px**、行间距默认 1px(`SetRowSpace(0)` 去掉);`List_Item_.RcItem` 注释写"未使用"但**实际有效**, `XE_LIST_DRAWITEM` 按**单元格**触发、矩形首尾相接 → 逐格画行底与行边框;纵向滚动条会吃掉内容宽度(自适应列要让出 ~12px)(见 DESIGN 第八节 40)
- 窗口布局(EnableLayout): 布局项必须 LayoutEle; 主轴剩余用 **Weight** 不是 Fill(Fill 独占整行/列与兄弟重叠); 客户区已避开阴影带; 详见 DESIGN 第八节 39

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
| select 触发区 h-8 rounded-lg border-input **bg-transparent(放卡上 = Card 色)** + 右侧 chevron-down 用 **text-muted-foreground 全色**(非前景半透明);触发区对滚轮无响应(`EnableEvent_XE_MOUSEWHEEL(false)`);弹层 = **item-aligned 定位**(盖住触发元素, 选中项正对触发位置, select.tsx `position="item-aligned"`);弹层条目 = p-1 容器 + 条目 h-7(28) rounded-md hover:bg-accent + 选中项右侧 Check 图标 + 打开时自动高亮选中项;ring-1 描边画在 `XE_PAINT_END`(内缩 1px, 见 DESIGN 第八节 30);内容高度自适应(≤10 项不出滚动条) | select.tsx(已落盘) + combobox.go(弹层接管路线见 DESIGN 第八节 30) |
| menu 弹层 = bg-popover rounded-lg p-1 ring-1 ring-foreground/10(内缩 1px); 条目 h-7(28) rounded-md px-1.5, hover/focus bg-accent + 文字变 accent-foreground; destructive 红字 + hover bg-destructive/10(dark /20); disabled 文字 50% alpha 不响应; 勾选/单选选中项右侧 Check 图标(right-2 size-4), 单选组内互斥; 分组标签 text-xs muted; 分隔栏 1px bg-border(原生槽高仅 4px); 快捷键右对齐 text-xs muted | dropdown-menu.tsx(已落盘) + menu.go(原生菜单皮肤, 见 DESIGN 第八节 36) |
| table 表头 = h-10(40) px-2, 文字 foreground, 只有一条 1px 底边(border);数据行 = p-2 + text-sm(36), 行 `border-b`, **最后一行不画**;行悬浮 `hover:bg-muted/50`, 选中 `data-[state=selected]:bg-muted`;表体无外框/无圆角/无纵向网格线(外框交给 Card), 容器 `overflow-x-auto`, `caption-bottom text-sm` | table.tsx(已落盘) + table.go(原生 List 接管路线见 DESIGN 第八节 40) |
| tree 行高 32(h-8), 视觉矩形左右内缩 4(mx-1), 圆角 rounded-md(6);行悬浮/选中都 bg-accent(同 select/menu 条目);每级缩进 16(size-4), 祖先层级画 1px 参考线(border 色);chevron size-4 text-muted-foreground, 收起朝右/展开朝下(= rotate-90 两态), 叶节点不画但留空槽位(文字全树对齐);条目文字 text-sm foreground;shadcn 无官方 tree, 语义为社区通行实现 + 本项目令牌 | App.tsx 手写 TreeRow 对照卡(无官方 tree.tsx) + tree.go(原生 Tree 接管路线见 DESIGN 第八节 42) |
| tooltip 绑定元素自身有弹层(如 ComboBox)时, 弹层打开要主动调 `HideTooltip(hEle)`——弹层窗口接管鼠标后触发元素收不到 MouseLeave | tooltip.go / combobox.go |
| **disabled:opacity-50 是组级透明**:文字/勾/滑块等"内容"先按原色合成再整体压暗,核心像素保持原色(实测浏览器禁用实心钮白字核心=255,给文字加 50% alpha 只有 194);只有壳体/轨道变淡。radio 的 Label 不跟随禁用 | button/checkbox/radio/tabs/switch/slider.tsx |
| dialog 面板 = w-full sm:max-w-sm(384) rounded-xl bg-popover p-4 gap-4 + ring-1 ring-foreground/10;标题 text-base(16)/描述 text-sm(14) muted,header gap-2 | dialog.tsx |
| dialog 关闭钮 = 幽灵图标钮 size-7(28×28) top-2 right-2(X 图标 size-4);页脚 -mx-4 -mb-4 贴面板左右与底边、border-t、bg-muted/50、p-4、gap-2、右对齐(只有下两角随圆角) | dialog.tsx |
| dialog 遮罩 bg-black/10(固定黑色,不随主题;**不做 backdrop-blur-xs**——xcgui 无背景模糊,见 DESIGN 第八节 33) | dialog.tsx |
| dialog 打开时焦点进第一个可聚焦控件,Tab/Shift+Tab 在对话框内循环(焦点陷阱),ESC / 点击遮罩 / 关闭钮关闭 | radix dialog 语义 |
| **中文 500 字重 ≈ 常规**: Label 默认常规、按钮文字常规(用粗体字形实现 500 会明显偏粗); 标签比描述"看着粗"是**颜色**差(foreground 近黑 vs muted 灰), 实测笔画同宽 | label.tsx 500 + 实测像素 |
| 令牌表 nova 预设 neutral 色板(oklch 已换算 hex) | index.css :root/.dark |
| **字体单位**: xcgui 字体 size 是 pt 不是 px, FontHandle 统一按 px×0.75 换算(14px→11pt); 之前全局文字放大 1.33× | font.go(见 DESIGN 第八节 31) |

## 视觉验证工作流(本项目惯例)

按模型能力分两条路径,像素级事实(颜色值/位置/尺寸/占比)在两条路径里都**用脚本测量、不靠目测**——"看起来的颜色"经常是错的,本项目多次靠像素测量纠偏:

**主模型有视觉**:

1. 构建 + 运行 demo → computer-use 截图(返回图片直接看)
2. `crop`(vision-skills 工具)或 computer-use zoom 放大细部目检
3. 精确数值用像素扫描脚本 `python scripts/pixel_scan.py <png> v|h|box|mix ...` 测量(如圆盘/白点直径占比、hover 前后色值)

**无视觉能力的模型**:

- vision CLI 在 `C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`:glance 看图/ground 定位/crop 放大
- 或主模型自己看完,把坐标与结论转述给子智能体

**通用**:

- 疑难渲染问题写探针程序(`.verify/probeN` 独立 main 包, `go build -o .verify/probeN/probeN.exe ./.verify/probeN` 后运行;`.verify/` 已被 go 工具链忽略),完事删掉
- 需要"真实鼠标"验证 hover/点击时, 探针里用 `wapi.SetCursorPos` 比外部脚本可靠;从 PowerShell 移动鼠标要注意它是 DPI-unaware 进程, 坐标会被虚拟化/偏移, 别拿截图坐标直接当屏幕坐标用(踩过坑)
- 浏览器对照:shadcn 参考页 + xui demo 同屏对比
- 同一张截图里量两边(ours vs ref)是最可靠的对照法

## 已知待办 / 注意事项

- shadcn dev server 由会话后台启动,机器重启后需按上文命令重启
- Button Link 变体(下划线)未实现;text 手绘后 Input 仍是原生文字;hover 过渡动画未做(ani 包可用)
- 用户项目须 `app.New(true)`(D2D)+ `EnableAutoDPI`,demo 是标准样板

## 已完成: 图标抗锯齿(方案二 D2D 直通填充)

- 图标出图改为**直通 D2D 几何填充**(`d2dicon.go` + `icons.go`): lucide path → CPU 描边展开(段四边形 + 圆帽)
  → `ID2D1PathGeometry` → `RT::FillGeometry`; 无需 cgo/汇编/字体/SVG 库。GDI 渲染(`app.New(false)`)
  或资源创建失败时自动回落到旧折线描边。
- 关键坑与结论全部记入 **docs/DESIGN.md 第八节 43**(含 COM 指针/vtable 索引表/ABI 三规则/void 方法陷阱/
  "直通调用必须自己 `XDraw_ConvRect` 换算坐标"/WINDING 同向/隐形窗口); 验证清单已逐项过:
  四个标题栏图标 + 下拉框 chevron + 弹层勾选 + 菜单勾选/子菜单箭头 + 对话框关闭钮 + IconSearch 全家桶(150% DPI);
  暗色主题 + hover + tooltip; GDI 回落不崩; 其余组件视觉无回归。
- 探针程序与临时计划已完成使命并删除(`probe/`、`.verify/`)。像素测量用的是脚本 + PIL, 未靠目测。

## 已完成: lucide 图标资源(预处理)

- `icons/` 子包内置 **2130 个** lucide 图标(lucide-static **1.52.0**), 用法
  `p.Icon(icons.Path("eye"), &rc, color, 2)`;数据由 `scripts/gen_icons.py` 从 `lucide-static` 的 SVG 生成
  (图元转换 / 相对指令绝对化 / 数字归一 / `!` 填充标记)。**生成器默认取最新版**, `--version` 可钉版
  (要跟参考实现严格对齐时钉 lucide-react 的同版本)。
- 组件内部用的 11 个图标保留成 `xui.IconXxx` 常量: 实测 import 资源表 +425KB 二进制, 而组件只用 4 个;
  常量零成本, 漂移由 `TestBuiltinIconsMatchResource`(几何一致性)守着 —— **这条测试上线当天就抓到
  `eye`/`moon` 常量是旧版 lucide 的图形**, 已更正。
- 转换规则与三个隐蔽坑(SVG 每个图元独立坐标系必须绝对化;arc 标志位是单字符, `a.25.25 0 00.407.195`
  按数字切会静默丢半张图;`fill=currentColor` 图元同时继承描边, 要输出"填充+描边"两条)记在
  **DESIGN.md 第八节 44**;运行时解析器同步修好, 现在贴 lucide 原始 path 也不会断。
- 验收: 同屏对照浏览器渲染的原始 SVG(150% DPI 对齐), 32 个图标像素差 ≤4.2/255、新版的 9 个 ≤4.7/255;
  `go test ./...` 会把全部 2130 个图标过一遍解析器(完整性回归)。
- **demo 第 6 个页签「图标」**是这份资源的浏览入口: 9x5 格自绘格子 + 上下翻页 + 悬浮 bg-accent,
  亮/暗两套主题都验过(`XUI_HOME_PAGE=5` 直接启动到该页)。
