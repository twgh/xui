package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 内嵌附加件的几何(逻辑像素)。取值对齐 shadcn 的 input-group:
// 图标 size-4(16)、按钮 size-6(24) + 圆角 calc(--radius - 5px)、
// 按钮间距 gap-1(4)、尾部 addon 内缩 pr-1.5(6)、文字与附加件之间 6。
const (
	inputPadX int32 = 10 // 文字内边距(原 SetBorderSize 的左右值)
	// inputEdge 是内嵌按钮与壳体内缘的距离。别取 4: 焦点环/校验环是**画在元素内部**的
	// (见 onPaint 的 inset=3), 按钮的悬停底色从 4 开始会正好压住环的内描边, 看起来
	// "悬停方块啃掉了一块边框"(实测)。取 6 = 环 3 + 留 3, 与 shadcn 的 pr-1.5 一致。
	inputEdge      int32 = 6  // 内嵌按钮与壳体内缘的距离
	inputGap       int32 = 4  // 内嵌件之间的间距
	inputTextGap   int32 = 6  // 文字与附加件之间的间距
	inputIconSz    int32 = 16 // 前置图标边长
	inputBtnSz     int32 = 24 // 内嵌 ghost 按钮边长(size-6)
	inputBtnR      int32 = 5  // 内嵌按钮圆角(calc(var(--radius)-5px))
	inputGlyphSz   int32 = 14 // 按钮里的图标边长
	inputCounterSz int32 = 12 // 计数文字字号(text-xs)
	inputStepColW  int32 = 20 // 上下三角形态的箭头列宽
	inputArrowBox  int32 = 8  // 箭头 SVG 的绘制方块边长(正方形, 图标管线按短边铺满)
	inputArrowPad  int32 = 2  // 箭头列上下端的留白(叠加列在输入框里自带的 4px 边距, 上下各 ~6px)
)

// 步进器的上下箭头, 24 网格 SVG path(`!` 前缀 = 填充形状, 走 FillGeometry)。
// 三角占满 24 宽、高 19.2(上下各留 2.4), 三个角用 Q 曲线倒约 2.4 的圆角 ——
// 走图标管线(D2D 几何 + 按路径缓存)画出来的圆角是真正的曲线,
// 手算多边形逼近圆弧在 10px 尺寸下不是歪就是糊(实测对比)。
const (
	arrowUpSVG   = `!M14.02 5.64 L21.72 17.95 Q24 21.6 19.67 21.6 L4.33 21.6 Q0 21.6 2.28 17.95 L9.98 5.64 Q12 2.4 14.02 5.64 Z`
	arrowDownSVG = `!M14.02 18.36 L21.72 6.05 Q24 2.4 19.67 2.4 L4.33 2.4 Q0 2.4 2.28 6.05 L9.98 18.36 Q12 21.6 14.02 18.36 Z`
)

// inputPart 是输入框内部自绘件的命中区标识(鼠标悬停/点击都按它分派)。
type inputPart int32

const (
	partNone inputPart = iota
	partClear
	partToggle
	partStepUp
	partStepDown
)

// inputPartRect 是一个内嵌件在元素客户区里的矩形(逻辑像素)。
// 每次 paint 重算并存进 Input.parts, 供点击命中与悬停判定复用 ——
// 不用"按下标反推坐标"是因为内嵌件是按宽度从右往左排的, 反推容易和绘制走样。
//
// dim = 该内嵌件当前处于"不可用"态(如步进器到值域边界): 点击仍然要被吞掉
// (防双击穿透到原生 Edit, 见 onLButtonDown), 但**不触发按压反馈也不执行动作**。
type inputPartRect struct {
	part inputPart
	rc   xc.RECT
	dim  bool
}

// inputStep 是步进器的值域与步进。decimals 由 step 的小数位推出,
// 保证 0.1 的步长不会写出 "1.2000000000000002" 这种尾巴。
type inputStep struct {
	min, max, step float64
	decimals       int
}

func (s *inputStep) clamp(v float64) float64 {
	if v < s.min {
		return s.min
	}
	if v > s.max {
		return s.max
	}
	return v
}

// snap 把 v 夹到值域并吸附到 min + n*step 的刻度上。
func (s *inputStep) snap(v float64) float64 {
	if s.step > 0 {
		v = s.min + math.Round((v-s.min)/s.step)*s.step
	}
	return s.clamp(v)
}

func (s *inputStep) format(v float64) string {
	return strconv.FormatFloat(s.clamp(v), 'f', s.decimals, 64)
}

