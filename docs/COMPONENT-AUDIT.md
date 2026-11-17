# COMPONENT-AUDIT — shadcn 组件特性核查台账

> **用途**:按组件常用性依次核查「还有哪些上游特性未实现、但 xcgui 侧可实现」,记录进度与结论,供后续逐批实现时取用。本文是仓库内部台账,**不随 `xui init` 下发**,不写设计原理(见 DESIGN.md)与完成记录(见 CHANGELOG.md)。
>
> **依据优先级**:① 官方索引页 https://ui.shadcn.com/docs/components 与各组件 `base/` 文档页(示例 / 变体 / 状态);② 文档 API Reference 指向的 Base UI 上游文档(如 Select Root props);③ 本地参照实现 `D:\GOProject\src\ui\shadcn\src\components\ui\*.tsx`(交互语义唯一权威);④ 本库 `components/*.go`、`registry.json`、`DESIGN.md` 第四节末语义表与第七节差异表。
>
> **排除项(不计入「可实现」)**:
> - **RTL / 方向性**:用户明确不做。官方几乎每个组件页都有 RTL 示例,本台账一律不列(范围决策见 DESIGN 第七节 Direction / Calendar 条目)。
> - **已记为 xcgui 能力边界的差异**:DESIGN 第七节里「改组件代码也补不上」的条目(原生 Edit 文字/光标/IME、圆角裁剪、弹层 zoom/slide 等)不算可实现项。第七节里标注「后续有需求再加」「暂不提供」的,属于本台账要列的项。**嵌套控件的 Tab 遍历已由运行时补齐**(`focus.go` 的跨层遍历, 见 DESIGN 第八节 218), 不再算能力边界。
>
> **状态**:`已核查` = 已逐条比对(备注区分「补齐已落地」与「清单待确认」);`待核查` = 按序等待;`范围外` = 明确不做。
>
> **排序口径**:人工按桌面应用中的使用频次排序 —— 表单输入与基础操作 > 反馈/浮层 > 导航 > 数据展示 > 辅助/装饰 > 聊天等新组件;同一档内按官方示例出现频次。这是**可调整的执行顺序**,不是客观统计。核查按序号推进,跳过 `已核查` 与 `范围外`;已完成 #1..#51(逐组件结论见第三节,实现进度见第五节),**下一轮核查从 #52 Spinner 起**。

## 一、按常用性排序的总表

| # | 官方组件 | 本库 key / 实现 | 状态 | 备注 |
|---:|---|---|---|---|
| 1 | Button | `button` | 已核查 | 补齐清单已实现,见 `3342cad` |
| 2 | Input | `input` | 已核查 | 结论见 3.1;8 项可实现,I2 / I7 / I3 / I4 已落地(见第五节;I3 走 Field 必填与校验) |
| 3 | Checkbox | `checkbox` | 已核查 | 补齐清单已实现,见 `3342cad` |
| 4 | Select | `select` | 已核查 | 结论见 3.2;9 项可实现,S3 / S9 / S4 / S8 已落地(见第五节;S9 走 Field 必填);S1 定位开关 / S2 typeahead 已落地(第九批) |
| 5 | Combobox | `combobox` | 已核查 | 三批补齐,见 `23bdfc5` / `725d329` / `7441a77` |
| 6 | Dialog | `dialog` | 已核查 | 结论见 3.3;6 项可实现,D1 / D2 / D3 已落地(见第五节);D4 内容槽已落地(第九批) |
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
| 26 | Alert Dialog | `alert-dialog` | 已核查 | 结论见 3.7;AD1 已落地, AD2 / AD4 见第八批, AD3 内容槽见第九批 |
| 27 | Sheet | `sheet` | 已核查 | 结论见 3.7;SH1~SH4 已落地, SH6 / SH7 见第八批, SH5 内容槽见第九批 |
| 28 | Separator | `separator` | 已核查 | 结论见 3.7;SEP1 / SEP2 已落地;余 demo 对照 |
| 29 | Skeleton | `skeleton` | 已核查 | 结论见 3.7;SK1 已落地(配色改 bg-muted);余示例卡 |
| 30 | Scroll Area | `scroll-area` | 已核查 | 结论见 3.7;SA1 已落地(滚动条常显);余横向滚动 |
| 31 | Toast | `toast` | 已核查 | 结论见 3.8;TS1 / TS2 已落地;swipe / F6 待确认 |
| 32 | Toggle | `toggle` | 已核查 | 结论见 3.8;TG1~TG4 已落地 |
| 33 | Toggle Group | `toggle-group` | 已核查 | 结论见 3.8;TGR2~TGR4 已落地(间距已有) |
| 34 | Pagination | `pagination` | 已核查 | 结论见 3.8;PG1 已落地;href/aria 为能力边界 |
| 35 | Command | `command` | 已核查 | 结论见 3.8;CMD1~CMD4 已落地;尾部勾选待确认 |
| 36 | Calendar | `calendar` | 已核查 | 结论见 3.8;CAL1~CAL5 已落地;纵向折叠待确认;RTL / Jalali / timeZone 为能力边界 |
| 37 | Date Picker | `date-picker` | 已核查 | 结论见 3.8;DP1~DP3 已落地;自然语言宽解析为能力边界 |
| 38 | Hover Card | `hover-card` | 已核查 | 结论见 3.9;HC1 默认几何/延时/聚焦已落地;HC2 动画深度暂缓 |
| 39 | Collapsible | `collapsible` | 已核查 | 结论见 3.9;CL1 否决已落地;CL2 任意布局待确认 |
| 40 | Breadcrumb | `breadcrumb` | 已核查 | 结论见 3.9;BC1 文字分隔符 / BC2 可点击省略号已落地;BC3 链接焦点待确认 |
| 41 | Navigation Menu | `navigation-menu` | 已核查 | 结论见 3.9;NM1 base-nova 几何/配色/active/图标已落地;NM2 嵌套子菜单待确认 |
| 42 | Menubar | `menubar` | 已核查 | 结论见 3.9;MB1 整条禁用/开合通知、MB4 单选组值已落地;MB2/MB3 几何与指示器待确认 |
| 43 | Sidebar | `sidebar` | 已核查 | 结论见 3.9;SB1 骨架行已落地;SB2 自定义按钮内容待确认 |
| 44 | Resizable | `resizable` | 已核查 | 结论见 3.9;文档示例已覆盖;RS1/RS2/RS4 待确认 |
| 45 | Carousel | `carousel` | 已核查 | 结论见 3.10;循环 / 自动播放 / 拖拽已具备;CR1 多页同屏待确认、CR2 页码指示可直接排期 |
| 46 | Aspect Ratio | `aspect-ratio` | 已核查 | 结论见 3.10;唯一 prop `ratio` 已覆盖,圆角裁剪是能力边界 |
| 47 | Kbd | `kbd` | 已核查 | 结论见 3.10;键帽 / 组 / 量宽已具备;KB1~KB3 暂缓 |
| 48 | Empty | `empty` | 已核查 | 结论见 3.10;三槽 + 两种媒体已具备;EM2 最大宽待确认 |
| 49 | Item | `item` | 已核查 | 结论见 3.10;三变体 / 两档 / 五槽已具备;IT1 xs 可直接排期,IT2 / IT3 待确认 |
| 50 | Native Select | `native-select` | 已核查 | 结论见 3.10;value 语义已具备;NS1 禁用可直接排期,NS2 / NS3 分组与 value 分离待确认 |
| 51 | Input OTP | `input-otp` | 已核查 | 结论见 3.10;文档示例全部覆盖;OTP1 readOnly 待确认,OTP2 / OTP3 暂缓,跨格选择为能力边界 |
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
| BA1 | `focus-visible` 环与 `aria-invalid` 校验态 | `badge.tsx` 的 `focus-visible:ring-[3px]` / `aria-invalid:border-destructive`(注意:`aria-invalid:ring-destructive/20` 只有环色、**没有环宽**, Input / Textarea / Select 才有 `aria-invalid:ring-3`) | Badge 是纯 Element,没有焦点环也没有 `SetInvalid` | 可点击形态启用焦点 + 手绘环(同 Button);补 `WithBadgeInvalid` / `SetInvalid(bool)` 只画 destructive 描边, 不画常驻红环 | 小-中 / 中 |
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

**明确不做**:RTL;视口 focus-visible 环与键盘滚动(视口是原生 LayoutFrame, 不是可聚焦控件);`rounded-[inherit]` 圆角裁剪(能力边界);Corner(只纵向时不存在);`size-full` / `data-slot`。

**本批实现顺序建议**:ACC1 → ACC2 → ACC3 → SL1 → SL2 → SL3 → SK1 → SEP1 → SEP2 → SA1 → SH1~SH4 → AD1(本批已全部落地,见第五节);ACC4 / AD3 / SH5 是同一形态(内容槽)建议一起设计;SL4 / SL5 / SA2 / SH7 需先确认需求或成本较高。

### 3.8 Toast / Toggle / Toggle Group / Pagination / Command / Calendar / Date Picker(#31 ~ #37)

上游取自 base-nova:`toast.tsx`(Base UI Toast)+ `sonner.tsx`(旧皮肤仍下发)、`toggle.tsx`、`toggle-group.tsx`、`pagination.tsx`、`command.tsx`(cmdk)、`calendar.tsx`(react-day-picker v9)、`date-picker` 文档与 6 个示例。xui 的 Toaster 一直是 **sonner 形态**(窗口内覆盖层),不是 Base UI 的视口卡片形态;这条形态差异单列为需确认。

