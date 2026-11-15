# COMPONENT-AUDIT — shadcn 组件特性核查台账

> **用途**:按组件常用性依次核查「还有哪些上游特性未实现、但 xcgui 侧可实现」,记录进度与结论,供后续逐批实现时取用。本文是仓库内部台账,**不随 `xui init` 下发**,不写设计原理(见 DESIGN.md)与完成记录(见 CHANGELOG.md)。
>
> **依据优先级**:① 官方索引页 https://ui.shadcn.com/docs/components 与各组件 `base/` 文档页(示例 / 变体 / 状态);② 文档 API Reference 指向的 Base UI 上游文档(如 Select Root props);③ 本地参照实现 `D:\GOProject\src\ui\shadcn\src\components\ui\*.tsx`(交互语义唯一权威);④ 本库 `components/*.go`、`registry.json`、`DESIGN.md` 第四节末语义表与第七节差异表。
>
> **排除项(不计入「可实现」)**:
> - **RTL / 方向性**:用户明确不做。官方几乎每个组件页都有 RTL 示例,本台账一律不列(范围决策见 DESIGN 第七节 Direction / Calendar 条目)。
> - **已记为 xcgui 能力边界的差异**:DESIGN 第七节里「改组件代码也补不上」的条目(原生 Edit 文字/光标/IME、圆角裁剪、Tab 焦点链、弹层 zoom/slide 等)不算可实现项。第七节里标注「后续有需求再加」「暂不提供」的,属于本台账要列的项。
>
> **状态**:`已核查` = 已逐条比对(备注区分「补齐已落地」与「清单待确认」);`待核查` = 按序等待;`范围外` = 明确不做。
>
> **排序口径**:人工按桌面应用中的使用频次排序 —— 表单输入与基础操作 > 反馈/浮层 > 导航 > 数据展示 > 辅助/装饰 > 聊天等新组件;同一档内按官方示例出现频次。这是**可调整的执行顺序**,不是客观统计。核查按序号推进,跳过 `已核查` 与 `范围外`;已完成 #1..#30(逐组件结论见第三节,实现进度见第五节),**下一轮核查从 #31 Toast 起**。

## 一、按常用性排序的总表

| # | 官方组件 | 本库 key / 实现 | 状态 | 备注 |
|---:|---|---|---|---|
| 1 | Button | `button` | 已核查 | 补齐清单已实现,见 `3342cad` |
| 2 | Input | `input` | 已核查 | 结论见 3.1;8 项可实现,I2 / I7 / I3 / I4 已落地(见第五节;I3 走 Field 必填与校验) |
| 3 | Checkbox | `checkbox` | 已核查 | 补齐清单已实现,见 `3342cad` |
| 4 | Select | `select` | 已核查 | 结论见 3.2;9 项可实现,S3 / S9 / S4 / S8 已落地(见第五节;S9 走 Field 必填) |
| 5 | Combobox | `combobox` | 已核查 | 三批补齐,见 `23bdfc5` / `725d329` / `7441a77` |
| 6 | Dialog | `dialog` | 已核查 | 结论见 3.3;6 项可实现,D1 / D2 / D3 已落地(见第五节) |
| 7 | Card | `card` | 已核查 | 结论见 3.3;5 项可实现(含 1 项能力边界重评估),C1 / C2 已落地(见第五节) |
| 8 | Label | `label` | 已核查 | 结论见 3.3;4 项可实现,L1 / L2 / L3 / L4 已落地(见第五节;L3 由 Field 统一出星号) |
| 9 | Textarea | `textarea` | 已核查 | 结论见 3.4;与 Input 同源 + 3 项特有,TA1(随 Input 的 I2 / I7 与 Field 必填)已落地 |
| 10 | Tabs | `tabs` | 已核查 | 结论见 3.4;4 项可实现,TB1 / TB2 已落地(见第五节) |
| 11 | Switch | `switch` | 已核查 | 结论见 3.4;3 项可实现,SW1 / SW2 已落地(见第五节) |
| 12 | Radio Group | `radio-group` | 已核查 | 结论见 3.4;3 项可实现(+1 扩展),RG1 / RG2 / RG3 已落地(见第五节;RG3 走 Field 必填) |
| 13 | Dropdown Menu | `menu` | 已核查 | 结论见 3.4;5 项可实现,DM1 已落地(见第五节);#56 Context Menu 同一实现 |
| 14 | Tooltip | `tooltip` | 已核查 | 结论见 3.5;3 项可实现 |
| 15 | Badge | `badge` | 已核查 | 结论见 3.5;3 项可实现,BA2 / BA3 已落地(见第五节);文件 `separator.go` |
| 16 | Alert | `alert` | 已核查 | 结论见 3.5;2 项可实现,AL1 已落地(见第五节) |
| 17 | Table | `table` | 已核查 | 结论见 3.5;3 项可实现 |
| 18 | Data Table | `data-table` | 已核查 | 结论见 3.5;3 项可实现,DT2 已落地(见第五节);官方是组合模式,差异见 DESIGN 第七节 |
| 19 | Popover | `popover` | 已核查 | 结论见 3.6;6 项可实现,PO1/PO2/PO3 已落地(见第五节);本批开合控制最完整 |
| 20 | Progress | `progress` | 已核查 | 结论见 3.6;4 项可实现,PR1 min / PR3 轨道配色与圆角 / PR2 宽度过渡已落地(见第五节) |
| 21 | Field | `field` | 已核查 | 结论见 3.6;4 项可实现,FL1 / FL2 / FL3 / FL4 已全部落地(见第五节);Form 亦收敛在此;本地无 `field.tsx` |
| 22 | Input Group | `input-group` | 已核查 | 结论见 3.6;2 项可实现,IG1/IG2 已落地(见第五节);11 个官方示例仅缺 Custom Input |
| 23 | Avatar | `avatar` | 已核查 | 结论见 3.6;5 项可实现,AV1/AV2/AV3/AV4 已落地(见第五节) |
| 24 | Accordion | `accordion` | 已核查 | 结论见 3.7;ACC1 / ACC2 / ACC3 已落地;余 Trigger 内容槽与 demo 缺口 |
| 25 | Slider | `slider` | 已核查 | 结论见 3.7;SL1 / SL2 / SL3 已落地;余碰撞策略与单滑块禁用 |
| 26 | Alert Dialog | `alert-dialog` | 已核查 | 结论见 3.7;AD1 已落地;其余与 Dialog D4 / D5 同源 |
| 27 | Sheet | `sheet` | 已核查 | 结论见 3.7;SH1~SH4 已落地;余内容槽与遮罩配色 |
| 28 | Separator | `separator` | 已核查 | 结论见 3.7;SEP1 / SEP2 已落地;余 demo 对照 |
| 29 | Skeleton | `skeleton` | 已核查 | 结论见 3.7;SK1 已落地(配色改 bg-muted);余示例卡 |
| 30 | Scroll Area | `scroll-area` | 已核查 | 结论见 3.7;SA1 已落地(滚动条常显);余横向滚动 |
| 31 | Toast | `toast` | 待核查 | sonner 形态;下一轮起点 |
| 32 | Toggle | `toggle` | 待核查 | |
| 33 | Toggle Group | `toggle-group` | 待核查 | |
| 34 | Pagination | `pagination` | 待核查 | |
| 35 | Command | `command` | 待核查 | |
| 36 | Calendar | `calendar` | 待核查 | RTL / Jalali / timeZone 已列能力边界 |
| 37 | Date Picker | `date-picker` | 待核查 | |
| 38 | Hover Card | `hover-card` | 待核查 | |
| 39 | Collapsible | `collapsible` | 待核查 | |
| 40 | Breadcrumb | `breadcrumb` | 待核查 | |
| 41 | Navigation Menu | `navigation-menu` | 待核查 | |
| 42 | Menubar | `menubar` | 待核查 | |
| 43 | Sidebar | `sidebar` | 待核查 | |
| 44 | Resizable | `resizable` | 待核查 | |
| 45 | Carousel | `carousel` | 待核查 | |
| 46 | Aspect Ratio | `aspect-ratio` | 待核查 | |
| 47 | Kbd | `kbd` | 待核查 | |
| 48 | Empty | `empty` | 待核查 | |
| 49 | Item | `item` | 待核查 | |
| 50 | Native Select | `native-select` | 待核查 | |
| 51 | Input OTP | `input-otp` | 待核查 | |
| 52 | Spinner | `spinner` | 待核查 | |
| 53 | Chart | `chart` | 待核查 | |
| 54 | Drawer | `drawer` | 待核查 | |
| 55 | Button Group | `button-group` | 待核查 | |
| 56 | Context Menu | `menu` | 待核查 | 与 Dropdown Menu 同一实现 |
| 57 | Trace | `trace` | 待核查 | 官方索引页 New Components 之一 |
| 58 | Message | `message` | 待核查 | |
| 59 | Bubble | `bubble` | 待核查 | |
| 60 | Marker | `marker` | 待核查 | |
| 61 | Attachment | `attachment` | 待核查 | |
| 62 | Message Scroller | `message-scroller` | 待核查 | |
| 63 | Questionnaire | `questionnaire` | 待核查 | |

**范围外(官方侧栏 66 个中的另 3 个 + 侧栏分组)**:

| 上游条目 | 状态 | 理由 |
|---|---|---|
| Code Block | 范围外 | 全量对齐要在 Go 侧实现 Shiki tokenizer;若提供降级版须在 DESIGN 第七节声明部分对齐 |
| Direction | 范围外 | 文字渲染只有 LTR,无 RTL 布局能力(用户本轮再次明确不做) |
| Typography | 范围外 | 纯样式文档页,无可安装组件 |
| AI Elements | 范围外 | 侧栏独立分组,暂无需求 |

## 二、已核查并落地的三个组件(存档)

| 组件 | 提交 | 当时补的「未实现但可实现」项 |
|---|---|---|
| Button | `3342cad`(+ `a6f283b` 修残余) | `SetSize` 运行时尺寸档、`WithButtonLoadingAtEnd/Icon/ColorFunc`、`NewLinkButton`(As Link)、`SetDisabled`/`SetFieldState`、`UsePointerCursor` 全局手型与禁用光标 |
| Checkbox | `3342cad`(+ `a6f283b` 修残余) | `WithCheckboxChecked`(defaultChecked)、`WithCheckboxReadOnly`/`SetReadOnly`、`WithCheckboxDisabled`、可否决的 `OnCheckedChange`、纯指示器 40×32 热区、`SetFocusRingProxy`;新增 `CheckboxGroup` 与 `Field.Attach` 的 19 控件联动 |
| Combobox | `23bdfc5` / `725d329` / `7441a77` | `WithComboboxChips`、`WithComboboxClear`、`WithComboboxAutoHighlight`、`WithComboboxInput`(Basic 输入框触发)、`WithComboboxEditable`(可编辑组合框)、Text 空回落 Value |

三次核查的结论均已写入 CHANGELOG 与 DESIGN 第四节末语义表,本表只留索引,不重复内容。

## 三、逐组件核查结论(#1 ~ #30)

> 本节收纳**全部逐组件核查**, 按批次分成小节(每批一组组件/一轮核查), 新批次往末尾追加小节, **不再新增 H2 章节**。
> 每小节形如: 依据 → 已具备 → 未实现但可实现(表格, 编号 + 实现路径 + 代价/优先级) → 明确不做。落地状态只在第五节记录, 这里不改写。

### 3.1 Input(#2)

**官方 base/input 文档页示例**:Basic / Field / Field Group / Disabled / Invalid / **File** / Inline / Grid / **Required** / Badge / Input Group / Button Group / Form / RTL(排除)。组件源码 `input.tsx` 只有一层样式壳,其余全部是原生 `<input>` 属性转发 + Field 组合。

**已具备(不必再列)**:Field / FieldGroup / FieldSet / FieldLegend / FieldSeparator / FieldError 及 Inline / Grid / Form / Badge-in-label 组合;Disabled(`Enable` / `SetFieldState`);Invalid(`WithInputInvalid` / `SetInvalid` + aria-invalid 红环);Input Group(`NewInputGroup` 四向 addon);Button Group;搜索清除(`WithClearButton`);密码显隐(`WithPasswordToggle`);数字步进(`WithStepper`);字数计数(`WithCounter`);焦点环 / 禁用态 / 暗色配色;多行 Textarea 自动长高。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| I1 | `type="file"` 文件输入形态 | 官方 Input#File;`input.tsx` 的 `file:*` 样式 | 只有独立 `file-dialog`(`Open/OpenMulti/Save/PickDir`,含 filters / MaxFiles),没有输入框形态 | `WithFileInput(...)` 或 `NewFileInput`:只读文件名 + 幽灵「选择文件」钮(对齐 `file:` 样式)+ 调 FileDialog + 多选显示 `+N` + 清除 + 空值校验联动;accept 复用 `WithFileDialogFilters`,多文件复用 `WithFileDialogMaxFiles` | 中 / **高**(桌面高频) |
| I2 | `readOnly` 只读 | 原生 `<input readOnly>`(`...props` 转发);本库 Checkbox 已实现同语义 | Input 无 `SetReadOnly/ReadOnly`;只能 `Enable(false)`,会丢焦点与复制能力 | xcgui 已提供 `XEdit_EnableReadOnly` / `XEdit_IsReadOnly`,直接暴露 `WithInputReadOnly` / `SetReadOnly` / `ReadOnly()`,配色与禁用态区分 | 极小 / **高** |
| I3 | `required` 必填与校验联动 | 官方 Input#Required(标签 `*` + 说明);Base UI 表单校验 | Field 有 `SetInvalid` / `FieldError` / `Attach(FieldControl...)`,但无必填标记与校验入口 | `WithFieldRequired()` + `Field.Validate()` / `Required()`;空值 → `SetInvalid` + 错误文案,复用现有级联,不新增控件接口 | 小-中 / 中 |
| I4 | `maxLength` 硬限制 | 原生 `<input maxLength>` | `WithCounter(max)` **只显示 `n/max`、超限变红,仍可继续输入**(`input.go` 的 `paintAddons` 只读文本长度) | `WithMaxLength(n)` / `SetMaxLength`:在 `AddEvent_Edit_Changed` 里截断(InputOTP 已有输入过滤先例),与 `WithCounter` 配套 | 小 / 中 |
| I5 | `pattern` / 自定义校验器 | 原生 `<input pattern>`;官方表单示例 | 无校验器,校验失败态只能外部手动 `SetInvalid` | `WithValidator(func(string) error)` + `Validate()`,失败时置 invalid 并写 `FieldError` | 中 / 中 |
| I6 | 原生 Edit 已开放但未接线的视觉/行为 API | xcgui `XEdit_*` API 清单 | 当前只用 `SetCaretColor` / `SetDefaultTextColor` / `SetTextAlign` / `EnablePassword` / `EnableAutoSelAll(false)` | `XEdit_SetSelectBkColor`(对应 `::selection`,切主题时同步)、`XEdit_SetCaretWidth`、`XEdit_EnableAutoSelAll(true)`(聚焦自动全选)、`XEdit_Undo/Redo` | 小 / 低-中 |
| I7 | 运行时 `SetDisabled` / `Disabled` | 与 Button / Select / InputGroup API 一致性 | 只有继承的 `Enable()` 与 `SetFieldState()`(后者会同时改 invalid) | 补一对方法,并同步禁用文字预混色(`syncTextColor`) | 极小 / 中 |
| I8 | 标签内联 Badge / Grid / Inline / Form 的 demo 对照 | 官方同页 4 个示例 | 组件能力齐全,缺同构示例卡 | 只补 demo 表单页与文档示例,不改组件 | 小 / 低(非功能缺口) |

