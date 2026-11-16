package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 面包屑几何常量(逻辑像素), 对齐 shadcn breadcrumb.tsx:
//   - List: text-sm(14) / 行高 20, 条目间距 sm:gap-2.5(10, 桌面档) + flex-wrap;
//   - Item: inline-flex gap-1.5(6);
//   - Separator: chevron size-3.5(14);
//   - Ellipsis: size-9(36), 图标 size-4(16)。
const (
	breadcrumbGap           int32 = 10
	breadcrumbItemGap       int32 = 6
	breadcrumbTextH         int32 = 20
	breadcrumbTextSize      int32 = 14
	breadcrumbIconSize      int32 = 14
	breadcrumbEllipsisSize  int32 = 36
	breadcrumbEllipsisGlyph int32 = 16
)

func breadcrumbFont(t *xui.Theme) int {
	spec := t.FontDefault
	spec.Size = breadcrumbTextSize
	return xui.FontHandle(spec)
}

func breadcrumbTextWidth(t *xui.Theme, text string) int32 {
	w := xui.TextWidth(text, breadcrumbFont(t))
	if w < 1 {
		w = 1
	}
	return w
}

// Breadcrumb 面包屑列表: 横向流式 + 自动换行, 宽铺满、高按内容自撑。
//
// 与 shadcn breadcrumb.tsx 对齐: 文字 text-sm muted-foreground, 条目间距 10,
// 当前页用前景色。根节点就是上游的 BreadcrumbList(nav 本身没有样式);
//
//	bc := ui.NewBreadcrumb(0, 0, 0, 0, parent)
//	xui.ItemFill(bc)
//	it := ui.NewBreadcrumbItem(bc.Handle)
//	ui.NewBreadcrumbLink(it.Handle, "首页").OnClick(func() { ... })
//	ui.NewBreadcrumbSeparator(bc.Handle)
//	it = ui.NewBreadcrumbItem(bc.Handle)
//	ui.NewBreadcrumbPage(it.Handle, "组件")
type Breadcrumb struct {
	*widget.LayoutEle
}

// NewBreadcrumb 创建面包屑列表。宽铺满(放进布局时可由布局项覆盖), 高自撑。
func NewBreadcrumb(x, y, cx, cy int32, hParent int) *Breadcrumb {
	l := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(true)
	l.SetSpace(breadcrumbGap)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return &Breadcrumb{LayoutEle: l}
}

// List 返回列表容器本身(别名, 便于语义化链式调用)。
func (b *Breadcrumb) List() *widget.LayoutEle {
	if b == nil {
		return nil
	}
	return b.LayoutEle
}

// NewBreadcrumbItem 创建一个条目容器: 横向、间距 6、宽高自撑, 用来包住链接/当前页。
func NewBreadcrumbItem(hParent int) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if l == nil {
		return nil
	}
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(breadcrumbItemGap)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// BreadcrumbLink 可点击的面包屑链接: muted 文字, 悬停过渡到前景色。
//
// 用透明 Element 自绘文本(不用 Label): Label 的原生 Button 载体带内部文本内边距,
// 按文字量宽会被裁剪成省略号。自绘宽度就是文本宽度, 悬停走运行时 150ms 过渡。
type BreadcrumbLink struct {
	*breadcrumbText
}

// breadcrumbText 是链接/当前页共用的自绘文本元素。
type breadcrumbText struct {
	*widget.Element
	text    string
	link    bool
	hover   bool
	onClick func()
}

