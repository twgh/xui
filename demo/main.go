// xui 组件演示: light/dark 一键换肤 + 全部组件(含表格)。
// 卡片内容用 LayoutEle 流式布局, 不再手算 y。
// 顶层用 Tabs 分页展示(表单/控件/反馈/数据), 主题用环境变量 XUI_THEME=dark 切换。
// 控制启动首页可用环境变量 XUI_HOME_PAGE = 0/1/2/3 分别对应 表单/控件/反馈/数据。
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/objectbase"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

var (
	tip       *ui.Tooltip
	demoTable *ui.Table
)

func main() {
	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)

	w := window.New(0, 0, 800, 590, "xui · shadcn 风格自绘组件", 0,
		xcc.Window_Style_Center|xcc.Window_Style_Border|xcc.Window_Style_Drag_Border|xcc.Window_Style_Drag_Window|xcc.Window_Style_Allow_MaxWindow)
	w.SetTransparentType(xcc.Window_Transparent_Shadow)
	w.SetShadowInfo(12, 128, 8, false, xc.RGBA(0, 0, 0, 128))
	w.SetTransparentAlpha(255)
	w.SetBorderSize(0, 0, 0, 0)
	w.SetMinimumSize(800, 590)
	w.SetFont(xui.FontHandle(xui.Current().FontDefault))

	xui.RegisterWindow(w.Handle)
	xui.BindWindowBackground(w.Handle) // 窗口底色随主题切换
	tip = ui.NewTooltip()

	// ===== 布局骨架: 纵向 [标题栏行 30][内容区 Fill] =====
	w.EnableLayout(true)
	root := widget.NewLayoutEle(0, 0, 0, 0, w.Handle)
	root.EnableHorizon(false)
	root.EnableBkTransparent(true)
	root.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	root.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)

	// 标题栏行: logo/标题居左, 按钮组浮到最右(中间标题 Fill 撑开)。
	// 行自身鼠标穿透 —— 不挡 Drag_Window 的拖动命中, 子元素(按钮)不受影响。
	cap := widget.NewLayoutEle(0, 0, 0, 30, root.Handle)
	cap.EnableHorizon(true)
	cap.EnableBkTransparent(true)
	cap.SetSpace(6)
	cap.SetAlignV(xcc.Layout_Align_Center)
	cap.EnableMouseThrough(true)
	cap.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	cap.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 30)

	// logo: Primary 圆角块 + 白色 "x", 绘制时取令牌随主题换色
	logo := widget.NewLayoutEle(0, 0, 18, 18, cap.Handle)
	logo.EnableBkTransparent(true)
	logo.AddEvent_Paint(func(hEle, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		p.FillRound(&p.Rc, t.Primary, 5)
		if f := xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size,
			Bold:   true,
		}); f != 0 {
			p.Draw.SetFont(f)
		}
		p.TextCenter("x", t.PrimaryFore)
		return 0
	})
	logo.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 18)
	logo.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 18)

	// 标题文字: Weight 宽度占满剩余空间把按钮组顶到行尾
	// (横向行里主轴 Fill 会独占整行把兄弟挤到下一行, 剩余空间必须用 Weight)
	title := widget.NewLayoutEle(0, 0, 0, 30, cap.Handle)
	title.EnableHorizon(false)
	title.EnableBkTransparent(true)
	title.EnableMouseThrough(true)
	title.AddEvent_Paint(func(hEle, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		p.Text("xui · shadcn 风格自绘组件", &p.Rc, xui.Current().Foreground,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		return 0
	})
	title.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	title.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)

	// 控制按钮(幽灵按钮, hover 才有底色): 关闭贴右缘, 最小化在其左, 主题按钮再往左
	const (
		capBtnW int32 = 36
		capBtnH int32 = 28
	)

	// 启动主题: 环境变量 XUI_THEME=dark 暗色, 其它/不设为亮色, 免去改代码重建。
	// 运行中仍可用右上角按钮一键切换。
	dark := strings.EqualFold(os.Getenv("XUI_THEME"), "dark")
	if dark {
		xui.SetTheme(xui.Dark())
	}
	const toggleW int32 = 60

	showLayoutFrame := false
	// 显示布局对象边界按钮
	btnShowLayoutFrame := ui.NewButton(0, 0, toggleW, capBtnH, "显", cap.Handle,
		ui.WithVariant(ui.ButtonOutline))
	tip.Bind(btnShowLayoutFrame.Handle, "显示/隐藏布局对象边界")
	itemFixed(btnShowLayoutFrame, toggleW, capBtnH)
	btnShowLayoutFrame.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		showLayoutFrame = !showLayoutFrame
		app.ShowLayoutFrame(showLayoutFrame) // 显示布局对象边界
		btnShowLayoutFrame.SetText(common.Choose(showLayoutFrame, "隐", "显"))
		w.Redraw(false)
		return 0
	})

	// 主题切换按钮
	toggle := ui.NewButton(0, 0, toggleW, capBtnH, "Dark", cap.Handle,
		ui.WithVariant(ui.ButtonOutline))
	tip.Bind(toggle.Handle, "切换主题")
	itemFixed(toggle, toggleW, capBtnH)
	if dark {
		toggle.SetText("Light")
	}
	toggle.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		dark = !dark
		toggle.SetText(common.Choose(dark, "Light", "Dark"))
		xui.SetTheme(common.Choose(dark, xui.Dark(), xui.Light()))
		// 切换主题会重绘窗口, 所以 toggle 不用单独重绘
		return 0
	})

	// 控制按钮
	btnMin := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(xui.IconMinus))
	btnMin.SetTypeEx(xcc.Button_Type_Min)
	itemFixed(btnMin, capBtnW, capBtnH)

	btnClose := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(xui.IconX))
	btnClose.SetTypeEx(xcc.Button_Type_Close)
	itemFixed(btnClose, capBtnW, capBtnH)

	// 内容区: 页签 + 页面容器都绝对定位在里面(见 buildPage)。
	// 必须用 LayoutEle 才能作为布局项被 Fill 排版(普通 Element 布局不排, 实测空白);
	// 但不要 EnableLayout —— 关闭排版后它自己的子元素保持绝对定位。
	content := widget.NewLayoutEle(0, 0, 0, 0, root.Handle)
	content.EnableHorizon(false)
	content.EnableLayout(false) // 子元素(tabs/pages)保持绝对定位
	content.EnableBkTransparent(true)
	content.EnableMouseThrough(true) // 纯容器, 空白处透给窗口拖动
	content.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	content.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)

	buildPage(w.Handle, content.Handle)

	// 双击标题栏行最大化/还原
	w.AddEvent_LButtonDBClick(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		if pPt.Y > 30 {
			return 0
		}
		if w.IsMaxWindow() {
			w.ShowWindow(xcc.SW_RESTORE)
		} else {
			w.MaxWindow()
		}
		return 0
	})

	w.AdjustLayout()
	w.Show(true)
	a.Run()
	a.Exit()
}