// Input 自绘单行输入框: 载体为原生 Edit, 光标/输入法/选择等行为全部原生,
// 本组件只绘制圆角壳体与焦点框, 并实时同步文字/占位符颜色。
//
// 与 shadcn input.tsx 对齐: h-8, border-input, 聚焦时 border-ring +
// ring-ring/50(文本输入控件点击聚焦也显示, 同浏览器 :focus-visible 行为),
// 禁用为 bg-input/50。
//
// 单行模式下还可以在壳体内部塞"附加件"(仿 shadcn 的 input-group, 全部自绘在
// 文本区两侧, 不建子元素 —— 子元素会被原生 Edit 的绘制盖住, 悬停也会互相打架):
//
//	WithLeadingIcon   左侧图标(如搜索/邮箱)
//	WithPrefixText    左侧文字(如 "https://"、"$")
//	WithSuffixText    右侧文字(如 ".com"、单位)
//	WithCounter       右侧计数 "n/max", 超出上限变红
//	WithClearButton   右侧清除按钮(有内容才画)
//	WithPasswordToggle 右侧密码显隐按钮(眼睛)
//	WithStepper       右侧 −/+ 步进按钮 + 数字语义
//
// 文本区的左右内边距由附加件宽度自动算出(syncPadding), 文字永远不会跑到
// 按钮底下; 多行(文本域)形态忽略全部附加件。
type Input struct {
	*widget.Edit

	placeholder string
	multiline   bool
	invalid     bool
	textColor   uint32 // 上次下发给原生 Edit 的字色(避免在 paint 里反复设置)

	leadingIcon string // 前置图标(lucide path)
	prefix      string // 前置文字
	suffix      string // 尾部文字
	clearBtn    bool   // 显示清除按钮
	passToggle  bool   // 显示密码显隐按钮
	password    bool   // 当前是否处于密码模式(true = 内容被遮挡)
	counterMax  int    // >0 时显示 "n/max" 计数

	step    *inputStep      // 非空 = 步进器
	lastNum float64         // 最近一个有效数值(输入不合法时回退用)
	onStep  func(v float64) // 步进回调
	// stepArrows 用"上下三角"画步进按钮(原生 number 输入框那种), 默认是并排的 −/+
	stepArrows bool
	// stepAuto 步进按钮只在悬停输入框或输入框有焦点时显示(移开/失焦隐藏)
	stepAuto bool

	hover      inputPart       // 当前悬停的内嵌件
	hoverInput bool            // 鼠标是否停在输入框上(stepAuto 用)
	pressed    inputPart       // 当前按下的内嵌件(partNone = 无; 按压反馈: 图标缩小)
	parts      []inputPartRect // 内嵌件命中区(paint 时重算)
}

// InputOption 组件配置项。
type InputOption func(*Input)

// WithPlaceholder 设置占位文本。
func WithPlaceholder(s string) InputOption { return func(i *Input) { i.placeholder = s } }

// WithMultiline 启用多行(文本域)模式。
func WithMultiline() InputOption { return func(i *Input) { i.multiline = true } }

// WithInputInvalid 设置校验失败态(对应 shadcn 的 aria-invalid):
// border-destructive + ring-3 destructive/20(暗色 /40)。与聚焦同时成立时红色优先。
func WithInputInvalid() InputOption { return func(i *Input) { i.invalid = true } }

// WithLeadingIcon 在文字区左侧嵌一枚图标(lucide path, 如 icons.IconSearch)。
// 图标 size-4(16), 颜色 muted-foreground(禁用时再乘 50%)。
func WithLeadingIcon(icon string) InputOption { return func(i *Input) { i.leadingIcon = icon } }

// WithPrefixText 在文字区左侧嵌一段弱化文字(如 "https://"、"$"), 与输入同字号。
func WithPrefixText(s string) InputOption { return func(i *Input) { i.prefix = s } }

// WithSuffixText 在文字区右侧嵌一段弱化文字(如 ".com"、单位), 位于按钮与计数左边。
func WithSuffixText(s string) InputOption { return func(i *Input) { i.suffix = s } }

// WithClearButton 在右侧嵌一个清除按钮(幽灵图标钮, 内容为空时不画但保留宽度,
// 避免输入过程中文字区左右跳动)。点击清空内容并触发 OnChange。
func WithClearButton() InputOption { return func(i *Input) { i.clearBtn = true } }

// WithPasswordToggle 启用密码输入 + 右侧眼睛图标切换明/密文。
// 图标语义同主流做法: 当前是密文时画眼睛(点它显示明文), 已是明文时画划线眼睛。
func WithPasswordToggle() InputOption { return func(i *Input) { i.passToggle = true } }

// WithCounter 在右侧显示 "已输入字数/上限"(text-xs muted), 超出上限变红。
// 只做提示, 不拦截输入(原生 Edit 没有 maxlength, 硬截断会和输入法的候选上屏打架)。
func WithCounter(max int) InputOption {
	return func(i *Input) {
		if max > 0 {
			i.counterMax = max
		}
	}
}

// WithStepper 把输入框变成步进器: 右侧 −/+ 按钮、↑/↓ 键、失焦/回车时校正数值。
// min/max 是闭区间值域, step 是步长(<=0 按 1); 小数位数由 step 推出。
// 输入过程中允许任何文本(含负号/小数点), 只有步进、失焦或回车才解析并规范化:
// 超出值域夹取, 不落在刻度上吸附到最近的刻度, 解析失败则回退到最近一次有效值。
func WithStepper(min, max, step float64) InputOption {
	return func(i *Input) {
		if step <= 0 {
			step = 1
		}
		if max < min {
			max = min
		}
		i.step = &inputStep{min: min, max: max, step: step, decimals: decimalsOf(step)}
	}
}

// WithStepperArrows 把步进按钮画成"上下三角"(原生 number 输入框那种纵向双箭头),
// 而不是右侧并排的 −/+ 幽灵钮。需配合 WithStepper 使用。
func WithStepperArrows() InputOption { return func(i *Input) { i.stepArrows = true } }

