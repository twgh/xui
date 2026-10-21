package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ButtonGroupOrientation 按钮组取向, 对应 shadcn button-group 的 orientation。
type ButtonGroupOrientation int

const (
	ButtonGroupHorizontal ButtonGroupOrientation = iota // 横向: 子项并排, 只保留首尾外角
	ButtonGroupVertical                                 // 纵向: 子项堆叠, 只保留首尾外角
)

// ButtonGroup 把一组按钮拼成无缝的条(对应 shadcn 的 ButtonGroup):
//   - 容器是 LayoutEle, 宽高都按内容自撑(Auto), 子项间距为 0, 相邻边框自然并成一条;
//   - 直系子项直接以 g.Handle 为父创建, 按钮在绘制时按"首/中/尾"收起相邻圆角;
//   - 组里再放组时, 自动给直系子项 8px 间距(shadcn 的 has-[>[data-slot=button-group]]:gap-2);
//   - 纵向组自动消掉按钮底部 1px 按压预留造成的缝(见 fit)。
//
// 用法:
//
//	g := ui.NewButtonGroup(row.Handle)
//	ui.NewButton(0, 0, 0, 0, "Archive", g.Handle, ui.WithVariant(ui.ButtonOutline))
//	ui.NewButton(0, 0, 0, 0, "Report", g.Handle, ui.WithVariant(ui.ButtonOutline))
type ButtonGroup struct {
	*widget.LayoutEle
	vertical bool
	space    int32 // 上次写入的间距, 用于避免在尺寸事件里反复 SetSpace
	fitting  bool
}

// ButtonGroupOption 按钮组配置项。
type ButtonGroupOption func(*ButtonGroup)

// WithButtonGroupOrientation 设置按钮组取向(默认横向)。
func WithButtonGroupOrientation(o ButtonGroupOrientation) ButtonGroupOption {
	return func(g *ButtonGroup) { g.vertical = o == ButtonGroupVertical }
}

// NewButtonGroup 创建按钮组。组本身宽高按内容自撑, 放进 HStack / 行列时无需指定尺寸。
func NewButtonGroup(hParent int, opts ...ButtonGroupOption) *ButtonGroup {
	g := &ButtonGroup{}
	for _, o := range opts {
		o(g)
	}
	ele := widget.NewLayoutEle(0, 0, 0, 0, hParent)
	if ele == nil {
		return nil
	}
	g.LayoutEle = ele
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(0)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	g.applyOrientation()
	ele.AddEvent_Size(g.onSize)
	return g
}

// applyOrientation 把取向写到布局与用户数据标记上(子项绘制时读标记判断首尾角)。
func (g *ButtonGroup) applyOrientation() {
	g.EnableHorizon(!g.vertical)
	if g.vertical {
		g.SetUserData(uiRoleGroupV)
		g.SetAlignH(xcc.Layout_Align_Left)
	} else {
		g.SetUserData(uiRoleGroupH)
		g.SetAlignV(xcc.Layout_Align_Top)
	}
}

// SetOrientation 运行时切换取向(子项圆角按新取向重算, 布局立即重排)。
func (g *ButtonGroup) SetOrientation(o ButtonGroupOrientation) *ButtonGroup {
	v := o == ButtonGroupVertical
	if g.vertical == v {
		return g
	}
	g.vertical = v
	g.applyOrientation()
	g.space = -1 // 强制 fit 重新写间距
	g.fitting = false
	if p := g.GetParentEle(); p != 0 {
		xc.XEle_AdjustLayout(p)
	} else {
		g.AdjustLayout()
	}
	var handled bool
	g.onSize(g.Handle, 0, 0, &handled)
	g.Redraw(false)
	return g
}

// Orientation 返回当前取向。
func (g *ButtonGroup) Orientation() ButtonGroupOrientation {
	if g.vertical {
		return ButtonGroupVertical
	}
	return ButtonGroupHorizontal
}