// ===== 布局助手 =====
// LayoutEle 实测语义(xcgui 1.4.2, 见 DESIGN.md 第八节 29):
//   - EnableHorizon(false)=纵向列布局, 主轴间距用 SetSpace, 交叉轴对齐用 SetAlignH;
//   - EnableHorizon(true)=横向行流+自动换行, 行内间距用 SetSpace, 换行行距用 SetSpaceRow,
//     交叉轴(垂直)对齐用 SetAlignV;
//   - Fill = 独占一整行/列(兄弟被挤到下一行), 等分剩余空间用 Weight;
//   - 项级 XWidget_LayoutItem_SetAlign 无效, 用布局级 SetAlignV/H 代替。
// 布局项默认尺寸类型不可靠, 所有项都显式声明:
// 宽度要么 Fill 占满要么 Fixed, 高度一律 Fixed 锁定。

// vStack 卡片内容区纵向流式布局。
func vStack(card *ui.Card, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(card.ContentLeft(), card.ContentTop(),
		card.ContentWidth(), card.GetHeight()-card.Padding*2, card.Handle)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.SetSpace(space)
	// 关掉"纵向溢出换列": LayoutEle 在主轴放不下下一项时会把它甩到下一列
	// (x += 容器宽度, y = 0), 表现为控件整块消失(实测, 见 DESIGN 第八节 41)
	l.EnableAutoWrap(false)
	return l
}

// hRow 横向子布局: 宽度占满父布局, 高度 ht, 项间距 space, 子项垂直居中。
func hRow(parent int, ht, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, ht, parent)
	l.EnableHorizon(true)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.SetSpace(space)
	l.SetAlignV(xcc.Layout_Align_Center)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ht)
	return l
}