// WithStepperAutoHide 让步进按钮只在"鼠标停在输入框上"或"输入框有焦点"时出现,
// 移开鼠标且失焦后隐藏。隐藏时**保留占位**(文字与其它附加件不会跳), 该区域的点击
// 落到文本区(同浏览器: 悬停才出现微调箭头, 点空白处就是放光标)。
func WithStepperAutoHide() InputOption { return func(i *Input) { i.stepAuto = true } }

// decimalsOf 取 step 的小数位数(0.1 → 1, 1 → 0, 0.25 → 2), 上限 6 位。
func decimalsOf(step float64) int {
	s := strconv.FormatFloat(step, 'f', -1, 64)
	n := 0
	if k := strings.IndexByte(s, '.'); k >= 0 {
		n = len(s) - k - 1
	}
	if n > 6 {
		n = 6
	}
	return n
}

// NewInput 创建输入框。cy 传 0 时使用默认高度 32。
func NewInput(x, y, cx, cy int32, hParent int, opts ...InputOption) *Input {
	if cy <= 0 {
		cy = 32
	}
	in := &Input{Edit: widget.NewEdit(x, y, cx, cy, hParent)}
	if in == nil {
		return nil
	}
	for _, o := range opts {
		o(in)
	}

	if in.multiline {
		// 文本域形态: 上下都是正文, 没有可以放按钮的边, 附加件一律忽略
		in.clearBtn, in.passToggle, in.step = false, false, nil
		in.leadingIcon, in.prefix, in.suffix, in.counterMax = "", "", "", 0
		in.EnableMultiLine(true).EnableAutoWrap(true).EnableAutoShowScrollBar(true)
		// 多行内容溢出时出滚动条: 收敛成 shadcn 规格的细圆头条(见 scrollbar.go)
		in.SetScrollBarSize(scrollBarW)
		showScrollBarOnHover(in.Handle, in.GetScrollBarVObj())
	} else {
		if in.passToggle {
			in.EnablePassword(true)
			// 遮挡符用实心圆点(U+25CF), 与浏览器默认的 -webkit-text-security: disc
			// 以及 Windows 原生密码框一致; xcgui 的默认值是 '*' —— 星号在大多数字体里
			// 只有 8px 左右的字形且贴基线, 看起来像"小一号的字"
			xc.XEdit_SetPasswordCharacter(in.Handle, '●')
			in.password = true
		}
		if in.step != nil {
			in.lastNum = in.step.snap(0)
		}
	}

	// 灭掉原生外观
	in.EnableBkTransparent(true)
	in.EnableDrawBorder(false).EnableDrawFocus(false)
	// 关掉"获得焦点自动全选"(xcgui 默认开): 内嵌按钮点完都要把焦点交回输入框,
	// 全选会留下蓝色的选中态, 取消选中后还可能残留一层没擦掉的选中文字。
	in.EnableAutoSelAll(false)
	// 文字内边距: 左右按附加件宽度算, 避免文字贴到圆角/跑到按钮底下
	in.syncPadding()

	// 文字颜色需要实时同步给原生 API, 订阅主题切换
	in.applyTheme(xui.Current())
	xui.OnThemeChangeFor(in.Handle, func(t *xui.Theme) { in.applyTheme(t) })

	in.AddEvent_Paint(in.onPaint)
	in.AddEvent_LButtonDown(in.onLButtonDown)
	// 双击必须单独吞: 连点时第二次**按下**是 WM_LBUTTONDBLCLK, 不走 LButtonDown,
	// 原生 Edit 会拿它去做"选中单词" —— 只在 LButtonDown 里吞是挡不住的(实测)。
	in.AddEvent_LButtonDBClick(in.onLButtonDown)
	in.AddEvent_LButtonUp(in.onLButtonUp)
	if in.step != nil {
		in.AddEvent_KeyDown(in.onKeyDown)
		in.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
			in.commitNumber()
			return 0
		})
	}
	if in.clearBtn {
		// 清除按钮的显隐跟着内容走: 文字变化时要重绘附加件区
		in.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
			in.Redraw(false)
			return 0
		})
	}
	in.trackHover()
	xui.AttachWindow(in.Handle)
	return in
}

func (i *Input) applyTheme(t *xui.Theme) {
	// 字号/字族必须显式下发给原生 Edit: 编辑框不继承窗口字体, 用的是 xcgui 内置默认
	// 字体(宋体小字号), 不设就会出现"输入框里的字比周围小一圈、数字挤在一起"(实测)。
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		i.SetFont(f)
	}
	i.syncPadding() // 前后缀/计数的宽度按主题字体量, 换字体后要重算
	i.syncTextColor()
	if i.placeholder != "" {
		i.SetDefaultText(i.placeholder)
		i.SetDefaultTextColor(uint32(t.MutedForeground))
	}
	i.SetCaretColor(uint32(t.Primary))
	if i.multiline {
		// 多行滚动条: 轨道透明(露出壳体底色), 滑块走嵌入式皮肤(前景色淡化 + 悬停加深)
		styleScrollBar(i.GetScrollBarVObj(), 0, embedScrollBarSkin(t))
	}
	i.Redraw(false)
}