func newBreadcrumbText(hParent int, text string, link bool) *breadcrumbText {
	b := &breadcrumbText{text: text, link: link}
	w := breadcrumbTextWidth(xui.Current(), text)
	h := breadcrumbTextH
	ele := widget.NewElement(0, 0, w, h, hParent)
	if ele == nil {
		return nil
	}
	b.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	ele.AddEvent_Paint(b.onPaint)
	xui.AttachWindow(ele.Handle)
	xui.OnThemeChangeFor(ele.Handle, func(t *xui.Theme) {
		b.syncWidth(t)
		b.Redraw(false)
	})
	if link {
		ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
			if !b.hover {
				b.hover = true
				b.Redraw(true)
			}
			return 0
		})
		ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			if b.hover {
				b.hover = false
				b.Redraw(true)
			}
			return 0
		})
		// 必须有 Down 处理器, 原生 Element 才会把同一次按下的 Up 派发过来
		// (只注册 MouseStay/Leave + Up 时点击不触发)。
		ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if b.onClick != nil {
				b.onClick()
			}
			return 0
		})
	}
	return b
}

func (b *breadcrumbText) syncWidth(t *xui.Theme) {
	if b == nil || b.Element == nil {
		return
	}
	w := breadcrumbTextWidth(t, b.text)
	b.SetWidth(w)
	b.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
}

func (b *breadcrumbText) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	color := t.Foreground
	if b.link {
		color = xui.Fade(t.MutedForeground, t.Foreground, xui.Hover(hEle, b.hover))
	}
	if f := breadcrumbFont(t); f != 0 {
		p.Draw.SetFont(f)
	}
	p.Text(b.text, &p.Rc, color,
		xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
	return 0
}

// NewBreadcrumbLink 创建链接。宽度按文字量测, 换主题后自动重新量宽。
func NewBreadcrumbLink(hParent int, text string) *BreadcrumbLink {
	b := newBreadcrumbText(hParent, text, true)
	if b == nil {
		return nil
	}
	return &BreadcrumbLink{breadcrumbText: b}
}

// OnClick 设置点击回调。
func (l *BreadcrumbLink) OnClick(fn func()) *BreadcrumbLink {
	if l == nil {
		return nil
	}
	l.onClick = fn
	return l
}

// BreadcrumbPage 是当前页节点: 前景色、常规字重、不可点击。
type BreadcrumbPage struct {
	*breadcrumbText
}

// NewBreadcrumbPage 创建当前页。
func NewBreadcrumbPage(hParent int, text string) *BreadcrumbPage {
	b := newBreadcrumbText(hParent, text, false)
	if b == nil {
		return nil
	}
	return &BreadcrumbPage{breadcrumbText: b}
}

// BreadcrumbSeparatorOption 分隔符配置项。
type BreadcrumbSeparatorOption func(*BreadcrumbSeparator)

// WithBreadcrumbSeparatorIcon 换成自定义图标(默认 lucide chevron-right)。
func WithBreadcrumbSeparatorIcon(icon string) BreadcrumbSeparatorOption {
	return func(s *BreadcrumbSeparator) { s.icon = icon }
}

// BreadcrumbSeparator 面包屑分隔符: 默认 14×14 的 muted 图标; 文字分隔符按文字量宽。
type BreadcrumbSeparator struct {
	*widget.Element
	icon string
	text string
}

// NewBreadcrumbSeparator 创建分隔符(默认 chevron-right)。
func NewBreadcrumbSeparator(hParent int, opts ...BreadcrumbSeparatorOption) *BreadcrumbSeparator {
	s := &BreadcrumbSeparator{icon: icons.IconChevronRight}
	for _, o := range opts {
		o(s)
	}
	ele := widget.NewElement(0, 0, breadcrumbIconSize, breadcrumbIconSize, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, breadcrumbIconSize)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, breadcrumbIconSize)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

