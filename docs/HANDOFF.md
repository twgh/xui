# HANDOFF — 交接文档

> **本文只回答"现在到哪了"**:当前状态、待办、命令、仓库结构、验证工作流、去哪查。
> 设计原理 / xcgui 踩坑细节 / 交互语义**一律不在这里重复** —— 见 **DESIGN.md**(第八节 = 踩坑全集按编号;第四节末 = 语义权威汇总表)。agent 会话硬规则见 **AGENTS.md**,面向用户的用法见 **README.md**。
> ⚠️ **DESIGN.md 会随 `xui init` 原样拷进用户项目的 `ui/`**, 按使用者视角写:不写日期、报障时间线、待办、发布流程、仓库结构 —— 那些只放在本文。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn`)的源码为唯一权威。

## 当前状态(截至 2026-10-25)

**阶段**:P0 运行时 + P1/P2/P3 组件全部完成并视觉验收(**P3 已收尾**, 2026-10-17 补上 combobox 弹层进场动画;CLI 远程 registry 已完成);2026-10-23 新增 Collapsible 可折叠面板;2026-10-24 新增 Empty / Field / Item / Breadcrumb / Kbd / Skeleton / Spinner / Toggle / Toggle Group;**批次 2 收尾: 2026-10-24 补齐 Native Select / Select / Input-OTP / Accordion 四项**(上游语义权威 tsx + App.tsx 对照卡已同步); 2026-10-25 补齐批次 3: Popover / Hover Card / Avatar / Command(上游 tsx 已同步), demo 页签再拆为 **14 页**(基础 4 / 表单 4 / 导航 2 / 反馈 2);**v0.1.0 已发布(2026-10-19, GitHub 公开 + tag 已推,依赖 xcgui v1.4.3)**;P4(文档站、主题画廊)未开始。组件覆盖与已知限制见 DESIGN 第七节。

**已完成(括号里是根因/踩坑所在的 DESIGN 第八节条目号)**:

- ✅ **demo 页签再拆分**(2026-10-25, 基础/表单/导航/反馈单页仍然过长):
  - **基础1** = 按钮 / 按钮组 / 切换按钮 / 切换按钮组; **基础2** = 徽章·分隔线·标签 / 头像 / 快捷键 / 加载指示器 / 骨架屏; **基础3** = 弹出层 / 卡片 / 比例容器; **基础4** = 滚动容器 / 列表项 / 手风琴。
  - **表单1** = 输入框 / 验证码输入 / 滑动条; **表单2** = 选择控件 / 下拉框 / 独立 Select; **表单3** = Field / FieldSet; **表单4** = 日期与时间 / 月历 / 范围与预设。
  - **导航1** = 命令面板 / 下拉菜单 / 右键菜单; **导航2** = 标签页 / 分页 / 面包屑 / 可折叠面板。
  - **反馈1** = 对话框 / 提示气泡 / 警示对话框 / 对话框变体; **反馈2** = 提示条 / 破坏性·动作 / 通知 / 进度 / 空状态。
  - 页签 **14 个**; `XUI_HOME_PAGE` 映射改为 **0..13 = 基础1/基础2/基础3/基础4/表单1/表单2/表单3/表单4/导航1/导航2/反馈1/反馈2/数据/图标**; 页面序号、`pageTitles`、构建函数同步。
  - **Tabs 换行**: 页签行宽度改为跟随窗口(`xui.ItemFill`), `Tabs` 新增 `WithTabsWrap`: 宽度装不下单页最小 72 时按"行数最少、每行均衡"换行, 同步自身高度并逐级触发祖先重排; line 变体的下划线改为贴**当前行**底边(不再固定列表底边)。选换行而不是横向滚动, 是为了页面入口不被藏起来。960 宽下 14 页排成 7+7 两行, 1200 宽自动回到一行。
  - 交互密集卡仍保持在页首(基础3 弹出层、导航1 命令面板), 规避 DESIGN 118 的长页命中偏移; 基础1 用 5:4 两列让按钮与切换按钮并排, 按钮组放回左列与按钮卡同宽; 导航2 标签页与面包屑并排。
  - 验收: `go build/vet/test ./...` 全绿; 150% DPI 亮/暗主题逐页截图, 960×660 下页签换两行、点第二行页签可切换、窗口拉回 1200 后恢复一行且内容跟随上移; 基础3 的 Popover 弹层正常打开。

- ✅ **批次 3: Popover / Hover Card / Avatar / Command**(2026-10-25, 上游 tsx 同步):
  - **前置完成**: 本地 shadcn 参考实现补齐 `popover.tsx` / `hover-card.tsx` / `command.tsx` / `avatar.tsx`(取自 shadcn.com.cn registry), App.tsx 加"基础/表单/导航"对照卡, `pnpm build` 通过。
  - **Popover(★★)**: 公开 `NewPopover(trigger, WithPopoverContent(...))`; 弹层底座在 popup.go 上扩展出"布局内容 + 离屏量高"(第八节 115), 默认 w-72(288)/p-4; 点击触发(Button 走 BnClick、普通元素 LButtonUp)、失焦/ESC 关闭、ESC 回焦点; side/align 支持下方优先 + 自动翻转 + 工作区夹取; `PopoverHeader/Title/Description` 助手; 固定高/自适应高、`Refresh`。
  - **HoverCard(★★)**: 非激活弹层(`WS_EX_NOACTIVATE` + SW_SHOWNOACTIVATE, 不调 SetForegroundWindow、不依赖 KillFocus, 第八节 116); 悬停 700ms 打开/移出 300ms 关闭, 指针在卡片/触发区上时关闭检查续期; 内容可交互、不抢输入焦点; 支持 side/align/delay/width/padding。
  - **Avatar(★★)**: 圆形 sm 24/default 32/lg 40; 本地图片或 `image.Image` → CPU 居中裁剪 + 4× 抗锯齿圆形蒙版 → `XImage_LoadFromDataRGBA` + `XDraw_ImageEx`(第八节 117), 无图回落首字母; 右下角徽标(圆环内缩, 可带图标); AvatarGroup 用横向布局负间距 -8 叠放 + 组内 2px 背景色环 + `AddCount`。
  - **Command(★★★)**: 搜索行(自绘搜索图标 + 透明原生 Edit + 设备像素对齐下边框) + ScrollArea; 分组标题/条目/快捷键/禁用/空状态; 输入即过滤(不区分大小写 + keywords), ↑↓ 循环跳过禁用、Home/End/PageUp/PageDown、Enter 选中、ESC 清空; 鼠标悬停高亮、点击选中并回焦搜索框; 选中不自动关闭, 用 Dialog + Command 组合成命令面板弹窗。
  - **验收**: 探针逐项验证 —— Popover 自动高度/锚点居中/翻转; HoverCard 非激活显示、悬停进入卡片保持、移出 300ms + 轮询后续期关闭; Avatar 圆形裁图/徽标/叠放组/负间距; Command 键盘 ↓×2+Enter、输入过滤、空状态、ESC 清空、点击回焦; demo 基础2 页首加弹出层/头像卡、导航页首加命令面板卡(避开长页面滚到底后深层卡片的命中偏移, 见第八节 118)。逐组件安装 avatar / popover / hover-card / command 与 `add --all` 均编译通过; `go test ./...` 全绿。
  - **新结论**: 隐藏窗口里 LayoutEle 的 Auto 高恒为 0, 弹层要离屏显示一次再量(115); 非激活悬停弹层的关闭改用 MouseLeave + 指针位置轮询(116); 圆形头像用 RGBA 蒙版 + XDraw_ImageEx(117); 长页面滚到底后深层嵌套卡片的命中区可能比绘制位置低 20~30 逻辑像素(118, 根因未定位, 交互密集卡片放页首规避)。
  - **复验(阴影过宽, 二轮)**: 用户二次反馈"阴影还是在很外面"且"文字贴到按钮" —— ① 首轮把阴影带从 8/48 收到 5/50 只解决了一半; 探针枚举窗口发现弹层窗口 671×356 物理, 卡片只有 288 逻辑宽 —— `popoverPanel.resize` 把窗口尺寸乘了 DPI 再传给 `SetWindowPos`, 而该 API 的 **cx/cy 是逻辑像素**(x/y 才是物理像素), 尺寸被二次缩放, 多出来的区域就是"阴影"; 改为尺寸传逻辑值后窗口 447×237, 与卡片+5px 阴影带严格贴合(见 DESIGN 第八节 119)。② 内容区子项默认间距 0 导致末行文字贴按钮, Popover/HoverCard 内容宿主改为 gap 8; demo 的说明拆两行。Select / DatePicker / TimePicker 不调用 resize, 不受尺寸 bug 影响, 但共用收紧后的阴影带。

- ✅ **批次 2 收尾: Native Select / Select / Input-OTP / Accordion**(2026-10-24, 上游 tsx + App.tsx 对照卡同步):
  - **前置完成**: 本地 shadcn 参考实现补齐 `native-select.tsx` / `input-otp.tsx` / `accordion.tsx`(`npx shadcn add`), select.tsx 原有; App.tsx 基础页加手风琴对照卡、表单页加「独立 Select / NativeSelect」「验证码输入」对照卡, `pnpm build` 通过。
  - **NativeSelect(★)**: ComboBox 的 value 语义封装 —— 触发区/原生弹层与 NewComboBox 完全一致, 补 `<option value>` 字符串 API(`Value/SetValue/OnValueChange`, SetValue 匹配不上取消选中)。
  - **Select(★★★)**: radix Select(item-aligned)的自绘弹层独立版 —— 触发区与 ComboBox 同一套 paint 规则; 弹层 = popoverPanel + ScrollArea + 自绘条目按钮(悬停/焦点 = bg-accent、选中右侧 check、禁用置灰、分组标签/分隔线); item-aligned = 选中项正对触发区(中心对齐推导同 alignComboPopup), 内容超高时算好 `pendingScroll` 落位后 `ScrollPosYV`; 键盘 ↑↓(跳过禁用)/Home/End/Enter 走 `popoverPanel` 新增的 `OnKeyDown/OnEsc` 钩子(ESC 把焦点还给触发区), 打开即把焦点放到选中项。实现期修掉两个 bug: ① `syncValue`/`Selected` 查懒构建的 flat(没弹过层是空表)导致"SetValue 后触发区还显示占位符", 改查源数据 groups; ② build 的循环闭包没拷贝循环变量(Go 1.18 旧语义, HANDOFF 硬规则), 所有条目画成最后一条。
  - **InputOTP(★★★)**: 每格一个原生 Edit(32×32) + 自绘壳; 字符在 `AddEvent_Char` 拦截后自己落格(打进已填格 = 替换并前进、非法字符不入格, 原生没有 maxlength 放行会越填越长); ←→/Home/End 移动、Backspace 空格退格清前一格、Ctrl+V 读剪贴板分发(过滤非法字符, 焦点停最后写入格); 全满触发 OnComplete。文字居中走原生 `SetTextAlign(Center|Center_V)`(单行 Edit 程序 SetText 同样生效; 实现期一度误判"对齐不生效"改成按字宽设左右 border, 结果把数字右半截裁掉, 见 DESIGN 第八节 110); aria-invalid 的组外圈画在容器 PAINT_END(普通 paint 阶段画的环会被格子盖掉); `WithInputOTPValue` 在选项阶段格子还不存在, 先记 initial 格子建好后落位。剪贴板读 = wapi 的 Clipboard 系列 + GlobalLock(vet 坑: uintptr 别直转 unsafe.Pointer, 用同布局自定义 slice 头读 UTF-16)。
  - **Accordion(★★)**: 每项 = Toggle 式全自绘触发钮(check 类型, 键盘空格/回车自补, 高度固定 40) + 复用 Collapsible 的内容高度过渡; 触发区 hover:underline **手绘 1px 线**(字体侧 `FontStyle_Underline` 不渲染), 箭头随开合在 ChevronUp/Down 间互换(上游就是互换无旋转); 条目间 1px 分隔线(末项没有); single 单开互斥可全收 / multiple 独立, 互斥收起只通知一次; 根 `Value/SetValue/OnChange`。
  - **验收**: 探针(独立 main 包, AppWindow 样板)亮/暗双主题逐项截图 —— Select 弹层(分组/图标/禁用/勾/高亮/item-aligned)、OTP(逐格前进跨分隔符、居中、退格、预填/禁用/invalid 红环)、Accordion(单开互斥、箭头切换、分隔线)、NativeSelect(原生弹层+value 流); 键盘导航(↓×2 跳禁用项 + Enter 选中回写)与 ESC 关闭实测通过。**两个排查教训**: ① "弹层开着但看不见"这轮是测试环境问题 —— 用 PowerShell 把探针窗口设成 `HWND_TOPMOST` 后, 非 topmost 的弹层被压在探针后面(点击全被探针截走 → KillFocus 秒关), 换 `HWND_TOP` + SetForegroundWindow 即正常, 组件代码没有问题; ② 合成滚轮/滚动条拖动不可用是已知限制(DESIGN 66 末), 弹层类验收改走"程序化开合 + 点击/键盘注入"的探针路线。逐组件安装 native-select / select / input-otp / accordion 与 `add --all` 均编译通过; `go build/vet/test ./...` 全绿。registry 新增 4 项(accordion 依赖 button+collapsible+separator, 首跑漏了后两个已补), demo 表单1/基础2 页各加展示卡。
  - **复验(同日, 用户报障后)**: ① 手风琴展开抖动 —— 根因是"每条 Collapsible 各挂一条动画定时器 + 每帧 apply 各自立即整窗重绘"(一帧两张中间态), 以及"内容区 pb-2.5 用外边距导致动画目标 34 / 终态 Auto 24, 收尾跳 10px"; 改为过渡归属宿主窗口 + 占位元素表达内边距(DESIGN 111)。② 触发区 hover 下划线看不到 —— `FontSpec.Underline`(FontStyle_Underline)在 D2D 文本路径下不渲染, 改手绘 1px 线; 运行时 `FontSpec` 的 `Underline` 字段已删。③ 验证码文字显示不全 —— DESIGN 110 结论错误, 对齐本来生效, border 技巧才是裁字根源, 已回退。④ 顺带修: 手风琴 demo/文档示例的内容标签漏了 `xui.ItemFill`(标签 0 宽, 展开后一片空白)、触发区高度按预设 33 而非 40。
  - **二轮复验(用户截图报障)**: ① 手风琴焦点环整条触发区被填成灰色圆角块 —— 环带只画了"整块铺环色", 漏了"内芯铺回宿主底色", 补上(照 Button 的 focus-visible 画法, DESIGN 112); ② 内容区顶部那个"看起来像多出来的分隔线"的细矩形 —— 是 pb-2.5 占位元素的原生 1px 边框(`EnableBkTransparent` 只清背景), 补 `EnableDrawBorder(false)`(DESIGN 111 末); ③ 验证码: 第二组首格没有左边框/圆角、格子顶底横线在竖线处断 1px、聚焦时最左格环带方角 —— 圆角与外侧边改按"组"算、擦边只擦中段、环带用同一套 corners(DESIGN 113); ④ 禁用态验证码点进去还有光标 —— `XEle_Enable` 不禁鼠标聚焦, 补 `EnableFocus(false)` + caret 染底色兜底 + `Focus()` 在禁用时直接返回(DESIGN 114), 分隔符的减号也跟着 50% 透明; ⑤ Select 组标签贴左边框 —— 标签文字补 6px 左内距, 与条目内容对齐。
  - **三轮复验(手风琴内容间距)**: 描述离触发区太远、离分隔线太近 —— pb-2.5 的占位在 `ContentStack` 里就挂了, 排在第一个子项**之前**; 改成开合时挂到末尾(幂等, 用户再加内容会把占位挪回末尾), 占位高度扣掉内容区行间距。实测内容区 = 标签 24 + 间距 6 + 占位 4 = 34, 文字上下各 12 逻辑像素, 与 shadcn 的 `pt-0 + pb-2.5` 观感一致(DESIGN 111 末)。
  - **四轮复验(焦点环压字 + 分隔线一粗一细)**: ① 焦点环盖到文字/箭头上 —— 上游 `ring-3` 画在框外, 我们的环画在元素内必然吃 3px 内容区; 触发区文字加 `accordionPadX = 4` 让位、环带不再描内缘; 箭头绘制块贴右缘(chevron 墨迹自带内缩, 视觉间隙与文字一致, 逐像素探针确认左右都 ≈2 设备像素)(DESIGN 112 末)。② 两条分隔线一粗一细(实测 2 物理行 vs 1 行)—— 根因是 `widget.NewElement` 的原生 1px 边框在 1px 高的元素上就是那条线, 框架对它按 y 相位取整; 关掉边框后由自绘画"对齐整设备像素"的 1 物理行(`FillRectColorF` + 中心设备行), 全 DPI 恒定。顺带修: `Painter.Scale` 在 `BeginPaint` 里一直是 0, 现在填 `DPI/96`(DESIGN 10)。

- ✅ **Toggle / Toggle Group**(2026-10-24): 补齐 shadcn 批次 2 前两项 —— `ui.NewToggle`(default/outline 变体、h-9/h-8/h-10 三档、行内图标、aria-invalid、禁用、焦点环; 鼠标/空格/回车切换 + `OnChange`)、`ui.NewToggleGroup`(单选默认互斥且可全部取消 / `WithToggleGroupMultiple` 多选, `AddItem(value,text,...)` + `Value/SetValue/OnChange/SetSpacing`, spacing=0 首尾外角、组级 variant/size 覆盖条目); demo 基础1页第四行; 亮/暗 150% DPI 验收, 点击/空格/Tab 焦点环/单选互斥与全部取消/多选/组回调/首尾圆角/间距逐项确认; 新结论: 原生 Check 按钮不响应空格/回车(元素 KeyDown 能收到), 组件自补键盘激活并与鼠标共用回调, 见第八节 108; 逐组件安装 toggle / toggle-group 与 `add --all` 均编译通过
- ✅ **Label 下伸裁剪修复**(2026-10-24): 卡片说明 "ToggleGroup:" 的 g 尾巴被切平; 探针对比 20/21/22/23/24/26 行高, 根因是原生 Button 默认四边 border=1, 20 高标签的文字区上下各少 1px, 14px 字体的行盒装不下; Label 构造时 `SetBorderSize(1,0,1,0)`, 只清上下内缩、不增加布局高度; 探针与实际 demo 说明行均确认 20 高墨迹与 24 高一致(见第八节 109)
- ✅ **Item / Breadcrumb / Kbd / Skeleton / Spinner**(2026-10-24): 补齐 shadcn 批次 1 剩余五项 —— `ui.NewItem`(Media/Content/Actions/Header/Footer + 3 变体 × 2 尺寸 + ItemGroup/Separator/Title/Description/Media)、`ui.NewBreadcrumb`(Item/Link/Page/Separator/Ellipsis, 链接 hover 过渡 + OnClick)、`ui.NewKbd` / `NewKbdGroup`、`ui.NewSkeleton`(2s 脉冲, 运行时新增 `xui.Pulse`)、`ui.NewSpinner`(Loader2 1s/圈, 运行时 `xui.Spin`); demo 基础1页第三行 + 基础2页列表项 + 导航页面包屑, 亮/暗 150% DPI 截图验收, Spinner 逐帧确认旋转、Skeleton 逐帧确认脉冲、Breadcrumb hover 过渡与 OnClick 回调确认; 三处新结论: `XEle_SetPadding` 可替代外边距 pad、Item 内容区要按实测宽度锁 Fixed、Label 精确按 TextWidth 设宽会被原生 Button 内边距截断(见第八节 103~105); 另修暗色 ItemMuted 上徽章露出深色方底(Badge 不再给不透明变体铺 Card 方底), 以及首页多动画掉帧/GPU 高(Spin/Pulse 共享宿主窗口定时器、按动画元素矩形分组局部重绘(XWnd_RedrawRect)、60fps + 前台暂停; 后台/最小化/滚出视口自动停表, app GPU ≈0.5%、DWM 后台 ≈0.03%、前台约 10~14%(30fps 实测仍 ~10%, 已回退 60fps 保证流畅), 见第八节 106); 逐组件安装 spinner / skeleton / kbd / breadcrumb / item 与 `add --all` 均编译通过
- ✅ **Empty / Field**(2026-10-24): 新增空状态容器 `ui.NewEmpty`(Header/Content/Body 槽位 + 图标底板/无底板媒体 + 标题/描述)与表单字段布局 `ui.NewField`(Label/Title/Description/Content/SetInvalid + FieldSet/Group/Legend/Separator/Error);demo 反馈页 + 表单1页亮暗验收,校验错误区显示/隐藏时卡片高度正确伸缩;横向 Content 的嵌套布局拿不到 Weight/Percent 宽,默认改 Fixed 240(见第八节 101);隐藏错误区后旧位置文字会留重影,重排结束补整窗重绘(见第八节 102);逐组件安装 empty / field 与 `add --all` 均编译通过
- ✅ **Menu 右键触发 / 前置图标 / inset 扩展**(2026-10-23): 新增 `NewContextMenu(元素, build)`(窗口级右键命中触发区, 子元素上右键也生效; `SetEnabled` / `OnOpenChange`)、`MenuIcon`(条目/Sub 前置图标)、`Label` / `Sub` 支持 `MenuInset`;Check/Radio 未选中也保留右侧 24(`pr-8`); 弹层宽度改为**逐条按自绘字体量宽**(原生字体与自绘不一致, 内容自适应的长中文会被裁, 见第八节 99);demo 导航页新增虚线右键触发区 + shadcn ContextMenuDemo 同构菜单。**菜单进场动画整个去掉**(用户录屏: 整窗 alpha 会透出背景文字, 改成"面板不透明 + 只淡内容"又会先闪一帧空白面板, 两条路都闪; 位移/缩放会崩)—— 现在与原生菜单一样弹出即全内容, 见第八节 99; 顺带修正触发区文字居中对齐(`XDraw_SetFont` 会重置对齐, 顺序必须是先字体后对齐, 见第八节 100), 并去掉 demo 里 `人员` 分组标签的 `MenuInset`(本菜单勾选在右侧, 条目文字从 6 起, 加了 inset 反而错位); 右键触发区补 hover / 展开反馈(`bg-accent` + 虚线变 `Ring`, `OnOpenChange` 驱动, 透明元素先铺 `t.Card` 擦底), 让右键不再像凭空出现, 与左侧按钮菜单的 hover→展开节奏一致。其余验收:亮/暗两主题右键触发、子菜单、勾选/单选状态持久、破坏性样式;独立探针验证自适应长文本不裁 + 短菜单 128 下限;`go test ./components -run Menu` 通过
- ✅ **P0 运行时**:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter、字体缓存(单位坑见 31)、focus-visible 焦点来源跟踪(16)
- ✅ **P1 组件**:Button(6 变体×8 尺寸档)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**:Switch(2 尺寸)、Progress、Slider、Tabs(default/line)、Tooltip、ComboBox、图标系统
- ✅ **Slider 渲染缺陷彻底修复**:悬浮/拖动冒方块、两端 ring 被裁、灰环残留(根因与结论见 25/26/27/28)
- ✅ **CLI**:`init / add / list`(registry.json 驱动,组件源码拷入用户项目);`update / --all` **未实现**
- ✅ **demo + 用户项目 E2E**:拷贝出的组件在独立用户模块编译/运行/渲染全部正常
- ✅ **demo 布局化 + 布局 API 收口**(2026-10-04~08):改用 `widget.NewLayoutEle` 流式布局;布局项统一到 `xui.Item*`(只声明要改变的那条轴, 哨兵 `xui.ItemKeep=-1`), 容器骨架统一到 `xui.HStack` / `xui.VStackWeight` / `ui.Card.ContentStack`;语义与"纵向溢出换列"坑见 29/41
- ✅ **参考实现补齐**(本地 shadcn 对照页):switch/progress/slider/tabs/tooltip/select/table/dropdown-menu 源码落盘 + `App.tsx` 对照卡(含新第 6 页「图标」);`pnpm build` 通过
  - 2026-10-14 再补:`scroll-area.tsx`(`npx shadcn add scroll-area`, 交互语义权威)+ 基础页"滚动容器"对照卡(与 demo 同构)。
- ✅ **ComboBox 弹层对齐 shadcn**:ring 四边完整(画在 `XE_PAINT_END` 内缩 1px)、item-aligned 定位 + 屏幕边界截断滚动(30)
- ✅ **Dialog**(2026-10-04):路线 = 窗口内覆盖元素(全窗遮罩 + 居中面板 + 页脚 + 幽灵图标关闭钮), ESC / 点遮罩 / 关闭钮关闭 + Tab 焦点陷阱;backdrop-blur-xs(33/34/47)
- ✅ **Menu**(2026-10-05):原生菜单承载行为 + 窗口级 `XWM_MENU_DRAW_*` 自绘皮肤(36)
- ✅ **Table**(2026-10-05):原生 List 载体 + 外观全自绘(表头/行底/边框/滚动条)(40/41)
- ✅ **Tree**(2026-10-06):原生 Tree 载体 + `XE_TREE_DRAWITEM` 自绘;"隐藏容器首显空白"回归已改为组件内自愈(42/51)
- ✅ **hover 过渡动画**(2026-10-06):150ms cubic-bezier(0.4,0,0.2,1), 运行时 `anim.go` 驱动, 组件侧只用 `xui.Hover / HoverItem + Fade`(45)
- ✅ **重绘节流调优**:应用初始化 `a.SetPaintFrequency(10)`, 实测 30.9 → 59.0 fps;`XC_SetTextRenderingHint` 在 D2D 下是空操作(46)
- ✅ **弹层/对话框进出场动画**(2026-10-06):`xui.Presence(...)` + `SetElementsAlpha / FadeWindowAlpha / SlideWindowAlpha`;窗口 alpha 永远不精确 = 0(47)。⚠️ **异型透明窗上不要用整窗 alpha 做动画**(只认第一次调用, 见 57), 淡入淡出改成元素 alpha
- ✅ **backdrop-blur-xs**(2026-10-07):抓窗快照 + Go 盒式模糊 + BkM 图片填充;改物理像素管线后全屏弹窗 160ms → 61ms(47)
- ✅ **Pagination + demo 图标页改造**(2026-10-08):恒定 7 槽位折叠, 搜索 / 翻页 / 悬浮 Tooltip / 点击复制图标名(53)
- ✅ **图标系统**:抗锯齿改**直通 D2D 几何填充**(`d2dicon.go`);**lucide 资源预处理**(2130 个图标, `icons/` 子包)—— 原理 / 坑 / 性能见 43/44,速览见下方「图标系统」
- ✅ **与 shadcn 逐条核对变体/尺寸并补齐**(2026-10-09):核对本地 18 个 tsx 的 cva variants/props 后补齐 —— Button 补 Link 变体 + 8 档尺寸(xs/sm/default/lg + 4 个方形图标档)、Badge 补 ghost/link、Card 补 sm 尺寸与说明/页脚、ComboBox 补 sm 触发区、Slider 补 min/max/step、Separator/Tabs 补纵向取向;像素实测按钮高度 24/28/32/36 与 shadcn h-6/7/8/9 完全吻合。映射与**仍然存在的差异**(aria-invalid / CardAction / Menu inset / chart·sidebar 令牌)记在 DESIGN 第四节末「变体/尺寸与上游的映射、已知差异」;1px 细条的绘制结论见 10
- ✅ **demo 全面布局化 + 分类重规划 + `Card` 改布局容器**(2026-10-09):窗口 800×590 → **1200×860**(最小 960×660), 页面/卡片去掉全部"算出来的绝对坐标", 改为流式布局(窗口最大化即整页自适应, 像素实测: 卡片左边距 36 = 阴影 12 + 页边距 24 ✓, 最大化时 24 ✓, 数据页表格跟随长到 ~1000 逻辑高)。页签重规划为 **基础 / 表单 / 导航 / 反馈 / 数据 / 图标**(原"控件/结构/新增变体"散布的内容各归其位, 不再挤在一页)。`Card` 底层换 `LayoutEle`、内容区随卡片尺寸联动(见 DESIGN 第八节 56);布局尺寸语义(尤其"`Fill` ≠ 剩余空间"、"`Weight` 需父级确定尺寸")见第八节 55
- ✅ **页面宿主的"叠放换页"不需要绝对定位**(2026-10-09):布局会跳过 `Show(false)` 的子项(排成 0x0), 所以 6 个页是宿主里的 6 个 Fill/Weight 兄弟, 切页只 `Show` + `AdjustLayout`(实测, 见 55)
- ✅ **Tooltip 气泡"显示着却看不见"根因修复**(2026-10-09):症状是悬停后提示不出现(用户侧描述"点几次切换主题后 ToolTip 就不显示了"), 实际窗口**被显示出来了**(可见、矩形正确、位置还随动画滑动), 但 layered alpha 被永久钉在 3/255 —— 异型透明窗的整窗透明度**只认第一次调用**(探针复现, 见 57)。修法: 进出场改淡**元素** alpha, 整窗 alpha 只在显示后设一次 1; 验收=亮/暗双主题各 3 轮、连续 6 次切主题后每轮气泡像素增量完全一致
- ✅ **Tooltip 跟随主题切换**(2026-10-09):气泡挂在屏幕上时点"切换主题", 它过去会保持旧配色 —— 自建的独立置顶小窗不在 `RegisterWindow` 重绘清单里, 得自己 `OnThemeChangeFor` 订阅并重算+重绘(见 58); 顺带把窗口扩展样式改成 `window.NewEx` 一次传入。验收=光标停在主题按钮上直接点它, 气泡当场从深色块变浅色块

- ✅ **表格"首屏自适应列宽塌成最小值"修复**(2026-10-10):数据页刚进来时最后一列只有 20px(表头裁成"日"、单元格全显示 `...`), 拉一下窗口才正常 —— 根因是"列宽按元素宽度算, 而 `Show()` 之前元素宽度是 0"; 现在组件自己订阅 `List.AddEvent_Size` 自愈(`lastW` 防震荡), 调用方不必再记得调 `FitColumns`(见 60)
- ✅ **Menu 弹层阴影 + p-1 边距 + 进场淡入**(2026-10-10):弹窗转 `Window_Transparent_Shadow`(官方 DrawMenu 同款, 圆角+投影+整窗 alpha 一次解决); `SetBorderSize(4)` 落实 shadcn p-1(高亮距弹层左右边缘 4 逻辑像素对称, 修掉"与参考差 1px"); 分隔栏改全宽(-mx-1); 进场 fade-in 100ms(Presence 挂宿主窗驱动)。三处深坑: 过渡闭包捕获 `Menu_PopupWnd_` **指针**会读到 0/垃圾(0=淡入冻结, 垃圾=崩进程), 必须拷贝句柄值 + `XC_IsHXCGUI` 守卫; 对菜单弹窗 `SetWindowPos` 位移直接崩进程 → slide-in 不可实现, 只做淡入; `XMenu_SetItemWidth` 实测有效(早期"无效"结论已修正, 见 64)
- ✅ **暗色禁用/未选态配色按 shadcn 逐像素对齐**(2026-10-10):复选框未选态(暗色 `bg-input/30`)、复选框/单选框/开关的禁用态、禁用输入框过去都错用 `WithAlpha` 覆盖半透明令牌(暗色下画出发亮灰块 / 亮白框线 / 全亮白滑块)。现在统一为"**先铺宿主底色(Card) + 每个图层各乘 50%**":形状全部跟随, 原生文字保持原色, 输入框禁用文字预混变淡(见 61); 顺带修掉透明元素重绘**不擦底**留下的旧像素(焦点环 / 上一个状态)。验收=让"切换后的渲染"与"直接建成目标状态"逐像素比, 亮/暗各一轮都是 **0 像素差**(色值与 shadcn 的整组 50% 合成理论值逐条吻合)
- ✅ **本地 shadcn 参照页(界面展示)与新 demo 同构**(2026-10-10):`shadcn/src/App.tsx` 重整为与 demo 一致的六页(**基础 / 表单 / 导航 / 反馈 / 数据 / 图标**), 内容逐条对应(6 变体 × 4 档尺寸 + 4 方形图标档、徽章 6 变体、CardFooter、多行与禁用输入、滑动条两档、开关 default/sm、Tabs default/line/纵向、恒 7 槽分页、表格 12 行 + 树); 新增 `?tab=form` 这类初始页签参数(对应 demo 的 `XUI_HOME_PAGE`), 便于分页截图对照; `pnpm build` 通过

- ✅ **SVG 解析层上移到上游 `xcgui/svgp`**(2026-10-11): path 语法解析、SVG 文档遍历(`<g>` 继承/viewBox)、
  子路径分组与绕向统一、d2d 几何组装全在那边(纯 Go, 14 个单测, 不经过 DLL —— 而 xcgui 自带的 svg 是
  DLL 内 C++ 且不支持svg的一些特性, 纯 Go 是唯一可继续改进的路径); xui 只留图标语义(24 网格归一化、
  `!` 填充前缀、SVG 原文缓存、GDI 折线回落), 删掉本地解析器约 500 行。验收: 改动前后各编一份 demo,
  图标页 light 148/2322000(0.0064%) / dark 185 / 基础页 **4** 像素, 全是 `Z` 闭合语义的 AA 级微差;
  墨迹量 143083 → 143082。

- ✅ **图标支持整段 SVG 字符串**(2026-10-11): 与 path 数据**共用 `Painter.Icon` 入口**(自动识别: 首字符
  是不是 `<`), 另有 `ParseIconSVG(svg)` 供"先校验/自己缓存" ——
  提取 `<path>`(多个, 支持 `<g>` 继承 fill/stroke/stroke-width)、按 viewBox 归一到 24 网格(任意网格)、
  按 fill/stroke 判定填充子路径、建议线宽取 stroke-width; 解析结果按 SVG 原文缓存, 几何照旧走图标几何缓存。
  于是 heroicons / feather / iconfont 的 SVG 可以直接贴, "只吃 24 网格 path"这条限制解除。
  验收见 DESIGN 第八节 44 末尾; 顺带修了 `parseIconPathEx` 对尾部空白的误判。

- ✅ **CLI 远程 registry(v2)+ `update` / `add --all`**(2026-10-11):数据源四态 —— 默认源
  `raw.githubusercontent.com/twgh/xui/main`(**自动选路:直连 → Gitee 镜像 `gitee.com/twgh/xui/raw/main`
  → GitCode 镜像 `api.gitcode.com/api/v5/repos/twgh/xui/contents`(v5 API, JSON+base64;
  raw 直链对 `.go` 匿名访问只回 403; 匿名 **50 次/分钟**, 实测 0.4~0.8s/请求) → 加速代理 ghfast / gh-proxy**,
  探测 = 拉一次 registry.json, 6s 超时, 第一个通的跑完本次命令, 选路
  过程逐行打 stderr; **拷到一半单文件失败(限速/挂起)按候选顺序换下一个源并把新源粘住**
  (`source.rest`, 免得留下半套 `ui/`); 组件跟 main 永远最新)、`XUI_REGISTRY` 指向本地目录或任意 `http(s)://` 基址
  (离线 / 锁组件版本; **写什么用什么, 不套镜像/代理**)、全不通时降级 `moduleRoot()` 内嵌源码树
  (显式设了 `XUI_REGISTRY` 则报错不降级);`update` 无 manifest
  语义 = 缺则建 / 一致跳过 / 用户改过警告跳过(`-o/--overwrite` 覆盖),省略组件名 = 更新 `ui/` 里已装的全部;
  新增 `-o/--overwrite` / `--all` 两个旗标。验收全绿
  (本地路径 / 本地 HTTP / 404 / 不可达 / 降级 / update 四态 / 逐组件单独安装后编译 / init 后编译),
  行为已固化为 `cmd/xui/main_test.go`(候选顺序 / 数据源解析 / httptest 端到端 / 显式源失败退出 / GitCode API 解码 / 单文件换源)。
  **顺带修掉 registry.json 的依赖缺陷**:`dialog` 漏 `backdrop.go`、`combobox`/`menu` 漏 `tooltip` 依赖
  —— 过去 `xui add dialog` 拷出来的组件根本编译不过。

