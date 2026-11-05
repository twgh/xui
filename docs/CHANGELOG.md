# CHANGELOG — 已完成记录

> 按时间记录的完成 / 修复条目, 保留现象、根因、修法与验收证据(新条目加在最上面)。
> 当前状态 / 待办 / 命令 / 仓库结构 / 验证与发布流程在 [HANDOFF.md](HANDOFF.md); 设计原理与踩坑索引在 [DESIGN.md](DESIGN.md), 踩坑完整条目在 [DESIGN-PITFALLS.md](DESIGN-PITFALLS.md)。

**已完成(括号里是根因/踩坑所在的 DESIGN 第八节条目号)**:

- ✅ **v0.10.0 发布(2026-11-05)**: 新增 status-bar 状态栏, 并修掉 Tooltip 的跟随 / 裁剪命中 / 滚动收起与抓手拖动气泡残留, 走 minor(组件变更)。
  - **顺序**: 组件与文档提交(`ca55bd7`)→ 版本示例 + `fallbackXuiVersion` 提交(`d9bc9e6`)→ 轻量 tag `v0.10.0` → `runtime` 抬到 v0.10.0(`a19c89d`, tag 之后)。
  - **推送**: `git push origin main v0.10.0` 四个远端全部成功(GitHub / Gitee / GitCode / cnb.cool)。
  - **Release**: https://github.com/twgh/xui/releases/tag/v0.10.0 , 名称 `v0.10.0 — StatusBar 状态栏 / Tooltip 跟随与命中修复`; 本机 `gh` token 失效, 按 HANDOFF 用 `git credential fill` 取 token 后 `Invoke-RestMethod` 创建(name/body 用 `[IO.File]::ReadAllText(..., UTF8)` 读)。
  - **下限回归**: 临时项目 `go mod edit -require=github.com/twgh/xui@v0.10.0` → 本地 registry `add --all` → `go mod tidy && go build -mod=mod ./ui` 通过(默认代理已收录新 tag, 不用 `GOPROXY=direct`)。

- ✅ **调整 `status-bar` 尺寸抓手: 贴到窗口右下角, 并缩成 16×16 方块**(2026-11-05, 用户报障):
  - **现象**: 右下角那三条斜线离窗口角还有一截(实测离右边 9 逻辑像素、离底 6), 得往右挪才"对上角"; 抓手跟整条一样高(28), hover 高亮在右端竖出一道高高的色块。
  - **根因**: 抓手是条里的最后一个布局项, 条的右内边距 8 把它推离窗口边; 元素高度又直接用了整条高。
  - **修法**: 抓手外面套一个整条高的 `LayoutEle` 容器并 `SetAlignV(Layout_Align_Bottom)`, 抓手本体改成 16×16 贴底; 条的右侧内边距在抓手启用时收到 0(`gripPadRight`), `SetGrip` 切换时同步改内边距并连容器一起显隐(只隐抓手会让中槽一直偏半个抓手宽)。抓手不再顶到条的上沿, 也就去掉了原来补画顶边那一步。
  - **验收**: 探针读回条 `(8,0)-(392,28)`、抓手 `(376,12)-(392,28)` = 16×16 且右边/底边与条严丝合缝; 150% DPI 截图里三条斜线离窗口角约 3 逻辑像素。`go build ./...` / `go vet ./...` / `go test ./...` 全绿。

- ✅ **修复 `Tooltip`: 弹出后滚动页面, 气泡不收起**(2026-11-05, 用户报障, 见 DESIGN 第八节 190):
  - **现象**: 悬停控件弹出提示后直接滚轮翻页, 气泡留在原地, 而它描述的控件已经被滚走。
  - **根因**: 内容位移不产生鼠标事件 —— 滚动期间没有 `MouseMove`, 元素级 `MouseLeave` 与窗口级 `hitTest` 都不触发, "按事件隐藏"的链路整体失效。
  - **修法**: 宿主窗口(`watchHost`)挂一次 `AddEvent_MouseWheel`, 滚轮一到就收起当前气泡并取消排队中的延迟显示; 与浏览器 tooltip 的"滚动即消失"一致。
  - **验收**: 150% DPI 亮色, 表单2 页用 `Tooltip.Show` 程序化弹出"切换运行时环境"(截图确认在屏), 在页面上滚 5 格后再截图 —— 气泡消失, 页面区域像素对比 2169 点变化证明确实滚动了。`go build ./...` / `go vet ./...` / `go test ./...` 全绿。

- ✅ **修复 `Tooltip`: 滚出视口的控件会在视口外弹提示**(2026-11-05, 用户报障, 见 DESIGN 第八节 189):
  - **现象**: 表单2 页滚到末尾再往上滚两下, 光标放到标签栏 / 标题栏一带, 直接弹出页面顶部那个默认尺寸 ComboBox 的提示("切换运行时环境"), 光标下面并没有该控件。
  - **根因**: 窗口级命中测试(`hitTest`)只判"点在元素矩形内 + 可见", 而 `XEle_GetWndClientRect` 给的是**未裁剪**矩形 —— 滚出 ScrollArea 视口的控件坐标仍在内容位置上(正好落在标签栏那一带);滚动不影响 `XWidget_IsShow`, 原有的可见性判断挡不住。
  - **修法**: 命中后沿父元素链补一次裁剪判定 `clippedByAncestor`(点必须同时落在每个有尺寸祖先的矩形内), 视口外的部分不再参与命中, 元素只在自己露出来的区域响应。
  - **验收**: 探针(视口 (20,90)-(580,210) + 跨上沿的子元素 (40,50)-(240,130)): 点 (45,55) 判为裁剪、(45,95) 正常弹提示;注释掉该步则 (45,55) 会弹提示(复现报障)。`go build ./...` / `go vet ./...` / `go test ./...` 全绿。

- ✅ **修复 `status-bar` 抓手: 拖完窗口大小后提示气泡不消失**(2026-11-05, 用户报障, 见 DESIGN 第八节 188):
  - **现象**: 按住右下抓手拖动松手后, "拖动调整窗口大小"气泡停在拖动开始的位置一直挂着, 鼠标移出窗口也不再消失。
  - **根因**: 抓手 `SetCapture` 之后持续收到 `MouseMove/MouseStay`, 拖动中提示被反复 `request`; `Tooltip.request` 对"已显示在同一个元素上"直接返回、不重新落位, 抓手随窗口缩放移动而气泡停在旧坐标;收手时既没有 `MouseLeave`, 光标又常在缩小后的窗口之外(没有窗口级 `hitTest`), 于是永久残留。
  - **修法**: ① `Tooltip` 新增元素级临时压制 `suppress(hEle, on)`(压制期间 `request` 直接返回并立即 `Hide` 正在显示的那条, 元素销毁时清登记);抓手 `onDown` 起手压制、`endDrag` 解除压制并补 `HideTooltip`。② `request` 对"已显示在同一元素上"不再直接返回, 而是比对落位时记录的元素屏幕矩形, 元素动过就重新落位(只挪气泡窗 + 重画, 不重播进场) —— 拖窗口/窗口缩放/布局重排/滚动时提示都会跟着目标走, 不再僵在旧坐标。静止悬停只多两次矩形查询, 不做重量。
  - **验收**: `go build ./...` / `go vet ./...` / `go test ./...` 全绿;150% DPI 亮色实测: 抓手按住拖动(窗口 1620 → 1542 物理宽)松手后尺寸读数同步刷新、屏幕上无气泡残留。

- ✅ **新增 `status-bar`: 窗口底部状态栏**(2026-11-05, HANDOFF 待办; 上游 shadcn 无 tsx, 交互语义由本库定死并写进 DESIGN 第四节末):
  - **组件**: `components/statusbar.go` 单文件(registry key `status-bar`, deps `tooltip`)。整条 = `LayoutEle`(宽 Fill / 高 Fixed 28, 可 `WithStatusBarHeight`/构造 cy 覆盖), 左/中/右三槽各返回 `LayoutEle` 可放任意控件; `NewStatusBarItem` 自绘条目(图标 12 + text-xs + hover 150ms 过渡 + 按下 + 禁用 + active), `NewStatusBarSeparator` 画槽内 1px 竖线, `NewStatusBarGrip` 形态的右下 16 宽 NWSE 抓手 + 可选 `W × H` 读数。选项 `WithStatusBarGrip/SizeReadout/Height/MinWindowSize/Tooltip` + `SetGrip/SetSizeReadout` + 条目的 `WithStatusBarItemIcon/Tooltip/OnClick/Disabled/Active/Width`。
  - **语义定死**: 左 Weight1 / 中 Auto / 右 Weight1, 右槽开头一段 Weight 撑子把条目顶到行尾 ⇒ 中槽按**整条**居中(右下抓手占 16 只让中心左偏半抓手宽, 不是"左右剩余空间"的中心); 左右内边距走元素 padding(不用子项外边距); 点击在 `LButtonUp` 触发且禁用态不响应, 程序化 `SetText/SetIcon` 不触发; 提示固定朝上(`TipTop`), 空串即解绑; 抓手拖动以**屏幕物理**增量 ÷ 窗口 DPI 换算成逻辑像素并 `SetCapture` 跟手(元素坐标会随窗口缩放漂), 最大化态与缩到下限(默认 320×240)都不动作, 读数跟宿主 `WM_SIZE` 刷新; 条目是自绘 Element, 不进 Tab 焦点链(同 157)。顶边 1px 按设备像素对齐, 抓手自己补画被它盖住的那一段。
  - **验收**: `go test ./...` / `go vet ./...` / `go build ./...` 全绿; `xui add status-bar` 单组件闭包与 `add --all` 全量各两套(带 `replace` 到工作区、以及不带 `replace` 走 v0.9.0 运行时)编译通过; 150% DPI 亮/暗截图 —— 中槽实测居中(条 78..1526, 文字中心 801)、hover 底色 `#F5F5F5`、气泡朝上(卡片样例里盖住上方"点击条目"一行属预期)、点条目后回显"点击条目: main"、禁用条目 50% 字色且底色不变、窗口改尺寸后读数从 `1176 × 836` 变 `1056 × 636`、暗色下 `Go` active 转浅底深字、抓手三条斜线正常; `components/statusbar_test.go` 覆盖读数格式、下限夹取、选项落位(非法值忽略)与条目选项。
  - **登记**: `registry.json` 加 `status-bar`(`files: ["statusbar.go"]`, `deps: ["tooltip"]`); demo 基础4 新增"状态栏"卡(28 默认 + 24 带读数两条样例)并在窗口底部挂一条真实状态栏(`XUI_HOME_PAGE` 0..20 不变); README 组件表、DESIGN 语义表 / 差异表 / 第九节 API 清单、HANDOFF 最近一批与待办同步。

- ✅ **新增 `tag-input`: 可就地删除的 chips 输入**(2026-11-05, HANDOFF 待办首项; 本地 shadcn 参考实现没有对应 tsx, 交互语义由本库定死并写进 DESIGN 第四节末):
  - **组件**: `components/taginput.go` 单文件(registry key `tag-input`, 无 deps);外壳自绘胶囊 + 焦点环/校验环/禁用态, 内部一个 `SetRectEx` 定位的裸 `Edit` 只负责文字/光标/IME。回车提交、分隔符(默认 `,` / `，` / 换行)即时切段、空输入退格删末个、点 × 删除;胶囊排满换行、外壳按行数自撑。`WithTagInputTags/Max/AllowDuplicates/Separators/Validator/Disabled/Invalid/Placeholder` + `Tags/SetTags/AddTag/RemoveTag/Clear/Value/SetValue/Focus/SetMax/OnChange`。
  - **语义定死**: trim 首尾空白、空串忽略、默认精确匹配去重(重复静默忽略);到上限或被校验器拒绝时**输入文字原样保留**(不清空, 免得用户以为内容被吞);输入框非空时 Backspace 按原生删字符;双击 × 也要命中(只挂 `LButtonDown` 时连点会漏一半);`OnChange([]string)` 只由交互触发, `SetTags/AddTag/RemoveTag/Clear/SetValue` 等程序化改值不广播, 避免回调里再改值递归。
  - **修复①(换行压住下一张卡)**: 输入 8 个标签后组件长到 3 行, 但"上限 3 个"提示与第二个 TagInput 仍停在旧位置, 胶囊直接压在下一张卡上。根因: 在 `XE_EDIT_CHANGED` / `XE_SIZE` 回调里同步 `SetHeight` + `LayoutItem_SetHeight` 重入布局趟, 且沿父链逐级 `AdjustLayout` 时中间层 `Weight` 把卡片/内容区临时排成 0 宽、兄弟位置不更新。修法: 量高走 `app.AfterFunc(0)` 合并同轮请求(同 Textarea / Alert), 只重排最近的 `LayoutFrame`/窗口;胶囊矩形与裸 Edit 定位仍立即更新(不碰自身高度)。验收: 亮色 150% DPI 截图 —— 8 标签 3 行, 提示/第二个输入框/禁用/校验卡依次下移无重叠;连按 6 次退格回到单行、兄弟项同步上移且无残影(见 DESIGN 第八节 187)。
  - **hover 跟踪**: 首轮像素对比 hover 前后差 0, 诊断日志显示是测试点落在 × 命中区下方(元素内 y=29, 命中区 8..24), 事件本身正常;顺手把跟踪从窗口级 `MouseMove` 简化为普通 Element 的元素级 `AddEvent_MouseMove` + `MouseLeave`(每次移动都触发, pPt 就是元素客户区坐标, 不用换算窗口坐标、不用判祖先可见), 命中矩形与点击共用。验收: 正确坐标下 × 区域 1040 像素变化、最大通道差 100, `xui.HoverItem/Fade` 过渡正常。
  - **验收**: `go test ./...` / `go vet ./...` / `go build ./...` 全绿;`xui add tag-input` 单组件闭包与 `add --all` 全量两套回归(带 `replace` 到工作区、以及不带 `replace` 走 v0.9.0 运行时)都编译通过;150% DPI 亮/暗截图 —— 回车新增 Design 并触发 OnChange、逗号一次切成 A/B、退格删末个、点 × 删除、焦点环、上限 3 / 禁用 / 校验失败三态、暗色胶囊配色均正常;`components/taginput_test.go` 覆盖多字节分隔符切分、去重/上限/校验器与 `Tags()` 副本。
  - **登记**: `registry.json` 加 `tag-input`(`files: ["taginput.go"]`, `deps: []`); demo 表单1 右列新增"标签输入"卡(不新增页签, `XUI_HOME_PAGE` 0..20 不变); README 组件表、DESIGN 语义表 / 第八节索引 187 / 第九节 API 清单、DESIGN-PITFALLS 187 同步。

