package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/wapi/wnd"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ComboBox 自绘下拉选择框: 载体为原生 ComboBox(弹出、选择、键盘/滚动行为全部原生),
// 外壳、文字与右侧箭头自绘; 下拉列表是原生弹出窗口 + 原生 ListBox, 无法逐像素接管,
// 因此只在弹出时做主题上色(见 stylePopup)。
//
// 与 shadcn select.tsx(radix-nova) 对齐:
//   - 触发区: h-8 rounded-lg border-input bg-background, 左侧 px-2.5 文字, 右侧 chevron-down 图标;
//   - 未选择时显示占位符(muted 色), 且箭头半透明;
//   - focus-visible: 边框变 Ring + ring-3 ring-ring/50(键盘焦点才显示);
//   - 禁用 opacity-50。
type ComboBox struct {
	*widget.ComboBox
	placeholder string
	cb          func(index int32)
}

// ComboBoxOption 下拉框配置项。
type ComboBoxOption func(*ComboBox)

// WithComboPlaceholder 设置未选择时的占位文本。
func WithComboPlaceholder(s string) ComboBoxOption {
	return func(c *ComboBox) { c.placeholder = s }
}

// NewComboBox 创建下拉框。cy 传 0 时使用默认高度 32; items 是初始选项。
func NewComboBox(x, y, cx, cy int32, hParent int, items []string, opts ...ComboBoxOption) *ComboBox {
	if cy <= 0 {
		cy = 32
	}
	cb := widget.NewComboBox(x, y, cx, cy, hParent)
	if cb == nil {
		return nil
	}
	c := &ComboBox{ComboBox: cb}
	for _, o := range opts {
		o(c)
	}

	// 组合框必须有适配器表才能装数据(没有它 AddItemText 会报"句柄无效")
	cb.CreateAdapter()
	for _, it := range items {
		cb.AddItemText(it)
	}
	cb.SetSelItem(-1) // 默认不选中, 显示占位符
	if c.placeholder != "" {
		cb.SetDefaultText(c.placeholder)
	}
	cb.EnableEdit(false)
	cb.EnableDrawButton(false) // 原生箭头不要, 自绘
	cb.EnableBkTransparent(true)
	cb.SetBorderSize(10, 6, 30, 6) // 右侧留出箭头的位置
	// 触发区文字用主题字体(原生默认字体与 shadcn text-sm 差一个家族/字号)
	if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
		cb.SetFont(f)
	}
	// 触发区不吃滚轮: 原生会滚动内部文本(实测滚轮使文字上移), shadcn 的触发区对
	// 滚轮无响应; 禁用后事件交给父元素, 弹层 ListBox 是独立元素不受影响。
	cb.EnableEvent_XE_MOUSEWHEEL(false)
	// 弹层高度: 条目数 × 条目高 + p-1 上下边距, 在构造时设置(实测必须在弹层打开前
	// 设置才生效; 固定模式原生才使用该值)。超过 10 项给最大高度, 交给滚动条。
	if n := int32(len(items)); n > 0 {
		h := n*comboItemH + 8
		if maxH := int32(10)*comboItemH + 8; h > maxH {
			h = maxH
		}
		cb.EnableDropHeightFixed(true)
		cb.SetDropHeight(h)
	}
	cb.AddEvent_Paint(c.onPaint)
	cb.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	cb.AddEvent_Combobox_Select(func(hEle int, iItem int32, pbHandled *bool) int {
		cb.Redraw(false)
		if c.cb != nil {
			c.cb(iItem)
		}
		return 0
	})
	// 下拉列表弹出时才拿得到窗口/列表句柄
	cb.AddEvent_Combobox_Popup_List(func(hEle int, hWindow int, hListBox int, pbHandled *bool) int {
		HideTooltip(hEle) // 弹层窗口接管鼠标后触发元素收不到 MouseLeave, 提示不会自己消失
		styleComboPopup(hEle, hWindow, hListBox)
		return 0
	})
	// 弹层退出后窗口句柄即失效(实测: 关闭后切主题, 用旧句柄调 XWnd_GetDPI 会弹
	// "句柄无效"错误框), 置空让主题回调安全跳过, 下次弹出会重新取句柄。
	cb.AddEvent_Combobox_Exit_List(func(hEle int, pbHandled *bool) int {
		comboPopup.win, comboPopup.list = 0, 0
		return 0
	})
	xui.AttachWindow(cb.Handle)
	return c
}

// Selected 返回当前选中项索引, 未选中返回 -1。
func (c *ComboBox) Selected() int32 { return c.GetSelItem() }

// SetSelected 设置选中项。
func (c *ComboBox) SetSelected(i int32) *ComboBox {
	c.SetSelItem(i)
	c.Redraw(false)
	return c
}

// ItemText 返回指定项文本。
func (c *ComboBox) ItemText(i int32) string { return c.GetItemText(i, 0) }

