package ui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// Alert 几何常量(对齐 shadcn alert.tsx 的 radix-nova 样式, 单位逻辑像素):
//   - 卡片: rounded-lg(radius-lg) + border, bg-card, px-2.5 py-2;
//   - 图标列: size-4(16), 与文本列间距 gap-x-2(8), 垂直下移 translate-y-0.5(2);
//   - 标题/描述: text-sm(14), 行间 gap-0.5(2);
//   - 动作槽: 贴右上 top-2 right-2, 有动作时右侧留白收到 8(文本列缩短)。
const (
	alertPadX      int32 = 10 // px-2.5
	alertPadY      int32 = 8  // py-2
	alertGapX      int32 = 8  // gap-x-2
	alertGapY      int32 = 2  // gap-0.5
	alertIconSize  int32 = 16 // size-4
	alertIconShift int32 = 2  // translate-y-0.5
	alertLineH     int32 = 20 // text-sm 的默认行高 1.25rem
	alertActionOff int32 = 8  // top-2 right-2
	// 原生 Button 的文字排版区比元素窄/矮一圈(默认内边距): 量高时宽度要收一点,
	// 多行高度也要补一点, 否则最后一行的字形会被裁掉并出现省略号(实测)。
	alertTextInsetX int32 = 12
	alertTextPadY   int32 = 4
)

// AlertVariant 提示条变体, 对应 shadcn alert.tsx 的 variant。
type AlertVariant int

const (
	AlertDefault     AlertVariant = iota // bg-card + text-card-foreground
	AlertDestructive                     // 标题/图标 destructive, 描述 destructive/90
)

// AlertOption 提示条配置项。
type AlertOption func(*Alert)

// WithAlertVariant 设置变体(默认 AlertDefault)。
func WithAlertVariant(v AlertVariant) AlertOption { return func(a *Alert) { a.variant = v } }

// WithAlertIcon 设置前置图标(path 数据或整段 SVG, 见 xui.Painter.Icon; 传空串 = 无图标)。
func WithAlertIcon(path string) AlertOption { return func(a *Alert) { a.icon = path } }

// WithAlertTitle 设置标题(text-sm, 中等字重)。传空串 = 不显示标题。
func WithAlertTitle(text string) AlertOption { return func(a *Alert) { a.titleText = text } }

// WithAlertDescription 设置描述(text-sm, 弱化色; destructive 变体用 destructive/90)。
func WithAlertDescription(text string) AlertOption { return func(a *Alert) { a.descText = text } }

// Alert 静态提示条: 图标 + 标题 + 描述 + 右上动作槽。
//
// 与 shadcn alert.tsx(radix-nova) 对齐:
//   - 卡片 bg-card + 1px border + rounded-lg, 内边距 10/8;
//   - 可选前置图标 size-4, 与文字列间距 8、顶部下移 2;
//   - 标题 text-sm 中等字重, 描述 text-sm muted-foreground, 行间 2;
//   - destructive 变体: 标题与图标用 destructive, 描述用 destructive/90;
//   - 动作槽贴右上(top-2 right-2), 放 Button / Badge 等元素。
//
// 高度按内容自撑: 标题与描述会在当前宽度下折行, 组件量出高度后重排(见 DESIGN 第八节 90)。
// 宽度仍由调用方声明(通常 `xui.ItemFill(alert)`), 组件只把高度声明成 Auto。
type Alert struct {
	*widget.LayoutEle
	variant AlertVariant
	icon    string

	iconEle   *widget.Element
	row       *widget.LayoutEle // 内边距容器: 图标 / 文本列 / 动作槽
	text      *widget.LayoutEle // 文本列(图标右侧, 宽度占剩余)
	title     *Label
	desc      *Label
	action    *widget.LayoutEle
	titleText string
	descText  string

	reflowPending bool  // 已预约一次量高重排(合并同一轮里的多次触发)
	measuredW     int32 // 上次量高时的文本列宽度(见 onPaint)
}

