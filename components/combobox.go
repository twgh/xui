package ui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wnd"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// ComboBox 自绘下拉选择框: 载体为原生 ComboBox(弹出、选择、键盘/滚动行为全部原生),
// 外壳、文字与右侧箭头自绘; 下拉列表是原生弹出窗口 + 原生 ListBox, 无法逐像素接管,
// 因此只在弹出时做主题上色(见 stylePopup)。
//
// 与 shadcn select.tsx(radix-nova) 对齐:
//   - 触发区: size=default h-8 rounded-lg / size=sm h-7, border-input bg-transparent
//     (放卡上 = Card 色), 左侧 px-2.5 文字, 右侧 chevron-down 图标;
//   - 未选择时显示占位符(muted 色);
//   - focus-visible: 边框变 Ring + ring-3 ring-ring/50(键盘焦点才显示);
//   - 禁用 opacity-50。
type ComboBox struct {
	*widget.ComboBox
	placeholder string
	cb          func(index int32)
	size        ComboBoxSize
}

// ComboBoxSize 触发区尺寸预设, 对应 shadcn select.tsx 的 size
// (data-[size=default]:h-8 / data-[size=sm]:h-7)。
type ComboBoxSize int

const (
	ComboSizeDefault ComboBoxSize = iota // 高 32
	ComboSizeSm                          // 高 28, 圆角 min(--radius-md, 10)
)

// ComboBoxOption 下拉框配置项。
type ComboBoxOption func(*ComboBox)

// WithComboPlaceholder 设置未选择时的占位文本。
func WithComboPlaceholder(s string) ComboBoxOption {
	return func(c *ComboBox) { c.placeholder = s }
}

// WithComboSize 设置触发区尺寸预设(sm = h-7 = 28)。
func WithComboSize(s ComboBoxSize) ComboBoxOption {
	return func(c *ComboBox) { c.size = s }
}

