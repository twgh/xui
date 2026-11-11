package main

import (
	"fmt"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

func buildPopoverCard(body *widget.LayoutEle) {
	_, vpop := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "弹出层", vpop.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Popover: 点击触发、外部/ESC 关闭; HoverCard: 悬停预览且不抢焦点", vpop.Handle))

	popRow := xui.HStack(vpop.Handle, 36, 12)
	popBtn := ui.NewButton(0, 0, 0, 0, "打开 Popover", popRow.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonHasPopup(),
		ui.WithButtonTrailingIcon(icons.IconChevronDown))
	xui.ItemWidth(popBtn, 140)
	ui.NewPopover(popBtn.Handle, ui.WithPopoverContent(func(c int) {
		h := ui.PopoverHeader(c)
		ui.PopoverTitle(h.Handle, "维度设置")
		ui.PopoverDescription(h.Handle, "调整弹层宽度与内容间距。")
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "宽度 288 · 内边距 16", c).SetMuted().SetFontSize(14))
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "下方优先, 空间不足自动翻转", c).SetMuted().SetFontSize(14))
		btns := xui.HStack(c, 32, 8)
		ui.NewButton(0, 0, 72, 0, "保存", btns.Handle)
		ui.NewButton(0, 0, 72, 0, "取消", btns.Handle, ui.WithVariant(ui.ButtonOutline))
	}))

	hoverBtn := ui.NewButton(0, 0, 0, 0, "悬停 @xui 查看资料", vpop.Handle, ui.WithVariant(ui.ButtonLink))
	xui.ItemWidth(hoverBtn, 160)
	ui.NewHoverCard(hoverBtn.Handle, ui.WithHoverCardContent(func(c int) {
		r := xui.HStack(c, 40, 12)
		ui.NewAvatar(0, 0, 0, 0, r.Handle, "X",
			ui.WithAvatarSize(ui.AvatarSizeLg), ui.WithAvatarImageData(demoAvatar()))
		col := xui.VStackWeight(r.Handle, 1, 2)
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "@xui", col.Handle).SetMedium().SetFontSize(14))
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "shadcn 风格 xcgui 自绘组件库", col.Handle).SetMuted().SetFontSize(14))
		xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "加入于 2026 年 10 月", c).SetMuted().SetFontSize(14))
	}))

}

// buildAvatarCard 演示头像的尺寸 / 图片 / 徽标 / 叠放组。

// ===== 页 3: 基础3(弹出层 / 轮播 / 分栏 / 卡片 / 比例容器) =====

func buildBase3Page(body *widget.LayoutEle) {
	buildPopoverCard(body)

	// 轮播也放在页首: 交互密集的卡片长页滚到底后命中区会滞后(DESIGN 118)。
	buildCarouselCard(body.Handle)

	// 分栏同样放在页首(拖动 + 键盘焦点)。
	buildResizableCard(body.Handle)

	// ---- 卡片本身(直接以真卡片呈现, 不套外壳) ----
	cardDemo, vd := autoCard(body.Handle, 10)
	// 卡片头: 左列标题+说明, 右列动作区(CardAction) —— 动作贴右上、跨标题与说明两行。
	ch := ui.NewCardHeader(0, 0, 0, vd.Handle, 4)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "卡片", ch.TitleStack().Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "圆角面板 + ring 描边; 内容区随卡片尺寸自适应, 页脚贴底且只有下两角圆角", ch.TitleStack().Handle))
	cardAct := ui.NewButton(0, 0, 0, 0, "", ch.ActionStack().Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconEllipsis))
	tip.Bind(cardAct.Handle, "CardAction")
	rd := xui.HStack(vd.Handle, 36, 12)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "了解详情", rd.Handle, ui.WithVariant(ui.ButtonOutline)), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "开始使用", rd.Handle), 104)

	footer := cardDemo.Footer(0)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0,
		"CardFooter · p-4 + items-center + border-t + bg-muted/50, 只随卡片下两角圆角", footer.Content().Handle))

	// ---- 比例容器(AspectRatio) + sm 卡片 ----
	rr := row(body.Handle, 0, gap)
	_, vr := autoCard(xui.VStack(rr.Handle, 2, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "比例容器", vr.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"AspectRatio: 宽度由布局/布局项决定, 高度 = 宽度 / ratio; 内容放 Content() 里铺满比例框", vr.Handle))
	ritems := row(vr.Handle, 0, 16)
	ratioPreview(ritems.Handle, 2, 16.0/9.0, "16 : 9")
	ratioPreview(ritems.Handle, 1, 1, "1 : 1")
	ratioPreview(ritems.Handle, 1, 9.0/16.0, "9 : 16")

	// sm 卡片: 内边距与页脚都是 p-3(12), 标题 text-sm(14)
	smCard := ui.NewCard(0, 0, 0, 0, xui.VStack(rr.Handle, 1, gap).Handle, ui.WithCardSize(ui.CardSizeSm))
	xui.ItemFill(smCard)
	smCard.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	vsm := smCard.ContentStack(12)
	chsm := ui.NewCardHeader(0, 0, 0, vsm.Handle, 4)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "小卡片", chsm.TitleStack().Handle).SetFontSize(smCard.TitleSize()))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "size=sm: 内边距与页脚都是 12, 标题 14", chsm.TitleStack().Handle))
	xui.ItemFill(ui.NewButton(0, 0, 0, 0, "操作", vsm.Handle, ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline)))
	smFooter := smCard.Footer(0)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "CardFooter · p-3", smFooter.Content().Handle).SetMuted())
}

