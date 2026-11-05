package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Tag Input 几何(逻辑像素): 外壳与 Input 同为 h-8(32), 内部是 h-5(20) 的
// Badge 胶囊与等高的一行裸输入。胶囊排满一行就换行, 外壳高度随行数自撑。
const (
	tagInputMinH     int32 = 32
	tagInputPadX     int32 = 8
	tagInputPadY     int32 = 5
	tagInputRowH     int32 = 22
	tagInputRowGap   int32 = 4
	tagInputChipH    int32 = 20
	tagInputChipPadL int32 = 8
	tagInputChipPadR int32 = 6
	tagInputChipGap  int32 = 4
	tagInputXGap     int32 = 3
	tagInputXSz      int32 = 12
	tagInputXHit     int32 = 16 // × 的命中方块(比图标大一圈, 免得难点)
	tagInputMinEditW int32 = 48 // 输入区不足这个宽就整体换行, 否则光标贴着右边缘
)

// TagInputOption 组件配置项。
type TagInputOption func(*TagInput)

// WithTagInputPlaceholder 设置占位文本(无标签且输入为空时由原生 Edit 绘制)。
func WithTagInputPlaceholder(s string) TagInputOption {
	return func(t *TagInput) { t.placeholder = s }
}

// WithTagInputTags 设置初始标签(会按去重 / 上限 / 校验器清洗)。
func WithTagInputTags(tags ...string) TagInputOption {
	return func(t *TagInput) { t.initial = append([]string(nil), tags...) }
}

// WithTagInputMax 限制标签数量(0 = 不限)。达到上限后提交被忽略, 输入文字原样保留。
func WithTagInputMax(n int) TagInputOption {
	return func(t *TagInput) {
		if n > 0 {
			t.max = n
		}
	}
}

// WithTagInputAllowDuplicates 允许重复标签(默认精确匹配去重, 重复值静默忽略)。
func WithTagInputAllowDuplicates() TagInputOption {
	return func(t *TagInput) { t.allowDup = true }
}

// WithTagInputSeparators 设置提交分隔符(默认 ",，\n"; 传空串关闭分隔符提交)。
// 输入 / 粘贴时遇到任一分隔符, 前面那段立即成为标签, 最后一段留在输入框。
func WithTagInputSeparators(s string) TagInputOption {
	return func(t *TagInput) { t.separators = s }
}

// WithTagInputValidator 设置标签校验器(返回 false 的段被拒绝并留在输入框里)。
func WithTagInputValidator(fn func(string) bool) TagInputOption {
	return func(t *TagInput) { t.validator = fn }
}

// WithTagInputDisabled 禁用(不能输入、不能删、不能聚焦; 整组降到 50%)。
func WithTagInputDisabled() TagInputOption {
	return func(t *TagInput) { t.disabled = true }
}

// WithTagInputInvalid 设置校验失败态(aria-invalid: 红边框 + 红色环)。
func WithTagInputInvalid() TagInputOption {
	return func(t *TagInput) { t.invalid = true }
}

// tagInputChip 是一个胶囊在元素客户区里的矩形(逻辑像素)。close 是 × 的命中方块,
// 绘制时按 tagInputXSz 居中缩小 —— 命中区比图标大, 连点删除才不会漏。
type tagInputChip struct {
	chip  xc.RECT
	close xc.RECT
}