// syncTextColor 把当前应有的字色下发给原生 Edit(字色只能通过 SetTextColor 同步)。
// 禁用态整体 opacity-50: 文字色 = 前景色与卡片底色对半混(原生文本不吃 alpha, 只能预混)。
// 值没变就不重复调用 —— 它可能在 paint 回调里被调用, 要避免自激重绘。
func (i *Input) syncTextColor() {
	t := xui.Current()
	c := uint32(t.Foreground)
	if !i.IsEnable() {
		c = uint32(t.Foreground.Mix(t.Card, 0.5))
	}
	if c == i.textColor {
		return
	}
	i.textColor = c
	i.SetTextColor(c)
}

// ===== 附加件 =====

// inputFont 取主题字体在指定逻辑像素字号下的句柄。
func inputFont(t *xui.Theme, size int32) int {
	return xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: size})
}

// inputTextWidth 量测附加件文字的单行宽度(必须与绘制同字体, 否则内边距会差一两像素)。
func inputTextWidth(t *xui.Theme, s string, size int32) int32 {
	if s == "" {
		return 0
	}
	return xui.TextWidth(s, inputFont(t, size))
}

// counterReserve 是计数区预留的文字(取最宽形态 "max/max")。
// 预留固定宽度而不是按当前字数动态量宽: 否则每输入一个字右边距都变一次, 文字会左右抖。
func counterReserve(max int) string {
	return fmt.Sprintf("%d/%d", max, max)
}

// trailingInset 是尾部内嵌件与壳体内缘的距离: 按钮贴边 4(同 shadcn addon 的 pr-1),
// 文字类(后缀/计数)按文字内边距 10 —— 与左侧图标的内缩对称, 否则 ".com" / "15/120"
// 会紧贴右边框(看起来"没有边距")。
func (i *Input) trailingInset() int32 {
	if i.step != nil || i.passToggle || i.clearBtn {
		return inputEdge
	}
	return inputPadX
}

// syncPadding 把文字区的左右内边距设为"基础值 + 附加件占位"。
// 只在构造/换主题时调用 —— 不要在 paint 里调, SetBorderSize 会触发重排与重绘。
func (i *Input) syncPadding() {
	if i.multiline {
		// 文本域必须自己给四周内边距: 原生 Edit 默认内边距为 0, 正文会压住圆角描边、
		// 纵向滚动条也会贴着右边框(多行没有附加件, 所以这里不用算宽度)。
		i.SetBorderSize(inputPadX, 6, inputPadX, 6)
		return
	}
	t := xui.Current()

	left := inputPadX
	if i.leadingIcon != "" {
		left += inputIconSz + inputTextGap
	}
	if i.prefix != "" {
		left += inputTextWidth(t, i.prefix, t.FontDefault.Size) + inputTextGap
	}

	// 尾部各项自右向左的宽度(必须先步进块、再眼睛、再清除, 与 paintAddons 同序)
	var items []int32
	if i.step != nil {
		if i.stepArrows {
			items = append(items, inputStepColW)
		} else {
			items = append(items, inputBtnSz*2+inputGap)
		}
	}
	if i.passToggle {
		items = append(items, inputBtnSz)
	}
	if i.clearBtn {
		items = append(items, inputBtnSz)
	}
	if i.counterMax > 0 {
		items = append(items, inputTextWidth(t, counterReserve(i.counterMax), inputCounterSz))
	}
	if i.suffix != "" {
		items = append(items, inputTextWidth(t, i.suffix, t.FontDefault.Size))
	}

	right := inputPadX
	if len(items) > 0 {
		total := int32(0)
		for k, w := range items {
			if k > 0 {
				total += inputGap
			}
			total += w
		}
		right = i.trailingInset() + total + inputTextGap
	}
	// 上下不对称各 1px: 原生 Edit 把行盒居中在 text rect 里, 而数字/字母的墨迹只占
	// 上半部分(下半是 descender 空白), 对称留白时会看起来"文字偏上、下面空得多"
	// (实测 32 高输入框: 上方留白 14 物理像素、下方 17)。把 text rect 整体下移 1 逻辑
	// 像素后上下都是 15.5, 与自绘前后缀(按整高 Vcenter)也对齐了。
	i.SetBorderSize(left, 7, right, 5)
}

// partAt 返回元素客户区坐标 (x,y) 落在哪个内嵌件上(含其 dim 标记)。
func (i *Input) partAt(x, y int32) (inputPartRect, bool) {
	for _, s := range i.parts {
		if x >= s.rc.Left && x < s.rc.Right && y >= s.rc.Top && y < s.rc.Bottom {
			return s, true
		}
	}
	return inputPartRect{part: partNone}, false
}

// setHover 切换悬停的内嵌件并立即重绘(延迟重绘会被合并, 高亮要等到鼠标停下才出现)。
func (i *Input) setHover(part inputPart) {
	if i.hover == part {
		return
	}
	i.hover = part
	i.Redraw(true)
}