**明确不做 / 能力边界**:

- RTL(排除)、Direction。
- `type=email/tel/url/date` 的移动端键盘与浏览器原生校验 UI(桌面无对应物;`search` / `number` / `password` 已由 `WithClearButton` / `WithStepper` / `WithPasswordToggle` 覆盖)。
- `name` / `form` / `autoComplete` / 原生表单提交(Go 无表单提交语义,布局与校验展示归 Field 系列)。
- 文字/光标/选区/IME 完全自绘(DESIGN 第七节能力边界);`::selection` 之外的富文本排版。

### 3.2 Select(#4)

**官方 base/select 文档页**:Usage / Composition / **Align Item With Trigger** / Groups / Scrollable / Disabled / Invalid / RTL(排除) / API Reference(指向 Base UI)。本地参照实现 `select.tsx` 是 base-nova 版本,暴露 `alignItemWithTrigger` 与 `SelectScrollUpButton/ScrollDownButton`,Root props 由 Base UI 提供。

**已具备(不必再列)**:触发区 / Value / Popup / Group / GroupLabel / Separator / Item / Check 的组合;Groups、Scrollable(ScrollArea 承载)、Disabled(整体 + `SelectItem.Disabled`)、Invalid;item-aligned 定位与选中项对位;打开即定位选中项;键盘 ↑↓ / Home / End / Enter / ESC(ESC 还焦点);`Value/SetValue/Selected/SetItems/SetGroups/OnChange/FitWidth/Open/Close/IsOpen`;`FieldControl` 联动;暗色触发区 hover。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| S1 | `alignItemWithTrigger=false` 与 `side/align/sideOffset/alignOffset` | 官方 Select#Align Item With Trigger 有开关示例;`select.tsx` 的 `SelectContent` 参数 | `itemAlignedPos()` 恒按 item-aligned 定位,无配置 | `WithSelectAlignItem(bool)`(默认 true 保持现状),false 走 `popoverPanel` 常规锚定(下方优先 / 翻转 / 夹工作区);side/align 可给枚举,几何已有 | 中 / 中-高 |
| S2 | typeahead 键入跳转 | Base UI 文档明示「aside from basic keyboard typeahead」;**DESIGN 第七节已写「后续有需求再加」** | `panel.OnKeyDown` 只处理 ↑↓/Home/End/Enter | 聚合可打印字符(约 500ms 窗口),按 `selectItemLabel` 前缀/包含匹配后 `setActive(..., true)`;Combobox 的过滤输入是另一语义,不算覆盖 | 小-中 / 中 |
| S3 | `OnOpenChange` 开合回调 | Base UI Root `onOpenChange`;Popover / Dialog / Combobox 本库已有同类回调 | 有 `Open/Close/IsOpen`,**无回调**;`popup.go` 的 `popoverPanel.OnHide` 注释已写明用于 `OnOpenChange(false)` | `Select.OnOpenChange(func(bool))`,在 Open / Hide / ESC / 失焦四条路径统一触发 | 极小 / **高** |
| S4 | `readOnly` 只读 | Base UI Root `readOnly`(默认 false:不能从弹层改选) | 只有 `SetDisabled`(整体置灰、打不开) | `WithSelectReadOnly/SetReadOnly/ReadOnly`:可打开查看与高亮,但 `choose` 不落值、触发区不置灰;与 Checkbox 的 readOnly 同语义 | 小 / 中 |
| S5 | `multiple` 多选 | Base UI Root `multiple`(默认 false)与官方 Multiple selection 示例;shadcn 文档页本身未给示例 | `Value()/SetValue()` 单值;多选能力在 Combobox | `WithSelectMultiple()` + `Values/SetValues` + 条目勾选切换且不关闭 + 触发区文本约定(首个 `+N` 或 chips);`OnChange` 需定新签名或另加 `OnValuesChange` | 大 / 中-低(桌面多选优先 Combobox,需先确认需求) |
| S6 | `highlightItemOnHover=false` | Base UI Root 默认 true | `paintItem` 用 `xui.Hover` 把 hover 与 focus 合成同一条高亮,无法区分 | 开关关闭时悬停只画淡底、不移动键盘高亮 | 小 / 低 |
| S7 | ScrollUpArrow / ScrollDownArrow 悬停自动滚 | 本地 `select.tsx` 明确导出两个部件;**DESIGN 第七节记为「没有 radix 的悬停自动滚按钮」** | ScrollArea 只有滚轮与键盘 reveal | 内容区上下叠 16px 命中带,只在对应方向还能滚时出现;按运行时 `xui.Hover` 的过渡值驱动逐帧滚动(**不自起定时器**,遵守运行时过渡约定) | 中 / 低-中 |
| S8 | `Select.Value` 自定义格式化 | Base UI `<Select.Value>` 可传函数;`items` 与 `itemToStringLabel/Value` | `SelectItem.Text`(空回落 `Value`)已覆盖 label 映射,但无自定义渲染 | `WithSelectValueText(func(SelectItem) string)` 或 `SetValueText(string)`,用于前缀/图标/组合文案 | 小 / 中 |
| S9 | `required` 字段级必填 | Base UI Root `required`;官方 Invalid 示例走 Field | Field 侧暂无校验入口(见 I3) | 与 I3 同一套 Field 校验基础设施,Select 通过 `FieldControl` 接入 | 小 / 中 |

**明确不做 / 能力边界**:

- RTL(排除)。
- 弹层 fade / zoom 进场(DESIGN 第七节:异型透明窗整窗 alpha 只认第一次调用,Select 明确不做淡入)。
- `name` / `form` / `autoComplete` / 隐藏 input 提交(Go 无表单提交);`modal=false`(桌面弹层是独立窗口,点外部失焦即关闭,「锁页面滚动 / 禁用外部交互」无等价需求);Arrow/Indicator(shadcn 默认示例不启用,桌面弹层无箭头语义)。
- `data-slot` / `data-highlighted` / `render` / `asChild` 等 DOM 概念(React 专属,Go 侧无对应物)。

### 3.3 Dialog / Card / Label(#6 / #7 / #8)

依据:官方 `base/dialog`、`base/card`、`base/label` 文档页(含 Card 的 API Reference 与 Spacing Variable Changelog)、Base UI Dialog Root props 表、本地 base-nova `dialog.tsx` / `card.tsx` / `label.tsx`、本库 `dialog.go` / `card.go` / `label.go`、DESIGN 第四节末映射表与第七节差异表。RTL 三页都有示例,按约定排除。

#### Dialog(#6)

**官方页**:Usage / Composition / Custom Close Button / No Close Button(`showCloseButton={false}`) / Sticky Footer / Scrollable Content / RTL(排除) / API Reference → Base UI。

**已具备**:`DialogTrigger`(BindTrigger)、`DialogContent` 面板与遮罩(`bg-black/10` + backdrop-blur-xs 的盒式模糊实现)、`DialogTitle/Description`(SetTitle/SetDescription)、`DialogFooter`(AddFooter / BindClose)、Custom Close Button(BindClose 任意控件)、No Close Button(`WithDialogCloseButton(false)`)、Sticky Footer 与 Scrollable Content(`WithDialogScroll` 头/页脚固定 + ScrollArea)、宽度(`WithDialogWidth`)、`Refresh`、ESC / 点遮罩关闭、打开自动聚焦首个可聚焦控件、Tab 焦点循环、模态栈(嵌套对话框)、100ms 淡入淡出。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| D1 | `open` 受控与 `onOpenChange` | Base UI Dialog Root `open` / `onOpenChange` | 有 `Show/Hide/Close/IsOpen` + `OnClose`,但**语义分裂**:`Hide()` 不触发回调、`Close()` 才触发,也没有打开回调 | `OnOpenChange(func(bool))` 统一四条开合路径(Show / Hide / ESC / 遮罩 / 关闭钮),`SetOpen(bool)` 作受控入口;保留 `OnClose` 兼容 | 小 / **高** |
| D2 | `disablePointerDismissal` + ESC 关闭开关 + 可否决关闭 | Base UI Root `disablePointerDismissal`(默认 false) | `dismissOutside` 字段**已存在**,但只在 `alertdialog.go` 内部置 false,普通 Dialog 不可配;ESC 恒关,关闭不可否决 | 把 `WithDialogDismissOutside(bool)` / `WithDialogEscClose(bool)` 导出,加 `OnClosing(func() bool)` 让「未保存内容」可拦下 ESC / 遮罩 / X | 极小 / **高** |
| D3 | `initialFocus` / `finalFocus` | Base UI Popup props(文档有专节示例) | `initialFocus` 字段**已存在**,但只给 AlertDialog 聚焦 Cancel 用;普通 Dialog 固定 `defaultFocus(fs)` 取首项,关闭后 `restoreModalFocus` 固定还原下层 | `WithDialogInitialFocus(hEle)` / `WithDialogFinalFocus(hEle)` 两个选项,复用现有 `XWnd_SetFocusEle` 与 `restoreModalFocus` | 极小 / 中-高 |
| D4 | `DialogHeader` / `DialogFooter` 任意内容 | 官方 Composition 与 base `dialog.tsx` 的容器部件 | 头部只有 `SetTitle/SetDescription` 两条文字,页脚只能 `AddFooter(text, variant, onClick)` 加按钮 | `Header()` / `FooterStack()` 返回容器,或 `Title()` / `Description()` 返回 `*Label` 供改字号/配色/图标 | 中 / 中 |
| D5 | 遮罩与面板样式可配 | `DialogOverlay` 可传 `className`;Popup 的 `p-4` / `gap-4` | 遮罩固定黑 10%、模糊固定盒式,面板内边距/组间距写死 | `WithDialogOverlayAlpha(float64)` / `WithDialogPadding(px)` / `WithDialogGap(px)` | 小-中 / 低-中 |
| D6 | `modal=false` / `'trap-focus'` | Base UI Root `modal` | 对话框恒为「覆盖窗口内容区 + 拦截下层鼠标 + Tab 焦点圈禁」的模态层 | 桌面下非模态 = 独立窗口,是另一套形态而非开关;若要提供需新建 window 载体 | 大 / 低(建议不做并写明理由) |

**明确不做 / 能力边界**:RTL;zoom-in-95 缩放进场(DESIGN 第七节:元素级缩放对自绘元素无效,已用淡入覆盖);DOM 概念(`handle` / `triggerId` / `actionsRef` / `children` render / `onOpenChangeComplete`);`aria-describedby` 自动关联;原生 CSS 层背景模糊(已用抓窗快照 + Go 盒式模糊替代,见 DESIGN 第八节 47)。

#### Card(#7)

**官方页**:Usage / Composition(CardHeader / Title / Description / Action / Content / Footer) / `size="sm"` / **Spacing(`--card-spacing` 16/20/24/32)** / 负边距通栏内容 / Image(首图通栏、自动圆角) / RTL(排除) / API Reference / Spacing Variable Changelog。

**已具备**:`size` 档(`WithCardSize(CardSizeSm)` = 16→12)、`CardHeader/Title/Description/Action`(`NewCardHeader` + `TitleStack` / `ActionStack`)、`CardContent`(`ContentStack`)、`CardFooter`(`Footer` / `NewCardFooter` + `WithCardFooterPadding`,含 `border-t` + `bg-muted/50` + 下两角圆角)、`ring-1 ring-foreground/10`、`TitleSize()`、Auto 高自撑、元素 padding 表达左右内距(DESIGN 8.129 硬规则)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| C1 | `--card-spacing` 任意值 | 官方 Spacing 示例(16/20/24/32)+ Changelog 明说「用 `--card-spacing` 控制段间距与内缩」 | 只有 `WithCardSize` 两档;`Card.Padding` 是导出字段,但**直接改不会同步** `ContentStack` 的 margin/padding 与 `Footer` 内边距 | `WithCardSpacing(px)` / `SetSpacing(px)`:同步刷新内容区与页脚并重排;保留 `WithCardSize` 作为预设 | 小-中 / **高** |
| C2 | 有页脚时去掉卡片底部内距 | `card.tsx` 的 `has-data-[slot=card-footer]:pb-0` | `ContentStack` 上下外边距恒为 `Padding`,`Footer()` 创建页脚后未改 → 内容区与页脚之间多出一整段内距 | `Footer()` 里把内容区下外边距置 0 后重排(上下内距用布局项外边距表达,符合硬规则) | 极小 / **高** |
| C3 | edge-to-edge 通栏内容 | 官方 Spacing 第二例用 `-mx-(--card-spacing)` / `-mb-(--card-spacing)` 让分隔线/内容贴到卡片边 | 无对应物;内容区一律受左右 padding 约束 | `ContentFullBleed()` 或 `AddFullBleed(child)`:**不要照搬负外边距**(DESIGN 8.129),改为单独一层不受元素 padding 约束的通栏容器 | 中 / 中 |
| C4 | 首图 / 末图自动圆角 + 卡片图片槽 | 官方 Image 示例 + `*: [img:first-child]:rounded-t-xl` | 无图片槽;DESIGN 第七节记为「xcgui 没有圆角裁剪语义」的能力边界 | **需重新评估**:`Avatar` 的 `SetImageData` 已在 CPU 侧用 RGBA 蒙版做过圆角裁剪,组件级「图片按卡片上/下两角裁剪」同理可实现(或提供 `CardImage` 助手)。若仍判不做,需在 DESIGN 第七节写明与 Avatar 的差异理由 | 中 / 中 |
| C5 | header 的容器查询响应式 | `card.tsx` 的 `@container/card-header`(窄卡时动作区栅格自适应) | 头部栅格固定一行两列 | 在尺寸事件里按卡片宽度切 `ActionStack` 排布;桌面窗口宽度稳定,收益有限 | 中 / 低 |