#### Toast(#31)

**官方页**:默认 / Types / Action / Promise / API(Base UI)。Base UI 语义:`limit` 默认 3(超出的旧条以 `data-limited` 隐藏而不是删除)、`timeout` 默认 5000、折叠堆叠(`--peek:12px`、每层 `scale` 递减 0.1)、悬停展开、四向 swipe 关闭、F6 进入通知区、`add/update/close/promise` 与同 ID upsert。

**已具备**:窗口内右下角堆叠(Toaster)与屏幕级堆叠(ScreenToaster);4 种状态 + loading;richColors;动作按钮;关闭钮;负 Duration 常驻;整组悬停暂停倒计时;Promise(loading → success/error);`OnAutoClose` / `OnDismiss` 区分超时与主动关闭;`ToastOptions.ID` 同 ID 原地更新;`Toaster.Update / Close / Get` 按 ID 寻址。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TS1 | 同 ID upsert / 按 ID 关闭与查询 | manager `toast.add({id})` / `update` / `close` 与 Base UI 的 upsert | 以前只能 Show 出新卡 | 已落地:`Show` 同 ID 复用旧条目并 `Update`(保留堆叠位置与 key);`Toaster.Update/Close/Get`;退场中的旧条不复用 | 已落地 |
| TS2 | 折叠堆叠(peek / scale / 悬停展开) | `limit=3` + `data-limited`;`--peek:12px`、每层 `scale` 递减 0.1、`data-expanded` 时展开 | 容量已有(`WithToastMaxVisible`, 默认 3, 超出移除最旧), 但全部平铺展开、无层叠缩放 | 需确认:折叠视觉 + 悬停整组展开;容量沿用现有选项;若要对齐"超出隐藏不删"再引入 `data-limited` 等价状态 | 中 / 中 | **已落地**(peek 12 / 每层 0.9 / 悬停展开) |
| TS3 | 四向 swipe 关闭 | Root 的 `data-swipe-direction` | 无 | 需确认:自绘卡按住拖动 + 方向/距离判定 + 出屏退场;桌面鼠标收益低 | 中 / 低 |
| TS4 | F6 进入通知区 + Tab 遍历 | Base UI 键盘文档 | 关闭钮按钮不进 Tab 链 | 需确认:窗口级 F6 钩子 + 通知列表焦点态;要先让出 xcgui 的 Tab 链,风险大于收益 | 中 / 低 |

**明确不做**:ARIA(`role=status` / `aria-live` / `aria-label`)、RTL、Portal/DOM render、`data-swipe-ignore` 这类 DOM 属性;`priority` 与 TS2 一起考虑,不单列。

#### Toggle(#32)

**官方页**:默认 / Outline / With Text / Size / Disabled / RTL。tsx:default `h-8 min-w-8 px-2.5 gap-1`,sm `h-7 min-w-7 px-2.5 text-[0.8rem]`,lg `h-9 min-w-9 px-2.5`;悬停与按下**同为** `bg-muted`,文字 `text-foreground`;outline 描边 `border-input`;图标 16(默认)/14(sm)。

**已具备**:双状态、点击/空格/回车切换、受控初始值、图标、禁用、aria-invalid 等价的校验失败态、悬停过渡、尺寸/变体选项。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TG1 | 尺寸/内距/配色严格对齐 | 见上;`hover:bg-muted` + `data-[state=on]:bg-muted` | 旧 radix 值(悬停 accent、内距/字号不符) | 已落地:32/28/36 高,px 10,`gap 4`,字号 14/13/14,图标画布 18/16/18(视觉等价 16/14),圆角 lg/sm;悬停与按下都取 `Muted` | 已落地 |
| TG2 | 禁用态选项 | `disabled` 原生属性 | 只能 `Enable(false)` 手工调用 | 已落地:`WithToggleDisabled` / `SetDisabled` / `Disabled`;自身 / 组级 / Field 级联三层互不覆盖 | 已落地 |
| TG3 | 运行时换尺寸/变体 | className 可换 | 只能构造时定 | 已落地:`SetSize` / `SetVariant`,连量宽与布局项一起重算 | 已落地 |
| TG4 | 图标按钮宽度 | `min-w` = 高度,图标档方形 | 图标 + 文字的量宽公式 | 已具备:纯图标时宽 = 高度档(32/28/36),有文字时按内容量宽 | 已具备 |

**明确不做**:`aria-pressed` / `data-state`(DOM 元数据)、RTL。

#### Toggle Group(#33)

**官方页**:默认 / Outline / Size / Spacing / Vertical / Disabled / Custom / RTL。上游默认 `spacing=2`(=8px),**spacing=0 才是无缝**;`orientation="vertical"` 时 `items-stretch`;整组 `disabled`;条目切换可由 `eventDetails.cancel()` 否决。

**已具备**:单选/多选、组值读写、组级 variant/size、`WithToggleGroupSpacing(px)`、`WithToggleGroupConnected()`(spacing=0,首尾外角 + 相邻直角)、纵向条目排版、整组禁用、组级否决、光标/键盘切换。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| TGR1 | 纵向组 | `orientation="vertical"` + `items-stretch` | 旧版只能横向 | 已落地:`WithToggleGroupVertical` / `SetVertical`;组与条目统一成组内最宽,纵向首尾外角、相邻边直角,connected 时描边合并 | 已落地 |
| TGR2 | 整组禁用 | 组级 `disabled` | 只能逐条禁用 | 已落地:`WithToggleGroupDisabled` / `SetDisabled`;只忽略用户交互,程序化 `SetValue` 仍生效;条目自身禁用状态独立保留 | 已落地 |
| TGR3 | 可否决切换 | `eventDetails.cancel()` | 无 | 已落地:`OnValueChangeVeto(func(values []string) bool)`,参数是切换后的完整集合;否决时回滚按下态,用户回调不触发 | 已落地 |
| TGR4 | 任意间距 | `spacing` 数值 | 已有 | 已具备:默认 8(上游 spacing=2),0 = 无缝 | 已具备 |

**明确不做**:`data-spacing` / `data-orientation` 等 DOM 属性、RTL 镜像。

#### Pagination(#34)

**官方页**:默认 / Simple / Icons Only / Next.js / RTL。tsx:上一页/下一页是 Button `size="default"`(`h-8 px-2.5 gap-1.5`)再单侧覆盖 `pl-1.5!` / `pr-1.5!`;页码链接默认 `size="icon"`(32×32);省略号 `size-8`。

**已具备**:完整 / 简洁 / 仅导航 / 仅图标四种形态;折叠槽位固定 7 个;上一页/下一页文案可配;当前页 outline、其余 ghost;方向键 / Home / End 移动焦点 + Enter 激活;禁用边界 50% 透明。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| PG1 | 上一页/下一页内距与图标间距 | 靠图标侧 6(`pl-1.5!`)、靠文字侧 10(`px-2.5`)、图标-文字 `gap-1.5`=6 | 两侧都是 6,图标-文字 4 | 已落地:宽度 = 6 + 16 + 6 + 文字 + 10(Next 对称);隐藏文字仍是 32 方块(无 gap) | 已落地 |
| PG2 | 链接尺寸档 | `PaginationLink size` 可换 | 固定 32 方形 | 暂缓:桌面分页没有换档场景,确有需求再加 `WithPaginationLinkSize` | 小 / 低 |

**明确不做**:`href` / Next.js `<Link>`(桌面用 `OnPageChange`,不渲染 DOM 链接)、`aria-current` / `aria-label`、`hidden sm:block` 的响应式断点(桌面对应显式 `WithPaginationHideText`)、RTL。

#### Command(#35)

**官方页**:默认 / Shortcuts / Groups / Scrollable / RTL。tsx(cmdb):根 `rounded-xl! bg-popover p-1 text-popover-foreground`;搜索行是 `p-1 pb-0` 包着的 InputGroup(`h-8 rounded-lg border-input/30 bg-input/30`,搜索图标 `pl-2`);列表 `max-h-72`(288);条目 `rounded-sm px-2 py-1.5 text-sm`,`data-selected:bg-muted` 且文字/图标/快捷键转 `foreground`;自带尾部 CheckIcon(`data-checked`);快捷键 `text-xs tracking-widest`。