// NewBreadcrumbSeparatorText 用文字做分隔符(如 "/" 或 ">"), 对应上游给
// BreadcrumbSeparator 传任意 children(官方示例是 DotIcon); 文字 muted、单行。
func NewBreadcrumbSeparatorText(hParent int, text string) *BreadcrumbSeparator {
	s := &BreadcrumbSeparator{text: text}
	w := breadcrumbTextWidth(xui.Current(), text)
	h := breadcrumbTextH
	ele := widget.NewElement(0, 0, w, h, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	xui.OnThemeChangeFor(ele.Handle, func(t *xui.Theme) {
		w := breadcrumbTextWidth(t, text)
		s.SetWidth(w)
		s.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
		s.Redraw(false)
	})
	return s
}

func (s *BreadcrumbSeparator) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	if s.text != "" {
		if f := breadcrumbFont(t); f != 0 {
			p.Draw.SetFont(f)
		}
		p.Text(s.text, &p.Rc, t.MutedForeground,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		return 0
	}
	p.Icon(s.icon, &p.Rc, t.MutedForeground, 2)
	return 0
}

// BreadcrumbEllipsisOption 省略号配置项。
type BreadcrumbEllipsisOption func(*BreadcrumbEllipsis)

// WithBreadcrumbEllipsisOnClick 把省略号变成可点击的收起项(对应官方 Dropdown 示例里
// 包在 DropdownMenuTrigger 里的用法): 进入 Tab 链、悬停出底色、Enter/空格/点击触发;
// 不传时仍是纯装饰(不进焦点、不响应鼠标)。
func WithBreadcrumbEllipsisOnClick(fn func()) BreadcrumbEllipsisOption {
	return func(e *BreadcrumbEllipsis) { e.onClick = fn }
}

// BreadcrumbEllipsis 折叠省略号: 36×36 区域居中画 more-horizontal(16)。
type BreadcrumbEllipsis struct {
	*widget.Element
	onClick func()
	hover   bool
}

// NewBreadcrumbEllipsis 创建省略号。
func NewBreadcrumbEllipsis(hParent int, opts ...BreadcrumbEllipsisOption) *BreadcrumbEllipsis {
	ele := widget.NewElement(0, 0, breadcrumbEllipsisSize, breadcrumbEllipsisSize, hParent)
	if ele == nil {
		return nil
	}
	e := &BreadcrumbEllipsis{Element: ele}
	for _, o := range opts {
		o(e)
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, breadcrumbEllipsisSize)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, breadcrumbEllipsisSize)
	ele.AddEvent_Paint(e.onPaint)
	if e.onClick != nil {
		ele.EnableFocus(true)
		ele.EnableSwitchFocus(true)
		ele.EnableDrawFocus(false)
		ele.SetCursor(linkCursor())
		ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
			e.hover = true
			ele.Redraw(false)
			return 0
		})
		ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			e.hover = false
			ele.Redraw(false)
			return 0
		})
		// 原生 Element 要先接住按下, 同一次操作的抬起才会派发到它。
		ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			e.fire()
			return 0
		})
		ele.AddEvent_KeyDown(func(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
			if wParam == breadcrumbVKEnter || wParam == breadcrumbVKSpace {
				e.fire()
				*pbHandled = true
			}
			return 0
		})
		ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
			ele.Redraw(false)
			return 0
		})
		ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
			ele.Redraw(false)
			return 0
		})
	}
	xui.AttachWindow(ele.Handle)
	return e
}

const (
	breadcrumbVKEnter uintptr = 0x0D
	breadcrumbVKSpace uintptr = 0x20
)

func (e *BreadcrumbEllipsis) fire() {
	if e != nil && e.onClick != nil {
		e.onClick()
	}
}

func (e *BreadcrumbEllipsis) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	if e.onClick != nil {
		radius := t.RadiusMd()
		st := xui.StateOf(hEle)
		hv := xui.Hover(hEle, e.hover && !st.Disabled)
		if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
			hole := t.Card
			if hv > 0 {
				hole = t.Card.Mix(t.Accent, hv)
			}
			p.FocusFrame(t.Ring, hole, radius)
		} else if hv > 0 {
			p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv), radius)
		}
	}
	inset := (breadcrumbEllipsisSize - breadcrumbEllipsisGlyph) / 2
	rc := p.Rc
	rc.Left += inset
	rc.Top += inset
	rc.Right -= inset
	rc.Bottom -= inset
	p.Icon(icons.IconMoreHorizontal, &rc, t.MutedForeground, 2)
	return 0
}