**明确不做 / 能力边界**:RTL;CSS 变量与 `className` / `data-slot` / `group-data-[size]` 选择器(Go 无 DOM,以选项 API 表达);背景 `overflow-hidden` 语义(DESIGN 第七节,但 C4 的图片裁剪不受此限)。

#### Label(#8)

**官方页**:Usage(`<Label htmlFor="email">`)、Checkbox 组合、Label in Field(FieldLabel/FieldDescription/FieldError)、RTL(排除)、API Reference → Base UI Label;`label.tsx` 是原生 `<label>` + `group-data-[disabled]` / `peer-disabled` 变淡样式。

**已具备**:文字/弱化色/粗体/中等字重/字号/对齐(Left、Center)/折行(`SetWrap` / `SetAutoWrap` + `WrappedHeight`)、实时取令牌(`SetColorFunc` + `OnThemeChangeFor`)、省略号截断、不参与 Tab 焦点、Field 内的标签/说明统一变淡与校验变红(`field.go` 的 `applyLabelColor`)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| L1 | `htmlFor` 点击关联 | 官方 Usage 全文就是 `htmlFor`;Base UI Label 继承原生 `<label>` 行为 | Label 载体是「不响应点击」的原生 Button,**没有任何关联能力**;Field 也没有点击标签聚焦控件 | `BindFor(hEle)`:点击时 `xui.FocusElement` 聚焦;对实现 `Toggle()` 的控件(Checkbox / Switch)顺带切换,对齐浏览器 `<label for>` 语义;Field 的 `FieldLabel` 同步走这条路径 | 小-中 / **高** |
| L2 | 独立 Label 的禁用 / 校验态 | `label.tsx` 的 `group-data-[disabled]:opacity-50` 与 `peer-disabled:opacity-50` | 只有 Field 内部 `applyLabelColor` 会联动,独立 Label 无 `SetDisabled` / `SetInvalid` | 复用 Field 的配色逻辑补 `SetDisabled(bool)` / `SetInvalid(bool)` 及对应 getter + 禁用光标 | 小 / 中 |
| L3 | `required` 标记 | 官方 Field 示例里的必填星号(与 Input 的 Required 同源) | 无 | 与 I3 / S9 同一套 Field 校验基础设施一起做:`WithLabelRequired()` 或 Field 层统一 | 小 / 中 |
| L4 | 右对齐 | 上游 label 在行尾/表单行右对齐常见;本库只有 Left / Center | 无 `SetAlignRight()` | 补一个对齐方法(与 `SetAlignCenter` 同一处) | 极小 / 低 |

**明确不做 / 能力边界**:RTL;`htmlFor` 的字符串 ID 与 `aria-*` 元数据(Go 无 DOM,以元素句柄绑定替代,见 L1);富文本内联链接与 `*: [a]:underline` 样式(Label 是原生 Button 载体);`font-medium` 在中文下回落常规字重(源码注释已说明,可用 `SetMedium` 手动覆盖)。

**本批实现顺序建议**:C2 → C1 → D1 → D2 → L1 → D3(小改动、对齐收益明显);随后 C3 / C4 / L2 / L3 / D4;C5 / L4 / D5 低优先,D6 建议不做。C4 需先决定 DESIGN 第七节那条能力边界是否修订。

### 3.4 Textarea / Tabs / Switch / Radio Group / Dropdown Menu(#9 ~ #13)

依据:官方 `base/textarea`、`base/tabs`、`base/switch`、`base/radio-group`、`base/dropdown-menu` 文档页(含 Dropdown Menu 的 Composition 与 11 个示例)、Base UI Tabs / Menu / Switch / Radio Root props 表、本地 base-nova 五份 `*.tsx`、本库 `textarea.go` / `tabs.go` / `switch.go` / `radio.go` / `radiogroup.go` / `menu.go` / `menu_model.go`、DESIGN 第四节末映射表与第七节差异表。RTL 五页都有示例,按约定排除。

> **共用实现提示**:Textarea 内嵌 `*Input`(`textarea.go`),Input 侧的 I2 / I3 / I4 / I5 / I6 / I7 一旦落地,Textarea 自动获得;Dropdown Menu 与 #56 Context Menu 共用 `menu.go`,本节的结论可直接复用,只需另核右键触发特有的行为。

#### Textarea(#9)

**官方页**:Basic / Field / Disabled / Invalid / Button / RTL(排除)。**没有 API Reference** —— Base UI 也未提供独立的 textarea 组件(取 `base-ui.com/react/components/textarea.md` 失败),tsx 就是原生 `<textarea>` 加一层样式壳,因此属性全部来自 HTML 标准。

**已具备**:`field-sizing-content` 自动长高(最小 64、上下内边距 8、内容装得下故不挂滚动条,DESIGN 8.166)、Field 组合、Disabled / Invalid(继承 `WithInputInvalid` / `SetInvalid` / `SetFieldState`)、焦点环与暗色、`SetValue` / `Refresh` / `OnChange`、Button 组合(调用方拼)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TA1 | 与 Input 同源的 6 项 | 原生 `<textarea>` 属性(readOnly / required / maxLength / pattern 等) | Textarea 内嵌 `*Input`,但 Input 侧 I2 / I3 / I4 / I5 / I6 / I7 都还没做 | **在 `input.go` 补一次即同时生效**,不必为 Textarea 单独写一遍 | 见第三节 / **高** |
| TA2 | `resize` 拖拽调整高度 | 浏览器 `<textarea>` 默认右下角可拖拽 | 自动长高 + 不挂滚动条,没有 resize 抓手 | 右下角抓手 + 拖拽改 `minH`(或切到固定高度 + 滚动条形态),与自动长高互斥 | 中 / 低-中 |
| TA3 | `rows` 初始行数 | 原生 `rows` 属性 | `cy` 目前表达的是最小高度 | `WithTextareaRows(n)` 按行高换算初始高度;与 `cy` 二选一 | 小 / 低 |
| TA4 | Button 组合示例 | 官方 Button 示例(文本域 + 发送钮) | 组件能拼,缺 demo 对照卡 | 只补 demo | 小 / 低(非功能缺口) |

**明确不做**:RTL;固定高度 + 溢出滚动(已有替代方案 `Input + WithMultiline`,DESIGN 437);`cols` / `wrap` 的浏览器细节(Edit 折行已覆盖);`name` / `form` 表单提交。

#### Tabs(#10)

**官方页**:Usage / Composition / Line / Vertical / Disabled / Icons / RTL(排除) / API → Base UI Tabs。Root props 含 `defaultValue` / `value` / `onValueChange` / `orientation`,并明确 **`value=null` 时无选中项**、`onValueChange` 的 `eventDetails.cancel()` 可取消用户切换、选中项被禁用或移除时有自动回退(reason=`disabled` / `missing`)。

**已具备**:default / line 变体、vertical、换行(扩展)、单标签禁用、行内图标、受控 index / value(`SetActive` / `SetActiveValue` / `OnValueChange` / `WithTabsValues`)、`BindPanel` 面板显隐、`activationMode`、整块一个 Tab 站点 + ←/→/↑/↓/Home/End/Enter/空格、**focus-visible 环**(`onPaintTab` 已画)、`forceMount` 语义(Go 侧元素常驻不卸载,天然等价)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TB1 | 可否决切换 | Root `onValueChange` 的 `eventDetails.cancel()` | `OnValueChange` 只注册通知,`onClick` 直接换 active | `OnValueChangeVeto(func(value string) bool)`(或让回调返回 bool),在 `SetActive` 的用户路径上判定;程序化 `SetActive` 不经过否决 | 小 / 中-高(未保存内容禁止切页) |
| TB2 | `value=null` 无选中态 | Root props 明确 | `SetActive(i)` 要求 `i>=0`,`ActiveValue()` 无 values 时返回空串,但没有"无选中"状态 | `ClearActive()` / 允许 `active=-1`,`syncPanels` 全部隐藏;受控值用空串表达 | 小 / 中 |
| TB3 | 选中项被禁用 / 移除时自动回退 | Root 的 `disabled` / `missing` 自动 fallback | `SetDisabled` 只迁移 `focus`,**`active` 保持不变**(注释写明"禁用项也可程序化选中") | 在 `SetDisabled` / `SetItems` 里按策略落到 `nearestEnabled`(或保留受控值,由选项决定) | 小 / 中 |
| TB4 | 活动指示条滑动过渡 | DESIGN 427 记「未做拖动滑块式指示器」 | 选中项高亮/下划线直接切换 | 用运行时 `xui.PresenceOnce`(每次切换新键)驱动指示条位置,不自起定时器 | 中 / 低-中 |

**明确不做**:RTL;CSS 选择器 / `render` / `data-activation-direction`;事件 reason 枚举(DOM 概念)。

#### Switch(#11)

**官方页**:Basic / Description / Choice Card / Disabled / Invalid / **Size** / RTL(排除) / API → Base UI Switch Root(`checked` / `defaultChecked` / `onCheckedChange` / `disabled` / **`readOnly`** / `required` / `name` / `value` / `uncheckedValue` / `form`)。

**已具备**:default 32×18 / sm 24×14 两档(与 tsx 数值一致)、`WithSwitchChecked`(defaultChecked)、disabled、invalid、空格切换、FieldControl 联动、`SetFocusRingProxy`、**热区外扩 `switchRingPad=3`**(注释明确等效 `after:-inset-x-3 -inset-y-2`)、hover 配色与 focus-visible 环、`Checked/SetChecked/Toggle/OnChange`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SW1 | `readOnly` | Root 默认 false(「unable to activate or deactivate」) | 无 `SetReadOnly/ReadOnly`;只能 `SetDisabled`(会灰掉、丢焦点) | 复用 Checkbox 已落地的做法:原生 Check 按钮在事件前已改勾选位,在 `XE_BUTTON_CHECK` 里 `SetCheck(!bCheck)` 回滚 + `Redraw`(DESIGN 8.198);保留焦点与键盘可达 | 小 / **高** |
| SW2 | 可否决 `onCheckedChange` | Root `onCheckedChange` 附 `eventDetails.cancel()` | `OnChange` 只通知,`Toggle` / 点击 / 空格各自直接改状态 | 把三条路径收敛到一个可否决入口(同 Checkbox 的 `OnCheckedChange(func(bool) bool)`),程序化 `SetChecked` 不经过否决 | 小 / 中-高 |
| SW3 | 滑块位移 / 轨道色过渡 | tsx 的 `transition-all` 与 thumb `transition-transform` | `onPaint` 按 `checked` 直接画终点位置,没有位移过渡 | 用运行时过渡表(每次切换 `xui.PresenceOnce` 取新键)插值 thumb 的 x,配色继续用 `xui.Fade`;不要组件自起定时器 | 中 / 中 |

**其余**:`required` 归 Field 层(I3 / S9 / RG3 同一套);Choice Card 整卡可点依赖 L1(Label 点击关联)。**明确不做**:RTL;`name` / `value` / `uncheckedValue` / `form` 表单提交;`nativeButton` / `render`;自定义 thumb children(Go 无 Slot)。

#### Radio Group(#12)

**官方页**:Basic / Composition / Description / Choice Card / Fieldset / Disabled / Invalid / RTL(排除) / API → Base UI Radio Root(`value` / `disabled` / **`readOnly`** / `required`)。

**已具备**:组值 `Value/SetValue/OnChange`、组级 `SetDisabled` / `SetInvalid` / `SetFocusRingProxy` / `SetSpacing`、`AddItem` / `Attach` 跨父入组、方向键循环 + Home/End + 空格、条目 invalid/disabled/focus 环、FieldSet / FieldLegend、Choice Card(FieldLabel 代理焦点环)、竖向 gap-2。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| RG1 | `readOnly` | Radio Root 明确有 `readOnly`(「unable to select the radio button」) | 组与条目都没有 `SetReadOnly/ReadOnly` | 组级开关 + 条目在 `XE_BUTTON_CHECK` 里回滚(同 SW1 / Checkbox),键盘方向键走同一判定 | 小 / **高** |
| RG2 | 可否决组值变化 | 上游 `onValueChange` 的 cancel 语义 | `OnChange(fn func(string))` 只通知;`onItemSelected` 直接改组值 | `OnValueChangeVeto(func(value string) bool)`,用于"未保存内容禁止换选项" | 小 / 中-高 |
| RG3 | `required` | Radio Root `required` | 无 | 与 I3 / S9 / L3 共用 Field 校验基础设施 | 小 / 中 |
| RG4 | 横向排列 | 上游默认 `grid gap-2` 竖向,官方无横向示例 | 只有竖向 | `WithRadioGroupHorizontal()`(扩展),成本小但非上游特性 | 小 / 低 |

