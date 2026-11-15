package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// FieldChoiceCard 是"整卡可选"的字段卡片(shadcn 的 Choice Card: FieldLabel 包裹
// 一个横排 Field)。卡片本身是唯一命中区: 整卡点击 = 聚焦指示器 + 切换/选中;
// 选中态 border-primary + bg-accent; 键盘焦点环由卡片代理。
//
// 用法(指示器在右):
//
//	card := ui.NewFieldChoiceCard(0, 0, 0, 0, parent)
//	xui.ItemFill(card)
//	card.Label("启用通知")
//	card.Description("你可以随时开启或关闭通知。")
//	cb := ui.NewCheckbox(0, 0, 24, 24, "", card.Handle)
//	card.Attach(cb)
//	cb.OnChange(func(on bool) { fmt.Println(on) }) // 仍可继续挂业务回调
//
// 指示器在左则先创建控件再调用 card.Label / card.Description(布局顺序就是调用顺序)。
//
// 坑都在 Attach 里处理了: 指示器与内容文字鼠标穿透(点击统一落在卡片, 事件不冒泡,
// 见 DESIGN 185)、指示器宿主底色跟随卡片选中态(否则元素矩形留下 Card 色方块)、
// 指示器不再画自己的键盘焦点环。
//
// RadioGroup 里被别的卡片换下时 Radio 不会收到 OnChange(false), 仍需在组
// OnChange 里对其余卡片批量 SetActive(false)。
type FieldChoiceCard struct {
	*widget.LayoutEle
	content  *widget.LayoutEle
	control  FieldChoiceCardControl
	active   bool
	focusEle int
}

// FieldChoiceCardControl 是能挂到选择卡片上的指示器控件。只有内置的 Checkbox /
// Radio / Switch 实现了它(接口含未导出方法), 调用方直接把控件传给 Attach 即可。
type FieldChoiceCardControl interface {
	choiceCardHandle() int
	choiceCardToggle()
	choiceCardActive() bool
	choiceCardOnChange(func(bool))
	choiceCardFocusProxy()
	choiceCardBackdrop(func(*xui.Theme) xui.Color)
	choiceCardSize() (int32, int32)
}

// 选择卡片的内距与间距(对齐 demo 里已验证的几何): 左右内距用元素 padding,
// 上下内距用内容列的布局项外边距(见 DESIGN 129)。
const (
	choiceCardPadL int32 = 12
	choiceCardPadR int32 = 8
	choiceCardPadY int32 = 12
	choiceCardGap  int32 = 6
)

// NewFieldChoiceCard 创建选择卡片。宽铺满、高自撑; 卡片自绘底色与描边。
func NewFieldChoiceCard(x, y, cx, cy int32, hParent int) *FieldChoiceCard {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	c := &FieldChoiceCard{LayoutEle: ele}
	// 布局元素默认是"纯排版壳"(鼠标穿透), 要自己当命中区必须显式关掉穿透
	ele.EnableMouseThrough(false)
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(12)
	ele.SetAlignV(xcc.Layout_Align_Top)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ele.SetPadding(choiceCardPadL, 0, choiceCardPadR, 0)
	ele.SetCursor(linkCursor())
	ele.AddEvent_Paint(c.onPaint)
	// 有 Down 才会派发同一次按下的 Up(同 Attachment / Breadcrumb)
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标点出来的焦点不画键盘焦点环
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		c.activate()
		return 0
	})
	xui.AttachWindow(ele.Handle)
	return c
}

// Content 返回卡片内的内容列(Weight 1, Auto 高, gap 6)。首次调用时创建,
// 所以调用顺序决定指示器在左还是在右: 先建指示器 → 内容列在其右侧。
//
// 直接往内容列里放的自定义子项需要自己 EnableMouseThrough(true), 否则点击
// 落在子项上、卡片收不到; 用 Label / Description 创建的文本已处理。
func (c *FieldChoiceCard) Content() *widget.LayoutEle {
	if c == nil {
		return nil
	}
	if c.content == nil {
		c.content = xui.VStack(c.Handle, 1, choiceCardGap)
		xui.ItemMarginTop(c.content, choiceCardPadY)
		xui.ItemMarginBottom(c.content, choiceCardPadY)
		c.content.EnableMouseThrough(true)
	}
	return c.content
}

