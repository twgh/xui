package ui

import (
	"strings"
	"unicode/utf16"
	"unsafe"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// otpSlotSz 是单格边长(size-8 = 32); otpSepW 是组分隔符宽度(Minus 图标 size-4)。
const (
	otpSlotSz int32 = 32
	otpSepW   int32 = 16
)

// InputOTPPattern 单格字符过滤器: 返回 false 的字符不入格(同 input-otp 的
// pattern 正则, 如 /^\d+$/)。nil = 不限。
type InputOTPPattern func(r rune) bool

// InputOTPPatternDigit 只收数字(shadcn 文档的 regulate 示例)。
func InputOTPPatternDigit(r rune) bool { return r >= '0' && r <= '9' }

// InputOTPPatternAlphanumeric 只收字母与数字。
func InputOTPPatternAlphanumeric(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// InputOTPPatternAny 收任意可打印字符。
func InputOTPPatternAny(r rune) bool { return r >= 32 }

// InputOTP 一次性验证码输入框, 对应 shadcn input-otp.tsx(底层 input-otp 包):
//   - 载体: 每格一个原生 Edit(32×32, 文字水平居中), 光标/输入法/焦点由原生承载,
//     壳体与边框自绘; 上游的"假光标"动画这里不需要 —— 真格子里就是真光标;
//   - 键入: 打进已填的格是**替换**(同 input-otp)并前进一格; 非法字符不入格;
//   - 键盘: ←/→/Home/End 移动, Backspace 清空当前格、空格时退格清前一格,
//     Ctrl+V 从当前格起粘贴(非法字符自动跳过), Tab 按原生焦点顺序走格;
//   - 全部填满触发 OnComplete; Value() = 各格字符拼接。
//
// 上游的"跨格拖动选择"没有对应物(每格是独立 Edit), 不做。
type InputOTP struct {
	*widget.LayoutEle
	length    int32
	pattern   InputOTPPattern
	seps      []int32 // 在第 N 格后画分隔符(N 为 1 基格序号)
	initial   string  // WithInputOTPValue 记下的初始值(格子建好后落位)
	slots     []*widget.Edit
	invalid   bool
	disabled  bool
	changes   []func(value string)
	completes []func(value string)
}

// InputOTPOption 验证码输入框配置项。
type InputOTPOption func(*InputOTP)

// WithInputOTPLength 设置格数(默认 6)。
func WithInputOTPLength(n int32) InputOTPOption {
	return func(o *InputOTP) {
		if n > 0 {
			o.length = n
		}
	}
}

// WithInputOTPPattern 设置单格字符过滤器。
func WithInputOTPPattern(p InputOTPPattern) InputOTPOption {
	return func(o *InputOTP) { o.pattern = p }
}

// WithInputOTPSeparatorAfter 在第 n 格后画一枚 Minus 分隔符(同 InputOTPSeparator;
// n 为 1 基, 如 3 = 前 3 格一组)。可以多次调用画多组分隔符。
func WithInputOTPSeparatorAfter(n int32) InputOTPOption {
	return func(o *InputOTP) { o.seps = append(o.seps, n) }
}

// WithInputOTPValue 设置初始值(超出格数/非法字符被忽略)。
func WithInputOTPValue(v string) InputOTPOption {
	// 选项在格子创建**之前**执行, 这里只能先记下, 格子建好后统一落位(见 NewInputOTP)
	return func(o *InputOTP) { o.initial = v }
}

// WithInputOTPInvalid 设置校验失败态(aria-invalid: 边框变红 + 组外圈红环)。
func WithInputOTPInvalid() InputOTPOption {
	return func(o *InputOTP) { o.invalid = true }
}

// WithInputOTPDisabled 禁用(整组 opacity-50, 同上游 has-disabled)。
func WithInputOTPDisabled() InputOTPOption {
	return func(o *InputOTP) { o.disabled = true }
}

// NewInputOTP 创建验证码输入框。宽高传 0 按内容自撑(格数×32 + 分隔符×16)。
func NewInputOTP(x, y, cx, cy int32, hParent int, opts ...InputOTPOption) *InputOTP {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	o := &InputOTP{LayoutEle: ele, length: 6}
	for _, fn := range opts {
		fn(o)
	}
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableAutoWrap(false)
	ele.EnableBkTransparent(true)
	ele.SetSpace(0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	// 擦底(透明元素重绘不擦底): 父元素先画, 不会盖住格子; invalid 的组外圈
	// (shadcn 在 InputOTPGroup 上 has-aria-invalid:ring-3)必须画在 PAINT_END ——
	// 格子会铺满整个容器并把普通 paint 阶段画的环盖掉, PAINT_END 在子元素之后。
	ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		p.FillRound(&p.Rc, xui.Current().Card, 0)
		return 0
	})
	ele.AddEvent_Paint_End(func(hEle int, hDraw int, pbHandled *bool) int {
		if !o.invalid || o.disabled {
			return 0
		}
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		halo := t.Destructive.WithAlpha(51) // ring-destructive/20
		if t.IsDark() {
			halo = t.Destructive.WithAlpha(102) // dark:ring-destructive/40
		}
		rc := p.Rc
		rc.Left, rc.Top, rc.Right, rc.Bottom = rc.Left+1, rc.Top+1, rc.Right-1, rc.Bottom-1
		p.StrokeRoundCorners(&rc, halo, xui.UniformCorners(t.RadiusLg()), 3)
		return 0
	})

	for i := int32(0); i < o.length; i++ {
		e := widget.NewEdit(0, 0, otpSlotSz, otpSlotSz, ele.Handle)
		if e == nil {
			continue
		}
		e.EnableMultiLine(false)
		e.EnableBkTransparent(true)
		e.EnableDrawBorder(false).EnableDrawFocus(false)
		e.EnableAutoSelAll(false)
		e.SetBorderSize(0, 0, 0, 0)
		// 对齐是位掩码: 只给水平位会把垂直位清成"顶部", 水平垂直一起置
		e.SetTextAlign(xcc.Edit_TextAlign_Flag_Center | xcc.Edit_TextAlign_Flag_Center_V)
		i := i
		e.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
			o.paintSlot(i, hEle, hDraw, pbHandled)
			return 0
		})
		// 字符一律拦截后自己落格: 原生 Edit 没有 maxlength, 放行会越填越长;
		// 拦下来才有"替换已填格 + 非法字符不入格"的 input-otp 语义。
		e.AddEvent_Char(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
			o.onChar(i, wParam, pbHandled)
			return 0
		})
		e.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
			o.onKeyDown(i, wParam, pbHandled)
			return 0
		})
		e.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int { e.Redraw(false); return 0 })
		e.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int { e.Redraw(false); return 0 })
		o.slots = append(o.slots, e)
		if containsInt32(o.seps, i+1) && i+1 < o.length {
			sep := widget.NewElement(0, 0, otpSepW, otpSlotSz, ele.Handle)
			if sep != nil {
				sep.EnableBkTransparent(true)
				sep.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
					p := xui.BeginPaint(hEle, hDraw)
					if p == nil {
						return 0
					}
					*pbHandled = true
					th := xui.Current()
					p.FillRound(&p.Rc, th.Card, 0) // 擦底(透明元素重绘不擦底)
					fore := th.Foreground
					if o.disabled {
						fore = fore.ScaleAlpha(0.5) // 组级 disabled:opacity-50 也管分隔符
					}
					rc := p.Rc // 分隔元素本身就是 16 宽, Minus 图标铺满即可
					p.Icon(icons.IconMinus, &rc, fore, 2)
					return 0
				})
			}
		}
	}
	// WithInputOTPValue 在选项阶段只能先记值, 格子建好后在这里落位
	if o.initial != "" {
		o.distribute(o.initial, 0)
	}
	o.applyTheme(xui.Current())
	xui.OnThemeChangeFor(ele.Handle, func(*xui.Theme) { o.applyTheme(xui.Current()) })
	if o.disabled {
		o.SetDisabled(true)
	}
	xui.AttachWindow(ele.Handle)
	return o
}