// TagInput 可就地删除的 chips 输入框。
//
// 上游 shadcn 没有对应的 tsx, 交互语义由本库自定(权威汇总见 DESIGN 第四节末):
//   - 回车提交当前输入; 输入中遇到分隔符(默认逗号/中文逗号/换行)立即提交前一段;
//   - 输入框为空时退格删掉最后一个标签, 非空时按原生行为删字符;
//   - 点胶囊上的 × 删除该标签; 点其它位置把焦点交回输入框;
//   - 空白串忽略, 默认精确匹配去重, 上限 / 校验器拒绝时输入文字保留。
//
// 载体是"自绘外壳 + 一个裸 Edit 子控件": chips 全部画在外壳的 Paint 里,
// 输入框只是排在最后一个胶囊后面的一块透明原生 Edit, 由 relayout 用 SetRectEx
// 定位。这样换行、× 命中、焦点环都由本组件自己算, 不用给每个标签建一个子控件
// (copy-not-install 下也没法动态增删布局项而不踩 Auto 宽塌成 0 的坑)。
type TagInput struct {
	*widget.Element
	edit *widget.Edit

	tags        []string
	initial     []string
	placeholder string
	separators  string
	max         int
	allowDup    bool
	disabled    bool
	invalid     bool
	validator   func(string) bool
	changeFns   []func([]string)

	chips        []tagInputChip // 每次 measure 重算, 供绘制与命中复用
	inputRc      xc.RECT        // 裸 Edit 的位置
	measuredW    int32          // 上次 measure 的宽
	neededH      int32          // 按当前标签数算出的目标高度
	lastEditRc   xc.RECT
	editApplied  bool
	relayouting  bool
	fitPending   bool // 量高已排进下一轮 UI 消息, 合并同轮触发
	suppress     bool // 程序化 SetText 期间屏蔽 Edit_Changed
	hoverClose   int  // 当前悬停的 × 索引, -1 = 无
	pressClose   int  // 按下的 × 索引, -1 = 无
}

// NewTagInput 创建标签输入框。cy 传 0 使用默认高度 32(会随标签行数自撑);
// cx 传 0 时通常配 xui.ItemFill 铺满父级。
func NewTagInput(x, y, cx, cy int32, hParent int, opts ...TagInputOption) *TagInput {
	if cy <= 0 {
		cy = tagInputMinH
	}
	outer := widget.NewElement(x, y, cx, cy, hParent)
	if outer == nil {
		return nil
	}
	t := &TagInput{
		Element:    outer,
		separators: ",，\n",
		hoverClose: -1,
		pressClose: -1,
	}
	for _, o := range opts {
		o(t)
	}
	outer.EnableBkTransparent(true)
	outer.EnableDrawBorder(false)
	outer.EnableFocus(false)
	outer.EnableSwitchFocus(false)
	// LayoutEle 默认鼠标穿透, 普通 Element 也要显式关掉才稳(见 DESIGN 185)。
	outer.EnableMouseThrough(false)
	outer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, cy)

	for _, s := range t.initial {
		t.add(s)
	}
	t.initial = nil

	// 裸输入: 只负责文字/光标/IME, 边框与焦点环都由外壳画。位置由 relayout 决定,
	// 初值给一个最小可见矩形, 免得 0 宽的 Edit 在某些缩放比下不被创建。
	e := widget.NewEdit(0, 0, tagInputMinEditW, tagInputRowH, outer.Handle)
	if e == nil {
		return nil
	}
	t.edit = e
	e.EnableMultiLine(false)
	e.EnableBkTransparent(true)
	e.EnableDrawBorder(false).EnableDrawFocus(false)
	e.EnableAutoSelAll(false)
	e.EnableAutoShowScrollBar(false)
	if h := e.GetScrollBarHObj(); h != nil {
		h.Show(false)
	}
	if v := e.GetScrollBarVObj(); v != nil {
		v.Show(false)
	}
	e.SetBorderSize(0, 0, 0, 0)

	outer.AddEvent_Paint(t.onPaint)
	outer.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		t.relayout()
		return 0
	})
	outer.AddEvent_LButtonDown(t.onLButtonDown)
	outer.AddEvent_LButtonDBClick(t.onLButtonDBClick)
	outer.AddEvent_LButtonUp(t.onLButtonUp)
	e.AddEvent_KeyDown(t.onKeyDown)
	e.AddEvent_Edit_Changed(t.onEditChanged)
	e.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int { t.Redraw(false); return 0 })
	e.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int { t.Redraw(false); return 0 })

	t.applyTheme(xui.Current())
	xui.OnThemeChangeFor(outer.Handle, func(th *xui.Theme) { t.applyTheme(th) })
	xui.AttachWindow(outer.Handle)
	t.trackHover()
	if t.disabled {
		t.SetDisabled(true)
	}
	t.relayout()
	return t
}