// trackHover 用**窗口级** MouseMove 跟踪悬停的是哪个内嵌件。
//
// 为什么不用元素级 MouseStay/MouseLeave: 原生 Edit 只在鼠标进入时上报一次
// MouseMove(实测), 之后在文本区/按钮之间移动都收不到, 分不出"悬停在 − 还是 +";
// 而 MouseLeave 在部分形态下也不来(见 DESIGN.md 第八节 21)。
// 窗口级每次移动重算一次最稳: GetStayEle 判定鼠标是否还在本元素上,
// 再用 GetWndClientRect 把窗口坐标换成元素内坐标。
func (i *Input) trackHover() {
	if i.multiline || (i.step == nil && !i.passToggle && !i.clearBtn) {
		return
	}
	hWindow := xc.XWidget_GetHWINDOW(i.Handle)
	if hWindow == 0 {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	// setInput 记录"鼠标是否在输入框上"(stepAuto 用它决定步进按钮的显隐),
	// 翻转时要立即重绘 —— 只重算 part 不会触发, 因为显隐跟 part 无关。
	setInput := func(inside bool) {
		if i.hoverInput == inside {
			return
		}
		i.hoverInput = inside
		if i.stepAuto {
			i.Redraw(true)
		}
	}
	w.AddEvent_MouseMove(func(hWin int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		inside := xc.XWnd_GetStayEle(hWin) == i.Handle
		setInput(inside)
		if !inside {
			// 按住拖出输入框: 按压反馈跟着取消(松开时也不会再命中)
			i.setPressed(partNone)
			i.setHover(partNone)
			return 0
		}
		var rc xc.RECT
		xc.XEle_GetWndClientRect(i.Handle, &rc)
		pr, _ := i.partAt(pPt.X-rc.Left, pPt.Y-rc.Top)
		i.setHover(pr.part)
		return 0
	})
	w.AddEvent_MouseLeave(func(hWin int, pbHandled *bool) int {
		setInput(false)
		i.setPressed(partNone)
		i.setHover(partNone)
		return 0
	})
}

// paintAddons 画壳体内部的附加件(图标/前后缀/计数/按钮)并登记命中区。
// 绘制顺序无所谓(位置互不重叠), 但命中区与 syncPadding 的宽度累加必须同序。
func (i *Input) paintAddons(p *xui.Painter, t *xui.Theme, st xui.State) {
	i.parts = i.parts[:0]
	if i.multiline {
		return
	}
	base := t.MutedForeground
	if st.Disabled {
		base = base.ScaleAlpha(0.5) // disabled:opacity-50
	}
	cy := p.Rc.Top + p.H()/2

	// ---- 前置: 图标 / 前缀文字 ----
	x := p.Rc.Left + inputPadX
	if i.leadingIcon != "" {
		p.Icon(i.leadingIcon, &xc.RECT{
			Left:   x,
			Top:    cy - inputIconSz/2,
			Right:  x + inputIconSz,
			Bottom: cy + inputIconSz/2,
		}, base, 2)
		x += inputIconSz + inputTextGap
	}
	if i.prefix != "" {
		w := inputTextWidth(t, i.prefix, t.FontDefault.Size)
		p.Draw.SetFont(inputFont(t, t.FontDefault.Size))
		p.Text(i.prefix, &xc.RECT{Left: x, Top: p.Rc.Top, Right: x + w, Bottom: p.Rc.Bottom},
			base, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		x += w + inputTextGap
	}

	// ---- 尾部: 自右向左排(- + / 眼睛 / 清除 / 计数 / 后缀)----
	// 起点用 trailingInset: 按钮贴边 4, 文字类(后缀/计数)按 10, 与 syncPadding 同源。
	rx := p.Rc.Right - i.trailingInset()
	btn := func(part inputPart, icon string, dim bool) {
		rc := xc.RECT{
			Left:   rx - inputBtnSz,
			Top:    cy - inputBtnSz/2,
			Right:  rx,
			Bottom: cy + inputBtnSz/2,
		}
		enabled := !st.Disabled
		hot := enabled && !dim && i.hover == part
		if enabled {
			// 命中区**永远**登记(含到边界的 +/−): 否则这一下会落到底下的原生 Edit,
			// 连点时第二下变成"双击选中单词"(实测: 值到 100 再连点 + 会把 100 选成蓝底)。
			// 到边界时只是变灰、不响应动作(activate 里的 stepBy 会夹住), 但点击仍被吞掉。
			i.parts = append(i.parts, inputPartRect{part: part, rc: rc, dim: dim})
		}
		hv := xui.HoverItem(i.Handle, int32(part), hot)
		if hv > 0 {
			// 幽灵按钮的悬停底: bg-accent, 圆角比壳体小一档。**按下期间保持不变** ——
			// shadcn/tailwind 的按钮没有 active 变换, 按下 = 悬停态延续;
			// 若按下时撤掉底色, 松开又回来, 视觉上就是一次闪烁(实测)。
			p.FillRound(&rc, t.Accent.ScaleAlpha(hv), inputBtnR)
		}
		glyph := xc.RECT{
			Left:   rc.Left + (inputBtnSz-inputGlyphSz)/2,
			Top:    rc.Top + (inputBtnSz-inputGlyphSz)/2,
			Right:  0,
			Bottom: 0,
		}
		glyph.Right = glyph.Left + inputGlyphSz
		glyph.Bottom = glyph.Top + inputGlyphSz
		c := xui.Fade(base, t.AccentForeground, hv)
		if dim {
			c = base.ScaleAlpha(0.4) // 到值域边界: 按钮不可用(视觉上变淡, 同 shadcn 的禁用钮)
		}
		p.Icon(icon, &glyph, c, 2)
		rx = rc.Left - inputGap
	}

	if i.step != nil {
		// 自动隐藏形态: 只有"鼠标在输入框上"或"有焦点"时才画, 否则只占位
		// (占位让文字/其它附加件保持原位, 该区域点下去落到文本区 = 放光标)。
		if !i.stepVisible(st) {
			if i.stepArrows {
				rx -= inputStepColW + inputGap
			} else {
				rx -= inputBtnSz*2 + inputGap
			}
		} else if i.stepArrows {
			rx = i.paintStepArrows(p, t, st, rx, cy)
		} else {
			cur := i.Number()
			btn(partStepUp, icons.IconPlus, i.atMax(cur))
			btn(partStepDown, icons.IconMinus, i.atMin(cur))
		}
	}
	if i.passToggle {
		icon := icons.IconEye
		if !i.password {
			icon = icons.IconEyeOff // 已是明文 → 点它回到密文
		}
		btn(partToggle, icon, false)
	}
	if i.clearBtn {
		if i.IsEmpty() {
			rx -= inputBtnSz + inputGap // 空内容留位不画, 输入过程中文字区不跳
		} else {
			btn(partClear, icons.IconX, false)
		}
	}
	if i.counterMax > 0 {
		n := len([]rune(i.GetText()))
		s := fmt.Sprintf("%d/%d", n, i.counterMax)
		c := base
		if n > i.counterMax && !st.Disabled {
			c = t.Destructive
		}
		w := inputTextWidth(t, s, inputCounterSz)
		p.Draw.SetFont(inputFont(t, inputCounterSz))
		p.Text(s, &xc.RECT{Left: rx - w, Top: p.Rc.Top, Right: rx, Bottom: p.Rc.Bottom},
			c, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		rx -= w + inputGap
	}
	if i.suffix != "" {
		w := inputTextWidth(t, i.suffix, t.FontDefault.Size)
		p.Draw.SetFont(inputFont(t, t.FontDefault.Size))
		p.Text(i.suffix, &xc.RECT{Left: rx - w, Top: p.Rc.Top, Right: rx, Bottom: p.Rc.Bottom},
			base, xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter)
		rx -= w + inputGap
	}
}

// stepVisible 返回步进按钮当前是否显示(见 WithStepperAutoHide)。
func (i *Input) stepVisible(st xui.State) bool {
	if i.step == nil {
		return false
	}
	if !i.stepAuto {
		return true
	}
	return i.hoverInput || st.Focused
}

// paintStepArrows 画"上下三角"形态的步进按钮, 返回尾部下一个元素该从哪结束。
// 上半 = +(值加), 下半 = −(值减), 与原生 number 输入框的方向一致。
func (i *Input) paintStepArrows(p *xui.Painter, t *xui.Theme, st xui.State, rx, cy int32) int32 {
	colR := rx
	colL := colR - inputStepColW
	top := cy - inputBtnSz/2
	half := inputBtnSz / 2

	base := t.MutedForeground
	if st.Disabled {
		base = base.ScaleAlpha(0.5)
	}

	// 每枚箭头一个**正方形**绘制块(图标管线按短边铺满, 24 网格里三角占满 24 宽、
	// 高 19.2 → 9px 方块画出来正好 9×7.2), 上下各留 inputArrowPad 的白边 ——
	// 原生 number 输入框的箭头组上下也是有留白的, 顶满会显得箭头很大(用户对照)。
	cx := (colL + colR) / 2
	upBox := xc.RECT{
		Left:   cx - inputArrowBox/2,
		Top:    top + inputArrowPad,
		Right:  cx + inputArrowBox/2,
		Bottom: top + inputArrowPad + inputArrowBox,
	}
	dnBox := xc.RECT{
		Left:   cx - inputArrowBox/2,
		Top:    top + inputBtnSz - inputArrowPad - inputArrowBox,
		Right:  cx + inputArrowBox/2,
		Bottom: top + inputBtnSz - inputArrowPad,
	}

	cur := i.Number()
	halfBtn := func(part inputPart, rc xc.RECT, box xc.RECT, svg string, dim bool) {
		enabled := !st.Disabled
		hot := enabled && !dim && i.hover == part
		if enabled {
			// 命中区无条件登记(理由同 btn: 到边界时也不能让点击穿透到原生 Edit)
			i.parts = append(i.parts, inputPartRect{part: part, rc: rc, dim: dim})
		}
		hv := xui.HoverItem(i.Handle, int32(part), hot)
		pressedNow := i.pressed == part && enabled
		// 悬停**没有底色**(原生微调控件也是) —— 反馈只有箭头变色; 按下再叠加"缩小"
		c := xui.Fade(base, t.AccentForeground, hv)
		if pressedNow {
			// 按下: 颜色回到常态, 只靠"缩小"表达按压
			c = base
		}
		if dim {
			c = base.ScaleAlpha(0.4)
		}
		p.Icon(svg, &box, c, 0, pressScale(i.pressOf(part, pressedNow)))
	}
	halfBtn(partStepUp, xc.RECT{Left: colL, Top: top, Right: colR, Bottom: top + half}, upBox, arrowUpSVG, i.atMax(cur))
	halfBtn(partStepDown, xc.RECT{Left: colL, Top: top + half, Right: colR, Bottom: top + inputBtnSz}, dnBox, arrowDownSVG, i.atMin(cur))
	return colL - inputGap
}

// ===== 交互 =====

// pressOf 取内嵌按钮的按压过渡值(0..1) —— 按下时图标缩小的动画走 hover 同一套
// 过渡内核(150ms), 键从 8 起避开 hover 占用的 1..4。
func (i *Input) pressOf(part inputPart, active bool) float64 {
	return xui.HoverItem(i.Handle, int32(part)+8, active)
}

// pressScale 由按压过渡值算图标缩放: 松开 1.0, 按到底 0.8(原生微调控件的按压幅度)。
func pressScale(pv float64) float32 { return float32(1 - 0.2*pv) }

// setPressed 记录/清除按下的内嵌件并驱动缩放动画(过渡内核会负责期间的重绘)。
func (i *Input) setPressed(part inputPart) {
	if i.pressed == part {
		return
	}
	old := i.pressed
	i.pressed = part
	if old != partNone {
		i.pressOf(old, false)
	}
	if part != partNone {
		i.pressOf(part, true)
	}
	i.Redraw(true)
}

// onLButtonDown 把落在内嵌按钮上的按下吞掉(光标不会挪到按钮底下), 动作放在抬起时执行。
// 同时挂在 LBUTTONDBLCLK 上: 连点的第二次按下是**双击消息**, 不走 LButtonDown,
// 只绑 Down 的话原生 Edit 会拿双击去"选中单词"(现象: 双击 + 把数字选成蓝底)。
//
// 为什么动作不在按下时做: 按住不放会连续触发, 且原生 Edit 会先接管这次按下;
// 抬起消息每次都有, 放在这里每点都生效。
func (i *Input) onLButtonDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pr, ok := i.partAt(pPt.X, pPt.Y); ok && i.IsEnable() {
		// dim(到值域边界)的按钮: 点击照吞(防双击穿透到原生 Edit), 但**不进按压反馈**
		// —— 禁用态的图标不该跟着缩放/变色(实测: 值为 0 时按 ↓, 灰箭头也做了缩小动画)。
		if !pr.dim {
			i.setPressed(pr.part)
		}
		*pbHandled = true
		return 0
	}
	i.setPressed(partNone)
	xui.NotePointerPress()
	return 0
}

// onLButtonUp 在抬起处执行内嵌按钮的动作(见 onLButtonDown 的说明), 并结束按压反馈。
func (i *Input) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if pr, ok := i.partAt(pPt.X, pPt.Y); ok && i.IsEnable() {
		*pbHandled = true
		if !pr.dim {
			i.activate(pr.part)
		}
	}
	i.setPressed(partNone)
	return 0
}

// activate 执行内嵌按钮的动作。
// 每个动作都把焦点交回输入框(浏览器里清除/眼睛按钮点完也还留在输入框里继续打字),
// 随后 MoveEnd —— SetFocus 会顺手全选, 不全选会连带覆盖掉后续输入。
func (i *Input) activate(part inputPart) {
	switch part {
	case partClear:
		i.SetText("")
		i.SetFocus()
	case partToggle:
		i.setMasked(!i.password)
		i.SetFocus()
		i.MoveEnd() // 切换可见性后光标归位, 否则会留在旧的位置
	case partStepUp:
		i.SetFocus()
		i.stepBy(1)
	case partStepDown:
		i.SetFocus()
		i.stepBy(-1)
	}
	i.Redraw(false)
}

// setMasked 切换明文/密文(密码框专用, 未启用 WithPasswordToggle 时无效)。
//
// 切换之后必须让原生 Edit 重新排版一次: `EnablePassword` 只换遮挡开关, 字符步进
// 还留在上一个模式里 —— 实测切到明文后字距被撑成密码字符的全角宽度("s e c r e t - 1 2 3"),
// 切回密文时圆点同样被撑开; 换主题能恢复正常(因为主题回调重排了)。
// 重设内边距(改 text rect)就能触发重排, 而且**不会**触发 OnChange —— 重新 SetText
// 虽然也能修好, 但会给用户一个"值变了"的假通知。
func (i *Input) setMasked(masked bool) {
	if !i.passToggle || i.password == masked {
		return
	}
	i.password = masked
	i.EnablePassword(masked)
	if masked {
		xc.XEdit_SetPasswordCharacter(i.Handle, '●') // 重开密码模式会重置成 '*'
	}
	i.syncPadding()
	i.Redraw(false)
}

// SetPasswordVisible 设置密码框显示明文还是密文(等价于点右侧的眼睛按钮)。
// 未启用 WithPasswordToggle 的输入框调用无效。
func (i *Input) SetPasswordVisible(visible bool) *Input {
	i.setMasked(!visible)
	return i
}

// PasswordVisible 返回密码框当前是否显示明文。
func (i *Input) PasswordVisible() bool { return i.passToggle && !i.password }

// onKeyDown 让步进器接管 ↑/↓(按一个步长)与回车(提交校正)。
func (i *Input) onKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	if i.step == nil || !i.IsEnable() {
		return 0
	}
	switch int(wParam) {
	case 0x26: // VK_UP
		i.stepBy(1)
		*pbHandled = true
	case 0x28: // VK_DOWN
		i.stepBy(-1)
		*pbHandled = true
	case 0x0D: // VK_RETURN
		i.commitNumber()
	}
	return 0
}