// vCol 纵向子布局: 等分横向父布局的剩余宽度(Weight=1), 高度占满。
func vCol(parent int, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.SetSpace(space)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	return l
}

// layoutItem 所有 widget 都提升自 objectbase.Widget, 自带 LayoutItem_* 包装。
type layoutItem interface {
	LayoutItem_SetWidth(t xcc.Layout_Size_, w int32) *objectbase.Widget
	LayoutItem_SetHeight(t xcc.Layout_Size_, h int32) *objectbase.Widget
	LayoutItem_SetMargin(left, top, right, bottom int32) *objectbase.Widget
}

// itemFixed 锁定布局项为固定尺寸。
func itemFixed(ele layoutItem, w, ht int32) {
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, w)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ht)
}

// itemFill 布局项宽度占满剩余空间(列布局中独占一行), 高度锁定为 ht。
func itemFill(ele layoutItem, ht int32) {
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ht)
}

// itemMarginTop 顶部额外间距。
func itemMarginTop(ele layoutItem, m int32) {
	ele.LayoutItem_SetMargin(0, m, 0, 0)
}

// ===== 页面 =====

// buildPage 演示页: 顶层 Tabs 分页(表单/控件/反馈/数据), 每页是一个绝对定位的
// 容器元素, 切换时 Show/Hide + 重绘。页面内容静态创建一次即可
// (主题切换走整窗重绘, 不需要重建)。
// 后续新增组件展示时: 开一个新页面容器 + 对应 Tab 即可, 页面之间互不挤压。
func buildPage(hWindow, hParent int) {
	pageCount := 4
	homePage, _ := strconv.Atoi(os.Getenv("XUI_HOME_PAGE"))
	if homePage < 0 || homePage > pageCount-1 {
		homePage = 0
	}

	pages := make([]*widget.LayoutEle, pageCount)
	for i := range pages {
		// hParent 是内容区元素(布局项, 客户区宽 784); 页面在其中绝对定位互叠
		pages[i] = widget.NewLayoutEle(0, 44, 784, 500, hParent)
		pages[i].EnableBkTransparent(true).EnableDrawBorder(false).EnableFocus(false)
		pages[i].EnableLayout(false)      // 禁用布局, 因为内部用的绝对坐标
		pages[i].EnableMouseThrough(true) // 纯容器, 空白处透给窗口拖动
		pages[i].Show(i == homePage)
	}
	pForm, pCtrl, pFeed, pData := pages[0], pages[1], pages[2], pages[3]

	pageTabs := ui.NewTabs(16, 6, 340, hParent, []string{"表单", "控件", "反馈", "数据"})
	pageTabs.OnChange(func(i int32) {
		for j, p := range pages {
			vis := int(i) == j
			p.Show(vis)
			// 布局元素在父容器隐藏期间不排版, 恢复可见后必须手动重排,
			// 否则页面内容空白(实测只有卡片壳)
			if vis {
				p.AdjustLayout()
				p.Redraw(false)
			}
		}
	})
	pageTabs.SetActive(int32(homePage))

	// ===== 页 1: 表单类 =====
	// ===== 左卡: 表单类 =====
	card1 := ui.NewCard(24, 0, 340, 470, pForm.Handle)
	// 结构对齐 shadcn App.tsx 的 Account 卡: CardContent space-y-4(16) 分组,
	// 组内 CardHeader/space-y-2 是 8 —— 用 SetSpace(8) 打底 + itemMarginTop 补到组间距
	v1 := vStack(card1, 8)

	itemFill(ui.NewCardTitle(0, 0, 0, "Account", v1.Handle), 24)
	itemFill(ui.NewLabel(0, 0, 0, 0, "修改你的账户信息, 保存后立即生效。", v1.Handle).SetMuted(), 24)

	lblEmail := ui.NewLabel(0, 0, 0, 0, "邮箱地址", v1.Handle)
	itemFill(lblEmail, 24)
	itemMarginTop(lblEmail, 8) // 组边界: 8 + 8 = 16
	email := ui.NewInput(0, 0, 0, 0, v1.Handle, ui.WithPlaceholder("m@example.com"))
	itemFill(email, email.GetHeight())

	lblBio := ui.NewLabel(0, 0, 0, 0, "个人简介", v1.Handle)
	itemFill(lblBio, 24)
	itemMarginTop(lblBio, 8)
	bio := ui.NewInput(0, 0, 0, 72, v1.Handle,
		ui.WithPlaceholder("介绍一下自己..."), ui.WithMultiline())
	itemFill(bio, bio.GetHeight())

	c1 := ui.NewCheckbox(0, 0, 0, 0, "接收邮件通知", v1.Handle)
	c1.SetChecked(true)
	itemFill(c1, c1.GetHeight())
	itemMarginTop(c1, 8)
	cb2 := ui.NewCheckbox(0, 0, 0, 0, "公开展示个人主页", v1.Handle)
	itemFill(cb2, cb2.GetHeight())
	itemMarginTop(cb2, 4) // 开关组内 gap-3(12)

	rowFormBtns := hRow(v1.Handle, 36, 8)
	itemMarginTop(rowFormBtns, 16) // 组间距 16 + pt-2(8)
	saveBtn := ui.NewButton(0, 0, 0, 0, "保存", rowFormBtns.Handle)
	itemFixed(saveBtn, 96, saveBtn.GetHeight())
	cancelBtn := ui.NewButton(0, 0, 0, 0, "取消", rowFormBtns.Handle, ui.WithVariant(ui.ButtonOutline))
	itemFixed(cancelBtn, 96, cancelBtn.GetHeight())

	// ===== 右卡: 展示类 =====
	card2 := ui.NewCard(392, 0, 380, 470, pForm.Handle)
	// shadcn App.tsx 的 Components 卡: CardContent space-y-5(20) 分组,
	// 组内的按钮行/徽章行各自有 hRow 间距
	v2 := vStack(card2, 20)

	itemFill(ui.NewCardTitle(0, 0, 0, "Components", v2.Handle), 24)

	rowBtns1 := hRow(v2.Handle, 36, 8)
	b := ui.NewButton(0, 0, 0, 0, "Primary", rowBtns1.Handle)
	itemFixed(b, 104, b.GetHeight())
	b = ui.NewButton(0, 0, 0, 0, "Secondary", rowBtns1.Handle, ui.WithVariant(ui.ButtonSecondary))
	itemFixed(b, 112, b.GetHeight())
	b = ui.NewButton(0, 0, 0, 0, "Outline", rowBtns1.Handle, ui.WithVariant(ui.ButtonOutline))
	itemFixed(b, 104, b.GetHeight())

	rowBtns2 := hRow(v2.Handle, 36, 8)
	b = ui.NewButton(0, 0, 0, 0, "Ghost", rowBtns2.Handle, ui.WithVariant(ui.ButtonGhost))
	itemFixed(b, 104, b.GetHeight())
	b = ui.NewButton(0, 0, 0, 0, "Destructive", rowBtns2.Handle, ui.WithVariant(ui.ButtonDestructive))
	itemFixed(b, 112, b.GetHeight())
	b = ui.NewButton(0, 0, 0, 0, "Disabled", rowBtns2.Handle)
	itemFixed(b, 104, b.GetHeight())
	b.Enable(false)

	sep := ui.NewSeparator(0, 0, 0, v2.Handle)
	itemFill(sep, sep.GetHeight())
	itemMarginTop(sep, 12)

	rowBadges := hRow(v2.Handle, 20, 8)
	badge := ui.NewBadge(0, 0, 64, 0, "Default", rowBadges.Handle)
	itemFixed(badge, 64, badge.GetHeight())
	badge = ui.NewBadge(0, 0, 0, 0, "Secondary", rowBadges.Handle, ui.WithBadgeVariant(ui.BadgeSecondary))
	itemFixed(badge, 78, badge.GetHeight())
	badge = ui.NewBadge(0, 0, 0, 0, "Outline", rowBadges.Handle, ui.WithBadgeVariant(ui.BadgeOutline))
	itemFixed(badge, 72, badge.GetHeight())
	badge = ui.NewBadge(0, 0, 0, 0, "Destructive", rowBadges.Handle, ui.WithBadgeVariant(ui.BadgeDestructive))
	itemFixed(badge, 88, badge.GetHeight())

	lb := ui.NewLabel(0, 0, 0, 0, "通知偏好", v2.Handle).SetMuted()
	itemFill(lb, lb.GetHeight())
	itemMarginTop(lb, 12)

	r1 := ui.NewRadio(0, 0, 0, 0, "仅错误消息", v2.Handle)
	r1.SetCheck(true)
	itemFill(r1, r1.GetHeight())
	r2 := ui.NewRadio(0, 0, 0, 0, "所有消息", v2.Handle)
	itemFill(r2, r2.GetHeight())
	r3 := ui.NewRadio(0, 0, 0, 0, "不接收", v2.Handle)
	itemFill(r3, r3.GetHeight())
	r3.Enable(false)

	// ===== 页 2: 控件 =====
	card3 := ui.NewCard(24, 0, 728, 290, pCtrl.Handle)
	v3 := vStack(card3, 14)

	// 第一组: 开关 | 滑动条
	g1 := hRow(v3.Handle, 56, 24)

	colSw := vCol(g1.Handle, 8)
	itemFill(ui.NewLabel(0, 0, 0, 0, "开关", colSw.Handle).SetMuted(), 24)
	rowSw := hRow(colSw.Handle, 24, 8)
	sw := ui.NewSwitch(0, 0, rowSw.Handle, ui.WithSwitchChecked(true))
	itemFixed(sw, sw.GetWidth(), sw.GetHeight())
	sw = ui.NewSwitch(0, 0, rowSw.Handle)
	itemFixed(sw, sw.GetWidth(), sw.GetHeight())
	sw = ui.NewSwitch(0, 0, rowSw.Handle, ui.WithSwitchSize(ui.SwitchSm))
	itemFixed(sw, sw.GetWidth(), sw.GetHeight())
	sw = ui.NewSwitch(0, 0, rowSw.Handle, ui.WithSwitchSize(ui.SwitchSm))
	itemFixed(sw, sw.GetWidth(), sw.GetHeight())
	sw.Enable(false)

	colSl := vCol(g1.Handle, 8)
	itemFill(ui.NewLabel(0, 0, 0, 0, "滑动条", colSl.Handle).SetMuted(), 24)
	// Slider 元素比可见轨道左右各宽 9px(ring 绘制空间), cx=270 → 元素宽 288
	rowSl := hRow(colSl.Handle, 24, 8)
	sl := ui.NewSlider(0, 0, 270, 0, rowSl.Handle)
	itemFixed(sl, 288, sl.GetHeight())
	sliderValue := ui.NewLabel(0, 0, 40, 24, "40", rowSl.Handle).SetMuted()
	itemFixed(sliderValue, 40, 24)
	sl.SetValue(40)
	sl.OnChange(func(v float64) { sliderValue.SetText(strconv.Itoa(int(v + 0.5))).Redraw() })

	// 第二组: 进度 | 标签页 | 下拉框
	g2 := hRow(v3.Handle, 136, 24)

	colPg := vCol(g2.Handle, 8)
	itemFill(ui.NewLabel(0, 0, 0, 0, "进度", colPg.Handle).SetMuted(), 24)
	pg := ui.NewProgress(0, 0, 0, 0, colPg.Handle)
	itemFill(pg, pg.GetHeight())
	itemMarginTop(pg, 6)
	pg.SetValue(64)
	pg2 := ui.NewProgress(0, 0, 0, 0, colPg.Handle)
	itemFill(pg2, pg2.GetHeight())
	itemMarginTop(pg2, 12)
	pg2.SetValue(100)

	colTabs := vCol(g2.Handle, 8)
	itemFill(ui.NewLabel(0, 0, 0, 0, "标签页", colTabs.Handle).SetMuted(), 24)
	// Tabs 内部标签宽度按创建时的 cx 等分, 不能被布局拉伸, 用固定宽
	tabs := ui.NewTabs(0, 0, 180, colTabs.Handle, []string{"账户", "密码", "通知"})
	itemFixed(tabs, 180, tabs.GetHeight())
	panel := ui.NewLabel(0, 0, 0, 22, "账户设置面板", colTabs.Handle).SetMuted()
	itemFill(panel, panel.GetHeight())
	tabs.OnChange(func(i int32) {
		panel.SetText([]string{"账户设置面板", "密码设置面板", "通知设置面板"}[i]).Redraw()
	})
	lineTabs := ui.NewTabs(0, 0, 180, colTabs.Handle, []string{"概览", "详情"},
		ui.WithTabsVariant(ui.TabsLine), ui.WithTabsActive(1))
	itemFixed(lineTabs, 180, lineTabs.GetHeight())

	colCombo := vCol(g2.Handle, 8)
	itemFill(ui.NewLabel(0, 0, 0, 0, "下拉框", colCombo.Handle).SetMuted(), 24)
	combo := ui.NewComboBox(0, 0, 180, 0, colCombo.Handle,
		[]string{"开发环境", "测试环境", "预发环境", "灰度环境", "生产环境", "本地调试"},
		ui.WithComboPlaceholder("选择环境"))
	itemFixed(combo, 180, combo.GetHeight())
	combo.SetSelected(1)

	// 提示气泡: 绑到"保存"按钮与下拉框上
	tip.Bind(saveBtn.Handle, "保存当前修改(Ctrl+S)")
	tip.Bind(combo.Handle, "切换运行时环境")

	// ===== 页 3: 反馈(对话框 + 菜单) =====
	card4 := ui.NewCard(24, 0, 728, 108, pFeed.Handle)
	v4 := vStack(card4, 14)
	itemFill(ui.NewLabel(0, 0, 0, 0, "对话框", v4.Handle).SetMuted(), 24)

	rowDlg := hRow(v4.Handle, 36, 8)
	editBtn := ui.NewButton(0, 0, 0, 0, "编辑资料", rowDlg.Handle, ui.WithVariant(ui.ButtonOutline))
	itemFixed(editBtn, 96, editBtn.GetHeight())
	delBtn := ui.NewButton(0, 0, 0, 0, "删除项目", rowDlg.Handle, ui.WithVariant(ui.ButtonDestructive))
	itemFixed(delBtn, 96, delBtn.GetHeight())
	itemFixed(ui.NewLabel(0, 0, 0, 24, "ESC / 点击遮罩 / 右上角 X 关闭, Tab 在对话框内循环", rowDlg.Handle).SetMuted(), 360, 24)

	buildDialogs(hWindow, editBtn, delBtn)

	// 菜单卡
	card5 := ui.NewCard(24, 124, 728, 108, pFeed.Handle)
	v5 := vStack(card5, 14)
	itemFill(ui.NewLabel(0, 0, 0, 0, "下拉菜单", v5.Handle).SetMuted(), 24)

	rowMenu := hRow(v5.Handle, 36, 8)
	menuBtn := ui.NewButton(0, 0, 0, 0, "打开菜单", rowMenu.Handle, ui.WithVariant(ui.ButtonOutline))
	itemFixed(menuBtn, 96, menuBtn.GetHeight())
	itemFixed(ui.NewLabel(0, 0, 0, 24, "对齐 shadcn DropdownMenu: 条目/勾选/单选/子菜单/快捷键/禁用/破坏性", rowMenu.Handle).SetMuted(), 480, 24)
	menuBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		showDemoMenu(hEle)
		return 0
	})

	// ===== 页 4: 数据(表格) =====
	card6 := ui.NewCard(24, 0, 728, 320, pData.Handle)
	v6 := vStack(card6, 14)
	itemFill(ui.NewCardTitle(0, 0, 0, "数据表格", v6.Handle), 24)

	desc := ui.NewLabel(0, 0, 0, 0,
		"原生 List 载体 + 全自绘皮肤: 表头/行底/行边框/滚动条; 行悬浮 hover:bg-muted/50, 选中 bg-muted, 最后一行不画底边。", v6.Handle).SetMuted()
	desc.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter | xcc.TextTrimming_EllipsisCharacter)
	itemFill(desc, 48)

	demoTable = ui.NewTable(0, 0, card6.ContentWidth(), 180, v6.Handle, []ui.TableColumn{
		{Title: "客户", Width: 200},
		{Title: "状态", Width: 120},
		{Title: "金额", Width: 120},
		{Title: "日期"}, // Width<=0: 占满剩余宽度
	})
	itemFixed(demoTable, card6.ContentWidth(), 180)
	for _, r := range [][4]string{
		{"张三", "已支付", "¥1,250.00", "2026-10-01"},
		{"李四", "待支付", "¥300.00", "2026-10-02"},
		{"王五", "已退款", "¥99.00", "2026-10-03"},
		{"赵六", "已支付", "¥2,400.00", "2026-10-04"},
		{"钱七", "处理中", "¥780.00", "2026-10-05"},
		{"孙八", "已支付", "¥1,120.00", "2026-10-06"},
		{"周九", "待支付", "¥450.00", "2026-10-07"},
		{"吴十", "已支付", "¥3,000.00", "2026-10-08"},
		{"郑一", "已退款", "¥260.00", "2026-10-09"},
		{"王二", "已支付", "¥880.00", "2026-10-10"},
		{"冯三", "处理中", "¥640.00", "2026-10-11"},
		{"陈四", "已支付", "¥1,990.00", "2026-10-12"},
	} {
		demoTable.AddRow(r[0], r[1], r[2], r[3])
	}
	demoTable.Select(1)
	demoTable.OnSelect(func(row int32) {
		fmt.Printf("选中行: %d\n", row)
	})
}