// ===== 状态 =====

// Tags 返回当前标签(副本)。
func (t *TagInput) Tags() []string {
	if t == nil {
		return nil
	}
	return append([]string(nil), t.tags...)
}

// Count 返回标签数量。
func (t *TagInput) Count() int {
	if t == nil {
		return 0
	}
	return len(t.tags)
}

// SetTags 替换标签列表(不触发 OnChange)。会按去重 / 上限 / 校验器清洗。
func (t *TagInput) SetTags(tags []string) *TagInput {
	if t == nil {
		return t
	}
	t.tags = nil
	for _, s := range tags {
		t.add(s)
	}
	t.relayout()
	return t
}

// AddTag 追加一个标签, 返回是否真的加入。不触发 OnChange。
func (t *TagInput) AddTag(text string) *TagInput {
	if t != nil {
		if added, _ := t.add(text); added {
			t.relayout()
		}
	}
	return t
}

// RemoveTag 删除第 index 个标签, 返回是否删掉。不触发 OnChange。
func (t *TagInput) RemoveTag(index int) *TagInput {
	if t != nil && t.remove(index) {
		t.relayout()
	}
	return t
}

// Clear 清空所有标签与输入框内容。不触发 OnChange。
func (t *TagInput) Clear() *TagInput {
	if t == nil {
		return t
	}
	t.tags = nil
	t.setEditText("")
	t.relayout()
	return t
}

// Value 返回输入框里尚未提交的文字。
func (t *TagInput) Value() string {
	if t == nil || t.edit == nil {
		return ""
	}
	return t.edit.GetText()
}

// SetValue 设置输入框里尚未提交的文字(不切成标签, 不触发 OnChange)。
func (t *TagInput) SetValue(s string) *TagInput {
	if t != nil {
		t.setEditText(s)
		t.Redraw(false)
	}
	return t
}

// SetMax 修改标签数量上限(0 = 不限)。已有标签不受影响。
func (t *TagInput) SetMax(n int) *TagInput {
	if t != nil {
		t.max = n
	}
	return t
}

// Focus 把焦点放到输入框。禁用态不动作。
func (t *TagInput) Focus() *TagInput {
	if t != nil && !t.disabled && t.edit != nil {
		t.edit.SetFocus()
	}
	return t
}

// SetDisabled 禁用/启用整组。
func (t *TagInput) SetDisabled(v bool) *TagInput {
	if t == nil {
		return t
	}
	t.disabled = v
	if t.edit != nil {
		t.edit.Enable(!v)
		// Enable(false) 只挡输入, 点上去 xcgui 仍会给它焦点并显示光标(见 InputOTP 同款坑)。
		t.edit.EnableFocus(!v)
		t.edit.EnableSwitchFocus(!v)
	}
	if v {
		t.hoverClose, t.pressClose = -1, -1
	}
	t.applyTheme(xui.Current())
	t.Redraw(false)
	return t
}

// Disabled 返回是否禁用。
func (t *TagInput) Disabled() bool { return t != nil && t.disabled }

// SetInvalid 设置/清除校验失败态。
func (t *TagInput) SetInvalid(v bool) *TagInput {
	if t != nil && t.invalid != v {
		t.invalid = v
		t.Redraw(false)
	}
	return t
}

// Invalid 返回是否处于校验失败态。
func (t *TagInput) Invalid() bool { return t != nil && t.invalid }

// OnChange 注册标签变化回调。只由交互(提交 / 退格 / ×)触发; SetTags / AddTag /
// RemoveTag / Clear / SetValue 这些程序化改值都不广播, 避免回调里再改值递归。
func (t *TagInput) OnChange(fn func([]string)) *TagInput {
	if t != nil {
		t.changeFns = append(t.changeFns, fn)
	}
	return t
}

// ===== 标签增删 =====