**已具备**:搜索行 + 滚动列表 + 分组标题 + 条目图标/快捷键/禁用;自定义过滤;↑/↓ 循环跳过禁用、Home/End、PageUp/PageDown、Enter 选中、ESC;可选前置勾选框与右侧计数(分面筛选);底部"分割线 + 清除行";`CommandDialog` 可由 Dialog + Command 组合。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| CMD1 | 选中底色与文字 | `data-selected:bg-muted` + `text-foreground`(图标/快捷键同步) | `Accent` / `AccentForeground` | 已落地:选中 `Muted` + `Foreground`;图标与快捷键也在选中态转 `Foreground` | 已落地 |
| CMD2 | 根圆角 / 列表上限 | `rounded-xl!`;`max-h-72` | `RadiusMd`;300 | 已落地:`RadiusXl`;288 | 已落地 |
| CMD3 | 自定义匹配 | cmdk 的 `filter`(默认 fuzzy score) | 只有小写包含 | 已落地:`WithCommandFilter` / `SetFilter(func(text, keywords, query string) bool)`,query 已小写去空白;传 nil 回默认 | 已落地 |
| CMD4 | 搜索行 InputGroup 化 | wrapper `p-1 pb-0` + 32 高圆角 chip(`bg-input/30` + `border-input/30` + 图标) + 根 `p-1` | 整宽 36 行 + 下边框 | 需确认:重建为"根 4 + 行内 4 + 32 chip",去掉下边框;会改 DatatableFacet 的精确高度与 demo 几何,需要整页回归 | 中 / 中 | **已落地**(含 DatatableFacet 高度) |
| CMD5 | 尾部勾选指示 | 条目自带 `ml-auto` CheckIcon(`data-checked`) | 分面筛选是前置勾选框 | 需确认:分面筛选保留前置框;另给 `WithCommandItemCheckMark()`(单选用尾部勾) | 小 / 低 |
| CMD6 | 快捷键字距 | `tracking-widest` | 12px 常规字距 | 不做:xcgui 文本字距没有等价 API,用空格排版会破坏对齐 | 不做 |

**明确不做**:cmdk 的 vim 绑定 / fuzzy 高亮 / `aria-activedescendant` 等 DOM 语义、RTL、任意 React children。

#### Calendar(#36)

**官方页**:Basic / Range / Month-Year Selector / Presets / Date and Time / Booked dates / Custom Cell Size / Week Numbers / RTL / API。tsx(react-day-picker v9):根 `bg-background p-2`,`--cell-size:--spacing(7)`(28),`--cell-radius:radius-md`;`captionLayout` 默认 `"label"`,`showOutsideDays` 默认 true;范围中段 `bg-muted`;键盘网格由 react-day-picker 提供。

**已具备**:单选/范围、多月并排(1~4)、年月下拉、周数列、页脚与预设、禁用/划线日期、固定 6 行不跳高、宽度驱动等比缩放、`WithCalendarCaptionLabel`、`WithCalendarHideOutsideDays`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| CAL1 | 纯文字标题 | `captionLayout="label"`(默认) | 只有年月下拉 | 已落地:`WithCalendarCaptionLabel()` 画居中的"2026年10月",无热区/chevron,左右翻页保留 | 已落地 |
| CAL2 | 隐藏越界日期 | `showOutsideDays=false` | 恒定显示 | 已落地:`WithCalendarHideOutsideDays()`;越界格留空、不可点、无悬停,6 行高度不变 | 已落地 |
| CAL3 | 键盘网格 | react-day-picker:方向键移动、PageUp/Down 换月、Shift+PageUp/Down 换年、Home/End 到行首尾、Enter/Space 选中 | 元素不进焦点链 | 需确认:日历开 `EnableFocus` + focused day(初始 = 选中/今天),按 7 列移动并自动翻月;焦点环 3px ring/50%;要验证在 Popover 内的 Tab 顺序 | 中-高 / 中 | **已落地** |
| CAL4 | 默认几何/皮肤 | cell 28 / `p-2` / `bg-background`;范围中段 `bg-muted` | cell 32×1.12、`p-3`、`bg-popover` + ring、中段 accent | 需确认:按上游改基准(28 / p-2 / 背景色透明,中段 Muted),DatePicker 弹层尺寸同步;担心 100% DPI 可读性时用 CAL5 放大,而不是默认 1.12 | 中 / 高 | **已落地** |
| CAL5 | 自定义格子尺寸 | `--cell-size` 任意值/响应式 | 只按元素宽度等比缩放 | 与 CAL4 合并:`WithCalendarCellSize(px)` 映射到缩放比,0 = 跟随宽度 | 小-中 / 中 | **已落地** |
| CAL6 | 多月窄容器纵向堆叠 | `flex-col md:flex-row` | 始终横排,窄容器靠夹取 | 暂缓:容器宽 < 两月最小宽时改纵向;桌面半宽面板少见 | 中 / 低 |

**明确不做**:Jalali / Hijri / Persian 日历、指定 timeZone、完整 locale 数据(周起始与中文星期已可配)、RTL、DOM aria 元数据。

#### Date Picker(#37)

**官方页**:Basic / Range / Date of Birth / Input / Time Picker / Natural Language / RTL。基础示例 `align="start" sideOffset={4}`;Input 与 Natural Language 用 InputGroup + `align="end" alignOffset={-8} sideOffset={10}`,输入过程中一旦解析成功就同步选中日期。

**已具备**:自绘触发区(outline 按钮) + 自绘月历弹层;单选/范围/预设/年份范围/禁用日期;可输入形态(InputGroup,支持中文日期与相对日期,Enter/失焦回写);编程 `Open/Close/SetValue/SetRange`;`OnChange` / `OnRangeChange`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| DP1 | 弹层开合回调 | Popover 的 `onOpenChange` | 无 | 已落地:`WithDatePickerOnOpenChange` / `OnOpenChange`;打开(含编程 `Open`)true,选中 / ESC / 失焦 / `Close` 走弹层 `OnHide` 回 false | 已落地 |
| DP2 | 输入即解析 | Input / NaturalLanguage 示例在 onChange 里解析并更新 date | 只在 Enter/失焦 | 已落地:文字一构成合法日期就更新选中值并回调 `OnChange`;解析失败保持原值、不回写输入框(不打断键入) | 已落地 |
| DP3 | 可输入形态的弹层定位 | `align="end" alignOffset={-8} sideOffset={10}` | 一律左对齐、间距 4 | 已落地:`popoverPanel.ShowAligned`;可输入形态自动右对齐 / 10 / -8,普通形态仍左对齐 / 4 | 已落地 |
| DP4 | 自然语言宽解析 | chrono-node(英文短语、复杂相对日期) | 今天/明天/后天/大后天、N 天后、N 周后与常见日期格式 | 能力边界:Go 侧不引入 chrono-node 等价物;需要更宽语法时用 `OnChange` + `SetValue` 自定义 | 不做 |

**明确不做**:RTL、指定 timeZone、React 受控/非受控语义;时间选择由独立 `TimePicker` 组件承担(组合式,不塞进 DatePicker)。

**本批实现顺序**:TS1 → TG1~TG3 → TGR1~TGR3 → PG1 → CMD1~CMD3 → CAL1 / CAL2 → DP1~DP3(本批已全部落地);TS2、CMD4、CAL3~CAL5 已按第五节 Q5 拍板落地;CMD5 与 swipe / F6 仍待需求;其余为能力边界或暂缓。

### 3.9 Hover Card / Collapsible / Breadcrumb / Navigation Menu / Menubar / Sidebar / Resizable(#38 ~ #44)

上游取自 base-nova:`hover-card.tsx`(Base UI PreviewCard)、`collapsible.tsx`(Base UI Collapsible)、`breadcrumb` 文档(本地无 tsx)、`navigation-menu.tsx`(Base UI NavigationMenu)、`menubar.tsx`(Base UI Menubar + DropdownMenu)、`sidebar.tsx`、`resizable.tsx`(react-resizable-panels v4)。Hover Card 官方示例显式传 `delay={10} closeDelay={100}`,Base UI Trigger 的默认值是 **600/300**(本库取默认);Sidebar / Resizable 没有 Base UI 底层,以 shadcn 文档与 tsx 为准。

#### Hover Card(#38)

**官方页**:Basic(delay 10 / closeDelay 100 覆盖)、Trigger Delays、Positioning(side/align)、Sides、RTL。tsx:`w-64 p-2.5 rounded-lg bg-popover text-sm shadow-md ring-1 ring-foreground/10`,sideOffset 4 / alignOffset 4;进场 `fade-in-0 + zoom-in-95` 加按方位的 `slide-in-from-*`(100ms)。Base UI PreviewCard 默认 `delay=600` / `closeDelay=300`,键盘聚焦触发区即打开(reason `trigger-focus`),失焦按 closeDelay 关闭。

**已具备**:任意元素触发;600ms 悬停打开 / 300ms 离开关闭;指针移入卡片保持打开;非激活弹层不抢焦点、卡片内容可鼠标交互;宽度 256 / 内边距 10 / side + align + alignOffset + sideOffset 可配(默认 bottom / center / 4 / 4);编程 `Open/Close/SetOpen/IsOpen`、`OnOpenChange`、`Refresh`、内容构建后测高;工作区夹取与翻转;主题令牌实时取色。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| HC1 | 默认几何/延时 + 聚焦打开 | tsx 的 w-64 / p-2.5 / sideOffset 4 / alignOffset 4;Base UI 默认 delay 600 / closeDelay 300、`trigger-focus` | 700ms、卡片 288 / 内边距 12、无 alignOffset,不支持聚焦打开 | 已落地:600/300、256/10/4/4,`WithHoverCardAlignOffset`;触发区 `SetFocus` 立即打开、`KillFocus` 按 300ms 关 | 已落地 |
| HC2 | 进场 fade + zoom + slide | tsx 的 `animate-in fade-in-0 zoom-in-95` 与 `slide-in-from-*` | 共享弹层只做整面板淡入 | 需确认:分层阴影窗的整窗缩放/位移有闪烁与崩溃风险(见 DESIGN 第八节 47/64),元素级缩放只能逐帧改矩形;暂缓,与其它弹层统一考虑 | 中 / 低 |

