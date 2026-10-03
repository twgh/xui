package ui

import (
	"github.com/twgh/xcgui/widget"
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
		styleComboPopup(hWindow, hListBox, cy)
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

	bg, border := t.Background, t.Input
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
	fore := t.Foreground
	if c.GetSelItem() < 0 {
		fore = t.MutedFore
	}
	placeholder := t.MutedFore
	if st.Disabled {
		fore = fore.WithAlpha(128) // disabled:opacity-50
		placeholder = placeholder.WithAlpha(128)
	}
	c.SetTextColor(uint32(fore))
	c.SetDefaultTextColor(uint32(placeholder))

	// 右侧 chevron-down(size-4, 半透明)
	iconColor := t.Foreground.WithAlpha(128)
	if st.Disabled {
		iconColor = iconColor.WithAlpha(128)
	}
	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(xui.IconChevronDown, &icon, iconColor, 2)
	return 0
}

// comboPopup 记录当前下拉弹层的句柄, 供主题切换时重新上色。
var comboPopup = struct {
	win   int
	list  int
	itemH int32
	once  bool
}{}

// styleComboPopup 给原生下拉弹层做主题上色。
//
// 为什么这里碰了背景管理器: 下拉弹层是炫彩自己创建的弹出窗口与原生 ListBox, 我们
// 只拿得到句柄, 没有可用的自绘入口。上色前先 Clear(而不是靠计数守卫), 这样主题切换
// 时可以直接重刷一遍而不会累积背景对象(炫彩上限 20 个)。
func styleComboPopup(hWindow, hListBox int, itemH int32) {
	comboPopup.win, comboPopup.list, comboPopup.itemH = hWindow, hListBox, itemH
	applyComboPopupStyle()

	if !comboPopup.once {
		comboPopup.once = true
		xui.OnThemeChange(func(*xui.Theme) { applyComboPopupStyle() })
	}

	lb := widget.NewListBoxByHandle(hListBox)
	if lb == nil {
		return
	}
	lb.AddEvent_ListBox_Temp_Create_End(func(hEle int, pItem *xc.ListBox_Item_, nFlag int32, pbHandled *bool) int {
		if nFlag != 1 || pItem == nil {
			return 0
		}
		if h := lb.GetTemplateObject(pItem.Index, 1); h != 0 {
			xc.XShapeText_SetTextColor(h, uint32(xui.Current().PopoverFore))
			if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
				xc.XShapeText_SetFont(h, f)
			}
			xc.XWidget_LayoutItem_SetMargin(h, 8, 0, 0, 0)
		}
		return 0
	})
}

// applyComboPopupStyle 按当前主题刷新弹层配色(窗口 + 列表 + 滚动条)。
func applyComboPopupStyle() {
	if comboPopup.list == 0 {
		return
	}
	t := xui.Current()
	if hw := comboPopup.win; hw > 0 {
		if bkm := xc.XWnd_GetBkManager(hw); bkm != 0 {
			xc.XBkM_Clear(bkm)
			// 弹层用 Body 状态的填充(与窗口 Leave 状态不同, 探针实证)
			st := xcc.CombinedState(xcc.Window_State_Flag_Body_Leave)
			xc.XBkM_AddFill(bkm, st, uint32(t.Popover), 0)
			xc.XBkM_AddBorder(bkm, st, uint32(t.Foreground.WithAlpha(26)), 1, 0)
		}
	}
	lb := widget.NewListBoxByHandle(comboPopup.list)
	if lb == nil {
		return
	}
	lb.ClearBkInfo()
	lb.SetItemHeightDefault(comboPopup.itemH, comboPopup.itemH)
	lb.SetDrawItemBkFlags(xcc.List_DrawItemBk_Flag_Leave | xcc.List_DrawItemBk_Flag_Stay | xcc.List_DrawItemBk_Flag_Select)
	lb.AddBkFill(xcc.Element_State_Flag_Leave, uint32(t.Popover))
	lb.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.Popover))
	lb.AddBkFill(xcc.List_State_Flag_Item_Leave, uint32(t.Popover))
	lb.AddBkFill(xcc.List_State_Flag_Item_Stay, uint32(t.Accent))   // hover:bg-accent
	lb.AddBkFill(xcc.List_State_Flag_Item_Select, uint32(t.Accent)) // 选中项同样高亮
	// 原生滚动条是系统配色, 与主题完全不搭: 收窄成 8px、去掉上下箭头、轨道透明,
	// 滑块用边框色(悬浮/按下提亮)。
	lb.SetScrollBarSize(8)
	lb.EnableAutoShowScrollBar(true)
	styleComboScrollBar(lb.GetScrollBarVObj(), t)
	styleComboScrollBar(lb.GetScrollBarHObj(), t)
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