func containsInt32(list []int32, v int32) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ===== 状态 =====

// Value 返回各格字符拼接(空格保留为空)。
func (o *InputOTP) Value() string {
	var b strings.Builder
	for _, e := range o.slots {
		b.WriteString(e.GetText())
	}
	return b.String()
}

// SetValue 设置值(超出格数截断, 非法字符跳过), 不触发回调。
func (o *InputOTP) SetValue(v string) *InputOTP {
	o.distribute(v, 0)
	o.Redraw(false)
	return o
}

// Clear 清空所有格。
func (o *InputOTP) Clear() *InputOTP { return o.SetValue("") }

// Focus 把焦点放回第一个空格(全满则最后一格)。禁用态不动作(同 disabled input)。
func (o *InputOTP) Focus() *InputOTP {
	if o.disabled {
		return o
	}
	for i, e := range o.slots {
		if e.GetText() == "" {
			o.focusSlot(int32(i))
			return o
		}
	}
	o.focusSlot(int32(len(o.slots)) - 1)
	return o
}

// SetInvalid 设置/清除校验失败态。
func (o *InputOTP) SetInvalid(v bool) *InputOTP {
	if o.invalid == v {
		return o
	}
	o.invalid = v
	o.Redraw(false)
	return o
}

// Invalid 返回当前是否处于校验失败态。
func (o *InputOTP) Invalid() bool { return o.invalid }