- ✅ **页面滚动容器 `ui.ScrollArea`**(2026-10-14):载体是原生 **`LayoutFrame`(布局框架 = 布局盒子 + 滚动视图)**, 内容总大小**按子项自动撑开**、溢出自动出纵向滚动条(不需要手动 `SetTotalSize`)。探针实测定死一批语义(子项 `Fill` = 视口高且内层溢出**不算**滚动范围、`Weight` 恒为 0、`EnableOverflowHide` 是"布局时把子项高度裁掉"**不是**裁剪显示、首帧内容宽少一个滚动条宽要补排一次、`ScrollPosYV` 才是像素)与滚动条皮肤(轨道透明 + `bg-Border` 圆头滑块 + `OnThemeChangeFor` 换肤跟随)—— 全部记进 **DESIGN 第八节 66** + API 清单。demo 六个页全部换成滚动容器(不再被 `EnableOverflowHide` 裁掉): **自撑型**(基础/表单/导航/反馈/图标, 内容区 Auto 高, 窗口小于内容时滚)与**铺满型**(数据页 `SetContentFill`, 表格/树仍跟随窗口长高、整页不滚);demo 的 `row` 骨架同步改成行高 Auto(`rowStretch` 给数据页)。验收: 980×700 下表单页出滚动条 + 真实滚轮滚动(先点一下页内空白让窗口拿到焦点)+ 滚动条滑块随主题亮(#e5e5e5)→暗(#232323)现场切换;大窗(1500×880)无滚动条且卡片右缘 = 视口右缘 − 24 页边距(无"12px 空白"回归);六页 + 对话框浮层在两种尺寸/两种主题下逐页抓图核对;基础页加了一张"滚动容器"卡片(组件形态, 嵌套滚动)。⚠️ 滚动条**滑块拖动**在合成输入下未复现(裸 LayoutFrame 同样拖不动、表格 List 的能拖 —— 非组件引入的差异, 见 DESIGN 66 末尾)。

- ✅ **D2D COM 直通底座上游化到 xcgui**(2026-10-11):xcgui "交给 Go 侧裸 `ID2D1Factory` /
  `ID2D1RenderTarget` 指针却不给封装"留下的空白, 现在收进 xcgui 新子包 **`github.com/twgh/xcgui/wapi/d2d`**,
  COM 基础件(`ComProc`/`IUnknown`/`IUnknownVtbl`/`InterfacePtr`)抽到 **`wapi/com`** 子包共用, d2d 以
  类型别名引用(edge 自带一份同源实现, 迁移面大暂未动)。该包还提供 **amd64 的汇编转发器**(`CallWithFloat32`/
  `CallWithFloat64`/`CallWithFloat32RetFloat`)—— 让"形参含浮点"的 COM 方法也能调(SyscallN 设不了 XMM);
  三个实测坑(必须经 SyscallN 借 cgocall 切系统栈、调用点 16 字节对齐 + 32 字节 shadow、浮点槽号要按接口
  实测)与 386/amd64 的参数规则都写在包的注释里, 见 DESIGN 第八节 65。封装形式照 **`edge` 包(WebView2)**那套:每个 COM 接口一个文件、**按 SDK 声明顺序完整列出**的虚表结构体
  (索引由字段顺序决定, 不写魔数) + `ComProc.Call`(带 `//go:uintptrescapes`) + 中文注释与 MSDN 链接;
  接口结构体按 COM 继承链**值嵌入**(`IUnknown` → `ID2D1Resource` → …), 三件套与 `GetFactory` 各只实现
  一次, 自己的方法用 `vtbl()` 取 `i.Vtbl` 重解释虚表。含 ABI 三条硬规则、几何/画刷的创建释放、几何自检
  (`Usable`)、`NewGeometry`/`FillGeometryColor`。8 个单测:颜色换算 / 点打包 / **虚表槽位**
  (`TestVtblIndexLayout`, `unsafe.Offsetof` 钉住 FillGeometry=23、IsSupported=50 等) / **虚表解析**
  (`TestVtblResolution` —— 曾把"对象地址"当虚表, demo 启动即 panic PC=0x989680) / 结构体布局 / 继承复用 /
  nil 安全 / 空句柄。
  顺手把 `XDraw_GetD2dRenderTarget` / `XDraw_ConvRect` / `XC_GetD2dFactory` 的注释补齐了"直通调用拿的是
  父级坐标系, 必须先 `ConvRect`"这条坑。本仓库 `d2dicon.go` 只剩图标语义(描边展开 + 盒计算 + 缓存 +
  元素 alpha), 少约 200 行重复底座。**验收**:图标页/基础页 × 亮/暗三组 PrintWindow 抓图逐像素 **0 差**;
  图标页鼠标扫动 CPU 交替多轮无可辨差异(±10% 噪声); 细节见 DESIGN 第八节 43。**联调方式**: 本仓库用
  **`go.work`** 把本地 xcgui 工作副本接进来(该文件已在 `.gitignore`), `go.mod` 保持干净的 `require v1.4.2`。
  ⚠️ 在 xcgui 工作副本里**只跑 `go test ./d2d/ ./wapi/com/`**, 别跑 `go test ./...` —— 那是 GUI 库,
  全仓测试会不停弹真窗口(实测弹到没法干活); `go build ./...` / `go vet ./d2d/ ./wapi/com/` 是安全的。
  ⚠️ **本仓库 `go.mod` 的 go 指令是 1.18(用户明确不升级)**: 即使工具链是 Go 1.22+, `for range` 的循环
  变量也按**旧语义共享** —— 在循环里注册回调/起 goroutine 必须 `v := v` 拷一份(探针踩过: 三个元素全画成
  最后一个)。xui 自身的循环注册点已扫过(vet + 人工, 只有两处"for opts"是误报), 新代码保持这个习惯。
  后来又补了 `SetTransform` / `GetTransform`(矩阵按**指针**传, 不受"按值浮点走 XMM"的限制)——
  矩阵往返一致(单位矩阵 → 平移 12,34 → 设回单位矩阵)。
  借它把图标**位置从几何缓存里外置**了: 几何只按 (path, size, strokeW) 缓存、绘制时用平移矩阵移过去 ——
  窗口 resize 不再重建几何(实测尺寸来回切 10 次的 CPU 78/93ms → 62/62ms), 像素验收两组 0 差、图标页
  6/2322000 像素(AA 边缘); 前提是**成对恢复变换**(实测 XDraw 用的是绘制当时的 RT 变换, 见 DESIGN 43/65)。
  **再进一步(2026-10-11): 描边也交给 D2D 了** —— `wapi/com` 新增 `CallWithFloat32Stack`(最多 4 个栈参数)
  后, `ID2D1RenderTarget::DrawGeometry`(线宽在 XMM3、style 压栈)可用, 图标折线不再在 CPU 侧展开:
  `d2dicon.go` 删掉 `strokeOutline/addArc/miterPoint` 约 300 行, 缓存 key 精简成 (path, size), 角/端点由
  D2D 按 SVG 的 round 语义画(实测"角变圆"、墨迹量 +0.09%)。**性能: 微观 4 倍(2 万次 20ms vs 80ms)、
  真实场景在噪声内** —— 收益是质量与代码量, 不是帧率(详见 DESIGN 43 末尾)。