// onSize 在组尺寸确定后调整子项: 文本块/分隔条拉伸到组高, 纵向组消缝并拉齐宽度。
// fit 内部的布局调整会再次触发 Size 事件, 用 fitting 防止递归。
func (g *ButtonGroup) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	if g.fitting {
		return 0
	}
	g.fitting = true
	defer func() { g.fitting = false }()
	for i := 0; i < 3; i++ {
		if !g.fit() {
			break
		}
		// 子项尺寸/间距变了, 要让**父级**重新给组分配 Auto 宽高; 只调组自身的
		// AdjustLayout 只会重排子项, 组的外框尺寸仍是旧值(嵌套组会溢出, 实测)。
		if p := g.GetParentEle(); p != 0 {
			xc.XEle_AdjustLayout(p)
		} else {
			g.AdjustLayout()
		}
	}
	g.Redraw(false)
	return 0
}

// fit 是一次子项尺寸校准, 返回是否有改动。
func (g *ButtonGroup) fit() bool {
	var rc xc.RECT
	g.GetRect(&rc)
	w, h := rc.Right-rc.Left, rc.Bottom-rc.Top
	if w <= 0 || h <= 0 {
		return false
	}
	n := int(g.GetChildCount())
	if n == 0 {
		return false
	}
	roles := make([]int, n)
	hasButton, hasGroup := false, false
	for i := 0; i < n; i++ {
		child := g.GetChildByIndex(int32(i))
		roles[i] = xc.XEle_GetUserData(child)
		if roles[i] == uiRoleButton {
			hasButton = true
		}
		if roles[i] == uiRoleGroupH || roles[i] == uiRoleGroupV {
			hasGroup = true
		}
	}

	changed := false
	space := int32(0)
	if hasGroup { // shadcn: 组里嵌组时 gap-2
		space = 8
	}
	if g.space != space {
		g.SetSpace(space)
		g.space = space
		changed = true
	}

	// 文本/分隔条的目标高:
	//   - 组高是 Auto: 取最高的按钮视觉高(元素高 − 按压预留), 而不是组高 ——
	//     否则文本块初始 32 会把 sm(28) 组的 Auto 高顶成 32, 再拿它当目标就错 1 档;
	//   - 组高是显式高度(Fixed/Fill/Weight): items-stretch 到组高, 减掉按压预留;
	//   - 组里没有按钮: 用默认按钮高 32(只有文本块的组也能立起来)。
	targetH := int32(0)
	if hasButton {
		for i, role := range roles {
			if role != uiRoleButton {
				continue
			}
			if bh := xc.XEle_GetHeight(g.GetChildByIndex(int32(i))) - pressShift; bh > targetH {
				targetH = bh
			}
		}
	}
	gHt := layoutHeightType(g.Handle)
	explicitH := gHt != xcc.Layout_Size_Auto && gHt != xcc.Layout_Size_Disable
	if explicitH {
		targetH = h
		if hasButton {
			targetH -= pressShift
		}
	}
	if targetH <= 0 {
		targetH = buttonSpecs[ButtonSizeDefault].height
	}

	heights := make([]int32, n)
	prevButton := false
	for i, role := range roles {
		child := g.GetChildByIndex(int32(i))
		if !g.vertical {
			// 横向组不需要消缝; 从纵向切回来时把遗留的 -1 上边距清掉。
			var m xc.RECT
			xc.XWidget_LayoutItem_GetMargin(child, &m)
			if m.Top != 0 {
				xc.XWidget_LayoutItem_SetMargin(child, m.Left, 0, m.Right, m.Bottom)
				changed = true
			}
		}
		if g.vertical {
			// 按钮元素高 = 视觉高 + 1px 按压预留; 竖向相邻按钮要吃掉这条预留缝,
			// 只在"前一个是按钮"时给 -1 上边距(文本/分隔条之后没有预留, 不能叠)。
			mt := int32(0)
			if prevButton {
				mt = -pressShift
			}
			var m xc.RECT
			xc.XWidget_LayoutItem_GetMargin(child, &m)
			if m.Top != mt {
				xc.XWidget_LayoutItem_SetMargin(child, m.Left, mt, m.Right, m.Bottom)
				changed = true
			}
			// items-stretch: 未显式声明宽度的子项拉齐到组宽(声明过 Fixed/Fill/Weight 的保持)。
			var wt xcc.Layout_Size_
			var wv int32
			xc.XWidget_LayoutItem_GetWidth(child, &wt, &wv)
			if (wt == xcc.Layout_Size_Disable || wt == xcc.Layout_Size_Auto) && wv != w {
				xc.XWidget_LayoutItem_SetWidth(child, xcc.Layout_Size_Fixed, w)
				changed = true
			}
		}
		switch role {
		case uiRoleGroupText:
			// 横向组: 文本块拉伸到按钮视觉高(items-stretch);
			// 纵向组: 主轴是高度, 文本保持自身高度(默认 32), 只拉宽。
			th := targetH
			if g.vertical && explicitH {
				th = buttonSpecs[ButtonSizeDefault].height // 主轴不 stretch, 显式高组也用默认按钮高
			}
			heights[i] = th
			if setLayoutHeight(child, th) {
				changed = true
			}
		case uiRoleGroupSep:
			if g.vertical { // 横向分隔条: 宽铺满, 高 1
				heights[i] = 1
				if setLayoutWidth(child, w) || setLayoutHeight(child, 1) {
					changed = true
				}
			} else { // 纵向分隔条: 宽 1, 高对齐视觉高
				heights[i] = targetH
				if setLayoutWidth(child, 1) || setLayoutHeight(child, targetH) {
					changed = true
				}
			}
		default:
			heights[i] = xc.XEle_GetHeight(child)
		}
		prevButton = role == uiRoleButton
	}
	// 组的 Auto 高只会按初始子项算一次(实测: 文本块从 32 压到 28 后, sm 组仍停在
	// 32; 嵌套组改间距后宽也不动)。按本轮的布局结果自己算一次组高并写回。
	if gHt == xcc.Layout_Size_Auto || gHt == xcc.Layout_Size_Disable {
		desired, prev := int32(0), false
		for i, role := range roles {
			var m xc.RECT
			xc.XWidget_LayoutItem_GetMargin(g.GetChildByIndex(int32(i)), &m)
			mt := int32(0)
			if g.vertical && prev {
				mt = -pressShift
			}
			if g.vertical {
				desired += heights[i] + mt + m.Bottom
				if i > 0 {
					desired += space
				}
			} else if v := heights[i] + mt + m.Bottom; v > desired {
				desired = v
			}
			prev = role == uiRoleButton
		}
		if desired > 0 && g.GetHeight() != desired {
			g.SetHeight(desired)
			changed = true
		}
	}
	if changed {
		// 子项尺寸/间距变了以后, 组的 Auto 宽高不会自己重算(实测: 文本块从 32
		// 压到 28 后, sm 组的 Auto 高仍停在 32; 嵌套组改间距后宽仍旧)。重新声明一次
		// Auto 并交给父级 AdjustLayout, 组的外框才会按新的子项尺寸量一遍。
		var wt, ht xcc.Layout_Size_
		var wv, hv int32
		xc.XWidget_LayoutItem_GetWidth(g.Handle, &wt, &wv)
		xc.XWidget_LayoutItem_GetHeight(g.Handle, &ht, &hv)
		if wt == xcc.Layout_Size_Auto || wt == xcc.Layout_Size_Disable {
			xc.XWidget_LayoutItem_SetWidth(g.Handle, xcc.Layout_Size_Auto, 0)
		}
		if ht == xcc.Layout_Size_Auto || ht == xcc.Layout_Size_Disable {
			xc.XWidget_LayoutItem_SetHeight(g.Handle, xcc.Layout_Size_Auto, 0)
		}
	}
	return changed
}