// AddItem 追加一个选项, 返回索引。
func (c *ComboBox) AddItem(text string) int32 { return c.AddItemText(text) }

// OnChange 注册选中项变化回调。
func (c *ComboBox) OnChange(fn func(index int32)) *ComboBox {
	c.cb = fn
	return c
}

func (c *ComboBox) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 外壳与文字全部自绘
	t := xui.Current()
	st := xui.StateOf(hEle)

	// 触发区: select.tsx 是 border-input bg-transparent —— 组件都放在卡片上,
	// 透明底等价于 Card 色(用 Background 会在暗色下与卡片出现色差)。
	bg, border := t.Card, t.Input
	if st.Disabled {
		bg = t.Input.WithAlpha(128) // disabled:bg-input/50
		border = xui.None
	}
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		// focus-visible:border-ring + ring-3 ring-ring/50
		p.FocusFrame(t.Ring, bg, t.RadiusLg())
	} else {
		p.FillRound(&p.Rc, bg, t.RadiusLg())
		p.StrokeRound(&p.Rc, border, t.RadiusLg(), 1)
	}

	// 文字交给原生绘制(自己再画一遍会与原生文字重叠成重影), 这里只按状态同步颜色。
	// 未选中时原生显示 defaultText(占位符), 用 MutedFore 色。
	// 禁用是组级透明(disabled:opacity-50): 文字核心保持原色, 只有壳体变淡。
	fore := t.Foreground
	if c.GetSelItem() < 0 {
		fore = t.MutedFore
	}
	c.SetTextColor(uint32(fore))
	c.SetDefaultTextColor(uint32(t.MutedFore))

	// 右侧 chevron-down: select.tsx 用 text-muted-foreground(全色, 不随状态加透明)
	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(xui.IconChevronDown, &icon, t.MutedFore, 2)
	return 0
}

// comboPopup 记录当前下拉弹层的句柄, 供主题切换时重新上色与条目自绘。
var comboPopup = struct {
	win   int
	list  int
	combo int
	once  bool
}{}

// comboPopupBound 记录已注册过事件的弹层列表(窗口复用, 避免重复注册)。
var comboPopupBound = map[int]bool{}

// comboItemH 是弹层条目高度: shadcn SelectItem = text-sm(20) + py-1(4×2)。
const comboItemH int32 = 28

// styleComboPopup 给原生下拉弹层做主题上色与样式对齐。
//
// 为什么这里碰了背景管理器: 下拉弹层是炫彩自己创建的弹出窗口与原生 ListBox, 我们
// 只拿得到句柄, 没有可用的自绘入口。上色前先 Clear(而不是靠计数守卫), 这样主题切换
// 时可以直接重刷一遍而不会累积背景对象(炫彩上限 20 个)。
func styleComboPopup(hCombo, hWindow, hListBox int) {
	comboPopup.combo, comboPopup.win, comboPopup.list = hCombo, hWindow, hListBox
	applyComboPopupStyle()

	if !comboPopup.once {
		comboPopup.once = true
		xui.OnThemeChange(func(*xui.Theme) { applyComboPopupStyle() })
	}

	lb := widget.NewListBoxByHandle(hListBox)
	if lb == nil {
		return
	}
	// 事件只注册一次: 弹层窗口/列表在多次开关间复用, 每次弹出都注册会累积回调。
	if !comboPopupBound[hListBox] {
		comboPopupBound[hListBox] = true
		lb.AddEvent_ListBox_DrawItem(comboPopupDrawItem)
		// ring-1 ring-foreground/10: 元素背景的 border 对象是方角, 圆角处会被窗口
		// 区域裁掉(实测), 改在 paint 事件里自绘圆角描边(bk 之后、条目之前绘制,
		// 条目内容距边缘 4px 不会盖住它)。
		lb.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
			p := xui.BeginPaint(hEle, hDraw)
			if p == nil {
				return 0
			}
			t := xui.Current()
			p.StrokeRound(&p.Rc, t.Foreground.WithAlpha(26), t.RadiusLg(), 1)
			return 0
		})
		lb.AddEvent_ListBox_Temp_Create_End(func(hEle int, pItem *xc.ListBox_Item_, nFlag int32, pbHandled *bool) int {
			if nFlag != 1 || pItem == nil {
				return 0
			}
			if h := lb.GetTemplateObject(pItem.Index, 1); h != 0 {
				xc.XShapeText_SetTextColor(h, uint32(xui.Current().PopoverFore))
				if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
					xc.XShapeText_SetFont(h, f)
				}
				xc.XWidget_LayoutItem_SetMargin(h, 8, 0, 0, 0) // px-2
			}
			return 0
		})
	}
}