// NewAlert 创建提示条。cx 为初始宽度(布局项声明会覆盖), cy 仅作初始高度, 实际高度按内容自撑。
func NewAlert(x, y, cx, cy int32, hParent int, opts ...AlertOption) *Alert {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	a := &Alert{LayoutEle: ele, variant: AlertDefault}
	for _, o := range opts {
		o(a)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.AddEvent_Paint(a.onPaint)
	xui.AttachWindow(ele.Handle)

	a.row = widget.NewLayoutEle(0, 0, 0, 0, ele.Handle)
	a.row.EnableHorizon(true)
	a.row.EnableLayout(true)
	a.row.EnableBkTransparent(true)
	a.row.EnableAutoWrap(false)
	a.row.SetSpace(alertGapX)
	a.row.SetAlignV(xcc.Layout_Align_Top)
	a.row.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	a.row.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	a.row.LayoutItem_SetMargin(alertPadX, alertPadY, alertPadX, alertPadY)

	a.iconEle = widget.NewElement(0, 0, alertIconSize, alertIconSize, a.row.Handle)
	a.iconEle.EnableBkTransparent(true)
	a.iconEle.EnableDrawBorder(false)
	a.iconEle.EnableFocus(false)
	a.iconEle.EnableSwitchFocus(false)
	a.iconEle.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, alertIconSize)
	a.iconEle.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, alertIconSize)
	a.iconEle.LayoutItem_SetMargin(0, alertIconShift, 0, 0)
	a.iconEle.AddEvent_Paint(a.onIconPaint)

	a.text = widget.NewLayoutEle(0, 0, 0, 0, a.row.Handle)
	a.text.EnableHorizon(false)
	a.text.EnableLayout(true)
	a.text.EnableBkTransparent(true)
	a.text.EnableAutoWrap(false)
	a.text.SetSpace(alertGapY)
	a.text.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	a.text.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)

	a.title = NewLabel(0, 0, 0, alertLineH, "", a.text.Handle).SetMedium().SetWrap()
	a.title.SetColorFunc(a.foreground)
	a.title.SetText(a.titleText)
	a.title.Show(a.titleText != "")
	a.title.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	a.title.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, alertLineH)

	a.desc = NewLabel(0, 0, 0, alertLineH, "", a.text.Handle).SetMuted().SetWrap()
	a.desc.SetColorFunc(a.descriptionColor)
	a.desc.SetText(a.descText)
	a.desc.Show(a.descText != "")
	a.desc.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	a.desc.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, alertLineH)

	// 动作槽默认隐藏: 隐藏子项不参与布局, 没有动作时不占列也不产生间距。
	a.action = widget.NewLayoutEle(0, 0, 0, 0, a.row.Handle)
	a.action.EnableHorizon(true)
	a.action.EnableLayout(true)
	a.action.EnableBkTransparent(true)
	a.action.EnableAutoWrap(false)
	a.action.SetSpace(alertGapX)
	a.action.SetAlignV(xcc.Layout_Align_Top)
	a.action.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	a.action.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	a.action.Show(false)

	a.iconEle.Show(a.icon != "")
	a.action.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		a.scheduleReflow()
		return 0
	})
	// 文本列宽度变化 = 外层宽度或动作区变了, 需要重新量标题/描述折行高度。
	a.text.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		a.scheduleReflow()
		return 0
	})
	a.scheduleReflow()
	return a
}

// Action 返回右上角动作槽(横向流式容器)。首次调用时显示动作列。
// 往里面放按钮/徽章时用 xui.ItemWidth 或让控件自量宽; 动作列按内容自撑, 顶端对齐。
func (a *Alert) Action() *widget.LayoutEle {
	if a == nil || a.action == nil {
		return nil
	}
	if !a.action.IsShow() {
		a.action.Show(true)
		// 动作贴右上时右侧外边距收到 8(与 shadcn 的 right-2 一致); 无动作时保持 10。
		a.row.LayoutItem_SetMargin(alertPadX, alertPadY, alertActionOff, alertPadY)
		a.scheduleReflow()
	}
	return a.action
}

// Title 返回标题 Label(可继续改字号/字重; 改完调用 Refresh 重量高度)。
func (a *Alert) Title() *Label {
	if a == nil {
		return nil
	}
	return a.title
}

// Description 返回描述 Label(可继续改字号/字重; 改完调用 Refresh 重量高度)。
func (a *Alert) Description() *Label {
	if a == nil {
		return nil
	}
	return a.desc
}

// SetVariant 切换变体(实时换色, 不需要重排)。
func (a *Alert) SetVariant(v AlertVariant) *Alert {
	if a == nil {
		return a
	}
	a.variant = v
	a.Redraw(false)
	a.title.Redraw(false)
	a.desc.Redraw(false)
	a.iconEle.Redraw(false)
	return a
}

// SetIcon 设置前置图标(path 数据或整段 SVG; 空串 = 隐藏图标)。
func (a *Alert) SetIcon(path string) *Alert {
	if a == nil {
		return a
	}
	a.icon = path
	a.iconEle.Show(path != "")
	a.iconEle.Redraw(false)
	a.scheduleReflow()
	a.Redraw(false)
	return a
}

// SetTitle 设置标题; 传空串隐藏标题。
func (a *Alert) SetTitle(s string) *Alert {
	if a == nil {
		return a
	}
	a.titleText = s
	a.title.SetText(s)
	a.title.Show(s != "")
	a.scheduleReflow()
	return a
}

// SetDescription 设置描述; 传空串隐藏描述。
func (a *Alert) SetDescription(s string) *Alert {
	if a == nil {
		return a
	}
	a.descText = s
	a.desc.SetText(s)
	a.desc.Show(s != "")
	a.scheduleReflow()
	return a
}

// Refresh 重新量测折行高度并重排(外部改过 Title/Description 的字体或文本后调用)。
func (a *Alert) Refresh() *Alert {
	if a == nil {
		return a
	}
	a.scheduleReflow()
	a.Redraw(false)
	return a
}

// scheduleReflow 把量高/重排推迟到下一轮 UI 消息: 尺寸事件里直接改高度并 AdjustLayout
// 会重入布局, 走 AfterFunc 合并同轮触发。
func (a *Alert) scheduleReflow() {
	if a == nil || a.reflowPending {
		return
	}
	a.reflowPending = true
	app.AfterFunc(0, func() {
		a.reflowPending = false
		a.reflow()
	})
}