// SetDisabled 禁用/启用整组。
func (o *InputOTP) SetDisabled(v bool) *InputOTP {
	// 不做"状态没变就返回"的短路: 构造时 WithInputOTPDisabled 已经让 o.disabled=true,
	// 构造末尾那次 SetDisabled(true) 会被短路掉, 格子根本没被禁用(实测: 禁用组照样
	// 能点进去、能打字)。状态落地的动作必须无条件跑。
	o.disabled = v
	for _, e := range o.slots {
		e.Enable(!v)
		// Enable(false) 只挡键盘/输入, 鼠标点上去 xcgui 照样会给它焦点并显示光标
		// (原生 Win32 的 disabled EDIT 完全不可聚焦)。焦点开关必须单独关掉,
		// 否则禁用组里会出现"能点进去、光标在闪"的假禁用态。
		e.EnableFocus(!v)
		e.Redraw(false)
	}
	o.applyTheme(xui.Current())
	o.Redraw(false)
	return o
}

// Disabled 返回是否禁用。
func (o *InputOTP) Disabled() bool { return o.disabled }

// OnChange 注册值变化回调(键入/退格/粘贴触发; SetValue 不触发)。
func (o *InputOTP) OnChange(fn func(value string)) *InputOTP {
	o.changes = append(o.changes, fn)
	return o
}

// OnComplete 注册"全部填满"回调。
func (o *InputOTP) OnComplete(fn func(value string)) *InputOTP {
	o.completes = append(o.completes, fn)
	return o
}

// ===== 内部 =====

// setSlotText 落格文字(所有写文字的路径都走这里)。
//
// 居中由原生 `XEdit_SetTextAlign(Center|Center_V)` 负责(单行模式下有效, 程序 SetText
// 同样生效)。**不要**为了"顶位置"去设左右 border: 文本区被压到字宽以下时字形右半截
// 会被 Edit 自己裁掉(见 DESIGN 第八节 110)。唯一的例外是空格的 caret, 见 syncInset。
func (o *InputOTP) setSlotText(i int32, s string) {
	if i < 0 || i >= int32(len(o.slots)) {
		return
	}
	o.slots[i].SetText(s)
	o.syncInset(i)
	o.slots[i].Redraw(false)
}

// syncInset 只处理空格子的原生 caret 位置: 空文本时 xcgui 的 caret 贴在文本区左缘
// (实测 x≈3), 而 input-otp 的空格 caret 在格子中央。对称内缩半格把 caret 顶到中心
// (实测 14/14 → x≈17), 对称内缩不改变"居中"的语义; 一旦格子里有字就清零, 否则文本区
// 会窄于字形、把字裁掉。
func (o *InputOTP) syncInset(i int32) {
	if i < 0 || i >= int32(len(o.slots)) {
		return
	}
	e := o.slots[i]
	if e.GetText() == "" {
		k := (otpSlotSz - 1) / 2
		e.SetBorderSize(k, 0, k, 0)
		return
	}
	e.SetBorderSize(0, 0, 0, 0)
}

// distribute 从 from 号格开始把字符串逐字落格(过滤非法字符), 其余格清空。
func (o *InputOTP) distribute(v string, from int32) {
	chars := o.filterRunes([]rune(v))
	i := from
	for _, ch := range chars {
		if i >= int32(len(o.slots)) {
			break
		}
		o.setSlotText(i, string(ch))
		i++
	}
	for ; i < int32(len(o.slots)); i++ {
		o.setSlotText(i, "")
	}
}