// applyComboPopupStyle 按当前主题刷新弹层配色(窗口 + 列表 + 滚动条)。
func applyComboPopupStyle() {
	if comboPopup.list == 0 {
		return
	}
	t := xui.Current()

	// 弹层窗口圆角: Win32 区域裁剪(SetWindowRgn)。xcgui 的 shaped 透明对原生弹层
	// 实测无效(窗口白底不受 BkM 控制且设透明后仍方角), 区域裁剪在 Win32 层生效。
	// SetWindowRound 的 radius 是物理像素, 需自行按 DPI 换算。
	if hw := comboPopup.win; hw > 0 {
		if w := window.NewByHandle(hw); w != nil {
			dpi := xc.XWnd_GetDPI(hw)
			if dpi <= 0 {
				dpi = 96
			}
			r := int32(float64(t.RadiusLg()) * float64(dpi) / 96)
			wnd.SetWindowRound(w.GetHWND(), r)
		}
	}

	lb := widget.NewListBoxByHandle(comboPopup.list)
	if lb == nil {
		return
	}
	lb.ClearBkInfo()
	lb.SetItemHeightDefault(comboItemH, comboItemH)
	lb.SetPadding(4, 4, 4, 4) // p-1: 条目距弹层边缘 4px
	lb.SetDrawItemBkFlags(0)  // 原生项背景(方角灰块)全弃, 交给 DrawItem 自绘
	lb.EnableAutoShowScrollBar(true)
	// 弹层底色 + ring-1 ring-foreground/10: 元素背景对象置圆角(select.tsx 的
	// rounded-lg + ring), 窗口异型透明后圆角外真透明。
	if b := lb.GetBkManagerObj(); b != nil {
		b.Clear()
		st := xcc.Element_State_Flag_Leave
		b.AddFill(st, uint32(t.Popover), 1)
		b.AddBorder(st, uint32(t.Foreground.WithAlpha(26)), 1, 2)
		r := t.RadiusLg()
		if o := b.GetObjectObj(1); o != nil {
			o.SetRectRoundAngle(r, r, r, r)
		}
		if o := b.GetObjectObj(2); o != nil {
			o.SetRectRoundAngle(r, r, r, r)
		}
	}
	styleComboScrollBar(lb.GetScrollBarVObj(), t)
	styleComboScrollBar(lb.GetScrollBarHObj(), t)
}

// comboPopupDrawItem 自绘条目: hover/选中画 rounded-md 的 accent 底(focus:bg-accent),
// 选中项右侧画 check 图标(select.tsx: CheckIcon absolute right-2 size-4)。
// 文字是项模板里的 shapeText 子元素, 在项背景之后绘制, 不受本回调影响。
func comboPopupDrawItem(hEle int, hDraw int, pItem *xc.ListBox_Item_, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// 选中项: 原生在弹层打开时不会给选中项置 Select 态(探针实证), 这里自己比对
	// 当前选中索引, 复刻 shadcn 打开时选中项自动高亮(focus 选中项)的行为。
	sel := int32(-1)
	if comboPopup.combo > 0 {
		sel = xc.XComboBox_GetSelItem(comboPopup.combo)
	}
	hot := pItem.NState == xcc.List_Item_State_Stay || pItem.NState == xcc.List_Item_State_Select ||
		(pItem.NState == xcc.List_Item_State_Leave && pItem.Index == sel)
	if hot {
		// rounded-md 在 shadcn 里 = var(--radius)-4 = 6, 对应本主题的 RadiusSm
		p.FillRound(&pItem.RcItem, t.Accent, t.RadiusSm())
	}
	if pItem.Index == sel {
		rc := pItem.RcItem
		icon := xc.RECT{
			Left:   rc.Right - 24, // right-2 + size-4
			Top:    rc.Top + (rc.Bottom-rc.Top-16)/2,
			Right:  rc.Right - 8,
			Bottom: rc.Top + (rc.Bottom-rc.Top+16)/2,
		}
		p.Icon(xui.IconCheck, &icon, t.PopoverFore, 2)
	}
	return 0
}

// styleComboScrollBar 把弹层列表的滚动条改成细条 + 主题色。
func styleComboScrollBar(sb *widget.ScrollBar, t *xui.Theme) {
	if sb == nil || sb.Handle <= 0 {
		return
	}
	sb.ShowButton(false) // 隐藏上下箭头, 只留滑块
	sb.SetSliderPadding(2)
	sb.EnableBkTransparent(true)
	sb.ClearBkInfo()
	sb.AddBkFill(xcc.Element_State_Flag_Leave, uint32(t.Popover))
	sb.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.Popover))
	if sl := sb.GetButtonSliderObj(); sl != nil && sl.Handle > 0 {
		sl.ClearBkInfo()
		sl.AddBkFill(xcc.Element_State_Flag_Leave, uint32(t.Border))
		sl.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.MutedFore))
		sl.AddBkFill(xcc.Element_State_Flag_Down, uint32(t.MutedFore))
	}
}