**明确不做**:Portal / Container / `render` prop、`duration` 数值、`data-side` / `data-open` 等 DOM 属性、RTL。

#### Collapsible(#39)

**官方页**:Basic / Settings Panel / File Tree / Controlled State / RTL;tsx 是 Base UI Collapsible 的薄封装(Root / Trigger / Panel)。Basic 的根加 `data-open:bg-muted`;Trigger 可换任意元素(render),Panel 可放在根树的任意位置(示例放在两列栅格里,触发区在栅格外侧)。

**已具备**:`open` / `SetOpen` / `Toggle` / `OnOpenChange`;多个 Trigger 绑定(点击 / 空格 / 回车);Disabled(程序化 `SetOpen` 仍生效);内容高度 200ms 过渡、收起后隐藏且不占布局;嵌套折叠(文件树);主题实时取色。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| CL1 | 用户切换可否决 | Base UI Root 的 `onOpenChange` 里 `eventDetails.cancel()` | 无 | 已落地:`OnOpenChangeVeto(func(open bool) bool)`;点击 / 空格回车 / Toggle 走否决,程序化 `SetOpen` 不经过;被否决时不改状态、不播过渡 | 已落地 |
| CL2 | 触发区/内容区任意布局 | 根是普通 div,Trigger / Panel 可分别摆(设置面板的内容在栅格里、触发区在旁) | 根固定竖向流,内容固定排在全部触发区之后 | 待确认:需要把内容改为"可挂到任意父级"的独立元素,由调用方声明位置并仍由组件接管高度;会改动根管线与自动高度测量 | 中-高 / 中 |
| CL3 | 展开态根底色 | Basic 示例的 `data-open:bg-muted` | 根透明 | 暂缓:调用方可用 Card / 自绘容器包一层;做成选项只省一行 | 小 / 低 |
| CL4 | 内容是否常驻(`keepMounted`) | Base UI 默认收起时卸载 Panel | 收起只把高度归 0,元素常驻 | 不做:常驻能保留输入控件状态,桌面侧更符合预期;卸载语义需要调用方重建 UI,没有对应需求 | 不做 |

**明确不做**:`data-state` / `aria-expanded` 等 DOM 属性、RTL、任意 React children。

#### Breadcrumb(#40)

**官方页**:Basic / Custom separator / Dropdown / Collapsed / Link component / RTL;列表 `flex flex-wrap items-center gap-1.5 text-sm`,链接 `hover:text-foreground`,当前页前景色;分隔符默认 ChevronRight(14),可传任意 children;省略号 36×36 + MoreHorizontal(16);省略号常与 DropdownMenu 组合。

**已具备**:列表自动换行;条目 gap 6;链接 hover 过渡;当前页;图标 / 文字分隔符(文字按字号量宽、换主题重量);省略号 36×36 可选点击(悬停底色、键盘焦点环、Enter / 空格);主题令牌。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| BC1 | 文字分隔符 | `BreadcrumbSeparator` 的 children 可为任意内容(官方示例用 DotIcon) | 只能图标 | 已落地:`NewBreadcrumbSeparatorText(parent, "/")`,muted 单行、按字号量宽 | 已落地 |
| BC2 | 可点击省略号 | 官方 Dropdown 示例把省略号包在 DropdownMenuTrigger 里 | 纯装饰 | 已落地:`WithBreadcrumbEllipsisOnClick`;悬停 Accent 底、焦点环、Enter / 空格触发;下拉菜单由回调里自行组合 | 已落地 |
| BC3 | 链接进入 Tab 链 | `BreadcrumbLink` 渲染 `<a>`,天然可聚焦、Enter 激活 | 链接是自绘文本,不参与焦点链 | 待确认:需给每个链接加可聚焦元素 + 键盘焦点环 + Enter / 空格,会改变整页 Tab 顺序;现有差异见 DESIGN 第七节 | 小-中 / 中 |
| BC4 | 链接/当前页的任意内容 | children 可为图标 + 文字或自定义组件 | API 只接受 text | 暂缓:需要内容槽或接受元素句柄;面包屑以纯文字为主 | 中 / 低 |

**明确不做**:`href` / Next Link(桌面用 `OnClick`)、`aria-current` / `aria-hidden`、RTL。

#### Navigation Menu(#41)

**官方页**:Default(3 个触发区 + ListItem + With Icon + 顶级普通链接)、Link Component、RTL;tsx viewport 模式:列表 gap-0,触发区 `h-9 px-2.5 rounded-lg text-sm`,悬停 / 展开 `bg-muted`(展开 50%),视口 sideOffset 8 + p-1;ListItem 标题 + `line-clamp-2` 说明,内容链接可带 16px 图标,当前页 `bg-muted/50`。

**已具备**:共享视口;悬停 50ms 打开 / 离开 50ms 收起、关闭后 300ms 跳过延时;点击开合;内容切换 350ms 滑入 208 + 淡入;chevron 300ms 旋转;顶级普通链接;链接前置图标、active 底色;标题 + 说明行(说明可空);多列网格(`LinkIn`);键盘 ←→ / ↓ / Enter / 空格 / ESC / Tab、Home / End;禁用项;值回调。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| NM1 | base-nova 几何/配色/图标/当前页 | 见上 | 老 radix 值:悬停 accent、无图标 / active | 已落地:gap 0 / p-1 / sideOffset 8;悬停 / 按下 / 展开取 `Muted`(展开 50%,悬停回全亮);链接 `SetIcon` / `SetActive`(`bg-muted/50`);视口圆角 `RadiusLg` | 已落地 |
| NM2 | 嵌套子菜单 | Base UI NavigationMenu 支持内容里再开一层 | 只有一级内容页 | 待确认:共享视口里要再弹子视口并维护 hover intent / 键盘层级,成本高;常见导航用一级网格已够 | 高 / 低 |
| NM3 | 纵向 `orientation` | Root 的 orientation | 只有横向 | 暂缓:侧边导航用 Sidebar;顶部横向是主场景 | 中 / 低 |
| NM4 | 视口宽高过渡 | Base UI 弹层的尺寸过渡 | 切页时宽高即时变化,只有内容滑入 | 需确认:宽高过渡要逐帧重排内容,且视口贴在列表左下;当前与官方差距主要在内容滑入,已具备 | 中 / 低 |
| NM5 | 说明两行(`line-clamp-2`) | ListItem 说明最多两行 | 固定一行 20,长文省略 | 暂缓:两行会让视口高度随文案跳变;长文案建议缩短或自定义内容 | 小 / 低 |

**明确不做**:Sub / `viewport=false` / Indicator(见 DESIGN 第七节)、DOM `data-*` 属性、RTL。

#### Menubar(#42)

**官方页**:Default(File / Edit / View / Profiles)、Checkbox、Radio、Submenu、With Icons、RTL;tsx(base-nova):根 `h-8 p-[3px] gap-0.5 rounded-lg border`,触发区 `px-1.5 py-[2px] text-sm medium`,悬停 / 展开 `bg-muted`;弹层 `min-w-36 p-1 rounded-lg`,条目 `px-1.5 py-1`(28),勾选 / 单选指示器在右(`pr-8` / `right-2`),子菜单 `min-w-24`,内联图标 16、inset 文字 28。

**已具备**:自绘根条 + 每个菜单一个弹层;悬停切换;点击开合;条目 / 勾选 / 单选(组内互斥)/ 子菜单 / 快捷键 / Label / 分隔 / 禁用 / destructive / 图标 / inset;键盘 ↑↓ / Home / End / Enter / ←→ / ESC / Tab;子菜单非激活不抢焦点;整条禁用 + `OnOpenChange`;单选组按组读写值;主题实时取色;工作区夹取。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| MB1 | 整条禁用 + 开合通知 | Base UI Menubar Root 的 `disabled` / `onOpenChange` | 只能逐菜单禁用,无整条状态 | 已落地:`WithMenubarDisabled` / `SetDisabled` / `Disabled`(触发器置灰、鼠标与键盘均不响应,已开的弹层关闭);`OnOpenChange(bool)` 只在"打开任一 / 全部关闭"变化时发 | 已落地 |
| MB2 | base-nova 根/条目几何 | h-8 p-[3px] gap-0.5、条目 28、弹层 min-w-36、子菜单 min-w-24 | 根 36 / 内距 4 / 间距 4、条目 32、弹层 192、子菜单 128 | 待确认:触发区要 24 高 + 6 内距,现 Button 没有高度 / 内距覆盖;需给 Button 加覆盖选项或改自绘触发区,再整页回归 | 中 / 中 |
| MB3 | 勾选/单选指示器在右 | base-nova 的 check / radio 在 `right-2`,文字从 6 起(单行图标在左) | 指示器在左,文字从 32 起 | 待确认:要改行绘制、宽度公式与快捷键让位;与 Menu(原生菜单)已右侧不一致,建议与 MB2 同批做 | 中 / 中 |
| MB4 | 单选组 value / onValueChange | MenubarRadioGroup 的 `value` / `onValueChange` | 只有每项 checked + `MenuGroup` / `MenuOnCheck` | 已落地:`RadioGroupValue` / `SetRadioGroupValue` / `OnRadioGroupChange`(条目文本即组值,与 Menu 同一口径);组内互斥已具备 | 已落地 |