- ✅ **Rating 补齐 Ant Design Rate 的 6 项特性**(2026-11-05, 用户需求"看看 rating.go 有没有值得添加的特性", 参照 [Ant Design Rate](https://ant.design/components/rate-cn) 与 `rc-rate` 源码; 第 6 项焦点回调按用户追加要求一并做):
  - **allowClear**: 默认开, 再次点击当前值归 0 并触发 `OnChange(0)`; `WithRatingAllowClear(false)` / `SetAllowClear` 可关。清除后用 `cleaned` 压住悬停预览(同 `rc-rate` 的 cleanedValue), 否则指针没动、MouseMove 会立刻把刚清掉的星又点亮, 画面与读数不一致。
  - **预览与文案**: `OnHoverChange(fn)`(离开/清除传 -1) + `WithRatingLabels([]string)` 在星星区右侧画 ceil(当前值或预览)-1 对应文案, 半星按 ceil 取上一档, 值为 0 不画; 文案宽用 `xui.TextWidth` 量一次并计入元素自然宽, 文案区单独铺 `Card` 底擦掉上一帧文字残影(透明元素重绘不擦底)。
  - **keyboard 开关**: `WithRatingKeyboard(false)` / `SetKeyboard` 把 ←/↓/→/↑/Home/End 交回 xcgui 的焦点切换; 观察型 `OnKeyDown` 在开关关闭时仍转发按键。
  - **尺寸档**: `RatingSmall/Medium/Large` = 星边长 15/20/25(AntD `controlHeight×0.625`); `ratingNaturalWidth/Area/StarRect/ValueAt` 全部改成按实例 `starSize` 计算, 奇数尺寸的中线判定要向上取整(15 的一半仍属左半)。只在构造时给, 运行中改会让元素矩形与自然宽脱节。
  - **自定义图标与 hover 反馈**: `WithRatingIcons(outline, full, half)` 三态 SVG(口径同 `StepItem.Icon` / `Painter.Icon`); `Painter` 没有裁剪 API, 半星必须自带沿 x=12 闭合的左半 path。悬停那颗星用运行时 `xui.HoverItem` 取过渡值, 经 `Painter.Icon` 的可选 `scale` 放大 1.1 倍(AntD `starHoverScale`), 命中测试矩形不动。
  - **焦点/按键回调(第 6 项)**: `OnFocus` / `OnBlur` / `OnKeyDown(vk)` 透出事件, `Blur()` 用宿主窗口 `XWnd_SetFocusEle(0)` 把焦点交还窗口(xcgui 没有元素级 KillFocus API); 失焦顺带清悬停预览。实测 `SetFocusEle(0)` 会触发 KillFocus, `Blur()` 有效。
  - **未纳入**: AntD 的黄色 `starColor`/`starBg` 组件 token(保持 shadcn 的 `primary` + `muted-foreground/50` 语义)、RTL、`autoFocus`、受控 `value`、逐星 `tooltips` 弹层(整块单元素 + `Tooltip.Bind` 按元素绑定会与组件自己的 MouseMove 抢时序)、运行时改尺寸档/文案(元素矩形不会自己变宽)。
  - demo 表单2 "选择控件"卡评分段扩展为: 半星 + 文案 + 清除、只读/禁用/不允许清除、三档尺寸、心形三态 + 关闭键盘、焦点/按键回调 + `Blur()` 两个按钮; registry 无变化(仍只有 `rating.go`)。
  - **验收**: `go test ./components -run TestRating`(值吸附、三档几何/命中映射、清除取值、文案与悬停索引、选项复制、SVG 解析、焦点回调/Blur 安全)、`go build/vet/test ./...` 全绿; `xui add rating` 闭包与 `add --all` 逐组件编译通过; 150% DPI 亮/暗截图 —— 3.5 半星 + "好"、再点当前星归 0(5 空星、文案消失)、悬停第 5 颗显示"很好 / 预览: 5"、点击提交到 5、Home 归零且出现焦点环与"键 0x24"、`Blur()` 后显示"焦点离开"、keyboard=false 行点第 4 颗变 4 但按 → 不变、三档尺寸与心形三态在亮/暗两套配色下均正常。
  - **追加修复(hover 放大不回退)**: 鼠标移出评分组件后, 被悬停的那颗星仍保持 1.1 倍放大, 预览星与文案也一并残留。根因: `clearHover()` 只清状态没重绘 —— 逐项 hover 过渡的回退靠 paint 每帧调用 `xui.HoverItem(active=false)` 推进, 而 MouseLeave 之后不会再有 MouseMove, 少这一帧就永远停在放大终态。修法: `clearHover()` 无条件 `Redraw(true)`(hover 已是 -1 也要重绘, 过渡值可能还停在 1)。验收: 150% DPI 截图 —— 悬停第 1 颗星显示"预览: 1 / 很差"且该星放大, 移开后恢复 3.5 半星 + "好"、5 颗星等大、预览文案消失; `go build/vet/test ./...` 全绿。

- ✅ **Steps 补齐状态 / 进度 / 可点 / 尺寸**(2026-11-05, 用户需求"看看步骤条有没有值得添加的特性", 参照 Ant Design Steps):
  - **状态**: 新增 `StepStatus`(`Auto / Wait / Process / Finish / Error`)与 `StepItem.Status` 单步覆盖; 出错步 = 破坏色实心圆 + X, 指向它的连接线也转破坏色; `WithStepsStatus` 指定当前步的默认状态(如整条流程标红)。
  - **进度**: `StepItem.Percent`(1..100) 在进行中步骤的圆点环上画主色进度弧, 并新增 `SetItemPercent(i, pct)` 供上传/安装流程动态推进; 连接线保持状态色, 不重复表达进度。
  - **可点性**: `WithStepsClickable` 放开任意非禁用、非当前步(默认仍只可点回已完成步); `StepItem.Disabled` 单步禁用但保留显示与键盘浏览; 命中/键盘激活统一走 `canActivate`。
  - **尺寸与编号**: `WithStepsSize(StepsSmall)` 走独立几何/字号表(圆点 20、标题 18/说明 14); `WithStepsInitial(n)` 只偏移显示编号, 不影响索引与点击语义。
  - **绘制**: 进度弧用扇环多边形拼(同图表扇区), 填充走 D2D 直通 `FillGeometry` 获得抗锯齿 —— `XDraw_FillPolygonF` 在 D2D 下是硬填充, 小圆点上的弧会出现一圈台阶(见 DESIGN 第八节 160); 公共助手抽到 `d2dfill.go`, chart 与 steps 的 registry 闭包共同包含(单组件安装仍可独立编译); 四态圆点/图标/标题/说明配色与错误段连接线在 `onPaint` 内实时取主题令牌。
  - **未纳入**: AntD 的 `type`(dot / navigation / inline / panel)、`progressDot`、`maxCount`、`responsive`、`subTitle`、`labelPlacement` / `titlePlacement`、`variant` 与逐条 `content` —— 与现有单元素自绘的布局方向/组件边界不匹配或收益不明确, 不增加公共 API。
  - demo 导航3 步骤条卡: 自定义图标行改为可点 + 第 4 步禁用; 新增出错状态行(红色连接线 + X)与小号/进度/起始编号行(进度按钮从 35% 动态推进)。
  - **验收**: `go test ./components -run TestSteps`(自然高度含小号档、横/竖向布局、命中谓词、statusAt 覆盖优先级、canActivate 禁用与可点), `go build/vet/test ./...` 全绿; 150% DPI 亮/暗截图 —— 错误节点与入线变红、进度弧从 35% 随按钮推进到 95%、小号几何与编号 3..6、点图标行第 3 步三行联动、点禁用第 4 步无变化; `xui add steps` 闭包与 `add --all` 编译通过。

- ✅ **demo 导航2 页面拆分**(2026-11-05, 用户反馈"导航2页面太长了"):
  - **导航2** = 标签页 / 面包屑 / 分页; 新增 **导航3** = 步骤条 / 可折叠面板; 新增 **导航4** = 时间轴(竖向 start/交替/end + 水平三行)。原导航3/4(导航菜单/菜单栏、侧边栏)顺延为 **导航5/6**。
  - `pageTitles`、页面枚举、构建函数与页头注释同步; `XUI_HOME_PAGE` 映射变为 **0..20**(导航2=10 / 导航3=11 / 导航4=12 / 图标页=20), HANDOFF 的启动页清单与图标页说明一并更新。
  - **验收**: `go build/vet/test ./...` 全绿; 150% DPI 截图确认页签 21 项按 11+10 两行均衡换行, 导航2 停在分页卡、导航3 从步骤条滚到可折叠面板、导航4 覆盖全部时间轴示例、导航5 菜单与底部图标页(20)映射正确; 各页滚动条范围正常, 无空白页。

- ✅ **Timeline 支持两侧交替 / 水平取向 / 节点样式**(2026-11-05, 用户需求"支持交替展现, 内容在时间轴两侧轮流出现; 支持水平方向", 并参照 Ant Design Timeline 补齐值得做的特性):
  - **取向与模式**: `WithTimelineOrientation(ui.TimelineHorizontal)`; `WithTimelineMode(ui.TimelineStart / TimelineEnd / TimelineAlternate)`。竖向 start 轴线在左、内容在右, end 轴线在右、内容右对齐, alternate 轴线居中、左右交替(第一条在左); 水平 start 轴线在顶、内容在下, end 轴线在底、内容在上, alternate 轴线居中、上下交替(第一条在上), 各条等分容器宽度。
  - **节点样式**: `WithTimelineItemColor(ui.TimelineColorPrimary / Destructive / Muted)` 或 `WithTimelineItemColorFunc` 自定义主题色; `WithTimelineItemIcon` 用 16px 图标代替圆点; `WithTimelineItemLoading` + `SetLoading` 画运行时旋转指示器; `WithTimelineVariant(ui.TimelineOutlined)` 把实心点改成空心环(中心用宿主底色挖洞, 轴线不会穿过)。优先级: 加载 > 图标 > 空心环 > 实心点。
  - **水平几何**: 条目先按 1fr 参与首轮布局, 尺寸确定后由延迟 reflow 锁成等宽 Fixed(横向 Weight 列不会把宽度下发给 Fill 子项); 实测各条内容高, 按共享轴线写上下内边距, 使每条总高相等(start = 内容 + 32, end/alternate 按两侧最高内容对齐), 线条才能横贯一条直线。竖向 alternate 的轴线位置与左右内边距按容器宽度实时计算, 窗口缩放后重新居中。
  - **踩坑(窗口卡死 / 未响应)**: 初版在 `AddItem` 里同步 `AdjustLayout` 祖先、且 reflow 每次都无条件上推最近框架; 水平时间轴第一次排完后的尺寸事件又触发下一轮 reflow, 滚动条/卡片 Auto 高在两次布局结果间来回, 页 10 启动即 100% CPU、白屏不绘制。**修法**: 构建期只排自己, 祖先重排只针对最近的 ScrollArea 框架(或窗口)一次; reflow 先比对宽度/内边距/轴线值, 没变化就直接返回, 不再触发外层布局。修复后同一页面连续缩放、亮暗切换均稳定。
  - **未纳入**: AntD 的 `reverse`(调用方加条目时自行倒序即可)、`titleSpan`(本库节点列宽固定 32)、逐条 `placement` 覆盖(与 `mode` 重叠)、旧版 `pending` 虚线尾巴(用 loading 节点表达进行中)。这些没有明确收益, 不增加公共 API。
  - demo 导航2: 原竖向 start 卡保留, 新增"时间轴 · 两侧交替"卡(语义色/图标/加载态 + 竖向 end)与"时间轴 · 水平"卡(start / end / alternate + 空心节点三行)。
  - **验收**: `go test ./components -run TestTimeline`(模式取向对应侧、竖向 alternate 左右等宽与轴线间距、水平三种模式的内边距总和与轴线间距)、`go build/vet/test ./...` 全绿; 150% DPI 亮/暗截图 —— 竖向 start/end/alternate、水平 start/end/alternate、图标/空心/灰点/红色节点/旋转加载态、轴线逐段连续且首末不越界; 窗口缩窄后水平条目等分重算、竖向 alternate 轴线重新居中、无卡死; `xui add timeline`(自动带 label)与 `add --all` 编译通过。

- ✅ **Timeline 时间轴组件**(2026-11-05, 组件缺口盘点中"timeline"一条; shadcn 无官方组件, 语义按社区通行做法定死并写进 DESIGN 第四节末):
  - `components/timeline.go`: `NewTimeline(x, y, cx, cy, parent, WithTimelineGap/Backdrop)` + `AddItem(WithTimelineItemTime/Title/Description)` + `SetBackdrop`。条目本身是竖向 `LayoutEle` 内容槽, 内置 时间/标题/说明 三个 Label, 之后可继续加任意 Label / Item / Card。
  - **绘制**: 左侧轴线 2px `border` + 实心圆点 10px `primary`(外加 7px 宿主底色光环盖住轴线), 圆点对齐条目**第一个可见子项**的垂直中心; 首条不画上方线、末条不画下方线。
  - **关键几何**: 条目间距用**元素底部内边距**表达(默认 24), 轴线画满这段内边距才能和下一段接上; 如果改用布局项外边距, 线会被条目矩形裁掉, 相邻两段之间永远缺一截。
  - **擦除**: 每个条目先把自己左侧节点列(32px)铺成宿主底色再画线和点 —— 线条长度随内容高度变化, 透明元素重绘不擦底会留下上一帧的线头; 默认 `Card`, 非 Card 容器用 `WithTimelineBackdrop`/`SetBackdrop`。
  - demo 导航2 加"时间轴"卡片(订单时间线 4 条: 已创建 / 已付款 / 已发货 / 待收货, 第三条内容槽里嵌一个 outline `Item` 展示物流信息); registry 加 `timeline`(files: `timeline.go`, deps: `label`)。
  - **验收**: `go build/vet/test ./...` 全绿; 150% DPI 亮/暗截图确认四条时间线轴线连续、圆点与首行(时间)垂直居中、首条无上方线、末条无下方线、内嵌 Item 后轴线延伸到条目底部; `xui add timeline` 闭包(自动带上 label)与 `add --all` 编译通过。

- ✅ **Rating 星级评分组件**(2026-11-05, 组件缺口盘点中"rating"一条; shadcn 无官方组件, 语义按社区通行做法定死并写进 DESIGN 第四节末):
  - `components/rating.go`: `NewRating(x, y, cx, cy, parent, WithRatingValue/Max/Half/Readonly/Disabled)` + `Value / SetValue / SetReadonly / SetDisabled / OnChange`。
  - **语义**: 5 星(可配)、整星/半星(0.5 步进); 悬停预览、点击提交(命中星左半 = i+0.5, 右半 = i+1); 只读只展示、禁用 50% 透明且不响应; 整块一个焦点元素, ←/↓ 减一档、→/↑ 加一档、Home/End 两端(到端点不循环), 键盘改值立即 OnChange。
  - **绘制**: 空星用 lucide `IconStar` 描边(`muted-foreground/50`), 实心/半星用同色 **实心 SVG 多边形**叠画(`primary`); 半星 = 星形沿 x=12 中轴线闭合的左半。焦点环只围星星区(不用 `p.FocusFrame` 的元素整矩形), 元素被 `ItemFill` 拉宽时不会围出一整行空环。
  - **踩坑(形状对不上)**: 第一版填充多边形按"外径 10 / 内径 4.2"估算, 比 lucide 星形瘦一圈, 实心星描边内侧露出一圈背景白缝(像空心星套了个描边); 按 lucide 的实际曲率改成内径约 6.0 后贴合, 半星也沿中轴线闭合无缝。
  - demo 表单2 左列"选择控件"卡加评分段(3.5 半星 + 只读 4 星 / 禁用 3 星 + 实时标签); registry 加 `rating`(files: `rating.go`)。
  - **验收**: `go test ./components -run TestRating`(值吸附/自然宽度/命中映射/实心星 SVG 解析)、`go build/vet/test ./...` 全绿; 150% DPI 亮/暗截图 —— 3.5 半星、悬停第 5 颗预览 5 星、点第 1 颗变 1、按 → 两次(半星步进)到 2 且焦点环出现、Home 归零、点第 3 颗左半到 2.5、只读悬停/点击与禁用点击均不变化; `xui add rating` 与 `add --all` 的闭包编译通过。

- ✅ **Steps 支持自定义图标**(2026-11-05, 用户需求"支持自定义图标, 而不只是序号"):
  - `StepItem` 新增 `Icon string`(与 `Button.WithIcon` / `Painter.Icon` 同口径: `icons.IconXxx` 常量、path 数据或整段 SVG)。非空时圆点里**三态都画该图标**, 代替当前/未达态的序号与已完成态的对勾; 完成与否仍由圆点填充/环色与连接线颜色表达(同 Ant Design / Element Plus 的 step icon 口径, 写进 DESIGN 第四节末)。
  - 图标边长 14(对勾 12): 24px 圆点、2px 环的洞是 20px, 14 留 3px 呼吸; 复杂 lucide 轮廓在 12px 下容易糊。
  - demo 导航2 步骤条卡片加第三行"自定义图标"(用户 / 邮件 / 锁 / 完成四枚图标, 与横向、竖向联动)。
  - **验收**: `go build/vet/test ./...` 全绿; 150% DPI 截图确认横向/竖向三态图标都居中、已完成白图标压在主色实心圆上清晰、当前主色图标、未达弱化图标; 点击已完成步回退时图标态随当前步联动; 亮/暗主题各看一遍。

- ✅ **v0.9.0 发布(2026-11-04)**: 新增 FileDialog / Steps 两个组件, 走 minor。
  - **顺序**: 组件与文档两笔提交 → 版本示例 + `fallbackXuiVersion` 提交(`9783427`)→ 轻量 tag `v0.9.0` → `runtime` 抬到 v0.9.0(`2258748`, tag 之后)。
  - **推送**: `git push origin main v0.9.0` 四个远端全部成功(GitHub / Gitee / GitCode / cnb.cool), `git ls-remote` 逐一对上; GitHub / Gitee 的 raw `registry.json` 已能看到 `runtime: v0.9.0` 与 `file-dialog` / `steps`。
  - **Release**: https://github.com/twgh/xui/releases/tag/v0.9.0 , 名称为 `v0.9.0 — FileDialog / Steps / 文档配套下发`; 正文与 `.verify/v0.9.0.md` 逐行一致(只差 GitHub 去掉的尾部空行)。
  - **下限回归**: 临时项目 `go mod edit -require=github.com/twgh/xui@v0.9.0` → 本地 registry `add --all` → `GOPROXY=direct GOSUMDB=off go mod tidy && go build ./ui` 通过。
  - **踩坑**: PowerShell 5.1 把无 BOM 的 `.ps1` 当 ANSI 读, 脚本里直接写中文会让 release 名称乱码; `Get-Content -Raw` 的正文经 `ConvertTo-Json` 还会被当成对象写进发布页。改为 `[IO.File]::ReadAllText(path, UTF8)` 读 name/body 后 PATCH 修复。工作流已补进 HANDOFF「发布流程」。

- ✅ **文档体积优化: DESIGN 第八节拆到 DESIGN-PITFALLS.md + HANDOFF 瘦身**(2026-11-04, 用户反馈 HANDOFF 单行太长/文件太大、DESIGN 400 多 KB 费 token):
  - **DESIGN.md**: ≈439KB/2097 行 → 156KB/825 行。第八节的 186 条正文原样移到同目录的 `DESIGN-PITFALLS.md`(310KB/1474 行), DESIGN 第八节只留编号 + 标题索引, 两处编号一一对应; 正文逐条原样搬运(空白归一化后与原第八节逐行相等), 顺带把 48~52 条误缩进在 47 下的嵌套列表提回顶层。
  - **HANDOFF.md**: 1205 行 → 237 行。按时间记录的 129 条完成 / 修复记录原样移到 `docs/CHANGELOG.md`, HANDOFF 只留当前状态、待办、仓库结构、命令与验证工作流。
  - **CLI**: `init` / `add DESIGN.md` 同时下发 `DESIGN-PITFALLS.md`; 老项目(拆分前 init 过、DESIGN.md 已在)跑 `update` 会补装缺失的配套文件(它属于 DESIGN.md 第八节正文, 不算 update 新增), `--dry-run` 只预告; `update` 点名 DESIGN.md 也连配套一起刷新。`xui --help`、init 完成提示、两份下发的 AGENTS 模板与生成项目的 README 目录清单同步。
  - **Markdown 坑(已写进 AGENTS 硬规则)**: 有序列表内容缩进 = 编号位数 + 2 空格, 三位编号 `100.` 之后仍用 4 空格会让 100 条以后整段渲染成缩进代码块; 拆分前已把 372 行各加 1 空格修正(GitHub Markdown API 复验只剩第 129 条里那段有意的 Go 代码块)。
  - **验收**: `go build/vet/test ./...` 全绿; 真 binary 临时目录回归 —— `init -n` / `init` / `add DESIGN.md` 三路都出两份文档且不误写 AGENTS.md, 老项目 `update` 补装、`--dry-run` 不落盘、空目录 `update DESIGN.md` 只指路不建目录; 逐组件安装编译与 demo 构建不受影响。
  - **DESIGN 第五/六节的 CLI 说明同步**: 补齐 `init -n` 的 winres/ 与 xc 生成、普通 `init` 在空目录等同 `init -n .`、组件目录里的 `DESIGN-PITFALLS.md`/`AGENTS.md` 清单、裸 `update` 连已下发文档一起刷新与老项目补装配套文件、`add/update DESIGN.md` 的配套关系、`--no-install` 对 `init -n` 还跳过 tidy/xc; 并改掉"普通 init 不覆盖已有文件"的旧说法(实际按基线刷新)与 `-f` 的错误提示文案。

- ✅ **Steps 步骤条**(2026-11-04, 组件缺口的第二条; shadcn 无官方 stepper, 语义按社区通行做法定死并写进 DESIGN 第四节末):
  - `components/steps.go`: `NewSteps(x, y, cx, cy, parent, []StepItem{{Title, Description}}, WithStepsOrientation/Current/Backdrop)` + `SetCurrent / Current / OnChange / SetBackdrop`。
  - **形态**: 横向 = 等宽列、圆点居中、标题/说明在圆点下方居中、圆点间横线; 竖向 = 圆点在左、文字在右、行间竖线; 元素高于自然高度时余量均摊到竖向行距。
  - **三态**: 已完成 = 主色实心圆 + 对勾; 当前 = 主色环 + 主色编号; 未达 = 边框环 + 弱化编号; 已完成段连接线取主色。圆点/环用两层实心 `FillEllipse`(同 ColorPicker), 挖洞色取 `SetBackdrop`(默认 `Card`)。
  - **交互**: 只有已完成步可点回(`OnChange`), 当前/未达步不响应鼠标; 整块一个 Tab 站点, ←/→/↑/↓ 在**所有步骤**间移动焦点(未达步可只读浏览)、Home/End 首末、Enter/空格回退到已完成步; 键盘挂窗口级(DESIGN 152), 焦点环画在圆点外侧(元素四周预留 3px)。
  - **修复(同日用户报障"键盘我按下没用")**: 原实现方向键只在内部焦点里挪(范围还被夹在 0..current), 鼠标点过组件后 `FocusedByKeyboard()` 为假 → 按 ←/→ 既不画焦点环也不改当前步, 画面完全不动; 按 → 从当前步出发更是无路可走。**修法**: 焦点范围放开到全部步骤, 并在 `onKeyDown` 开头调 `xui.NoteKeyboard()` —— 只要用了键盘就切到键盘焦点态, 鼠标点过后直接按方向键也能立即看到焦点环移动; Enter/空格仍只回退已完成步(前进口径未变, 仍是"只能点回已完成步")。复测: 点当前步后按 ←/→ 焦点环左右移动、按 ← 到已完成步后 Enter 回退、Home/End 首末、连续按 ←/→ 到头不循环。
  - registry 加 `steps`(files: `steps.go` + `textopt.go`); demo 导航2 加卡(横向 + 竖向右联动、上一步/下一步按钮与状态标签, 按钮随端点禁用)。
  - **验收**: `go test ./components -run TestSteps`(自然高度 / 横向等宽 / 竖向行距分配 / 命中只认已完成步)、`go build/vet/test ./...` 全绿; 150% DPI 亮/暗主题截图 —— 4 步横向 3 态、竖向 4 步、点击横向/竖向已完成步回退、当前/未达步点击无响应、键盘 ←/Enter 回退与焦点环、1200 → 745 宽横向列等比收缩; `xui add steps` 闭包与 `add --all` 编译通过。

- ✅ **FileDialog 文件对话框门面**(2026-11-04, 组件缺口盘点后的第一条):
  - `components/filedialog.go`: `NewFileDialog(hWindow, WithFileDialogTitle/Filters/InitialDir/DefaultName/DefaultExt/MaxFiles)` + `Open / OpenMulti / Save / PickDir`; `FileFilter{Name, Patterns}` 自动格式化成 "名称 (通配符)"; 取消返回空串 / nil。
  - 底层走 `wutil.OpenFileEx / SaveFileEx`(保存默认带覆盖提示); 多选 `OFN_ALLOWMULTISELECT | OFN_EXPLORER | OFN_PATHMUTEXIST`, 默认最多返回 10 个; 选择文件夹自己拼 `BrowseInfoW` + `SHBrowseForFolderW`(`wutil.OpenDir` 标题写死"请选择目录", 这里要支持 `WithFileDialogTitle`)。
  - registry 加 `file-dialog`; demo 表单3 加卡(打开 / 打开多个 / 保存 / 选择文件夹四个按钮 + 结果标签), 卡片挂在 demo 主窗口上作 owner。
  - **验收**: 四种对话框逐一点开截图确认 —— 打开(过滤器显示"图片 (*.png;*.jpg;*.jpeg;*.gif;*.bmp)")、多选(`Ctrl+点` 两个 .go 返回 2 个、首个路径正确)、保存(默认名 `demo.png` + 类型过滤 + 同名文件弹"确认另存为"覆盖提示, 选否后取消)、选择文件夹(自定义标题生效、返回选中的目录); 取消后标签显示"已取消"。`go test ./...`、`go vet`、demo build 通过; 逐组件安装(`xui add file-dialog` → `go build ./ui`)与 `add --all` 编译通过。

- ✅ **`xui init` 在空目录里生成完整项目**(2026-11-04, 用户报障: 新建的空目录里 `xui init` 只得到 `ui/`):
  - **现象**: 空目录(用户那个是 `D:\...\ui\myapp2`)里跑 `xui init`, 只创建组件目录, 然后下一步让人 `go get github.com/twgh/xui github.com/twgh/xcgui` —— 可现场根本没有 `go.mod`, `go get` 直接失败, 目录里也没有 `main.go`, 既不能 `go run` 也不能装依赖。用户的预期是"空目录 = `xui init -n 目录名`"。
  - **根因**: `init` 的两种形态按**有没有 `-n`** 分派, 与目录内容无关; 空目录走进"已有项目只写组件目录"那条, 而空目录里根本没有"已有项目"。
  - **修法**: 分派从 `main` 的 switch 抽成 `initCommand(opts)`(便于单测直接打这条路径), 新增判据 `isBlankProjectDir(cwd, uiDir)` —— 四条同时成立才在当前位置建项目: ① 目录里没有内容(`.git` 不算, `git init` 后建仓是常规顺序); ② 不在别人的 Go 模块里(`findGoMod(cwd)==""`, 否则组件该并进那个模块, 不该在子目录里再起一个 `go.mod`); ③ 目录名能当项目名(`projectNameFromDir!=""`, 盘根这类拿不到名字就退回旧行为, 不报错); ④ 组件目录落在其内(`uiDir=="."` 或绝对/含 `..` 时用户要的是"组件放别处", 沿用 `checkUIWithinProject`)。满足即 `initNewProject(".", uiDir, force=true, w)` —— **这里的 force 不是放开非空闸门**, 空/只有 `.git` 是上一步自己判的。
  - **不满足**时旧行为不变(含 `init .` 与 `init -p ..\out`: 组件目录不是一个"项目里的子目录" = 用户要的是"组件放别处/铺在根上", 不改建项目), 只补一句指路: `initProject` 收尾时若 `findGoMod(dir)==""`, 说明第 1 步的 `go get` 没有地方写, 给出"自己 `go mod init` 或 `xui init -n <项目名>`"。
  - **文案**: 走脚手架时先打一行"当前目录是空的: 在这里生成完整项目 (项目名 X, 同 xui init -n .)"; CLI 帮助、README「新建项目/已有项目/命令清单」同步。
  - **测试**: `TestInitCommandInEmptyDirBuildsProject` 六条 —— 空目录出全套(go.mod/main.go/README/AGENTS/.gitignore/winres/ui)、只有 `.git` 的目录同样且 `init components` 跟着目录名、非空目录只写组件目录并给指路、别人模块里的空子目录不起第二个 go.mod、组件目录跳出当前目录不改建项目、`--dry-run` 一个条目都不落盘。把 `isBlankProjectDir` 短路成 `false` 复跑确认前三条断言会红(不是空过)。
  - **实测(真 binary + 临时目录)**: 空目录 `xui init --no-install` → 骨架齐全, 现场 `go mod tidy` + `GOWORK=off go build ./...` 通过; `-c <空目录> init`、`init components`、`init .`、`init -p ../out`、`--dry-run`、只有 `.git` 的目录六种写法分派逐一对上;`go build/vet/test ./...` 与 gofmt 全绿。

- ✅ **运行时升级后补 `go mod tidy`(2026-11-03, 用户报)**:
  - **现象**: `xui update` / `add` 打完"运行时已升到 v0.8.2 (go.mod 与 go.sum 一起更新)", 项目 `go.sum` 里 v0.8.1 的 h1 / go.mod 两行却原样留着 —— 看着像没升干净。
  - **根因**: `ensureRuntime` 只跑 `go get module@tag`。`go get` 只往 `go.sum` **追加**新版本条目, 旧版本的条目不会删(Go 的既定行为); `go.sum` 的过期条目只有 `go mod tidy` 会清。
  - **修法**: `go get` 成功后补一次 `go mod tidy`(新测试缝 `tidyModules`, `TestMain` 默认 stub 掉)。tidy 失败不算升级失败(`go.mod` 已抬上去), 只 warn 并给手动命令与残留说明 —— 大项目 tidy 可能撞上 `runGo` 的 2 分钟超时, 不该把整次安装判失败。
  - **实测**: 钉 v0.8.1 的项目(go.sum 只有 v0.8.1 两行)跑 `xui add checkbox` → 输出"整理依赖 (go mod tidy) ..." → `go.sum` 只剩 v0.8.2 两行; 旧版本条目消失。
  - **文案同步**: CLI 帮助的"运行时版本"段、`--dry-run` / `--no-install` 的预览命令、README「运行时版本」、DESIGN 第五节、两份下发的 AGENTS 模板都改成"go get + go mod tidy 两条"。
  - **注意**: v0.8.2 已发布, 不含这个修复 —— 用户要拿到它需要下一个 patch tag。
- ✅ **v0.8.2 发布(2026-11-03)**: 只有 CLI 与文档变更, 走 patch。
  - **顺序**: 先把 `fallbackXuiVersion` 与 README 的版本示例抬到 v0.8.2 并提交(`805b857`) → 打轻量 tag `v0.8.2`(指向该提交, devel 构建回退的版本与 tag 一致) → 同一批把 `registry.json` 的 `runtime` 抬到 v0.8.2(`6d9e9f9`, 写早了 `go get` 拉不到 tag)。
  - **推送**: `git push origin main v0.8.2` 一条命令三远端都成功(Gitee / GitHub / GitCode), 无需逐远端重复。
  - **下限回归**(发布后): 临时项目 `go mod edit -require=github.com/twgh/xui@v0.8.2` → `add --all`(71 个 .go) → `go build ./ui` 通过。**踩坑**: goproxy.cn 对新 tag 还没收录(返回 `stream error: INTERNAL_ERROR`), `GOPROXY=direct` 又卡在 sum.golang.org(被墙); 实际可用组合是 `GOPROXY=direct GOSUMDB=off` —— 下次发布直接这么跑, 省一轮。
  - **GitHub Release 未建**: 本机 `gh auth status` 报 keyring token 失效(`gh auth refresh -h github.com` 是交互式, 不能在 agent 会话里跑), 且访问 api.github.com 报 `x509: certificate signed by unknown authority`。notes 正文在 `.verify/v0.8.2.md`(gitignored), 认证恢复后 `gh release create v0.8.2 --title 'v0.8.2 — CLI 参数审计收尾' --notes-file .verify/v0.8.2.md` 即可; 或网页建 Release 粘贴正文。
  - **文档**: 状态行与本文条目记录; README `@v0.8.2` 示例、`fallbackXuiVersion`、`registry.runtime` 三处版本引用已同步(测试夹具里的 v0.8.x 是样例, 不动)。
- ✅ **审计后又实测出三个缺陷(2026-11-03)**:
  - **`update <组件>` 仍是 add**: 裸 update 已按 `installedComponents` 过滤, 但**显式点名绕过它** —— 空目录里 `xui update button` 把 button 全套"创建"出来(`ui/` 目录都一起建了)。与 `update --all` 被拒掉是同一条纪律的残留。修法 `dropUninstalled`: **一个文件都不在本地**的组件只 warn 指路 `xui add`, 不进 `install`; 部分文件缺失仍补齐 —— 跨版本升级会在组件里新增共享文件, 不补直接编译不过(与 `installedComponents` 的"全都有才算已装"互补)。测试: `TestUpdateDoesNotInstallUninstalledComponents`(不建目录、不建文件、提示 add)+ `TestUpdateFillsMissingFileOfInstalledComponent`(缺 press.go 会补回)。
  - **AGENTS.md 正文随 `-p` 写法抖动**: 模板会把组件目录写进正文("本目录叫 X" + "-p X"), 而当时直接渲染 `opts.uiDir` 原样 —— `-p ui` / `-p .\ui` / `-p D:\proj\ui` 生成**三份不同内容**。后果: 同一份没改过的文件下次 update 被判"版本升级"静默重写(内容里还写着绝对路径), 用户改过的那份换个写法就误报"本地已修改"。修法 `docUIDir`: 绝对路径折成相对当前目录、反斜杠转斜杠, 落在 cwd 之外才保留绝对路径(要能照抄进命令)。测试 `TestDocUIDirIsStable` 覆盖五种写法 + 越界绝对路径。
  - **多 GOPATH 时找不到刚装的 xc**: `winres.go` 的 `goBinDirs` 对 `go env GOPATH`(**Windows 用 `;` 分隔多目录**)整串做 `filepath.Join(d, "bin")` → 拼出一个不存在的路径, `findXc` 装完仍报"已执行 go install 但没找到 xc"。改用 `filepath.SplitList`(与 `queryGoBinDir` 同一写法)。
  - **`init -n -p` 的脏目录名会进 import**: `-p .\components` / `-p components/` 被原样渲染进 `main.go` 的 `ui "app/./components"`, 而 Go 的 import 路径不认点段与尾斜杠(直接编译不过); 同一个值还会进 README/AGENTS 的路径。修法: 解析阶段 `filepath.ToSlash(filepath.Clean(...))`, 并拒掉 `-p .`(组件是 `package ui`, 与 `main` 同目录编译不过)与空位置参数(以前会 Clean 成 `.`)。真 binary 复验: `-p .\components` → `ui "app/components"`, `go mod tidy && go build` 通过。
  - **`--diff` 的缺失文件提示改成按"组件装没装"分岔**: 原来一律说"update 会创建", 但 update 已经不装新组件了 —— 整个组件没装过时改说"要装它请用 xui add X"(JSON 的 `state` 仍是 `missing`, 契约不变)。
  - **`--dry-run` 之外的小修**: `self-update --check` 在本地版本**高于** `@latest`(未发布 / 代理滞后)时 switch 没有分支、只打两行版本号不给结论 → 补 default「本地版本不低于 @latest, 无需升级」; `reportSkipped` 的汇总句不再断言"本地有改动"(无基线的那类只是"判断不了来源", 逐行已分开说); `unifiedDiff` 新增/删除空文件的一侧 hunk 头改为 `@@ -0,0 +1,N @@`(标准写法, 原 `-1,0` 会被 patch 当成"删了第 1 行"); 数据源选路/换源的进度行改受 `-s` 约束(降级到内嵌源码那句仍始终打), 显式 `XUI_REGISTRY` 不再印出空的 `()`。
  - **`--view`/`--diff` 的过滤值没命中改为失败(退出码 1)**: 原来只 warn 就退出 0, 与"匹配上了但没差异"只差一行字 —— 脚本(尤其 `-j`)会把"你写的过滤值不存在"当成"拿到了空结果"。改法: `viewFiles`/`diffFiles` 返回 matched, 调用方 `requireMatched` 走 `fatal`(JSON 模式先照打结果再失败, 结构里的 `matched=false` 仍是权威)。`TestDiffFilterMissExitsNonZero` 用子进程验退出码(fatal 路径不能直接调, 会把测试进程杀掉 —— 与 `TestSkippedSummaryAndStrictExitCode` 同一范式); `TestJSONOutputShapes` 改成直接调 `diffFiles` 断言返回值。
  - **`init -n` 换 `-p` 重跑会另起一套**: 忘了带 `-p`(默认 `ui`)或换了目录名时, 项目里多一套组件与清单, 而骨架文件的基线在旧清单里 —— 之后 `main.go`/`README.md` 全被报"无法判断来源"。新增 `otherProjectComponentDir`(扫项目根下三层, 找清单里带 `project_files` 的目录), 命中就 warn 一句**但不阻断**(一仓两套罕见但合法)。`TestInitNewProjectWarnsAboutSecondComponentsDir` 验首次不提醒、第二次提醒、两套都真的写出来。
  - **文档**: DESIGN 第五节 `update` 那条补"点名未装组件只提示、不替你装; 已装组件缺的个别文件照补", 另补过滤没命中的退出码与 `init -n` 换 `-p` 的提醒; README 命令表、选项表(`--view/--diff`)与退出码段同步。
  - **实测**: 真 binary 空目录 `update button` → 只提示且 `ui/` 不存在; `update AGENTS.md` 连跑 `-p .\ui` / 绝对路径 / `-p ui` → 一次归一化后两次"已一致", 正文只剩 `ui`; `go test ./cmd/xui/... -count=1` 全绿。
- ✅ **裸 update 也刷已在的两份文档(2026-11-03)**:
  - **缺口**: 上一批只做到"点名才刷", 于是 `xui update` 一次都不提 `DESIGN.md`/`AGENTS.md`(实测 grep 计数 0), 可这两份**已经记在清单里**、是 CLI 交付过的东西。我当时把"`update` 不新增它没装过的东西"错用成了"`update` 不管它们" —— 刷新已在的文件不是新增, 那条纪律管不着。
  - **改法**: `existingDocs(uiDir)` —— 裸 update 时把目录里**已有的**那两份并入目标;缺的一份都不补、也**不提示**(只用 `add` 装组件的项目不该每次被提醒), 点名时才给"要单独拿它: `xui add X`"的指路。含义变成: 裸 `update` = "把这个目录里 CLI 给过的都刷新"。
  - **测试**: `TestNamedDocTargets` 补四断言 —— 裸 update 提到两份文档;删掉 AGENTS.md 后裸 update 既不重建也不吭声;点名时才提示 `xui add`;`--view` 模式仍不拆文档名。
  - 实测(真 binary): 裸 update 输出含 DESIGN.md/AGENTS.md;`update DESIGN.md` 仍是只刷那一份。
  - 文档: CLI 帮助的 update 行、DESIGN 第五节(随 init 下发)、README 命令表一行。
- ✅ **修掉自己引进的基线缺陷: 按"交付后盘上内容"记(2026-11-03)**:
  - **缺陷**: 上一批把模板基线记成"**打算写的字节**", 而同一次 `init -n` 后面会**自己改写 `go.mod`** —— `ensureRuntime` 抬配套运行时(`registry.runtime` 比 CLI 自身版本新时, 即刚发版而用户 CLI 还旧)和 `go mod tidy` 都动它。于是文件一交付就和自己的基线不一致, 下次 `init -n --force` 会说"**本地已修改**, 跳过", 可用户根本没动过。方向是保守的(不会覆写, 安全), 但**措辞是假话** —— 正是这一整轮在消灭的那类。
  - **修法**: 循环里记下 `delivered`(这次真交付了哪些), 全流程结束后重新加载清单、按盘上内容重算这些键再存一次。**只重算交付过的**: 被跳过的那份(用户改过)绝不洗成基线, 否则下次模板升级会连用户的改动一起替换掉 —— 测试里专门断了这条。清单必须**重新加载**, 因为 `scaffoldUI` 里的 `add` 中途已存过一份, 手上的 `inst` 是旧的。
  - **验证有效性**: 把修复临时关掉跑新测试 → 立刻变红并复现出 `xui: app\go.mod 本地已修改, 跳过`(且基线哈希 ≠ 盘上), 开上即绿 —— 证明缺陷可复现、测试真能抓到, 不是走过场。桩必须**真的改写 go.mod**(只记录调用复现不出来)。
  - 另核一件事: 用户问"`project_files` 五项有什么用" —— 读者只有 `initNewProject` 的模板循环一处, 即**只在重跑 `init -n`(需 `--force` 过闸门)时生效**; `add`/`update`/普通 `init` 都不读它。价值是"拒绝覆写 + 说清原因"而非"帮你刷新"(`main.go` 是应用入口、必然被改)。真 binary 自查: `init -n app --no-install` 后五个键全等于盘上哈希, 重跑 `--force` 五次"已一致"。
- ✅ **文档可以点名 add/update(2026-11-03)**:
  - **动机**: 上一批只让 `init` 能刷文档, 于是 ① 只用 `add` 装过组件、没跑 `init` 的项目拿不到这两份文档; ② 想只刷一份也得连组件一起动。
  - **做法**: `cliDocs = {DESIGN.md, AGENTS.md}` + `splitForWrite`(只按**基名**比对, `ui/DESIGN.md` 也认;`--view/--diff` 模式一律不拆, 文档名留给 registry 解析去报错) + `placeDocs(create)`。`add` 传 `create=true`(可创建), `update` 传 `false`(只刷已存在的, 缺了就答"没有 X (它由 xui init 写入; 要单独拿它: xui add X)") —— 与"`update` 不顺手装新组件"是同一条纪律。`lookupComponent` 的报错也指路: `"DESIGN.md" 不是组件, 是 CLI 下发的文档`。
  - **`scaffoldUI` 改为复用 `placeDocs`**(判定/清单只有一条路);顺带去掉 `docContent` 里对 DESIGN 的无意义 `renderUI`(见上一条的更正)。
  - **踩坑**: 把文档写入搬进 `placeDocs` 后**没人建目录了** —— `xui add DESIGN.md` 直接 fatal "写入失败: open ui\DESIGN.md"。修法: `placeDocs` 在真要写之前 `MkdirAll`(与 `install` 同一纪律), 且只在真写过之后才存清单 —— 否则 `update` 空跑会留下一个空 `ui/`。
  - **测试**: `TestNamedDocTargets`(点名 add 只创建被点的那份 + 混合目标 `add DESIGN.md AGENTS.md card` + update 不新增 + 改过的保留 + 基名容忍 + `--view` 模式不拆)。**测不出来的一次**: 该 bug 在 `captureOutput` 里走 `fatal` → `os.Exit`, 测试只剩 `exit status 1` 没有断言输出, 是真 binary 一把跑出来的 —— 记着: `fatal` 路径要用子进程测(如 `TestSkippedSummaryAndStrictExitCode` 那样), 不然静默变成"测试挂了但没说为什么"。
  - **实测**(真 binary 四步): `add DESIGN.md AGENTS.md card` 三个都创建;`update DESIGN.md` 只碰 DESIGN.md(手改的 AGENTS.md 计数不变);空目录里 `update DESIGN.md` → 只提示、不建 `ui/`;`init -n app -p components` 后 `update DESIGN.md -p app/components` → `已一致`。
- ✅ **组件目录的 DESIGN.md / AGENTS.md 纳入清单(2026-11-03)**:
  - **缺口**: 这两份以前是"存在就永远跳过" —— 等于第一次 `init` 那一刻定版, 之后 CLI 改了文档也传不到已有项目。实测证据: `myapp`(v0.8.x 建的)两份 AGENTS 里"API 看第九节"的错指(第九节其实是 **xcgui** 的清单)无法自愈。
  - **做法**: 新增 `placeFile`(内部就是 `classifyPlace` + 同一套措辞 + 写基线), `scaffoldUI` 先加载清单、写 DESIGN.md/AGENTS.md 并记录基线、存清单, 再进 `add()`(它 load 时会带上这两条, 合并保存不丢)。AGENTS.md 的写入从 `initProject` 移进 `scaffoldUI`(加 `dirAgents` 参数标明两种 init 的差异), 顺带补上以前漏做的 `{{UIDIR}}` 渲染 —— **给 `components/` 项目拷过去的 DESIGN.md 里写的还是 `ui/xxx` 路径**。
  - **`theme.go` 按用户要求不纳入**: 它就是给用户改主题的, 保持"存在就跳过, 加 `-o` 也不覆写"。
  - **写位置不变**: 目录级 AGENTS.md 仍只由普通 `init` 写(`init -n` 写项目根那份, 已在 `project_files` 清单里) —— 这次只解决"能不能更新", 不改"放哪"。
  - **更正(下一批核实的)**: 上面"做法"里那句"顺带补上 DESIGN.md 漏做的 `{{UIDIR}}` 渲染、以前拷进 `components/` 还写着 `ui/xxx`"**说过头了** —— `docs/DESIGN.md` 里 `{{UIDIR}}` 出现 0 次, 它的行文按 AGENTS 的规矩本就是同目录裸文件名、与目录叫什么无关, 没有漏渲染这回事。代码里那个 `renderUI(DESIGN 内容)` 是无意义调用, 已删并在 `docContent` 注释写明"需要渲染的只有 AGENTS.md"。
  - **测试**: `TestInitRefreshesTrackedDocs` 覆盖清单登记、改过保留、没改过随版本刷新、`-o` 才丢弃、`theme.go` 永不动、`--dry-run` 不改文件。**三条断言我自己写反过两次**(② 基线设成改动前的哈希 → 落进 placeEdited 而非 placeUpgrade; ③ 用非 -o 分支的措辞去断言 -o 分支; ② 的"刷新后应等于 designData"恰好是正确结果), 修正后全绿。
  - **实测**(真 binary, 六步): 首次 init 创建两份并登记清单; 手改 AGENTS.md 重跑 → `本地已修改, 跳过` 且内容保住; 改仓库 `docs/DESIGN.md` 模拟 CLI 升级 → `更新 ui\DESIGN.md (版本升级)` 且内容刷新; 还原后再跑 → 仍是"版本升级"不说"丢弃"; 未改动重跑 → `已一致`; 两次 init 幂等(16 行"已一致/跳过")。依赖回归 `add alert` / `add --all` + `go build ./ui` 通过。
  - **文档**: DESIGN 第五节加一条"文档也会跟着升级"(随 init 下发), README 未动(等用户对"已有项目"段用哪版表态)。
- ✅ **README 瘦身: 只留"怎么用"(2026-11-03, 461 → 394 行)**: 用户反馈太长。按 AGENTS 的分工把不属于"面向用户的安装与用法"的五类内容清出去, 规则已写进 AGENTS 的 README 条目(避免下次再长回来)。
  - **A 修复史与辩驳口吻(9 处)**: 「以前 `--force` 一个旗同时干两件事…能把你改过的 main.go 无声覆掉」「以前这条是写死 ui/ 的」「init 以前完全没有预览, 只能赌一把」「不会拿"没有差异"糊弄你」「收下 `-j` 又不生效是骗人」「默认 0 是有意的」「不该去改那个不相干模块的依赖」等 —— 全是向 reviewer 解释"为什么改", 用户只需要"怎么做"。**这些改动的过程与实测证据本来就在本文各批条目里, 没有信息损失**。
  - **B CLI 内部机制**: 探测就是拉一次 `registry.json`(6s 超时)、选路逐行打 stderr、拷到一半换源并"粘住"、清单里 `project_files` 这个字段名、`undefined: xui.PresenceOnce` 的成因推导。留结论(会换源、不会留半套、五种情况不动依赖), 细节以代码与 DESIGN 为准。
  - **C 三份重复**: 同一约定在 README / DESIGN 第五节 / `xui --help` 各抄一份(grep 证据: `只压进度输出` 1/1/3, `配套运行时` 2/1/7)。定死归属 —— **开关清单与改动判定以 `xui --help` 为准**, `-j` 的 JSON 字段表只在 README(别处没这份), 原理与坑在 DESIGN, 时间线在本文。README 的选项从 17 条散段改成一张 15 行的表。
  - **D 仓库内部/贡献者内容**: 删 `## 设计要点`(与 DESIGN 一~三节重复)、`## 参考实现` 一节(shadcn 仓库内路径 + "demo/ 是视觉验证程序")、"改任何组件的交互前先看 DESIGN 第四节末"那条**给 AI 的规则**(AGENTS 已有)。**核实过没有信息损失**: AGENTS 记的本地参考实现路径更准确(令牌在 `src/index.css`, README 原写的 `app/globals.css` 是上游仓库的形状), 逐组件的上游来源本文各条目也各有记录; hover 过渡/`xui.Hover`/`Presence` 机制在 DESIGN 有 17 处, 所以 README 的动画段压成"要做的两件事 + 指针"。
  - **E 组件表**: 逐行去掉 `With*/Set*` 全量枚举(59 → 58 行, 合并了 Avatar/AvatarGroup 与 Toggle/ToggleGroup 两行), 每行只留"作用 + 会影响选型的注意点"(ComboBox/Combobox 的命名陷阱、Popover 内嵌自弹窗要 `WithPopoverKeepOnBlur`、Tray 单实例警告、Textarea 随内容长高 vs Input+WithMultiline)。**踩到一次静默缺失**: 整段替换先吃掉了 `Label / Card / Badge / Separator`, 补回后仍漏了 Carousel / Resizable / Tabs / Tooltip —— 是拿 `registry.json` 逐键比对才发现的, 只 diff 上一版看不出来(上一版本身就是错的对照)。教训已写进上面的验证条。
  - **顺带修准一处死指针**: README 原说 DESIGN 第九节是"已验证 API 清单", 实际那是**本库依赖的 xcgui** 的 API 清单, 不是 xui 各组件的选项表 —— 组件完整选项改指向 pkg.go.dev 与组件源码注释(每个 `With*` 都有注释)。AGENTS 的 DESIGN 条目同步写清, 免得以后再指错。
  - **验证**: 组件名与 `registry.json` 逐键比对(65 键, 唯一两个不匹配是 `combobox-search` 这个文件键与顶层 `components` 键, 都不是组件名);表格行都以 `|` 收尾;`#cli` 锚点仍在(第 45 行引用它);被删的辩驳措辞 grep 无残留;`docs/` 与 AGENTS 没有指向被删章节的死引用。
- ✅ **CLI: 一批参数设计问题(2026-11-03, 延续 -o 措辞那条规则)**:
  - **规则**: 参数被接受就必须生效; 措辞不能超出证据; 结果(payload)与进度(progress)分两层。
  - **`-s/--silent` 不再吞掉命令结果**: 新增 `out`/`outRaw`(结果, 永远打 stdout)与 `info`/`infoRaw`(进度, 受 -s 压)。改为结果输出的有 `list` 的组件表与计数、`view`/`--diff` 的源码与差异头、`infoCommand`、`self-update --check` 的对比、`xui version`(原来走裸 `fmt.Println`, 与 -s 的关系不明)。实测修复前 `xui list -s` / `xui info -s` = **0 字节 + exit 0**(脚本读到"什么都没发生"), 修复后分别 3207 / 252 字节。
  - **升级提示改走 stderr**: `notifyAvailableUpdate` 原来 `info` 到 stdout, `xui view button.go > button.go` 会把"有新版"那行写进源码文件。
  - **`--diff`/`--view` 的过滤值匹配不到时不再谎报"没有差异"**: `diffFiles` 把 `matched`(过滤命中)与 `differs`(真有差异)拆开; 一个都没命中 → stderr `没有匹配的文件: X (给组件名或文件基名, 组件见 xui list)`, 命中但相同 → stdout `没有差异`。过滤值新增 `filterHit`: 文件基名、带路径的基名、**组件名**(`--diff=button` 与 `--diff=button.go` 等价, 用户刚在命令里打过它)。
  - **`self-update --check` 不接受版本参数**: 解析抽成 `parseSelfUpdateArgs`, `target + --check` 直接报错(原来把 `v9.9.9` 收下又丢掉)。
  - **`--view`/`--diff` 与 `--dry-run` / `-o` / `--no-install` 互斥报错**: 原来 `add()` 的 switch 让 view/diff 先赢、其余参数静默失效。
  - **`--no-install` 泛化到 add / update / 普通 init**: 字段 `componentOptions.noInstall` 早就存在且已接到 `ensureRuntime`, 只是解析器不给填 —— 于是"写组件但不动 go.mod"这个组合只有 `init -n` 能用。同时删掉"`--no-install` 只用于 `init -n`"那条限制(普通 init 也走 add 路径、也会抬依赖)。
  - **`update` 拒绝 `--all`**: `parseComponentArgs(cmd, args)` 现在带命令名。原来 `update --all` = `allComponents(reg)` 实测把 59 个从没装过的文件"将创建"出来 —— 名字叫 update、行为是 add。省略组件名本来就是"已装的全部"(实测 12 个)。
  - **测试**: `TestSilentKeepsCommandResults` / `TestDiffAndViewFilterTakeComponentName` / `TestParseSelfUpdateArgs` + 扩充 `TestParseComponentArgs`、`TestParseInitArgs`。
  - **文档**: CLI 帮助、README 选项表与「运行时版本」段、DESIGN 第五节(随 init 下发)。
  - **`--force` 与 `-o/--overwrite` 拆开(第二批)**: 以前一个 `-f` 同时干两件事 —— 既是"目录已存在且非空"的闸门([main.go 的 initNewProject](../cmd/xui/main.go)), 又透传成组件 install 的 overwrite, 而 `go.mod` / `main.go` / `README.md` / `AGENTS.md` / `.gitignore` 这五个模板是**无条件 `os.WriteFile`**、不看内容(实测 `--force` 后用户改的 `main.go`、`README.md` 直接没了, 只打一句"覆盖";同批的 `ui/button.go` 反倒有基线判定)。现在:
    - `force` 只开闸门;`-o/--overwrite` 才允许丢弃改动, 普通 `init [ui目录]` 给它 `-f` 直接报错(它本来就不覆盖已有文件, 收下不生效是骗人)。
    - 判定抽成共用的 `classifyPlace`(五种: create/same/upgrade/edited/unknown), `install` 的组件文件与 init 的模板文件走**同一套**;措辞与 `-o` 那条规则一致(基线对得上=升级直接替换, 对不上=本地已修改默认保留, 没基线=只说"无法判断来源")。
    - 模板基线记在同一份 `.xui-manifest.json` 的新字段 `project_files`(键相对项目根) —— `installation.Project` 由 `loadManifest`/`save` 全程保留, `initNewProject` 先写模板+存清单, 再由 `scaffoldUI` 里的 `add` 读回同一份补组件哈希(所以顺序必须是模板在前: `ensureRuntime` 要靠 `go.mod` 已在)。
    - **实测**: 改过 main.go/README.md/ui/button.go 后 ① `--force` → 三处都"本地已修改, 跳过"、内容保住;② `--force -o` → 三处"覆盖 (本地修改已丢弃)";③ 覆盖后基线重建, 再跑一次不再说丢弃;④ 清单里 `project_files` 五个键齐。
    - 测试: `TestInitForceKeepsEditedTemplates`(覆盖三条路径 + 无基线分支)、`TestParseInitArgs` 新增 `-o`/普通 init 的 `-f` 用例;`initNewProject` 签名多一个参数(4 个调用点同步)。
  - **参数一致性第四批(2026-11-03)**: 四项都是"能跑但不合逻辑"的那类。
    - **`list`/`search` 接位置参数当搜索词**: 实测 `xui search button` 原来报"未知参数 button"(最自然的写法走不通)。现在位置参数 ≡ `-q`;给两个不同的词报错不猜(`搜索词给了两个: -q "table" 与 "card"`), 同一个词用 `-q` 和位置各给一次不算矛盾、不报错。
    - **`-v` 的报错要指去向**: 上游 shadcn 是 `add -v, --view`, 我们的 `-v` 归了全局版本号(取舍不变), 但 `xui add -v button` 原来只说"未知选项 -v", 照文档敲的人会以为 `--view` 不存在。现在报错直接给"打印组件源码用 --view[=组件或文件]"。
    - **空值统一当场报错**: 新增 `inlineValue` 与 `nextValue` 的空值分支, 所有带值选项(`-c/--cwd`、`-n/--name`、`-p/--path`、`--view=`、`--diff=`、`-q/--query`、`-l/--limit`、`-o/--offset`)两种写法都不许给空值。原来只有 `-p=`/`-c=`/`-n=` 报错, 而 `--view=` / `-q=` 静默当"没给" —— `--view=` 于是从"只看一个文件"翻转成"全部打印";更狠的是 `xui add button -p ""` 会让 `filepath.Join("", f)` 退化成裸文件名, 把组件**直接拷进当前目录根**, `xui init -n ""` 则静默降级成普通 init。值被 shell 变量吃掉时必须当场失败。
    - **`info` 认 `-p/--path`**: 它原来写死读 `ui/` 且拒绝一切参数, 组件装在 `components/` 的项目里回答 "ui/: 未初始化" 是错的。新增 `parseInfoArgs`(`-p 目录` / `-p=目录` / `--path=目录`), 报告里的目录名跟着实际目录走。
    - 测试: `TestParseListArgsPositionalQuery`、`TestParseInfoArgs`、`TestParseComponentArgs` 补空值与 `-v` 用例;`TestParseListArgs` 里"裸 `button` 应报错"那条旧断言删除(行为已改)。
    - 实测(真 binary): `search button` 出 2 条;`-q table card` / `list button card` 报错;`add -v button` 指向 `--view`;`--view=` / `-p ""` / `-n ""` / `-q=` 全部当场失败;`info -p components` 报 `components: 未初始化`。
  - **第五批: 补上缺的三个能力(2026-11-03, 参数审计到此清完)**:
    - **`-j/--json`**: 新增 `printJSON` 与五个结果类型(`listJSON` / `infoJSON` / `viewJSON` / `diffJSON` / `checkJSON`), 覆盖 `list`/`search`、`info`、`view`、`add/update --view/--diff`、`self-update --check`。**只在真有结构化输出的命令上接受** —— `xui add card --json` 直接报错(写文件那条路径输出的是过程日志, 收下 `-j` 又不生效是骗人), `self-update --json` 必须配 `--check`。两条约定: 空数组给 `[]` 不给 `null`; 过滤没命中用 `matched=false` 表达(否则空 `files` 会被读成"这组件没文件"/"没有差异")。`--diff` 的 `state` 直接复用 `classifyPlace` 的四值(`missing`/`upgrade`/`edited`/`unknown`), 与人类可读输出同源。
    - **跳过汇总 + `--strict` + 退出码**: `install` 返回 `skipped`, 新增 `reportSkipped` 与 `exitPartial=2`。默认只向 stderr 打"共 N 个文件被跳过(…)"并 **exit 0** —— 保守跳过本地改动是这套设计的正确行为, 不该让人类手里的 `xui update` 报错; `--strict`(add/update)才用 2 失败, 与 `fatal` 的 1 分开, CI 由此能区分"跑挂了"与"没更新干净"。`--strict --dry-run` 成立(预览"会被跳过" + exit 2), 那就是 CI 的检查姿势。init 不给 `--strict`(建仓一步的取舍归人), 只给同一句汇总。
    - **`init --dry-run`**: 散装 bool 加到 5 个已经读不出含义(`initNewProject("app", true, true, false)` 里第二个 `true` 是什么?), 收成 `writeOpts{overwrite, noInstall, dryRun}`, `initProject` / `scaffoldUI` / `initNewProject` / `scaffoldProjectWinres` 统一走它。预览纪律 = 不建目录、不写文件、不写清单、不联网、不调 `xc`, 措辞一律"将…"。两个细节: ① 预览不落清单(写清单本身就是副作用, 还会洗掉旧条目); ② 预览新项目时 `ensureRuntime` 原来打"没找到 go.mod, 跳过运行时版本检查"—— 而 go.mod 正是这次"将创建"的东西之一, 这句会让人以为依赖不会被抬, 改成"(预览) 项目里还没有 go.mod; 真跑时会把配套运行时抬到 …"。
    - 测试: `TestJSONOutputShapes`(含 state 四值与 matched 语义)、`TestJSONFlagOnlyWhereItDoesSomething`、`TestSelfUpdateCheckJSON`、`TestSkippedSummaryAndStrictExitCode`(子进程验真实退出码 0/2 且改动保住)、`TestInitDryRunWritesNothing`(目录没被创建 / 不用完成时态 / 不调 xc / 改动保住)。setup 一律 `noInstall`, 单测不联网跑 `go mod tidy`。
    - 文档: CLI 帮助(退出码一段)、README(`-j` 字段表 + 退出码 + `--dry-run` / `--strict`)、DESIGN 第五节(随 init 下发)。
    - 实测: 退出码 6 条路径逐一对上; `init -n demo --dry-run` 前后 `ls` 为空; `info -j` 的 `componentsDir` 跟着 `-p` 走; `--diff nomatch.go -j` 给 `matched=false`。
  - **CLI 参数审计清完**(第一~五批)。唯一**故意不改**的: `-o` 在 `add/update` 是 `--overwrite`、在 `list/search` 是 `--offset` —— 那是 shadcn 自己的形状, 两个命令永不重叠, 跟着改反而破坏对齐。
  - **第六批: 审计时漏掉的两条(2026-11-03)**:
    - **`init -n` 的组件目录名以前写死 `ui/`**: 想要 `components/` 的人拿不到一个目录名正确的完整项目 —— `init -n` 必然先建 `ui/`, 再靠 `xui init components` 补第二个目录、手工删 `ui/`, 而且生成的 `main.go`/README/AGENTS 说的都是 `ui`。改法: `parseInitArgs` 让 `-p` 对两种 init 都成立(`init -n` 时它是"新项目里的组件目录", 普通 init 时与位置参数是同一个东西, 两个都给且不同则报错不猜), `initNewProject(name, uiSub, force, w)` 带着它走。模板里 16 处路径改成 token `{{UIDIR}}` + `renderUI` 渲染 —— **不用再加一个 `%s`**: 这些模板已有 `%s` 在填模块路径/项目名, 再加动词会让 `Sprintf` 参数顺序一眼看不出来, 填错一位就是一份能编译但说假话的 README。校验: 绝对路径与含 `..` 段的目录一律拒(清单落在组件目录里, 跳出去之后就找不到自己的基线)。**踩坑两条**: ① 校验里想用 `filepath.Join("root", uiDir)` 判跳出是无效的 —— Join 会把 `..` 直接消化掉(`../ui` → `ui`), 改成逐段找 `..`; ② 替换文案里写了反引号, 而模板本身是 Go raw string → 字符串被提前终止(本会话第二次踩这条, DESIGN 里就写着"原始字符串不能包含反引号")。
    - **`ensureRuntime` 会改到无关模块**: `findGoMod` 从组件目录**向上**找最近的 `go.mod`, 原来只挡了"找到 xui 仓库自己"这一种。在临时目录 / 别人 monorepo 的上层跑 `xui add`, 会去 `go get` 改那个不相干模块的依赖。判据两条满足其一才动手: 那份 `go.mod` 在当前目录之下(= 用户站着的项目, `-c` 已把 cwd 定在项目根), 或者它本来就 `require github.com/twgh/xui`(在子目录里跑的合法形态保住); 都不满足只提示不动。
    - 测试: `TestInitNewProjectCustomComponentsDir`(components/ 与清单落点 + 不该再生成 ui/ + `ui "app/components"` + 成品里不许漏未替换的占位符 + 之后 `add -p` 指得回去)、`TestEnsureRuntimeDoesNotTouchForeignGoMod`(无关父模块不动 / 父模块本来用 xui 则要抬)、`TestEnsureRuntimeStillBumpsProjectRoot`(守卫不能过头)、`TestParseInitArgs` 补 8 条 `-p` 用例。
- ✅ **CLI: 使用时提示可升级(2026-11-02)**:
  - **做法**: 缓存文件 `UserCacheDir/xui/update-check.json`(拿不到退 Temp)记 `checked_at + latest`; 每条命令正常结束后 `notifyAvailableUpdate(cmd)` 读缓存决定提不提示, 缓存放空/过期(24h)时**先写占位时间戳再派生** `xui __update-check`(隐藏命令, `Start` 不 `Wait`, 输出丢弃)后台跑 `go list -m github.com/twgh/xui@latest` 写回缓存 —— 前台永远不等网络。`fatal` 走 `os.Exit` 不触发 `defer`, 失败的命令不唠叨。
  - **提示条件**: 只有 `isReleaseTag(displayVersion())` 且缓存里也是发布 tag 且更低才提示(devel/go run 构建不提示, 那本来就是开发者在仓库里跑的);文案同时给 `self-update --check` / `xui self-update` / 关闭方式。
  - **关闭**: `-s/--silent`(连派生一起关, 静默模式不该起子进程)、`XUI_NO_UPDATE_CHECK=1`、`self-update` 与 `__update-check` 自身跳过。`xui info` 显示上次检查时间与结果(排查用)。
  - **测试**: `updateNotice`/`shouldProbeUpdate` 表驱动(有新版/已最新/本地更新/devel/没查到/缓存里是垃圾; 从没查过/23h/25h/时间戳在未来)、缓存读写与损坏退回空、路径形状。实跑验收: 首条命令秒回且几秒后缓存出现 `latest=v0.8.0`(证明异步不挡前台)、24h 内时间戳不变(不重复派生)、`XUI_NO_UPDATE_CHECK=1` 与 `-s` 都不写缓存、`info` 打出一行检查信息。提示文案本身由单测断言(devel 构建不会走到打印分支)。
- ✅ **CLI: `xui self-update` 自我升级(2026-11-02)**:
  - **命令**: `xui self-update [版本]`(默认 `latest`, 也吃 `main` / `HEAD` / `vX.Y.Z` / commit 短哈希; 裸 `0.8.1` 自动补 `v`; 认不出的直接报错不跑命令), `--check` 只比较当前版本与 `@latest` 解析到的版本不安装。流程 = `go install github.com/twgh/xui/cmd/xui@<目标>`。
  - **三个坑**: ① **正在运行的 exe 不许就地覆写**(Windows 拒绝访问、Linux 是 `ETXTBSY`)—— `go install` 失败且"跑的就是 GOBIN 那份"时, 把它改名成 `xui.exe.old` 让路再重试一次(改名两边都允许), 失败就报错给手动命令; ② **成功与否以新二进制跑出来的版本为证据**, 不看 `go install` 的退出码; ③ **跑的不是安装出来的文件**(路径里有 `go-buildXXXX`, 即 `go run` / `go test` 的临时产物)时直接拒绝并给出 `go install` 那条命令 —— 只认这个标记, 不认"在临时目录下", 免得把装在自定义 Temp 里的二进制误挡。
  - **GOBIN 与实际运行的那份不是一个文件**时: 照常升 GOBIN 并明确警告"你 PATH 里敲的 xui 是另一份"(本机就是这样: PATH 首选 `D:\GOProject\src\ui\xui\xui.exe`, GOBIN 是 `D:\GOProject\bin\xui.exe`)。
  - **查最新 tag 用 `go list -m github.com/twgh/xui@latest`** —— 查**包路径** `.../cmd/xui` 会一无所获(cmd/xui 不是独立模块), 这条是实测踩出来的; 走 go 命令而不是抓 tag 接口, 用的就是用户机器上的 GOPROXY, 与他 `@latest` 的结果(包括缓存滞后)一致。
  - **测试**: `cliInstallSpec` / `isTempBuildPath` 表驱动; `selfUpdateAt(exe, current, target, checkOnly) error` 是可测内核(seams: `goInstall` / `goListLatest` / `goBinDir` / `readBinaryVersion` 都是 var), 覆盖 默认目标/幂等/版本没变要说代理滞后/改名让路重试/网络失败给手动命令/临时构建拒绝/无 GOBIN 不谎报。实跑三条: `--check` 报 (devel) vs v0.8.0、`go run ./cmd/xui self-update` 被拒、`xui self-update v0.8.0` 升 GOBIN 副本并核对新二进制版本。
  - **文档**: README 安装/升级与命令表、DESIGN 第五节命令表、CLI 帮助、包注释。
- ✅ **`registry.runtime` 语义改成「配套版本」, 每次发布都抬(2026-11-02, v0.8.1)**:
  - **报障**: 用户 `xui update -o` 后 CLI 自己升到 v0.8.1, 但项目 `go.mod` 的运行时只升到 v0.8.0(= 当时 `runtime` 的值), 只能再手动 `go get -u github.com/twgh/xui`。
  - **根因**: 字段按"组件**要求的最低**运行时版本"来定 —— v0.8.1 只改 CLI、没新增运行时符号, 组件确实不需要它, 于是"够用就不抬"。但组件跟 `main` 走, 用户拿的是最新组件, 运行时停在旧 tag 就等于**少拿运行时的修复**(过渡表泄漏这类只改运行时的改动不会触发下限抬升)。
  - **改法**: 语义定为「与 main 上这批组件**配套的**运行时版本」= 刚发布的那个 tag,**每次发布都抬, 只改 CLI 也抬**;CLI 代码逻辑一行没动(目标本来就读 `runtime`), 改的是数据 + 措辞 + 发布流程。顺序仍是"先打 tag 再改 runtime"(写早了 `go get` 拉不到, CI 红)。
  - **同步**: `registry.json` → `v0.8.1`、schema 的 description/$comment、CLI 帮助与 `ensureRuntime` 三条提示语("配套运行时是 …")、README「运行时版本」段、DESIGN 第五节、AGENTS 硬规则、`init` 下发的 README/AGENTS 模板。
  - **验收**: `go vet/test ./cmd/xui` 绿;下限回归 —— `require=github.com/twgh/xui@v0.8.1`(goproxy.cn 还没收录, 用 `GOPROXY=direct`)后 `add --all` + `go build ./ui` 通过, 证明"配套 v0.8.1"编得过。
- ✅ **运行时版本下限: registry.runtime + CLI 自动抬 go.mod(2026-11-02)**:
  - **背景**: 组件跟 `main`, 运行时 `github.com/twgh/xui` 只能按 tag 取 —— 组件用了 main 新增的运行时助手(如 `xui.PresenceOnce`)而用户项目还钉着旧 tag 时, `add` 装得成功、`go build` 直接 `undefined`。v0.8.0 批次就踩过(this: toast/drawer/navigationmenu/screentoast 都要 PresenceOnce, 而 v0.7.1 tag 里没有)。
  - **做法**: `registry.json` 顶层加 `runtime`(起初定成"这批组件要求的最低运行时 tag",**同日改成"与 main 组件配套的版本 = 最新 tag",见上面那条**;schema 同步 + `parseRegistry` 校验格式, 认不出就忽略并警告); CLI 新增 `ensureRuntime(uiDir, floor, dryRun, autoInstall)` —— 从组件目录向上找 `go.mod`, 解析 `github.com/twgh/xui` 的 require/replace, 低于目标就 `go get github.com/twgh/xui@<tag>`(go.sum 一起更新)。挂在 `add`/`update` 的写入分支与 `scaffoldUI`(经 `add`)上。
  - **不动用户依赖的四种情况**: `--dry-run` 只预览、`init --no-install` 只打印命令、`go.mod` 把该模块 `replace` 到本地工作副本(开发者自己管版本)、找到的 `go.mod` 就是 xui 仓库自己(在仓库里跑 CLI)。
  - **CI 才算得住**: `registry-closure` 逐组件编译**不再 replace 到工作区**, 改成 `go mod edit -require=github.com/twgh/xui@<registry.runtime>` —— 组件引用未发布 tag 的运行时符号时直接编不过, 把"先打 tag 再上 main"变成机器保证(以前 replace 写法等于用本地最新运行时, 永远测不出这个洞)。
  - **测试**: `TestMain` 默认把 `goGetModule` 换成桩(单测绝不联网), `stubGoGet` 记录 dir+spec; 覆盖 add 抬版本/dry-run 不动/够新不动/更高不动/replace 不动/xui 自身 go.mod 不动/缺 require 也抬, 外加 `goModRuntimePin` 与 `versionBefore`/`isReleaseTag`/`parseRegistry(runtime)` 表驱动。端到端实测: 钉 v0.7.0 的临时项目 `xui add checkbox` 后 go.mod 变 v0.8.0 且 `go build ./ui` 通过。
  - **顺带**: `fallbackXuiVersion` v0.2.0 → v0.8.0(devel 构建生成项目时的回退版本, 旧值早就编不过)。
- ✅ **CLI: `-o` 不再把纯版本升级说成"本地修改已丢弃"(2026-11-02)**:
  - **背景**: 用户报障 —— 0.7.0 初始化、一个字符没改过的项目跑 `xui update -o`, 每个升级过的文件都打"覆盖 ui\checkbox.go (本地修改已丢弃)", 看着像自己有改动被丢了。
  - **根因**: `install` 的 switch 里 `case overwrite` 排在"本地哈希 == 清单基线(版本升级)"**之前**, `-o` 直接把所有差异文件归到覆盖分支、共用一句"本地修改已丢弃"。
  - **修法**: 把基线判定提到 `overwrite` 之前(未改动一律"更新 (版本升级)", `-o` 也是这句);`-o` 覆盖分支再按**有没有该文件的基线**分两种措辞 —— 基线对不上才说"本地修改已丢弃", 没基线(老项目 / 手删清单)说"无安装清单基线, 无法判断是否改过";默认跳过分支同样把原来"整份清单为空"的判定改成逐文件 `inst.Files[f]` 缺失, 避免"清单里没这个文件"被说成"你改过"。`--diff` 的来源标注同步。行为不变(该写的照样写), 只改判定顺序与措辞。
  - **测试**: 新增 `TestOverwriteWordingTellsUpgradeFromLostEdit`(干净项目 `-o` → 版本升级且无"已丢弃" / 真改过 `-o` → 本地修改已丢弃 / 无基线 → 默认只说"无法判断来源"、`-o` 只说"无安装清单基线"), 用新加的 `captureOutput`(换 `os.Stdout`+`os.Stderr` 到同一 pipe)抓文案;`install` 的函数注释与 README「更新时的判定」表同步成五种本地状态。
  - **验收**: 真实场景复现 —— 用 v0.7.0 的 registry 装 `checkbox radio`、再拿本地 main 跑 `update -o`, 输出改为"更新 ui\checkbox.go (版本升级)";`go vet/test/build`(amd64+386)全绿。
- ✅ **过渡条目不再随交互累积(2026-11-02)**:
  - **运行时**(`presence.go` / `anim.go`): 新增 `xui.PresenceOnce(hOwner, visible, dur, apply, done)`(键自分配、到位即回收)与 `xui.DropPresence(hOwner, key)`;导航菜单换页、抽屉拖拽回弹改用一次性键(固定键会命中已到位条目、动画只跑第一次);`newHoverOwner` 给宿主窗口挂 `WM_DESTROY` 摘表(只删表, 不对销毁中的句柄 `KillXCTimer`);通知被超量挤出 / `Clear` 直接销毁时用 `DropPresence` 清自己那条;图表宿主窗口的悬停监听登记同样在 `WM_DESTROY` 摘(残留失效句柄会让句柄复用后的新窗被误判"已挂过回调", 悬停提示再也不弹)。见 DESIGN 183。
  - **验收**: 探针 2 万次 `PresenceOnce` 只多几十个对象(同参数 `Presence` 多 2 万个);`go vet/test/build ./...`(amd64+386)与逐组件安装 + `add --all` 编译全绿。
- ✅ **ColorPicker 颜色选择器 + CLI `xui version`(2026-11-02)**:
  - **组件**(`components/colorpicker.go`): 触发区(色块 + `#rrggbb` + chevron, 走 `BnClick` —— 原生 Button 不向外部派发 `LButtonUp`)+ 弹层(SV 方块 / 色相条 / 预设色 / Hex 输入), 颜色以 `#rrggbb` 进出;`NewColorPicker` + `WithColorPickerValue/OnChange/Width/Placeholder/Presets` + `Value/SetValue/OnChange/Open/Close/Toggle/IsOpen`。圆与环一律两层实心 `FillEllipse`(1~2px 描边落在半像素上会毛边);选中环与圆点留 1px 缝(环色与色块常同色, 否则糊成"双圈甜甜圈");色相位置先夹进条内(否则 `h>360` 被 `math.Mod` 折回另一端);预设色列宽浮点四舍五入(整数截断让圆心列列左移)。见 DESIGN 184 前后与第四节组件表。
  - **CLI**: `xui version`(同 `-v` / `--version`)离线可用、不拉数据源, 本地构建带短 commit 与 `+dirty`。
  - **demo**: 表单5 新增「颜色选择器」卡。registry: 新增 `color-picker`(闭包 colorpicker / popup / input / scrollbar / group / textopt)。
  - **验收**: `xui add color-picker` 单独安装 + `go build ./ui` 通过;`add --all` 编译通过。
- ✅ **demo 主题面板 + 标题栏双击 / 弹层嵌套修复(2026-11-02)**:
  - **demo**: 亮暗切换独立成太阳/月亮按钮, 主题按钮改弹 Popover 卡片 —— 可点选的主题色块 + `ColorPicker` 自定义主色(前景色按亮度自动黑/白), 色块选中态用中央对勾而不是外圈环。
  - **Popover**(`popover.go`): 新增 `WithPopoverKeepOnBlur(bool)` —— 内容里嵌 ColorPicker 这类"自己会弹窗"的控件时必须开(否则一点宿主卡片先失焦关掉);打开时给**宿主窗口**挂 `WM_LBUTTONDOWN` 关卡片(元素级 `LButtonDown` 在点元素时被消费、窗口收不到, 只有空白区能关), 点触发区那一下先关再被 `RecentlyClosed` 挡下 = "再点一下关掉"。见 DESIGN 184。
  - **Window**(`window.go`): 标题栏自定义按钮上的双击不再把窗口顶成最大化 —— 元素级 `LButtonDBClick` 吞双击 + `pointOnCaptionButton` 矩形命中兜底(双击序列里 `GetStayEle` 可能已归零, 光靠它拦不住)。
- ✅ **DataTable 双击编辑 + Radio/Checkbox 编程接口(2026-11-02)**:
  - **DataTable**(`datatable.go`): `WithDataTableEditing()` 双击原地编辑(Enter/失焦提交、Esc 取消)、`OnCellEdit(fn(row, col int, old, next string) bool)` 校验回调、`DataColumn.ReadOnly`;自绘列也可编辑(编辑期间该格让位);编辑框挂窗口绝对定位, 单元格矩形由表头缓存 + 行高取模反推(不依赖滚动偏移);`releaseEditor` 用 `XC_IsHELE` 守卫 —— 窗口退出已先行销毁编辑器, 不再报"句柄无效"。
  - **Radio / Checkbox**: 新增 `Select()` / `SetSelected()` / `Toggle()` 与 `SetBackdrop(fn(*Theme) Color)`(整卡可点时需要: 指示器设鼠标穿透、由卡片统一调);`table.go` 配合小改。
  - **demo**: 表单5 新增 RadioGroup 选择卡片与整卡可点字段卡片;数据页表演示双击编辑。DESIGN 185(`LayoutEle` 默认鼠标穿透, 当命中区必须 `EnableMouseThrough(false)`)。
- ✅ **勾号 / 减号改 `DrawLineF`(2026-11-02)**: 整数版 `DrawLine` 把比例算出的亚逻辑像素坐标统一截断(且方向一致), 勾号整体偏左、右臂偏短, 减号不对称;改按 lucide 24 网格映射到 14px 内容区、减号两端对称取中心。线宽是物理像素、不随 DPI 缩放, 与坐标口径不同。DESIGN 186。
- ✅ **ScreenPopup 屏幕右下角常驻弹窗(2026-11-01)**:
  - **组件**(`components/screenpopup.go`): 复用 `popoverPanel` 的非激活阴影窗承载任意 xui 内容; `NewScreenPopup(hAnchor, WithScreenPopupWidth/Height/Offset/Padding/Gap/Duration/Focusable/CloseButton/Header/OnOpenChange/Content)` + `Show/Hide/Toggle/IsOpen/FitContent/Content/Handle/Destroy`; 自动高(隐藏时离屏量、打开中新增内容后同步父布局再 resize)、贴锚点窗口所在显示器工作区右下角、随锚点销毁(主动 Destroy 反注册监听)、主题切换重量重排; 不是通知: 单面板、手动显隐、不堆叠。
  - **退场淡出**: `Hide`/X/超时走 `popoverPanel.HideWithFade`(100ms 元素 alpha + 阴影浓度同步); 逻辑状态立即关闭, 窗口淡出结束才隐藏。阴影是窗口级绘制、不跟元素 alpha, 屏幕弹窗用 `SetShadowInfo` 的 depth=50×v 一起降(整窗 alpha 对复用透明窗会冻结, 不能用)。进出场共用面板的 `presenceKey`, 淡出中途重新 Show 从当前透明度反向淡入; `alwaysAnimate` 禁用快速重开跳过淡入, 保证退场有过渡可反向; `Destroy` 仍立即销毁。
  - **焦点配置**: 默认 `activate=false` 不抢焦点; `WithScreenPopupFocusable(true)` 传正常窗口(去 `WS_EX_NOACTIVATE`/不拦 `WM_MOUSEACTIVATE`), 但 Show 与离屏量高仍用 `SW_SHOWNOACTIVATE`、不调 `SetForegroundWindow` —— 显示不抢前台, 用户点击弹窗才由系统激活, 之后输入框可用、ESC 收起。配套 `keepOnBlur=true`: 失焦不自动关, 保持常驻面板语义。
  - **自动关闭与标题栏**: `WithScreenPopupDuration` 每次 Show 重新计时、Hide/Destroy 取消, 用自增 `autoGen` 丢弃已取消但已排队的旧回调(避免把重新显示的弹窗提前关掉), 计时不因悬停暂停; 到点走 Hide, 所以 `OnOpenChange(false)` 必触发。`WithScreenPopupHeader(title)` 在内容前建 28 高标题行(Weight 标题 + 28x28 ghost X 调 Hide), `WithScreenPopupCloseButton` 只要 X; `WithScreenPopupOnOpenChange` 给外部按钮/文案同步开关状态。
  - **修复(DESIGN 181)**: `WS_EX_NOACTIVATE` 只能拦住系统的点击激活, xcgui 处理元素点击时仍会把窗口顶到前台 —— 非激活弹层补 `WM_MOUSEACTIVATE → MA_NOACTIVATE`; 显示后与点击后 `GetForegroundWindow` 都保持主窗口。该修复同时覆盖 HoverCard / NavigationMenu / 菜单子层。
  - **demo**: 反馈2 新增「屏幕弹窗」卡(显示/隐藏、弹窗内「加一行」调 FitContent 向上长高、弹窗内「关闭」), 弹窗用 10s 自动关 + 「传输队列」标题 + X; 同卡加「可输入弹窗」演示 `WithScreenPopupFocusable(true)`(显示不抢前台, 点击后激活可输入, 常驻不超时, ESC/X/按钮收起); 两个弹窗互斥显示, 卡片状态文案随开关更新, 文案说明与 ScreenToaster 的分工。
  - **验收**: 150% DPI + 双屏真实鼠标 —— 主/副屏都贴对应工作区右下角(卡片距工作区 24, 窗口外含 5px 阴影带); 显示/点按钮/追加行/关闭后前台始终是 demo 主窗口; 追加 1 行窗口高 219→261 物理像素、底边不动, 新行可见; 标题 + X 显示在右上角, 10s 内点 X 立即收起、等待约 10s 自动收起, 两种关闭后前台都仍是 demo 且卡片按钮/状态同步(修复了 Button.SetText 后不重绘); 「可输入弹窗」显示后前台仍是 demo(不抢焦点), 点输入框后弹窗才激活并可键入文字, 点主窗口不自动收起(失焦保活), ESC 与 X 都能关且关后前台回到 demo; 关闭播放 100ms 淡出(XUI_ANIM_SCALE=5 逐帧抓到半透明中间帧); 阴影带亮度随内容一起升到背景值(外圈不再残留), 重新打开后阴影恢复; 淡出中途重新 Show 能反向淡入且 1.2s 后仍可见(旧退场回调没把窗口藏掉); 打开中切暗色卡片同步换色; 宿主销毁时弹窗一起销毁; `go build/vet/test ./...` 全绿; `xui add screen-popup` 闭包与 `add --all` 编译通过。
  - registry: 新增 `screen-popup`(files: screenpopup.go + popup.go; deps: button + label —— 标题栏的标题/关闭按钮, popup.go 是弹层底座、不是独立组件 key)。
- ✅ **winres 资源 + CLI init 生成图标资源(2026-11-01)**:
  - **demo 资源重新生成**: 删掉 `rsrc_windows_arm64.syso` —— xcgui 不支持 ARM, 只出 386/amd64; `demo/winres/winres.json` 用 `xc res init` 重新生成, 补齐 `RT_MANIFEST`(per monitor v2)与 `RT_VERSION`(项目名/文件名), 不再是只有图标的精简版; 自定义图标放回 `winres/icon.ico` 后执行 `cd demo && xc res make --arch amd64,386` 重出 syso。
  - **CLI `init -n` 生成 winres**: 新文件 `cmd/xui/winres.go` —— 内嵌默认图标(`cmd/xui/assets/app.ico`) + `xc res init` 同款完整 winres.json 模板(项目名按 JSON 转义后填进 RT_VERSION); 生成项目时写 `winres/`(已存在则跳过), 再执行 `xc res make --arch amd64,386`(只有 386/amd64)。本机没有 `xc` 时自动 `go install github.com/twgh/xc@latest`, 到 `GOBIN` / `GOPATH/bin` 找可执行文件; `--no-install` 不联网装, 只提示命令。
  - **测试**: `winresMake` 留测试缝(桩不依赖本机 xc), 断言 `winres/icon.ico`、完整 `winres.json`(三段 + APP + 项目名)、`rsrc_windows_{386,amd64}.syso` 且无 arm64; `projectWinresJSON` 覆盖项目名转义; `findXc(false)` 覆盖 xc 缺失报错。
  - **窗口图标一起设**: 只在托盘上设图标不会同步任务栏 / Alt+Tab / 任务栏预览窗。`xui init -n` 生成的 main.go 内置 `setWindowIcons`(从 APP 资源取 24x24 小图标与默认大图标)并按下面方式调用, demo 也补了同样两行; 生成的 README / AGENTS.md、仓库 README 与 DESIGN 180 写清三者分工与 `LR_SHARED` 不释放。
  - **配套**: 生成的 README / AGENTS.md 增加「图标与版本信息」小节(换图标 → `xc res make --arch amd64,386`, 没装先 `go install`, 只支持 386/amd64, 资源名 APP); 仓库 README 的 `init -n` 说明与 `--no-install` 口径同步。
  - **验收**: `go build/vet/test ./...` 全绿; 临时目录实跑 `xui init -n myapp` 生成 winres + 两个 syso, `go build` 出的 exe 带图标; 删掉 PATH 里的 xc 后重跑会走 `go install` 兜底。
- ✅ **托盘 + 主题跟进(2026-11-01)**:
  - **托盘左键保持最大化**: `showHost` 不再无条件 `SW_RESTORE`; `w.IsMaxWindow()` 时走 `SW_SHOWMAXIMIZED`, 否则 `SW_RESTORE`(见 DESIGN 178); demo 菜单的「显示主窗口」同样处理。实测: 最大化时点托盘图标仍最大化; 最大化 → 最小化 → 点托盘, 回到最大化并前台。
  - **托盘图标改用 exe 资源**: `demo/winres/winres.json` + `icon.ico` 由 `xc res make` 生成 `rsrc_windows_{386,amd64}.syso`(随仓库提交); 启用托盘时 `LoadImageW(GetModuleHandleW(""), "APP", IMAGE_ICON, 0, 0, LR_SHARED|LR_DEFAULTSIZE)` + `WithTrayIconHandle`, 删掉原来"写临时 .ico"的路径, 也顺带给 demo.exe 带上图标。README / DESIGN 178 补充该用法。
  - **主题色演示**: 标题栏调色板菜单新增「主题色」分组, 基础1 右列新增「主题色」卡片(默认/蓝色/绿色/紫色/玫瑰/橙色); `applyDemoTheme` 克隆内置主题后替换 `Primary / PrimaryForeground / Ring / SidebarPrimary / SidebarPrimaryForeground`, 亮/暗各一套, 全窗口实时生效(不重建窗口); README「换肤」补运行时改主色的代码示例。
  - **验收**: 亮色点"蓝色"后 Default 系列/步进器箭头/加载条变蓝, 切暗色仍保持蓝色且前景色自动换成深色; 卡片标签与菜单勾选同步; 托盘图标出现在任务栏溢出区(UIA 按名 `xui demo` 找到并可左键); 最大化/最小化恢复如上; `go build/vet/test ./...` 全绿; `xui add tray` 闭包与 `add --all` 编译通过。
- ✅ **ScreenToast 屏幕通知 + Tray 托盘图标/菜单 + demo 主题菜单**(2026-11-01):
  - **ScreenToast**(`components/screentoast.go`): 每条通知一个 `Window_Transparent_Shaped + WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE` 置顶小窗, 贴显示器工作区右下角; `NewScreenToaster(hAnchor, WithScreenToastWidth/Offset/Gap/Duration/MaxVisible/RichColors/CloseButton)` + 与 Toast 同形的 `Show/Success/Error/Warning/Info/Loading/Promise/Update/Close/Clear/Count`。卡片排版/绘制复用 `toast.go`; 过渡挂锚点窗口(`withToastAnimOwner`)、转圈走 `xui.SpinDetached`(定时器挂宿主), 悬停整组暂停; 宿主 `WM_DESTROY` 自动销毁全部在屏小窗; 主题切换重新量高/重排(见 DESIGN 177)。
  - **Tray**(`components/tray.go`): `NewTray(hWindow, iconPath, tips, WithTrayMenu/WithTrayIconHandle/WithTrayLeftClick)`; 左键默认还原宿主, 右键弹出每次重建的 themed 菜单(`Menu.PopupAtCursor`); 监听 `TaskbarCreated` 在 Explorer 重启后恢复; HICON 所有权/宿主销毁清理已处理; xcgui 托盘 API 是进程级全局状态(见 DESIGN 178)。
  - **demo**: 反馈2 新增「屏幕通知」卡(成功/错误/警告/信息/带动作/常驻长文本)与「系统托盘」卡(启用/停用托盘; 菜单: 显示主窗口 / 发送屏幕通知 / 退出); 标题栏调色板按钮弹出「外观: 亮色/暗色」主题菜单; `demo/winres/icon.ico` 经 `xc res make` 编进 exe, 托盘用 `LoadImageW` 取 HICON(见下方 winres 条目)。
  - **修复(DESIGN 179)**: 任何承载 xui 组件的窗口都会经 `AttachWindow` 登记进主题重绘表, 但此前只有 AppWindow 在销毁时反注册 —— 关掉屏幕通知后再切主题, `XWnd_Redraw` 打到失效句柄会弹模态错误框并退出。现在 `AttachWindow` 在窗口 `WM_DESTROY` 时自动 `UnregisterWindow`, `SetTheme` 再用 `XC_IsHXCGUI` 过滤失效句柄。验证: 通知在屏时切亮/暗(窗口与通知同步换色)、关掉通知后再切主题不再崩。
  - **验收**: 150% DPI 真实点击 —— 成功通知 4s 自动关; 常驻通知手动关; 悬停 6s 不关闭、移开后 4s 关闭; 两条通知堆叠(新的在下); 切暗色时通知窗口同步变暗; 启用托盘后图标进入任务栏溢出区, 通过 UIA 触发图标左键回调(showHost 日志), 托盘菜单(同一 builder 弹出)亮/暗主题渲染与条目正确。`go build/vet/test ./...` 全绿; `xui add screen-toast` / `xui add tray` / `add --all` 闭包编译通过。
  - registry: 新增 `screen-toast`(files: screentoast.go; deps: toast)、`tray`(files: tray.go; deps: menu)。
- ✅ **CLI: 下发文档不再写死 `ui/` + 新增 AGENTS.md**(2026-10-31):
  - **DESIGN.md 目录无关**: 源码引用从 `ui/button.go` 改为同目录裸文件名(`button.go`), "ui/ 目录"表述改为"组件目录(默认 ui/)"; 目录名是 `widgets` / `src/widgets` 时下发的文档不再指向不存在的 `ui/`。
  - **AGENTS.md**: `xui init -n` 在项目根生成(运行/目录/添加更新/最小用法/改组件规则); `xui init [目录]` 在组件目录生成(不写死目录名)。README 归用户、可能被改写, 所以用法留一份在 AGENTS.md。已存在则跳过, 不覆盖用户改动。
  - **配套**: 生成的 README 模板、仓库 README、DESIGN 第五/六节同步; 仓库 AGENTS.md 增加"DESIGN 文件引用写裸文件名"与"用法说明随 init 下发 AGENTS.md"两条规则。
  - **验收**: `go test ./cmd/xui` 新增 `TestInitProjectDocsAreDirAgnostic`(自定义目录名、DESIGN 无 `ui/*.go`、重复 init 不覆盖 AGENTS.md)与 init -n 的 AGENTS 断言; 临时项目实跑 `init -n myapp` + `init widgets`。
- ✅ **DESIGN 面向用户的清理**(2026-10-31, 用户反馈"下发用户的文档不要写内部验证文本"):
  - 删掉内部验证过程与仓库流程: 探针/demo/验收与验证清单/上游参考同步/pnpm/测试文件名/本地参考实现路径/报障与录屏口径(约 250 行);"验证来源"列改"参考";组件路径统一 `components/` → 同目录裸文件名。
  - 合并重复的"实测"口径说明;保留全部技术结论、踩坑与第八节编号(源码注释依赖)。AGENTS 的 DESIGN 写作规则同步补充"不写内部验证过程"。
- ✅ **批次 12: Input Group 复合 addon**(2026-10-31, 参 shadcn input-group 文档 + 本地 App.tsx 对照卡):
  - **前置**: 本地 shadcn 参考实现补 `input-group.tsx` / `textarea.tsx` / `kbd.tsx` / `spinner.tsx`, 表单页加「输入组」对照卡(图标 / 文字 / 按钮 / Kbd / 下拉菜单 / Spinner / Textarea / block-start-end / 按钮尺寸档 / 禁用与校验), `pnpm build` 通过。
  - **组件**(`components/inputgroup.go`): 新增 `InputGroupAddon.MenuButton(text, icon, build)` —— 组内下拉菜单触发(默认 ghost + icon-xs, 菜单 click 弹出, 只 build 一次);addon 支持一个容器里放多个控件;按钮 xs/sm/icon-xs/icon-sm 的**布局高按尺寸档**算(24/28, 此前后者被压成 23);外扩量改加在 **addon** 上(含按钮 5 / 只含 Kbd 2, 每个 addon 只外扩一次)。
  - **踩坑(DESIGN 165)**: ① 外扩照搬到每个按钮上会让多按钮 addon 外扩两次(输入框多占 10px、按钮间距少 5px);② `fixButtonBox` 写死 24 会把 sm 按钮压扁;③ MenuButton 的菜单只 build 一次以保留勾选状态, 原生菜单是模态的(与 radix 非模态下拉不同)。
  - **demo**: 表单1 输入组卡新增两行(下拉菜单 / 一个 addon 里两个按钮 sm + icon-sm)。
  - **验收**: 探针矩形(单按钮 addon 溢出 5 / 按钮右缘距边框 3、两按钮只溢出 5 且间距 8 / 元素高 29、Kbd 溢出 2 且距边框 6、菜单点击出现 All/Files)+ demo 亮/暗 150% DPI(无裁切/重叠)+ `go build/vet/test ./...` 全绿;`xui add input-group` 闭包与 `add --all` 编译通过。
  - registry: `input-group` 的 deps 加 `menu`(MenuButton)。
- ✅ **批次 11C: 分面筛选升级成 shadcn 的 Popover+Command 面板**(2026-10-31, 对齐 tasks 的 `DataTableFacetedFilter`):
  - **面板**(新文件 `datatablefacet.go`): `Popover + Command` 取代原来的 `Menu.Check` —— 顶部搜索选项、条目左侧勾选框 + 右侧**当前行数**、点一项**不关闭**(连续多选)、底部"清除筛选"按钮;计数用 `facetCounts`(统计时排除本列自身的分面筛选, 与 TanStack `getFacetedUniqueValues` 同口径)。
  - **触发区自绘**(`facetTrigger`): Button 画不出"列名 | 竖分隔 | 选中胶囊", 改成自绘元素 —— 有选中铺 secondary、无选中 1px 描边; 胶囊最多两个, 3 个起收敛成"N 个已选"。
  - **Command 扩展**: `WithCommandItemChecked / WithCommandItemCount` + `SetChecked/SetCount`(勾选框 16px 在图标左侧, 数量右对齐 12px muted)。
  - registry: `data-table` 的 `files` 加 `datatablefacet.go`, `deps` 加 `command`、`popover`。
  - **验收**: 打开面板 → 4 项带图标与计数(7/7/6/4)、搜索框可用; 勾 1 个 → 表实时过滤、触发区出现一个胶囊; 勾 3 个 → 胶囊收敛"3 个已选"、面板不关闭、页脚同步; `go build/vet/test ./...` 全绿; 单装 + `add --all` 编译通过。
  - **用户复看后的修**: ① 面板改 `PopoverAlignStart` 左对齐; ② **每列只建一个常驻 Popover**(原来每次点击都新建 → 触发元素上的 toggle 回调越挂越多, 表现为"越开越闪"); ③ `WithCommandItemCheckable` 让未选中也画空框(否则勾选行左移 24px 对不齐); ④ Command 高度去掉多余的 heading 高度(最后一项与清除按钮之间原来空一大块); ⑤ 清除按钮改固定宽 + 左右 6px 边距(原来 `ItemFill` 让悬停底色溢出弹窗); ⑥ 列表与清除按钮之间补 1px 分割线。
  - **第二轮复看**: ⑦ 清除按钮高度 26 → 32(悬停底色太窄看着不舒服), 左右边距 6 → 4; ⑧ **Command 的布局项高度一起钉住**(`ItemSize`), 只设宽时分割线被推到更低的位置 → 那才是"空白太大"的真因, `cy = 输入行 + 条目数×条目高 + 8`(再少就裁行 + 出滚动条); ⑨ **未勾选时不显示清除按钮**(与 shadcn 一致), 收起要连布局项一起归零, 高度用新加的 `Popover.SetContentHeight` 直接改窗口(打开中 Refresh 不改高度, 会整块空白); ⑩ 点清除按钮时 `Command.ClearActive()`, 否则列表里那行还留着高亮(看着像两处都有焦点)。
  - **第三轮复看(2026-11-01, 前两轮未解决)**: 根因是**结构错了**, 不是数值没调准 —— 清除行是 Popover 内容流里的独立 Button, 内容区给每个子项自带 8px 间距, 于是"最后一项→分割线""分割线→按钮"各多 8px; 窗口高度再按内容补偿, 补少了按钮被裁到底边、补多了又变成白带。这轮把清除行搬进 `Command` 列表内部(内部分隔线 + 居中条目, 对应 shadcn 的 `CommandSeparator` + `CommandGroup` + `CommandItem`), 去掉 `p.line` / `p.clear` 与全部高度余量 → 最后一项→分割线 4、分割线→清除行 4、清除行→底边 4(150% DPI 原生截图逐像素量过)。
  - **第三轮工具栏**: 顺序对齐 tasks = 定宽筛选框(`dataTableSearchW=200`) → 分面触发区 → 重置(文字 + X) → `Weight` 撑开 → 视图。原来搜索框 `Weight 1` 会让胶囊变宽时输入框变窄、分面按钮整体左滑, 弹层却锚在原位 → 就是"状态按钮跑到弹层左边"; 现在输入框定宽, 分面左缘固定、只向右长, 视图始终贴右。窗口缩到 960 时固定 200 + 分面 + 重置会把视图挤出卡片, 所以工具栏 Size 时按可用宽度收窄搜索框(下限 100); 1200 宽下仍恒为 200, 已在 960/1200 双向缩放验证(窄窗收窄时触发区会随输入框移动, 这是固定宽与视图贴右不可兼得时的取舍)。
  - **第三轮行为**: 清除行现在走 CommandItem 的选中路径, 点它**只清选择、不关闭面板**(shadcn 的 Popover 语义), 搜索框焦点保持; 面板高度随底部行显隐前移/后移, 不再出现"按钮贴底"。`Command.ClearActive()` 的 `setActive(-1)` 修复保留(鼠标从底部行移回列表时最后一项能重新高亮)。
- ✅ **批次 11D: Data Table 多列排序 / 表头悬停 / 列拖拽 / 跨页全选**(2026-11-01, 补齐 11C 之后的待办):
  - **多列排序**: `DataSortKey{Col,Dir}` + `SetSortKeys/SortKeys/OnSortKeysChange`;`SetSort/Sort` 保留为主键入口。列头菜单仍是单列排序(勾选另一列 = 换成只按它排), **Shift+点击可排序表头**直接在该列上循环 升序 → 降序 → 取消, 未排序时追加为次键; 次键箭头后画次序数字(10px 加粗, 避免 9pt 常规被画成宋体)。`dataTableViewKeys` 按主 → 次逐键比较, 全键相等保持稳定顺序。
  - **跨页全选**: 表头勾选框普通点击仍只切**整页**; **Shift+点击**跨页勾满/清空当前筛选结果, 新增 `SelectAllFiltered/AllFilteredSelected`; 页脚"已选 x / y"本来按筛选结果统计, 跨页后自动正确。
  - **表头悬停 / 列拖拽**: 原生 List 表头只改 `NState` 不重绘 → 订阅表头元素 `MouseMove/MouseLeave/LButtonDown/LButtonUp`, 命中缓存的 `hdrRects` 后改状态并 `Redraw(false)`; 悬停底色走 `xui.HoverItem(表头句柄, 列号, active)` 过渡。只有可排序 / 可隐藏的表头可交互(勾选列、`NoHide` 操作列静止)。4px 起判拖拽, 目标边界画 2px 主色竖线; **起拖时**就置"吞点击"标志, 防止原生补发 `Header_Click` 弹出菜单; 松手在表头外时用左键位收掉残留状态。`DataTable.order` 保存全部列的显示顺序(隐藏列位置也保留), 拖拽只重排它, `DataColumn` 下标与排序 / Facet / Render 语义不变; "视图"菜单按当前显示顺序列。
  - **修复(用户复看)**: 悬停表头时第 N 行数据也跟着亮 —— 表头绘制回调收到的 `hEle` 是 List 本体, 与行悬停共用 `HoverItem` 的 owner+索引键; 改用 `GetHeaderHELE()` 的表头元素句柄做 owner, 行/表头键空间分开(见 DESIGN 176)。
  - **行拖拽未做**: 原生 List 没有行拖拽; 排序列 / 筛选 / 分页把"看到的位置"与原始行下标解耦, 拖拽落点要由业务定义(是否改原始顺序、跨页与隐藏行怎么算), 组件不替调用方决定 —— 见 DESIGN 差异表「DataTable(未做)」。
  - **文档**: DESIGN 新增 176(表头事件/拖拽), 162/170 更新, 交互语义表同步; 差异表只留下行拖拽与"拖拽不改列下标"两条。
  - **验收**: 150% DPI demo 真实鼠标 —— 悬停标题出现内侧圆角底色; Shift+点标题(升)+ Shift+点状态(次键, 显示 ↑2); 普通点标题菜单仍锚在该列下方; 把状态列拖到标题左侧立即换位且不弹菜单; 隐藏/显示优先级后拖拽顺序保留; Shift+点表头复选框页脚"已选 24 / 24 行"、翻到第 3 页仍全勾, 再 Shift+点清空; 普通点表头复选框只选当前页(4 / 24)。`go build/vet/test ./...` 全绿。
- ✅ **批次 11B: Data Table 分面筛选 / 列头菜单**(2026-10-31, 参 shadcn tasks 示例的 `data-table-faceted-filter.tsx` / `data-table-column-header.tsx`):
  - **分面筛选**: `DataColumn.FilterOptions []DataFilterOption{Label,Value,Icon}` —— 工具栏按列生成 `⊕ 列名` 按钮, 点开 `Menu.Check` 勾选清单(带图标), 有选中时按钮文字变"列名 N"并切 `ButtonSecondary`, 菜单底部有"清除筛选"; 判定 = 单元格**原始文本**相等, 列内 OR / 列间 AND, 与关键字叠加; 新增 `SetFacet/FacetValues/ResetFilters/OnFacetChange`, 关键字或分面生效时工具栏出现"重置"。
  - **列头菜单**: 表头点击不再循环排序, 改为弹 `升序 / 降序 / 隐藏` 菜单(`Menu.PopupAtCursor`); 升序降序是勾选项, 再点当前项 = 取消排序(保住原来的"升→降→取消"); `DataColumn.NoHide` 让行操作列不出现在"隐藏"里; 删掉 `nextDataSort`。
  - **踩坑(DESIGN 171/172)**: 分面筛选必须比原始文本(自绘列的原生文本是空串, 拿渲染结果没法比); 布局项"收起"要 `ItemWidth(0)` + `ItemMargin(0,…)` + `Show(false)` 三件套, 展开后按 `naturalWidth` 重算并 `XEle_AdjustLayout`, 只 `Show(false)` 会留一块空白。
  - **demo**: 状态列挂 4 个筛选项(待办/进行中/已完成/已取消, 带状态图标)、优先级列挂 3 个(高/中/低, 带箭头), 操作列 `NoHide`; 卡片描述改为"分面筛选 + 列头菜单"。
  - **验收**: 真实鼠标点 `状态` → 菜单四项带图标 → 勾"进行中" → 表只剩 7 行、按钮显示"状态 1"、"重置"出现、页脚"筛选出 7 / 24 行"、页码回 1/1; 点列头 → 升序/降序/隐藏菜单(锚在该列正下方);`go build/vet/test ./...` 全绿;`xui add data-table` 闭包 + `add --all` 编译通过。
  - **后续修复(用户报障, DESIGN 173/174)**: ① 工具栏在菜单里改状态后残留旧像素(旧输入框边框线 / 取消勾选后少一段边框) —— 模态菜单关闭时会用自己保存的屏幕盖回去, 改成菜单返回后 `updateToolbar() + Redraw(true)`;② 列头菜单跟着鼠标松开处跑 —— 表头绘制时缓存 `RcItem`, 用 `Menu.PopupAtElementRect` 锚到该列下方, 并试过 `LButtonDown` 按下即弹(会被随后的松开消息立刻关掉, 已回退)。
- ✅ **批次 11A: Data Table 富单元格 / 行勾选 / 列显隐**(2026-10-31, 参 shadcn tasks 示例的 `columns.tsx` / `data-table-view-options.tsx`):
  - **Table**(`components/table.go`): 加 `TableColumn.Align`(原生文本对齐)、`Table.SetColumns(cols)` 换列结构(`DeleteColumnAll` 后**必须重建行适配器**, 见 DESIGN 169)、内部钩子 `onCellExtra`(按单元格的扩展绘制, 见 167/168)。
  - **DataTable**: `WithDataTableSelection`(勾选列 + 表头整页全选 / 不确定态, 勾选按**原始行**存, 翻页 / 排序 / 筛选不丢, 勾选行整行铺 `Muted`)、`WithDataTableColumnToggle`(工具栏"视图"下拉 = `Menu.Check` 列清单)、`DataColumn{Render,Align,Hidden}`(富单元格不写原生文本)、`OnCellClick(row,col)`、`Selected/SetSelected/ClearSelection/IsSelected/SelectedCount/OnSelectionChange`、`SetColumnVisible/ColumnVisible/SetColumns`; 有勾选时页脚把总行数换成"已选 x / y 行"。
  - **踩坑(DESIGN 167~170)**: ① 单元格扩展必须挂 Table 内部钩子(另注册 `AddEvent_List_DrawItem` 会被行底色 + 行底边盖掉), 列索引/矩形直接取 `pItem.ISubItem` / `RcItem`;② 自绘列必须清空原生文本 —— 单元格文字是**项模板子元素**, `pbHandled` 拦不住, 会在自绘内容上再叠一层;③ `DeleteColumnAll` 不重建行适配器, 加回的新列文本全丢(`CreateAdapter` 才是重建点);④ 勾选模式给表格挂 `AddEvent_LButtonDown`, 用 `List.HitTestOffset`(元素客户区坐标)取行/列, 处理完 `*pbHandled = true` 吞掉消息, 原生单选才不会把行点亮。
  - **demo**: 数据页换成 tasks 风格任务表(勾选列 / 徽章 + 标题 / 状态图标 / 优先级图标 / `ellipsis` 行操作 + 官方 Edit/Copy/Favorite/Labels 子菜单/Delete), 行数扩到 24、页大小 10/20/30; 表格卡描述改成一行(长描述会被卡片裁掉)。
  - **验收**: ① 勾选 3 行 → 表头变不确定态(白色横杠)、3 行铺 Muted、页脚"已选 3 / 24 行"; ② 隐藏优先级列 → 列结构重建、标题列吃掉空出来的宽度、勾选与页码保留; ③ 真实鼠标点行 ≈ `OnRowClick` 触发且**没有**原生单选高亮(勾选模式); ④ `go build/test ./...` 全绿; ⑤ `xui add data-table` 闭包 + `add --all` 编译通过。
  - registry: `data-table` 的 deps 加 `menu`(视图下拉), 否则单装 `data-table` 会缺 `NewMenu`/`MenuOnCheck`。
- ✅ **批次 11: Data Table 数据表格**(2026-10-31, 参 shadcn data-table 文档 / TanStack Table 的 tasks 示例):
  - **组件**(`components/datatable.go`): `NewDataTable(x,y,cx,cy,parent, []DataColumn{Title/Width/Sortable/Compare}, WithDataTableSearch/NoSearch/PageSize/PageSizes/FilterColumns)`;`SetRows/AddRow/Refresh`、`SetFilter/Filter`、`SetSort/Sort`、`SetPage/Page/PageCount`、`SetPageSize/PageSize`、`TotalRows/FilteredRows`、`OnSortChange/OnPageChange/OnFilterChange/OnRowClick`。
  - **交互**: 可排序表头点击循环 **升序 → 降序 → 取消**(后来在 11B 改为列头菜单, 11D 加 Shift+点击多列排序; 图标 arrow-up-down/up/down, 平局保持原顺序);排序与筛选变化都回第一页;顶部筛选不区分大小写、子串匹配、默认全列;页脚 = 总行数(筛选时"筛选出 x / y 行")+ 每页行数 Select + "第 x / y 页" + 首页/上一页/下一页/末页(到端点禁用);空态"暂无数据 / 没有匹配的数据";行点击沿用 Table 的原生单选, 回调回传**原始数据**行下标。
  - **排序口径**: 数字(允许货币符号/千分位/百分号)按数值;英文与数字段按自然顺序(同 `localeCompare numeric:true`);中文按 UTF-8 码位;`DataColumn.Compare` 可覆盖。
  - **踩坑(DESIGN 162~164)**: ① 排序图标必须在 Table 的表头绘制回调里追加 —— 另注册 `AddEvent_List_Header_DrawItem` 会被组件自己的不透明底色盖掉;空状态走 List 的 `PAINT_END`;② Input 的清除按钮走程序化 `XEdit_SetText`, **不发原生变更事件**, 清空后筛选不刷新;修在 Input(内部 `changeFns` + 清除后 `emitChange`), 并在程序化 `SetFilter` 后补 `Redraw`(内嵌件命中区是 paint 时重建的);③ `cy=0` 时 Auto 父级里的 Weight 表格塌成 0, 连 `ItemMinSize` 都不算 —— 根元素要先把初始高度设成"工具栏 + 每页行数 + 页脚"。
  - **附带修复**: `Input` 的清除按钮现在会触发 `OnChange`(所有 `WithClearButton + OnChange` 的输入框同步受益, 此前点 × 清空后回调不执行)。
  - **demo**: 数据页「数据表格」卡改用 DataTable(12 行客户数据、页大小 10/20/30、全列可排序), 树卡片保留;页签仍 19 页。
  - **验收**: 探针(清除筛选 6→12 行、表头点击 sort col/dir、页脚 Select 弹出窗口 1→2、下一页/首页、Auto 高 484)+ demo 150% DPI 亮/暗(排序三态与图标、输入筛选 6/12、空态、每页行数 10→20→10、翻到第 2 页、运行中切主题)+ `go build/vet/test ./...` 全绿;`xui add data-table` 闭包与 `add --all` 编译通过。
  - registry: 新增 `data-table`(files: datatable.go; deps: table/input/button/label/select)。
- ✅ **CLI: `update` 区分"版本升级"与"本地改动"**(2026-10-30):
  - **背景**: 以前 `install` 只比对"本地内容 vs 远端内容", 分不清"用户改过"和"版本升级导致源码变了" —— 跨版本 update 时每个文件都算"有差异", 默认全部跳过, `xui update` 实际等于空转(只刷一串警告)。
  - **做法**: `add` / `update` 结束时在组件目录写 `.xui-manifest.json`(CLI 版本 + 文件名 → sha256, 点开头 go 工具链忽略)。update 时逐文件: 缺失 → 创建; 内容一致 → 跳过; **本地哈希 == 清单哈希**(装完没人动过) → 直接替换并报"更新: 版本升级"; 哈希对不上 / 没有清单 → 保持"警告 + 跳过", `-o` 覆盖后清单重建。
  - **兼容**: 老项目没有清单 → 退回保守模式(一律跳过, 提示"没有安装清单可判断来源"), 加一次 `-o` 即重建清单。
  - **验收**: 新增 `TestUpdateUsesManifestToTellUpgradeFromLocalEdit`(add 写清单 → 远端升级自动替换 → 手改后跳过 → 删清单回退保守); 真实 CLI 在临时项目走通四条路径(add 生成清单 / 换源升级"更新 (版本升级)" / 手改后跳过 / `-o` 丢弃改动)。帮助文本与 README 同步(update 行改写 + "更新时的判定"表)。
- ✅ **CLI 参数一致性审查**(2026-10-30):
  - **未知命令的退出码**: 以前打 usage 后返回 0, 脚本里 `xui foo || exit 1` 判不出来; 现在 `xui: 未知命令 "foo" (见 xui --help)` + 退出 1。
  - **`-h` / `--help`**: 任何位置都打帮助(`xui add --help` 以前报"未知选项 --help"), 退出 0; usage 的选项表补上这一行。
  - **需要值的选项不再吞掉后面的选项**: `xui add -p --all` 以前把 `--all` 当目录名, 最后报出莫名其妙的"用法: xui add <组件...>"; 现在 `-p 需要一个值 (后面紧跟的是选项 "--all")`。统一走 `nextValue`(负数不算选项, `-l -1` 仍落到"需要正整数")。
  - **短选项支持 `-x=值`**: `-p=src` / `-c=dir` / `-q=词` / `-l=5` / `-o=2`, 以前只有 `-n=` 与长选项支持, `-p=ui` 会报"未知选项 -p=ui"。
  - **全局选项可放在命令前**: `xui -s list` 以前把 `-s` 当未知命令; 现在先剥全局选项再认命令。
  - **`--diff` 标出差异来源**: 每个文件头注明「本地没有 / 本地未改动, 纯版本差异 —— update 会直接替换 / 本地已修改 —— update 会跳过, 需 -o」, 与 update 的判定对齐。
  - **dry-run 文案**: 跳过类提示改为"将跳过"(其余分支都已是"将创建/将覆盖/将更新")。
  - 单测: 未知命令退出码(子进程)、`-h` 任意位置、短选项 = 形式、选项后紧跟选项报缺值; `go build/vet/test ./...` 全绿, 真实 CLI 逐条实测退出码与文案。
- ✅ **批次 10: Chart 图表**(2026-10-30, 参 shadcn chart / recharts 示例):
  - **组件**(`components/chart.go`): `NewChart(x,y,cx,cy,parent, WithChartKind(ChartBar/ChartLine/ChartArea/ChartPie/ChartDonut), WithChartData(labels, ChartSeries{Label, Values}...), WithChartStacked/Curve/Dots/BarLabels/Grid/Tooltip/Legend, WithChartYFormat/YDomain, WithChartInnerRatio, WithChartCenter(title, sub))`;运行时 `SetData/SetKind/SetStacked/SetCurve/SetDots/SetGrid/SetTooltip/SetLegend/SetBarLabels/SetYFormat/SetYDomain/SetCenter/SetInnerRatio`,点击数据点 `OnPointClick`。
  - **形态**: 柱状(多路并排, 柱顶数值 + 数据端圆角 4)、**横向柱(`WithChartHorizontal`)**、堆叠柱、折线(平滑采样 + 数据点)、堆叠面积、饼图、环形图(中心两行文字)、**径向图(`ChartRadial`, 每分类一层同心弧 + 背景轨)**、**雷达图(`ChartRadar`, 等角轴 + 闭合多边形 + 悬停轴高亮)**;数值轴按数据自动取整齐刻度(1/2/2.5/5 × 10^n),图例可点(点后临时隐藏该分类)。
  - **交互**: 悬停柱/曲线出竖向指示线 + 提示卡(分类 + 各路数值),饼环悬停扇区外扩 + 提示卡;坐标柱可点(`OnPointClick(cat, series, value)`)。
  - **踩坑(DESIGN 159 / 160)**: ① D2D 下曲线绘制是空实现, 平滑曲线自己采样成折线;② ScrollArea 里的元素收不到元素级 MouseMove, 悬停挂宿主窗口级 `AddEvent_MouseMove` + `XEle_GetWndClientRect`;③ 隐藏页图表矩形还在原位, 命中前沿父链查 `XWidget_IsShow`;④ 中性主题 `Chart1..5` 是灰阶, 片内百分比按亮度差在 Foreground/Background 里取更对比的(不能写死 background, 也不能按"片色亮就用 Foreground" —— 暗色主题的 Foreground 也接近白);⑤ 提示卡画在元素内(翻转/夹紧), 不建小窗;⑥ **扇区边缘锯齿**: `XDraw_FillPolygonF` 无抗锯齿, 改走 D2D 直通 `FillGeometryColor`, 且必须先用 `XDraw_ConvRect` 换算坐标 + 按 `sx/sy/DX/DY` 设变换(否则高 DPI 下整个图形缩到 1/1.5 并偏左上), 变换要 defer 恢复;⑦ 百分比标签卡着 `TextWidth` 给框会被折行(`DrawText` 自动换行), 左右各留 4px。
- **本轮(2026-10-30)用户报障二次修**: 环形图/饼图外缘锯齿 + 扇区内数字折成 `48` / `%` 两行 —— 见 DESIGN 160;探针像素扫描确认边缘出现 1px 中间灰度(255→207→64)、标签为单行, 亮/暗主题各验一次。
- **同日补齐三种形态**: 横向柱(`ChartBar + WithChartHorizontal`)、径向图(`ChartRadial`)、雷达图(`ChartRadar`), demo 图表页 6 卡 → 9 卡(新增"横向柱状图/径向图"一行 + "雷达图"整行), 实现坑见 DESIGN 161。验收: 探针三图截图(横条数值/分类对齐、径向四层弧按值扫角、雷达六边形网格与两多边形/标签不被裁)+ 真实光标悬停三图各出提示卡(横向柱命中"4月"、径向命中"平板 130/15%"、雷达命中"性能 186/80")。
  - **demo**: 新增**图表**页(17 号页, 9 张卡: 柱状 / 堆叠柱 / 折线 / 面积 / 环形 / 饼 / 横向柱 / 径向 / 雷达——末两张一行, 雷达整行), 页签总数 19(基础4 / 表单5 / 导航4 / 反馈3 / 图表 / 数据 / 图标);目录 `registry.json` key = `chart`。
  - **回归**: `add chart` 闭包与 `add --all` 均编译通过;`go test ./...` 通过;截图核对柱/折线/面积与探针里的饼/环(含悬停提示卡)。
- ✅ **批次 9: Sidebar 侧边栏**(2026-10-30, 参 shadcn sidebar / radix-nova):
  - **组件**(`components/sidebar.go`): `NewSidebar(x,y,cx,cy,parent, WithSidebarSide/Variant/Collapsible/Width/IconWidth/Open/Rail/NoShortcut/OnOpenChange)`;外壳 = 左侧(或右侧)面板 + 主内容区 `Main()`, 只接受确定高度(内部 Content 是 ScrollArea, 吃剩余高度)。`Header()` / `Content()` / `Footer()` 三段 + `AddHeaderItem` / `AddFooterItem`;`AddGroup(label)` 返回分组(标签/组动作/菜单), `AddItem(text, icon, WithSidebarItemActive/Size/Variant/Badge/Tooltip/Disabled/OnClick/Action/ActionOnHover)`;`AddSub()` 子菜单(`mx-3.5 + 1px 竖线 + px-2.5`, 右侧 chevron, 点击父项 200ms 开合, `SubOpen` / `SetSubOpen` / `ToggleSub`);`AddSeparator` / `AddTrigger`;`Open` / `SetOpen` / `Toggle` / `OnOpenChange`;`SetActive` / `SetBadge` / `SetTooltip` / `SetAction` / `SetDisabled`。
  - **交互**: 宽度 200ms 过渡(offcanvas 面板滑出 / icon 收成 3rem 图标栏 / none 固定);icon 折叠时菜单项 32×32、分组标签/子菜单/徽章/动作隐藏, 菜单项悬停弹右侧 tooltip(delay 0);子菜单点击父项开合、chevron 旋转;Ctrl+B 开合;rail 16px 热区悬停出 2px 线、点击开合;三种外观(sidebar 内缘 1px 分隔线 / floating 8px 边距 + 圆角 + 描边 / inset 外层 sidebar 底色 + 主内容区圆角卡片)。
  - **踩坑(DESIGN 158)**: ① 原生 Button 会截走子元素鼠标事件且 BnClick 仍发到整行 —— 菜单动作钮改成按钮内部 20×20 命中区;② Button 事件 `pPt` 与元素客户区口径不一致, 用 `GetCursorPos + ClientToScreen + XEle_GetWndClientRectDPI` 反算;③ 普通 Element 悬停不自动重绘(In ScrollArea 里 MouseStay 也不可靠), rail 改窗口级 MouseMove 命中;④ 构造期 `SetZOrderEx` 访问违例, 用 `SetZOrder(0)`;⑤ 面板高度必须确定(Content 的 Weight);⑥ icon 折叠必须同时关掉动作命中, 否则不可见命中区会盖住中央图标;⑦ 初始 `WithSidebarOpen(false)` 要直接把 progress 落 0。
  - **demo**: 新增**导航4**(13 号页) —— inset 外观 + icon 折叠 + 分组/子菜单/徽章/显示在悬停的动作钮 + rail + Ctrl+B;主内容区状态文字跟随点击/动作/分组动作实时更新。页签总数 18(基础4 / 表单5 / 导航4 / 反馈3 / 数据 / 图标)。
  - **上游参考**: `shadcn/src/components/ui/sidebar.tsx`(registry new-york-v4, 727 行)+ `src/hooks/use-mobile.ts` + `skeleton.tsx`, App.tsx 导航页加对照卡(固定高容器内 override `sidebar-container:absolute`), `pnpm build` 通过。
  - **验收**: 探针(展开 256 / icon 收起 48、floating 收起 66(面板 50)、offcanvas 收起 main 全宽且面板滑出、右侧镜像; item/label/sub 矩形; 长文本省略; 禁用降色)+ 真实交互(demo 150% DPI: 点击触发器/菜单项/动作钮/分组 +、子菜单开合与 chevron 旋转、Ctrl+B、rail 悬停 2px 线并点击、折叠态 tooltip、亮/暗主题、状态文字跟随)+ `go build/vet/test ./...` 全绿; `xui add sidebar` 闭包与 `add --all` 编译通过。
  - registry: 新增 `sidebar`(files: sidebar/scrollarea/scrollbar/tooltip/button/group/textopt/press; deps: 无)。

- ✅ **批次 8: Resizable 可调整分栏**(2026-10-30, 参 shadcn resizable / react-resizable-panels v4):
  - **组件**(`components/resizable.go`): `NewResizableGroup(x,y,cx,cy,parent, WithResizableVertical/Disabled/OnResize)` + `AddPanel(WithResizableSize/Min/Max)` + `AddHandle(WithResizableGrip)`; 面板按百分比分主轴空间(手柄固定 4px, 窗口缩放保持相对比例), 像素宽用累计边界取整; 分隔条 1px `bg-border` + 4px 热区 + 可选 4×24 抓手; 拖动捕获到分组(手柄随布局移动, 不能当坐标原点), 相邻面板到 min 后继续挤压同侧更远的、增长端受 max; 方向键 5%、Home/End 到边界、双击"有显式默认值的一侧"复位; 点击分隔条聚焦后方向键可用(嵌套元素不在 xcgui 的 Tab 链上, 见 DESIGN 157)、focus-visible 一条更粗的 ring 单线、col/row-resize 光标; 一侧面板收到 0 时分隔线贴到分组边缘、抓手隐藏; 面板是流式容器, 内容直接放进去。
  - **踩坑(DESIGN 157)**: ① 分组主轴必须确定尺寸(Auto 下百分比面板停在创建时的 1px); ② 百分比→像素按累计边界取整, 逐块 round 会留 1px 缝/向右溢出; ③ 鼠标捕获必须落到**分组**而不是手柄(手柄只有 4px 且拖动中会移动), 坐标换算到分组客户区; ④ 键盘同 152 挂窗口级; ⑤ 探针合成鼠标: `XEle_GetWndClientRect` 是逻辑坐标而 `ClientToScreen/SetCursorPos/mouse_event` 走物理坐标, 150% 下要 `XEle_GetWndClientRectDPI`; `PostMessage(WM_LBUTTONDBLCLK)` 不会派发到元素, 双击要真实 `mouse_event`(间隔小于系统双击时间), 并先把光标移进窗口刷新悬停命中。
  - **demo**: 基础3 新增「可调整分栏」卡(横向 35/65 + 右侧嵌套纵向 40/60, 带抓手), 百分比文案跟随 `OnResize` 实时更新; 各面板都设 `WithResizableMin(20)`, 分隔条拖到端点时不会贴到 demo 外框(否则分隔线 + 外框看起来像"两条线")。
  - **验收**: 探针全绿(初始 35/65 与嵌套 40/60、窗口缩放保持比例、SetSizes、→/Home/End、真实双击复位 30/70、真实拖动 +120 逻辑像素 = 41.19%、纵向 ↓、禁用后键盘无效) + demo 150% DPI 拖动/键盘/双击/切页回归 + 暗色焦点环截图; `go build/vet/test ./...` 全绿; `xui add resizable` 与 `add --all` 闭包编译通过。
  - **后续修正**(用户截图): ① 抓手没居中(左多右少)—— 抓手用整数 `(Left+Right)/2`, 分隔线按设备像素取整, 150% 下差半个逻辑像素; 改为两者共用 `(c+0.5)/scale` 设备中心, 抓手用 `FillRoundRectF`。② 纵向分隔条拖到上面时出现两条线 —— 焦点环原本是内缩 1px 描边, 在 4px 厚的手柄上变成上下两条平行线; 改为以分隔线为中心的一条 ring 色带(约 3 逻辑像素, 线画在带中央)。③ 二轮反馈"还是偏左": 4 逻辑像素的抓手在 150% 下是 6 物理像素(偶数), 以 1 物理像素的线为中心永远一边多 1px; 厚度轴改取**奇数设备像素**(5), 实测抓手与线中心差 <0.1px。④ 二轮反馈"上面还是两条线": 焦点态不再叠"色带 + 细线", 直接画一条更粗的 ring 色**单线**(约 3 逻辑像素), 从根上避免任何"两条线"的观感; 聚焦/未聚焦都只有一条线, 只是粗细和颜色不同。
  - **再修正**(二轮截图): ⑤ 用户要求允许面板收到 0 —— demo 去掉 min, 改为"收起侧的分隔线贴到手柄外缘、抓手隐藏": 线画在中心时距外框 2px, 顶边是两条线、底边因取整只差 1px 看着像一条; 贴边后顶/底都并进外框(探针: 顶 95/底 481 各只剩 1 行)。⑥ **Tab 到不了分隔条是 xcgui 的限制**: Tab 链只覆盖窗口内容区的直接子元素, 套一层布局容器就被跳过(探针: 直挂内容的可聚焦 Element 可达, 放进 LayoutEle/Element 后 Button 也跳过; EnableKeyTab 是"接收 Tab 输入"不是加入焦点链); 已从 demo 文案与 DESIGN 中移除"Tab 可聚焦"的承诺, 并在 DESIGN 第七节登记; 分隔条改为点击聚焦后方向键操作。
  - ⚠️ 如果是在**已拷贝组件**的项目里验收: 本轮修复只在本仓库 `components/resizable.go`, 需要在该项目里 `xui update resizable`(或重新 `xui add resizable`)才会生效。
  - registry: 新增 `resizable`(单文件, 无依赖)。

- ✅ **Tabs / Tooltip 补齐上游特性**(2026-10-29, 参 shadcn radix tabs / tooltip 文档, DESIGN 155/156):
  - **Tabs**(`components/tabs.go`): `NewTabsItems` + `TabsItem{Text, Icon, Disabled}` 支持行内图标(size-4 + gap-1.5)与单标签禁用(文字/图标 50% 预混、点击与方向键都跳过); 键盘改为 **radix 的 roving focus**: 整块列表一个 Tab 站点, 标签按钮全部 `EnableFocus(false)`, ←/→/↑/↓ 移动内部焦点、Home/End 首末、端点不循环, `WithTabsActivation`/`WithTabsManual` 控制"移动即切换 / Enter·空格才切换"(默认 automatic), 焦点环只画在内部焦点标签、且仅键盘来源; value 语义 `WithTabsValues` + `ActiveValue`/`SetActiveValue`/`OnValueChange`; `BindPanel` 按选中项显隐面板并重排父级; line 变体补 gap-1 与"下划线贴当前行底边"; 暗色选中态改 `bg-input/30 + border-input`。
  - **Tooltip**(`components/tooltip.go`): 四方向 `WithTipSide` + `WithTipAlign` + `WithTipOffset`, 放不下自动翻对侧再夹进工作区(箭头始终指向元素中心并夹进圆角); 显示延迟 `WithTipDelay`/`SetDelay` 与跳过延迟 `SetSkipDelay`(radix 的 skipDelayDuration, 默认 300ms); 键盘焦点进入也显示(鼠标点击产生的焦点不算); 内容右侧可挂快捷键块 `WithTipShortcut`(Kbd, 同时把右内边距收到 6); 受控 `Show`/`Hide`/`IsOpen`。
  - **踩坑**: ① 多个 Tooltip 实例共用固定过渡键 → 抢同一条 presence, 后显示的停在元素 alpha 初始值(气泡发灰), 改为每实例 `xui.NextPresenceKey()`; ② 显示中切换目标时上一次的进场帧还在跑, 会用旧 px/py 把窗口拉回旧位置(表现为"时好时坏"), 加 `fadeGen` 代际作废过期帧; ③ **ScrollArea(LayoutFrame) 里的元素收不到元素级 MouseMove/MouseStay**(窗口级回调有、元素计数器为 0) → 页面里 tooltip 全不弹, 加窗口级命中测试兜底; ④ `windowOf` 一步 `XC_hWindowFromHWnd(XWidget_GetHWND(ele))` 对滚动区元素返回 7, 沿 HWND 父链找真窗口; ⑤ 命中测试必须判祖先可见性(隐藏页面的元素矩形还在原位, 会弹出别的页的文案); ⑥ 标签按钮不可聚焦后点它列表拿不到键盘焦点, 在 `BnClick` 里补 `XWnd_SetFocusEle`; ⑦ 焦点环不能整块填充(未选中标签透明, 内圈会透出来发灰), 画外扩 3px 的**环带**描边; ⑧ 键盘处理仍必须挂窗口级(同 152)。
  - 附带修掉的历史问题: ① demo 内容全面改用 ScrollArea 后, 页面上所有 tooltip 其实一直不弹(③④), 只是没人注意; ② 同一根因还让 `AttachWindow` 在滚动区里拿不到窗口 → 滚区里的组件不参与主题重绘、Tab 键也进不去。统一新增根包 `xui.HostWindowOf(hEle)`(沿 HWND 父链找真窗口), `AttachWindow` / Tabs 键盘钩子 / Tooltip 全部改用它。
  - **后续修正**(用户截图反馈): ① 侧向气泡的箭头过大 —— `tipArrow` 从 10 降到 7(shadcn `size-2.5` 旋转 45° 的露出高度), 并给夹紧区间加 `hi < lo → 居中` 兜底(单行气泡下界 > 上界时旧写法会退化成贴边大三角); ② Tabs 用 Tab 显示焦点环后再鼠标点击, 环不消失 —— 环画在列表上而点已选中项不触发列表重绘, 标签与列表的 `LButtonDown` 里补 `Redraw`。
  - 验收: 探针键盘断言全绿(automatic 跳过禁用 / 端点不循环 / Home·End / manual 需 Enter / value / 面板显隐), tooltip 探针断言四方向贴边、align=end、offset、快捷键块占宽、焦点触发与失焦隐藏、延迟 400ms 与 skipDelay 跳过、`go test ./components -run Tabs`(换行网格 + 键盘 seek); 亮/暗两套截图(真实鼠标 hover 触发路径、四个方向内联、kbd 内容、焦点环); demo 导航2 标签页卡与反馈1 提示气泡卡改成新特性; `xui add tabs|tooltip` 各自闭包 + `add --all` 编译通过; `go build/vet/test ./...` 全绿。
  - registry: `tabs.files` 补 `textopt.go`(中文 Vcenter 半步校正)。

- ✅ **分页 icon-only 形态残留文字竖条**(2026-10-29, 用户截图报障, DESIGN 153):
  - **现象**: `WithPaginationHideText` 那行, 上一页 chevron 右侧一条竖线(`< |`)、下一页左侧一个冒号(`: >`)。
  - **根因**: 隐藏文字只把量出的文字宽置 0, 绘制分支仍然调用 `drawText`, 退化成一两像素宽的文字矩形 → "上一页/下一页"被裁成一条竖条。
  - **修法**: `hideText` 时直接跳过 `drawText`(两条分支都改); 顺带把 icon-only 链接宽收回 32 方形(与上游 default 按钮 pl-1.5/pr-2.5 + size-4 一致), 图标继续保持居中。
  - 验收: 探针窗口 150% 截屏逐列扫描墨迹 —— icon-only 行只有 chevron 与页码两类列簇, chevron 与相邻条目之间没有任何 1~3px 残条; `go build/vet/test ./...` 全绿。

- ✅ **Pagination 三种形态 + 键盘可达, Toast Loading / Promise**(2026-10-29, 参 shadcn pagination / sonner 文档, DESIGN 152/153/154):
  - **Pagination**: 补 `WithPaginationSimple`(只有页码)、`WithPaginationNavOnly`(只有上一页/下一页, 配"每页行数"选择器)、`WithPaginationHideText`(只留 chevron, 对应文字上的 `hidden sm:block`); 条目序列抽成 `sequence()` 并加 `components/pagination_test.go`(三形态槽位恒定 / 键盘 seek 跳过省略号与禁用项 / 焦点下标收敛)。键盘: 整块一个 Tab 站点, 进入后 ←/→/↑/↓ 移动、Home/End 首末、Enter/空格翻页, 到端点不循环; 焦点环只画在当前条目、且只在键盘获焦时出现。
  - **踩坑(键盘)**: 元素开 `EnableSwitchFocus`(Tab 才进得来)后, 框架**先**把 ←/→ 当"切换焦点"执行, 元素 `AddEvent_KeyDown` 里焦点已经跑到下一个控件, 处理不了; 回车还会被当成"激活"发给新焦点控件(实测把另一个分页翻了一页)。改挂**窗口级** `AddEvent_KeyDown`(先判 `XWnd_GetFocusEle` 是自己, 处理过置 pbHandled), 探针确认焦点保持不动、Tab 仍能走到分页。
  - **Toast**: 补 `ToastLoading` 变体(转圈, 默认常驻)、`Toaster.Loading`、`Toaster.Promise(fn, ToastPromise{Loading,Success,Error})`(fn 在 goroutine 跑, 结果经 `xc.Auto` 回 UI 线程改同一条通知, panic 用 recover 转成错误态)、`WithToastCloseButton`(整组带关闭钮)、`OnAutoClose` / `OnDismiss`(区分倒计时到点与主动关闭, `OnClose` 仍两种都发)。
  - **踩坑(转圈)**: `xui.Spin` 是一次登记持续运行, 后续不再调用并不会停表 —— 切出加载态必须显式 `xui.Spin(handle, false)`, 否则卡片一直按帧重绘(DESIGN 154)。
  - demo: 导航2 分页卡补三行(简洁 / 仅导航 / 只留图标 + 键盘说明), 反馈2 通知卡补 Promise、失败 Promise、常驻加载三钮。
  - 验收: 探针(键盘: 右右+Enter 翻到第 12 页、End/Home 到端点不越界、Tab 从普通按钮可达; Toast: Promise 成功与 panic 两路都转成结果态、OnAutoClose 与 OnDismiss 各自触发、Clear 清空计数归零) + demo 两页渲染 + `go build/vet/test ./...` 全绿。

- ✅ **InputGroup addon 按钮字号/圆角对齐上游**(2026-10-29, 用户截图报障, DESIGN 147):
  - **现象**: 输入组里的"查询/发送"比旁边的输入文字小一圈、看起来"粗细不同"。用 `ButtonSizeXs` 建 addon 按钮, 字号被降到 12px(周围是 text-sm 14px), 圆角也用了 Button xs 的 10(上游是 calc(--radius-5px)=5)。
  - **修法**: `applyInputGroupButton`: 字号固定跟主题正文(`fontPx` + `FitWidth`), 圆角按 InputGroupButton 语义覆写(Button 新增未导出的 `radiusMode`: xs=radius-5, sm=radius-md)。IconButton 同步。
  - **发送按钮偏上**: 两处叠加 —— ①块级行的按钮布局高被压成 24, 视觉壳只剩 23, 按钮高 2 物理像素(`fixButtonBox`: 布局高 25 + 底边距 -1, 块级行补 1px 底部内边距给按下位移); ②说明文字用整型 `p.Text`、按钮文字走了 151 的中文上移半像素, 按钮仍高 1 物理像素(`InputGroupAddon.Text` 改 `TextF` + `textVDy`, 仅 addon 内有文字按钮时启用, 行内 addon 因旁边是原生 Edit 保持不偏移)。实测"我们会发送验证码/发送"墨迹 203–223 完全重合, 按下壳体整体 +2 物理且底边不被裁; 行内 "example.com" 前缀与占位符保持原对齐。
  - 验收: 探针截图 + 像素测量, 查询/发送墨迹高 20/21(修前 18), 与周围 14px 文字一致; hover 圆角 5px; 按下文字与壳体同步 +2 物理; `go build/vet/test ./...` 全绿, `xui add input-group` 闭包编译通过。

- ✅ **中文文本偏低 + 分页按下图标/文字不同步**(2026-10-29, 用户截图报障, DESIGN 151/88):
  - **中文偏低**: 先报导航菜单"开始使用", 后报卡片页"了解详情/开始使用"。150% 实测: 同一套 Vcenter 下拉丁墨迹中心在行盒中心上方约 1 物理像素, **中文**在下方约 1 物理像素, 14px→11pt 再低一点, 叠加后中文低 1.5 物理像素。修法: 新增 `components/textopt.go` 的 `textVDy(text)`(含中文返回 0.5 逻辑像素, 纯拉丁 0), Button/ButtonGroupText/Toggle/Select/ComboboxSearch/NavigationMenu/Pagination/Accordion 用 `TextF` 浮点矩形上移半步。实测卡片页两按钮 +0.5(Chrome 同尺寸 +0.5, 修前 +1.5), 导航 0.0; 拉丁按钮维持 0/-0.5/-1.0。
  - **分页按压**: 上一页/下一页按下时图标与文字位移不一致(和按钮当年同一个坑, DESIGN 88)。修法: 壳体量实际物理位移 `shellPressShift`, 内容按**未按下矩形**排版, 文字 `TextF` 加 `N/scale`、图标 `IconRotatedOffset(...,N)`。150% 实测按下后文字与 chevron 都 +2 物理像素。
  - 验收: demo 导航2/导航3/基础3 150% DPI 原生截图 + 逐像素测量; `go build/vet/test ./...` 全绿。

- ✅ **12px 文字被画成宋体 + xs/sm 按钮文字偏高**(2026-10-29, 用户截图报障, DESIGN 149/150):
  - **字体**: 徽章的 "打开文档"(12px → 9pt)显示成宋体。探针确认是 xcgui 对 **<10pt 的常规字重**整体替换成宋体(微软雅黑/Arial/Consolas 的 9pt 常规字形完全相同, 10pt 起正常, 9pt 粗体正常)。抬到 10pt 会让 text-xs 从 12px 变 13.33px、xs/sm 字号档合并, 故改为 `FontHandle` 对 <10pt 常规字重从字体文件创建: GDI+ 取英文族名("微软雅黑"→"Microsoft YaHei")+ Fonts 注册表(HKLM/HKCU)找文件 → `font.NewByFile`, 字号保持 12px; 解析失败回退 `font.NewEX`。新增 `fontfile.go` + `font_test.go`(解析/缓存/不存在字族)。
  - **居中**: xs/sm 文字整体偏高 1 物理像素。根因是早期给文字矩形减 1px 的 "字形偏下" 回补 —— 同尺寸 Chrome HTML 按钮实测墨迹上下留白相等, 该回补本身就是偏差。`Button`(纯文字 + 行内图标)、`ButtonGroupText`、`Toggle`、`Select`、`ComboboxSearch`、`NavigationMenu`、`Pagination`、`Accordion` 的文字矩形统一去掉 `Top-1/Bottom-1`。
  - 验收: demo 基础1/2/表单2/导航2/导航3 150% DPI 逐页截图; "Xs"/"Sm"/"Default" 墨迹中心与壳体中心差 0.0 / -0.5 / -1.0 物理像素(Chrome 同尺寸 0 / 0 / -0.5); 徽章/链接文字恢复雅黑; `go build/vet/test ./...` 全绿。

- ✅ **Combobox 弹层两处交互修复**(2026-10-28, 用户报障, DESIGN 148):
  - **过滤后弹层跳边**: 触发器在屏幕底部、弹层翻上方后, 输入过滤让列表变矮, resize 按 sideBottom 重算导致弹层跳到下方。修法: 打开/resize 后记录弹层实际在哪一侧(`lastTop`, 弹层底边 ≤ 触发区顶边), resize 沿用。探针数据验证: 过滤后弹层底边固定、顶边随高度上移(保持上方)。
  - **多选勾选刷新不及时**: 勾选画在弹层行上, 点击切换后只重绘了触发区。修法: 多选分支补 `refreshRows()` 且行改用 `Redraw(true)`(立即重绘, 弹层行独立于宿主待重绘区域)。
  - 验收: Combobox 弹层方向探针数据(打开 (650,618)-(1040,812) → 过滤后 (650,702)-(1040,812), 底边贴触发区保持上方); `go build/vet/test ./...` 全绿。

- ✅ **Badge 图标/加载/自定义配色 + Input Group + Combobox 自动补全**(2026-10-28, 参 shadcn badge / input-group / combobox 三篇文档):
  - **Badge**(`separator.go`): 补 `WithBadgeLeadingIcon` / `WithBadgeTrailingIcon`(行内图标, data-icon)、`WithBadgeLoading` / `SetLoading`(前置 spinner, 相位由 `xui.Spin` 驱动)、`WithBadgeColors` / `SetColors`(自定义配色)、`WithBadgeOnClick` / `OnClick`(链接形态, 点击回调)、`SetText`;`cx` 传 0 时按内容量宽(图标 + 文字 + 图标整体居中, link 下划线按文字块画), 文字变化后自动重量。
  - **Input Group**(`components/inputgroup.go`, 新): `NewInputGroup` + `Addon(InlineStart/InlineEnd/BlockStart/BlockEnd)`, addon 内容支持 `Text` / `Icon` / `Button`(默认 ghost+xs)/ `IconButton` / `Kbd` / `Spinner` / `Content()`;`Input()` / `Textarea(cy)` / `Search(placeholder)`;`SetInvalid` / `SetDisabled` / `Focus`。内层输入是**裸输入**(`input.go` 新增 `WithInputBare` + `setBareInsets`): 不画壳体/环, 边框、焦点环、校验环、禁用底色全由容器画, 焦点用 `AddEvent_SetFocus/KillFocus` 让容器重绘。block-* 存在时自动切成"标题行 + 输入行 + 页脚行"。踩坑见 DESIGN 147(addon 宽不能用 Auto、inline-start 用 `SetZOrder(0)` 调顺序、`InsertChild` 会卡死)。
  - **Combobox 自动补全**(`components/combobox_search.go`, 新): 触发按钮(outline + chevrons-up-down)+ 弹层(顶部搜索框 + 建议列表 + 空态), 输入即过滤(不区分大小写)、↑↓ 循环跳过禁用、Home/End、Enter 选中、ESC 关闭;单选选中即关闭,`WithComboboxMultiple` 多选点击切换且保持打开(触发区"首个值 +N");条目模型复用 `SelectItem` / `SelectGroup`(从 select.go 抽出 `select_model.go`);弹层高度随过滤结果 `resize + Relocate`(坑: 布局容器位置参数被覆盖、内边距要用元素 padding + 子项锁 Fixed 宽, 见 DESIGN 148)。
  - registry 新增 `input-group` / `combobox-search`(select 的 files 补 `select_model.go`);demo: 基础2 徽章卡加"图标/加载/自定义/可点击"两行、表单1 新增"输入组"卡(8 种形态)、表单2 新增"自动补全"卡(单选/多选/校验/禁用)。
  - 验收: 探针截图逐项(输入组 8 形态 / 徽章 6 变体 + 图标 + loader + 自定义 + 点击回调触发 / Combobox 弹层过滤 + 多选切换 + 空态)、demo 三页 150% DPI 截图、`xui add input-group|combobox-search|select|badge|input` 各自闭包 `go build ./ui` 通过 + `add --all` 通过、`go build/vet/test ./...` 全绿。

- ✅ **Checkbox / Radio 三态、校验态与 RadioGroup**(2026-10-28, 参 shadcn radio-group / checkbox 文档): Checkbox 补不确定态(`WithCheckboxIndeterminate` / `SetIndeterminate`, 减号 + 点击落到选中)、aria-invalid(红边框 + 常驻红环, 选中边框保持 primary)与空格切换;Radio 补同款校验态、`SetSelected` 与空格选中。新增 `components/radiogroup.go`:`NewRadioGroup` + `AddItem` / `Attach`(跨 Field 父级入组)+ `Value` / `SetValue` / `OnChange` / `SetDisabled` / `SetInvalid` / `SetSpacing`;↑↓←→ 循环移动并选中、Home/End 首末、空格选中。根因: 同父兄弟手动 `SetCheck(false)` 会让原生单选把刚选中的项一起取消, 键盘 `SetFocusEle` 需延迟到消息循环, 见 DESIGN 143。验收: 独立探针(点击互斥、方向键/Home/End、Space、Field 嵌套项、SetValue、禁用) + demo 新增表单5页两卡(单选组: 默认/Field 说明项/禁用组/可切换校验态; Checkbox 三态与校验);亮暗两套截图核对不确定态与红环;`go build/vet/test ./...`、`xui add radio-group` 与 `add --all` 闭包编译通过。
- ✅ **菜单栏触发区垂直居中**(2026-10-28, 用户截图报障, DESIGN 142): 触发区悬停/展开的底色块与文字整体偏上 1 逻辑像素, 条本身也比上游高 2。根因: Button 元素高 = 视觉高 + `pressShift`(1px, 留在底边), 菜单栏却按元素高居中, 条高又按旧预留量写成 38。修法: 条高改回上游 h-9 = 36, 触发区按壳体高居中, 弹层锚点改从视觉底边量 sideOffset。验收: 探针 + demo 导航3页 150% DPI 像素实测(条 36、壳体 28、上下各 4; 文字墨迹中心与条中心差 0.5 物理px; 弹层卡片顶边 = 条底边 + 4), `go build/vet/test ./...` 全绿, `xui add menubar` 与 `add --all` 闭包编译通过。

- ✅ **手风琴收起后卡片文字重影**(2026-10-26, 用户报障, DESIGN 136): 收起第一项后卡片说明文字在旧位置留一整行没擦掉, 且这一次收起没有 200ms 过渡(直接跳到底)。根因在 `Collapsible`: demo 建页时该页还是隐藏页, `expand()` 量到内容高度 0 走早退分支 → 这个键的 presence 值一直停在 0; 第一次收起的 target 也是 0 → `Presence` 幂等不起过渡、**done 永不回调**, 而终态(隐藏内容)与整窗重绘都写在 done 里 → 布局变了、屏幕一次都没重绘, 旧帧留在屏上。修两处: `startPresence` 接住返回值, 值已在目标上就自己落终态 + `redrawNow`; `expand()` 早退分支把值抬到 1(让后续收起回到正常过渡)。验收: 探针慢放对比(修前旧帧实测停留 20s 不消失、内容 `show=true`;修后逐帧缩、`show=false`、旧帧擦净)、demo 两条进入路径各连点 8 次 + 可折叠面板页 6 次, 像素级"同一行文字出现两遍"检测全 0、`go build/vet/test ./...` 全绿、`accordion collapsible card` 与 `add --all` 闭包编译通过。

- ✅ **副屏第二批: 下拉框弹层偏下 / 子菜单错位一行**(2026-10-26, 用户报障, 同一块右上副屏):
  - **ComboBox 弹层(DESIGN 135)**: 表单2 的下拉框点开, 弹层比触发区低 300+ 物理像素。根因是 `alignComboPopup` 的边界写死 `screenTop=0` + `SM_CYSCREEN`(只有主屏), 副屏 y 为负 → item-aligned 的期望位置恒被压到 y=0。改为两端都取 `xui.MonitorWorkArea(窗口真实句柄)`。实测副屏上「默认尺寸」弹层首项重新压在触发区上(选中项与触发区中心重合), 顶/底夹住时仍走原来的"滚动列表让选中项作首项"。
  - **子菜单 Y(DESIGN 134, 五组窗口位置 × 两块屏探针数值实锤)**: 触发区贴近窗口底部、父菜单翻到上方后, 子菜单整体低一行。上一版的"给子菜单补一份父级纵向位移"不成立 —— 原生算子菜单 Y 时会把它的底边夹进**父弹窗创建时矩形**的底边(我们 `SetRect` 之前的位置), 截断一旦发生位移与结果不再同比例(实测父级位移 +93 逻辑时子菜单只跟到 +139 而非 +200)。改为彻底自己算: 子菜单 Body 顶 = 父弹窗 Body 顶 + 该条目在同组内的行偏移(条目 28 / 分隔栏 4, 与 `SetItemHeight`/原生槽高一一对应), 分组弹层顶记在 `Menu.groupTop` 里支持任意层嵌套, 再夹进工作区; X 保留原生(它锚父弹窗实际矩形右缘, 会跟随)。实测五组配置子菜单 Body 顶统一 = 触发行顶 − 4 逻辑, 下拉菜单与右键菜单都齐平。
  - **顶级弹窗夹取边界(并入 DESIGN 132)**: 工作区改按**弹窗自己的句柄**取 —— 宿主窗口横跨两屏时 `MonitorFromWindow` 只认占多数的那块, 实测会把弹层夹到另一块屏的顶边。
  - 验收: 副屏 demo 截图(下拉框弹层贴触发区、子菜单首项与触发行齐平)、`probe_sub2` 五组矩形逐项吻合、`go build/vet/test ./...` 全绿、`menu combobox tooltip` 与 `add --all` 闭包编译通过。

- ✅ **副屏三问修复: 菜单弹层错位 / ToolTip 恒朝下 / 失焦停转**(2026-10-26, 用户报障, 右侧上方副屏 y 为负):
  - **菜单弹层位置(DESIGN 132, 探针数值实锤)**: 窗口在副屏时下拉菜单弹到触发区右上方几百像素外。根因是 `XMenu_Popup` 把"显示器**物理**矩形 × DPI"当钳制边界 —— 150% 下副屏左缘 2570 被当成 3855, 触发区物理 x=2618 的弹窗被强行推到 3855(钳制特征: 2618/3018 都吸到 3855, 4618 越线才正确); y 同理(右下副屏 (2618,1024) → (3840,1286))。位置输入无法绕过, 修法 = `Menu_PopupWnd` 回调里按锚点重算 Body 落位(下贴 +4 / 放不下翻上方 / 夹进 `xui.MonitorWorkArea` 工作区), 用 `SetRect`(逻辑像素)落地。探针验证: 大位移不崩、子菜单跟随父窗实际矩形(父右缘 +1)、挪过的菜单悬停/点击正常; 副屏三组 + 主屏三组落位/夹取/上翻逐项吻合(±1~2px 为原生取整噪声)。demo「更多操作」实测贴回触发钮下方; 「筛选」在合成滚动后命中区滞后点不动(旧版 HEAD 构建同样复现, 与本改动无关, 属 DESIGN 118 类现象), 其菜单与「更多操作」同走 `PopupAtElement`。
  - **ToolTip 方向(DESIGN 133)**: 标题栏按钮悬停时气泡翻到下方压住页面, 窗口上方明明还有空间。根因同属"拿虚拟屏原点 (0,0) 当边界"—— 副屏 y 为负, `y < 0` 恒成立; 改为按窗口所在显示器工作区判断翻转/夹紧(取不到时回退旧行为)。实测副屏上「切换主题」气泡显示在按钮上方。
  - **失焦停转 → 可见即转(DESIGN 106 修订)**: 用户确认窗口可见但失焦时加载/骨架屏应继续转(此前为省 DWM 做的前台暂停, 在副屏上看起来像卡死)。去掉 `animForeground` 前台判定, 只在最小化/隐藏/滚出视口/祖先隐藏时停表。实测焦点切到主屏浏览器后, 副屏 demo 的 Spinner 仍在转。
  - **打包连带**: `monitorWorkArea` 原在 `popup.go`(弹层底座), ToolTip/Menu 引用它会破坏逐文件安装闭包 → 上移为运行时 `xui.MonitorWorkArea`(`monitor.go`), registry 不变。
  - 验收: 副屏截图(菜单贴钮、气泡朝上、失焦转圈)、边缘数值探针(主/副屏各 3 组)、`go test ./...` 全绿、`xui add menu tooltip` 与 `add --all` 闭包编译通过。

- ✅ **下游问题清单复核与修复**(2026-10-26, sitless 项目报告, 13 条):
  - **布局内边距溢出(DESIGN 129/130, 真问题)**: 480 宽 Fill 卡片探针确认 —— ContentStack 四边外边距 16 时自身 448 宽, 内部 Fill 行却是 480(右溢 16); 改为左右 `SetPadding` + 上下外边距后行宽 448。同一写法一并修掉 `CardFooter.Content`、`Alert` 行容器、`Empty.Body`、`Command` 分组、`popup.LayoutContent`(Popover/HoverCard); `ScrollArea` 文档样例同步改为安全写法。Alert 改用正确内边距后暴露第二个问题: 横向行里 Weight 宽文本列的 Fill 标题/描述宽度变 0(嵌套 Weight 不下发宽度, DESIGN 101), reflow 现在按文本列实测宽把两者锁 Fixed(130)。demo 反馈2页 Alert 文字恢复、行尾 Undo/New 不再被卡片边界裁掉。
  - **复验回归(同日)**: 内边距修复还让卡片内容区里的嵌套 Weight 列(Fill 子项)宽变 0 —— 基础3 的 CardHeader 标题/说明、基础4 的 Item、导航2 的 Collapsible 正文整块消失。CardHeader 增加 `syncTitleWidth()`(按标题列实测宽锁标题/说明 Fixed); demo 的 `col` helper 用 `synced` map 在列宽变化时重新锁 Fill 子项。修正后 1200 与 960×660 下页面 2/3/9 正常, 16 页缩略图逐页核对。**踩坑**: 锁过一次后子项布局项类型已从 Fill 变 Fixed, 后续 resize 若只按 Fill 筛选就会跳过它们 —— 行尾按钮会停在旧宽被卡片裁掉; 必须记录本容器锁过的子项(见 DESIGN 130)。
  - **弹层 z 序(DESIGN 2, 真问题, A/B 截图)**: 宿主 `SetTop(true)` 时 `popup.go` 的 `HWND_TOP` 弹层被整个盖住(只有触发区可见); 改为 `HWND_TOPMOST` 后 Select 下拉正常显示, 与 tooltip 一致。`resize` 里带 `SWP_NOZORDER` 的一处保持不动。**全量复核**(DESIGN 131): Tooltip 原本就是 TOPMOST; 原生 `XMenu_Popup` 菜单与 ComboBox/NativeSelect 下拉是宿主拥有的 popup 窗口, 探针截图确认在 TOPMOST 宿主上仍可见, 不需要改; Dialog/Sheet/Drawer/AlertDialog/Toast 是宿主窗口内覆盖元素, 不涉及独立窗口 z 序。
  - **AppWindow NoMinMax**: 双击标题栏处理器补 `noMinMax` 守卫, 只留关闭钮的窗口不再被最大化到没有还原入口。
  - **HStack**: 补 `EnableAutoWrap(false)`, 行内超宽不再把子项甩到下一行/整块消失(DESIGN 41)。
  - **CLI**: `update` 无参数改为"files 全部存在才算已安装" —— 只装 select 时不再把 datepicker/popover/dialog/table/tree/combobox/toggle 等共享 `popup.go/scrollarea.go/scrollbar.go` 的组件整套拷进 ui/; `init -n .` 支持当前空目录, 且当前目录为空、目录名 == 项目名时直接铺在当前目录(不再生成 sitless/sitless)。两项都加了测试。
  - **主题模板**: `init` 生成的 `ui/theme.go` 增加 `UseDark(v bool)`(切换本文件里 light/dark 副本, 保留逐字段定制)。
  - **Toast**: 新增 `ToastTopCenter` / `ToastBottomCenter`(sonner 六种位置补齐), 居中按 Body 水平居中, 进场沿堆叠方向、退场向上/下; 探针双 Toaster 截图确认上下居中与堆叠顺序。补 `richColors`: `WithToastRichColors()`(整组)/ `ToastOptions.RichColors`(单条)开启后按 sonner 的 `--{success,info,warning,error}-{bg,border,text}` 给卡面/描边/文字上色(亮/暗两套), 色值取自 sonner styles.css, 不是 xcgui 限制; demo 通知卡加 richColors 按钮。
  - **未复现/非缺陷**: Label 的 `SetBorderSize(1,0,1,0)` 在 150% DPI 下与无 border 组的省略号阈值完全相同(38/40/42 截断、44 起完整), 不移除(保留 g/p 下伸修复, DESIGN 109 已补复核); HStack 行内未声明布局项的 Input(96)/Label(100)保持创建尺寸, 未复现"被收缩"现象; `AddCaptionButton` 默认 14 适用于满网格图标, 线条类图标用返回按钮的 `SetIconSize(18)` 即可, 不新增 API; 固定尺寸窗口 SetRect 白屏属下游自建窗口场景(上游 AppWindow 始终带 Drag_Border), 未改代码。
  - 验收: `go test ./...` 全绿; demo 页面 0/4/7/12 亮色 + 页面0暗色 150% DPI 截图; 探针元素矩形逐项核对(Card 480/448、Empty、Popover 内容 288/256、Alert 文本 506); 弹层 z 序 A/B 截图(旧 HWND_TOP 不可见 / 新 TOPMOST 可见)。逐组件安装编译见下方回归记录。
- ✅ **标题栏按钮图标墨迹对齐**(2026-10-25, 用户报"最大化图标看着比最小化/关闭大"):
  - **根因**: lucide 24 网格里各图形墨迹占比差很多 —— minus 14/24、x 12/24(对角线 0.707)、square 18/24、copy/eye 20/24; 同一名义尺寸下闭合方框类比线条类大一圈, 还原态的 copy 比 square 还大。旧值(minus/x=20、square/自定义=16)是随手定的, 换算墨迹 square(12) 仍大于 X(10)。
  - **修法**: 常量改为**两档按墨迹相等校准**(`window.go`): `appWndIconLine=18`(minus/x, 墨迹 10.5)与 `appWndIconBox=14`(square/copy 与 AddCaptionButton 的 eye/moon 等满网格图形, 墨迹 10.5); 描边 2(24 网格)随尺寸缩放 1.5/1.17px, 与 shadcn 模板(16/12)同款比例; 18/36 = 图标占按钮宽一半也与模板相同。标题左侧应用图标用独立 `appWndTitleIcon=16`(Logo 不参与校准)。
  - **校准过程**: 探针用生产同款绘制路径渲染 6 组候选(20/16·20/15·20/14·18/14·18/13·16/12), PIL 量墨迹 + 放大目测 —— JPEG/抗锯齿噪声(±1 逻辑px)分辨不出 0.75px 级差异, 目测定案 18/14(范围与描边同时匹配, minus 不再是"粗短杠")。
  - 验收: demo 标题栏亮/暗两主题 + 最大化(restore 的 copy 图标)放大截图, 五键(eye/moon/─/□/×)体量一致; `go vet/test ./...` 与逐组件安装编译通过。

- ✅ **修复: 通知在缩放窗口时跑到左侧 + 页签行与卡片间距收窄**(2026-10-25, 用户报障):
  - **现象**: 通知显示中拖拽缩放窗口(或最大化/还原), 通知跑到窗口左侧 —— 与此前"切页后 Toast 跑到左下角"同源(DESIGN 96), 但那次只堵了"切页别调 `XWnd_AdjustLayout`"一个触发点。demo 里复现出的完整链条: 最大化让页签从两行变一行 → Tabs 延迟重排 → `reflowAncestors` 逐级升到 `XWnd_AdjustLayout` → 窗口级布局趟把窗口直属的绝对定位 Toast 当布局项排到流式位置(左侧)。
  - **为什么不能事后补救**: 探针实测 `XE_ADJUSTLAYOUT_END`(布局完成事件)在 1.4.3 **注册即崩**(事件总线与 `XEle_RegEventC` 两条路径都崩, 访问违例); `AfterFunc(0)` 延迟重摆又与组件自己的延迟重排(Tabs 换行同为 AfterFunc(0))存在排队竞态。探针还确认: 窗口内部布局在 Size 事件**之前**就把 toast 排到左侧(x=8), Size 事件里的重摆本身有效。
  - **修法(DESIGN 128)**: `toast.go` 新增 `pin(bodyRect)` —— 每次落位用 `XWidget_LayoutItem_SetPosition` 把通知元素四边钉成相对 Body 的 inset。探针实测钉位语义: ① 布局趟把钉住的项摆到钉位(缩放/重排后按新父尺寸重算), 不再流式; ② 钉在 `XEle_SetRect` 之后仍然存活(进出场动画逐帧挪动不清钉); ③ 四边都钉的项在布局趟里被拉伸到四边之间 —— insets 必须与元素矩形一致, 窗口 Size 事件里的 `layout()` 必须按新 Body 重钉, 否则缩放瞬间元素变形; ④ 只改钉不触发布局趟(对"干净"布局显式 AdjustLayout 是空操作), 落位仍靠 SetRect。钉住后任何来源的窗口级布局趟反而会把通知摆回正确位置。
  - **页签行与卡片间距**(同日报障"空白区域缩小"): PIL 逐像素实测页签块底边到卡片顶边 30 逻辑px = 页签行下边距 6 + 页面体上边距 24; 改为页签行下边距 0 + 页面体上边距 16(`newPage`), 总间隙 30 → 16, 与页内行距 `gap` 一致(复测 16.2)。
  - 验收: 常驻通知(长文本)下最大化 / 还原 / 拖拽缩放 / Win+方向键分屏四种缩放, 通知始终右下角; `go vet ./...` / `go test ./...` 全绿; 逐组件安装编译(toast 闭包 + 全量)通过。

- ✅ **demo 页签再拆分**(2026-10-25, 基础/表单/导航/反馈单页仍然过长):
  - **基础1** = 按钮 / 按钮组 / 切换按钮 / 切换按钮组; **基础2** = 徽章·分隔线·标签 / 头像 / 快捷键 / 加载指示器 / 骨架屏; **基础3** = 弹出层 / 轮播 / 卡片 / 比例容器; **基础4** = 滚动容器 / 列表项 / 手风琴。
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

- ✅ **批次 4: Sheet / Drawer / Menubar**(2026-10-25, 上游 tsx 同步):
  - **前置完成**: 本地 shadcn 参考实现补齐 `sheet.tsx` / `drawer.tsx`(vaul)/ `menubar.tsx`(取自 shadcn.com.cn registry), `pnpm add vaul`, App.tsx 反馈页加「抽屉」对照卡、导航页加「菜单栏」对照卡, `pnpm build` 通过。
  - **Sheet(★★)**: 复用 Dialog 覆盖层与模态栈(焦点陷阱/ESC/点遮罩/最小化挂起), 面板贴 Body 四边: 左右占满高、宽 min(3/4, 384), 上下宽度铺满、高按内容自撑; 遮罩 bg-black/50(比 Dialog 的 10% 深), 面板 bg-background + 贴边侧 1px border; 头部 p-4、页脚按钮纵向铺满; 打开/关闭从贴边侧滑入/滑出(300/200ms, 可用 WithSheetDuration 覆盖), 面板与直接子元素按偏移整体平移(第八节 120/125)。
  - **Drawer(★★)**: Sheet 的底部形态 + 顶部抓手(100×8 胶囊, 热区 100×24)、max-h-[80vh]、上方两角圆角; 抓手按住捕获鼠标, 下拖过面板高 1/3 关闭、否则回弹; 方向可改 top/left/right(抓手只在底部)。**踩坑**: 起拖时 DropTransitions 会删掉主 presence 键, 导致关闭动画的 done 永不执行、透明遮罩一直吞点击(第八节 121, 已修)。
  - **Menubar(★★★)**: 原生 `XMenu_Popup` 阻塞, 期间主窗口收不到鼠标移动, 无法实现 radix 的"打开后悬停其它触发区即切换", 所以弹层改自绘(复用 popoverPanel 阴影窗 + 自绘条目 Element); 条目模型与 Menu 共用 `menu_model.go`(MenuOpt 构造器原样可用), 支持条目/勾选/单选/分组标签/分隔线/快捷键/禁用/破坏性/前置图标/inset/子菜单; ↑↓/Home/End/Enter/←→/ESC/Tab 键盘导航, ESC 关闭后焦点还给触发区; 子菜单是第二个激活弹层, 父层 `keepOnBlur` 防止焦点转移时失焦关闭, 移出父/子矩形才收起(第八节 122~125)。
  - **demo**: 新增 **导航3**(菜单栏)与 **反馈3**(右侧表单 Sheet / 左侧导航 Sheet / 底部 Sheet + 底部 Drawer), 页签总数 **16**; 交互密集卡片放在页首, 规避第八节 118。
  - **验收**: `go build/vet/test ./...` 全绿; 逐组件安装 sheet / drawer / menubar 与 `add --all` 均编译通过(dialog / menu / alertdialog 的文件清单变更也一并回归); 150% DPI 亮色实机验证 —— Sheet 四向打开/遮罩/ESC/X/遮罩点击/焦点落输入框, Drawer 抓手拖拽关闭与短拖回弹、关闭后页面仍可点击, Menubar 打开/悬停切换/子菜单点击/勾选/禁用/键盘 ↓↓Enter、ESC 后 ↓ 重新打开同一菜单。
  - **新结论**: 面板滑入靠平移直接子元素(120); 拖拽关闭不能清主 presence 键(121); 原生菜单无法承载悬停切换, 弹层自绘(122); 子菜单必须激活 + 父层 keepOnBlur(123); 普通 Element 的 Auto 宽在布局里是 0(124); XWnd_GetRect 是逻辑像素(125); 原生 Edit 自带的两个滚动条子元素默认矩形在左上角(126)。
  - **复验(用户截图三问)**: ① **菜单栏字号没跟上** —— 触发区走 Button sm 预设是 13px, 上游是 text-sm(14): Button 加内部 `fontPx` 覆盖字段(不新增公开 API), Menubar 设 14 后 `FitWidth()` 重量宽, 实测触发区与弹层条目墨迹等高(150% DPI 下都是 11 图像像素), 高度仍按 sm 的 28。② **右侧面板输入框左上角出现两个灰条** —— 150% DPI 下本机未复现, 但探针确认原生 Edit 一创建就带 H/V 两个滚动条子元素, 矩形 (0,0,100,12) / (0,0,12,100) 起点就在左上角, 一旦被"内容超出自动显示"点亮就是截图里那块带箭头的灰条; 单行输入框构造时 `EnableAutoShowScrollBar(false)` + 两个滚动条 `Show(false)`(多行分支不受影响, 表单页文本域回归正常), 见第八节 126。③ **底部面板直角** —— 与上游一致: sheet.tsx 四向都没有圆角, 只有 vaul 的 Drawer 有 rounded-t-lg / rounded-b-lg(我们的 Drawer 有圆角, Sheet 没有)。
  - **二轮复验(用户再报两点)**: ① **灰条"第二次打开才出现"** —— 探针实测: 单行 Edit 的两个滚动条子元素是**控件首次真正显示时才补建**的, 第一次整树 `Show(true)` 枚举不到它们所以正常, 关闭再开就被 `showEleTree` 一起点亮(矩形 (0,0,100,12)/(0,0,12,100) → 画在输入框左上角)。修两处: 单行输入框构造时关掉自动显示并隐藏滚动条 + `showEleTree` 跳过 `XC_SCROLLBAR` 子树(只做前者仍会翻车), 见第八节 126; 探针复验第二次打开 hShow/vShow 仍为 false, 实机第二次打开截图干净。② **关窗后进程不退** —— 菜单栏的弹层窗口是懒创建、只隐藏不销毁的, 主窗口关了它们还在 → 消息循环不退出(与 Tooltip 同一个坑, 第八节 77); 补 `bar.AddEvent_Destroy` + 宿主窗口 `WM_DESTROY` 两级销毁钩子(递归销毁子菜单弹层)。**A/B 实测**: 两个钩子都停用 → 关窗后枚举到 2 个残留 XCGUI 窗口、进程不退; 装回钩子 → 开过顶级菜单 + 子菜单后关窗, 进程正常退出。
  - **三轮复验(子菜单移不过去)**: 子菜单与父菜单之间有一条缝, 鼠标穿过就关掉 —— ① 定位把"重叠 4"写成了"间隙 4"(符号错); ② 光标归属按**卡片矩形**判定, 而弹层窗口的透明阴影带也属于该窗口, 光标一进阴影带父面板就收不到 MouseMove, 于是"两边都不在"被误判为离开。改为与父面板重叠 4(radix `alignOffset: -4` 的本意)+ 判定按**窗口矩形**(卡片矩形外扩阴影带宽), 见第八节 127。实机验证: 悬停「分享」→ 移入子菜单条目保持高亮 → 移回父面板其它行只收子菜单、父菜单留着。

- ✅ **批次 5: Navigation Menu**(2026-10-27, 上游 tsx 同步):
  - **前置完成**: 本地 shadcn 参考实现补齐 `navigation-menu.tsx`(取自 registry, radix-nova), App.tsx 导航页加对照卡(共享视口 + 两个菜单 + 一个普通链接), `pnpm build` 通过。
  - **组件(`components/navigationmenu.go`)**: `NewNavigationMenu(parent)` + `AddMenu(title, func(c *ui.NavigationMenuContent), WithNavigationMenuWidth(w))` + `AddLink(title, fn)`; 触发区/顶级普通链接是"原生 Button 行为 + 全自绘外观"(ghost 底、focus-visible 环、chevron 300ms 旋转 180°); **所有菜单共用一个非激活 popoverPanel 视口** —— 贴列表左缘下方 6px, 宽高随当前菜单内容变化, 切换时不收起重弹; 内容页用 `c.Link(标题, 说明, fn)` / `c.LinkIn(父容器, …)`(多列网格用 `xui.HStack` + `xui.ItemWeight`) 或 `c.Handle()` 放任意控件。
  - **交互**: 悬停 200ms 打开、离开 150ms 收起、关闭后 300ms 内切其它触发区立即打开(之后恢复 200ms); 点击触发区开合; ESC/Tab/点击关掉后指针还停在触发区上时不会被一次轻微移动重新打开(离开再进入恢复, radix 的 wasEscapeClose/wasClickClose 语义); 打开后悬停其它触发区立即切换; 切换时视口立即 resize + 新内容从切换方向滑入 208px 淡入(200ms); ESC / Tab / 宿主窗口失焦 / 移动 / 缩放关闭; 视口不抢焦点(悬停打开不会打断页面里的输入)。
  - **键盘**: ←→ 在顶级项之间移动焦点、↓/Enter/Space 打开、↓ 进入内容; 内容里 ↑↓←→/Home/End 移动链接高亮、Enter 激活、ESC 关闭; `Value/SetValue/OnValueChange`。
  - **新结论(DESIGN 137~139)**: ① 非激活弹层里普通元素拿不到 Stay/hover、也收不到 MouseMove —— 打开期间用 30ms 观察器 + `XWnd_GetStayEle` 做链接悬停与收起判定; ② KeyDown 里同步 `XWnd_SetFocusEle` 会让按键重派发给新焦点, ←→ 一次跳两项, 改动推迟到消息循环; ③ `panel.resize` 后的布局趟会复位手动滑入位移, 动画延后到布局落定再起、每帧重写偏移。
  - **验收**: 探针(临时 main 包)全绿 —— 200ms 延迟/150ms 收起、打开中悬停其它触发区立即切换、关闭后 300ms 内立即打开、各菜单视口宽高(400/520/300, C 菜单 136 高)、视口左缘对齐列表且间距 6、视口非激活(前台窗口不变)、指针进入内容不收起/离开后收起、内容滑入矩形与 alpha(213→150→8)、宿主缩放关闭; demo 导航3页亮/暗 150% DPI 实机 —— 悬停打开、链接行 hover 底色、键盘高亮环、两列网格、普通链接点击、ESC/Tab、点触发区关闭; `go build/vet/test ./...` 全绿; `xui add navigation-menu` / `add menubar` / `add --all` 闭包编译通过。

- ✅ **批次 6: Carousel**(2026-10-28, 上游 tsx 同步):
  - **前置完成**: 本地 shadcn 参考实现补齐 `carousel.tsx`(registry radix-nova, 依赖 embla-carousel-react, 已 `pnpm add`), App.tsx 基础页加对照卡(4:3 宽卡 + 循环 + 两侧圆形钮), `pnpm build` 通过。
  - **组件(`components/carousel.go`)**: `NewCarousel(x,y,cx,cy,parent, opts...)` + `AddSlide(func(s *ui.CarouselSlide))`; 每页宽 = 视口宽 − 16(-ml-4 + pl-4 的等价写法), 整页位移; 上/下张 = outline + icon-sm + rounded-full, 放在内容外侧 48 的槽位里(等价 shadcn 示例外层 px-12), 边界禁用、loop 始终可用; 循环用"模总长"折回, 首尾相接无克隆页; 300ms 位移过渡走 `xui.Presence`, 连点用 `pending` 目标页(第二次点击不会被旧 index 吃掉); 拖拽从窗口级事件起手 + `SetCaptureEle` 跟手, 松手按最近一页吸附(6px 阈值区分点击); ←/→ 在轮播内控件获焦时翻页(上游 onKeyDownCapture 语义); 纵向用上下箭头, 选项 `WithCarouselLoop/Vertical/Controls/Gap/Duration/InitialIndex/Autoplay/OnChange`; `SetIndex/Next/Prev/SetLoop/SetAutoplay/PrevButton/NextButton`。
  - **自动翻页**: 非 shadcn 核心(embla autoplay 插件), 按插件默认语义实现 —— 用户拖动/点箭头/按键后停止(stopOnInteraction), 循环模式一直转、非循环到末页停, 隐藏页(尺寸 0)不跑; 定时器用元素 `XEle_SetXCTimer`, 销毁时一并停。
  - **新结论(DESIGN 140/141)**: ① 元素鼠标事件不冒泡 —— 指针在子元素上时父元素收不到任何 Mouse 事件, 只有子元素与窗口各一份; 拖拽必须窗口级起手, 越阈值后 `SetCaptureEle(viewport)` 跟手, 窗口外松手没有 UP、用 MouseMove 的 `nFlags & 1` 收尾; 视口的 LButtonDown 不能无条件清拖拽预备状态(窗口与元素会各收到一次)。② Go `math.Mod` 符号跟被除数, 循环轮播折回必须先归正(否则第一张永远进不来); `carousel_test.go` 固定了 0/末页/拖过末页三种位置。
  - **验收**: 探针全绿 —— 初始几何(视口 840/滑片 824, 步进 840)、300ms 中间帧、末页按钮禁用、连点 0→2、SetIndex 跳转、循环 2→0 与 0→2(中间帧第一张从右侧/左侧进入)、自动翻页 400ms 后变化、用户交互后停止、纵向滑片 280×68 + 上下按钮; demo 基础3 实机 150% DPI 亮/暗 —— 点箭头、拖拽 240 松手吸附、键盘 ←/→、连点三次 2→3→4→1、切主题、像素测量(滑片 974×200 @150%, 按钮 28, 圆点 6/间隙 6, 槽位 48); `go build/vet/test ./...` 全绿; `xui add carousel` 与 `add --all` 闭包编译通过。

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
- ✅ **Item / Breadcrumb / Kbd / Skeleton / Spinner**(2026-10-24): 补齐 shadcn 批次 1 剩余五项 —— `ui.NewItem`(Media/Content/Actions/Header/Footer + 3 变体 × 2 尺寸 + ItemGroup/Separator/Title/Description/Media)、`ui.NewBreadcrumb`(Item/Link/Page/Separator/Ellipsis, 链接 hover 过渡 + OnClick)、`ui.NewKbd` / `NewKbdGroup`、`ui.NewSkeleton`(2s 脉冲, 运行时新增 `xui.Pulse`)、`ui.NewSpinner`(Loader2 1s/圈, 运行时 `xui.Spin`); demo 基础1页第三行 + 基础2页列表项 + 导航页面包屑, 亮/暗 150% DPI 截图验收, Spinner 逐帧确认旋转、Skeleton 逐帧确认脉冲、Breadcrumb hover 过渡与 OnClick 回调确认; 三处新结论: `XEle_SetPadding` 可替代外边距 pad、Item 内容区要按实测宽度锁 Fixed、Label 精确按 TextWidth 设宽会被原生 Button 内边距截断(见第八节 103~105); 另修暗色 ItemMuted 上徽章露出深色方底(Badge 不再给不透明变体铺 Card 方底), 以及首页多动画掉帧/GPU 高(Spin/Pulse 共享宿主窗口定时器、按动画元素矩形分组局部重绘(XWnd_RedrawRect)、60fps(前台暂停已于 2026-10-26 取消, 见最新条); 最小化/隐藏/滚出视口自动停表, app GPU ≈0.5%、DWM 后台 ≈0.03%、前台约 10~14%(30fps 实测仍 ~10%, 已回退 60fps 保证流畅), 见第八节 106); 逐组件安装 spinner / skeleton / kbd / breadcrumb / item 与 `add --all` 均编译通过
- ✅ **Empty / Field**(2026-10-24): 新增空状态容器 `ui.NewEmpty`(Header/Content/Body 槽位 + 图标底板/无底板媒体 + 标题/描述)与表单字段布局 `ui.NewField`(Label/Title/Description/Content/SetInvalid + FieldSet/Group/Legend/Separator/Error);demo 反馈页 + 表单1页亮暗验收,校验错误区显示/隐藏时卡片高度正确伸缩;横向 Content 的嵌套布局拿不到 Weight/Percent 宽,默认改 Fixed 240(见第八节 101);隐藏错误区后旧位置文字会留重影,重排结束补整窗重绘(见第八节 102);逐组件安装 empty / field 与 `add --all` 均编译通过
- ✅ **Menu 右键触发 / 前置图标 / inset 扩展**(2026-10-23): 新增 `NewContextMenu(元素, build)`(窗口级右键命中触发区, 子元素上右键也生效; `SetEnabled` / `OnOpenChange`)、`MenuIcon`(条目/Sub 前置图标)、`Label` / `Sub` 支持 `MenuInset`;Check/Radio 未选中也保留右侧 24(`pr-8`); 弹层宽度改为**逐条按自绘字体量宽**(原生字体与自绘不一致, 内容自适应的长中文会被裁, 见第八节 99);demo 导航页新增虚线右键触发区 + shadcn ContextMenuDemo 同构菜单。**菜单进场动画整个去掉**(用户录屏: 整窗 alpha 会透出背景文字, 改成"面板不透明 + 只淡内容"又会先闪一帧空白面板, 两条路都闪; 位移/缩放会崩)—— 现在与原生菜单一样弹出即全内容, 见第八节 99; 顺带修正触发区文字居中对齐(`XDraw_SetFont` 会重置对齐, 顺序必须是先字体后对齐, 见第八节 100), 并去掉 demo 里 `人员` 分组标签的 `MenuInset`(本菜单勾选在右侧, 条目文字从 6 起, 加了 inset 反而错位); 右键触发区补 hover / 展开反馈(`bg-accent` + 虚线变 `Ring`, `OnOpenChange` 驱动, 透明元素先铺 `t.Card` 擦底), 让右键不再像凭空出现, 与左侧按钮菜单的 hover→展开节奏一致。其余验收:亮/暗两主题右键触发、子菜单、勾选/单选状态持久、破坏性样式;独立探针验证自适应长文本不裁 + 短菜单 128 下限;`go test ./components -run Menu` 通过
- ✅ **P0 运行时**:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter、字体缓存(单位坑见 31)、focus-visible 焦点来源跟踪(16)
- ✅ **P1 组件**:Button(6 变体×8 尺寸档)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**:Switch(2 尺寸)、Progress、Slider、Tabs(default/line)、Tooltip、ComboBox、图标系统
- ✅ **Slider 渲染缺陷彻底修复**:悬浮/拖动冒方块、两端 ring 被裁、灰环残留(根因与结论见 25/26/27/28)
- ✅ **CLI**:`init / add / list`(registry.json 驱动,组件源码拷入用户项目);`update` 与 `--all` 后续补齐(见上面 2026-10-30 那条与 2026-11-03 的参数审计批次)
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
  - **`--chart-1..5` / `--sidebar-*` 令牌进 `xui.Theme`**(Chart1..5 / Sidebar*): Chart 按分类序号取 Chart1..5(超额循环), Sidebar* 供侧栏/用户应用取用;
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
  - **载体**: 窗口内覆盖元素(每条通知一个自绘 Element, 无子元素), 右下角堆叠、跟随窗口移动/缩放/最小化
    (落位时用布局项置位置把元素四边钉成相对 Body 的 inset, 扛住一切窗口级布局趟, 见 DESIGN 128);
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
  - `xui init [ui目录]` 旧行为不变(空目录改走完整项目是后来的事, 见本文顶部 2026-11-04 那条); 参数解析、脚手架、dry-run、diff、过滤/分页均有单测。
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