// reflow 按文本列当前宽度量出标题/描述的折行高度并写回。
// 高度变化后要从自己开始逐级通知祖先重排 —— 只调自己或直接父级时, Auto 高的卡片/行
// 仍按旧高度排版; 遇到 ScrollArea 的 LayoutFrame 交给框架排(见 DESIGN 第八节 90)。
func (a *Alert) reflow() {
	if a == nil || !xc.XC_IsHXCGUI(a.Handle, xcc.XC_ELE) {
		return
	}
	w := a.text.GetWidth()
	if w <= 0 {
		return
	}
	a.measuredW = w
	// 原生 Button 的文字排版区域比元素窄一圈(默认内边距), 直接按元素宽量会少数一行;
	// 用收窄后的宽度量高。见 DESIGN 第八节 90。
	measureW := w - alertTextInsetX
	if measureW < alertLineH {
		measureW = w
	}
	titleH := int32(0)
	if a.title.IsShow() {
		titleH = alertTextHeight(a.title.WrappedHeight(measureW))
	}
	descH := int32(0)
	if a.desc.IsShow() {
		descH = alertTextHeight(a.desc.WrappedHeight(measureW))
	}
	changedTitle := a.setTextHeight(a.title, titleH)
	changedDesc := a.setTextHeight(a.desc, descH)
	if !changedTitle && !changedDesc {
		return
	}
	xc.XEle_AdjustLayout(a.Handle)
	for p := xc.XWidget_GetParentEle(a.Handle); p != 0; p = xc.XWidget_GetParentEle(p) {
		if xc.XC_IsHXCGUI(p, xcc.XC_LAYOUT_FRAME) {
			// ScrollArea 的框架自己会重排内容区并重算滚动范围。
			xc.XEle_AdjustLayout(p)
			break
		}
		if pp := xc.XWidget_GetParentEle(p); pp != 0 && xc.XC_IsHXCGUI(pp, xcc.XC_LAYOUT_FRAME) {
			// p 是框架的内容区: 直接对内容区 AdjustLayout 会把 Alert 宽度打成 0(实测),
			// 必须交给框架排。
			xc.XEle_AdjustLayout(pp)
			break
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_ELE) {
			xc.XEle_AdjustLayout(p)
			continue
		}
		if xc.XC_IsHXCGUI(p, xcc.XC_WINDOW) {
			xc.XWnd_AdjustLayout(p)
		}
		break
	}
	a.Redraw(false)
	a.text.Redraw(false)
	a.title.Redraw(false)
	a.desc.Redraw(false)
	if hwnd := xc.XWidget_GetHWND(a.Handle); hwnd != 0 {
		if hw := xc.XC_hWindowFromHWnd(hwnd); hw != 0 {
			xc.XWnd_Redraw(hw, false)
		}
	}
}

// alertTextHeight 把量测高度向上取整到整行, 再多补一点高度。
// GetTextShowRect 返回的 CY 略小于"行数 × 行高"(14px 下三行返回 59 而不是 60),
// 而原生 Button 的文字区上下还各留了一点和内边距 —— 只取整时最后一行会被裁掉并出现省略号,
// 实测三行需要 64(60+4) 才能完整显示。单行保持 20, 跟 shadcn 的行高一致。
func alertTextHeight(h int32) int32 {
	if h <= alertLineH {
		return alertLineH
	}
	return (h+alertLineH-1)/alertLineH*alertLineH + alertTextPadY
}

// setTextHeight 写回文本高度; 返回是否发生变化。0 = 文本隐藏, 保持原高度(隐藏项不参与布局)。
func (a *Alert) setTextHeight(l *Label, h int32) bool {
	if h <= 0 || l.GetHeight() == h {
		return false
	}
	l.SetHeight(h)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	return true
}

// foreground 返回标题/图标的语义色(对应 text-card-foreground / text-destructive)。
func (a *Alert) foreground(t *xui.Theme) xui.Color {
	if a.variant == AlertDestructive {
		return t.Destructive
	}
	return t.CardForeground
}

// descriptionColor 返回描述色(text-muted-foreground / text-destructive/90)。
func (a *Alert) descriptionColor(t *xui.Theme) xui.Color {
	if a.variant == AlertDestructive {
		return t.Destructive.ScaleAlpha(0.9)
	}
	return t.MutedForeground
}

func (a *Alert) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	// 尺寸事件在"布局赋予宽度"时不一定触发; 首次绘制时宽度已经确定, 在这里补一次量高预约。
	if w := a.text.GetWidth(); w > 0 && w != a.measuredW {
		a.scheduleReflow()
	}
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	p.FillRound(&p.Rc, t.Card, t.RadiusLg())
	p.StrokeRound(&p.Rc, t.Border, t.RadiusLg(), 1)
	return 0
}

func (a *Alert) onIconPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil || a.icon == "" {
		return 0
	}
	*pbHandled = true
	p.Icon(a.icon, &p.Rc, a.foreground(xui.Current()), 2)
	return 0
}