// layoutHeightType 返回布局项的高度类型。取高度值需要非空指针, 这里只关心类型。
func layoutHeightType(hEle int) xcc.Layout_Size_ {
	var t xcc.Layout_Size_
	var v int32
	xc.XWidget_LayoutItem_GetHeight(hEle, &t, &v)
	return t
}

// setLayoutWidth 只在值与当前布局项不一致时写入, 返回是否改动。
func setLayoutWidth(hEle int, w int32) bool {
	var t xcc.Layout_Size_
	var v int32
	xc.XWidget_LayoutItem_GetWidth(hEle, &t, &v)
	if t == xcc.Layout_Size_Fixed && v == w {
		return false
	}
	xc.XWidget_LayoutItem_SetWidth(hEle, xcc.Layout_Size_Fixed, w)
	return true
}

// setLayoutHeight 只在值与当前布局项不一致时写入, 返回是否改动。
func setLayoutHeight(hEle int, h int32) bool {
	var t xcc.Layout_Size_
	var v int32
	xc.XWidget_LayoutItem_GetHeight(hEle, &t, &v)
	if t == xcc.Layout_Size_Fixed && v == h {
		return false
	}
	xc.XWidget_LayoutItem_SetHeight(hEle, xcc.Layout_Size_Fixed, h)
	return true
}