**明确不做**:Alt 键访问 / typeahead(见 DESIGN 第七节)、RTL、DOM 属性。

#### Sidebar(#43)

**官方页**:SidebarProvider(structure / width / 快捷键 / controlled)、Sidebar props、useSidebar、Header / Footer / Content / Group / Menu / MenuButton / MenuAction / Sub / Badge / Skeleton / Trigger / Rail、Theming、Styling、RTL;tsx:展开 256 / 图标 48;分组 p-2;菜单按钮 h-8;骨架行 h-8 px-2 gap-2(16 图标块 + 16 高文字条,宽度随机 50~90%)。

**已具备**:单对象合并 Provider + 面板;side / variant(3)/ collapsible(3);宽度 / 图标宽 / 初始开合;Ctrl+B;rail;trigger;Header / Content(ScrollArea)/ Footer;Group(label / action / menu);菜单项 variant / size / active / badge / tooltip / disabled / action(showOnHover);子菜单;icon 折叠与 tooltip;200ms 宽度过渡;骨架行;主题令牌。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| SB1 | 菜单骨架行 | `SidebarMenuSkeleton`(h-8、图标块 + 50~90% 文字条、animate-pulse) | 无 | 已落地:`NewSidebarMenuSkeleton` / `SidebarGroup.AddSkeleton`;`WithSidebarSkeletonIcon` / `WithSidebarSkeletonWidth`(10~100);2s 不透明脉冲;图标栏只留图标块 | 已落地 |
| SB2 | 菜单按钮自定义内容 | `SidebarMenuButton` 接受任意 children(团队切换器 / 用户卡:头像 + 两行文字 + 尾部 chevron) | 只有 text + icon 的固定形态 | 待确认:需要内容槽(`Content()` 返回布局容器)与折叠态下的显隐约定;现有 Header / Footer 可自己拼容器绕过 | 中 / 中 |
| SB3 | 分组可折叠 | 文档用 Collapsible 包 SidebarGroup,组标签作 Trigger | 分组标签不是触发区;可用外部 Collapsible 包组,但标签无法绑定 | 暂缓:需要标签支持触发器绑定或开放自定义标签元素;真实需求再加 | 小-中 / 低 |
| SB4 | 移动端 Sheet / Provider 上下文 | `isMobile` 断点 + Sheet;`useSidebar` | 无移动分支,状态在 Sidebar 实例上 | 不做:桌面窗口没有移动断点;Go 侧显式持有 `Sidebar` 指针即可;`SidebarInput` 用 Input 组合 | 不做 |

**明确不做**:RTL、rem / CSS 变量宽度、DOM `data-*` 属性、移动断点。

#### Resizable(#44)

**官方页**:Default(嵌套组合)、Vertical、Handle、RTL;底层 react-resizable-panels v4(orientation / `defaultSize` 百分比 / min / max / collapsible / collapsedSize / onCollapse / onExpand / onLayoutChange / F6 / preserve-pixel-size / autoSaveId)。

**已具备**:横 / 纵向;百分比尺寸 + min / max;拖动(4px 热区 + 1px 线,可选 4×24 抓手);键盘 5% / Home / End、双击复位;`OnResize` / `SetSizes` / `Sizes`;组级禁用;嵌套分组;相对比例保持;尺寸变化按累计边界取整。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| RS1 | collapsible / collapsedSize + Enter 折叠 | 上游 Panel 的 `collapsible` / `collapsedSize`,Enter 折叠 / 展开 | 无折叠,只能靠 min 限制 | 待确认:折叠态要保留 collapsedSize、双击 / 方向键恢复,与 min 语义叠加;需要面板级状态与拖动阈值 | 中 / 中 |
| RS2 | preserve-pixel-size | 上游 PanelGroup 的 `preserve-pixel-size` | 始终保持相对比例 | 待确认:做成开关后窗口缩放时保住像素宽、压缩其它面板;需要重算归一化口径 | 中-高 / 低 |
| RS3 | F6 在分隔条间跳焦点 | 上游键盘文档 | 分隔条可 Tab 聚焦,无 F6 循环 | 暂缓:窗口级 F6 钩子与其它组件的 F6 预留冲突 | 中 / 低 |
| RS4 | 面板 / 分隔条级禁用 | 上游没有对应 prop(只能整组);本库扩展 | 只有组级 disabled | 暂缓:有单栏锁定场景再扩展 | 小 / 低 |
| RS5 | 布局持久化(`autoSaveId`) | 上游 PanelGroup 的 autoSaveId(localStorage) | 无 | 不做:Go 侧由调用方保存 `Sizes()` 并在下次构造时 `SetSizes`,更可控 | 不做 |

**明确不做**:RTL、任意尺寸单位(px / rem)、DOM `data-panel`。

**本批实现顺序**:HC1 → CL1 → BC1 / BC2 → NM1 → MB1 / MB4 → SB1(本批已全部落地);其余按 Q6 分流:MB2 / MB3、CL2 / SB2、BC3 / NM2、RS1 / RS2 / RS4 需先拍板;HC2、NM3~NM5、CL3、SB3、RS3 暂缓;能力边界见各组件"明确不做"。

### 3.10 Carousel / Aspect Ratio / Kbd / Empty / Item / Native Select / Input OTP(#45 ~ #51)

上游取自 base-nova:`carousel.tsx`(embla-carousel-react)、`aspect-ratio` 文档(Base UI AspectRatio,本地无 tsx)、`kbd.tsx`、`empty` 文档(本地无 tsx)、`item` 文档(本地无 tsx)、`native-select.tsx`、`input-otp.tsx`(input-otp 包)。本批多数核心特性已具备(Carousel 的循环 / 自动播放 / 拖拽吸附、InputOTP 的过滤 / 粘贴 / 完成回调均早于本轮落地);缺口集中在"官方示例第一档参数"(Carousel 的 basis、Item 的 xs、NativeSelect 的分组)与"DOM 组合能力"(render / children、头像图片媒体)两类。

#### Carousel(#45)

**官方页**:Basic(5 页 + 左右圆形按钮)、Sizes(`basis-1/3`、`md:basis-1/2 lg:basis-1/3`)、Spacing(`-ml-4` + `pl-4`)、Orientation(`horizontal` / `vertical`)、Options(`opts.align` / `opts.loop`)、API(`setApi` + `scrollSnapList()` / `selectedScrollSnap()` / `on("select")`)、Events、Plugins(embla autoplay)。tsx 是 embla 薄封装:`orientation` 选 axis、捕获阶段 ←/→ 翻页、到边界禁用按钮;`overflow-hidden` 视口 + `basis-full` 页;按钮 outline + icon-sm + rounded-full,横向在 `-left-12` / `-right-12`,纵向 `rotate-90` 放上下。

**已具备**:横 / 纵向;整页位移、gap 默认 16、循环无回卷跳帧;拖拽跟手 + 最近页吸附 + 两端阻尼、←/→ 在轮播内任意控件聚焦时翻页;上一张 / 下一张圆钮到边界禁用(loop 常可用),`PrevButton()` / `NextButton()` 可改体;初始页、300ms 过渡、`Index()` / `SlideCount()` / `OnChange` / `SetIndex` / `Next` / `Prev`;自动播放 `WithCarouselAutoplay`(用户交互后停、页面不可见不跑);按钮槽 48×2;主题令牌实时取色。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| CR1 | 多页同屏(`basis-1/2` / `basis-1/3`) | Sizes 示例 | 一页 = 视口宽 − gap,步进 = 视口宽 | 待确认:加"每屏页数"配置,页宽 = 视口/每屏页数 − gap、步进 = 视口/每屏页数;循环位移折回与边界按钮要同步改 | 中 / 中 |
| CR2 | 页码指示(current / count) | API 示例的 `setApi` + selectedScrollSnap / scrollSnapList | `Index()` / `SlideCount()` / `OnChange` 已有,无内置点状指示器 | 可直接排期:新增 `Carousel.Dots()` 返回自绘圆点行,跟随 `OnChange` 换当前页 | 小-中 / 中 |
| CR3 | 页对齐(`opts.align: start / center / end`) | Options 示例 | 固定 start(整页左 / 上对齐) | 暂缓:与 CR1 一起设计;单页铺满时 center / end 与 start 等价 | 中 / 低 |
| CR4 | 按钮变体/几何覆盖 | tsx 的 Previous / Next 透传 Button props | 按钮对象已暴露,位置固定在 48 槽 | 暂缓:不要按钮用 `WithCarouselControls(false)` 自放;槽宽可变要重排根几何 | 小 / 低 |

**明确不做**:任意 embla `opts` / `plugins` 透传(Go 侧没有插件生态,loop / autoplay / gap / duration / 初始页已逐项做成选项);velocity 惯性 / wheel / 触摸手势(DESIGN 第七节已记桌面口径);高度按内容自撑(xcgui 无可靠离屏量高,构造时显式给);`setApi` / DOM `data-*` / RTL。

#### Aspect Ratio(#46)

**官方页**:Usage(`ratio={16/9}` + 任意子元素)、Square(1/1)、Portrait(9/16);API 只有必填 `ratio: number` 与 `className`。tsx 是 Base UI AspectRatio 薄封装,容器按 ratio 定高,内容铺满。