// add 规范化后尝试加入。空串 / 重复值算"无内容"(静默清空输入框), 上限与校验器
// 拒绝算 rejected(输入文字保留, 让用户看见自己打的东西没被吞)。
func (t *TagInput) add(raw string) (added, rejected bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return false, false
	}
	if t.max > 0 && len(t.tags) >= t.max {
		return false, true
	}
	if !t.allowDup {
		for _, v := range t.tags {
			if v == s {
				return false, false
			}
		}
	}
	if t.validator != nil && !t.validator(s) {
		return false, true
	}
	t.tags = append(t.tags, s)
	return true, false
}

func (t *TagInput) remove(i int) bool {
	if i < 0 || i >= len(t.tags) {
		return false
	}
	t.tags = append(t.tags[:i], t.tags[i+1:]...)
	return true
}

// drainSeparators 把 s 中**已遇到分隔符**的段切出来, 最后一段原样返回。
// 只用最后一个未终止段留在输入框: 粘贴 "a,b,c" 会一次加 a、b, c 留着继续编辑。
func (t *TagInput) drainSeparators(s string) ([]string, string) {
	if t.separators == "" {
		return nil, s
	}
	var parts []string
	for {
		i := strings.IndexAny(s, t.separators)
		if i < 0 {
			break
		}
		parts = append(parts, s[:i])
		// 分隔符可能是多字节(中文逗号), 不能按 1 个字节挪。
		_, size := utf8.DecodeRuneInString(s[i:])
		s = s[i+size:]
	}
	return parts, s
}

// setEditText 替换输入内容并屏蔽随之而来的原生变更事件。
// 程序化 SetText 在 xcgui 里本来就不发 Edit_Changed(见 DESIGN 8.185), 但组合期
// 与不同版本的表现不一致, 显式屏蔽最稳; MoveEnd 是因为 SetText 会把光标放回开头。
func (t *TagInput) setEditText(s string) {
	if t.edit == nil {
		return
	}
	t.suppress = true
	t.edit.SetText(s)
	t.edit.MoveEnd()
	t.suppress = false
}

// commit 回车提交: 先按分隔符切, 再把剩下那段一起提交。
func (t *TagInput) commit() {
	parts, rest := t.drainSeparators(t.edit.GetText())
	parts = append(parts, rest)
	changed, rejected := false, false
	for _, p := range parts {
		a, r := t.add(p)
		changed = changed || a
		rejected = rejected || r
	}
	if !rejected {
		t.setEditText("")
	}
	t.afterChange(changed)
}

// onEditChanged 输入中遇到分隔符立即提交前一段。
func (t *TagInput) onEditChanged(hEle int, pbHandled *bool) int {
	if t.suppress || t.disabled {
		return 0
	}
	text := t.edit.GetText()
	parts, rest := t.drainSeparators(text)
	if len(parts) == 0 {
		return 0
	}
	// 已经到上限: 不再切分, 否则用户打一个逗号就会"吞掉"一段文字。
	if t.max > 0 && len(t.tags) >= t.max {
		return 0
	}
	changed := false
	for _, p := range parts {
		if a, _ := t.add(p); a {
			changed = true
		}
	}
	t.setEditText(rest)
	t.afterChange(changed)
	return 0
}

func (t *TagInput) afterChange(changed bool) {
	t.relayout()
	if !changed {
		return
	}
	if len(t.changeFns) == 0 {
		return
	}
	tags := t.Tags()
	for _, fn := range t.changeFns {
		fn(tags)
	}
}

// ===== 键盘与鼠标 =====

func (t *TagInput) onKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	if t.disabled {
		return 0
	}
	switch int(wParam) {
	case 0x0D: // VK_RETURN
		t.commit()
		*pbHandled = true // 不放给原生, 否则单行 Edit 会响铃 / 吞掉这次回车
	case 0x08: // VK_BACK
		if t.edit.GetText() == "" && len(t.tags) > 0 {
			t.remove(len(t.tags) - 1)
			t.afterChange(true)
			*pbHandled = true
		}
	}
	return 0
}

// closeAt 返回落在哪个 × 上(-1 = 没命中)。
func (t *TagInput) closeAt(x, y int32) int {
	for i := range t.chips {
		r := t.chips[i].close
		if x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom {
			return i
		}
	}
	return -1
}