**明确不做**:RTL;roving tabindex(DESIGN 307 已记桌面惯例差异:Tab 遍历各项);`name` / `form` 表单提交。

#### Dropdown Menu(#13)

**官方页**:Usage / Composition(Trigger / Content / Group / Label / Item / CheckboxItem / RadioGroup / RadioItem / Separator / Shortcut / Sub / SubTrigger / SubContent)+ 11 个示例(Basic / Submenu / Shortcuts / Icons / Checkboxes / Checkboxes Icons / Radio Group / Radio Icons / Destructive / Avatar / Complex)+ RTL(排除)+ API → Base UI Menu。Root props 含 `open` / **`onOpenChange`** / `highlightItemOnHover` / `closeParentOnEsc` / `loopFocus` / `modal` / `orientation` / `disabled` / `onItemHighlighted`;`Content` 还有 `align` / `alignOffset` / `side` / `sideOffset`(默认 align=start、side=bottom、sideOffset=4)。

**已具备**:Composition 的全部部件(`Item/Check/Radio/Label/Separator/Sub` + `MenuShortcut/MenuDisabled/MenuDestructive/MenuGroup/MenuInset/MenuIcon/MenuOnSelect/MenuOnCheck`)、三种弹出方式(`PopupAtElement` / `PopupAtElementRect` / `PopupAtCursor`)、下方放不下翻转 + 屏幕夹紧、`sideOffset=4` 与 `min-w-32=128`、显式 `Width`、自绘背景/条目/图标/勾选/快捷键、`ContextMenu` 的 `SetEnabled` / **`OnOpenChange`**。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| DM1 | `OnOpenChange` 普通菜单 | Root `open` / `onOpenChange` | **只有 `ContextMenu` 有**,`Menu` 没有;而 `AddEvent_Menu_PopupWnd` / `AddEvent_Menu_Exit` 已在用(DESIGN 826) | `Menu.OnOpenChange(func(bool))`,弹层创建/退出各发一次;与 ContextMenu 口径统一 | 小 / **高** |
| DM2 | `side` / `align` / `sideOffset` / `alignOffset` | Content 的默认定位参数 | 固定「锚点下方 + 翻转 + 夹紧」,`menuSideOffset` 是常量 | 把定位参数收进 `Menu` 选项,`menuOnPopupWnd` 里按 side/align 算落位(现有翻转/夹紧逻辑复用) | 中 / 中-高(靠边按钮、右键菜单常需) |
| DM3 | 受控 `RadioGroup` 组值 | `DropdownMenuRadioGroup` 的 value / onValueChange | 靠 `MenuGroup(n)` 手动分组 + `Radio(text, checked)` + `MenuOnCheck`,没有组值 API | `RadioGroupValue/SetRadioGroupValue/OnRadioGroupChange`,按 `group` 收敛;菜单是一次性对象,状态仍由调用方持有 | 小-中 / 中 |
| DM4 | Root 级小项 | `disabled` / `highlightItemOnHover` / `onItemHighlighted` / `loopFocus` / `closeParentOnEsc` | 只有条目级 `MenuDisabled`;整菜单禁用、悬停开关、高亮回调、焦点循环、子菜单 ESC 层级都不可配 | 弹层事件 + `menuDrawItem` 已具备,逐项加选项/回调 | 小-中 / 低-中 |
| DM5 | Avatar 触发区 / Complex 组合示例 | 官方两个示例 | 部件齐全,缺 demo 对照 | 只补 demo | 小 / 低(非功能缺口) |

**明确不做**:RTL;`modal=false` 与 `orientation=horizontal`(原生菜单恒模态/竖向,要改成自绘弹层成本高,Menubar 已有自绘实现但不建议为 Menu 重做);`handle` / `triggerId` / `actionsRef` / `render` 等 DOM 概念;弹层 zoom/slide 动画(DESIGN 第七节能力边界)。原生菜单超高滚动由系统处理,若实测缺失再单列。

**本批实现顺序建议**:SW1 → RG1 → DM1 → TA1(随 Input 一起) → TB1 → SW2 / RG2 / TB2 / TB3 / DM2 / DM3;其后 SW3 / TB4 / DM4 / TA2 / TA3。SW1 / RG1 与 Checkbox 的 readOnly 是同一套原生事件回滚做法,建议三处一起改并一起验收;TB1 / RG2 / SW2 的「可否决回调」也宜统一 API 形态。

### 3.5 Tooltip / Badge / Alert / Table / Data Table(#14 ~ #18)

依据:官方 `base/tooltip`、`base/badge`、`base/alert`、`base/table`、`base/data-table` 文档页(含 Badge / Alert 的 API Reference、Data Table 的 TanStack v9 指南目录)、本地 base-nova `tooltip.tsx` / `badge.tsx` / `table.tsx`(Alert 与 Data Table 无独立 tsx,按 HANDOFF 待办记录)、本库 `tooltip.go` / `separator.go` / `alert.go` / `table.go` / `datatable.go`、DESIGN 第四节末映射表与第七节差异表。RTL 五页都有示例(或阿拉伯语示例),按约定排除。

> **本批特点**:五个组件的主体能力都已对齐,缺口集中在少量 API 形态与配置开关;Tooltip / Badge / Table 甚至已有上游文档未明确列出的能力(键盘焦点触发、行内转圈、行 hover 过渡)。

#### Tooltip(#14)

**官方页**:Installation 明确要挂 `TooltipProvider`(delay=0)、Usage / Composition、Side(left/top/bottom/right)、With Keyboard Shortcut、Disabled Button(用 span 包禁用按钮)、RTL(排除)、API → Base UI Tooltip。本地 `tooltip.tsx` 的 Content 还有 `align` / `alignOffset` / `side` / `sideOffset` 与 `TooltipPrimitive.Arrow`。

**已具备**:`WithTipSide`(四方向 + 自动翻转夹紧)、`WithTipAlign`、`WithTipOffset`(sideOffset)、`WithTipDelay`、全局 `SetDelay` / `SetSkipDelay`(Provider 的共享延迟)、`WithTipShortcut`(内容右侧 Kbd,对应官方快捷键示例)、`WithTipMaxWidth`、受控 `Show` / `Hide` / `IsOpen`、**键盘焦点触发**(`AddEvent_SetFocus` + `FocusedByKeyboard`)、自绘箭头(`tipArrow`)、淡入淡出/滑动、一个管理器绑定任意多个元素、`Unbind`、宿主窗口销毁清理。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TT1 | `disableHoverableContent` | Base UI Tooltip Provider/Root;**DESIGN 428 明确记为「未做」** | 鼠标移入提示气泡即按离开锚点处理 | 弹层自身挂 MouseEnter/Leave,开时保持显示;做成 `WithTipHoverable(bool)` 或全局开关 | 小-中 / 中 |
| TT2 | `arrowPadding` / 箭头避开圆角 | DESIGN 428 记为「未做 arrowPadding 单独配置」 | `clampArrow(v, span)` 只保证三角形完整落在气泡内,把值夹在 `[tipArrow, span-tipArrow]`,不按圆角半径让位 | 给 `clampArrow` 加 padding 参数并暴露 `WithTipArrowPadding(px)` | 小 / 中 |
| TT3 | TooltipContent 放任意控件 | 官方 `<TooltipContent><p>…</p></TooltipContent>` 与 tsx 的 `children` | 弹层是 FloatWindow 自绘,只有文本 + Kbd 块 | `BindTipContent(hEle, build func(parent int))`:内容区改成可挂控件的容器,按内容量高(参考 ScreenPopup.FitContent) | 中 / 低-中 |

**低优先小项**:`alignOffset`(tsx 默认 0,本库 `tipConfig` 无该字段)、禁用按钮上的提示(官方用 span 包裹;本库可绑到父元素或提供矩形绑定)。**明确不做**:RTL;zoom 进场(异型透明窗整窗 alpha 冻结,DESIGN 8.57);`render` / `data-slot` 等 DOM 概念。

#### Badge(#15)

**官方页**:6 个 variant、With Icon(`data-icon=inline-start/end`)、With Spinner、Link(`render` 成 `<a>`)、Custom Colors、RTL(排除)、API Reference(6 变体 + className)。

**已具备**:六档变体、`WithBadgeLeadingIcon` / `WithBadgeTrailingIcon`、`WithBadgeLoading` / `SetLoading`(转圈)、`WithBadgeColors` / `SetColors`(自定义配色,对应 Custom Colors)、`WithBadgeOnClick` / `OnClick`、`SetText`、`WithVariant`、`link` 变体 hover 下划线、hover 过渡(`xui.Hover` + `Fade`)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| BA1 | `focus-visible` 环与 `aria-invalid` 校验环 | `badge.tsx` 的 `focus-visible:ring-[3px]` / `aria-invalid:border-destructive` | Badge 是纯 Element,没有焦点环也没有 `SetInvalid` | 可点击形态启用焦点 + 手绘环(同 Button);补 `WithBadgeInvalid` / `SetInvalid(bool)` 画 destructive 环 | 小-中 / 中 |
| BA2 | 行内图标侧内距收窄 | `badge.tsx` 的 `has-data-[icon=inline-start]:pl-1.5` / `pr-1.5` | `naturalWidth` 用固定 `badgePadX*2` 再加图标宽,**有图标时那一侧没有收窄** | 按 `leading` / `trailing` / `loading` 把对应侧内距从 `padX` 换成 6(同 Button 412 的做法) | 极小 / 中 |
| BA3 | Link 变体真正打开链接 | 官方 Link 示例用 `render` 渲染 `<a>` | link 只是样式,`OnClick` 要调用方自己写 ShellExecute | `NewLinkBadge(...)` / `SetLinkURL(url)`,复用 Button 的 `NewLinkButton` 逻辑 | 小 / 中 |

**明确不做**:RTL;`[a]:hover:*` 只有包在链接里才生效的条件(DESIGN 415 已记差异:Go 侧无链接概念,悬停恒生效);`render` / `className`。

#### Alert(#16)

**官方页**:Composition(Icon / Title / Description / Action)、Basic、Destructive、Action、Custom Colors、RTL(排除)、API Reference(variant = default / destructive + 三个子组件)。

**已具备**:`WithAlertVariant` / `SetVariant`、`WithAlertIcon` / `SetIcon`、`WithAlertTitle` / `SetTitle`、`WithAlertDescription` / `SetDescription`、`Action()` 动作槽、`Title()` / `Description()` 返回 `*Label` 可改字体、`Refresh()`、按宽度量高后写回标题/描述高度(`alertTextHeight` / `setTextHeight`)、整体 Auto 高自撑、破坏性配色。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| AL1 | 自定义配色 | 官方 Custom Colors 示例 | `foreground(t)` / `descriptionColor(t)` 是内部按 variant 取令牌,没有公开配色入口 | `WithAlertColors(bg, border, fore, desc)` / `SetColors`,复用 `xui.Hex` 与 Badge 的 `SetColors` 口径,主题切换仍现取 | 小 / 中 |
| AL2 | 描述多行折行 | 官方描述段落可多行;DESIGN 566 记 Field/Empty 描述默认单行,Label 用 `SetAutoWrap()` 折行 | `Description()` 返回 `*Label`,但 `reflow()` 里 `setTextHeight` 按 `alertTextHeight` 写回高度,是否与 `SetAutoWrap()` 协同**需实测**;不同步则折行被覆盖 | 把 `alertTextHeight` 接到 Label 的折行高度(`WrappedHeight`),或让 `Refresh()` 尊重 `autoWrap` 的自撑高度 | 小-中 / 中 |

**已知差异 / 明确不做**:RTL;动作槽按实际宽度缩短文本列 vs 上游固定 `pr-18=72`(DESIGN 420 已记,如需严格对齐可加 `WithAlertActionFixed(bool)`,低优先);`role=alert` / `aria-*` 无障碍元数据;可关闭(dismissible)上游未提供。

#### Table(#17)

**官方页**:Composition(`TableCaption` / `TableHeader` / `TableBody` / `TableFooter` / `TableRow` / `TableHead` / `TableCell`)、Footer 示例、Actions 示例(每行 `<DropdownMenu />`)、Data Table 组合指引、RTL(排除)。

**已具备**:原生 List 载体 + 表头/行底/边框/滚动条全自绘、`TableColumn.Align` 文本对齐、**行 hover `bg-muted/50` 过渡与选中行 `bg-muted`**(`onDrawItem` 已有)、表头悬停/拖拽换位、`SetColumns` / `FitColumns` / 自适应列宽、虚拟行 `setVirtualRows`、`AddRow` / `SetCell` / `SetRow` / `CellText` / `Clear`、`Select` / `OnSelect`、`HeaderCellRect` / `RowHeight`、内部单元格扩展绘制钩子 `onCellExtra`(DESIGN 8.167)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TBL1 | `TableCaption` | Composition 与 `caption-bottom` | 无标题/说明槽 | 表格下方一行弱化 Label,宽度跟随表格,计入 Auto 高度;提供 `SetCaption(text)` | 小-中 / 中 |
| TBL2 | `TableFooter` / 合计行 | 官方 Footer 示例(`border-t bg-muted/50`) | 无页脚行 | 在表格底部自绘一条固定合计行(或追加一个不参与数据下标的特殊行),配 `SetFooter(cells...)` | 中 / 中 |
| TBL3 | 单元格任意控件与点击回调 | 官方 Actions 示例每行放菜单按钮 | 只有**内部** `onCellExtra` 自绘钩子,没有公开的单元格渲染/点击/右键 API;DataTable 用 `Render` + `Menu.PopupAtElementRect` 做过类似事 | 暴露 `OnCellExtra(...)` 与 `OnCellClick(row,col)` / `OnCellMenu(...)`,配合 `HeaderCellRect` + 单元格矩形做命中 | 中 / 中 |

**低优先扩展**:列宽拖拽调整(表头 MouseDown/Move 已有,但当前只用于换列顺序;TanStack 有 column sizing,官方 Table 页未列,不计入本批清单)。**明确不做**:RTL;HTML `<colgroup>` / `colSpan` / `rowSpan` 语义(原生 List 无对应物);`data-state=selected` 之外的 DOM 状态钩子。