// ===== ButtonGroupText =====

// ButtonGroupText 按钮组里的静态文本块(shadcn 的 ButtonGroupText):
// bg-muted + border + rounded-md + 左右 16px; 高度跟随组内按钮, 只保留组首尾外角。
type ButtonGroupText struct {
	*widget.Element
	text string
}

// NewButtonGroupText 创建文本块。宽度按文字量出, 高度由按钮组拉伸。
func NewButtonGroupText(text string, hParent int) *ButtonGroupText {
	t := xui.Current()
	f := xui.FontHandle(t.FontDefault)
	w := xui.TextWidth(text, f) + 32 // px-4
	if w < 32 {
		w = 32
	}
	// 初始按默认按钮高; 横向组里若按钮更矮(sm/xs), 组尺寸事件会按按钮视觉高把它压下去。
	h := buttonSpecs[ButtonSizeDefault].height
	ele := widget.NewElement(0, 0, w, h, hParent)
	if ele == nil {
		return nil
	}
	b := &ButtonGroupText{Element: ele, text: text}
	ele.SetUserData(uiRoleGroupText)
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(b.onPaint)
	xui.AttachWindow(ele.Handle)
	return b
}

// SetText 改文本并重新量宽。
func (b *ButtonGroupText) SetText(text string) *ButtonGroupText {
	b.text = text
	if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
		w := xui.TextWidth(text, f) + 32
		if w < 32 {
			w = 32
		}
		b.SetWidth(w)
	}
	if p := b.GetParentEle(); p != 0 {
		xc.XEle_AdjustLayout(p)
	}
	b.Redraw(false)
	return b
}

func (b *ButtonGroupText) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	radius := t.RadiusMd() // shadcn 的 ButtonGroupText 是 rounded-md(按钮才是 rounded-lg)
	if gp := buttonGroupPlacementOf(hEle); gp.inGroup {
		p.FillRoundCorners(&p.Rc, t.Muted, gp.corners(radius))
		p.StrokeRoundCorners(&p.Rc, t.Border, gp.corners(radius), 1)
	} else {
		c := xui.UniformCorners(radius)
		p.FillRoundCorners(&p.Rc, t.Muted, c)
		p.StrokeRoundCorners(&p.Rc, t.Border, c, 1)
	}
	if f := xui.FontHandle(t.FontDefault); f != 0 {
		p.Draw.SetFont(f)
	}
	rc := p.Rc
	rc.Top-- // Vcenter 字形偏下 1px(同 Button)
	rc.Bottom--
	p.Text(b.text, &rc, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
	return 0
}

// ===== ButtonGroupSeparator =====

// ButtonGroupSeparator 按钮组里的分隔条(shadcn 的 ButtonGroupSeparator):
// 横向组里是 1px 竖线(高对齐按钮视觉高), 纵向组里是 1px 横线(宽铺满), 颜色用 Input 令牌。
type ButtonGroupSeparator struct {
	*widget.Element
}

// NewButtonGroupSeparator 创建分隔条。父级是纵向按钮组时自动变为横线。
func NewButtonGroupSeparator(hParent int) *ButtonGroupSeparator {
	// 初始 1×1, 组尺寸事件按取向把它拉成竖线(宽 1)或横线(高 1)。
	ele := widget.NewElement(0, 0, 1, 1, hParent)
	if ele == nil {
		return nil
	}
	s := &ButtonGroupSeparator{Element: ele}
	ele.SetUserData(uiRoleGroupSep)
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(s.onPaint)
	xui.AttachWindow(ele.Handle)
	return s
}

func (s *ButtonGroupSeparator) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	// 元素就是 1px 细条, 直接填满: 画线会有一半落在客户区外被裁掉(见 Separator 注释)。
	p.FillRound(&p.Rc, xui.Current().Input, 0)
	return 0
}