func (t *TagInput) onLButtonDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if t.disabled {
		return 0
	}
	if i := t.closeAt(pPt.X, pPt.Y); i >= 0 {
		t.pressClose = i
		*pbHandled = true
		return 0
	}
	t.pressClose = -1
	xui.NotePointerPress()
	t.edit.SetFocus()
	return 0
}

// 连点 × 时第二次按下是双击消息, 不走 LButtonDown; 这里照常命中并删一个,
// 否则快速连点会漏掉一半点击(同 Input 内嵌按钮的坑)。
func (t *TagInput) onLButtonDBClick(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if t.disabled {
		return 0
	}
	*pbHandled = true
	if i := t.closeAt(pPt.X, pPt.Y); i >= 0 {
		t.remove(i)
		t.afterChange(true)
	}
	return 0
}

func (t *TagInput) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	i := t.pressClose
	t.pressClose = -1
	if i < 0 || t.disabled {
		return 0
	}
	if t.closeAt(pPt.X, pPt.Y) != i {
		return 0
	}
	*pbHandled = true
	t.remove(i)
	t.afterChange(true)
	// 删除后焦点留在输入框, 接着打字 / 连按退格都不用再点一下。
	t.edit.SetFocus()
	return 0
}

// trackHover 用**元素级** MouseMove 跟踪悬停的是哪枚 ×。
//
// 普通 Element(显式关掉鼠标穿透后)的 MouseMove 每次移动都会来, pPt 就是元素客户区
// 坐标, 命中矩形可以和点击直接共用; 不用窗口级 MouseMove + GetStayEle 那套换算,
// 也不用判祖先可见。原生 Edit 只在进入时上报一次 MouseMove, 别照它的窗口级方案。
func (t *TagInput) trackHover() {
	t.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		idx := -1
		if !t.disabled {
			idx = t.closeAt(pPt.X, pPt.Y)
		}
		if idx != t.hoverClose {
			t.hoverClose = idx
			t.Redraw(true)
		}
		return 0
	})
	t.AddEvent_MouseLeave(func(hEle int, hEleStay int, pbHandled *bool) int {
		if t.hoverClose != -1 {
			t.hoverClose = -1
			t.Redraw(true)
		}
		return 0
	})
}

// ===== 排版与绘制 =====

func tagInputChipFont(th *xui.Theme) int {
	return xui.FontHandle(xui.FontSpec{Family: th.FontDefault.Family, Size: th.FontDefault.Size - 2})
}

func tagInset(rc xc.RECT, n int32) xc.RECT {
	return xc.RECT{Left: rc.Left + n, Top: rc.Top + n, Right: rc.Right - n, Bottom: rc.Bottom - n}
}

func tagCentered(rc xc.RECT, sz int32) xc.RECT {
	cx, cy := (rc.Left+rc.Right)/2, (rc.Top+rc.Bottom)/2
	return xc.RECT{Left: cx - sz/2, Top: cy - sz/2, Right: cx - sz/2 + sz, Bottom: cy - sz/2 + sz}
}