#### Data Table(#18)

**官方页**:定位是「用 TanStack Table v9 + `<Table />` 自己组合」的指南,**不是可安装组件**;目录覆盖 Set up Table Features / Basic Table / Cell Formatting / Row Actions / Pagination / Sorting / Filtering / Visibility / Row Selection / Reusable Components。示例里有「No results.」空态行、金额格式化/右对齐、页脚 `0 of 5 row(s) selected.`、分页 Previous/Next。

**已具备**:本库把它收敛成 `NewDataTable`(工具栏 / 表格 / 页脚三段)—— 排序(单列 + 多列键)、关键字筛选与分面筛选、分页与虚拟滚动、列显隐、行选择(含跨页全选)、单元格格式化(`DataColumn.Render` / `Align`)、行/单元格点击与双击原地编辑、表头菜单与拖拽换位、**空状态 `paintEmpty`**、页脚统计、自定义比较器 `Compare`。比官方指南还多了编辑、虚拟滚动、分面筛选与多列排序。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| DT1 | 服务端 / 手动分页 | 官方 Pagination 节提示「implementing manual pagination」 | `TotalRows()` = `len(rows)`,排序/筛选/分页全在本地内存数据上 | `WithDataTableRemote(total, fetch)` 或 `SetTotalRows` + `OnPageChange` 让调用方按页拉取;排序/筛选参数一并回传,避免组件再对整表排序 | 中-高 / 中(大数据量刚需) |
| DT2 | 自定义空态文案 / 内容 | 官方 `No results.` 空态行 | `paintEmpty` 固定文案与样式 | `WithDataTableEmptyText(s)` 或 `SetEmptyContent(build)`;成本小 | 小 / 中 |
| DT3 | 列宽拖拽调整 | TanStack column sizing(官方指南未列,但桌面表格常见);DESIGN 596 记表头拖拽只换显示顺序 | 表头拖拽只重排 `order`,不能改列宽 | 表头命中列缝时切到 resize 模式,调 `Table.SetColumnWidth` 并刷新;与现有换列拖拽互斥 | 中 / 低-中(需先确认需求) |

**明确不做**:RTL;行拖拽(DESIGN 595 已充分说明:排序/筛选/分页后显示顺序 ≠ 原始下标,落点语义属于业务);表头键盘排序(DESIGN 598:表头是原生 List 子元素,不进 Tab 链);中文拼音排序(DESIGN 597,用 `DataColumn.Compare` 覆盖);行展开 / 分组 / 聚合(官方指南未列)。

**本批实现顺序建议**:BA2 → BA3 → BA1 → TT2 → AL1 → TBL1 → DT2(小改动);随后 TT1 / AL2 / TBL2 / TBL3 / DT1;TT3 与 DT3 需要先确认需求。TT1 / TT2 是 DESIGN 428 里已白纸黑字记为「未做」的两项,若决定补齐,应同步更新第四节末映射表与第七节差异描述。

### 3.6 Popover / Progress / Field / Input Group / Avatar(#19 ~ #23)

依据:官方 base 五页(Field / Input Group 页自带 API Reference 表)、Base UI Popover / Progress / Avatar 的 Root·Positioner·Popup props、本地 base-nova `popover.tsx` / `progress.tsx` / `input-group.tsx` / `avatar.tsx`(本批本地参考实现**没有 `field.tsx`**,Field 以官方 API Reference 表为准)、`components/*.go`、DESIGN 第四节末语义表与第七节差异表。**只出结论,未改组件代码。**

#### Popover(#19)

**官方页**:Usage / Composition(`Popover` + `PopoverTrigger` + `PopoverContent`)、Basic、Align(start/center/end)、With Form、RTL(排除)、API Reference → Base UI Popover。`popover.tsx` 从 Positioner Props 里 Pick 了 `align` / `alignOffset` / `side` / `sideOffset`,默认 `center` / `0` / `bottom` / `4`。

**已具备**(本批最完整的一个):`WithPopoverSide` / `WithPopoverAlign` / `WithPopoverGap`(= sideOffset)、四方向自动翻转与工作区夹取(`popupPosition`)、`WithPopoverWidth/Height/Padding`、`WithPopoverOnOpenChange`(Open 通知 true、panel.OnHide 通知 false)、受控式 `Open/Close/SetOpen/IsOpen`、`WithPopoverKeepOnBlur`、ESC 关闭并还原触发区焦点、`PopoverHeader/Title/Description`、`Refresh` / `SetContentHeight` 打开中改高度、内容任意构建回调。Select S3 / Dialog D1 / Menu DM1 都在找的 `OnOpenChange`,这里**已经有了**。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| PO1 | `alignOffset` 交叉轴偏移 | `popover.tsx` 明确 Pick `alignOffset`;Base UI Positioner 默认 0 | `popupPosition(... gap int32)` 只有单一边距,`WithPopoverGap` 同时充当 sideOffset | 给 `popupPosition` 加 `alignOffset` 参数(带符号沿交叉轴平移),新增 `WithPopoverAlignOffset`;与 Menu DM2 共用一套定位参数 | 小 / 高 |
| PO2 | `initialFocus` / `finalFocus` | Base UI Popup Props 明确列出(默认开时聚焦首个可聚焦项、关时回触发区) | 无打开焦点管理;关闭后固定 `SetFocus(trigger)` 写在 `panel.OnEsc` 里,点外部关闭不还原 | 仿 Dialog D3(已有 `initialFocus` 字段)导出 `WithPopoverInitialFocus(hEle)` / `WithPopoverFinalFocus(hEle)`,打开后 `SetFocus`、所有关闭路径统一还原 | 小-中 / 高 |
| PO3 | 非 Button 触发区的键盘开合 + `defaultOpen` / 受控锁定 | Base UI Root `open` / `defaultOpen`;Trigger 支持键盘 | 非 `XC_BUTTON` 只挂 `LButtonUp`,空格/回车无效;`toggle()` 无条件开合,无「受控」状态锁 | 普通元素补 `AddEvent_KeyDown` 接管空格/回车;`WithPopoverDefaultOpen()`、`WithPopoverControlled(true)` 让 toggle 只发 `OnOpenChange`、由调用方 `SetOpen` 决定 | 小-中 / 中 |
| PO4 | 定位策略可配:`anchor` / `collisionAvoidance` / `collisionPadding` / `sticky` | Base UI Positioner Props(默认 flip + clipping-ancestors + padding 5) | 翻转与夹取硬编码在 `popupPosition`,不能换锚点、不能只 shift 不 flip、不能跟随锚点滚动 | 把现有翻转/夹取抽成策略枚举,`WithPopoverCollision(...)`;锚点另给 `WithPopoverAnchor(hEle)`(虚拟锚点不做) | 中 / 中(需确认需求) |
| PO5 | `Popover.Arrow` / `arrowPadding` | Base UI 有 Arrow 部件与 `arrowPadding=5`;shadcn 封装未暴露、官方页无示例 | 无箭头;Tooltip 已有自绘三角形与 `clampArrow` | 复用 Tooltip 的箭头绘制,挂在弹层边缘;与 Tooltip TT2 的 `arrowPadding` 一起做 | 中 / 低-中(扩展) |
| PO6 | `modal` / `'trap-focus'` 与内容内 `Popover.Close` | Base UI Root `modal`;Close 部件 | 恒为非模态(点宿主窗口即关),内容里没有一键关闭的现成部件 | 模态可复用 Dialog 的 `modalStack` 与焦点圈禁;Close 只需导出一个 `PopoverClose(p, text)` 便捷构造 | 中-高 / 低(桌面非模态更符合习惯,建议确认后再做) |

**已知差异 / 明确不做**:RTL;进场 zoom/slide(DESIGN 573 能力边界);`render` / `data-slot` / `aria-*` DOM 概念;`actionsRef` / `onOpenChangeComplete` 这类外部动画协同;函数式 `sideOffset` / `alignOffset`(读锚点尺寸动态算,桌面收益低)。

#### Progress(#20)

**官方页**:Usage `<Progress value={33} />`、Composition(`Progress` / `ProgressLabel` / `ProgressValue` / `ProgressTrack` / `ProgressIndicator`)、Label、Controlled(由 Slider 控制)、RTL(排除)、API Reference → Base UI Progress。Base UI Root 有 `value: number|null`(null = indeterminate)、`min=0` / `max=100`、`format` / `locale` / `getAriaValueText` / `aria-valuetext`。

**已具备**:`value` / `max`(`WithProgressMax` / `SetMax`)、`value=null` 不确定态(`WithProgressIndeterminate` / `SetIndeterminate`)、指示器颜色覆盖(固定色 + 主题函数)、`Percent()`、`SetValue` 夹取、`OnChange`、`ProgressField` 三件套(标题 + 百分比 + 条,数值格式可 `WithProgressFieldFormat` 覆盖,对应官方 Label / Controlled 示例)、4px 高与胶囊圆角自绘。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| PR1 | `min` 下限 | Base UI Root 明确有 `min`,默认 0 | 进度恒为 `0..max`,`syncPos()` 里直接 `value/max` | 加 `min` 字段与 `WithProgressMin` / `SetMin` / `Min()`,百分比改 `(value-min)/(max-min)`;负值域也可用 | 小 / 中 |
| PR2 | 指示器宽度过渡 | `progress.tsx` 的 Indicator 有 `transition-all` | `SetValue` 直接 `syncPos` + 重绘,数值跳变 | 用运行时 Spin 或缓动把 `barW` 从旧值插值到新值;与 Tabs TB4 / Switch SW3 共用一套缓动 | 中 / 中 |
| PR3 | 轨道 / 指示器几何与配色可配 | Track / Indicator 是独立部件,均可传 `className`(改高度、圆角、轨道底色) | 轨道固定 `bg-muted`、高随 `cy`、圆角恒 `h/2`;只有指示器颜色可覆盖 | `WithProgressTrackColor` / `SetTrackColor`、`WithProgressRadius`、`ProgressField` 暴露 `Bar()` 已有,再给独立 Track 配置入口 | 小-中 / 中 |
| PR4 | 独立的 `ProgressLabel` / `ProgressValue` 部件 | Composition 把两者列为可自由摆放的子部件 | 只有固定组合 `ProgressField`(标题行 + 条),标签与数值不能分开放、不能挂在条外 | 导出 `NewProgressLabel` / `NewProgressValue`,订阅 `OnChange` 自动刷新;或给 `ProgressField` 加 `Labels()` 与 `SetLabelPlacement` | 中 / 低(现有组合已覆盖主用例) |

**已知差异 / 明确不做**:RTL;不确定态「短条往返」vs 上游空轨道(DESIGN 589 已记,是主动视觉选择);`format` / `locale` / `getAriaValueText` / `aria-valuetext`(Intl 与无障碍元数据,桌面由 `WithProgressFieldFormat` 覆盖格式化);`data-state=complete` 上游也无独立视觉;`render` DOM 概念。

#### Field(#21)

**官方页**:自带完整 API Reference(FieldSet / FieldLegend `variant` / FieldGroup / Field `orientation` + `data-invalid` / FieldContent / FieldLabel / FieldTitle / FieldDescription / FieldSeparator / FieldError `errors` + Standard Schema `issues`)、Anatomy、Form(React Hook Form 等)、Input / Textarea / Select / Slider / Fieldset / Checkbox / Radio / Switch / Choice Card / Field Group 示例、Responsive Layout、Validation and Errors、Accessibility、RTL(排除)。本批本地参考实现无 `field.tsx`,依官方表比对。

**已具备**:`Field` / `FieldSet` / `FieldGroup` / `FieldLegend`(`FieldLegendVariant` 两档) / `FieldContent`(`Content()`) / `Label` / `Title` / `Description` / `FieldSeparator` / `FieldError`(`SetErrors` 多条去重、空列表隐藏并回收祖先高度)、`FieldVertical` / `FieldHorizontal`、`WithFieldInvalid` / `SetInvalid` / `Invalid`、`WithFieldDisabled` / `SetDisabled` / `Disabled`、`WithFieldFocusProxy` / `SetFocusRingProxy`、`Field.Attach(FieldControl...)` 的 19 控件级联(含「本来就单独禁用」的恢复)。上游 `FieldError errors` 数组的多条渲染语义已覆盖。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| FL1 | `orientation="responsive"` | 官方 Responsive Layout 明确列为第三档;源码注释亦写「Go 侧没有容器查询,需调用方按窗口宽度自行选」 | 只有两档,且 `SetOrientation` 都没有 | 加第三档:挂宿主窗口 `WM_SIZE`,按 `Field` 实际宽度在断点切 `EnableHorizon` 并 `AdjustLayout()`;或先补运行时 `SetOrientation()` 让应用层自己切 | 中 / 中 |
| FL2 | Choice Card(整块点击选中) | 官方专节:「Wrap `Field` components inside `FieldLabel`」,配 RadioItem / Checkbox / Switch | 只有 `WithFieldFocusProxy` 声明焦点环交给外层,没有「点整块即选中」的容器 | 复用 #8 Label L1 的 `htmlFor` 点击关联(`FocusElement` + 对实现 `Toggle()` 的控件切换),再加 `NewFieldChoiceCard(build)` 把 Field 放进可点击标签 | 中 / 中(与 L1 同批做) |
| FL3 | FieldGroup 级 `Validate()` / 提交聚合 | 官方 Form 节与 Validation and Errors;本台账 Input I3 / Select S9 / Label L3 都指向同一套 Field 基础设施 | `SetInvalid` 是逐字段手动调用,没有组级必填/校验器注册与一次性汇总 | `FieldGroup` 升级为带 `controls`/`validators` 的结构体,提供 `Register(field, validator...)`、`Validate() []error`、`OnSubmit(fn)`;I3/S9/L3 一并落地 | 中-高 / 中(跨组件共用) |
| FL4 | `Field` 与 `FieldError` 绑定 | 官方 Anatomy:FieldError 紧跟控件或放 FieldContent 内,与 `data-invalid` 同状态 | `FieldError` 是独立组件,调用方要自己 `SetInvalid(true)` 再 `SetErrors(...)`,两者不会联动 | `Field.AttachError(*FieldError)` / `Field.SetErrors(...)`:设错时自动置 invalid 并显示错误区,清错时恢复;多条错误仍走现有 `SetErrors` | 小-中 / 高 |