// buildCarouselCard 建一张轮播演示卡: 整页滑动 + 循环 + 指示点。
// 幻灯片背景与文字直接用主题令牌自绘, 演示"幻灯片里放任意自绘内容"的用法。
func buildCarouselCard(parent int) {
	_, vc := autoCard(parent, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "轮播", vc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Carousel: 整页滑动(300ms) · 循环 · ←/→ 键盘翻页 · 拖拽松手吸附; 上/下张按钮在内容外侧 48px", vc.Handle))

	car := ui.NewCarousel(0, 0, 0, 200, vc.Handle, ui.WithCarouselLoop(true))
	xui.ItemFill(car)
	car.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 200)

	captions := [...]string{
		"循环开启: 最后一张的下一页接着第一张",
		"用鼠标拖动后松手, 吸附到最近一页",
		"← / → 键在轮播内控件获焦时翻页",
		"按钮的悬停与焦点样式与 ui.Button 一致",
	}
	n := len(captions)
	for i := 0; i < n; i++ {
		label, caption := fmt.Sprintf("%d", i+1), captions[i]
		car.AddSlide(func(s *ui.CarouselSlide) {
			ph := widget.NewElement(0, 0, 0, 0, s.Handle)
			xui.ItemFill(ph)
			ph.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
			ph.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
				p := xui.BeginPaint(hEle, hDraw)
				if p == nil {
					return 0
				}
				*pbHandled = true
				t := xui.Current()
				p.FillRound(&p.Rc, t.Muted, t.RadiusLg())
				p.Draw.SetFont(xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 40, Bold: true}))
				numRc := xc.RECT{Left: p.Rc.Left, Top: p.Rc.Top, Right: p.Rc.Right, Bottom: p.Rc.Bottom - 20}
				p.Text(label, &numRc, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
				p.Draw.SetFont(xui.FontHandle(t.FontDefault))
				capRc := xc.RECT{Left: p.Rc.Left + 16, Top: p.Rc.Bottom - 44, Right: p.Rc.Right - 16, Bottom: p.Rc.Bottom - 16}
				p.Text(caption, &capRc, t.MutedForeground,
					xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
				return 0
			})
		})
	}

	// 指示点: 当前页实心, 其余用 Border 色; 页变化时重绘。
	dots := widget.NewElement(0, 0, 0, 10, vc.Handle)
	xui.ItemFill(dots)
	dots.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 10)
	dots.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		const d, gap int32 = 6, 6
		total := int32(n)*d + int32(n-1)*gap
		x0 := (p.Rc.Left + p.Rc.Right - total) / 2
		y := (p.Rc.Top + p.Rc.Bottom - d) / 2
		for i := 0; i < n; i++ {
			col := t.Border
			if i == car.Index() {
				col = t.Foreground
			}
			l := x0 + int32(i)*(d+gap)
			rc := xc.RECT{Left: l, Top: y, Right: l + d, Bottom: y + d}
			p.FillRound(&rc, col, d/2)
		}
		return 0
	})
	car.OnChange(func(i int) { dots.Redraw(true) })
}