// measure 按当前宽度算出每个胶囊的矩形、输入框矩形与所需高度。纯计算, 不动元素尺寸。
func (t *TagInput) measure(w int32) {
	t.measuredW = w
	t.chips = t.chips[:0]
	th := xui.Current()
	font := tagInputChipFont(th)
	left := tagInputPadX
	right := w - tagInputPadX
	if right < left+tagInputMinEditW {
		right = left + tagInputMinEditW
	}
	x, y := left, tagInputPadY
	for _, s := range t.tags {
		cw := tagInputChipPadL + xui.TextWidth(s, font) + tagInputXGap + tagInputXSz + tagInputChipPadR
		if cw > right-left {
			cw = right - left // 超长标签占满一行, 文字用 NoWrap 截在 × 左边
		}
		if x > left && x+cw > right {
			x = left
			y += tagInputRowH + tagInputRowGap
		}
		cr := xc.RECT{
			Left:   x,
			Top:    y + (tagInputRowH-tagInputChipH)/2,
			Right:  x + cw,
			Bottom: y + (tagInputRowH-tagInputChipH)/2 + tagInputChipH,
		}
		xr := xc.RECT{
			Left:   cr.Right - tagInputChipPadR - tagInputXSz - (tagInputXHit-tagInputXSz)/2,
			Top:    cr.Top + (tagInputChipH-tagInputXHit)/2,
			Right:  cr.Right - tagInputChipPadR + (tagInputXHit-tagInputXSz)/2,
			Bottom: cr.Top + (tagInputChipH-tagInputXHit)/2 + tagInputXHit,
		}
		t.chips = append(t.chips, tagInputChip{chip: cr, close: xr})
		x += cw + tagInputChipGap
	}
	if x > left && right-x < tagInputMinEditW {
		x = left
		y += tagInputRowH + tagInputRowGap
	}
	t.inputRc = xc.RECT{Left: x, Top: y, Right: right, Bottom: y + tagInputRowH}
	t.neededH = y + tagInputRowH + tagInputPadY
	if t.neededH < tagInputMinH {
		t.neededH = tagInputMinH
	}
}

// syncEditRect 把裸 Edit 摆到最后一个胶囊后面。父级是普通 Element(没开布局),
// 子控件用 SetRectEx 定坐标不会被布局趟覆盖; 矩形没变就不重复设。
func (t *TagInput) syncEditRect() {
	if t.edit == nil {
		return
	}
	rc := t.inputRc
	if rc.Right <= rc.Left {
		rc.Right = rc.Left + tagInputMinEditW
	}
	if t.editApplied && rc == t.lastEditRc {
		return
	}
	t.lastEditRc, t.editApplied = rc, true
	t.edit.SetRectEx(rc.Left, rc.Top, rc.Right-rc.Left, rc.Bottom-rc.Top,
		false, xcc.AdjustLayout_No)
}

// relayout 只做立即要生效的部分: 重算胶囊矩形、把裸 Edit 摆到新位置。
// 元素自身高度必须走 scheduleFit 推迟到下一轮 UI 消息 —— 在输入/尺寸事件里同步
// SetHeight 会重入布局趟, 实测换行后下面的兄弟项不会跟着下移(压住下一张卡)。
func (t *TagInput) relayout() {
	if t.relayouting || t.Element == nil {
		return
	}
	t.relayouting = true
	defer func() { t.relayouting = false }()
	w := t.GetWidth()
	if w <= 0 {
		return
	}
	t.measure(w)
	t.syncEditRect()
	t.scheduleFit()
	t.Redraw(false)
}

// scheduleFit 合并同一轮里的多次量高请求(同 Textarea / Alert 的做法)。
func (t *TagInput) scheduleFit() {
	if t.fitPending {
		return
	}
	t.fitPending = true
	app.AfterFunc(0, func() {
		t.fitPending = false
		t.fit()
	})
}

func (t *TagInput) fit() {
	if t.Element == nil || !xc.XC_IsHXCGUI(t.Handle, xcc.XC_ELE) {
		return
	}
	h := t.neededH
	if h <= 0 || h == t.GetHeight() {
		return
	}
	t.relayouting = true
	t.SetHeight(h)
	t.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	t.relayouting = false
	tagInputReflow(t.Handle)
	t.Redraw(false)
}

// tagInputReflow 只重排最近的 ScrollArea 框架(或窗口), 不在父链上逐级 AdjustLayout:
// 中间层的 Weight 布局会在量算趟里把卡片/内容区临时排成 0 宽, 兄弟位置也不更新
// (同 textareaReflow 的实测结论)。
func tagInputReflow(hEle int) {
	for p := xc.XWidget_GetParentEle(hEle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(p)
			xc.XEle_Redraw(p, false)
			return
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			xc.XEle_AdjustLayout(pp)
			xc.XEle_Redraw(pp, false)
			return
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
			return
		}
	}
}