// filterRunes 按 pattern 过滤出可入格字符。
func (o *InputOTP) filterRunes(rs []rune) []rune {
	out := make([]rune, 0, len(rs))
	for _, r := range rs {
		if r < 32 {
			continue
		}
		if o.pattern != nil && !o.pattern(r) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// onChar 键入: 替换当前格并前进(同 input-otp)。控制字符放行给原生(Ctrl+C 之类)。
func (o *InputOTP) onChar(i int32, wParam uintptr, pbHandled *bool) {
	ch := rune(wParam)
	if ch < 32 {
		return
	}
	*pbHandled = true
	if o.disabled || i < 0 || i >= int32(len(o.slots)) {
		return
	}
	if o.pattern != nil && !o.pattern(ch) {
		return
	}
	o.setSlotText(i, string(ch))
	o.notify()
	if i+1 < int32(len(o.slots)) {
		o.focusSlot(i + 1)
	}
}

// onKeyDown 导航/退格/粘贴。
func (o *InputOTP) onKeyDown(i int32, wParam uintptr, pbHandled *bool) {
	const (
		vkLeft, vkRight = 0x25, 0x27
		vkHome, vkEnd   = 0x24, 0x23
		vkBack, vkDel   = 0x08, 0x2E
		vkV             = 0x56
		vkControl       = 0x11
	)
	n := int32(len(o.slots))
	switch wParam {
	case vkLeft:
		if i > 0 {
			o.focusSlot(i - 1)
		}
		*pbHandled = true
	case vkRight:
		if i < n-1 {
			o.focusSlot(i + 1)
		}
		*pbHandled = true
	case vkHome:
		o.focusSlot(0)
		*pbHandled = true
	case vkEnd:
		o.focusSlot(n - 1)
		*pbHandled = true
	case vkBack:
		*pbHandled = true
		if o.disabled || i < 0 {
			return
		}
		if o.slots[i].GetText() != "" {
			o.setSlotText(i, "")
			o.notify()
		} else if i > 0 { // 空格退格: 清掉前一格并移过去(input-otp 语义)
			o.focusSlot(i - 1)
			o.setSlotText(i-1, "")
			o.notify()
		}
	case vkDel:
		*pbHandled = true
		if o.disabled || i < 0 {
			return
		}
		o.setSlotText(i, "")
		o.notify()
	case vkV:
		// 原生粘贴只落一格(没有 maxlength 也会整段塞进来), 自己分发
		if uint16(wapi.GetAsyncKeyState(vkControl))&0x8000 != 0 {
			*pbHandled = true
			o.paste(i)
		}
	}
}

// paste 从 from 号格起粘贴剪贴板内容(非法字符跳过), 焦点停在最后写入的格。
func (o *InputOTP) paste(from int32) {
	if o.disabled || from < 0 {
		return
	}
	chars := o.filterRunes([]rune(otpClipboardText()))
	if len(chars) == 0 {
		return
	}
	i := from
	for _, ch := range chars {
		if i >= int32(len(o.slots)) {
			break
		}
		o.setSlotText(i, string(ch))
		i++
	}
	o.focusSlot(i - 1)
	o.notify()
}

func (o *InputOTP) focusSlot(i int32) {
	if i < 0 || i >= int32(len(o.slots)) {
		return
	}
	o.slots[i].SetFocus()
}

func (o *InputOTP) notify() {
	v := o.Value()
	for _, fn := range o.changes {
		fn(v)
	}
	if utf8len(v) == int(o.length) {
		for _, fn := range o.completes {
			fn(v)
		}
	}
}

// utf8len 按字符数计长(Value 由单字符格拼接, rune 数 == 格数)。
func utf8len(s string) int { return len([]rune(s)) }

// applyTheme 同步原生侧的字体/文字色/光标色(组件侧绘制在 paint 里实时取令牌)。
func (o *InputOTP) applyTheme(t *xui.Theme) {
	fore := t.Foreground
	if o.disabled {
		fore = fore.Mix(t.Card, 0.5) // disabled:opacity-50 的文字预混
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		for _, e := range o.slots {
			e.SetFont(f)
		}
	}
	// 禁用态把原生 cursor 染成格底: EnableFocus(false) 已经挡住点击聚焦, 这一层是
	// 兜底 —— 万一有焦点漏进来(比如禁用瞬间焦点正在该格), 也看不到闪烁的光标。
	caret := t.Primary
	if o.disabled {
		caret = t.Card
	}
	for i := range o.slots {
		e := o.slots[i]
		e.SetTextColor(uint32(fore))
		e.SetCaretColor(uint32(caret))
	}
}

// paintSlot 绘制单格壳体。相邻格共享一条边: 每格画 top/bottom/right,
// 首格额外画 left(border-l), 其余格的 left 由前一格的 right 充当 ——
// 逐边手画而不是 StrokeRoundCorners 整框, 是为了不给中段格子叠出 2px 边。
func (o *InputOTP) paintSlot(i int32, hEle int, hDraw int, pbHandled *bool) {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return
	}
	*pbHandled = true
	t := xui.Current()
	st := xui.StateOf(hEle)
	n := int32(len(o.slots))

	// 先算组的圆角: 底色/环带内芯都要跟着走, 否则方角会盖在圆角环带/组边框上,
	// 看起来像"环带左圆角里嵌了个白方块"。
	groupR := t.RadiusLg() // 组圆角 rounded-lg
	// shadcn 的 InputOTP 是若干 InputOTPGroup(各自 rounded-lg + 组内 first:rounded-l-lg /
	// last:rounded-r-lg): 分隔符把格子切成组, **每组**的首格圆左侧两角、末格圆右侧两角。
	// 只按 i==0 / i==n-1 判会让第二组的首格既没左边框也不圆角(实测)。
	groupStart := i == 0 || containsInt32(o.seps, i)
	groupEnd := i == n-1 || containsInt32(o.seps, i+1)
	corners := xui.Corners{}
	if groupStart {
		corners.TopLeft, corners.BottomLeft = groupR, groupR
	}
	if groupEnd {
		corners.TopRight, corners.BottomRight = groupR, groupR
	}

	// 擦底(透明元素重绘不擦底) + 暗色的 bg-input/30
	bgDark := t.IsDark()
	fillBg := func(rc xc.RECT, c xui.Corners) {
		p.FillRoundCorners(&rc, t.Card, c)
		if bgDark {
			p.FillRoundCorners(&rc, t.Input.ScaleAlpha(0.3), c)
		}
	}
	fillBg(p.Rc, corners)

	border := t.Input
	if o.invalid && !o.disabled {
		border = t.Destructive
		if t.IsDark() {
			border = t.Destructive.WithAlpha(128) // dark:aria-invalid:border-destructive/50
		}
	}
	if o.disabled {
		border = border.ScaleAlpha(0.5)
	}

	active := st.Focused && !o.disabled
	if active {
		// data-[active=true]: ring-3 ring-ring/50(整格环带, 环带跟格子圆角走),
		// aria-invalid 时红环优先
		halo := t.Ring.WithAlpha(128)
		bc := t.Ring
		if o.invalid {
			halo, bc = t.Destructive.WithAlpha(51), t.Destructive
			if t.IsDark() {
				halo, bc = t.Destructive.WithAlpha(102), t.Destructive
			}
		}
		p.FillRoundCorners(&p.Rc, halo, corners)
		const inset = int32(3)
		inner := xc.RECT{
			Left:   p.Rc.Left + inset,
			Top:    p.Rc.Top + inset,
			Right:  p.Rc.Right - inset,
			Bottom: p.Rc.Bottom - inset,
		}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			// 内芯必须**圆角**铺底: 方角会从环带的圆角里露出白方块(实测截图)。
			fillBg(inner, corners.Inset(inset))
			p.StrokeRoundCorners(&inner, bc, corners.Inset(inset), 1)
		}
		return
	}

	// 非激活: 逐边画 1px。中段格的 left 边画完用底色擦掉(它属于前一格的 right 边),
	// 但**只擦中段**: 连上下两行一起擦会在分隔处留下 1px 缺口(顶/底横线断开, 实测)。
	p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	if !groupStart {
		fillBg(xc.RECT{Left: p.Rc.Left, Top: p.Rc.Top + 1, Right: p.Rc.Left + 1, Bottom: p.Rc.Bottom - 1}, xui.Corners{})
	}
}