// demo 菜单状态(菜单是一次性对象, 状态由调用方持有, 每次弹出重建)
var (
	demoLineNo  = true
	demoThemeG1 = true  // 单选组: 浅色
	demoThemeG2 = false // 单选组: 深色
)

// showDemoMenu 弹出演示菜单, 覆盖全部条目类型。
func showDemoMenu(hEle int) {
	m := ui.NewMenu()
	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuOnSelect(func() {}))
	m.Item("粘贴", ui.MenuShortcut("Ctrl+V"))
	m.Separator()
	m.Label("外观")
	m.Check("显示行号", demoLineNo, ui.MenuOnCheck(func(on bool) { demoLineNo = on }))
	m.Radio("浅色主题", demoThemeG1, ui.MenuGroup(1), ui.MenuOnCheck(func(on bool) {
		if on {
			demoThemeG1, demoThemeG2 = true, false
		}
	}))
	m.Radio("深色主题", demoThemeG2, ui.MenuGroup(1), ui.MenuOnCheck(func(on bool) {
		if on {
			demoThemeG1, demoThemeG2 = false, true
		}
	}))
	m.Sub("更多操作", func(s *ui.Menu) {
		s.Item("刷新", ui.MenuShortcut("F5"))
		s.Item("另存为", ui.MenuShortcut("Ctrl+Shift+S"))
	})
	m.Separator()
	m.Item("删除项目", ui.MenuDestructive())
	m.Item("灰掉的选项", ui.MenuDisabled())
	m.PopupAtElement(hEle)
}