// stepBy 按 dir(+1/-1)走一个步长。
func (i *Input) stepBy(dir int) {
	if i.step == nil || !i.IsEnable() {
		return
	}
	base, ok := i.parseNumber()
	if !ok {
		base = i.lastNum // 空内容/非法文本: 退到最近一次有效值再走一步
	}
	i.applyNumber(base + float64(dir)*i.step.step)
	if i.onStep != nil {
		i.onStep(i.lastNum)
	}
}

// commitNumber 解析当前文本并规范化(夹取 + 吸附 + 补足小数位), 解析失败回退。
func (i *Input) commitNumber() {
	if i.step == nil {
		return
	}
	if strings.TrimSpace(i.GetText()) == "" {
		return // 允许空(留给占位符), 不硬塞最小值
	}
	if v, ok := i.parseNumber(); ok {
		i.applyNumber(v)
		return
	}
	i.applyNumber(i.lastNum)
}

// applyNumber 把数值规范化后写回文本。
func (i *Input) applyNumber(v float64) {
	if i.step == nil {
		return
	}
	v = i.step.snap(v)
	i.lastNum = v
	i.SetText(i.step.format(v))
	i.MoveEnd() // SetText 会把光标放回开头, 连续点 +/− 时要留在末尾
	i.Redraw(false)
}