**已知差异 / 明确不做**:RTL;`FieldLabel htmlFor` 是 #8 Label L1,**不在此重复计项**;Standard Schema `issues` / `errors` 对象数组 / React Hook Form 等 JS 生态概念(DOM/框架语义,但 FL3 会提供等价的 Go 侧聚合入口);`role=group` / `aria-*` 无障碍元数据;横向 `Content` 只认 Fixed 宽(DESIGN 565 能力边界);描述/错误默认单行(DESIGN 566,`SetAutoWrap()` 已有替代)。

#### Input Group(#22)

**官方页**:Usage / Composition、Align 四档(inline-start/end、block-start/end,API Reference 明确 `align` 默认 inline-start 且注明「Input 用 inline、Textarea 用 block」)、Icon / Text / Button / Kbd / Dropdown / Spinner / Textarea / Custom Input(`data-slot="input-group-control"` 让第三方控件也吃到焦点态)、RTL(排除)、`InputGroupButton` 的 `size`(xs/icon-xs/sm/icon-sm)与 `variant` 六档。

**已具备**(本批覆盖度最高):`Input()` / `Textarea(cy)`、`Addon(align)` 四向、`Addon.Text/Icon/Button/IconButton/MenuButton/Kbd/Spinner`、`Search()` 便捷构造、`SetInvalid` / `SetDisabled` / `SetFieldState`(实现 `FieldControl`)、容器级焦点环与校验环自绘、`syncInsets()` 按 addon 位置收窄输入内边距、`markBlock()` 三段式布局、按钮四尺寸档(`applyInputGroupButton` 的圆角模式 + `fixButtonBox` 按档算高)、`WithInputBare` 裸输入。官方 11 个示例除 Custom Input 外都有现成 API。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| IG1 | 点击 addon 聚焦输入框 | `input-group.tsx` 的 `InputGroupAddon.onClick` 明确写「非按钮区域点击 → `querySelector('input').focus()`」 | `Addon.Text/Icon` 建的是 `EnableFocus(false)` 的静态元素,点文字/图标不会把焦点给输入框,与浏览器行为不一致 | 给 addon 容器挂 `LButtonUp`,命中非按钮子项时调 `g.Focus()`;按钮子项保持原样(上游也 `return`) | 小 / 高 |
| IG2 | 自定义控件接入(`data-slot=input-group-control`) | 官方 Custom Input 专节:第三方 resizable textarea 也能吃到容器焦点/禁用/校验态 | `focused()` / `Focus()` / `SetDisabled()` 全部写死 `g.input`(`*Input`),`Content()` 只返回 addon 容器句柄 | 加 `InputGroup.AttachControl(hEle)` 登记外部控件:焦点判断改为「任一登记控件持有焦点」,禁用时对登记控件 `Enable(!v)`;配合 IG1 把 `Focus()` 指向当前控件 | 中 / 中 |

**明确不做**:RTL;`data-slot` DOM 属性本身(只取其「统一焦点态」语义);`aria-invalid` / `role=group` 无障碍元数据;负外边距(项目硬规则,通栏用独立容器)。**已覆盖不要重复列**:Button 的 size/variant 档、block 组合、Dropdown(MenuButton)、Spinner、Kbd、Textarea。

#### Avatar(#23)

**官方页**:Usage / Composition(`Avatar` + `AvatarImage` + `AvatarFallback` + `AvatarBadge`;`AvatarGroup` + `AvatarGroupCount`)、Basic、Badge(可用 `className` 自定义颜色尺寸)、Badge with Icon、Avatar Group / Group Count / Group with Icon、Sizes(sm/default/lg)、Dropdown(头像作为菜单触发区)、RTL(排除)、API Reference。Base UI Avatar 的 Fallback 有 `delay`,Image 有 `onLoadingStatusChange`。