// buildResizableCard 建一张分栏演示卡: 横向拖动 + 右侧嵌套纵向分组。
func buildResizableCard(parent int) {
	_, vr := autoCard(parent, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "可调整分栏", vr.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Resizable: 拖动分隔条伸缩相邻面板 · 点击分隔条后 ←/→ 每次 5% · Home/End 到边界 · 双击复位", vr.Handle))

	box := borderedBox(vr.Handle, 240, false)
	g := ui.NewResizableGroup(0, 0, 0, 0, box.Handle)
	xui.ItemFill(g)
	xui.ItemHeight(g, 240)
	left := g.AddPanel(ui.WithResizableSize(35), ui.WithResizableMin(20))
	leftPane := newResizablePreview(left.Handle, "左 35%", "最小 20% · 拖一拖分隔条")
	g.AddHandle(ui.WithResizableGrip())
	right := g.AddPanel(ui.WithResizableSize(65))

	vg := ui.NewResizableGroup(0, 0, 0, 0, right.Handle, ui.WithResizableVertical())
	xui.ItemFill(vg)
	xui.ItemFillHeight(vg)
	top := vg.AddPanel(ui.WithResizableSize(40))
	topPane := newResizablePreview(top.Handle, "上 40%", "嵌套纵向分组")
	vg.AddHandle()
	bottom := vg.AddPanel(ui.WithResizableSize(60))
	bottomPane := newResizablePreview(bottom.Handle, "下 60%", "↑ / ↓ 调整")

	// 百分比文案跟着调整实时更新(自绘文字不重绘就只是静态数字)。
	g.OnResize(func(s []float64) {
		leftPane.SetText(fmt.Sprintf("左 %.0f%%", s[0]), "最小 20% · 拖一拖分隔条")
	})
	vg.OnResize(func(s []float64) {
		topPane.SetText(fmt.Sprintf("上 %.0f%%", s[0]), "嵌套纵向分组")
		bottomPane.SetText(fmt.Sprintf("下 %.0f%%", s[1]), "↑ / ↓ 调整")
	})
}

// resizablePreview 是面板里的演示内容: 圆角弱化底色 + 居中标题/说明,
// 标题与说明可运行时更新(跟随拖动/键盘调整后的百分比)。
type resizablePreview struct {
	*widget.Element
	title   string
	caption string
}

func newResizablePreview(parent int, title, caption string) *resizablePreview {
	v := &resizablePreview{title: title, caption: caption}
	ph := widget.NewElement(0, 0, 0, 0, parent)
	xui.ItemFill(ph)
	ph.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	ph.EnableBkTransparent(true)
	ph.EnableDrawBorder(false)
	ph.EnableFocus(false)
	ph.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		rc := p.Rc
		rc.Left += 6
		rc.Top += 6
		rc.Right -= 6
		rc.Bottom -= 6
		if rc.Right <= rc.Left || rc.Bottom <= rc.Top {
			return 0
		}
		t := xui.Current()
		p.FillRound(&rc, t.Muted, t.RadiusMd())
		mid := rc.Top + (rc.Bottom-rc.Top)/2
		titleRc := xc.RECT{Left: rc.Left, Top: rc.Top, Right: rc.Right, Bottom: mid}
		p.Draw.SetFont(xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 14}))
		p.Text(v.title, &titleRc, t.Foreground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		captionRc := xc.RECT{Left: rc.Left, Top: mid, Right: rc.Right, Bottom: rc.Bottom}
		p.Draw.SetFont(xui.FontHandle(xui.FontSpec{Family: t.FontDefault.Family, Size: 12}))
		p.Text(v.caption, &captionRc, t.MutedForeground, xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter)
		return 0
	})
	v.Element = ph
	return v
}

func (v *resizablePreview) SetText(title, caption string) {
	if v == nil || (v.title == title && v.caption == caption) {
		return
	}
	v.title, v.caption = title, caption
	v.Redraw(false)
}