// Label 在内容列里创建标题(text-sm / font-medium), 并设为鼠标穿透(点文字等同点卡片)。
func (c *FieldChoiceCard) Label(text string) *Label {
	l := NewLabel(0, 0, 0, 20, text, c.Content().Handle).SetFontSize(14).SetMedium()
	l.EnableMouseThrough(true)
	xui.ItemFill(l)
	return l
}

// Description 在内容列里创建说明(text-sm / muted), 并设为鼠标穿透。
func (c *FieldChoiceCard) Description(text string) *Label {
	l := NewLabel(0, 0, 0, 20, text, c.Content().Handle).SetFontSize(14).SetMuted()
	l.EnableMouseThrough(true)
	xui.ItemFill(l)
	return l
}

// Attach 把指示器控件接进卡片: 指示器鼠标穿透、焦点环代理给卡片、宿主底色跟随
// 卡片选中态, 并让卡片点击走控件的切换/选中路径。主动改选会同步卡片选中态。
// 同一张卡片只接一个控件, 重复调用忽略。
func (c *FieldChoiceCard) Attach(ctrl FieldChoiceCardControl) *FieldChoiceCard {
	if c == nil || ctrl == nil || c.control != nil {
		return c
	}
	hEle := ctrl.choiceCardHandle()
	if hEle == 0 {
		return c
	}
	c.control = ctrl
	c.focusEle = hEle
	e := widget.NewElementByHandle(hEle)
	if e != nil {
		e.EnableMouseThrough(true) // 点指示器也落到卡片, 只保留一条点击路径
	}
	ctrl.choiceCardFocusProxy()
	ctrl.choiceCardBackdrop(func(t *xui.Theme) xui.Color {
		if c.active {
			return t.Accent
		}
		return t.Card
	})
	ctrl.choiceCardOnChange(func(active bool) { c.SetActive(active) })
	if e != nil {
		if w, h := ctrl.choiceCardSize(); w > 0 && h > 0 {
			xui.ItemSize(e, w, h)
		}
		xui.ItemMarginTop(e, choiceCardPadY)
	}
	if btn := widget.NewButtonByHandle(hEle); btn != nil {
		btn.AddEvent_SetFocus(func(h int, pbHandled *bool) int {
			c.redraw()
			return 0
		})
		btn.AddEvent_KillFocus(func(h int, pbHandled *bool) int {
			c.redraw()
			return 0
		})
	}
	c.SetActive(ctrl.choiceCardActive())
	return c
}

// activate 是整卡点击的落点: 聚焦指示器(禁用时不动作)后走与点指示器相同的切换路径。
func (c *FieldChoiceCard) activate() {
	if c == nil || c.control == nil {
		return
	}
	if focusBoundControl(c.control.choiceCardHandle()) {
		c.control.choiceCardToggle()
	}
}

// SetActive 设置选中态(边框 Primary + 底色 Accent)。
func (c *FieldChoiceCard) SetActive(v bool) *FieldChoiceCard {
	if c == nil || c.active == v {
		return c
	}
	c.active = v
	c.Redraw(false)
	return c
}

// Active 报告当前是否处于选中态。
func (c *FieldChoiceCard) Active() bool { return c != nil && c.active }

func (c *FieldChoiceCard) redraw() {
	if c != nil {
		c.Redraw(false)
	}
}

func (c *FieldChoiceCard) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg, border := t.Card, t.Border
	if c.active {
		bg, border = t.Accent, t.Primary
	}
	// 卡片是透明的自绘元素, 整块铺底既擦上一帧也给出选中底色; 描边内缩 1px 免得被裁
	radius := t.RadiusLg()
	p.FillRound(&p.Rc, bg, radius)
	rc := p.Rc
	rc.Left++
	rc.Top++
	rc.Right--
	rc.Bottom--
	p.StrokeRound(&rc, border, radius-1, 1)
	if c.focusEle != 0 {
		if st := xui.StateOf(c.focusEle); st.Focused && xui.FocusedByKeyboard() {
			fr := p.Rc
			fr.Left += 2
			fr.Top += 2
			fr.Right -= 2
			fr.Bottom -= 2
			p.StrokeRound(&fr, t.Ring.WithAlpha(160), radius-2, 3)
		}
	}
	return 0
}