**已具备**:三档尺寸(24/32/40,构造期 `cx/cy` 可覆盖)、本地文件与 `image.Image` 两种图源、CPU 侧 4 倍抗锯齿圆形蒙版居中裁剪、加载失败自动回落首字母、`SetImage` / `SetImageData` / `SetFallback`、`WithAvatarRing` 背景色环、`WithAvatarBadge(icon)` / `SetBadge(on, icon)` 右下角徽标(尺寸随档位 8/10/12 + 2px 环)、`NewAvatarGroup` 负间距叠放、`Add` / `AddCount(n)`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| AV1 | 徽标自定义颜色 / 尺寸 / 任意内容 | 官方 Badge 节用 `className="bg-green-600 dark:bg-green-800"`;Badge with Icon 可放任意图标 | `paintBadge` 固定 `t.Primary` 底 + `t.PrimaryForeground` 图标色,直径由 `size.badgeD()` 定死 | `WithAvatarBadgeColor(c)` / `SetBadgeColor`、`SetBadgeSize(d)`;颜色沿用 `xui.Color` 或主题函数(与 Badge `WithBadgeColors` 同形态) | 小 / 高 |
| AV2 | `AvatarGroupCount` 放图标或任意内容 | 官方 Avatar Group with Icon 示例 | `AddCount(n)` 只能生成 `"+N"` 文字 | 给 `AddCount` 加 opts 或新增 `AddCountContent(build func(*Avatar))`;返回的 `*Avatar` 已可 `SetFallback`,但图标需自绘槽位 | 小-中 / 中 |
| AV3 | 头像可交互(作为 Dropdown / Popover 触发区) | 官方 Dropdown 示例 | 注释明确「纯展示,不接收焦点与鼠标」,`EnableFocus(false)` + `EnableSwitchFocus(false)` | 加 `WithAvatarInteractive()` 打开焦点与点击,再用现有 `NewPopover(a.Handle)` / `Menu.PopupAtElementRect` 组合;焦点环复用 `SetFocusRingProxy` 思路 | 中 / 中(与 #8 Label L1 同属「展示元素变触发区」) |
| AV4 | 运行时 `SetSize` / `SetRing` | 与 Button `SetSize` 等一致性;上游 `size` 是 Root prop | 尺寸与色环只在构造期通过 `AvatarOption` 设置,`SetBadge` 等已有运行时 API | `SetSize(AvatarSize)` 重设矩形并重生成圆形图片句柄(蒙版按新直径),`SetRing(bool)` 触发重绘 | 小-中 / 中 |
| AV5 | 图片状态回调 / fallback 延迟显示 | Base UI Fallback `delay`、Image `onLoadingStatusChange` | 同步解码,`syncImage()` 失败静默回落,调用方拿不到结果;也无加载态 | `SetImage` 返回成功标志或加 `OnImageStatus(func(ok bool))`;延迟回落用计时器(桌面本地解码几乎瞬时,价值低) | 小 / 低 |

**已知差异 / 明确不做**:RTL;远程 URL / 动图(DESIGN 578 能力边界);`alt` / `aria-*` 无障碍元数据;`object-cover` 由 CPU 裁剪等价实现;圆形裁剪本身已实现(#7 Card C4 提到要重评估第七节,这里不重复)。

**本批实现顺序建议**:PO1 → IG1 → FL4 → AV1 → PR1(小改动、收益明确);随后 PO2 / PO3 / PR2 / PR3 / FL1 / FL2 / IG2 / AV2 / AV3 / AV4;FL3 与 Input I3 / Select S9 / Label L3 必须同批设计;PO4 / PO5 / PO6 / PR4 / AV5 建议先确认需求。PO1 与 Menu DM2、PO2 与 Dialog D3、PO5 与 Tooltip TT2、FL2 与 Label L1、AV3 与 Label L1 都建议共用同一套 API 形态,避免各写一份。

### 3.7 Accordion ~ Scroll Area(#24 ~ #30)

依据:官方 `base/accordion`、`base/slider`、`base/alert-dialog`、`base/sheet`、`base/separator`、`base/skeleton`、`base/scroll-area` 七页(示例清单与正文行为说明),Base UI **v1.9.0** 的 Accordion / Slider Root·Item·Thumb props 表(含 `onValueChange` / `onOpenChange` 的 `eventDetails.cancel()`、`thumbAlignment`、`largeStep`),本地 base-nova 的 `accordion.tsx` / `slider.tsx` / `sheet.tsx` / `separator.tsx` / `skeleton.tsx` / `scroll-area.tsx`(alerts-dialog.tsx 本地未装,按官方页 + Base UI Dialog 比对),本库 `accordion.go` / `slider.go` / `alertdialog.go` / `sheet.go` / `dialog.go` / `separator.go` / `skeleton.go` / `scrollarea.go` / `scrollbar.go`,DESIGN 第四节末语义表与第七节差异表。RTL 七页都有示例,按约定排除。

#### Accordion(#24)

**官方页**:Basic(`defaultValue` 数组)/ Multiple / Disabled / Borders / Card / RTL(排除) / API → Base UI。tsx:root `flex w-full flex-col`;item `not-last:border-b`;trigger `py-2.5 text-sm hover:underline focus-visible:ring-3 aria-disabled:opacity-50` + 两枚 chevron 互换;content `data-open:animate-accordion-down` + 内层 div `pb-2.5`。

**已具备**:single(可全收)/ multiple、条目禁用、条目间 1px 分隔线(末项没有)、高度过渡 200ms(Collapsible)、hover 下划线、focus-visible 环、空格/回车切换、`Value/SetValue/OnChange`、箭头图标互换、内容 `pb-2.5`、面板常驻(等价 `keepMounted`)。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| ACC1 | 根级 `disabled` | Root props(`false` = 忽略全部用户交互) | 只有条目级;无法整组禁用 | `WithAccordionDisabled` / `SetDisabled` / `Disabled`:整组禁用与条目自身禁用分开记录,启用后各自恢复 | 小 / **高** |
| ACC2 | `onValueChange` 可否决 | Root 的 `eventDetails.cancel()` | `OnChange` 只通知,点开即展开 | `OnValueChangeVeto(func(next []string) bool)`,参数是"将要变成的完整集合"(single 互斥已算进去);程序化 `SetValue` 不经过 | 小 / **高** |
| ACC3 | Item `onOpenChange` 可否决 | Item 的 `eventDetails.cancel()` | 无条目级钩子 | `OnOpenChangeVeto(func(open bool) bool)`;被否决时把原生 Check 的勾选位翻回去(同 Checkbox, 第八节 198) | 小 / **高** |
| ACC4 | 触发区放任意内容 | Trigger 的 `children` | 触发区是 Button,只能文字 + 图标 | `Trigger()` 已能改文字/图标;要放任意控件需给条目加内容槽(与 D4 / SH1 同一形态) | 中 / 低 |
| ACC5 | Borders / Card 示例 | 官方两例(`border` + `last:border-b-0`;Accordion 放进 Card) | 组件能组合,缺 demo 对照 | 只补 demo | 小 / 低(非功能缺口) |

**明确不做**:RTL;`hiddenUntilFound`(浏览器页内查找专属);`orientation` / `loopFocus`(Base UI 已标废弃);`render` / `data-slot`。

#### Slider(#25)

**官方页**:Range / Multiple Thumbs / Vertical / Controlled / Disabled / RTL(排除)。tsx 明确写 `thumbAlignment="edge"`;Base UI Root 还有 `thumbCollisionBehavior` / `largeStep` / `minStepsBetweenValues` / `format` / `locale`。

**已具备**:值数组(单值/范围/多滑块)、`min/max/step`、垂直、禁用、`minStepsBetweenValues`、`inverted`(本库扩展)、`onValueChange` / `onValueCommitted`、点击跳转 + 拖拽 + 键盘(方向键/Home/End)、hover/拖动/焦点 ring、`FieldControl`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SL1 | `thumbAlignment="edge"` | tsx 显式设置;Base UI 默认 `center` | 滑块**中心**贴轨道端点(radix 老语义),两端溢出半个身位 | 轴区间改"两端各内缩半个滑块",`pointValue` / `thumbAt` / 指示器同口径;保留 `WithSliderThumbAlignment(SliderThumbCenter)` 回退 | 小-中 / **高** |
| SL2 | `largeStep` | Root props(默认 10;PageUp/Down 与 Shift+方向键) | PageUp/Down = step×10,Shift 无效 | `WithSliderLargeStep` + `normalStep/largeStepValue` 两个纯函数;Shift 用 `wapi.GetAsyncKeyState` | 小 / 中 |
| SL3 | 禁用时滑块也变淡 | Control 整体 `opacity-50` | 只淡轨道与已完成区间,滑块保持全亮 | 滑块白底与 border-ring 描边一起 `WithAlpha(alpha)` | 极小 / 中 |
| SL4 | `thumbCollisionBehavior` | Root props:`push`(默认)/`swap`/`none` | 相邻滑块互相夹住(等价 `none`) | 拖动时把被撞的相邻滑块一起推(`push`)、或交换索引(`swap`);需改 `setValueAt` 的夹取逻辑 | 中 / 中 |
| SL5 | 单滑块 `disabled` | Thumb props | 只能整组禁用 | 每滑块一个禁用位,命中/键盘/绘制分别判定 | 小-中 / 低 |

**明确不做**:RTL;`format` / `locale` / `getAriaValueText`(Intl 与无障碍元数据);每 thumb 独立 roving tabindex(已记第七节差异);`name` / `form` 表单提交。

#### Alert Dialog(#26)

**官方页**:Basic / Small(`size="sm"`)/ Media / Small with Media / Destructive / RTL(排除) / API(仅列出 `size`)。Composition = Trigger + Content(Header[Media/Title/Description] + Footer[Cancel/Action])。

**已具备**:`size` 两档(default 左对齐头 + 右对齐页脚 / sm 居中头 + 等宽双列页脚)、media 媒体块(40×40 + size-6 图标 + muted / destructive 两档)、Cancel 默认聚焦、点遮罩不关闭、ESC 关闭、`BindTrigger`、`AddAction` 任意变体、`OnClose`、与 Dialog 共用遮罩/焦点陷阱/模态栈/100ms 淡入。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| AD1 | `OnOpenChange` / `SetOpen` | Base UI Dialog Root(AlertDialog 复用) | 只有 `OnClose`,与 Dialog / Sheet 口径不一致 | 转发 `d.OnOpenChange` / `d.SetOpen` | 极小 / **高** |
| AD2 | media 放任意图片 | `AlertDialogMedia` 可放图标或图片 | 只能给 icon path / SVG | 加 `SetMediaImage(path)` 走图片句柄(同 Avatar 的加载路径) | 中 / 低 |
| AD3 | Header / Footer 任意内容槽 | Composition 的容器部件 | 头部只有标题/描述,页脚只有按钮 | 同 Dialog D4:导出容器或 `Title()/Description()` | 中 / 中 |
| AD4 | 遮罩 / 面板样式可配 | Overlay 可传 className | 遮罩固定黑 10%、内边距固定 | 同 Dialog D5 | 小-中 / 低-中 |

**明确不做**:RTL;zoom-in-95 与 text-balance(能力边界);`render` / `aria-*`。**不计入**:Initial Focus / Final Focus / 可否决关闭 —— D3 已给 AlertDialog 用(Cancel 聚焦),`OnClosing` 也已在 Dialog 侧可用。

#### Sheet(#27)

**官方页**:Side / No Close Button / RTL(排除) / API → Base UI Dialog(`side`、`showCloseButton`)。tsx:面板 **bg-popover**、`gap-4`、关闭钮 **top-3 right-3 + icon-sm**、SheetHeader **`gap-0.5`**、SheetFooter `gap-2 p-4`、四向平移 2.5rem + 200ms。

**已具备**:四向面板(左右 3/4 + max-w-sm、上下按内容自撑)、面板滑入/滑出、遮罩淡入、页脚纵向铺满、Tab 焦点陷阱与模态栈、`WithSheetWidth/CloseButton/Duration`、`BindTrigger/BindClose`、`OnClose`、`Content/ContentWidth`、`AddFooter`。

**未实现但可实现 / 本批修正**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SH1 | `OnOpenChange` | Base UI Dialog Root | 只有 `OnClose` | 转发 `Dialog.OnOpenChange`(同一状态不重复通知) | 极小 / **高** |
| SH2 | `OnClosing` 可否决 | Dialog 的关闭拦截 | 无 | 转发 `Dialog.OnClosing`("未保存内容") | 极小 / 高 |
| SH3 | `SetOpen` 受控入口 | Root 的 `open` | 只有 Show/Hide/Close | 转发 `Dialog.SetOpen` | 极小 / 中 |
| SH4 | 关闭钮位置与面板底色 | tsx:`top-3 right-3`、`bg-popover`、Header `gap-0.5` | `top-4 right-4`、`bg-background`、`gap-1.5`(radix 旧值) | 常量改 12 / 2,Sheet 面板取 `t.Popover`(Drawer 仍用 Background) | 小 / **高**(肉眼可见的对齐) |
| SH5 | Header / Footer 任意内容槽 | Composition | 头部两条文字 + 页脚按钮 | 同 Dialog D4 | 中 / 中 |
| SH6 | 遮罩 / 面板样式可配 | Overlay / Popup 的 className | 遮罩固定黑 10%(本库为黑 50) | 同 Dialog D5 | 小-中 / 低-中 |
| SH7 | 进场位移量对齐 2.5rem + opacity | tsx 的 `data-starting-style:translate-x-[2.5rem]` | 从整块面板宽滑入(300ms) | 改小位移 + 同时淡入;属观感选项,先确认需求 | 小 / 低-中 |

**明确不做**:RTL;shadow-lg(元素无圆角投影);内容超面板高自动滚动(裁剪,已在第七节);`render` / `aria-*`。

#### Separator(#28)

**官方页**:Vertical / Menu / List / RTL(排除) / API → Base UI Separator(`orientation`)。

**已具备**:水平/垂直、1 逻辑像素并按**整设备像素**对齐(150% DPI 不会一粗一细)、`WithSeparatorVertical/Orientation`、布局项可声明长度。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SEP1 | 颜色可覆盖 | tsx 的 `bg-border` 可被 className 换掉 | 恒 `t.Border` | `WithSeparatorColor` / `SetColor` / `Color()`(传 `xui.None` 回到令牌) | 极小 / 中 |
| SEP2 | 垂直 `self-stretch` | tsx 的 `data-vertical:self-stretch` | 垂直必须给高度(cx),传 0 就是 0 高 | `WithSeparatorStretch()` 设布局项高度 Fill(父级是布局容器时撑满) | 小 / 中 |
| SEP3 | Menu / List 示例 | 官方两例 | 组件能组合,缺 demo | 只补 demo | 小 / 低 |

**明确不做**:RTL;`decorative` / `role` / `aria-orientation`(无障碍 DOM 元数据)。

#### Skeleton(#29)

**官方页**:默认示例 + Avatar / Card / Text / Form / Table / RTL(排除);**没有 API Reference**,tsx 只有 `animate-pulse rounded-md bg-muted`。

**已具备**:圆角(主题 RadiusMd / 自定义 / 圆形)、`bg-*` 可换成窗口底色(`WithSkeletonOnBackground`)、静态模式、2s 脉冲(1 → 0.5 → 1,运行时 `xui.Pulse`)、宽高交给调用方/布局。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SK1 | 底色用 `bg-muted` | tsx(`radix-nova` 时代是 `bg-accent`) | 取 `t.Accent` | 改 `t.Muted`;默认主题两者同值,自定义主题才看得出差别 | 极小 / **高** |
| SK2 | 五种示例组合 | Avatar / Card / Text / Form / Table | 组件能拼,缺同构 demo | 只补 demo(卡片骨架 / 表格骨架) | 小 / 低 |

**明确不做**:RTL;`className` / `data-slot`。

#### Scroll Area(#30)

**官方页**:默认示例 / Horizontal / RTL(排除) / API → Base UI。tsx:Root `relative`、Viewport `size-full rounded-[inherit] focus-visible:ring-3`、ScrollBar 10px(`w-2.5 p-px`)+ thumb `rounded-full bg-border`、Corner。

**已具备**:纵向滚动(内容 Auto 高、超出视口即滚)、滚动条 hover 显隐(150ms 元素 alpha,不改布局宽)、10px 轨道 + 8px 圆头滑块、轨道透明、滑块悬停不换色、滚轮冒泡、`ScrollToTop`、`SetContentFill`(铺满型页面)、首帧滚动条宽度自愈。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SA1 | 滚动条常显 | `type="always"`(桌面习惯) | 恒"悬停才显示" | `WithScrollAreaScrollbarAlways()`:alpha 拉满、不挂悬停判定;内容不足仍由原生自动隐藏 | 小 / **高** |
| SA2 | 横向滚动 | 官方 Horizontal 示例 + ScrollBar 的 `orientation` | 明确只做纵向 | 加 `WithScrollAreaHorizontal`/横向 ScrollArea:内容区改 Auto 宽、`ShowSBarH(true)`;需要同步自愈闸门与滚动条皮肤 | 中-高 / 中 |
| SA3 | `ScrollTo(hEle)` / `ScrollIntoView` | 上游无(浏览器原生) | 只能 `ScrollToTop` / `ScrollPosYV` | 读目标元素相对内容区的 y,调 `ScrollPosYV`;本库扩展 | 小-中 / 低-中 |

**明确不做**:RTL;视口 focus-visible 环与键盘滚动(容器不进 xcgui 的 Tab 链,第八节 157);`rounded-[inherit]` 圆角裁剪(能力边界);Corner(只纵向时不存在);`size-full` / `data-slot`。

**本批实现顺序建议**:ACC1 → ACC2 → ACC3 → SL1 → SL2 → SL3 → SK1 → SEP1 → SEP2 → SA1 → SH1~SH4 → AD1(本批已全部落地,见第五节);ACC4 / AD3 / SH5 是同一形态(内容槽)建议一起设计;SL4 / SL5 / SA2 / SH7 需先确认需求或成本较高。

## 四、实现顺序建议(Input / Select 批次,已执行完)

1. **小改动、一致性收益**:S3 `OnOpenChange` → I2 `readOnly` → I7 `SetDisabled` → S4 `readOnly` → S8 `Value` 格式化 —— **已全部落地**(见第五节)。
2. **中改动、功能缺口**:I3/S9 必填校验、I4 上限 **已落地**;I1 文件输入、S1 定位开关、S2 typeahead **未做**(I1 见下方"待确认"之外,可直接排期)。
3. **需先确认需求或成本较高**:S5 多选、S7 滚动箭头、S6 悬停开关、I6 原生 Edit 视觉 API、I8 demo 对照 —— 均未做,取舍见第五节末尾的"待确认需求"。

落地时按仓库固定流程:先查 DESIGN 第四节末语义表与本地 `*.tsx`,改完补 DESIGN 语义/差异表、CHANGELOG,跑 `go build ./...` / `go vet ./...` / `go test ./...`、逐组件单独安装 + 编译,再上 demo 做亮暗与 150% DPI 视觉验收。

## 五、实现进度(已落地条目)

按「高优先、小改动」选的跨组件一致性条目,分批落地(验收记录见 CHANGELOG,语义写进 DESIGN 第四节末与差异表)。当前进度:第一~七批已落地。

**第一批**:

| # | 组件 | 落地内容 |
|---|---|---|
| C2 | Card | 有页脚时内容区底部内距归 0(`has-data-[slot=card-footer]:pb-0` 等价) |
| C1 | Card | `WithCardSpacing(px)` / `SetSpacing(px)` 任意 `--card-spacing`,同步内容区与页脚;`CardFooter.SetPadding` |
| D1 | Dialog | `OnOpenChange` 统一 Show / Hide / SetOpen / ESC / 遮罩 / 关闭钮 / Close;`SetOpen` 受控入口 |
| D2 | Dialog | `WithDialogDismissOutside` / `WithDialogEscClose` / `OnClosing`(可否决关闭) |
| D3 | Dialog | `WithDialogInitialFocus` / `WithDialogFinalFocus`(未指定时仍还焦点给下层模态) |
| L1 | Label | `BindFor(hEle)`(= `htmlFor` 聚焦)、`BindToggle(hEle, fn)`(聚焦 + 切换) |
| L2 | Label | `SetDisabled` / `Disabled`(peer-disabled 变淡)、`SetInvalid` / `Invalid`(破坏性字色) |
| L4 | Label | `SetAlignRight` |
| S3 | Select | `OnOpenChange`(打开后 true; ESC / 失焦 / 点外部 / Close 经弹层 `OnHide` 通知 false) |
| DM1 | Menu | `OnOpenChange`(弹出前 true / 关闭后 false);`ContextMenu` 口径不变 |
| I2 | Input / Textarea | `WithInputReadOnly` / `SetReadOnly` / `ReadOnly`(原生 `XEdit_EnableReadOnly`) |
| I7 | Input / Textarea | `SetDisabled` / `Disabled`;`SetFieldState` 收敛到 `SetInvalid` + `SetDisabled` |
| SW1 | Switch | `WithSwitchReadOnly` / `SetReadOnly` / `ReadOnly` |
| SW2 | Switch | `OnCheckedChange(func(bool) bool)` 可否决(鼠标 / 空格 / Toggle 同路径) |
| RG1 | Radio / RadioGroup | 条目与组级 `readOnly`(组级还原上一个被接受的值;方向键仍搬焦点) |
| RG2 | RadioGroup | `OnValueChangeVeto(func(string) bool)` |

**第二批**(同样是跨组件一致性小改动, 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| PO1 | Popover | `popupPositionOffset` 支持交叉轴偏移;`WithPopoverAlignOffset(px)`(= Base UI Positioner 的 `alignOffset`, 上下方位正数向右、左右方位正数向下) |
| IG1 | Input Group | 点 inline addon 的文字/图标/空白区聚焦输入框(`InputGroupAddon.onClick` 的 `querySelector('input').focus()`);按钮子项保持自己的点击语义 |
| FL4 | Field | `Field.AttachError(*FieldError)` + `Field.SetErrors(...)`: 写文案自动置 invalid, 清空自动恢复;`FieldError.SetErrors` 也会沿父链把所在字段切到校验态;新增 `HasErrors()` / `Errors()` |
| AV1 | Avatar | `WithAvatarBadgeColor` / `WithAvatarBadgeColorFunc` / `WithAvatarBadgeIconColor` / `WithAvatarBadgeSize` + 运行时 `SetBadgeColor/SetBadgeIconColor/SetBadgeSize`;自定义底色时图标色按亮度自动挑 |
| PR1 | Progress | `WithProgressMin` / `SetMin` / `Min()` + `ProgressField` 同款;百分比改 `(value−min)/(max−min)`, 支持负值域 |

**第三批**(对话框/弹层交互与展示控件的小步补齐):

| # | 组件 | 落地内容 |
|---|---|---|
| PO2 | Popover | 打开后默认聚焦内容区首个可聚焦控件(`WithPopoverInitialFocus` 可指定);关闭时只有 ESC 或显式 `WithPopoverFinalFocus` 才搬回焦点,点其它控件关掉不抢焦点 |
| PO3 | Popover | 非 Button 触发区补 Enter/空格开合;`WithPopoverDefaultOpen()`;`WithPopoverControlled(true)` = 触发区只发意图、由 `SetOpen` 定状态 |
| IG2 | Input Group | `AttachControl(hEle)` / `Controls()`:外部控件吃到容器的焦点环判断、`Focus()` 落点与禁用态(对应 `data-slot=input-group-control`) |
| AV2 | Avatar | `SetIcon(path)` + `AvatarGroup.AddCountIcon(icon)`:计数头像可放图标(上游 Avatar Group with Icon) |
| AV4 | Avatar | 运行时 `SetSize(AvatarSize)`(连布局项一起改 + 重建圆形蒙版)与 `SetRing(bool)` |
| AV3 | Avatar | `WithAvatarInteractive()` / `SetInteractive(bool)`:打开焦点与 Tab, 作 Popover/Menu 触发区;键盘焦点环画在圆内侧 2px |
| PR3 | Progress | 轨道配色 `WithProgressTrackColor/[...Func]`、`WithProgressRadius`,运行时 `SetTrackColorFunc/SetRadius`;`ProgressField` 同款 |

**第四批**(进度 / 字段取向 / 选择卡片,已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| PR2 | Progress | 值变化时指示器宽度 150ms 过渡(对应 Indicator 的 `transition-all`): 从当前显示宽度续跑, 中途再改值不跳变; 运行时 `PresenceOnce` 驱动, 不自起定时器、不新增运行时导出 |
| FL1 | Field | `FieldResponsive`(宽度 ≥ 448 = 上游 `@md/field-group` 容器查询时横排, 不足竖排) + 运行时 `SetOrientation` / `Horizontal`; 跨断点自动切横竖并重排; 切横排时直属「Fill 宽 + 非 Auto 高」控件收成 240、切回竖排恢复满宽; `Content` 横排固定 240 / 竖排满宽 |
| FL2 | Field | `NewFieldChoiceCard` + `Label/Description/Content` + `Attach(Checkbox/Radio/Switch)`: 整卡命中(子项与指示器鼠标穿透)、点整卡聚焦并切换、选中 = bg-accent + border-primary、指示器宿主底色跟随卡片、键盘焦点环代理; `Switch` 补 `SetBackdrop` |

**第五批**(Field 必填与校验基础设施, I3 / S9 / L3 / RG3 / FL3 同批落地):

| # | 组件 | 落地内容 |
|---|---|---|
| I3 | Field / Input | `WithFieldRequired` / `SetRequired` / `Required` + 标签后破坏性色 `*`(运行时显隐); `WithFieldRequiredMessage` / `SetRequiredMessage` 覆盖文案; `WithFieldValidator` / `AddValidator(func(value string) error)`; `Validate()` 一次收集必填 + 校验器文案并写入错误区, `ValidationErrors()` / `Value()` |
| S9 | Field / Select | 必填校验统一走 Field: 控件实现可选接口 `FieldValueProvider`(`FieldValue() string`), Select 等选择类返回选中值; `Field.Attach(select)` 后 `Validate()` 即按空选中判定 |
| L3 | Field / Label | 必填星号由 Field 统一出(不单独给 Label 加 `SetRequired`): 标签行为「主标签 + 破坏性色 `*`」, 星号跟随 `SetRequired` 显隐、`SetDisabled` 调暗 |
| RG3 | Field / Radio Group | RadioGroup 实现 `FieldValueProvider`(返回组值), 必填 = 组内无选中; 与 I3 / S9 共用同一套 Field 校验入口 |
| FL3 | Field / FieldGroup | `NewFieldGroup` 返回 `*FieldGroup`: `Fields()` 深度优先列出全部字段(含 Content / 嵌套组), `Validate()` 逐字段校验并各自展示错误, `Submit(fn)` 校验通过才执行提交回调 |

**第六批**(散落小改动,跨 6 个组件;已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| S4 | Select | `WithSelectReadOnly` / `SetReadOnly` / `ReadOnly`: 弹层仍可打开浏览与高亮, 但选中不落值、不发 `OnChange`, 触发区不置灰(与 Checkbox / Radio 的 readOnly 同语义) |
| S8 | Select | `WithSelectValueText(func(SelectItem) string)` / `SetValueText`: 只改触发区文字(前缀/单位/多字段拼接); 同时补上 `Text` 为空回落 `Value` 的显示与量宽 |
| I4 | Input / Textarea | `WithMaxLength(n)` / `SetMaxLength` / `MaxLength`: 变更事件里截掉超限尾巴(粘贴、输入法上屏同样拦截), 单行光标留在截断点、多行回末尾; `SetValue` 程序化写入同样受约束 |
| TB1 | Tabs | `OnValueChangeVeto(func(value string) bool)`: 点击 / 方向键(automatic) / Enter(manual) 三条路径统一可否决, 程序化 `SetActive` / `SetActiveValue` 不经过 |
| TB2 | Tabs | 无选中态: `WithTabsActive(-1)` / `SetActive(-1)` / `ClearActive()`; 绑定面板全部隐藏, 值回调收到空串, 键盘从第一个可用标签进入 |
| BA2 | Badge | 图标侧内距收窄到 6(`has-data-[icon]:px-1.5`), 无图标侧仍是 8; 内容在不称内距内居中 |
| BA3 | Badge | `NewLinkBadge(x, y, cx, cy, text, url, parent, opts...)` / `SetLinkURL` / `LinkURL`: link 样式 + 手型光标 + 点击 `ShellExecute` 打开 |
| AL1 | Alert | `WithAlertColors(bg, border, fore, desc)` / `SetColors`: 四层配色直接覆盖, 不随变体与主题取色 |
| DT2 | Data Table | `WithDataTableEmptyText(s)` / `SetEmptyText`: 空态与"无匹配"文案可覆盖(默认仍是内置两句) |

**第七批**(#24~#30 首轮核查的高优先小改动, 跨 7 个组件; 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| ACC1 | Accordion | `WithAccordionDisabled` / `SetDisabled` / `Disabled`: 整组禁用只忽略用户交互(点击 / 空格回车 / Toggle), 程序化 `SetValue` / `SetOpen` 仍生效; 条目自身的禁用状态独立记录, 整组启用后各自恢复 |
| ACC2 | Accordion | `OnValueChangeVeto(func(next []string) bool)`(Root 的 `eventDetails.cancel()`): 参数是"将要变成的完整集合"(single 互斥已算进去), 一次交互只问一次; 程序化路径不经过 |
| ACC3 | Accordion | 条目 `OnOpenChangeVeto(func(open bool) bool)`(Item 的 cancel); 被否决时把触发区原生 Check 的勾选位翻回去, 不会留下"亮了但没展开" |
| SL1 | Slider | 两端对齐改为 Base UI 的 `thumbAlignment="edge"`(对齐 tsx): 值在 min/max 时滑块**边缘**贴轨道端点、不再溢出; `WithSliderThumbAlignment(SliderThumbCenter)` 回到"中心贴端点"的旧语义; 轴区间抽成纯函数 `sliderAxisSpan`(点击跳转 / 拖动 / 命中 / 绘制同口径) |
| SL2 | Slider | `WithSliderLargeStep` + `Shift+方向键` 与 PageUp·PageDown 走大步进(默认 step×10, 连续模式为值域一半, 与旧行为一致) |
| SL3 | Slider | 禁用时滑块的白底与 border-ring 描边也逐层乘 50%(上游是 Control 整体 `opacity-50`, 原来只淡轨道) |
| SK1 | Skeleton | 底色由 `Accent` 改 `Muted`(base-nova 的 `bg-muted`); 默认主题两者同值, 自定义主题才看得出差别 |
| SEP1 | Separator | `WithSeparatorColor` / `SetColor` / `Color()`: 覆盖线色, 传 `xui.None` 回到 Border 令牌 |
| SEP2 | Separator | `WithSeparatorStretch()`: 垂直分隔线在布局里撑满父级高(= `self-stretch`), 长度参数可传 0 |
| SA1 | ScrollArea | `WithScrollAreaScrollbarAlways()`: 滚动条常显(对应 `type="always"`), alpha 拉满且不挂悬停判定, 内容不足时仍由原生自动隐藏 |
| SH1~SH3 | Sheet | 转发 `OnOpenChange` / `OnClosing` / `SetOpen`: 与 Dialog 同一套开合与"未保存内容"否决语义 |
| SH4 | Sheet | 对齐 base-nova: 关闭钮 `top-3 right-3`(12)、SheetHeader 间距 `gap-0.5`(2)、面板底色 `bg-popover`(Drawer 仍用页面底色) |
| AD1 | AlertDialog | 转发 `OnOpenChange` / `SetOpen`: 与 Dialog / Sheet 口径统一 |

**第八批**(可直接排期清单一次性落地, 跨 18 个组件; 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| I1 | Input / Textarea | `WithFileInput(dlgOpts...)` / `WithFileInputMulti` / `WithFileInputMaxFiles`: 只读显示文件名(多选 `首个 +N`) + 右侧"选择文件"文字钮调 FileDialog, 取消不改值; `SetFiles/Files/ClearFiles/OnFilesChange`, 文件形态下 `FieldValue()` 返回路径列表(必填 = 必须选文件) |
| I6 | Input / Textarea | `WithInputSelectColor`(选区底色, 默认主色 25%)、`WithInputCaretWidth`、`WithInputAutoSelect`、`Undo` / `Redo` |
| TA2 / TA3 | Textarea | `WithTextareaRows(n)` 最小高按实际行高换算(与 cy 二选一); `WithTextareaResize()` 右下角抓手拖拽改最小高度(屏幕物理增量 ÷ DPI, 自动长高继续生效) |
| S6 | Select | `WithSelectHighlightOnHover(false)` / `SetHighlightOnHover`: 纯悬停不画任何反馈, 高亮只跟键盘/初始选中项(默认模式悬停搬高亮) |
| TT1 / TT2 | Tooltip | `WithTipHoverable()`(气泡可移入: 动态去 `WS_EX_TRANSPARENT` + 锚点离开延迟确认)、`WithTipArrowPadding(px)` |
| BA1 | Badge | 可点击徽章进 Tab 链 + Enter/空格触发; focus-visible 与 aria-invalid 环画在胶囊内侧 2px; `WithBadgeInvalid` / `SetInvalid` |
| TB3 / TB4 | Tabs | 选中项被禁用时自动回退到最近可用标签(全禁用则清空); line 变体指示条 150ms 滑动 |
| SW3 | Switch | 滑块位移过渡 150ms(回滚同样跟着回, 初始值直接落位) |
| RG4 | Radio Group | `WithRadioGroupHorizontal` / `SetHorizontal`: 横向排列, 条目宽按文字量成固定宽 |
| C3 / C5 | Card | `FullBleed(cy)` 通栏块(卡片直接子项); 卡片头宽 < 300 时动作列换到标题下方 |
| DM2 / DM3 / DM4 | Menu | `Side` / `Align` / `Offset(side, align)`; `SetRadioGroupValue` / `RadioGroupValue` / `OnRadioGroupChange`; `SetDisabled` 整菜单禁用 |
| TBL1 / TBL2 | Table | `SetCaption` / `SetFooter`: 与表格同父的兄弟横带(说明在最下, 合计行 border-t + bg-muted/50 按列对齐) |
| DT1 | Data Table | `WithDataTableRemote()` / `SetTotalRows` / `OnQueryChange(DataTableQuery)`: 服务端分页, rows = 当前页 |
| PR4 | Progress | `NewProgressLabel` / `NewProgressValue(bar, ...)`: 独立摆放的标题与数值 |
| AV5 | Avatar | `OnImageStatus(func(ok bool))`(注册即回调一次)+ `ImageLoaded()` |
| SA3 | Scroll Area | `ScrollTo` / `ScrollIntoView` / `ScrollToCenter` |
| SH6 / SH7 / AD2 / AD4 | Dialog / Sheet / AlertDialog | 遮罩配色与不透明度可配(默认对话框 10%、Sheet 50%); Sheet 进场改 2.5rem(40px)小位移 + 整组淡入(`WithSheetSlideDistance` 可回退); AlertDialog 媒体块可放图片(`WithAlertDialogMediaImage` / `SetMediaImage`, 圆角矩形蒙版) |

**下一批建议**: 分两类 —— 需要先拍板的见下表(Q1~Q4), 其余暂无(可直接排期清单已清空)。下一轮**核查**(不是实现)从 #31 Toast 起。

**待确认需求(不动手前需要拍板)**:

| # | 涉及条目 | 要确认的问题 | 建议默认 |
|---|---|---|---|
| Q1 | D4 / AD3 / SH5 / ACC4 / TT3 / TBL3 | "面板/容器里放任意控件"要不要统一设计一套接口?六条是同一件事, 分散做会出六套形态 | 统一成 `Header()/Footer()/Content()` 返回 `LayoutEle`(Card/Alert 现成形态, 调用方自己 `ItemFill`), 第一批只做 Dialog / Sheet / AlertDialog; Accordion 触发区、Tooltip 内容、Table 单元格按需 |
| Q2 | S5 / S7 / S1 / S2 | Select 的四个大项哪个要做?S5 多选成本最高(触发区摘要 + chips + 全选 + 与只读/否决的组合), S7 悬停自动滚要新增逐帧滚动带, S1 定位开关与 S2 typeahead 相对便宜 | 先做 S1 + S2(定位与键盘键入跳转); S5 / S7 等真实需求 |
| Q3 | SL4 / SL5 / SA2 / D6 / PO6 / PO4 / DT3 | 桌面端收益存疑的大项: 滑块拖动推挤(`push`)、单滑块禁用、ScrollArea 横向滚动、非模态弹层(`modal=false`)、Popover 定位策略可配、表格列宽拖拽 | 全部暂不做; 有真实场景再单独立项(SA2 若要做, 需先定"加横向模式"还是"新建 HScrollArea") |
| Q4 | C4 / I8 / TA4 / DM5 / ACC5 / SEP3 / SK2 | 两类: C4 卡片图片槽(依赖圆角裁剪, 只能做直角图片块)与 5 组纯 demo 示例(不改组件) | demo 示例可以补(便宜, 提升文档完整度); C4 只做"直角图片槽 + 自适应高", 不做圆角 |

**可直接排期(无需确认)**: 已全部落地(第八批, 见第五节)。其中 I1 文件输入 / I6 原生 Edit 覆盖 / DM2 菜单定位参数为本批最有价值的几项。AL2(Alert 描述多行)经核对由 `reflow` + `SetWrap` 覆盖(量高按 `WrappedHeight` 写回, 不与折行冲突), 本轮无需改代码。