**已具备**:`NewAspectRatio(..., ratio, ...)` / `SetRatio` / `Ratio`、`Content()` / `ContentStack(space)`;宽由布局项决定、高 = 宽 / ratio;尺寸事件排队到下一轮,同批先写完全部高度再统一重排祖先(避免多块逐帧展开与 335↔0 自激);构造 `cy > 0` 或布局项声明 Fill / Weight 时比例不生效(调用方自管高);非法比例收 1:1。

**未实现但可实现**:无。上游唯一 prop 已覆盖;比例框内圆角图片裁剪不是组件能补的(见下)。

**明确不做**:圆角裁剪 / `overflow-hidden`(xcgui 按父矩形裁剪但没有圆角裁剪,见 DESIGN 第七节与第八节 97);任意 children / `render` / DOM 属性;RTL。

#### Kbd(#47)

**官方页**:只有 `Kbd` 与 `KbdGroup` 两个组件,无变体 / 状态。`h-5 min-w-5 w-fit px-1 rounded-sm bg-muted text-xs font-medium text-muted-foreground`,`pointer-events-none select-none`;组合键用 KbdGroup 的 `gap-1` 分隔;Tooltip 内容里反色(`bg-background/20 text-background`);内嵌 SVG 取 12px。

**已具备**:`NewKbd(hParent, text)` / `SetText`,20 高 + 按文字量宽(最小 20)+ rounded-sm + `Muted` 底 / `MutedForeground` 字 + 4px 内距 + 主题字号 -2 / medium;`NewKbdGroup` 4px 间距;纯展示、不聚焦、不接鼠标;换主题与改文字都重新量宽。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| KB1 | Tooltip 内容里反色 | `in-data-[slot=tooltip-content]:bg-background/20` | `WithTipShortcut` 已在气泡里放 Kbd,但仍是常规 muted 键帽 | 暂缓:给 Kbd 加 `SetContrast(true)`(按气泡底取前景 / 半透明底),由 Tooltip 组合时开启 | 小 / 低 |
| KB2 | 键帽内嵌 12px 图标 | `[&_svg:not([class*='size-'])]:size-3` | 只接受文字,图标只能并排放进 KbdGroup | 暂缓:给 `NewKbd` 加图标参数(字 + 12px 图标一起量宽) | 小 / 低 |
| KB3 | 几何覆盖(h / min-w / px / 字号) | `className` 可覆盖工具类 | 常量固定 | 暂缓:加 `WithKbdHeight/Padding/FontSize` 等选项;现场景用默认 20 足够 | 小 / 低 |

**明确不做**:`data-slot` / `aria-*` 等 DOM 属性、任意 `children` / `render`、RTL、按键点击语义(纯展示)。

#### Empty(#48)

**官方页**:Composition(`Empty > Header(Media / Title / Description) + Content`)、Outline(`border` 工具类)、Background(`bg-*` / `bg-gradient-*`)、Avatar 与 Avatar Group(放进 `EmptyMedia`)、InputGroup(放进 `EmptyContent`);API 只有 `className`,Media 的 `variant = default | icon`。

**已具备**:`NewEmpty` + `Body()` / `Header()` / `Content()` 三槽(预建隐藏,不用的槽不占间距);媒体 `NewEmptyMedia` + `WithEmptyMediaVariant`(default 透明 / icon = 40 圆角 `Muted` 底板 + 24 图标);标题 `NewEmptyTitle`(18 / medium)、描述 `NewEmptyDescription`(14 / muted);竖向居中、p-6 / gap-6 / Header gap-2 / Content gap-4;不画边框(上游 `border-dashed` 无宽度);头像 / 头像组 / InputGroup 直接往 `Header()` / `Content()` 加元素即可;描述多行由调用方对返回 Label 调 `SetAutoWrap()`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| EM1 | Outline / Background 选项 | 官方两页示例用 `border` / `bg-*` 工具类 | 不画边框、无背景选项 | 暂缓:官方就是让用户自己加类;Go 侧放进 Card 或给 `Body()` 自绘即可,做选项只省一行;渐变背景另议 | 小 / 低 |
| EM2 | Header 最大宽 384 | 上游 Header `max-w-[384px]` | Header 整宽(描述长文本会铺得很宽) | 待确认:要给 Header 加"最大 384 且窄容器取父宽"的钳制,需要在尺寸事件里量父宽 | 中 / 低 |
| EM3 | 响应式 md:p-12 | 上游 `md:` 断点 | 统一用基础档 p-6 | 不做:xcgui 没有容器查询,窗口宽度断点与上游容器断点不是一回事 | 不做 |

**明确不做**:DOM `className` / `data-*`、任意 `children` / `render`、RTL;Media 的 Avatar / InputGroup 组合已由槽位承载,不新增专用变体。

#### Item(#49)

**官方页**:Basic(Media + Content + Action)、Variant(default / outline / muted)、Size(default / sm / xs)、Icon / Avatar / Image(`ItemMedia` 的 `variant = default | icon | image`)、Group / Separator、Header / Footer、Link(`render={<a/>}`)、Dropdown;Composition 是 `ItemGroup > Item > ItemHeader / ItemMedia / ItemContent(Title / Description) / ItemActions / ItemFooter`。

**已具备**:`NewItem(... WithItemVariant(ItemDefault/Outline/Muted), WithItemSize(ItemSizeDefault/Sm))`;`Media()` / `Content()` / `Actions()` / `Header()` / `Footer()` 五槽(不用不占布局);整宽 1px `NewItemSeparator` 与 `NewItemGroup`;`NewItemTitle` / `NewItemDescription`、`NewItemMedia` + `WithItemMediaVariant(Default/Icon)`;列举与链接型内容都可直接往槽里放任意控件;变体 / 尺寸几何(default gap-16 / p-16,sm gap-10 / px-16 py-12)、Content 按媒体/动作实测宽锁 Fixed(见第八节 104)、主题实时取色。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| IT1 | xs 尺寸档 | Size 示例(default / sm / xs) | 只有 default / sm | 可直接排期:补 `ItemSizeXs`,把 gap / padX / padY 抽成按档查表,媒体尺寸同步 | 小 / 中 |
| IT2 | Media 的 image / avatar 内建变体 | Image / Avatar 示例 | 只有 default / icon,图标是自绘 path;`Media()` 是布局槽,可直接挂 `NewAvatar` 或图片元素,只是没有内建变体 | 待确认:新增 `WithItemMediaImage(...)` 走本地图片 / `image.Image`;圆角裁剪是 xcgui 边界,只能矩形或自己裁 | 中 / 中 |
| IT3 | 整行链接(render)的 hover / 焦点语义 | Link 示例 | 行是自绘容器,鼠标可点但链接焦点 / Enter 语义由调用方在子按钮上做 | 待确认:整行进 Tab 链并给 Enter / 空格激活,会改页面 Tab 顺序(同 BC3) | 中 / 低 |
| IT4 | Header / Footer 自动 space-between | 官方布局 | 没有 justify-between | 暂缓:自动给首个可见子项加 Weight(1) 会改变现有调用方的布局;现用 `xui.ItemWeight` 撑开已能表达,有真实需求再做选项 | 小-中 / 低 |
| IT5 | Actions 内部 gap-2 | 官方 `ItemActions` 的 gap-2 | 动作之间用 Item 行间距(default 16) | 暂缓:嵌套 Auto 宽量不到原生按钮(第八节 104),单独做动作行会引入另一套布局 | 中 / 低 |
| IT6 | 描述多行 | 文档示例长说明 | Label 默认单行,需调用方 `SetWrap` 并给高 | 已记 DESIGN 第七节:由调用方 `SetAutoWrap()` 处理 | 不做 |

**明确不做**:`render` / 任意 React children、DOM `data-*`、RTL;Dropdown 示例由 `ui.Menu` 组合,不新增专用入口。

#### Native Select(#50)

**官方页**:Basic(status)、Groups(`NativeSelectOptGroup label / disabled`)、Disabled(整体禁用,外壳 `opacity-50`)、Invalid(`aria-invalid` 红边红环);tsx 外壳 `h-8 w-fit rounded-lg border border-input pl-2.5 pr-8`,`size="sm"` 改 28 / 更小圆角,右侧 16px ChevronDown;选项 `value` 与显示文本分离,可单独 `disabled`。

**已具备**:`NewNativeSelect(..., values []string, WithNativeSelectPlaceholder/Size/Value/Invalid)` + `Value()` / `SetValue()` / `OnValueChange()`;载体是原生 ComboBox 弹窗(触发区自绘 + 系统下拉),默认 h-32 / sm h-28、红边红环校验态、占位文本;未选中 `Value()` 返回空串,`SetValue` 匹配不到会取消选中;原生弹层工作区夹取与滚动条复用 `scrollbar.go`。

**未实现但可实现**:

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| NS1 | 整体禁用 | `disabled` prop + `has-[select:disabled]:opacity-50` | 绘制与 Field 级联已就绪(`combobox.go` 的 `SetFieldState` 会禁焦点 / 禁 Tab,`onPaint` 有 `st.Disabled` 置灰分支),只缺 NativeSelect 的语义化公开开关 | 可直接排期:`WithNativeSelectDisabled()` / `SetDisabled(bool)` 转调内嵌 `SetFieldState`/`Enable`,不必重画 | 小 / 中 |
| NS2 | 分组(OptGroup) | Groups 示例 | `values []string` 平铺;原生 ComboBox / ListBox 弹层对分组标题支持有限,组标题更接近自绘弹层(Select)的能力 | 待确认:换 `[]NativeSelectItem{Value, Text, Group, Disabled}` 并保住 `[]string` 调用方;先拍板是否接受为此偏离"原生弹层"定位(富条目本就分工给 Select,见 `native_select.go` 文件头) | 中-高 / 低 |
| NS3 | `value` 与显示文本分离 / 单选项禁用 | `NativeSelectOption` 的 `value` / `disabled` | 用显示文本直接当 value | 待确认:与 NS2 同批换数据结构;否则只能显示值即数据值 | 中 / 中 |