// NewComboBox 创建下拉框。cy 传 0 时使用默认高度 32; items 是初始选项。
func NewComboBox(x, y, cx, cy int32, hParent int, items []string, opts ...ComboBoxOption) *ComboBox {
	c := &ComboBox{}
	for _, o := range opts { // 先应用配置: 尺寸预设决定默认高度
		o(c)
	}
	if cy <= 0 {
		if c.size == ComboSizeSm {
			cy = 28
		} else {
			cy = 32
		}
	}
	cb := widget.NewComboBox(x, y, cx, cy, hParent)
	if cb == nil {
		return nil
	}
	c.ComboBox = cb

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
	// 弹层高度: 按内容高公式设置(见 comboDropHeight), 在构造时设置(实测必须在弹层
	// 打开前设置才生效; 固定模式原生才使用该值)。超过 10 项给最大高度, 交给滚动条。
	if n := int32(len(items)); n > 0 {
		h := comboDropHeight(n)
		if maxH := comboDropHeight(10); h > maxH {
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
		alignComboPopup(hEle, hWindow)
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
	// 圆角: default = rounded-lg(RadiusLg); sm = rounded-[min(--radius-md,10px)]
	radius := t.RadiusLg()
	if c.size == ComboSizeSm {
		if md := t.RadiusMd(); md < 10 {
			radius = md
		} else {
			radius = 10
		}
	}
	if st.Focused && !st.Disabled && xui.FocusedByKeyboard() {
		// focus-visible:border-ring + ring-3 ring-ring/50
		p.FocusFrame(t.Ring, bg, radius)
	} else {
		p.FillRound(&p.Rc, bg, radius)
		p.StrokeRound(&p.Rc, border, radius, 1)
	}

	// 文字交给原生绘制(自己再画一遍会与原生文字重叠成重影), 这里只按状态同步颜色。
	// 未选中时原生显示 defaultText(占位符), 用 MutedForeground 色。
	// 禁用是组级透明(disabled:opacity-50): 文字核心保持原色, 只有壳体变淡。
	fore := t.Foreground
	if c.GetSelItem() < 0 {
		fore = t.MutedForeground
	}
	c.SetTextColor(uint32(fore))
	c.SetDefaultTextColor(uint32(t.MutedForeground))

	// 右侧 chevron-down: select.tsx 用 text-muted-foreground(全色, 不随状态加透明)
	icon := xc.RECT{
		Left:   p.Rc.Right - 26,
		Top:    p.Rc.Top + (p.H()-16)/2,
		Right:  p.Rc.Right - 10,
		Bottom: p.Rc.Top + (p.H()-16)/2 + 16,
	}
	p.Icon(icons.IconChevronDown, &icon, t.MutedForeground, 2)
	return 0
}

// comboPopup 记录当前下拉弹层的句柄, 供主题切换时重新上色与条目自绘。
var comboPopup = struct {
	win   int
	list  int
	combo int
	once  bool
}{}

// alignComboPopup 把弹层挪到 item-aligned 位置(radix Select 的默认定位):
// 弹层盖住触发元素, 当前选中项正对触发元素位置(select.tsx 的 SelectContent
// position="item-aligned")。两种盒子高度不同(触发区 32、条目 comboItemH),
// 按**文字中心**对齐(见下面 off 的推导), 否则点开瞬间选中项文字会往下跳。
//
// 坐标链路(探针实测): XEle_GetWndClientRect 是逻辑像素, 原点=所属窗口矩形原点
// (含阴影边距); wapi.GetWindowRect 是物理屏幕像素 → 逻辑差值 × DPI/96 换算。
// 原生默认把弹层放在触发区正下方, 这里用 SetWindowPos(物理像素)重新定位。
func alignComboPopup(hCombo, hWindow int) {
	if hCombo <= 0 || hWindow <= 0 {
		return
	}
	var trig xc.RECT
	xc.XEle_GetWndClientRect(hCombo, &trig)
	hw := xc.XWidget_GetHWINDOW(hCombo)
	if hw <= 0 {
		return
	}
	var mrc xc.RECT
	if !wapi.GetWindowRect(xc.XWnd_GetHWND(hw), &mrc) {
		return
	}
	dpi := xc.XWnd_GetDPI(hWindow)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96

	// 弹层顶部相对触发区顶部的偏移: 让"选中项"与触发区按**文字中心**重合(radix 的
	// item-aligned 就是把选中项盖在触发区上)。两种盒子高度不同(触发器高 32、条目
	// comboItemH), 所以按中心对齐: 选中项顶 = 触发区顶 + (触发区高-项高)/2。
	// 选中项在内容里的顶 = p-1 + ListBox 固有 3px 内缩 + 前面条目(步进 = 项高+1,
	// 见 comboDropHeight); 未选中时按 radix 把焦点放首项, 首项与触发区中心重合。
	sel := xc.XComboBox_GetSelItem(hCombo)
	if sel < 0 {
		sel = 0
	}
	selTop := comboPad + 3 + sel*(comboItemH+1)
	off := selTop - (trig.Bottom-trig.Top-comboItemH)/2
	desired := mrc.Top + int32(float64(trig.Top-off)*sc)
	left := mrc.Left + int32(float64(trig.Left)*sc)

	// 屏幕边界截断(物理像素, 主屏): 顶边贴屏时向下压并滚动列表让选中项仍可见。
	hwnd := xc.XWnd_GetHWND(hWindow)
	var prc xc.RECT
	wapi.GetWindowRect(hwnd, &prc)
	ph := prc.Bottom - prc.Top
	screenTop, screenBottom := int32(0), wapi.GetSystemMetrics(wapi.SM_CYSCREEN)
	if desired < screenTop {
		cut := int32(float64(screenTop-desired) / sc)
		desired = screenTop
		if lb := widget.NewListBoxByHandle(comboPopup.list); lb != nil && cut > 0 {
			lb.ScrollPosYV(selTop - comboPad) // 保留 p-1 顶边距, 选中项作为首个可见项
		}
	}
	if desired+ph > screenBottom {
		desired = screenBottom - ph
		if desired < screenTop {
			desired = screenTop
		}
	}
	wapi.SetWindowPos(hwnd, 0, left, desired, 0, 0,
		wapi.SWP_NOSIZE|wapi.SWP_NOZORDER|wapi.SWP_NOACTIVATE)
}

// comboItemH 是弹层条目高度: shadcn SelectItem = text-sm(20) + py-1(4×2)。
const comboItemH int32 = 28

// comboPad 是弹层内容四周内边距(shadcn 的 p-1)。
const comboPad int32 = 4

// comboDropHeight 返回 n 个条目时下拉弹层应有的高度(逻辑像素)。
//
// 探针实测(itemH=28, pad=4, 亦验证了 pad=0/8 与 n=3/5/6/10):
//   - ListBox 内容有固有内缩: 首项 top = pad+3(上下各固有 3px、左右固有 2px);
//   - 行步进 = 项高 + 1px(相邻项 top 相差 29);
//   - 内容高 = n*(项高+1) + 2*pad + 5(顶固有 3 + 底固有 3 - 最后 1px 间隙);
//   - 弹层高 < 内容高会出滚动(差多少滚多少, 实测 H=180/6 项差 7px),
//     高 > 内容高则底部多留白。
//
// 取高度 = 内容高: 无滚动, 且顶部/底部留白对称(pad+3 = 7px)。旧算法(28n+12)
// 在 6 项时少 7px, 底部贴边且能再滚一下。
func comboDropHeight(n int32) int32 {
	return n*(comboItemH+1) + 2*comboPad + 5
}

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
	// 弹层窗口/ListBox 每次弹出都是新建的(退出即销毁), 事件在这里注册即可, 不会重复。
	lb.AddEvent_ListBox_DrawItem(comboPopupDrawItem)
	// ring-1 ring-foreground/10: 描边画在 PAINT_END(该元素及子元素绘制完成事件)。
	// 实测两个坑: ①ListBox 自身的 paint 事件在这个弹层窗口上不渲染输出(内缩 10px
	// 的 6px 红色粗环都画不出来, 探针实证), 只有 BkM 与条目 DrawItem 能画; ②BkM
	// 的 border 对象贴边画, 被 SetWindowRgn 的窗口区域裁得只剩左边一条。PAINT_END
	// 在条目之后绘制(边框盖住内容边缘, 同 shadcn), 描边整体内缩 1px 落在窗口区域
	// 内, 四边完整; 圆角弹层上 1px 白边距不可感知。
	lb.EnableEvent_XE_PAINT_END()
	lb.AddEvent_Paint_End(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		t := xui.Current()
		rc := p.Rc
		rc.Left++
		rc.Top++
		rc.Right--
		rc.Bottom--
		p.StrokeRound(&rc, t.Foreground.WithAlpha(26), t.RadiusLg(), 1)
		return 0
	})
	lb.AddEvent_ListBox_Temp_Create_End(func(hEle int, pItem *xc.ListBox_Item_, nFlag int32, pbHandled *bool) int {
		if nFlag != 1 || pItem == nil {
			return 0
		}
		if h := lb.GetTemplateObject(pItem.Index, 1); h != 0 {
			xc.XShapeText_SetTextColor(h, uint32(xui.Current().PopoverForeground))
			if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
				xc.XShapeText_SetFont(h, f)
			}
			// 文字左边距: 目标是让条目文字与触发区文字**左边缘重合**——radix 的
			// SelectItemAlignedPosition 正是按文字对齐(left = 触发区文字 left − 条目文字
			// 相对内容左缘的偏移), 而不是对齐盒子。条目文字相对弹层左缘的内缩 = p-1(4)
			// + ListBox 固有 2 + 本边距; 边距 8(px-2, shadcn 原值)时实测条目文字比触发区
			// 文字右移 2px, 点开瞬间看着像"文字往内收缩", 取 5 后两者墨迹起点完全重合
			// (同为 x 813, 弹层关闭时触发区文字 813..862 / 打开时条目文字 813..859)。
			// 注意: 不能靠移动弹层窗口来对齐 —— 弹层宽度 = 触发区宽度, 窗口左移会使右侧
			// check 与弹层右缘的距离跟着变, 只有改条目的文字内距才能左右都对。
			xc.XWidget_LayoutItem_SetMargin(h, 5, 0, 0, 0)
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
	lb.SetPadding(comboPad, comboPad, comboPad, comboPad) // p-1: 条目距弹层边缘 4px(加固有 3px 内缩 = 视觉 7px)
	lb.SetDrawItemBkFlags(0)  // 原生项背景(方角灰块)全弃, 交给 DrawItem 自绘
	lb.EnableAutoShowScrollBar(true)
	// 滚动条显隐: 原生的显示判定在打开时用旧条目高算好, 我们把条目调矮后它不会
	// 自己收起(实测残影), 用滚动视级开关显式控制。≤10 项内容必然装得下; 单列文本
	// 永远不需要横向滚动。
	lb.ShowSBarV(lb.GetCount_AD() > 10)
	lb.ShowSBarH(false)
	// 弹层底色(select.tsx 的 bg-popover + rounded-lg): 元素背景对象置圆角,
	// 窗口区域裁剪后圆角外真透明。描边(ring)不在这里加——BkM 的 border 对象贴边
	// 画会被窗口区域裁得只剩左边一条(见 PAINT_END 处的注释), ring 改在 PAINT_END 画。
	if b := lb.GetBkManagerObj(); b != nil {
		b.Clear()
		st := xcc.Element_State_Flag_Leave
		b.AddFill(st, uint32(t.Popover), 1)
		r := t.RadiusLg()
		if o := b.GetObjectObj(1); o != nil {
			o.SetRectRoundAngle(r, r, r, r)
		}
	}
	styleComboScrollBar(lb.GetScrollBarVObj(), t)
	styleComboScrollBar(lb.GetScrollBarHObj(), t)
}

// comboPopupDrawItem 自绘条目: 高亮项画 rounded-md 的 accent 底(shadcn 只给 focus 项底色:
// focus:bg-accent), 选中项右侧画 check 图标(select.tsx: CheckIcon absolute right-2 size-4)。
// 文字是项模板里的 shapeText 子元素, 在项背景之后绘制, 不受本回调影响。
func comboPopupDrawItem(hEle int, hDraw int, pItem *xc.ListBox_Item_, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// 选中项: 原生在弹层打开时不会给选中项置 Select 态(探针实证), 这里自己比对
	// 当前选中索引。
	sel := int32(-1)
	if comboPopup.combo > 0 {
		sel = xc.XComboBox_GetSelItem(comboPopup.combo)
	}
	// 底色只属于"指针所在项": shadcn 的 item 是 focus:bg-accent(选中态由右侧 check 表达, 不带
	// 常驻底色)。指针不在条目上(弹层留白、弹层上下左右边缘外)就**没有任何底色** —— 对齐 radix:
	// 条目的 pointerleave 把焦点交回 content(onItemLeave = content.focus()), 高亮随之消失;
	// 既不停留在最后悬停项, 也不回落到选中项(这两版都被实测否掉了)。
	hot := pItem.NState == xcc.List_Item_State_Stay || pItem.NState == xcc.List_Item_State_Select
	// 高亮叠在弹层底色(popover)之上, 按过渡值淡入淡出
	if hv := xui.HoverItem(hEle, pItem.Index, hot); hv > 0 {
		// rounded-md 在 shadcn 里 = var(--radius)-4 = 6, 对应本主题的 RadiusSm
		p.FillRound(&pItem.RcItem, t.Accent.ScaleAlpha(hv), t.RadiusSm())
	}
	if pItem.Index == sel {
		rc := pItem.RcItem
		icon := xc.RECT{
			Left:   rc.Right - 24, // right-2 + size-4
			Top:    rc.Top + (rc.Bottom-rc.Top-16)/2,
			Right:  rc.Right - 8,
			Bottom: rc.Top + (rc.Bottom-rc.Top+16)/2,
		}
		p.Icon(icons.IconCheck, &icon, t.PopoverForeground, 2)
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
		sl.AddBkFill(xcc.Element_State_Flag_Stay, uint32(t.MutedForeground))
		sl.AddBkFill(xcc.Element_State_Flag_Down, uint32(t.MutedForeground))
	}
}