// buildDialogs 两个对话框: 表单型(编辑资料) 与 确认型(删除项目)。
func buildDialogs(hWindow int, editTrigger, delTrigger *ui.Button) {
	// 表单对话框: 标题 + 描述 + 两个带标签的输入框 + 页脚
	edit := ui.NewDialog(hWindow)
	edit.SetTitle("编辑资料")
	edit.SetDescription("修改你的公开资料, 完成后点击保存。")
	edit.BindTrigger(editTrigger)

	// 内容容器是普通元素, 子元素按显式坐标摆放(高度由对话框自动量出)。
	// 对齐 shadcn App.tsx: 两个 space-y-2 组(标签 14 + 8 + 输入框 32), 组间 gap-4
	cw := edit.ContentWidth()
	ui.NewLabel(0, -2, cw, 20, "显示名称", edit.Content())
	name := ui.NewInput(0, 22, cw, 0, edit.Content(), ui.WithPlaceholder("显示名称"))
	name.SetValue("twgh")
	ui.NewLabel(0, 68, cw, 20, "个人主页", edit.Content())
	ui.NewInput(0, 92, cw, 0, edit.Content(), ui.WithPlaceholder("https://"))

	edit.AddFooter("取消", ui.ButtonOutline, func() { edit.Close() })
	edit.AddFooter("保存", ui.ButtonPrimary, func() { edit.Close() })

	// 确认对话框: 只有标题 + 描述 + 页脚
	del := ui.NewDialog(hWindow)
	del.SetTitle("确认删除?")
	del.SetDescription("删除后无法恢复, 确定要删除该项目吗?")
	del.BindTrigger(delTrigger)
	del.AddFooter("取消", ui.ButtonOutline, func() { del.Close() })
	del.AddFooter("删除", ui.ButtonDestructive, func() { del.Close() })
}