**明确不做**:HTML `form` / `name` / `required` 提交语义(Go 侧由 `Field` 与调用方校验接管);`size` 档超过 default / sm;DOM `data-*` / RTL。

#### Input OTP(#51)

**官方页**:Basic(6 格)、Pattern(`REGEXP_ONLY_DIGITS` / `REGEXP_ONLY_DIGITS_AND_CHARS`)、Separator、Disabled、Controlled(`value` / `onChange`)、Invalid(`aria-invalid` 红环)、Four Digits、Alphanumeric、Form;Composition = `InputOTP > InputOTPGroup(Slot) / InputOTPSeparator / InputOTPGroup(Slot)`;格 32×32、激活格 `ring-3`、分隔符 Minus 16、`has-disabled:opacity-50`。

**已具备**:`NewInputOTP(..., WithInputOTPLength/SeparatorAfter/Value/Invalid/Disabled/Pattern)` + `Value()` / `SetValue()` / `Clear()` / `Focus()` / `SetInvalid()` / `SetDisabled()` / `OnChange()` / `OnComplete()`;预置 `InputOTPPatternDigit/Alphanumeric/Any`;键入替换并前进、非法字符不入格;←/→/Home/End 移动、Backspace 清当前格 / 空格退前一格、Delete 清当前格;Ctrl+V 从当前格起分发粘贴;全满触发完成回调;32 格 + 16 分隔符按内容自撑;激活 / 校验 / 禁用态、原生真光标居中、共享边逐边绘制;实现 `FieldValueProvider` / `FieldControl` 可挂 Field 校验。

**未实现但可实现**(官方文档示例里的格数 / 过滤 / 分隔符 / 禁用 / 受控 / 校验 / 粘贴 / 完成回调均已覆盖;以下三项来自本地参照实现的 `readOnly` 示例与 API Reference 指向的上游包 props):

| # | 特性 | 上游依据 | 本库现状 | 实现路径 | 代价 / 优先级 |
|---:|---|---|---|---|---|
| OTP1 | readOnly 只读态 | 本地 `App.tsx` 的 `value="240815" readOnly` 示例;上游继承 `InputHTMLAttributes.readOnly` | 无,预填值后仍可键入 / 退格 / 粘贴 | 待确认:在 `onChar` / `onKeyDown` / `paste` 里拦截改写但保留焦点与原生选中复制,禁用语义与 `disabled` 区分(同 Input 的 readOnly 先例) | 小-中 / 低 |
| OTP2 | 文本对齐 `textAlign` | 上游 `textAlign?: left / center / right` | 固定居中 | 暂缓:加 `WithInputOTPTextAlign`,要与 `syncInset` 的空格 caret 居中逻辑协同 | 小 / 低 |
| OTP3 | `pasteTransformer` 粘贴转换 | 上游 `pasteTransformer?: (pasted) => string` | 只按 pattern 过滤非法字符 | 暂缓:核心粘贴与过滤已覆盖,去空格 / 格式化等定制有真实需求再加 | 小 / 低 |

**明确不做**:跨格拖动选择 / Ctrl+A 全选 / 复制剪切整段(每格是独立原生 Edit,见 DESIGN 第七节);假光标动画(这里用真光标);DOM `render` / `data-*` / RTL。

**本批结论**:只出核查结论,未改组件源码。可直接排期(无需确认):CR2 页码指示、IT1 xs 尺寸档、NS1 整体禁用。待拍板项见 Q7;其余暂缓项按真实需求再立项。

## 四、实现顺序建议(Input / Select 批次,已执行完)

1. **小改动、一致性收益**:S3 `OnOpenChange` → I2 `readOnly` → I7 `SetDisabled` → S4 `readOnly` → S8 `Value` 格式化 —— **已全部落地**(见第五节)。
2. **中改动、功能缺口**:I3/S9 必填校验、I4 上限 **已落地**;I1 文件输入已在第八批落地;S1 定位开关、S2 typeahead 已在第九批落地。
3. **需先确认需求或成本较高**:S5 多选、S7 滚动箭头、S6 悬停开关、I6 原生 Edit 视觉 API、I8 demo 对照 —— S6 / I6 已在第八批落地;S5 / S7 按 Q2 暂缓;I8 demo 随组件改动补,取舍见第五节末尾。

落地时按仓库固定流程:先查 DESIGN 第四节末语义表与本地 `*.tsx`,改完补 DESIGN 语义/差异表、CHANGELOG,跑 `go build ./...` / `go vet ./...` / `go test ./...`、逐组件单独安装 + 编译,再上 demo 做亮暗与 150% DPI 视觉验收。

## 五、实现进度(已落地条目)

按「高优先、小改动」选的跨组件一致性条目,分批落地(验收记录见 CHANGELOG,语义写进 DESIGN 第四节末与差异表)。当前进度:第一~十一批已落地。

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
| BA1 | Badge | 可点击徽章进 Tab 链 + Enter/空格触发; aria-invalid 只有 1px 红描边(官方 ring 类只有颜色、无环宽), focus-visible 的 3px 环走"环带+内芯"; `WithBadgeInvalid` / `SetInvalid` |
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