// otpClipboardText 读剪贴板 Unicode 文本(粘贴分发用); 取不到返回空串。
func otpClipboardText() string {
	if !wapi.IsClipboardFormatAvailable(wapi.CF_UNICODETEXT) {
		return ""
	}
	if !wapi.OpenClipboard(0) { // 0 = 关联当前任务
		return ""
	}
	defer wapi.CloseClipboard()
	h := wapi.GetClipboardData(wapi.CF_UNICODETEXT)
	if h == 0 {
		return ""
	}
	ptr := wapi.GlobalLock(h)
	if ptr == 0 {
		return ""
	}
	defer wapi.GlobalUnlock(h)
	// GlobalLock 返回的是 uintptr, 直接转 unsafe.Pointer / reflect.SliceHeader 都会被
	// vet 判"可能的误用"; 用同布局的自定义头把指针当 UTF-16 缓冲区读(到 NUL 为止)。
	type sliceHeader struct {
		Data uintptr
		Len  int
		Cap  int
	}
	const otpClipMax = 8192
	sh := sliceHeader{Data: ptr, Len: otpClipMax, Cap: otpClipMax}
	mem := *(*[]uint16)(unsafe.Pointer(&sh))
	for i, c := range mem {
		if c == 0 {
			mem = mem[:i]
			break
		}
	}
	return string(utf16.Decode(mem))
}