func (t *TagInput) applyTheme(th *xui.Theme) {
	if t.edit == nil {
		return
	}
	// 编辑框不继承窗口字体, 不显式下发就会比周围小一圈(同 Input)。
	if f := xui.FontHandle(th.FontDefault); f != 0 {
		t.edit.SetFont(f)
	}
	c := th.Foreground
	if t.disabled {
		// 原生文本不吃 alpha, 禁用态只能把字色与卡片底色预混。
		c = th.Foreground.Mix(th.Card, 0.5)
	}
	t.edit.SetTextColor(uint32(c))
	t.edit.SetDefaultText(t.placeholder)
	t.edit.SetDefaultTextColor(uint32(th.MutedForeground))
	t.edit.SetCaretColor(uint32(th.Primary))
	t.Redraw(false)
}

func (t *TagInput) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	th := xui.Current()
	if p.W() != t.measuredW {
		// 首次绘制可能早于 XE_SIZE(元素建好但还没被布局趟摆过), 这里补量一次。
		// 只重算矩形 + 摆 Edit, 不在 Paint 里改自身高度 —— 那会触发同步 Size 递归重绘。
		t.measure(p.W())
		t.syncEditRect()
	}

	// 透明元素重绘不擦底: 先铺一层实色卡片底, 否则换行/删标签会留下残影。
	p.FillRound(&p.Rc, th.Card, 0)

	corners := xui.UniformCorners(th.RadiusLg())
	bg, border := th.Background, th.Input
	if t.disabled {
		a := 0.5
		if th.IsDark() {
			a = 0.8
		}
		bg = th.Input.ScaleAlpha(a).ScaleAlpha(0.5)
		border = th.Input.ScaleAlpha(0.5)
	}
	const inset = int32(3)
	focused := !t.disabled && xui.StateOf(t.edit.Handle).Focused
	switch {
	case t.invalid && !t.disabled:
		halo, borderColor := th.Destructive.WithAlpha(51), th.Destructive
		if th.IsDark() {
			halo, borderColor = th.Destructive.WithAlpha(102), th.Destructive.WithAlpha(128)
		}
		p.RingFrameCorners(halo, bg, corners)
		inner := tagInset(p.Rc, inset)
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRoundCorners(&inner, borderColor, corners.Inset(inset), 1)
		}
	case focused:
		p.FocusFrameCorners(th.Ring, bg, corners)
		inner := tagInset(p.Rc, inset)
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.StrokeRoundCorners(&inner, th.Ring, corners.Inset(inset), 1)
		}
	default:
		p.FillRoundCorners(&p.Rc, bg, corners)
		p.StrokeRoundCorners(&p.Rc, border, corners, 1)
	}

	// 胶囊。hover 过渡走运行时 HoverItem + Fade, 组件自己不起定时器也不缓存进度。
	if f := tagInputChipFont(th); f != 0 {
		p.Draw.SetFont(f)
	}
	fore := th.SecondaryForeground
	xBase := th.MutedForeground
	if t.disabled {
		fore = fore.ScaleAlpha(0.5)
		xBase = xBase.ScaleAlpha(0.5)
	}
	for i, c := range t.chips {
		hv := xui.HoverItem(t.Handle, int32(i), !t.disabled && t.hoverClose == i)
		p.FillRound(&c.chip, th.Secondary, tagInputChipH/2)
		tr := xc.RECT{
			Left:   c.chip.Left + tagInputChipPadL,
			Top:    c.chip.Top,
			Right:  c.close.Left - 2,
			Bottom: c.chip.Bottom,
		}
		if tr.Right > tr.Left && i < len(t.tags) {
			p.Text(t.tags[i], &tr, fore,
				xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		}
		if hv > 0 {
			p.FillRound(&c.close, th.Accent.ScaleAlpha(hv), 4)
		}
		glyph := tagCentered(c.close, tagInputXSz)
		p.Icon(icons.IconX, &glyph, xui.Fade(xBase, th.AccentForeground, hv), 2)
	}
	// 画笔字体是接着这个 hDraw 用的, 不还原会把原生 Edit 的正文也带成 12px。
	if f := xui.FontHandle(th.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	return 0
}