**第九批**(待确认项拍板后的第一批, 跨 4 个组件; 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| Q1 / D4 / AD3 / SH5 | Dialog / Sheet / AlertDialog | `Header()` / `Footer()` 返回布局容器(对应 DialogHeader / SheetHeader / DialogFooter / SheetFooter): 标题与描述成为头部槽的前两个子项, 追加控件排在它们后面; Dialog / AlertDialog 页脚横向(自定义在左、按钮靠右), Sheet 纵向(自定义在按钮上方), AlertDialog(sm) 在按钮行上方; 两个槽按布局占位量高, 标题/描述的上下外扩用负外边距抵消; 焦点顺序 = 头部槽 → 内容 → 页脚槽 → 页脚按钮 → 关闭钮; `Title()` / `Description()` 返回 Label 只改样式。`Content()` 维持返回句柄(已能放任意控件), 不另加 ContentStack |
| Q2 / S1 | Select | `WithSelectAlignItem(false)` 关闭 item-aligned, 改常规锚定; `WithSelectSide` / `WithSelectAlign` / `WithSelectSideOffset` / `WithSelectAlignOffset` 对应 Base UI Positioner 的 side/align/sideOffset/alignOffset(默认 bottom/center/4/0), 翻转与夹工作区复用 `popupPositionOffset` |
| Q2 / S2 | Select | 可打印字符 typeahead: A-Z / 0-9 / 小键盘 / 空格续接, 750ms 会话窗口, 标签前缀匹配(大小写不敏感、跳过禁用), 从当前高亮后一位环形查找; 同字符连按在匹配项间循环(含 llama 这类同首双字母时退回前缀增长); 打开/关闭清空会话 |

**第十批**(#31~#37 首轮核查的可直接排期项, 跨 7 个组件; 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| TS1 | Toast | `ToastOptions.ID` 同 ID 原地更新(Show upsert, 保留堆叠位置与 presence key);`Toaster.Update(id, opts)` / `Close(id)` / `Get(id)` 按 ID 寻址;退场中的旧条不复用,销毁时摘 ID 表 |
| TG1~TG3 | Toggle | base-nova 尺寸(32/28/36 高、px 10、gap 4、字号 14/13/14、图标画布 18/16/18)与配色(悬停/按下同为 `Muted`, outline `border-input`);`WithToggleDisabled` / `SetDisabled` 三层禁用;`SetSize` / `SetVariant` 运行时换档并重量宽 |
| TGR1~TGR3 | Toggle Group | `WithToggleGroupVertical` / `SetVertical`(组内等宽、首尾外角、connected 描边合并);`WithToggleGroupDisabled` / `SetDisabled`(程序化值仍生效);`OnValueChangeVeto`(切换后完整集合, 否决回滚按下态) |
| PG1 | Pagination | 上一页/下一页内距改为"图标侧 6、文字侧 10", 图标-文字间距 6;隐藏文字保持 32 方块 |
| CMD1~CMD3 | Command | 选中态改 `Muted`/`Foreground`(图标与快捷键同步);根圆角 `RadiusXl`、列表上限 288;`WithCommandFilter` / `SetFilter` 自定义匹配 |
| CAL1 / CAL2 | Calendar | `WithCalendarCaptionLabel()` 纯文字标题(无下拉热区);`WithCalendarHideOutsideDays()` 越界格留空且不参与命中 |
| DP1~DP3 | Date Picker | `WithDatePickerOnOpenChange` / `OnOpenChange`;输入过程中解析成功即同步选中值;可输入形态弹层改右对齐 / sideOffset 10 / alignOffset -8(`popoverPanel.ShowAligned`) |

**第十一批**(#38~#44 首轮核查的可直接排期项, 跨 7 个组件; 已全部落地):

| # | 组件 | 落地内容 |
|---|---|---|
| HC1 | Hover Card | 悬停 600ms / 离开 300ms、卡片 256 / 内边距 10 / sideOffset 4 / alignOffset 4(`WithHoverCardAlignOffset`); 触发区键盘聚焦立即打开、失焦按 300ms 关 |
| CL1 | Collapsible | `OnOpenChangeVeto(func(open bool) bool)`: 点击 / 空格回车 / Toggle 可否决, 程序化 `SetOpen` 不经过; 否决时不改状态、不播过渡 |
| BC1 / BC2 | Breadcrumb | `NewBreadcrumbSeparatorText` 文字分隔符(按字号量宽); `WithBreadcrumbEllipsisOnClick` 让省略号可点击(悬停 Accent 底、键盘焦点环、Enter / 空格触发) |
| NM1 | Navigation Menu | base-nova 几何与配色: 列表 gap 0 / 视口 p-1 + sideOffset 8、悬停/按下/展开取 `Muted`(展开 50%); 链接 `SetIcon`(16px 前置)与 `SetActive`(`bg-muted/50`); 视口圆角 `RadiusLg` |
| MB1 | Menubar | 整条 `WithMenubarDisabled` / `SetDisabled` / `Disabled`(触发器置灰、鼠标与键盘不响应, 已开弹层关闭); `OnOpenChange` 只在"任一打开 / 全部关闭"变化时发 |
| MB4 | Menubar | `RadioGroupValue` / `SetRadioGroupValue` / `OnRadioGroupChange`(条目文本即组值, 与 Menu 同一口径; 组内互斥沿用现有模型) |
| SB1 | Sidebar | `NewSidebarMenuSkeleton` / `SidebarGroup.AddSkeleton`(`WithSidebarSkeletonIcon` / `WithSidebarSkeletonWidth`): 32 高行 + 16 图标块 + 16 高文字条(宽度 10~100%), 2s 不透明脉冲, 图标栏只留图标块 |

**下一批建议**: 第十一批的可直接排期清单已清空, 待确认项见 Q6。剩余均为**已明确暂缓/不做**的项, 有真实需求再单独立项:

| # | 决定 | 说明 |
|---|---|---|
| Q1 余项(ACC4 / TT3 / TBL3) | 暂缓 | 内容槽统一形态已在 Dialog / Sheet / AlertDialog 落地; 手风琴触发区、Tooltip 内容、Table 单元格按需再做 |
| Q2 余项(S5 / S7) | 暂缓 | Select 多选(触发区摘要 + chips + 与只读/否决组合, 成本最高)与悬停自动滚(内容区上下命中带 + 逐帧滚动)等真实需求 |
| Q3(SL4 / SL5 / SA2 / D6 / PO6 / PO4 / DT3) | 不做 | 推挤碰撞、单滑块禁用、横向 ScrollArea、非模态对话框、Popover 模态与定位策略、表格列宽拖拽: 桌面收益存疑, 有场景再立项 |
| Q4(C4 / I8 / TA4 / DM5 / ACC5 / SEP3 / SK2) | 部分排期 | C4 只做"直角图片槽 + 自适应高"(不做圆角), 排在本批之后; 五组纯 demo 示例不单独立项, 随对应组件下次改动补 |

下一轮**核查**(不是实现)从 #52 Spinner 起。

**待确认需求(本轮已拍板)**:

| # | 涉及条目 | 拍板结果 | 落地状态 |
|---|---|---|---|
| Q1 | D4 / AD3 / SH5 / ACC4 / TT3 / TBL3 | 统一成 `Header()/Footer()` 返回 `LayoutEle`(Content 已有); 第一批只做 Dialog / Sheet / AlertDialog | 已落地(第九批); 余项暂缓 |
| Q2 | S5 / S7 / S1 / S2 | 先做 S1 定位开关 + S2 typeahead; S5 / S7 等真实需求 | 已落地(第九批); S5 / S7 暂缓 |
| Q3 | SL4 / SL5 / SA2 / D6 / PO6 / PO4 / DT3 | 全部不做, 有真实场景再单独立项 | 不做 |
| Q4 | C4 / I8 / TA4 / DM5 / ACC5 / SEP3 / SK2 | demo 随组件改动补; C4 只做直角图片槽 + 自适应高 | C4 待排期; demo 随批次 |
| Q5-1 | TS2 / TS3 / TS4 | 先做折叠堆叠(peek 12 / 每层 scale 0.9 / 悬停整组展开, 容量沿用 `WithToastMaxVisible`);swipe 与 F6 等触摸/无障碍需求再立项 | 已落地(折叠堆叠); TS3 / TS4 暂缓 |
| Q5-2 | CMD4 / CMD5 / CMD6 | 搜索行 InputGroup 化(CMD4)需带 DatatableFacet 整页回归, 建议单独一批;尾部勾选(CMD5)随选择类需求;字距(CMD6)不做 | 已落地(CMD4, 含 DatatableFacet 高度同步); CMD5 暂缓; CMD6 不做 |
| Q5-3 | CAL3 / CAL4 / CAL5 / CAL6 | 先改默认几何(28 / p-2 / bg-background, CAL4)并给 `WithCalendarCellSize`(CAL5), DatePicker 弹层同步;键盘网格(CAL3)随后;纵向折叠(CAL6)不做 | 已落地(CAL3 / CAL4 / CAL5); CAL6 不做 |
| Q5-4 | DP4 | 保持中文窄口径解析(今天/N 天后/常见格式), 不引入 chrono-node 等价物 | 不做 |

**待确认需求(第十一批, Q6)**:

| # | 涉及条目 | 建议 | 状态 |
|---|---|---|---|
| Q6-1 | MB2 / MB3 | Menubar 完整对齐 base-nova: 根 h-8 p-[3px] gap-0.5、条目 28、弹层 min-w-36、子菜单 min-w-24, 勾选/单选指示器移到右侧; 需先给 Button 补高度/内距覆盖或改自绘触发区, 再整页回归 | 待拍板 |
| Q6-2 | CL2 / SB2 / SB3 | 布局自由度: SB2(菜单按钮内容槽, 团队切换器/用户卡)优先; CL2(触发区/内容任意挂载)要改根管线, 成本最高; SB3(分组折叠)看侧栏需求 | 待拍板 |
| Q6-3 | BC3 / NM2 | 键盘与子导航: BC3(链接进入 Tab 链)会改整页 Tab 顺序; NM2(嵌套子菜单)成本高, 建议不排 | 待拍板 |
| Q6-4 | RS1 / RS2 / RS4 | Resizable 扩展: RS1(collapsible)优先, RS4(单栏锁定)顺手; RS2(preserve-pixel-size)低优先 | 待拍板 |
| Q6-5 | HC2 / NM3 / NM4 / NM5 / CL3 / RS3 | 暂缓: 弹层 zoom/slide 动画深度、纵向导航、视口尺寸过渡、两行说明、展开态根底色、F6 跳分隔条 | 暂缓 |
| Q6-6 | CL4 / BC4 / SB4 / RS5 / Menubar Alt | 不做或组合式: keepMounted 卸载语义、富内容面包屑、移动端 Sheet / Provider 上下文、布局持久化、Menubar Alt/typeahead | 不做 |

**待确认需求(第十二批, Q7)**:

| # | 涉及条目 | 建议 | 状态 |
|---|---|---|---|
| Q7-1 | CR1 / CR3 | Carousel 多页同屏(`basis-1/3`)与页对齐:会改页宽 / 步进 / 循环位移公式,不做默认变更;常见单页铺满已对齐 | 待拍板 |
| Q7-2 | NS2 / NS3 | NativeSelect 分组标题 / 选项禁用 / `value` 与显示文本分离:需要换数据结构(`[]NativeSelectItem`),同时保住现有 `[]string` 调用方 | 待拍板 |
| Q7-3 | IT2 / IT3 | Item 的 image / avatar 内建媒体态(圆角裁剪受限)与整行链接焦点 / Enter(会改 Tab 顺序) | 待拍板 |
| Q7-4 | EM2 | Empty Header 最大宽 384:需按父宽钳制,窄容器口径先确认 | 待拍板 |
| Q7-5 | OTP1 | InputOTP readOnly 只读态(本地参照实现有示例):拦改写但保留焦点与选中复制,与 disabled 语义分开 | 待拍板 |
| Q7-6 | CR4 / KB1~KB3 / EM1 / IT4 / IT5 / OTP2 / OTP3 | 按钮几何 / Tooltip 键帽反色 / 键帽图标与几何覆盖 / Empty 边框背景 / 头尾行 space-between / 动作行 gap / InputOTP 文本对齐与粘贴转换:都是小改动,等真实场景随组件下次改动补 | 暂缓 |

**可直接排期(无需确认)**: 第十二批的 CR2(Carousel 页码指示)、IT1(Item xs 尺寸档)、NS1(NativeSelect 整体禁用)。第八~十一批的清单已全部落地,其中 I1 文件输入 / I6 原生 Edit 覆盖 / DM2 菜单定位参数为本批最有价值的几项。AL2(Alert 描述多行)经核对由 `reflow` + `SetWrap` 覆盖(量高按 `WrappedHeight` 写回, 不与折行冲突), 本轮无需改代码。