- ✅ **滚动条统一: shadcn 几何 + 两套配色 + 悬停显隐**(2026-10-15, 两轮):五处原生滚动条载体(ScrollArea
  页面与卡片内 / Table / Tree / ComboBox 弹层 / **多行 Input, 原先是系统滚动条**)的几何、配色、显隐
  全部收敛到 `components/scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`。
  **几何**: 轨道 10(`w-2.5`)、滑块内缩 1(`p-px`) → 8 逻辑像素宽、`rounded-full`(圆角 4);
  **配色两套**(用户对照浏览器原版提出): 容器型 = `bg-border` 悬停不换色(radix 语义), 嵌入式
  (Input/Table/Tree/弹层, 原版里是浏览器原生滚动条) = 前景色淡化 + 悬停加深(对齐 Win11 Fluent,
  亮 #8b8b8b→#636363 的观感); 颜色一律 `flatOn` **展平成不透明**(暗色 Border 是半透明白, 不展平
  会"同一滑块在页面 #232323、卡片 #2f2f2f"地不一致 —— 用户报的 bug);
  **显隐**: 初始 `SetSubtreeAlpha(0)`, 窗口 MouseMove + `XWnd_GetStayEle` 判定(宿主子树 → `Presence`
  淡入/淡出 150ms; 滚动条子树 → 滑块换悬停色), 不用 Show(false)(会改视口宽、内容跳变)。
  三个踩坑(详见 DESIGN 第八节 21): ① 滑块三态的背景对象必须**各用各的 id** —— 共用一个 id 时只有
  第一个生效, hover 会回退默认绘制(表现为"悬停时变宽且圆角消失"); ② 元素级 MouseStay/MouseLeave
  不可靠(多行 Edit 收不到 leave、原生 List 不把悬停路由给滚动条) → 一律走窗口级 GetStayEle;
  ③ 淡出到位那帧 anim 不重绘, 要 done 回调补 `XEle_Redraw`, 否则留淡残影。Table 的自适应列让宽
  12→14; registry 的 5 个组件已登记 `scrollbar.go`(逐组件安装+编译复验过), demo 多行输入框加了
  初值。**验收**: 五处载体 × 亮/暗, 隐藏/显示/悬停加深三态全部像素扫描通过(滑块物理 12px @150%;
  亮色静止 #919191 / 悬停 #6a6a6a, 暗色两容器统一 #2f2f2f)。

- ✅ **combobox 弹层进场动画(P3 收尾, 2026-10-17)**:下拉弹层是原生自建普通窗口, `SetTransparentAlpha` 对它无效;
  在 `XE_COMBOBOX_POPUP_LIST`(此时窗口**还没显示**)里转 `Window_Transparent_Shaped` 后整窗 alpha 即可播放
  100ms 淡入, 且**连续改 alpha 每帧生效**(与 57 的 tooltip 结论不同, 三档抓图
  #40A0FF/#A0D0FF/#FFFFFF), 区域裁剪的圆角与 ring/条目绘制全部照常。过渡挂宿主窗驱动 + `XC_IsHXCGUI` 守卫 +
  每次弹出用 `xui.NextPresenceKey()` 取键(各组件自起计数器会撞车, 见下条修复); 真实组件逐帧抓图看到
  5 帧淡入曲线。原理与坑见 DESIGN 第八节 30/69/70。
  - ⚠️ **本条"每帧生效"的结论已被推翻**(见下方"快速选择后点触发器没反应"): 那次探针是孤立地隔帧设三个值, 真实交互下逐帧改会概率性冻结在第 2 帧前后, 弹层停在 alpha 3/255 —— 看不见但照样吞点击。进场淡入已删除。

- ✅ **与 shadcn 的边角补齐(2026-10-17)**:
  - **aria-invalid**(Button / Input / ComboBox): `WithButtonInvalid()` / `WithInputInvalid()` / `WithComboInvalid()`
    + 运行时 `SetInvalid(bool)`; 红边框 + ring-3 destructive/20(暗色 border /50、ring /40), 常驻不要求焦点,
    与 focus-visible 同时成立时红色优先。demo 基础页加了三个按钮、表单页加了输入框, 亮/暗各验一轮。
  - **CardHeader / CardAction**: `ui.NewCardHeader(parent, space)` + `TitleStack()` / `ActionStack()`, 左列标题+说明
    占 `1fr`、右列动作 `auto` 且跨两行顶端对齐; demo 的「卡片」示例改成头 + 省略号动作钮。
  - **`MenuInset()`**: 对应 shadcn dropdown-menu 的 `inset` prop(`data-inset:pl-7`), 条目文字左缘 = 28,
    普通条目借此与带勾选/图标的条目对齐;demo 菜单加了一项。
  - **`--chart-1..5` / `--sidebar-*` 令牌进 `xui.Theme`**(Chart1..5 / Sidebar*): 组件库不用, 供用户应用取用;
    `manager_test.go` 有守卫防漏填。DESIGN 第七节已无菜单宽度差异(原"`XMenu_SetItemWidth` 无效"结论有误, 已修正, 见 DESIGN 64)。

- ✅ **修复: 弹层键撞车导致"点击没反应"(2026-10-17, 用户报障)**: 选过 sm 下拉框后第一次点"打开菜单"无反应(第二次才开),
  反过来先开菜单再点下拉框也无反应。事件日志证明 BN_CLICK 已触发 —— 菜单确实弹了, 但 alpha 停在 `AlphaFloor`(全透明)且处于模态,
  把下一次点击吃掉。根因是 `xui.Presence` 的幂等语义 + menu/combobox **各自的计数器都从 1 开始**(键撞上已完成的过渡 →
  动画不跑、apply 不调)。修法: 新增 `xui.NextPresenceKey()` 全局分配键(menu/combobox/dialog 全改用), `Presence` 立即
  apply 一次兜底。详见 DESIGN 第八节 70。

- ✅ **逐组件单独安装 + 编译回归(2026-10-17)**:20 个组件各建临时项目 `xui init` + `xui add <组件>` + `go build ./ui`
  (本地 `XUI_REGISTRY` + 模块 replace), 另加 `add --all` 一项, 21/21 通过。

- ✅ **Calendar / DatePicker / TimePicker 三个新组件(2026-10-18, 参考 HUMUI HouTai017)**:
  - **`ui.Calendar` 月历卡片**(`components/calendar.go`): 全自绘(不用原生 MonthCal —— 其格子/比例不可控), 248×292、
    6×7 固定网格(高度不跳), 格 32 圆角 6; 选中实心/今天浅底/悬停过渡/非本月弱化; 周一开头可配(`WithCalendarWeekStart`);
    可独立摆放, 弹层内 `WithCalendarFlat` 免双层边框。
  - **`ui.DatePicker` 日期框**(`components/datepicker.go` + 弹层骨架 `popup.go`): 触发区 = outline 按钮(calendar 图标 +
    "2006年1月2日"/占位符), 弹层 = 自建阴影窗(卡片由 body 画, 月历 flat) + item 左下对齐(sideOffset=4)、
    下方放不下翻上方、夹屏幕内; 淡入 100ms(Presence 挂宿主窗 + `NextPresenceKey`); 选中即回写+关闭+回调;
    点外部(KillFocus)/ESC 关闭; **点触发区关闭后 250ms 内不重开**(KillFocus 先于 LButtonUp, 否则"点两次才关掉");
    Open/Close/IsOpen 程序化控制, 销毁时弹窗跟着销毁, `OnThemeChangeFor` 跟主题重绘。
  - **`ui.TimePicker` 时间框**(`components/timepicker.go`): 触发区同款(clock 图标 + "09:30"); 弹层两列原生 ListBox
    (时/分, 64 宽、8 项可见), 条目背景/文字自绘(选中 bg-primary 反色、悬停 bg-accent 过渡), 滚动条统一皮肤;
    选择即生效、弹层保持打开; 打开时选中项滚到中间(行步进 = 项高+1 + 首项固有内缩, 见 DESIGN 72)。
    ⚠️ ListBox 手动创建的一批语义(适配器先行/两套索引/模板文字静态色)记 DESIGN 第八节 72。
  - **弹层骨架 `popup.go`**(阴影窗 + 自绘卡片 body + KillFocus/ESC 关闭 + 定位翻转 + 淡入): datepicker/timepicker 的
    files 里各带一份, registry 已登记。
  - **验收**: 亮/暗、打开/选中回写/外部关闭/重开/翻月逐项通过、demo 表单页两张新卡片(深色滚动截图);
    逐组件安装编译 calendar/datepicker/timepicker + `add --all` 全部通过; registry.json 已登记 3 个组件。
  - 新坑记 DESIGN 第八节 **71**(阴影窗元素坐标原点 = 窗口矩形原点, 卡片要放 (band,band))、**72**(手动 ListBox 自绘列表语义)。

- ✅ **修复: 日期框/时间框弹层"闪烁/抖动"(2026-10-18, 用户报障 + 60fps 录屏, 四个根因全修)**:
  - **根因 A(白闪)**: 弹窗不设背景时 xcgui 画**白色默认底**, 淡入期间白底从半透明卡片下透出(帧级抓图: 第二次
    显示 mean=228 白帧)。修法: 弹窗创建时 `XBkM_Clear + XBkM_AddFill(alpha=0)` 垫透明填充(同第八节 30)。
  - **根因 B(淡入路线)**: 整窗 alpha 对透明窗只认第一次调用(57), 复用弹窗整段失效(两次显示都瞬间全亮)。
    修法: 淡入改**元素 alpha**(SetSubtreeAlpha(body) + 每帧重绘, tooltip 同款), 显示前压 0。
  - **根因 C**: `hoverTick` 对已到位的旧 presence 条目仍每帧调 apply, 二次弹出时互相打架(实测 12↔85 交替)。
    修法(anim.go): settled 条目不再调 apply; menu/combobox/dialog 同表受益。
  - **根因 D(用户 60fps 录屏"抖动", 快速点击更明显)**: 过渡挂**宿主主窗口** → 淡入 100ms 里每帧全量重绘
    整个表单页, 快速点击时重绘排队、UI 线程卡到出系统忙碌光标(用户录屏可见蓝圈)。修法: 整窗 alpha 时代挂宿主窗
    是被迫的(DWM 合成要宿主带动), 改元素 alpha 后**过渡挂弹窗自己**(400×500 物理像素, 每 tick 重绘无压力)。
  - **根因 E(用户逐帧"直接显示 → 消失 → 再淡入")**: 复用窗口再次 ShowWindow 首帧合成的是**表面残留的上一次
    内容**(关闭前全亮卡片), 元素 alpha=0 要等首次重绘才落地。修法: **屏幕外预显示** —— SetWindowPos(-30000) +
    Redraw 画好透明首帧再挪到目标位置。
  - **根因 F(时间框点击闪退 + "SetAlpha 句柄无效"弹窗)**: `GetChildByIndex` 对原生 ListBox 返回**非 0 但无效**
    的子句柄(type=62 延迟创建对象, 显示后依然存在, 每列约 8-16 个) → `XEle_SetAlpha(无效句柄)` 触发 DLL 错误
    弹窗 —— DLL 错误不产生 Go panic, 被吞掉后可能 access violation 闪退。修法: `fadeSubtreeAlpha` 与
    `setAlphaTree` 对每个句柄加 `XC_IsHXCGUI(h, XC_ELE)` 守卫 + fadeSubtreeAlpha 跳过滚动条子树(滚动条显隐自理)。
  - **根因 G(概率性半透明透字 + 滚动条不隐藏)**: 淡入中途被 hide(快速点击/选日期/点外部)时 alpha 停在中间值、
    条目未 settled —— ①300ms 内 fast 重开只设属性**没有重绘**, 屏幕上还是中间值的残留帧(弹层半透明透出页面文字,
    且滚动条随子树一起可见); ②未 settled 的旧条目每帧 apply(v=1.0) 与新状态互相覆盖。修法: Hide 时
    `xui.DropTransitions(弹窗)`(新 API, 清条目+关残留定时器) + `SetSubtreeAlpha(1)` 归位终态 + OnFaded 恢复
    滚动条隐藏; fast 路径补 `Redraw(true)` 让属性落地。验证: 60fps 录屏半透明卡住段 0, 快速交互进程存活。
  - **交互补充**: 关闭后 300ms 内重开(`popoverReopenFadeSkip`)**跳过淡入直接完成态** —— 快速连点时弹层反复从
    透明渐显会"眨眼", 对齐 CSS 过渡的中断语义(不重播动画)。
  - **滚动对齐(用户截图"两列选中项不在同一高度")**: 两个叠加原因 —— ①`SetSelectItem` 触发 xcgui 内建"选中项
    滚入视口"的异步滚动, 布局完成后执行并**覆盖**紧跟的 ScrollPosYV; ②`ScrollPosYV` 的落点被吸附到**滚动单位
    网格(默认 20px)**, 实测 170→160、779→760(都是 floor(p/20)×20), 两列请求值除 20 的余量不同 → 吸附后错位。
    修法: `syncLists` 不再 SetSelectItem(选中态由 DrawItem 按组件值绘制, 不依赖原生 Select) + 滚动定位挪到
    `OnFaded`(首帧绘制前) + **`SetLineSize(20, 行步进)`** 把滚动单位设为行步进(29), 两列 pos 相差整行数,
    吸附后仍差整行 → 选中项必然同高。PrintWindow 抓弹窗验证: 两列选中块同一水平线、滚动条隐藏。
  - 详见 DESIGN 第八节 73(含排查方法论: 帧级抓图 > 亮度采样; 弹窗进程内自抓 > 外部轮询; 录屏坐标要按实际
    内容校准; DLL 错误弹窗 ≠ Go panic)。验证: 60fps 录屏逐帧分析 —— 白帧 0, 弹层出现为纯渐显(19→31 无全亮相),
    快速连点段无爆闪、无忙碌圈; 时间框连点两次进程存活。

- ✅ **Calendar 三块新能力(2026-10-19, 参考用户截图)**:
  - **年月可选**: 标题行原来是一整块"2026年10月", 现在拆成「2026年 ⌄」「10月 ⌄」两个热区, 点开是贴着标题的自绘下拉
    (bg-popover + ring, 行高按 12 行塞满可用高度自适应, 当前值右侧 check, 超出高度滚轮滚动 + 2px 细滚动条); 面板内点击=取值、
    面板外=关闭、点另一个热区=直接切换; 只改显示月份不动选中日期。年份范围默认"视图年 ±50", `WithCalendarYearRange` 可配。
  - **禁用日期**: `WithCalendarDisabled(fn)` + 运行时 `SetDisabled`, 内置 `CalendarDisableBefore/Weekend/Dates/Any` 四个 helper;
    禁用格 text-muted-foreground opacity-50, **不可点、无悬停、不触发 OnSelect**(探针点周六验证无选中变化)。
  - **页脚**: `WithCalendarFooter(items...)` / `WithCalendarFooterToday()` / 构造后 `AddFooterItem/Button/Text` / `ClearFooter`;
    页面全宽 1px 分隔线 + 居中按钮行(ghost 化: 悬停 bg-accent), 元素未指定高度时自动长高 `CalFooterHeight()`(37);
    要放原生控件时用 `FooterRect()` 取内容区矩形自行摆放(子元素盖在自绘内容之上, 点击不穿透)。
  - **验收**: 探针(亮色, 两张卡片: 周末禁用 / 页脚)逐项截图 —— 年列表开合与滚轮滚动、选年切月、月份列表、页脚「今天」回本月并选中、
    点禁用日期无响应; demo 表单页「月历卡片」改为 周末禁用 + 页脚(今天/清除)。`go test ./...`、`go vet` 通过。
  - **注意**: 下拉是**元素内自绘**(不是独立弹层窗口), 因此受卡片尺寸限制 —— 行高会自动压缩以塞下 12 行; 鼠标离开卡片即收起
    (自绘下拉没有原生弹层的 KillFocus 失焦机制); 滚轮事件只在打开时 `EnableEvent_XE_MOUSEWHEEL(true)`, 关闭即还回外层 ScrollArea。

- ✅ **月历等比缩放 + 弹层不压任务栏(2026-10-19 用户报障)**:
  - **月历内部随尺寸缩放**: 原来几何是写死的 32px 格子/12px 内边距, demo 把卡片放大 1.12 倍只有空白变大。现在
    `Calendar.geom()` 按**元素宽度**算一整套几何(内边距/格子/标题行/星期行/页脚/字号全部等比), 高度只决定 6 行之间的
    行距(多余空间均摊到行间), 绘制与命中判定共用同一套值; 下拉面板的行高/字号/内边距同样跟着缩放。
  - **默认尺寸改为 1.12 倍**: `CalWidth/CalHeight/CalFooterHeight` = 278×327 / 41(设计基准 248×292 的比例), 即用户
    截图里"乘 1.12"的效果; demo 去掉手写的 ×1.12, 直接用 `NewCalendar(0,0,0,0,...)`。
  - **新增 `CalHeightFor(width)` / `CalFooterHeightFor(width)`**: 只想改宽度时 `cy` 传 0(NewCalendar 自动配平),
    或显式 `CalHeightFor(w)+CalFooterHeightFor(w)` —— 手写 `CalHeight()*1.5+CalFooterHeight()` 会少算页脚的缩放, 底行被页脚压住。
  - **弹层夹边界改用显示器工作区**(`MonitorFromWindow`+`GetMonitorInfo` 的 `rcWork`, 见 DESIGN 第八节 74): 最大化窗口 +
    页面滚到末尾时, 贴着任务栏的日期框向下弹会被任务栏盖住; 现在下方放不下先翻上方, 上方也放不下就贴工作区底边。
    ⚠️ **必须传 Win32 HWND**(`xc.XWnd_GetHWND`), 传 xcgui 窗口句柄会让 `MonitorFromWindow` 返回 0 并静默退回整屏 —— 这是
    第一次修完仍然"被任务栏遮挡"的根因(用户第二次报障); 现在取不到显示器时还会退回 `GetDesktopWindow()` 再查一次。
  - **验收**: 探针(默认 + 1.5 倍宽两张卡片, 高度传 0 自动)截图确认内部等比、6 行不裁切; **真实复现路径**——demo 表单页
    最大化 + 滚到最底部点日期框, 枚举弹层矩形 (48,531)-(489,1046), 底边 1046 < 任务栏 1520, 弹层翻到触发区上方完整可见;
    demo 表单页月历卡片改用默认尺寸。
  - **注意**: 探针里别在 goroutine 里 `dp.Open()` —— xcgui 创建窗口/元素不是线程安全的, 会出现"弹层白板不画月历"的假象(实测)。
  - **上游化(同一日)**: xui 里散落的 Win32 声明全部收进 xcgui 的 wapi —— 新增 `GetWindowDC / ReleaseDC /
    PrintWindow / MonitorFromWindow / GetMonitorInfo` + `PW_` / `MONITOR_` 常量 + `wapi.MONITORINFO`(types 包),
    gdi 侧复用已有的 `gdi.CreateCompatibleDC / CreateDIBSection / SelectObject / DeleteObject / DeleteDC`;
    `components/backdrop.go` 的 `modUser32/modGdi32` 与 `components/popup.go` 的 MONITORINFO 声明都已删除,
    xui 中**不再有裸 `syscall.NewLazyDLL`**。上游补了 `TestGetMonitorInfo / TestGetWindowDC`(实测工作区
    1707×1019 vs 整屏 1707×1067, 任务栏 48px 被扣掉)。

- ✅ **Toast 通知(2026-10-19, `components/toast.go`)**:
  - **载体**: 窗口内覆盖元素(每条通知一个自绘 Element, 无子元素), 右下角堆叠、跟随窗口移动/缩放/最小化;
    不占用原生控件, 关闭按钮/动作按钮的命中在卡片局部坐标里算。
  - **API**: `ui.NewToaster(w.Handle)` + `Show(ToastOptions{...})` / `Success / Error / Warning / Info`;
    `Duration` 传负数 = 常驻, `Action` + `OnAction` 动作按钮, `CloseButton` 显示关闭钮;
    选项 `WithToastPosition/Width/Offset/Gap/Duration/MaxVisible`, 实例 `Update/Close/Handle`。
  - **交互(对齐 sonner 默认值)**: 自动关闭 4s; **悬停任意一条暂停全部倒计时**(窗口级 MouseMove +
    `GetStayEle`, 元素级 Stay/Leave 不可靠, 见 DESIGN 21/75); 超出 3 条移除最旧;
    进场淡入 + 滑入 12px(200ms), 退场淡出 + 侧滑 24px(180ms); 主题切换重新量高重排。
  - **验收**: demo 反馈页「通知」卡片(成功/错误/警告/信息/带动作/长文本), 亮/暗两主题抓图:
    单条与三条堆叠、图标语义色、长描述折行、动作按钮与关闭按钮点击、悬停 6s 不消失、移开后自动关闭;
    `go build ./...` / `go vet ./...` / `go test ./...` 通过。
  - **内边距对称性(像素实测)**: 卡片排版坐标必须用"卡片矩形"(元素比卡片四周大 6px 画投影),
    否则卡片可见高度少 12、下内边距被吃掉 6(用户报障"底部空白比顶部小")。修后两行卡片
    topGap=botGap=28 物理像素(@150% DPI)、单行卡片 26/27; `GetTextShowRect` 的 CY 已含行距
    (13px: 单行 18 / 两行 36), 别再补偿(见 DESIGN 75)。
  - 原理与坑见 DESIGN 第八节 **75**(单元素自绘 / 窗口级悬停 / 倒计时用元素 XCTimer / 位置动画挂窗口整窗重绘)。

- ✅ **DESIGN.md 改造为"随源码分发的自定义指南"**(2026-10-19):
  - `xui init` 现在把 `docs/DESIGN.md` 拷到用户 `ui/DESIGN.md`(已存在则跳过, 读不到不阻断) —— 组件注释里
    "见 DESIGN 第 N 节"在用户项目里可达。
  - 内容改写: 去掉 73 处日期、10 处"用户报障"、43 处 demo 页签引用;
    `components/` 路径改 `ui/`; 第五~七节从"CLI 实现现状 / 仓库结构 / 路线图"改为
    "安装与更新 / ui 目录与组件依赖 / 组件覆盖与已知限制"。**章节与第八节条目编号未动**(源码注释靠编号引用)。
  - 开头加"按需查"导航 + 两条说明(示例程序 = 仓库 `demo/`;探针 = 临时验证小程序)。
  - AGENTS/HANDOFF 补规则: DESIGN 会拷给用户, 不写日期/报障时间线/待办/发布流程。
  - 验收: 临时项目 `xui init` → `ui/DESIGN.md` 存在且与源文件 MD5 一致; 重复 init 跳过;
    `go build ./...` / `go vet ./...` / `go test ./...` 通过。

- ✅ **AppWindow 主窗口封装**(2026-10-19, `components/window.go`):
  - `ui.NewAppWindow("标题", w, h)` = 圆角阴影 + 无系统标题栏 + 自绘三键(最小化/最大化/关闭) + 主题化底色;
    内容放进 `w.Content()`(纵向流式 Weight 1), `AddCaptionButton` 在标题栏加自定义按钮(排在最小化左边)。
  - 选项: `WithAppWindowMinSize/CaptionHeight/NoMinMax/Icon/ContentPadding`。
  - 封装期踩到两个坑(细节见 DESIGN 76): ① 窗口 `EnableLayout` 漏调 → 内容区高 0, 界面只剩标题栏;
    ② 双击判定用 `GetStayEle != 0` → 因标题/图标 MouseThrough 而整段失效, 改成只拦 `XC_BUTTON`。
  - 验收: 亮/暗两主题、最大化(图标自动切"还原")、双击最大化/还原、拖动、关闭;
    `ui/window.go` 单组件安装编译通过(依赖 button)。
  - **demo 已整体迁移**(标题栏的 logo / 布局边界 / 主题按钮走 `WithAppWindowIcon` + `AddCaptionButton`,
    tooltip 照常绑定): 六页切换、亮/暗、最大化/还原、双击标题栏、拖动全部回归通过; demo 自己那套
    窗口/标题栏/双击代码已删(`captionH` / `shadowSize` 常量一并移除)。
- ✅ **发布 GitHub + 首个版本 v0.1.0**(2026-10-19):
  - 仓库公开(https://github.com/twgh/xui),`main` 已推;tag **v0.1.0** 已打并推送(指向 `7d2e7a7`)。
  - 依赖: `go.mod` 升到 `github.com/twgh/xcgui v1.4.3`(含 `svgp` / `wapi/d2d`), `go.sum` 已更新。
  - 验收(`GOWORK=off`, 即不依赖本地 xcgui 工作副本): `go build/vet/test ./...` 全过;
    全新项目 `go run .../cmd/xui@v0.1.0 init` 生成 theme.go + 基础组件 + DESIGN.md;
    `GOPROXY=direct go list -m ...@latest` → v0.1.0。
  - ⚠️ goproxy.cn 的 `@latest` 元数据有 CDN 缓存延迟, 刚发版时可能返回旧伪版本(内容同样是 main 最新),
    显式 `@v0.1.0` 或 `GOPROXY=direct` 不受影响; 其 zip 下载偶发一次 `stream error`(重试即好)。
- ✅ **xcgui 已发 v1.4.3(含 `d2d`/`svgp`), `go.mod` 已升级**(2026-10-19): 不再依赖 `go.work` 联调,
  `GOWORK=off go build ./...` 通过;发布前必须保持"公共模块可编译" —— 改回本地联调时记得再验证一次。
- ✅ **D2D 直通能力的上游化, 一次做齐**(2026-10-11)(趁上游发版窗口把原待办三项做完, 细节见 DESIGN 第八节 43 末 / 65):
  ① **"按值浮点形参"的封装扩到基本图元**: `DrawLine`(线宽在**栈[0]** —— 5 个参数, 位置 4 已超出前 4 个 XMM 槽)/ `DrawRectangle` / `DrawEllipse` / `FillRectangle` / `FillEllipse`(各带 `*Color` 便利版); 转发器升级为"最多 4 个栈参数"的 `wapi/com.CallWithFloat32Stack`(栈上的 float 占 8 字节槽、值放低 4 字节)。
  ② **形状几何与参数上移**: `ID2D1Factory` 补 `Create{Rectangle,RoundedRectangle,Ellipse}Geometry`(配 `D2D1_ELLIPSE`/`D2D1_ROUNDED_RECT`); 绘制/填充口参数从 `*ID2D1PathGeometry` 上移到 `*ID2D1Geometry`(各几何加 `Base()`, `Usable()` 随之上移), 于是"任意几何"都能画。
  ③ **几何缓存策略**做成通用的 `d2d.GeometryCache[K comparable]`(按工厂作废 + 失败也记 + 统一 Release), 本仓库的图标几何改用它 —— 重构后墨迹量逐像素不变(143083), 鼠标扫动 734/765ms(此前 828/836ms)。
  ④ 原来的"描边展开算法上游化"**作废**: 描边走 D2D 真描边(见上一条), CPU 展开代码已从本仓库删除。
  8 种图元/几何一次抓图全对; `wapi/com`/`d2d` 单测(槽位/布局/nil 安全)在 CI 与离线均可跑。

- ✅ **CLI 参数对齐 shadcn + 新项目脚手架**(2026-10-20):
  - `xui init -n <项目名>` 创建完整 Go 项目: `go.mod`(依赖版本从 CLI build info 取,
    本地 devel 构建回退 `v0.2.0`)、`main.go`(AppWindow + 示例卡片)、`README.md`、
    `.gitignore`、`ui/`(theme.go + DESIGN.md + 基础组件); 非空目录需 `-f/--force`;
    生成后**自动在项目目录执行 `go mod tidy`**（`GOWORK=off`、2 分钟超时，失败只告警不阻断），
    离线可加 `--no-install` 跳过。
  - `add/update`: `-o/--overwrite`(移除旧 `--force` 命名)、`-p/--path`、`-a/--all`、
    `--dry-run`、`--view[=文件]`、`--diff[=文件]`; 新增 `view` / `info` 命令; `list/search` 支持
    `-q/-l/-o`; 全局 `-c/--cwd`、`-s/--silent`。
  - `xui init [ui目录]` 旧行为不变; 参数解析、脚手架、dry-run、diff、过滤/分页均有单测。
  - 验收: 临时目录生成(自动生成 go.sum) → `go build` + 启动截图 + 关闭进程正常;
    手动跑过 list/view/dry-run/path/overwrite/diff/silent; `go build/vet/test ./...` 通过。
- ✅ **Menu 弹层宽度对齐 shadcn**(2026-10-20, trigger 宽 + min-w-32 下限):
  - 依据: `XMenu_SetItemWidth` 逐项设"条目文本区宽度"实测有效 —— 弹窗宽 = max(各条目有效宽) + 固定开销
    (BorderSize(4)/条目高 28 配置下 48), 且**设定值完全覆盖自然宽**(自然文本宽 229 的长条目设 60 后弹窗仍 108,
    长条目不会撑宽弹层)。早期"实测无效"的结论已证伪修正(详见 DESIGN 64)。
  - 实现(`components/menu.go`): `PopupAtElement` 记录触发区宽(逻辑像素), build 时给每个**顶级**条目设
    `max(触发区宽, 128) − 48` → 弹层宽 = max(触发区宽, 128), shadcn `w-(--radix-dropdown-menu-trigger-width)
    min-w-32` 语义齐了; 右键菜单(PopupAtCursor)无触发区 → 内容自适应 + 128 下限; 子菜单条目不设宽
    (shadcn SubContent 是内容自适应 + min-w-96, 不跟触发区); 分隔栏不设。长文本由自绘 `XDraw_SetClipRect`
    硬裁剪(= overflow-x-hidden; 原生条目文本同样裁在弹层边缘, 截图验证)。显式覆盖 = `Menu.Width(w)`
    (对应 shadcn 在 Content 上加 `className="w-56"` 这类宽度类 —— **shadcn 参考卡 App.tsx:815 就是 w-56=224**,
    所以参考页里看到的"默认"其实带了这个覆盖; demo 已同步 `Width(224)` 对齐)。
  - 验收: 探针(直接用 `ui.Menu` + 真实触发钮)断言弹窗客户区宽 —— 触发钮 240 → 240 ✓, 触发钮 60 → 128 ✓,
    `Width(224)` → 224 ✓; 窄触发 + 超长条目截图放大: 文本裁在弹层边缘内、勾选/快捷键/destructive/label 全部正常;
    `go build/vet/test ./...` 通过。

- ✅ **关窗后进程不退修复**(2026-10-20, 用户报障):
  - 现象: demo 点关闭钮后主窗口消失, 进程仍在(枚举到一个 `XCGUI` 残留小窗,
    悬停过标题栏按钮时还可能直接留在屏幕上)。
  - 根因: 消息循环的退出条件是**所有用户创建的窗口对象都销毁**, 不看可见性。Tooltip 的置顶小窗只 `SW_HIDE`
    不销毁, 且宿主销毁后挂在宿主上的退场过渡不再执行 —— 于是主窗口关了, `a.Run()` 还在跑。
  - 修法(`components/tooltip.go`): 气泡首次在某宿主窗口显示时给宿主挂 `WM_DESTROY`, 销毁气泡窗并清状态
    (先清状态再 `DestroyWindow`; 宿主销毁期间 `dying` 集合拒绝迟到的 `show()`, 避免 `XEle_SetRect` 无效句柄错误框);
    气泡属于其他仍存活宿主时保留。
  - 验收: ①气泡可见时关主窗 → 进程退出; ②气泡已隐藏时关 → 进程退出; ③标题栏"布局边界/主题"两个按钮各来一轮,
    均无残留窗口; `go build/vet/test ./...` 通过。详见 DESIGN 77。

- ✅ **Input 补 shadcn 的 input-group 特性 + 步进器**(2026-10-20, 参 shadcn input 文档):
  - 新增内嵌附加件(全部自绘在壳体内部, 不建子元素): `WithLeadingIcon` / `WithPrefixText` / `WithSuffixText` /
    `WithClearButton` / `WithPasswordToggle` / `WithCounter(max)`; 文字区左右内边距按附加件实际宽度算
    (`syncPadding`), 清除钮空内容时留位不画(打字时文字不跳)。
  - 新增步进器 `WithStepper(min,max,step)` + `SetNumber/Number/OnStep`: 右侧 −/+ 内嵌按钮、↑/↓ 键 ±1 步、
    回车提交, 失焦/回车时夹取+吸附+非法回退; 步长小数位决定显示精度。到边界时对应按钮画成不可点。
  - 两个深坑(已记 DESIGN 78/79): ①内嵌按钮动作**不能放 LButtonDown** —— 连点时第二下是 `LBUTTONDBLCLK`,
    收不到 Down 事件且原生 Edit 会当成"选中单词"(现象: 点两下只加一步 + 数字被全选), 放 `LButtonUp` 才对,
    且 `SetFocus()` 后必须 `MoveEnd()`; ②密码遮挡符默认 `*` 太小, 显式 `XEdit_SetPasswordCharacter(hEle,'●')`
    (每次重开密码模式都会重置回 `*`)。
  - 验收: 亮/暗双主题截图放大核对(步进器/前后缀/清除/圆点/计数); 交互实测 —— 连点 + 三次 42→45(无选中)、
    999 失焦夹到 100、`abc` 失焦回退 100、按 ↓ 两次 100→98、点眼睛切明文 "secret-123" 且图标变划线眼、
    点 × 清空。`go build/vet/test ./...` 通过。
  - 2026-10-20 用户报障三连修(详见 DESIGN 80/81):
    ①**多行文本域贴边框** —— `syncPadding` 在多行分支直接 return, 原生 Edit 的内边距是 0; 现给 `10/6/10/6`,
    并把演示的文本域高度 72 → 84(否则加了纵向内边距后少显示小半行); 实测滚动条与描边之间有了间隙。
    ②**后缀 `.com` 贴右边框** —— 尾部第一件的内缩改成按类型取(`trailingInset`: 按钮 4, 文字/计数 10),
    与左侧图标的内缩对称; 绘制与内边距同源, 不会再走样。
    ③**到上限后连点 + 会选中数字** —— 到边界时不再"不登记命中区"(那样点击会穿透到原生 Edit, 连点的第二下
    变双击选中单词), 改成命中区无条件登记 + 图标画淡 40%, 动作由 `stepBy` 夹住(空操作); 另关掉
    `EnableAutoSelAll(false)`, 避免 `SetFocus()` 把内容整段选成蓝底/留下选中残影。
    验收: 值设 100 后连点 + 两次 → 内容不被选中、值仍是 100、"+" 显示为淡灰; 亮/暗双主题截图核对。
  - 2026-10-20 追加两条(详见 DESIGN 78 补充 / 82):
    ①**双击内嵌按钮仍然选中文字** —— 连点的第二次**按下**是 `WM_LBUTTONDBLCLK`, 不走 `LButtonDown`,
    只在 Down 里吞是挡不住的; 补 `AddEvent_LButtonDBClick`(与 Down 共用处理函数)。实测: 42 上双击 + → 44、
    连点 4 下 → 48、到 100 后再连点 4 下 → 仍是 100 且无选中。
    ②**输入框文字比周围小一圈 / 数字挤在一起** —— 原生 Edit 不继承窗口字体, 用的是 xcgui 内置
    **9pt 微软雅黑**(探针 `XEle_GetFontEx` 实测), 主题是 11pt; 构造与换主题时都 `SetFont(FontHandle(t.FontDefault))`。
    验收用"同一串 18 位数字的实际绘制宽度": 126 → 153 逻辑像素, 与自绘前缀的 152 基本一致。
  - 2026-10-20 再修一条(详见 DESIGN 84): **密码框切明文后字距被撑开**(`s e c r e t - 1 2 3`,换主题后自愈)、
    再切回密文圆点也被撑开 —— `XEdit_EnablePassword` 只换遮挡开关, 缓存的字符步进还是上个模式的。
    修法: 切换后 `syncPadding()` 触发一次重排(**不要用重设文本**, 会给用户假的 OnChange)。
    顺带公开 `SetPasswordVisible(bool)` / `PasswordVisible()`, 与点眼睛走同一路径。验收: 探针里明文行与
    同窗口普通输入框的墨迹宽度/字形块数完全一致(108 逻辑像素 / 10 块), 密文圆点行与切换前逐像素一致。
  - 2026-10-20 再修一条(详见 DESIGN 85): **有焦点环时右侧按钮的悬停底色压到边框上** ——
    环带画在元素内缩 3px 处, 而按钮的悬停底色只从 4px 起, 圆角处就贴上了;
    尾部按钮内缩 4 → **6**(环宽 + 3, 同 shadcn 的 `pr-1.5`)。验收: 深色主题下聚焦 + 悬停放大截图, 方块与环之间留 3px。
  - 2026-10-20 补齐步进器形态(用户对照原生 number 输入框提的): `WithStepperArrows()` 画**上下三角**
    (20×24 的列, 每枚箭头一个 8×8 正方形绘制块 + 上下各 2px 留白 —— 用户反馈"再缩小些, 上下要有留白",
    三角实际 8×6.4; 箭头 = 24 网格 SVG path 走**图标管线**
    (`!` 填充前缀 + Q 曲线倒角, D2D 几何按 path+size 缓存) —— 手写多边形逼近圆角试了三版
    (尖角/浮点/只圆尖角), 10px 尺寸下不是歪就是糊、底角圆弧还把墨迹洇到底边外, 最终改走 SVG);
    `WithStepperAutoHide()` 让步进按钮只在"鼠标停在输入框上 或 输入框有焦点"时出现, 否则只占位(文字不跳、
    该区域点击落回文本区)。步长本来就是 `WithStepper(min,max,step)` 的参数(1/10/0.1 均可, 小数位由 step 推出)。
    验收: 悬停 → 箭头出现; 点 ↑ 两次 320→340、点 ↓ 330; 移开鼠标但仍有焦点 → 保持显示;
    Tab 移走焦点 + 鼠标移开 → 箭头消失、值不变。
    按压反馈(用户要求"像原生按钮那样往下沉"): 按下时图标**下沉 1px**、松开回弹, 底色保持悬停态
    (上下三角的半高块底色也恢复了 —— 用户最终选择与 −/+ 观感统一, "无底色 + 缩放"方案已弃)。
    实现是"簿记 + 画时平移": Down 记 `pressed`(元素 Down 态拿不到 —— Down 事件被吞了, st.Pressed 恒 false),
    paint 时把图标矩形整体 +1; 千万别用 d2d 变换缩放那套 —— 每帧尺寸都变会击穿几何缓存。
    到值域边界的 dim 按钮(用户报障"禁用态还能点击, 图标也跟着变"): 命中登记带 `dim` 标记,
    Down/Up 仍然吞掉点击(防双击穿透)但**不进按压反馈也不执行动作** —— 禁用态图标必须完全静止。
  - 2026-10-20 再追加两条(详见 DESIGN 80 补充 / 83):
    ①**文字偏上(下面留白更大)** —— 修法: 单行内边距改成上 7 / 下 5(多行仍 6/6), 实测 32 高输入框的
    墨迹上下留白 14/17 → 16/15(物理像素)。
    ②**带计数的输入框字体/字距异常** —— `BeginPaint` 的建图对象在回调之后还给原生文字用,
    `paintAddons` 里为计数换的 12px 字体没还原, 正文就跟着变小、字距走样; 现在 `onPaint` 结尾统一
    `p.Draw.SetFont(主题字体)` 还原。验收: 搜索/密码/计数三行正文字形与字距一致, 计数 "15/120" 仍为 12px 灰字。

- ✅ **Button 补齐 shadcn 剩余可落地特性 + ButtonGroup**(2026-10-21, 参 shadcn button / button-group 文档):
  - Button 新增: **行内图标+文字** `WithButtonLeadingIcon` / `WithButtonTrailingIcon`(对应 `data-icon=inline-start/end`,
    gap 4/6、图标侧内边距收窄 6/8; 图标尺寸按档 14/16/18);**胶囊** `WithButtonRounded`;**弹层触发区态**
    `WithButtonExpanded`(aria-expanded: outline/ghost 换 bg-muted、secondary 保持 secondary 底)与
    `WithButtonHasPopup`(aria-haspopup: 按下不再整体下沉);暗色 outline/ghost/destructive 底色按上游
    `dark:bg-input/30` 等规则补齐。`cx=0` 的文字按钮现在按 `TextWidth` 自动量宽(纯图标仍用方形预设),
    `FitWidth()` 可手动重算。
  - 新增 `components/buttongroup.go`: `ButtonGroup`(横向/纵向、宽高按内容自撑、首尾外角、相邻边直角、
    嵌套组自动 8px 间距)+ `ButtonGroupText`(bg-muted 文本块)+ `ButtonGroupSeparator`(bg-input 分隔条,
    跟随组高);纵向组用逐项 -1 上边距消掉按钮 1px 按压预留缝。输入框/下拉框壳体也接入了同一套组内圆角
    (含焦点环/校验环),可直接放进组里。公共组语义抽到 `components/group.go`。非显然的布局语义记在 DESIGN 第八节 86。
  - `registry.json`: 新增 `buttongroup`(依赖 button);`button`/`input`/`combobox` 的文件清单都带上 `group.go`,
    并已在临时模块里逐组件拷贝编译验证(button / input / combobox / buttongroup 四种闭包均通过)。
  - 2026-10-21 追加修复: **暗色主题下禁用按钮看起来像可用** —— 旧实现只把壳体乘 50%, 文字/图标仍是全色
    (旧结论"文字保持原色"只在白底+白字时成立)。现在按钮文字/图标按宿主 `Card` 对半预混;暗色 outline
    禁用底用 `input/30 × 50%`、destructive 用 `/20 × 50%`, 亮色实心钮白字仍是白色。验收: 暗色探针里
    secondary/outline/primary/destructive 的禁用态与正常/展开态一眼可分;禁用示例是"Generating"。
  - 2026-10-21 demo 修正: "筛选" 之前用 `WithButtonExpanded` 永久展开、却没绑点击事件 —— 点击无效,
    浅底也一直挂着。现改成真实筛选菜单: 点击时 `SetExpanded(true)` + `Redraw(true)`(菜单是阻塞式原生弹窗,
    不立即重绘看不到展开色), 菜单关闭后 `SetExpanded(false)`;展开时的 bg-muted 是 shadcn `aria-expanded`
    的正常表现, 不应一直存在。
  - 2026-10-21 追加: **加载图标原来只是静态 lucide 图标**。新增 Button `WithButtonLoading()` / `SetLoading()`
    (行内前置位置绘制 1s/圈 linear 旋转的 loader;纯图标按钮替换图标;只负责视觉,禁用仍可转);运行时新增
    `xui.Spin(hOwner, active)`(复用每元素动画定时器,按墙钟算相位,`SetAnimationScale(0)` 可冻结截图)与
    `Painter.IconRotated`(旋转走 D2D 渲染变换,不重建几何;GDI 回落也旋转)。demo "Generating" 已改用。
    详见 DESIGN 第八节 87。
  - 2026-10-21 追加: **150% DPI 下按钮的文字/图标/壳体按下位移不同步**(用户主观感受"没一起下沉",
    sm 的 New Branch 一度正常)。60fps 逐帧质心实测: 壳体和文字可能移 2 物理像素、图标只移 1, 且
    **同一 DPI 换一个 y 位置, 文字也可能只移 1** —— 所以第一版"按 round(1×scale) 给图标补偿"是错的,
    会把原本正常的 New Branch 顶过头(用户录屏复现: 文字 +1、图标 +2)。
  - 最终修法: 按下时先用 `XDraw_ConvRect` 量**壳体自身**的实际物理位移 N(1 或 2), 然后让内容统一
    跟随 N: 文字用 `TextF` 的浮点矩形加 `N/scale` 逻辑偏移, 图标用 `Painter.IconRotatedOffset` 加
    N 物理偏移; 壳体保持原整数逻辑位移。四种组合(sm/default × 前导/尾随)逐帧验证壳体、文字、图标
    位移完全一致。详见 DESIGN 第八节 88。
  - demo 基础页新增"按钮组"卡片(横向/文本块/嵌套/纵向/尺寸档/行内图标/展开/胶囊/加载态),菜单触发钮改用
    `WithButtonHasPopup` + `SetExpanded`; `registry.json` 增加 `buttongroup`(依赖 button),README 组件表同步。
    按钮组行末的"…"也接了真实菜单(导出/归档/删除), 标记 `WithButtonHasPopup` —— 弹层触发区按下**不下沉**,
    打开期间只显示展开底色; 组内普通按钮仍照常下沉。
  - 验收: 探针断言组/文本块/分隔条矩形(嵌套组外框 120 = 56+8+56、纵向组 65 = 33+33−1、文本块 32 高、
    sm 文本组高 29(按钮元素高 29, 文本块压到 28));
    亮/暗主题整窗截图 + 4× 放大核对首尾圆角、无缝边框、文本块/分隔条对齐;hover/按压态截图确认组内按钮
    仍整体下沉、背景过渡正常;`go build` / `go vet` / `go test ./...` 通过。

- ✅ **Dialog 补齐文档能力 + AlertDialog**(2026-10-21, 参考 shadcn dialog / alert-dialog 文档):
  - **Dialog**: 标题/描述按面板宽折行(量高口径同 Alert, 见 DESIGN 90/91); 修正无页脚时漏算的下内边距;
    新增 `WithDialogScroll(maxHeight)`(内容区 ScrollArea, 头部/页脚固定, max 0 = 70vh)与 `BindClose`
    (任意控件绑成 DialogClose); 页脚按钮宽度改用 `xui.TextWidth`(旧的 `GetTextShowRect width=0 + CX*2`
    会把英文按钮挤成两行); 无关闭钮/自定义关闭钮均可配。
  - **`components/alertdialog.go`**: 对齐 alert-dialog.tsx(radix-nova) —— default/sm 两尺寸
    (default 桌面 384、sm 320; 头部 default 左对齐 / sm 居中)、media 40×40 图标块(破坏性配色)、
    sm 页脚等宽双列; 无右上角 X、点遮罩不关闭(radix onInteractOutside preventDefault)、ESC 关闭、
    打开时焦点强制 Cancel; `AddCancel` / `AddAction` 点击后关闭, 变体可经 `CancelButton()` 改。
  - **多层模态栈**: 同一窗口叠两层时只有栈顶响应 ESC/Tab/遮罩, z 序按打开顺序抬升, 关闭上层后焦点
    还给下层最近聚焦控件; 最小化还原时只有栈顶抢焦点。demo 反馈页加了「警示对话框」四例 +
    「对话框变体」(无关闭钮 / 嵌套警示 / 滚动内容), registry 新增 alertdialog(依赖 dialog)。
  - 验收: 探针亮/暗双主题逐例截图 —— default 左对齐 + 右对齐按钮、sm 居中 + 等宽双列、media 并排/堆叠、
    destructive 红底图标 + 默认 outline Cancel(有边框)、滚动内容头部/页脚固定且可滚、嵌套 Alert 在 Dialog 之上、
    ESC 第一次只关上层第二次关下层、关闭上层后焦点回到下层按钮、连点两次遮罩不关闭; 真实鼠标从触发钮打开时
    焦点落在 Cancel 且不显示焦点环; 对话框打开时滚轮被遮罩吞掉、页面不滚; `go build/vet/test` 与
    dialog/alertdialog/`add --all` 三种安装闭包编译全部通过。

- ✅ **修复: 对话框页脚带上沿出现深一档的条带**(2026-10-21, 用户截图):
  - 现象: 按钮上方一条 rgb(247,247,247) 横带, 与页脚其余部分(#fafafa)不同色。
  - 根因: 页脚带是**半透明**的 `bg-muted/50`; 旧画法先画整块圆角矩形, 再用方角矩形补上沿
    两个角, 交叠的 radius 高区域被合成两次。与第八节 89 的 Tooltip 箭头是同一类问题。
  - 修法: `onPaint` 改用 `Painter.FillRoundCorners` 一次画出"上平下圆"的页脚带; 实测截图沿
    页脚全高采样统一为 250, 条带消失。Dialog 与 AlertDialog 共用这块绘制, 两处同时修复。

- ✅ **Alert 提示条**(2026-10-21): `components/alert.go` 对齐 shadcn alert.tsx(radix-nova):
  图标(16, 下移 2)+ 标题/描述 + 右上动作槽;`AlertDefault` / `AlertDestructive`(标题/图标 destructive、
  描述 destructive/90);标题/描述随宽度自动折行, 高度按内容自撑, 动作槽可放 Button / Badge。
  折行高度不在 Layout 的 Auto 里自撑, 组件按"元素宽−12"量高、多行 +4, 改高后逐级重排祖先;
  ScrollArea 里要调 LayoutFrame 而不是它的内容区(否则 Alert 宽度被打成 0)。详见 DESIGN 第八节 90。
  Label 新增 `WrappedHeight(width)` 与 `SetColorFunc(fn)`(语义色实时解析)。demo 反馈页新增
  "提示条 / 破坏性·动作"两张卡片(仅标题、仅描述、长文本、动作、Badge);`registry.json` 增加
  `alert`(依赖 label), README 组件表同步。验收: 亮/暗、1200/960/1500 三种宽度, 长文本完整折行无省略号,
  动作贴右上; 运行时切换主题颜色实时跟随; `go test ./components` 通过(alert 纯计算/配色断言)。

- ✅ **Tooltip 淡入/淡出时"倒三角上方多一个正三角"**(2026-10-21, 用户逐帧录屏):
  - 现象: 气泡经过半透明阶段时, 下缘箭头上方浮出一个更深的三角鬼影; 不透明时几乎不可见。
  - 根因: `XEle_SetAlpha` 是**逐图元**混合, 不是整层合成。箭头画的是完整菱形, 再用气泡盖住内侧一半 ——
    半透明时重叠区被混了两次, 被盖住的半个菱形显形(实测 alpha≈0.3 时气泡 178、重叠区 120 灰)。
    这是旧画法固有的, 与窗口残留/擦除无关。
  - 修法(`components/tooltip.go`): 只画露在气泡外的三角形, 底边与气泡边线共线, 箭头与气泡填充零重叠;
    上下两种朝向都改。`SetAnimationScale(20~200)` 放慢到 alpha≈0.5 逐帧截图验证, 鬼影消失、接缝无暗线。
  - 详见 DESIGN 第八节 89。

- ✅ **Calendar / DatePicker 对齐 shadcn 文档能力(2026-10-21, 参 shadcn calendar / date-picker 文档页)**:
  - **Calendar 新能力**(`components/calendar.go`): **范围选择** `WithCalendarRange/SelectedRange/OnRangeSelect` + `SetRange/SelectedRange/RangeMode`(两次点击落区间, 早于起点自动交换, 完整后重开; 中段 bg-accent 色带 + 两端 primary, 起止同日只画一个实心格); **多月并排** `WithCalendarMonths(1~4)`(翻页钮在整块两端, 每月自己的年月下拉, 下拉只改被点的那个月); **ISO 周数列** `WithCalendarWeekNumbers`(取该行的周一算 ISOWeek, 周日开头的行 +1 天); **预订划线** `WithCalendarStrike` + `CalendarStrikeDates`(被划线格不再叠禁用的 50% 变淡 —— shadcn booked = disabled + opacity-100 的 line-through, 半透明线压半透明字实测不可见); **预设** `WithCalendarPresets(CalPreset(label,days,to))`; 页脚**自动换行**(行数多时高度跟着长); `CalendarDisableAfter` / `CalendarDisableBefore` 对称; 尺寸 helper `CalWidthFor(months)` / `CalWidthForWeekNumbers(months)` / `CalHeightForMonths(width,months)`; **宽度被布局拉伸时自动按新宽度重算高度并逐级重排祖先**(见 DESIGN 93)。单选 API 全兼容(hover 键位从 100 起, 页脚/下拉键挪到 1000+, 容纳 4 个月 ×42 格)。
  - **DatePicker 新能力**(`components/datepicker.go`): **范围模式** `WithDatePickerRange` + `WithDatePickerRangeValue` + `Range/SetRange/OnRangeChange`(弹层并排两个月; 起点未落终点时保持打开, 补上终点关闭; 触发区 "起点 - 终点"); **预设** `WithDatePickerPresets`; **年份范围** `WithDatePickerYearRange` / **禁用日期** `WithDatePickerDisabled`; **可输入形态** `WithDatePickerEditable` = InputGroup(文字区原生 Edit + 右侧 ghost 日历钮, Enter/失焦解析回写, ↓ 打开弹层; Edit 宽度随触发区 Size 事件同步, 见 DESIGN 93); `ParseDateText(s)` 解析 2026-10-22 / 2026年10月22日 / 今天/明天/后天/大后天 / N天后 / N周后。范围模式不支持可输入(自动退回普通触发区)。弹层高度含预设页脚的换行高度。
  - **修复: 弹层淡入停在半透明**(用户可见的"透字"弹层): `hoverTick` 里过渡**到位那一帧**不再重绘(active 已 false), 屏幕留在倒数第二帧, 等无关重绘才变实 —— 重绘条件改成 `active || len(applies)>0`(见 DESIGN 92)。60fps 抓帧验证: 弹层出现后 ≤2 帧到全亮、无半透明滞留。
  - **demo**: 表单页第二行新增 4 张卡(日期范围框+预设 / 可输入日期框 / 范围月历(禁用未来) / 周数+预订划线+预设); `registry.json` 无新文件(datepicker 本就带 calendar.go/popup.go)。
  - **不适用**(shadcn 文档里有但 Go/xcgui 侧做不了或无意义): RTL(dir=rtl)、波斯/Jalali 历、timeZone、自然语言解析(chrono-node 无等价物; 常用相对日期已被 `ParseDateText` 覆盖)。已记 DESIGN 第七节。
  - 验收: 亮/暗双主题整卡 + 双月弹层/预设/周数/删除线逐项截图; 交互实测 —— 范围两次点击(第一次保持打开、第二次关闭)、预设"未来 30 天"回写、可输入键入 2026-10-22 回车回写、图标钮打开弹层(年份 1900 起)、ESC/点外部关闭; `go build/vet/test ./...` 通过; calendar/datepicker 纯逻辑单测(范围状态机/ISO 周数/几何基准/ParseDateText/Text)在 `components/calendar_test.go`。
  - **2026-10-21 用户报障两连修 + 一条改判**: ①**点年份无效、点月份弹出年份** —— `headerHit` 的编码(块*2 + 0 年/1 月)被直接当 `calDrop*` 用, `calDropNone=0` 恰好是"年"的编码 → 点年份变成关面板、点月份落到 `calDropYear`; 新增 `calHeadKind(head)` 换算(DESIGN 语义表同条)。②**双月范围出现 3 个"选中"的 10** —— 相邻块的 outside 区按日期判定也会画端点/色带(9 月块拖尾 = 10 月上旬); 改为选中/范围标识只画在本月格上, outside 重复格只画弱化文字(DESIGN 多月并排条目)。③**跨多月范围重开停在起点** —— 曾按报障改成"终点对齐", 后确认上游 `defaultMonth={from}` 本就只认起点(区间比窗口长时终点在窗外、翻页可看), 属预期行为, 已按用户意见改回起点对齐(DESIGN 范围条目注明)。

- ✅ **Card 页脚补齐 + AspectRatio 新组件**(2026-10-22, 参 shadcn card / aspect-ratio 文档):
  - **CardFooter**(`components/card.go`): 从纯绘制元素改成横向布局容器 —— 内层 `Content()` 承担四边内边距(default 16 / sm 12)、子项垂直居中(`items-center`)、`cy=0` 时高度按内容自撑; 新增 `Card.Footer(cy)` 自动匹配卡片尺寸、`Card.TitleSize()`(16 / sm 14) 与 `ContentStack(space)`。demo 的「卡片」页脚改用新形态(文字垂直居中), 并新增一张 sm 卡片验证 12px 内边距与 14px 标题。
  - **`ui.AspectRatio`**(`components/aspectratio.go`): 宽度驱动高度(高 = 宽 / ratio); 尺寸事件只排队, 下一轮把所有待处理比例框的高度/内容尺寸**批量**写回, 再统一框架重排 + 整窗重绘(Auto 高卡片 / ScrollArea 会跟着长); `SetRatio` / `Ratio` / `Content()` / `ContentStack(space)`; `cy>0` 或布局项高度被 Fill/Weight/Fixed 接管时比例不生效(同 CSS 宽高都确定时 aspect-ratio 失效)。单测 `aspectratio_test.go`(几何换算 / 非法比例)。
  - **未做**: Card 的 `overflow-hidden` 与首末图片自动圆角 —— xcgui 没有元素级圆角裁剪 API, 已记 DESIGN 第七节; 需要图片横幅时用 AspectRatio 作卡片的第一个子项、内容自绘圆角。
  - registry 登记 `aspectratio`; demo 基础页新增「比例容器」(16:9 / 1:1 / 9:16)与 sm 卡片; README / DESIGN(技术路线、语义表、映射表、94/95 条)已更新。
  - 验收: `go build/vet/test ./...`; 探针 800→600 宽 + `SetRatio(1)` 几何断言 **784×441 → 584×329 → 584×584**, 内容区与比例框逐次等大; 亮/暗主题截图核对页脚内边距/居中与三种比例, 960×660 最小窗口下三块比例仍完整(Weight 宽会重新分配); 逐组件安装编译见下。
  - **2026-10-22 用户报障: 页脚上方一条 #f7f7f7 色带** —— 页脚"下两角圆角"旧画法是 `FillRound` 整块 + 方角块补上沿两角, 半透明 `bg-muted/50` 在补角区被合成两次(与对话框页脚带同因, DESIGN 第八节 89 同源坑条已补注); 改成 `Painter.FillRoundCorners` 一次画"上平下圆"。验收: 亮/暗主题扫描页脚列, border-t 往下到卡片底边全程单一色值(亮 250 / 暗 31), 无深一档条带。

- ✅ **demo 导航页拆分**(2026-10-22, 部分页内容过长):
  - **基础 → 基础1 / 基础2**: 基础1 = 按钮 / 徽章·分隔线·标签 / 按钮组; 基础2 = 卡片 / 比例容器 / 滚动容器。
  - **表单 → 表单1 / 表单2**: 表单1 = 输入 / 选择控件 / 滑动条 / 下拉框; 表单2 = 日期与时间 / 月历 / 日期范围 / 可输入日期 / 范围月历 / 周数与预订。
  - 页签 **8 个、总宽 640**(每个 80); `XUI_HOME_PAGE` 映射改为 **0..7 = 基础1/基础2/表单1/表单2/导航/反馈/数据/图标**; 页面构建函数拆成 `buildBase1Page/buildBase2Page/buildForm1Page/buildForm2Page`, 页面序号与注释同步。
  - **拆分后暴露并修复的 AspectRatio 问题**: ①隐藏页首次显示时卡片重叠(9:16 比例框画到下一张卡片上) —— 根因是重排后没有重绘被推走的兄弟; 现在批量写高后统一 `XWnd_Redraw`。②首次进入时比例块**逐帧展开**(先 16:9/1:1、再变高、最后 9:16 弹出) —— 根因是每块比例框各自跑一轮延迟布局; 改成**全局批量**(先写完所有高度, 再统一框架重排/整窗重绘), PrintWindow 逐帧抓页签切换首帧即最终布局。③中间一版把 `SetHeight` 放进尺寸事件里导致宽度 335↔0 布局震荡、**点基础2直接卡死**; 最终版不在布局趟里改尺寸, 只调最近的 ScrollArea 框架, 尺寸未变不重排, 切换后 CPU 稳定。见 DESIGN 94。验收: 基础2/表单2 首帧不再重叠/展开, 8 个页签逐页切换正常, 960×660 下比例块仍按宽重算。
  - **隐藏页首次显示后命中区域未刷新**(用户报"点日期框/时间框没反应, 滚一下鼠标滚轮才行"): ScrollView 的命中映射停在旧滚动位置, 画面却是对的; demo 切页显示后延迟 32ms 对页面调一次 `AdjustLayout` + `Redraw` 即可恢复。时间框也从 HStack 挪成卡片内容区的直接子项(嵌套水平容器同样有命中矩形滞后风险)。中途加过一个 `ScrollArea.RefreshHitTest()`(滚动位置往返), 实测页面级重排已够, 不需要额外 API, 已删。见 DESIGN 66。验收: 切到表单2 后不滚轮直接点日期框/时间框, 弹层均能打开。
  - **切页后 Toast 跑到左下角**(用户报): 页签 `show` 里为刷新命中区调了 `XWnd_AdjustLayout`, 窗口级布局把绝对定位的 Toast 卡片当布局项排到左侧; 已移除, 只保留页面级 `AdjustLayout` + `XWnd_Redraw`。见 DESIGN 96。验收: 反馈页右下角弹通知, 切到基础1后仍在右下角。

- ✅ **修复: ComboBox 快速连续选择后"点触发器没反应"**(2026-10-22, 用户报):
  - **现象**: 反复点开下拉框并选不同项, 若干轮之后点触发器不再弹出下拉列表。
  - **根因**: 弹层的 100ms 进场淡入用**整窗 alpha**(弹层窗口被转成 `Window_Transparent_Shaped`)。异型窗的整窗 alpha **逐帧改会概率性冻结**(停在第 2 帧前后的 3/255), 而淡入的 apply 日志显示 255 已经写进去了 —— 弹层其实开着, 只是完全透明, 又正好 item-aligned 盖住触发区, 于是后续点击全被这个透明窗吃掉。与 DESIGN 57 是同一条规律, 69 条早期"每帧生效"的结论是孤立探针得出的误判。
  - **修法**: 删掉 ComboBox 弹层的进场淡入(`fadeInComboPopup`), 弹出回调只做上色与定位(转类型也不再需要)。上游是 `fade-in-0 duration-100`, 属可牺牲的观感; 元素 alpha 路线不可行(弹层白底是窗口自绘, 不是某个元素)。
  - **排查手法(值得复用)**: 用 `EnumWindows`(空标题 + 尺寸阈值)判"弹层在不在", 用 `GetLayeredWindowAttributes` 判"看不看得见", 两者一合就能区分"没弹出"和"弹出了但透明"; 组件侧再挂 `AddEvent_Combobox_Popup_List/Exit_List/Select` 日志对时间戳。合成点击要用**客户端坐标**(窗口枚举给的是屏幕坐标, 差一个 `ClientToScreen` 原点, 混用会把"点不中项"误判成组件 bug)。
  - **验收**: 40 轮复现脚本在修复前 3/40 轮 alpha=3, 修复后 **70 轮"开→立刻选不同项"全部正常**(每轮都读到弹出窗口、无低 alpha、无残留窗); 弹层样式(item-aligned、ring、勾、圆角)截图与修复前一致。见 DESIGN 30/69。

- ✅ **Collapsible 可折叠面板(2026-10-23, 参 shadcn collapsible 文档)**:
  - `components/collapsible.go`: 根容器(竖向流)+ `Content()` / `ContentStack()` + `BindTrigger(*Button)`; `Open/SetOpen/Toggle/OnOpenChange/SetDisabled`; 展开态同步触发按钮的 aria-expanded 视觉(`Button.SetExpanded`), 禁用时触发按钮一起禁用; `WithCollapsibleOpen/Disabled/Duration`。
  - 内容收起时隐藏且不占布局; 展开/收起 200ms 高度过渡走 `xui.Presence`: 展开先 `Auto` 量高 → `Fixed 0` 起步逐帧长高, 每帧从根向上 `AdjustLayout` 并整窗立即重绘, 到位后恢复 `Auto`; 收起动画结束才 `Show(false)`。探针确认 xcgui 会按父矩形裁剪子元素(新增 DESIGN 97), 动画不需要额外遮罩。
  - Button 新增 `WithButtonAlign(ButtonAlignStart/Between)` + `SetAlign`: 整宽触发区文字靠左、尾随图标贴右(对齐 shadcn 示例里的 justify-start/justify-between)。
  - demo 导航页新增「可折叠面板」卡: 订单详情(图标触发区)+ 嵌套文件树; 本地 shadcn `App.tsx` 同步对照卡 + `src/components/ui/collapsible.tsx`(registry 原样); 新增 `XUI_ANIM_SCALE=5` 把动画慢放 5 倍, 便于抓中间帧。
  - **验收**: 订单(图标按钮)/ 文件树(整宽文字触发 + 嵌套)的展开、收起、慢放中间帧逐帧截图; 中间帧内容被内容区矩形裁切, 卡片高度与页面滚动范围同步长高/回缩; 状态行文字完整。`go test ./...`、`go vet`、demo build、本地 shadcn `pnpm build` 通过; registry 加 collapsible 后逐组件安装(`xui add collapsible` → `go build ./ui`)与 `add --all` 均编译通过。

- ✅ **修复: 可滚动对话框"有时候打开就显示滚动条"**(2026-10-23, 用户截图):
  - **现象**: 可滚动对话框打开时, 滚动条有时候一开始就是亮的; 鼠标不动就一直亮, 移出去才淡出。
  - **根因**: 对话框进出场用 `SetSubtreeAlpha` 给整棵子树逐个设 alpha, 把滚动条也一起设成了 1 —— 但滚动条的 alpha 是 `showScrollBarOnHover` 的悬停显隐状态(初始 0, 移入宿主才 1), 被整组淡入覆盖后就常亮了。
  - **修法**: 运行时新增 `xui.SetSubtreeAlphaSkipScrollBars`(遍历跳过 `XC_SCROLLBAR` 子树); 对话框**打开**的初始归零/淡入/最小化还原改用它, **关闭**淡出仍用普通 `SetSubtreeAlpha`(滚动条跟着淡出); `showScrollBarOnHover` 在宿主 `XE_SHOW(false)` 时归零并 `DropTransitions`, `Dialog.Show` 主动 `resetScrollBarHover` 覆盖淡出中途快速重开。见 DESIGN 98。
  - **验收**: 打开时鼠标移开 → 滚动条隐藏; 移入内容区 → 淡入; 滚动条亮着时关闭再打开 → 仍从隐藏开始; 页面切换/最小化恢复不残留亮条。

## 已知限制(不是待办, 是 xcgui 的能力边界)


- ℹ️ Input 的文字/光标/选区/IME 由原生 Edit 绘制(只开放字体/字色/占位符色/插入符色/内边距, 见 DESIGN 三 第 4 条 / 23 / 61)。
- ℹ️ Toast 只支持四角堆叠(无顶部/底部居中、无 richColors, 见 DESIGN 75)。
- 完整清单见 DESIGN 第七节。

## 待办

- ⬜ **P4(发布后, 非阻塞)**: 文档站、主题画廊(把 shadcn 社区主题批量转译成 Go 令牌)
- ⬜ **shadcn 组件补齐**(对照 https://www.shadcn.com.cn/docs/components 的 59 项: 39 项已有对应, 4 项部分覆盖, 16 项未实现)。每项落地都要 `registry.json` 登记 + demo 展示 + 逐组件安装编译回归; 发现 xcgui 未文档化行为再补 DESIGN 第八节。
  - **已有等价实现(不新增)**: Radio Group = `NewRadio`(原生父容器互斥)、Textarea = `Input` + `WithMultiline`、Sonner/Toast = `NewToaster`、Dropdown Menu / Context Menu = `NewMenu` / `NewContextMenu`。
  - **部分覆盖待补**: Input Group(现为 Input 内嵌简化形态, 无复合 addon)、Popover(`popoverPanel` 仅内部使用)、Data Table(只有 Table 原语, 无排序/过滤/分页模式)。
  - **批次 1 — 纯组合/绘制**: 已完成(2026-10-24): Empty / Field / Item / Breadcrumb / Kbd / Skeleton / Spinner。
  - **批次 2 — 表单闭环**: 已完成(2026-10-24): Toggle / Toggle Group / Native Select / Select / Input OTP / Accordion(复用 Collapsible)。
  - **批次 3 — 弹层基座**: 已完成(2026-10-25): Popover / Hover Card / Avatar / Command。
  - **批次 4 — 窗格/菜单**: Sheet → Drawer、Menubar(★★★)。
  - **大件延后**: Data Table(按 Table 的排序/过滤/分页 demo 模式做, 不新增控件)、Carousel、Resizable、Navigation Menu、Chart、Sidebar(★★★★~★★★★★)。
  - **不做**: Direction(xcgui 无 RTL)、Typography(文档级样式, 低价值)。
  - **前置(部分完成)**: 本地 shadcn 参考实现已有 20 个旧组件 + collapsible; 批次 2 落地时已随源码补齐 native-select / input-otp / accordion 的 tsx 与 App.tsx 对照卡 —— **后续批次实施前同样先补上游源码**(AGENTS 硬规则: 先读源码再改交互语义)。

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color / theme / themes_builtin / manager
│                                 draw(Painter) / icons(图标几何+D2D 直通出图) / focus / font / state
│                                 anim(hover 过渡) / presence(进出场) / container·layout(布局骨架)
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│                                 button checkbox radio input label separator(含 Badge) card alert
│                                 switch progress slider tabs collapsible tooltip toast window combobox dialog alertdialog
│                                 menu table tree pagination scrollarea
│                                 calendar datepicker timepicker(弹层骨架 popup.go)
│                                 empty field item breadcrumb kbd skeleton spinner toggle
│                                 native_select select inputotp accordion
                                 popover hovercard avatar command
├── icons\                        图标资源子包: lucide 预处理数据(2130 个) + 每图标一个
│                                 IconXxx 常量 + Path/Has/Names/Count/Version
├── scripts\gen_icons.py          lucide-static SVG → icons\lucide_gen.go(名字表)
│                                 + icons\lucide_icons_*.go(图标常量, 每 400 个一个文件)
├── cmd\xui\                      CLI(init / add / add --all / update / list; 数据源默认
│                                 远程 main 分支, XUI_REGISTRY 可覆盖, 失败降级内嵌)
├── icons_test.go                 解析器语法 + 资源完整性测试(go test ./...)
├── demo\                         演示应用(也是全组件编译测试目标)
├── registry.json                 组件清单(files + deps)
├── docs\DESIGN.md                知识库: 架构设计 + xcgui 踩坑全集(第八节必读)
├── docs\HANDOFF.md               本文
├── README.md                     用法
└── AGENTS.md                     agent 会话自动加载的入口规则

D:\GOProject\src\ui\shadcn\       ← 本地部署的 shadcn-ui 参考实现(语义权威)
├── src\components\ui\*.tsx       组件语义的唯一来源(hover/focus-visible/disabled 等)
├── src\index.css                 主题令牌(:root/.dark, nova 预设 neutral 色板)
└── src\App.tsx                   与 demo 同构的对照页
```

## 常用命令

```bash
# xui demo(全部组件的编译+视觉验证目标)
cd D:\GOProject\src\ui\xui
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe

# shadcn 参考实现 dev server(可能已在运行: netstat -ano | findstr 5173)
cd D:\GOProject\src\ui\shadcn && pnpm dev    # http://localhost:5173

# CLI 本地测试(默认源:直连→Gitee 镜像→GitCode 镜像→代理, 全不通才降级内嵌源码)
go run ./cmd/xui list / init <目录> / add <组件...> / add --all / update [组件...]
go test ./cmd/xui    # 候选顺序 / 数据源解析 / httptest 端到端 / 显式源失败退出
# 想验远程链路(无需 GitHub): 仓库根起 python -m http.server 8123,
# 再 XUI_REGISTRY=http://127.0.0.1:8123 go run ./cmd/xui add dialog
# 数据源也可指本地目录(完全离线): XUI_REGISTRY=D:\GOProject\src\ui\xui
# 依赖回归(每加组件/跨文件助手必跑): 逐组件单独装到临时项目再 go build ./ui(步骤见下面「依赖回归」小节)

# 暗色对照验收: 环境变量决定启动主题, 不用改代码
#     XUI_THEME=dark  启动 demo 即暗色(cmd: set XUI_THEME=dark && demo\demo.exe)
#     运行中仍可用右上角按钮切换(参考页 shadcn 同样支持按 d 键切换)
# 直接启动到指定页签(验收某一页时省得点): XUI_HOME_PAGE=0..13 = 基础1/基础2/基础3/基础4/表单1/表单2/表单3/表单4/导航1/导航2/反馈1/反馈2/数据/图标
#     XUI_ANIM_SCALE=5 把 hover / 弹层 / Collapsible 的过渡慢放 5 倍(抓中间帧用)

# 全部测试(图标解析器语法 + 资源完整性 + 缓动曲线 + CLI 选路/数据源/端到端)
go test ./...
```

### 依赖回归(逐组件安装 + 编译)

**每加组件/跨文件助手就跑一遍「逐组件单独安装 + 编译」**——这不是发布前检查, 而是每次改动后的固定流程。临时项目**必须先有 go.mod**, 否则 `go run` 起不来。

```bash
# 1. 构建本地 CLI, 并让临时项目指向本仓库(离线验收; 联网时可改用 go mod tidy)
go build -o "$TEMP/xui-cli.exe" ./cmd/xui
rm -rf "$TEMP/xui-check" && mkdir -p "$TEMP/xui-check" && cd "$TEMP/xui-check"
go mod init check
go mod edit -require=github.com/twgh/xui@v0.0.0
go mod edit -replace=github.com/twgh/xui=D:/GOProject/src/ui/xui
go mod edit -require=github.com/twgh/xcgui@v1.4.3
go mod edit -replace=github.com/twgh/xcgui=D:/GOProject/src/github.com/twgh/xcgui

# 2. 单组件闭包 + 全量(改 registry 时两者都跑)
XUI_REGISTRY=D:/GOProject/src/ui/xui "$TEMP/xui-cli.exe" add alert -p ui
go build ./ui
rm -rf ui
XUI_REGISTRY=D:/GOProject/src/ui/xui "$TEMP/xui-cli.exe" add --all -p ui
go build ./ui
```

- 本地 xcgui 工作副本路径按本机调整; 没接工作副本时删掉 xcgui 的 replace, 走模块代理。
- 重跑前清空 `ui/`(或换临时目录), 避免上次拷贝的文件掩盖依赖漏登记(如跨文件助手)。

### CLI registry 维护注意

- **默认源顺序与单文件换源由 `cmd/xui/main_test.go` 固化**:直连 → Gitee → GitCode(v5 API, JSON+base64) → ghfast → gh-proxy;改选路先改测试。
- 显式 `XUI_REGISTRY` **一律原样使用**(不套镜像、失败直接报错);只有未设置且全部候选不通时才降级 `moduleRoot()` 内嵌源 —— 别动 `moduleRoot`,它是离线回退的根。
- `registry.json` 的 `files` 是 `components/` 下的**扁平文件名**;`theme.go` 永不进清单;用户的 ui 目录名是 `init`/`-p` 参数,不要硬编码 `ui`。
- CLI 本体按 Go 模块版本锁(`go run ...@vX.Y.Z`),组件跟 `main`;`@latest` 可能被模块代理缓存拖旧,组件更新不受影响。

## 关键设计决策(一句话版, 详见 DESIGN.md)

1. **copy-not-install**:组件源码由 CLI 拷进用户 `ui/` 目录,归用户;运行时(theme/icons/helpers)走 go.mod
2. **绘制时取令牌**:组件永不缓存颜色,paint 回调里 `xui.Current()` 现取 → 切主题 = 换指针 + Redraw
3. **行为/视觉分层**:xcgui 原生控件 = 无头行为层,`AddEvent_Paint` + `drawx` 只管视觉
4. **交互语义逐条对齐 shadcn 源码**:改交互前先在 DESIGN 第四节末的语义汇总表确认,再读 `shadcn/src/components/ui/*.tsx`
5. **组件文件写薄**:绘制细节下沉运行时助手,降低拷贝模式下的升级冲突面
6. **界面用流式布局表达,不写绝对坐标**:窗口是唯一"天然确定尺寸"的父级,竖向主链从窗口开始一路 `Weight`(每级确定尺寸,下一级的 `Weight` 才算得出剩余空间);卡片高度用 `Auto` 自撑,要长高的面板整条链全 `Weight`。**别在中间垫 `Fill` 容器,也别用 `Fill` 当"剩余空间"**(尺寸语义见 DESIGN 第八节 55,样板在 `demo/main.go` 的 `row`/`colStretch`/`autoCard`)。页面本身是 `ui.ScrollArea` 滚动容器: 自撑型页面(内容 Auto 高,行高也必须 Auto)窗口小了就滚, 铺满型页面(数据页 `SetContentFill`)整链 Weight、面板跟随窗口长高(见第八节 66)

## 文档导航(去哪查什么)

| 想了解 | 去哪 |
|---|---|
| 当前进度 / 待办 / 命令 / 仓库结构 / 验证工作流 | **本文** |
| 架构原理、分层边界、主题令牌、组件解剖 | DESIGN.md 一~四节 |
| **交互语义(改交互前必看)** | DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」+ 本地 shadcn 源码 |
| **我们与 shadcn 的差异 / 变体尺寸映射** | DESIGN.md 第四节末「变体/尺寸与上游的映射、已知差异」 |
| 某个 xcgui 行为 / 为什么这么写(踩坑) | DESIGN.md 第八节 |
| 界面布局怎么写(容器/尺寸/随窗口自适应/滚动) | DESIGN.md 第八节 55(尺寸语义) + 56(Card) + 66(滚动容器), 样板在 `demo/main.go` |
| 用到的 xcgui API 清单 | DESIGN.md 第九节 |
| agent 会话硬规则 | AGENTS.md |
| 面向用户的安装与用法 | README.md |

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
- 布局类问题**用 `xc.XEle_GetRectLogic` 打印各元素矩形**判定, 比截图量像素快且准(探针里 `AdjustLayout()` 之后读;注意窗口必须 `Show` 过才有非零尺寸, 见 DESIGN 8.55)
- 要按**物理/逻辑像素精确量测**时, 自己用 Win32 抓窗口(GetWindowRect + CopyFromScreen)比 MCP 截图可靠: MCP 截图会被缩放到模型尺寸。**抓之前必须把抓图进程设为 per-monitor DPI-aware**(`SetProcessDpiAwarenessContext(-4)`), 否则 GetWindowRect 给虚拟化坐标、CopyFromScreen 抓物理像素, 两者差 150% —— 抓出来的图左边会多出一条桌面(踩过)
- 需要"真实鼠标"验证 hover/点击时, 探针里用 `wapi.SetCursorPos` 比外部脚本可靠;从 PowerShell 移动鼠标要注意它是 DPI-unaware 进程, 坐标会被虚拟化/偏移, 别拿截图坐标直接当屏幕坐标用(踩过坑)。脚本里出现中文时**必须存成 UTF-8 **带 BOM**** —— PowerShell 5.1 按 ANSI 读 `.ps1`, 无 BOM 的中文串会把语法读坏(报一堆莫名其妙的 "Unexpected token")
- ⚠️ **定位窗口一律用"枚举进程顶层窗口 + 取可见且面积最大"**, 不要用 `Process.MainWindowHandle`: 弹层/提示这类小窗(带 `WS_EX_TOOLWINDOW`)一旦创建, 它就会返回**小窗**, 之后用它算出来的坐标、抓出来的图全是错的(踩过: 整个复现流程因此偏了半天)。诊断脚本见 DESIGN 第九节的 Win32 诊断 API 一行
- **"窗口可见 ≠ 屏幕上有东西"**: 判断弹层/提示这类异型透明窗有没有真的显示, 不能只看 `IsWindowVisible`/矩形 —— 用 `PrintWindow` 抓它自己的内容(证明"画了")与 `CopyFromScreen` 抓同一块屏幕(证明"屏幕上有没有")对照, 再读 `GetLayeredWindowAttributes` 的 alpha;`可见 + 矩形正确 + 像素为空` ⇒ 整窗 alpha/形状问题(见 57)
- **验收探针必须跑 `a.Run()` 消息循环**(无泵时 Presence/定时器全不工作, 结论作废)
- 浏览器对照:shadcn 参考页 + xui demo 同屏对比;同一张截图里量两边(ours vs ref)是最可靠的对照法
- **弹层类组件的暗色验收一律 zoom/像素测量**:全屏缩略截图里暗色弹层会假显亮底(36)

## 环境 / 注意事项

- shadcn dev server 由会话后台启动,机器重启后需按上文命令重启
- hover 过渡用运行时 `anim.go` 的炫彩定时器(不是 ani 包), 时长可用 `xui.SetHoverDuration` 覆盖(传 0 = 关掉过渡);**新增 hover 反馈只用 `xui.Hover`/`HoverItem` + `Fade`, 别自己起定时器**
- 用户项目须 `app.New(true)`(D2D)+ `EnableAutoDPI`,demo 是标准样板
- 弹层/对话框过渡挂**宿主窗口**,别挂弹层小窗;要 Win32 HWND 用窗口 wrapper 的 `GetHWND()`,别拿 xcgui 窗口句柄喂 `XWidget_GetHWND`(弹模态错误框)

## 图标系统(速览)

- `icons/` 子包内置 **2130 个** lucide 图标(lucide-static **1.52.0**), **每图标一个 `IconXxx` 常量**;按名字取用 `icons.Path("eye")`(给"名字来自用户输入/配置"的场合)。数据由 `scripts/gen_icons.py` 生成(默认取最新版, `--version` 可钉版)
- 出图走**直通 D2D 几何填充**(`d2dicon.go`), GDI 模式(`app.New(false)`)自动回落折线描边
- 体积(实测, 同基准 `-s -w`): 只用常量 **+0 字节**;调 `Path/Names` 引用整张表 **+426KB**
- demo 第 6 页「图标」是浏览入口(`XUI_HOME_PAGE=5`):9×4 格 + 搜索 + 翻页, 悬浮出 Tooltip 显示图标名、点击复制
- **原理、坑、性能结论见 DESIGN 第八节 43(抗锯齿 / figure 数)与 44(资源预处理 / 名字单射 / arc 标志位)**