// parseNumber 解析当前文本为数值。
func (i *Input) parseNumber() (float64, bool) {
	s := strings.TrimSpace(i.GetText())
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func (i *Input) atMin(v float64) bool { return i.step != nil && v <= i.step.min+1e-9 }
func (i *Input) atMax(v float64) bool { return i.step != nil && v >= i.step.max-1e-9 }

// Number 返回当前数值(步进器专用)。内容为空或无法解析时返回最近一次有效值。
func (i *Input) Number() float64 {
	if v, ok := i.parseNumber(); ok {
		return v
	}
	return i.lastNum
}

// SetNumber 设置数值: 夹到值域、吸附到步长, 并按步长的小数位格式化后写入。
// 非步进器(未传 WithStepper)时无效果。
func (i *Input) SetNumber(v float64) *Input {
	i.applyNumber(v)
	return i
}

// OnStep 注册步进回调(点 −/+ 或按 ↑/↓ 后触发, v 为规范化后的值)。
// 文本变化本身仍走 OnChange。
func (i *Input) OnStep(fn func(v float64)) *Input {
	i.onStep = fn
	return i
}

// SetInvalid 设置/清除校验失败态(见 WithInputInvalid)。
func (i *Input) SetInvalid(v bool) *Input {
	if i.invalid == v {
		return i
	}
	i.invalid = v
	i.Redraw(false)
	return i
}

// Invalid 返回当前是否处于校验失败态。
func (i *Input) Invalid() bool { return i.invalid }

// Value 返回输入内容。
func (i *Input) Value() string { return i.GetText() }

// SetValue 设置输入内容。
func (i *Input) SetValue(s string) *Input {
	i.SetText(s)
	return i
}

// OnChange 注册内容变化回调。
func (i *Input) OnChange(fn func(text string)) *Input {
	i.AddEvent_Edit_Changed(func(hEle int, pbHandled *bool) int {
		fn(i.GetText())
		return 0
	})
	return i
}

func (i *Input) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	t := xui.Current()
	st := xui.StateOf(hEle)
	i.syncTextColor() // 禁用态在 paint 里发现时补一次(Enable(false) 不会走主题回调)

	bg := t.Background
	border := t.Input
	if st.Disabled {
		// shadcn: disabled:bg-input/50(暗色另有 dark:disabled:bg-input/80) + 组级 opacity-50;
		// 用 ScaleAlpha 逐层乘, 不要用 WithAlpha 覆盖(暗色 input 自带 15% alpha, 覆盖成 50%
		// 会画出一块发亮的灰饼)
		a := 0.5
		if t.IsDark() {
			a = 0.8
		}
		bg = t.Input.ScaleAlpha(a).ScaleAlpha(0.5)
		border = t.Input.ScaleAlpha(0.5)
	}

	// aria-invalid 与 focus-visible 共用"环带 + 内芯"画法, 红色优先(见 button.go 的说明)。
	const inset = int32(3)
	invalid := i.invalid && !st.Disabled
	switch {
	case invalid:
		halo, borderColor := t.Destructive.WithAlpha(51), t.Destructive
		if t.IsDark() {
			halo, borderColor = t.Destructive.WithAlpha(102), t.Destructive.WithAlpha(128)
		}
		p.RingFrame(halo, bg, t.RadiusLg())
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRound(&inner, borderColor, t.RadiusLg()-inset, 1)
		}
	case st.Focused && !st.Disabled:
		// focus-visible:border-ring + ring-3 ring-ring/50: FocusFrame 画 3px 半透明
		// ring 带(填充式, 会盖掉原来的 border-input), 实色的 border-ring 边框
		// 再描在 ring 带内缘 —— 与浏览器一致: 聚焦后边框从 input 色换成 ring 色。
		p.FocusFrame(t.Ring, bg, t.RadiusLg())
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRound(&inner, t.Ring, t.RadiusLg()-inset, 1)
		}
	default:
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}

	// 壳体内部的附加件(图标/前后缀/计数/按钮)。原生 Edit 的文字在本回调之后绘制,
	// 附加件都排在文字区之外(见 syncPadding), 不会互相压字。
	i.paintAddons(p, t, st)
	// 画完附加件把画笔字体还原成主题字体: 原生文字是**接着这个建图对象**画的, 附加件
	// 里为计数换过 12px 字体, 不还原会把正文也带成小字号 + 字距走样(实测: 带计数的
	// 输入框里"字体跟别的不一样、字距还变宽")。
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	return 0
}
