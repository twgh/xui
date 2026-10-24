// xui 组件演示: light/dark 一键换肤 + 全部组件。
//
// 布局: 窗口 → [标题栏行][内容区: 页签行 + 页面宿主], 页面与卡片全部走**流式布局**,
// 不写"算出来的绝对坐标" —— 窗口拉大/最大化时整页自适应:
//   - 页面 = 铺满宿主的竖向流式容器; 隐藏的页会被布局跳过(实测 0x0), 所以不需要叠放定位;
//   - 卡片 = 宽度铺满 + 高度按内容自撑(Auto); 表格/树这类"越高越好"的内容声明成撑满剩余高度;
//   - 页面留白与间距一律用布局项外边距 / SetSpace 表达。
//
// 主题用环境变量 XUI_THEME=dark 启动即暗色, 运行中可用右上角按钮切换;
// 启动页 XUI_HOME_PAGE = 0..7 = 基础1 / 基础2 / 表单1 / 表单2 / 导航 / 反馈 / 数据 / 图标。
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// 页面序号, 与 pageTitles 一一对应。
const (
	pageBase1 = iota // 基础1: 按钮 / 徽章 / 分隔线 / 标签 / 按钮组
	pageBase2        // 基础2: 卡片 / 比例容器 / 滚动容器
	pageForm1        // 表单1: 输入 / 复选 / 单选 / 开关 / 滑条 / 下拉框
	pageForm2        // 表单2: 日期与时间 / 月历 / 范围与预设
	pageNav          // 导航: 标签页 / 分页 / 菜单
	pageFeed         // 反馈: 对话框 / 提示气泡 / 进度
	pageData         // 数据: 表格 / 树
	pageIcons        // 图标: 图标总览
	pageCount
)

// pageTitles 的个数必须与 pageCount 一致(页签与页一一对应)。
var pageTitles = [...]string{"基础1", "基础2", "表单1", "表单2", "导航", "反馈", "数据", "图标"}

var (
	tip         *ui.Tooltip
	demoTree    *ui.Tree
	demoTreeSel int32
)

// 页面留白与行/列间距(逻辑像素)。标题栏高度(30)与阴影带(12)由 ui.AppWindow 内置。
const (
	pagePad int32 = 24 // 页面内容四周留白
	gap     int32 = 16 // 页面内行/列间距
)

func main() {
	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)
	// 安装 xcgui.dll 内部错误弹窗的拦截钩子
	xc.InstallErrorHook()
	// 重绘节流默认 30ms(hover 过渡只有约 5 帧); 设 10ms 让过渡/动画跑到 ~60fps。
	// 10 而不是 16: 节流按两次绘制的间隔算, 与运行时 16ms 的过渡定时器会周期性 beat 掉帧;
	// 低于 16 也不再涨(实测 ~16.6ms 硬地板)。见 DESIGN.md 第八节 46。
	a.SetPaintFrequency(10)

	// 启动主题: 环境变量 XUI_THEME=dark 暗色, 其它/不设为亮色, 免去改代码重建。
	// 运行中仍可用标题栏按钮一键切换。
	dark := strings.EqualFold(os.Getenv("XUI_THEME"), "dark")
	if dark {
		xui.SetTheme(xui.Dark())
	}
	// 动画慢放倍率(截图抓中间帧用): XUI_ANIM_SCALE=5 把 hover/展开收起放慢 5 倍。
	if v := os.Getenv("XUI_ANIM_SCALE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			xui.SetAnimationScale(f)
		}
	}

	// 窗口 = ui.NewAppWindow: 阴影/圆角/自绘标题栏(三键)/双击最大化/拖动/主题化底色全部内置。
	// 内容区是窗口的 Weight 布局项(纵向流式), 页面构建见 buildPages。
	w := ui.NewAppWindow("xui · shadcn 风格自绘组件", 1200, 860,
		ui.WithAppWindowMinSize(960, 660),
		ui.WithAppWindowIcon(icons.IconAppWindow))
	tip = ui.NewTooltip()

	// 标题栏自定义按钮(排在最小化左边): 布局边界 / 主题切换, 都带 tooltip。
	// 图标显示点击后的动作 —— 隐藏时 eye(去看), 显示时 eye-off(去关)。
	showLayoutFrame := false
	var btnFrame *ui.Button
	btnFrame = w.AddCaptionButton(icons.IconEye, func() {
		showLayoutFrame = !showLayoutFrame
		app.ShowLayoutFrame(showLayoutFrame) // 显示布局对象边界
		btnFrame.SetIcon(common.Choose(showLayoutFrame, icons.IconEyeOff, icons.IconEye))
		w.Redraw(false)
	})
	tip.Bind(btnFrame.Handle, "显示/隐藏布局对象边界")

	// 主题切换按钮: 图标显示切换目标 —— 亮色时 moon(去暗色), 暗色时 sun(去亮色)
	var btnTheme *ui.Button
	btnTheme = w.AddCaptionButton(icons.IconMoon, func() {
		dark = !dark
		btnTheme.SetIcon(common.Choose(dark, icons.IconSun, icons.IconMoon))
		xui.SetTheme(common.Choose(dark, xui.Dark(), xui.Light()))
		// 切换主题会重绘窗口, 所以按钮不用单独重绘
	})
	tip.Bind(btnTheme.Handle, "切换主题")
	if dark {
		btnTheme.SetIcon(icons.IconSun)
	}

	// 内容区: AppWindow 的纵向流式容器(宽铺满、高占满标题栏以下); 页面自己管留白
	content := w.Content()
	content.SetSpace(0)
	buildPages(w.Handle, content.Handle)

	w.Show(true)
	a.Run()
	a.Exit()
}

// ===== 布局脚手架 =====
//
// 容器骨架优先用现成入口: 横向行 `xui.HStack`(高固定、子项垂直居中)、等宽列
// `xui.VStackWeight`; 布局项尺寸走 `xui.Item*`(见 xui/layout.go 的布局约定:
// **只声明要改变的那条轴**, 元素自身尺寸由组件构造函数建好)。
// 下面这几个是 demo 自己的组合(容器行 / 卡片 / 分组标题), 都用布局表达, 不出现绝对坐标。

// row 建一行"放卡片的容器行": 宽度铺满, 高度 ht(传 0 = 按内容自撑 Auto),
// 子项间距 space, 交叉轴顶端对齐(卡片各自按内容自撑高度, 不要被拉成一样高)。
//
// 行高必须 Auto: 页面是滚动容器, 内容区高度按内容自撑(Auto), 里面再写 Weight/Fill 会塌成 0。
// 行是 Auto 高时"Weight 宽的列"仍然算得出宽度(实测: 内容区 352 时两列 201/135 = 3:2)。
func row(parent int, ht, space int32) *widget.LayoutEle {
	r := widget.NewLayoutEle(0, 0, 0, ht, parent)
	r.EnableHorizon(true)
	r.EnableLayout(true)
	r.EnableBkTransparent(true)
	r.EnableAutoWrap(false)
	r.SetSpace(space)
	r.SetAlignV(xcc.Layout_Align_Top)
	r.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	if ht > 0 {
		r.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, ht)
	} else {
		r.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}
	return r
}

// rowStretch 建一行"吃满父级剩余高度"的行 —— 只用在**铺满型页面**(SetContentFill 的数据页):
// 那时内容区高度 = 视口高(确定尺寸), 行用 Weight 才拿得到剩余空间, 行列里的 stretchCard /
// 表格 / 树才会跟着窗口长高。滚动型页面(内容区 Auto)里用它会塌成 0 高, 请用 row。
func rowStretch(parent int, space int32) *widget.LayoutEle {
	r := widget.NewLayoutEle(0, 0, 0, 0, parent)
	r.EnableHorizon(true)
	r.EnableLayout(true)
	r.EnableBkTransparent(true)
	r.EnableAutoWrap(false)
	r.SetSpace(space)
	r.SetAlignV(xcc.Layout_Align_Top)
	r.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	xui.ItemFillHeight(r)
	return r
}

// col 建一行里的一列: 宽度按权重等分, **高度按内容自撑(Auto)**, 子项间距 space。
//
// 高度为什么必须 Auto(而不是 xui.VStackWeight 的 Fill): 列里装的是"高度自撑"的卡片,
// 而 xcgui 对"Auto 高的卡片挂在**任意一轴是 Fill** 的父级里"的编排是坏的 —— 卡片内容区
// 里的 Fill 子项(原生 Edit/ComboBox 这种整宽有边框的控件)会按**卡片宽**排版, 右边缘正好
// 压到卡片边框上(实测: 输入框宽 = 卡片宽 568 而不是内容区宽 536, 见 DESIGN 第八节 59)。
func col(parent int, weight, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(space)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, weight)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

// colStretch 建一行里"能长高"的一列: 宽度按权重等分, 高度也按权重占满。
// 与 col 只差高度那一维 —— 它用 Weight(= 父级剩余空间), 供列内 stretchCard(高度 Fill)这类
// "越高越好"的面板用; col 是 Auto(按内容自撑), 只适合列内是"高度自撑"的卡片
// (两种高度写法对内容区 Fill 子项宽度的影响不同, 见 DESIGN 第八节 59)。
func colStretch(parent int, weight, space int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(space)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Weight, weight)
	xui.ItemFillHeight(l)
	return l
}

// autoCard 建一张"宽度铺满、高度按内容自撑"的卡片, 并返回它的内容区。
// 卡片内部是布局容器, 内容区随卡片宽度伸缩(见 ui.Card 的注释)。
func autoCard(parent int, space int32) (*ui.Card, *widget.LayoutEle) {
	c := ui.NewCard(0, 0, 0, 0, parent)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return c, c.ContentStack(space)
}

// stretchCard 建一张"宽度铺满、高度占满父级剩余高度"的卡片(给表格/树这类要长高的面板)。
func stretchCard(parent int, space int32) (*ui.Card, *widget.LayoutEle) {
	c := ui.NewCard(0, 0, 0, 0, parent)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	xui.ItemFillHeight(c)
	return c, c.ContentStack(space)
}

// hint 建一个弱化的小标题(分组说明), 高度用 Label 默认的 24。
func hint(parent int, text string) *ui.Label {
	return ui.NewLabel(0, 0, 0, 0, text, parent).SetMuted()
}

// desc 建一段多行说明文字(高度写在创建参数里, 布局只声明宽度)。
func desc(parent int, text string) *ui.Label {
	return ui.NewLabel(0, 0, 0, 36, text, parent).SetMuted().SetWrap()
}

// ratioPreview 建一个比例演示块: AspectRatio 的内容区里放自绘占位块
// (muted 底 + 图片图标 + 比例文字); 宽度按权重分, 高度由宽度推导。
func ratioPreview(parent int, weight int32, ratio float64, label string) *ui.AspectRatio {
	ar := ui.NewAspectRatio(0, 0, 0, 0, ratio, parent)
	xui.ItemWeight(ar, weight)
	ph := widget.NewElement(0, 0, 0, 0, ar.Content().Handle)
	ph.EnableBkTransparent(true)
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
		cx, cy := (p.Rc.Left+p.Rc.Right)/2, (p.Rc.Top+p.Rc.Bottom)/2
		const iconSize int32 = 28
		icon := xc.RECT{Left: cx - iconSize/2, Top: cy - iconSize - 8, Right: cx + iconSize/2, Bottom: cy - 8}
		p.Icon(icons.IconImage, &icon, t.MutedForeground, 2)
		p.Draw.SetFont(xui.FontHandle(t.FontDefault))
		text := xc.RECT{Left: p.Rc.Left, Top: cy + 6, Right: p.Rc.Right, Bottom: cy + 30}
		p.Text(label, &text, t.MutedForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		return 0
	})
	return ar
}

// envHomePage 读启动页环境变量并夹取到合法范围。
func envHomePage() int {
	i, _ := strconv.Atoi(os.Getenv("XUI_HOME_PAGE"))
	if i < 0 || i >= pageCount {
		return 0
	}
	return i
}

// ===== 页面骨架 =====

// buildPages 建页签行与页面宿主。
//
// 页面宿主是竖向流式容器, 每个页都是它的 Fill 布局项 —— **隐藏的页会被布局跳过**
// (实测: 隐藏页尺寸被排成 0x0), 所以"同一时刻只有一页可见"天然成立, 不需要绝对定位叠放。
// 页面内容静态创建一次即可(主题切换走整窗重绘, 不需要重建)。
func buildPages(hWindow, host int) {
	home := envHomePage()

	// 页签行: 左右与窗口留白对齐。8 个页签, 每个 80 宽。
	tabs := ui.NewTabs(0, 0, 640, host, pageTitles[:])
	xui.ItemMargin(tabs, pagePad, 8, 0, 6)

	pageHost := widget.NewLayoutEle(0, 0, 0, 0, host)
	pageHost.EnableHorizon(false)
	pageHost.EnableLayout(true)
	pageHost.EnableBkTransparent(true)
	pageHost.EnableAutoWrap(false)
	pageHost.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	// 高度必须用 Weight(剩余空间), 不能用 Fill: Fill 是"父级整高", 会把页签行那一段也算进去,
	// 整个页面区就往下溢出、页面底部的留白被挤出窗口(实测)。
	xui.ItemFillHeight(pageHost)

	pages := make([]*ui.ScrollArea, pageCount)
	bodies := make([]*widget.LayoutEle, pageCount)
	for i := range pages {
		pages[i], bodies[i] = newPage(pageHost.Handle)
	}
	// 数据页是"铺满型": 表格/树吃窗口剩余高度(整页内容由 Weight 自适应、永远装得下,
	// 不需要滚动)。其余页是自撑型: 内容比窗口高时滚动。
	pages[pageData].SetContentFill()

	buildBase1Page(bodies[pageBase1])
	buildBase2Page(bodies[pageBase2])
	buildForm1Page(bodies[pageForm1])
	buildForm2Page(bodies[pageForm2])
	buildNavPage(bodies[pageNav])
	buildFeedPage(bodies[pageFeed], hWindow)
	buildDataPage(bodies[pageData])
	buildIconPage(bodies[pageIcons])

	show := func(i int) {
		for j, p := range pages {
			p.Show(j == i)

			if j == i {
				// 隐藏页首次显示时, ScrollArea 的命中区域可能还停在旧滚动位置 ——
				// 表现为点日期框/时间框没反应, 滚一下滚轮才正常。延迟等页面真正上屏后
				// 再调整一次布局。
				page := pages[i]
				app.AfterFunc(2*16*time.Millisecond, func() {
					if !page.IsShow() {
						return
					}
					page.AdjustLayout()
					page.Redraw(false)
				})
			}
		}
		// 可见性变了必须重排宿主: 隐藏的页不参与排版, 显示的那页才会拿到整个高度。
		pageHost.AdjustLayout()
		pageHost.Redraw(false)
	}

	// 首页在**所有页面建完之后**才显示: 树的启动选中态在隐藏页上会丢(见 DESIGN 第八节 42)。
	show(home)

	tabs.OnChange(func(i int32) {
		show(int(i))
		if int(i) == pageData && demoTreeSel != xcc.XC_ID_ERROR {
			demoTree.Select(demoTreeSel)
			demoTreeSel = xcc.XC_ID_ERROR
		}
	})
	tabs.SetActive(int32(home)) // 会同步触发一次 OnChange(实测)
}

// newPage 建一个页面: 滚动容器(铺满宿主) + 其内带四周留白的内容区(竖向流式, 子项间距 gap)。
// 内容比窗口高时滚动查看, 不再被裁掉(见 ui.ScrollArea)。
func newPage(host int) (page *ui.ScrollArea, body *widget.LayoutEle) {
	page = ui.NewScrollArea(0, 0, 0, 0, host)
	page.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	xui.ItemFillHeight(page) // 高度 = 页面宿主的剩余空间(Weight)

	body = page.ContentStack(gap)
	body.LayoutItem_SetMargin(pagePad, pagePad, pagePad, pagePad)
	return page, body
}

// ===== 页 1: 基础1(按钮 / 徽章 / 分隔线 / 标签 / 按钮组) =====

func buildBase1Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: 按钮 ----
	_, v := autoCard(col(r.Handle, 3, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "按钮", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "6 个变体 · 4 档尺寸 + 4 个方形图标档, 悬停 150ms 过渡", v.Handle))

	xui.ItemFill(hint(v.Handle, "变体"))
	rb := xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Default", rb.Handle), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Secondary", rb.Handle, ui.WithVariant(ui.ButtonSecondary)), 112)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Outline", rb.Handle, ui.WithVariant(ui.ButtonOutline)), 104)
	rb = xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Ghost", rb.Handle, ui.WithVariant(ui.ButtonGhost)), 96)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Destructive", rb.Handle, ui.WithVariant(ui.ButtonDestructive)), 112)
	linkBtn := ui.NewButton(0, 0, 0, 0, "Link", rb.Handle, ui.WithVariant(ui.ButtonLink))
	xui.ItemWidth(linkBtn, 88)
	disBtn := ui.NewButton(0, 0, 0, 0, "Disabled", rb.Handle)
	xui.ItemWidth(disBtn, 104)
	disBtn.Enable(false)

	xui.ItemFill(hint(v.Handle, "尺寸档位(Xs / Sm / Default / Lg)"))
	rs := xui.HStack(v.Handle, 40, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Xs", rs.Handle, ui.WithSize(ui.ButtonSizeXs)), 56)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Sm", rs.Handle, ui.WithSize(ui.ButtonSizeSm)), 60)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Default", rs.Handle), 88)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Lg", rs.Handle, ui.WithSize(ui.ButtonSizeLg)), 68)

	xui.ItemFill(hint(v.Handle, "图标按钮(方形档, 宽度由预设给出)"))
	ri := xui.HStack(v.Handle, 40, 8)
	iconBtn := ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconPlus))
	tip.Bind(iconBtn.Handle, "Icon")
	iconBtn = ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconCheck))
	tip.Bind(iconBtn.Handle, "Icon(另一个)")
	ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconSettings))
	ui.NewButton(0, 0, 0, 0, "", ri.Handle, ui.WithSize(ui.ButtonSizeIconXs), ui.WithIcon(icons.IconX))
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "", ri.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconMoon)), 36)

	// 校验失败态(aria-invalid: 红边框 + 3px 红环, 常驻不要求焦点)
	xui.ItemFill(hint(v.Handle, "校验失败(aria-invalid)"))
	rv := xui.HStack(v.Handle, 36, 8)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Invalid", rv.Handle, ui.WithButtonInvalid()), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Outline", rv.Handle, ui.WithVariant(ui.ButtonOutline), ui.WithButtonInvalid()), 104)
	xui.ItemWidth(ui.NewButton(0, 0, 0, 0, "Destructive", rv.Handle, ui.WithVariant(ui.ButtonDestructive), ui.WithButtonInvalid()), 120)

	// ---- 列 2: 徽章 / 分隔线 / 标签 ----
	_, v2 := autoCard(col(r.Handle, 2, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "徽章 · 分隔线 · 标签", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "徽章 6 个变体; 分隔线横/竖; 标签含弱化与禁用态", v2.Handle))

	xui.ItemFill(hint(v2.Handle, "徽章"))
	rg := xui.HStack(v2.Handle, 24, 8)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Default", rg.Handle), 68)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Secondary", rg.Handle, ui.WithBadgeVariant(ui.BadgeSecondary)), 84)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Outline", rg.Handle, ui.WithBadgeVariant(ui.BadgeOutline)), 76)
	rg = xui.HStack(v2.Handle, 24, 8)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Destructive", rg.Handle, ui.WithBadgeVariant(ui.BadgeDestructive)), 92)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Ghost", rg.Handle, ui.WithBadgeVariant(ui.BadgeGhost)), 64)
	xui.ItemWidth(ui.NewBadge(0, 0, 0, 0, "Link", rg.Handle, ui.WithBadgeVariant(ui.BadgeLink)), 56)

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "分隔线与标签"))
	rl := xui.HStack(v2.Handle, 24, 12)
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 0, "常规文本", rl.Handle), 96)
	ui.NewSeparator(0, 0, 20, rl.Handle, ui.WithSeparatorVertical())
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 0, "弱化文本", rl.Handle).SetMuted(), 96)
	ui.NewSeparator(0, 0, 20, rl.Handle, ui.WithSeparatorVertical())
	disLabel := ui.NewLabel(0, 0, 0, 0, "禁用文本", rl.Handle)
	xui.ItemWidth(disLabel, 96)
	disLabel.Enable(false)

	// ---- 第二行: 按钮组 ----
	// 组是宽高自撑的 LayoutEle: 子按钮 cx 传 0 时按文字量宽, 组自动拼成一条,
	// 首尾保留外圆角、相邻边收成直角, 相邻边框自然并成 1px(见 buttongroup.go)。
	_, vbg := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "按钮组", vbg.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ButtonGroup 横向/纵向、首尾外角、自动量宽; 文本块与分隔条跟随组高; 嵌套组自动 8px 间距", vbg.Handle))

	xui.ItemFill(hint(vbg.Handle, "横向 · 分隔条 · 图标按钮"))
	rbg1 := xui.HStack(vbg.Handle, 40, 16)
	bg1 := ui.NewButtonGroup(rbg1.Handle)
	ui.NewButton(0, 0, 0, 0, "Archive", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Report", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Snooze", bg1.Handle, ui.WithVariant(ui.ButtonOutline))
	ui.NewButtonGroupSeparator(bg1.Handle)
	moreBtn := ui.NewButton(0, 0, 0, 0, "", bg1.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconMoreHorizontal),
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonHasPopup())
	tip.Bind(moreBtn.Handle, "更多操作 · 弹层触发区(按下不下沉)")
	moreBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		moreBtn.SetExpanded(true)
		moreBtn.Redraw(true) // 菜单阻塞, 先立即重绘出展开色
		defer moreBtn.SetExpanded(false)
		m := ui.NewMenu()
		m.Width(176)
		m.Item("导出", ui.MenuShortcut("Ctrl+E"))
		m.Item("归档")
		m.Separator()
		m.Item("删除", ui.MenuDestructive())
		m.PopupAtElement(hEle)
		return 0
	})

	xui.ItemFill(hint(vbg.Handle, "文本块 · 分隔条 · 行内图标"))
	rbg2 := xui.HStack(vbg.Handle, 40, 16)
	bg2 := ui.NewButtonGroup(rbg2.Handle)
	ui.NewButtonGroupText("https://", bg2.Handle)
	ui.NewButton(0, 0, 0, 0, "Copy", bg2.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	ui.NewButtonGroupSeparator(bg2.Handle)
	ui.NewButton(0, 0, 0, 0, "", bg2.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowUp), ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "嵌套组(组间距 8)"))
	rbg3 := xui.HStack(vbg.Handle, 40, 16)
	bg3 := ui.NewButtonGroup(rbg3.Handle)
	bg31 := ui.NewButtonGroup(bg3.Handle)
	for _, s := range []string{"1", "2", "3", "4", "5"} {
		ui.NewButton(0, 0, 0, 0, s, bg31.Handle,
			ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	}
	bg32 := ui.NewButtonGroup(bg3.Handle)
	ui.NewButton(0, 0, 0, 0, "", bg32.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowLeft), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "", bg32.Handle,
		ui.WithSize(ui.ButtonSizeIconSm), ui.WithIcon(icons.IconArrowRight), ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "纵向组 · 尺寸档(组内统一 rounded-lg)"))
	rbg4 := xui.HStack(vbg.Handle, 80, 16)
	bg4 := ui.NewButtonGroup(rbg4.Handle, ui.WithButtonGroupOrientation(ui.ButtonGroupVertical))
	ui.NewButton(0, 0, 0, 0, "", bg4.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconPlus), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "", bg4.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconMinus), ui.WithVariant(ui.ButtonOutline))

	bg5 := ui.NewButtonGroup(rbg4.Handle)
	ui.NewButton(0, 0, 0, 0, "Xs", bg5.Handle,
		ui.WithSize(ui.ButtonSizeXs), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Sm", bg5.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline))
	ui.NewButton(0, 0, 0, 0, "Default", bg5.Handle, ui.WithVariant(ui.ButtonOutline))

	xui.ItemFill(hint(vbg.Handle, "行内图标 · 展开态 · 胶囊 · 加载态"))
	rbg6 := xui.HStack(vbg.Handle, 40, 12)
	ui.NewButton(0, 0, 0, 0, "New Branch", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLeadingIcon(icons.IconGitBranch))
	ui.NewButton(0, 0, 0, 0, "下一步", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonTrailingIcon(icons.IconChevronRight))
	filterUnread, filterAttach := false, false
	filterBtn := ui.NewButton(0, 0, 0, 0, "筛选", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonTrailingIcon(icons.IconChevronDown),
		ui.WithButtonHasPopup())
	tip.Bind(filterBtn.Handle, "展开态: 点击打开筛选菜单")
	filterBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		filterBtn.SetExpanded(true)
		filterBtn.Redraw(true)             // 菜单是阻塞式原生弹窗, 不立即重绘就看不到展开色
		defer filterBtn.SetExpanded(false) // 菜单阻塞到关闭, 返回后恢复非展开色
		m := ui.NewMenu()
		m.Width(176)
		m.Label("筛选")
		m.Check("仅未读", filterUnread, ui.MenuOnCheck(func(on bool) { filterUnread = on }))
		m.Check("含附件", filterAttach, ui.MenuOnCheck(func(on bool) { filterAttach = on }))
		m.Separator()
		m.Item("清除筛选", ui.MenuOnSelect(func() { filterUnread, filterAttach = false, false }))
		m.PopupAtElement(hEle)
		return 0
	})
	genBtn := ui.NewButton(0, 0, 0, 0, "Generating", rbg6.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLoading())
	genBtn.Enable(false)
	ui.NewButton(0, 0, 0, 0, "", rbg6.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconArrowUp), ui.WithVariant(ui.ButtonOutline), ui.WithButtonRounded())

	xui.ItemFill(hint(vbg.Handle, "输入框 / 下拉框直接入组(壳体跟随首尾圆角)"))
	rbg7 := xui.HStack(vbg.Handle, 40, 16)
	bg7 := ui.NewButtonGroup(rbg7.Handle)
	search := ui.NewInput(0, 0, 0, 0, bg7.Handle, ui.WithPlaceholder("Search..."))
	xui.ItemWidth(search, 220)
	ui.NewButton(0, 0, 0, 0, "", bg7.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconSearch), ui.WithVariant(ui.ButtonOutline))

	bg8 := ui.NewButtonGroup(rbg7.Handle)
	combo := ui.NewComboBox(0, 0, 0, 0, bg8.Handle,
		[]string{"开发环境", "测试环境", "预发环境"}, ui.WithComboPlaceholder("选择环境"))
	xui.ItemWidth(combo, 180)
	ui.NewButton(0, 0, 0, 0, "", bg8.Handle,
		ui.WithSize(ui.ButtonSizeIcon), ui.WithIcon(icons.IconSettings), ui.WithVariant(ui.ButtonOutline))

	// ---- 第三行: 快捷键 / 加载指示器 / 骨架屏 ----
	r3 := row(body.Handle, 0, gap)

	// 快捷键键帽
	_, vkbd := autoCard(col(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "快捷键", vkbd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Kbd 键帽: 高 20、宽度按文字量测; KbdGroup 负责组合键间距 4", vkbd.Handle))

	xui.ItemFill(hint(vkbd.Handle, "单键 / 组合键"))
	rk := xui.HStack(vkbd.Handle, 24, 12)
	ui.NewKbd(rk.Handle, "Esc")
	gk := ui.NewKbdGroup(rk.Handle)
	ui.NewKbd(gk.Handle, "Ctrl")
	ui.NewKbd(gk.Handle, "K")
	gk = ui.NewKbdGroup(rk.Handle)
	ui.NewKbd(gk.Handle, "⌘")
	ui.NewKbd(gk.Handle, "⇧")
	ui.NewKbd(gk.Handle, "P")
	xui.ItemFill(hint(vkbd.Handle, "快捷键紧跟说明文字"))
	rk2 := xui.HStack(vkbd.Handle, 24, 8)
	gk2 := ui.NewKbdGroup(rk2.Handle)
	ui.NewKbd(gk2.Handle, "Ctrl")
	ui.NewKbd(gk2.Handle, "S")
	xui.ItemWidth(ui.NewLabel(0, 0, 0, 20, "保存", rk2.Handle).SetFontSize(14), 48)

	// 加载指示器
	_, vspin := autoCard(col(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "加载指示器", vspin.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Spinner: Loader2 图标持续旋转, 1s/圈; 相位由运行时驱动, 不占组件定时器", vspin.Handle))

	xui.ItemFill(hint(vspin.Handle, "与文字同行"))
	spRow := xui.HStack(vspin.Handle, 24, 8)
	ui.NewSpinner(spRow.Handle)
	spText := ui.NewLabel(0, 0, 0, 20, "正在加载…", spRow.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(spText, 1)

	xui.ItemFill(hint(vspin.Handle, "尺寸 16 / 20 / 24"))
	spSizes := xui.HStack(vspin.Handle, 32, 16)
	ui.NewSpinner(spSizes.Handle)
	ui.NewSpinner(spSizes.Handle, ui.WithSpinnerSize(20))
	ui.NewSpinner(spSizes.Handle, ui.WithSpinnerSize(24))

	xui.ItemFill(hint(vspin.Handle, "颜色: muted / primary"))
	spColors := xui.HStack(vspin.Handle, 24, 16)
	ui.NewSpinner(spColors.Handle, ui.WithSpinnerMuted())
	ui.NewSpinner(spColors.Handle, ui.WithSpinnerColorFunc(func(t *xui.Theme) xui.Color {
		return t.Primary
	}))

	// 骨架屏
	_, vsk := autoCard(col(r3.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "骨架屏", vsk.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Skeleton: bg-accent + rounded-md, 2s 脉冲(1 → 0.5 → 1)", vsk.Handle))

	xui.ItemFill(hint(vsk.Handle, "头像 + 两行文本"))
	skRow := xui.HStack(vsk.Handle, 40, 12)
	ui.NewSkeleton(0, 0, 40, 40, skRow.Handle, ui.WithSkeletonCircle())
	skCol := widget.NewLayoutEle(0, 0, 0, 0, skRow.Handle)
	skCol.EnableHorizon(false)
	skCol.EnableLayout(true)
	skCol.EnableBkTransparent(true)
	skCol.EnableAutoWrap(false)
	skCol.SetSpace(8)
	skCol.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	skCol.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	ui.NewSkeleton(0, 0, 180, 12, skCol.Handle)
	ui.NewSkeleton(0, 0, 120, 12, skCol.Handle)

	xui.ItemFill(hint(vsk.Handle, "卡片占位"))
	xui.ItemFill(ui.NewSkeleton(0, 0, 0, 72, vsk.Handle))
	xui.ItemFill(ui.NewSkeleton(0, 0, 0, 12, vsk.Handle))
	lastSk := ui.NewSkeleton(0, 0, 0, 12, vsk.Handle)
	xui.ItemFill(lastSk)
	xui.ItemMarginRight(lastSk, 80)
}

// ===== 页 2: 基础2(卡片 / 比例容器 / 滚动容器) =====

func buildBase2Page(body *widget.LayoutEle) {
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
	_, vr := autoCard(col(rr.Handle, 2, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "比例容器", vr.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"AspectRatio: 宽度由布局/布局项决定, 高度 = 宽度 / ratio; 内容放 Content() 里铺满比例框", vr.Handle))
	ritems := row(vr.Handle, 0, 16)
	ratioPreview(ritems.Handle, 2, 16.0/9.0, "16 : 9")
	ratioPreview(ritems.Handle, 1, 1, "1 : 1")
	ratioPreview(ritems.Handle, 1, 9.0/16.0, "9 : 16")

	// sm 卡片: 内边距与页脚都是 p-3(12), 标题 text-sm(14)
	smCard := ui.NewCard(0, 0, 0, 0, col(rr.Handle, 1, gap).Handle, ui.WithCardSize(ui.CardSizeSm))
	xui.ItemFill(smCard)
	smCard.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	vsm := smCard.ContentStack(12)
	chsm := ui.NewCardHeader(0, 0, 0, vsm.Handle, 4)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "小卡片", chsm.TitleStack().Handle).SetFontSize(smCard.TitleSize()))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "size=sm: 内边距与页脚都是 12, 标题 14", chsm.TitleStack().Handle))
	xui.ItemFill(ui.NewButton(0, 0, 0, 0, "操作", vsm.Handle, ui.WithSize(ui.ButtonSizeSm), ui.WithVariant(ui.ButtonOutline)))
	smFooter := smCard.Footer(0)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "CardFooter · p-3", smFooter.Content().Handle).SetMuted())

	// ---- 滚动容器(组件形态; 页面本身也是它, 见 newPage) ----
	_, vs := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滚动容器", vs.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ui.ScrollArea: 原生 LayoutFrame 载体, 内容比容器高时纵向滚动、滚动条自动显隐,"+
			"内容区高按内容自撑(Auto)", vs.Handle))
	box := ui.NewScrollArea(0, 0, 0, 180, vs.Handle)
	xui.ItemFill(box)
	for i := 1; i <= 12; i++ {
		l := ui.NewLabel(0, 0, 0, 0, fmt.Sprintf("第 %d 行 · 滚动容器里的普通内容", i), box.Content().Handle)
		xui.ItemFill(l)
	}
	box.Content().SetSpace(6)

	// ---- 列表项(Item): 媒体 / 内容 / 动作 + 页眉页脚 ----
	_, vi := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "列表项", vi.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Item: default / outline / muted 三个变体, default / sm 两档尺寸; Media + Content(吃剩余宽度) + Actions, 可选 Header/Footer", vi.Handle))

	ri := row(vi.Handle, 0, 16)
	itL := col(ri.Handle, 1, 10)
	itR := col(ri.Handle, 1, 10)

	group := ui.NewItemGroup(0, 0, 0, 0, itL.Handle)
	xui.ItemFill(group)

	// 1. outline + icon media + 尾部按钮
	it1 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemVariant(ui.ItemOutline))
	xui.ItemFill(it1)
	ui.NewItemMedia(it1.Media().Handle, icons.IconShieldCheck,
		ui.WithItemMediaVariant(ui.ItemMediaIcon))
	c1 := it1.Content()
	ui.NewItemTitle(c1.Handle, "安全提醒")
	ui.NewItemDescription(c1.Handle, "检测到一台未识别的设备登录了你的账户。")
	ui.NewButton(0, 0, 0, 0, "查看", it1.Actions().Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))

	ui.NewItemSeparator(group.Handle)

	// 2. muted + Header(标题吃权重, 徽章贴右) + Footer
	it2 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemVariant(ui.ItemMuted))
	xui.ItemFill(it2)
	itHead := it2.Header()
	headTitle := ui.NewItemTitle(itHead.Handle, "部署完成")
	xui.ItemWeight(headTitle, 1)
	okBadge := ui.NewBadge(0, 0, 0, 0, "成功", itHead.Handle, ui.WithBadgeVariant(ui.BadgeSecondary))
	xui.ItemWidth(okBadge, 56)
	c2 := it2.Content()
	ui.NewItemDescription(c2.Handle, "prod-2026.10.24 已发布到生产环境, 共 42 个文件。")
	itFoot := it2.Footer()
	footHint := ui.NewItemDescription(itFoot.Handle, "2 分钟前")
	xui.ItemWeight(footHint, 1)
	ui.NewButton(0, 0, 0, 0, "查看日志", itFoot.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm))

	ui.NewItemSeparator(group.Handle)

	// 3. default + sm 尺寸, 无动作区
	it3 := ui.NewItem(0, 0, 0, 0, group.Handle, ui.WithItemSize(ui.ItemSizeSm))
	xui.ItemFill(it3)
	ui.NewItemMedia(it3.Media().Handle, icons.IconFolderCode)
	c3 := it3.Content()
	ui.NewItemTitle(c3.Handle, "design-system")
	ui.NewItemDescription(c3.Handle, "昨天更新 · 12 个组件")

	// 右列: 内容 / 动作 / 自定义媒体槽
	group2 := ui.NewItemGroup(0, 0, 0, 0, itR.Handle)
	xui.ItemFill(group2)

	it4 := ui.NewItem(0, 0, 0, 0, group2.Handle, ui.WithItemVariant(ui.ItemOutline))
	xui.ItemFill(it4)
	c4 := it4.Content()
	ui.NewItemTitle(c4.Handle, "团队成员")
	ui.NewItemDescription(c4.Handle, "邀请成员加入当前工作区, 可随时移除。")
	act4 := it4.Actions()
	countBadge := ui.NewBadge(0, 0, 0, 0, "3 人", act4.Handle)
	xui.ItemWidth(countBadge, 52)
	ui.NewButton(0, 0, 0, 0, "邀请", act4.Handle,
		ui.WithSize(ui.ButtonSizeSm), ui.WithButtonLeadingIcon(icons.IconPlus))

	ui.NewItemSeparator(group2.Handle)

	it5 := ui.NewItem(0, 0, 0, 0, group2.Handle, ui.WithItemVariant(ui.ItemMuted), ui.WithItemSize(ui.ItemSizeSm))
	xui.ItemFill(it5)
	// 自定义媒体槽: 不限于图标, 任何元素都可以放进来。
	avatar := widget.NewElement(0, 0, 32, 32, it5.Media().Handle)
	avatar.EnableBkTransparent(true)
	avatar.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 32)
	avatar.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 32)
	avatar.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		th := xui.Current()
		p.FillRound(&p.Rc, th.Primary, th.Pill(32))
		return 0
	})
	xui.AttachWindow(avatar.Handle)
	c5 := it5.Content()
	ui.NewItemTitle(c5.Handle, "twgh")
	ui.NewItemDescription(c5.Handle, "工作区管理员")

	ui.NewItemSeparator(group2.Handle)

	it6 := ui.NewItem(0, 0, 0, 0, group2.Handle)
	xui.ItemFill(it6)
	h6 := it6.Header()
	bill := ui.NewLabel(0, 0, 0, 20, "本月账单", h6.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(bill, 1)
	price := ui.NewLabel(0, 0, 96, 20, "¥ 128.00", h6.Handle).SetFontSize(14).SetMedium()
	xui.ItemWidth(price, 96)
}

// 骨架是"一行两列、每列把两张卡竖着排", 不是"两行各两列":
// 行高只有 Weight(按剩余空间等分)可用, 卡片比行高时下一行就会压到上一行的卡片上
// (实测: 输入框卡 430 高, 行只分到 363, 第二行的卡片上沿 379 直接盖住它 —— 用户报的
// "下方两个卡片盖住上方卡片")。竖排在同一列里则天然按内容顺序往下走, 不重叠。
func buildForm1Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := col(r.Handle, 1, gap)
	colR := col(r.Handle, 1, gap)

	// ---- 左列 1: 输入框 ----
	_, v := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "输入框", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生 Edit 载体: 只画边框与焦点环, 光标/输入法/选区都是原生的", v.Handle))

	xui.ItemFill(hint(v.Handle, "邮箱地址"))
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithPlaceholder("m@example.com")))
	xui.ItemFill(hint(v.Handle, "带初始值"))
	filled := ui.NewInput(0, 0, 0, 0, v.Handle)
	xui.ItemFill(filled)
	filled.SetValue("twgh")
	xui.ItemFill(hint(v.Handle, "多行(高度写在创建参数里, 内容溢出会出现滚动条)"))
	ml := ui.NewInput(0, 0, 0, 100, v.Handle,
		ui.WithPlaceholder("介绍一下自己…"), ui.WithMultiline())
	xui.ItemFill(ml)
	ml.SetValue(strings.Repeat("测试多行输入, 滚动条样式。\n", 5))
	xui.ItemFill(hint(v.Handle, "禁用"))
	dis := ui.NewInput(0, 0, 0, 0, v.Handle)
	xui.ItemFill(dis)
	dis.SetValue("不可编辑")
	dis.Enable(false)
	xui.ItemFill(hint(v.Handle, "校验失败(aria-invalid)"))
	inv := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithInputInvalid())
	xui.ItemFill(inv)
	inv.SetValue("not-an-email")
	xui.ItemFill(hint(v.Handle, "步进器(−/+ 或 ↑↓ 调整, 失焦自动夹取/吸附; 步长可配 1 / 10 / 0.1)"))
	step := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithStepper(0, 100, 1))
	xui.ItemFill(step)
	step.SetNumber(42)
	xui.ItemFill(hint(v.Handle, "上下三角步进器(悬停/聚焦才出现, 步长 10)"))
	arrows := ui.NewInput(0, 0, 0, 0, v.Handle,
		ui.WithStepper(0, 1000, 10), ui.WithStepperArrows(), ui.WithStepperAutoHide())
	xui.ItemFill(arrows)
	arrows.SetNumber(320)
	xui.ItemFill(hint(v.Handle, "前置图标 + 前后缀(input-group)"))
	url := ui.NewInput(0, 0, 0, 0, v.Handle,
		ui.WithLeadingIcon(icons.IconAtSign), ui.WithPrefixText("https://"), ui.WithSuffixText(".com"))
	xui.ItemFill(url)
	url.SetValue("twgh")
	xui.ItemFill(hint(v.Handle, "清除按钮 / 密码显隐 / 字符计数"))
	search := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithLeadingIcon(icons.IconSearch), ui.WithClearButton())
	xui.ItemFill(search)
	search.SetValue("shadcn")
	pwd := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithPasswordToggle())
	xui.ItemFill(pwd)
	pwd.SetValue("secret-123")
	counter := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithCounter(120))
	xui.ItemFill(counter)
	counter.SetValue("这是一段用于演示字数统计的文本")

	// ---- 右列 1: 选择控件 ----
	_, v2 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "选择控件", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "复选框 / 单选框 / 开关, 尺寸与状态语义逐条对齐 shadcn", v2.Handle))

	xui.ItemFill(hint(v2.Handle, "复选框"))
	c1 := ui.NewCheckbox(0, 0, 0, 0, "接收邮件通知", v2.Handle)
	c1.SetChecked(true)
	xui.ItemFill(c1)
	xui.ItemFill(ui.NewCheckbox(0, 0, 0, 0, "公开展示个人主页", v2.Handle))
	c3 := ui.NewCheckbox(0, 0, 0, 0, "不可勾选", v2.Handle)
	xui.ItemFill(c3)
	c3.Enable(false)

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "单选框"))
	ra := ui.NewRadio(0, 0, 0, 0, "仅错误消息", v2.Handle)
	xui.ItemFill(ra)
	xui.ItemFill(ui.NewRadio(0, 0, 0, 0, "所有消息", v2.Handle))
	ra.SetCheck(true)
	rd := ui.NewRadio(0, 0, 0, 0, "不可选", v2.Handle)
	xui.ItemFill(rd)
	rd.Enable(false)

	xui.ItemFill(ui.NewSeparator(0, 0, 0, v2.Handle))

	xui.ItemFill(hint(v2.Handle, "开关(default / sm)"))
	rs := xui.HStack(v2.Handle, 24, 12)
	ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchChecked(true))
	ui.NewSwitch(0, 0, rs.Handle)
	ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchSize(ui.SwitchSm), ui.WithSwitchChecked(true))
	swDis := ui.NewSwitch(0, 0, rs.Handle, ui.WithSwitchSize(ui.SwitchSm))
	swDis.Enable(false)

	// ---- 左列 2: 滑动条 ----
	_, v3 := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滑动条", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "纯自绘: 点击跳转 / 拖拽 / 键盘左右 ±5%, hover 与拖动显示光环", v3.Handle))

	xui.ItemFill(hint(v3.Handle, "连续(0~100)"))
	rowS := xui.HStack(v3.Handle, 24, 8)
	sl := ui.NewSlider(0, 0, 200, 0, rowS.Handle)
	xui.ItemWidth(sl, 218) // 220 = 可见轨道 200 + 两端各 9(滑块 ring 的绘制余量)
	slVal := ui.NewLabel(0, 0, 40, 0, "40", rowS.Handle).SetMuted()
	xui.ItemWidth(slVal, 40)
	sl.SetValue(40)
	sl.OnChange(func(v float64) { slVal.SetText(strconv.Itoa(int(v + 0.5))).Redraw() })

	xui.ItemFill(hint(v3.Handle, "范围步进(0~10, step 1)"))
	rowS2 := xui.HStack(v3.Handle, 24, 8)
	st := ui.NewSlider(0, 0, 200, 0, rowS2.Handle,
		ui.WithSliderRange(0, 10), ui.WithSliderStep(1))
	xui.ItemWidth(st, 218)
	stVal := ui.NewLabel(0, 0, 40, 0, "3", rowS2.Handle).SetMuted()
	xui.ItemWidth(stVal, 40)
	st.SetValue(3)
	st.OnChange(func(v float64) { stVal.SetText(strconv.Itoa(int(v + 0.5))).Redraw() })

	// ---- 右列 2: 下拉框 ----
	_, v4 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "下拉框", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生 ComboBox 行为 + 自绘弹层(item-aligned 定位, 盖住触发区)", v4.Handle))

	xui.ItemFill(hint(v4.Handle, "默认尺寸"))
	combo := ui.NewComboBox(0, 0, 0, 0, v4.Handle,
		[]string{"开发环境", "测试环境", "预发环境", "灰度环境", "生产环境", "本地调试"},
		ui.WithComboPlaceholder("选择环境"))
	xui.ItemFill(combo)
	combo.SetSelected(1)
	tip.Bind(combo.Handle, "切换运行时环境")

	xui.ItemFill(hint(v4.Handle, "sm 尺寸"))
	xui.ItemFill(ui.NewComboBox(0, 0, 0, 0, v4.Handle,
		[]string{"开发环境", "测试环境", "生产环境"},
		ui.WithComboSize(ui.ComboSizeSm), ui.WithComboPlaceholder("sm 尺寸")))

	// ---- 第二行: Field 表单布局 ----
	rField := row(body.Handle, 0, gap)

	_, vf := autoCard(col(rField.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "表单字段(Field)", vf.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"标签 / 控件 / 说明 / 错误按 12px 间距成组; 标题与说明由 Field 统一持有", vf.Handle))

	fMail := ui.NewField(0, 0, 0, 0, vf.Handle)
	xui.ItemFill(fMail)
	fMail.Label("邮箱地址")
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, fMail.Handle, ui.WithPlaceholder("m@example.com")))
	fMail.Description("用于接收登录链接与安全通知。")

	fName := ui.NewField(0, 0, 0, 0, vf.Handle, ui.WithFieldInvalid())
	xui.ItemFill(fName)
	fName.Label("用户名")
	name := ui.NewInput(0, 0, 0, 0, fName.Handle, ui.WithInputInvalid())
	xui.ItemFill(name)
	name.SetValue("ab")
	fName.Description("3-20 个字符, 支持字母、数字与下划线。")
	nameErr := ui.NewFieldError(0, 0, 0, 0, fName.Handle,
		"用户名至少 3 个字符", "不能包含空格", "用户名至少 3 个字符")
	xui.ItemFill(nameErr)

	toggleInvalid := ui.NewButton(0, 0, 0, 0, "切换校验状态", vf.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithSize(ui.ButtonSizeSm))
	xui.ItemWidth(toggleInvalid, 120)
	toggleInvalid.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		invalid := !fName.Invalid()
		fName.SetInvalid(invalid)
		name.SetInvalid(invalid)
		if invalid {
			nameErr.SetErrors("用户名至少 3 个字符", "不能包含空格")
		} else {
			nameErr.SetErrors()
		}
		return 0
	})

	fDisplay := ui.NewField(0, 0, 0, 0, vf.Handle)
	xui.ItemFill(fDisplay)
	fDisplay.Label("显示名称")
	content := fDisplay.Content()
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "团队可见", content.Handle).SetMedium())
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "其他成员能看到这个名字。", content.Handle).SetMuted())
	displayName := ui.NewInput(0, 0, 0, 0, fDisplay.Handle)
	xui.ItemFill(displayName)
	displayName.SetValue("twgh")

	_, vf2 := autoCard(col(rField.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "分组与横排(FieldSet / FieldGroup)", vf2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"FieldSet 24 / FieldGroup 28; 横排标签吃剩余宽度、控件靠右", vf2.Handle))

	set := ui.NewFieldSet(0, 0, 0, 0, vf2.Handle)
	xui.ItemFill(set)
	ui.NewFieldLegend(0, 0, 0, 0, "通知设置", set.Handle)
	grp := ui.NewFieldGroup(0, 0, 0, 0, set.Handle)
	xui.ItemFill(grp)

	fMailNotify := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fMailNotify)
	fMailNotify.Label("邮件通知")
	ui.NewSwitch(0, 0, fMailNotify.Handle, ui.WithSwitchChecked(true))

	fPushNotify := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fPushNotify)
	fPushNotify.Label("推送通知")
	ui.NewSwitch(0, 0, fPushNotify.Handle)

	ui.NewFieldSeparator(0, 0, 0, 0, grp.Handle, "或")

	fUpdates := ui.NewField(0, 0, 0, 0, grp.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fUpdates)
	cbUpdates := ui.NewCheckbox(0, 0, 0, 0, "", fUpdates.Handle)
	xui.ItemWidth(cbUpdates, 24) // 只剩指示器, 标题/说明放进 FieldContent(同上游 checkbox 字段写法)
	updContent := fUpdates.Content()
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "接收产品更新", updContent.Handle).SetMedium())
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "每月一封, 随时可退订。", updContent.Handle).SetMuted())

	ui.NewFieldLegend(0, 0, 0, 0, "账户", set.Handle, ui.WithFieldLegendVariant(ui.FieldLegendLabel))
	grp2 := ui.NewFieldGroup(0, 0, 0, 0, set.Handle)
	xui.ItemFill(grp2)
	fLang := ui.NewField(0, 0, 0, 0, grp2.Handle, ui.WithFieldOrientation(ui.FieldHorizontal))
	xui.ItemFill(fLang)
	fLang.Label("界面语言")
	lang := ui.NewComboBox(0, 0, 0, 0, fLang.Handle, []string{"简体中文", "English", "日本語"})
	xui.ItemWidth(lang, 160)
	lang.SetSelected(0)

}

// ===== 页 4: 表单2(日期与时间 / 月历 / 范围与预设) =====

func buildForm2Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := col(r.Handle, 1, gap)
	colR := col(r.Handle, 1, gap)

	// ---- 左列 1: 日期与时间 ----
	_, v5 := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "日期与时间", v5.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "触发区自绘(outline 按钮), 弹层是自建阴影窗: 月历全自绘, 时间框两列滚动列表", v5.Handle))

	xui.ItemFill(hint(v5.Handle, "日期框(点击弹出月历卡片)"))
	dp := ui.NewDatePicker(0, 0, 0, 0, v5.Handle, ui.WithDatePickerPlaceholder("选择日期"))
	xui.ItemFill(dp)
	dpVal := ui.NewLabel(0, 0, 0, 0, "", v5.Handle).SetMuted()
	dpVal.SetText("未选择")
	xui.ItemFill(dpVal)
	dp.OnChange(func(t time.Time) {
		dpVal.SetText("选择: " + t.Format("2006年1月2日")).Redraw()
	})

	xui.ItemFill(hint(v5.Handle, "时间框(两列列表: 时 / 分)"))
	// 直接挂在卡片内容区, 不再套 HStack: 隐藏页首次显示时, 嵌套水平容器的
	// 元素矩形会停在旧位置, 时间框的命中区与画面错位(需滚一下滚轮才恢复)。
	tp := ui.NewTimePicker(0, 0, 0, 0, v5.Handle, ui.WithTimePickerValue(9, 30))
	xui.ItemFill(tp)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "时间框为两列滚动列表, 选择即生效", v5.Handle).SetMuted())

	// ---- 右列 1: 月历卡片 ----
	_, v6 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "月历卡片", v6.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"标题年月可下拉切换; 周末禁用(灰显不可点); 页脚 = 内置「今天」+ 自定义「清除」", v6.Handle))
	// 默认尺寸已是"格子 36 / 字号 16"的比例; 元素再放大时内部等比缩放
	var cal *ui.Calendar
	cal = ui.NewCalendar(0, 0, 0, 0, v6.Handle,
		ui.WithCalendarSelected(time.Now()),
		ui.WithCalendarDisabled(ui.CalendarDisableWeekend),
		ui.WithCalendarFooterToday(),
		ui.WithCalendarFooter(ui.CalendarFooterItem{Text: "清除", OnClick: func() {
			cal.SetSelected(time.Time{})
		}}))
	cal.OnSelect(func(t time.Time) { tip.Bind(cal.Handle, t.Format("2006年1月2日")) })

	// ---- 第二行: 月历 / 日期框的 shadcn 文档能力 ----
	r2 := row(body.Handle, 0, gap)
	colL2 := col(r2.Handle, 1, gap)
	colR2 := col(r2.Handle, 1, gap)

	// ---- 左列 1: 日期范围框 ----
	_, v7 := autoCard(colL2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "日期范围框", v7.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"两次点击落区间; 弹层并排两个月份 + 预设按钮; 起点未落终点时保持打开", v7.Handle))
	xui.ItemFill(hint(v7.Handle, "范围选择(预设: 今天 / 未来 7 天 / 未来 30 天)"))
	now := time.Now()
	rf := time.Date(now.Year(), time.January, 20, 0, 0, 0, 0, time.Local)
	rt := rf.AddDate(0, 0, 20)
	dr := ui.NewDatePicker(0, 0, 0, 0, v7.Handle,
		ui.WithDatePickerRangeValue(rf, rt),
		ui.WithDatePickerYearRange(2000, now.Year()+1),
		ui.WithDatePickerPresets(
			ui.CalPreset("今天", 0, 0),
			ui.CalPreset("未来 7 天", 0, 7),
			ui.CalPreset("未来 30 天", 0, 30)))
	xui.ItemFill(dr)
	drVal := ui.NewLabel(0, 0, 0, 0, "范围: "+dr.Text(), v7.Handle).SetMuted()
	xui.ItemFill(drVal)
	dr.OnRangeChange(func(from, to time.Time) {
		if to.IsZero() {
			drVal.SetText("起点: " + from.Format("2006年1月2日") + " · 再选终点").Redraw()
			return
		}
		drVal.SetText("范围: " + from.Format("2006年1月2日") + " - " + to.Format("2006年1月2日")).Redraw()
	})

	// ---- 左列 2: 可输入日期框 ----
	_, v8 := autoCard(colL2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "可输入日期框", v8.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"InputGroup 形态: 键入 2026-10-22 / 2026年10月22日 / 明天 / 3天后, Enter 或失焦回写; 右侧日历图标打开弹层", v8.Handle))
	xui.ItemFill(hint(v8.Handle, "出生日期(年份范围 1900 - 今天, 弹层带年月下拉)"))
	dp2 := ui.NewDatePicker(0, 0, 0, 0, v8.Handle,
		ui.WithDatePickerEditable(),
		ui.WithDatePickerYearRange(1900, now.Year()),
		ui.WithDatePickerPlaceholder("1990-01-01 或 明天"),
		ui.WithDatePickerValue(time.Date(1990, time.January, 1, 0, 0, 0, 0, time.Local)))
	xui.ItemFill(dp2)
	dp2Val := ui.NewLabel(0, 0, 0, 0, "输入值: "+dp2.Text(), v8.Handle).SetMuted()
	xui.ItemFill(dp2Val)
	dp2.OnChange(func(t time.Time) {
		if t.IsZero() {
			dp2Val.SetText("输入值: (空)").Redraw()
			return
		}
		dp2Val.SetText("输入值: " + t.Format("2006年1月2日")).Redraw()
	})

	// ---- 右列 1: 范围月历(宽度跟随列, 高度按宽度自动配平) ----
	_, v9 := autoCard(colR2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "范围月历", v9.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"两个并排月份 + 范围选择; 今天之后的日期禁用(CalendarDisableAfter)", v9.Handle))
	calRange := ui.NewCalendar(0, 0, 0, 0, v9.Handle,
		ui.WithCalendarMonths(2),
		ui.WithCalendarRange(),
		ui.WithCalendarSelectedRange(rf, rt),
		ui.WithCalendarDisabled(ui.CalendarDisableAfter(now)))
	xui.ItemFill(calRange)
	rangeInfo := ui.NewLabel(0, 0, 0, 0, "范围: "+dateRangeText(rf, rt), v9.Handle).SetMuted()
	xui.ItemFill(rangeInfo)
	calRange.OnRangeSelect(func(from, to time.Time) {
		rangeInfo.SetText("范围: " + dateRangeText(from, to)).Redraw()
	})

	// ---- 右列 2: 周数 / 预订日期 / 预设 ----
	_, v10 := autoCard(colR2.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "周数与预订日期", v10.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"ISO 周数列(周一开头); 预订日期删除线 + 禁用不可点; 页脚预设按钮", v10.Handle))
	booked := make([]time.Time, 0, 5)
	for i := 0; i < 5; i++ {
		booked = append(booked, now.AddDate(0, 0, 3+i))
	}
	calWeek := ui.NewCalendar(0, 0, 0, 0, v10.Handle,
		ui.WithCalendarWeekNumbers(),
		ui.WithCalendarSelected(now),
		ui.WithCalendarDisabled(ui.CalendarDisableDates(booked...)),
		ui.WithCalendarStrike(ui.CalendarStrikeDates(booked...)),
		ui.WithCalendarPresets(
			ui.CalPreset("今天", 0, 0),
			ui.CalPreset("明天", 1, 0),
			ui.CalPreset("一周后", 7, 0)))
	// 周数日历保持默认宽度: 被卡片拉满会把格子/字号一起放大。
	xui.ItemWidth(calWeek, ui.CalWidthForWeekNumbers(1))
}

// dateRangeText 是 demo 里展示范围用的短文本。
func dateRangeText(from, to time.Time) string {
	if from.IsZero() {
		return "未选择"
	}
	if to.IsZero() {
		return from.Format("2006年1月2日") + " - 再选终点"
	}
	return from.Format("2006年1月2日") + " - " + to.Format("2006年1月2日")
}

// ===== 页 5: 导航(标签页 / 分页 / 菜单) =====

func buildNavPage(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: 标签页 ----
	_, v := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "标签页", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "default / line 两个变体 + 纵向取向; 标签等宽、选中态走 background", v.Handle))

	xui.ItemFill(hint(v.Handle, "default"))
	tabs := ui.NewTabs(0, 0, 240, v.Handle, []string{"账户", "密码", "通知"})
	xui.ItemWidth(tabs, 240)
	panel := ui.NewLabel(0, 0, 0, 0, "账户设置面板", v.Handle).SetMuted()
	xui.ItemFill(panel)
	tabs.OnChange(func(i int32) {
		panel.SetText([]string{"账户设置面板", "密码设置面板", "通知设置面板"}[i]).Redraw()
	})

	xui.ItemFill(hint(v.Handle, "line"))
	// Tabs 的标签宽度按创建时给的 cx 等分, 不能被布局拉伸, 所以用固定宽度
	lineTabs := ui.NewTabs(0, 0, 200, v.Handle, []string{"概览", "详情"},
		ui.WithTabsVariant(ui.TabsLine), ui.WithTabsActive(1))
	xui.ItemWidth(lineTabs, 200)

	xui.ItemFill(hint(v.Handle, "纵向"))
	vTabs := ui.NewTabs(0, 0, 96, v.Handle, []string{"账户", "密码", "通知"}, ui.WithTabsVertical())
	xui.ItemWidth(vTabs, 96)

	// ---- 列 2: 下拉菜单 ----
	_, v2 := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "下拉菜单", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生菜单承载行为 + 窗口级自绘皮肤", v2.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0,
		"条目 / 勾选 / 单选组 / 子菜单 / 快捷键 / 分组标签 / 禁用 / 破坏性", v2.Handle).SetMuted())
	rm := xui.HStack(v2.Handle, 36, 8)
	menuBtn := ui.NewButton(0, 0, 0, 0, "打开菜单", rm.Handle,
		ui.WithVariant(ui.ButtonOutline), ui.WithButtonHasPopup(),
		ui.WithButtonTrailingIcon(icons.IconChevronDown))
	xui.ItemWidth(menuBtn, 120)
	menuBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		// PopupAtElement 阻塞到菜单关闭; 期间保持 aria-expanded 的底色, 且不整体下沉。
		menuBtn.SetExpanded(true)
		showDemoMenu(hEle)
		menuBtn.SetExpanded(false)
		return 0
	})
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "条目 hover 走 bg-accent, 与表格行 / 树行同一口径", v2.Handle).SetMuted())

	// ---- 列 3: 右键菜单 ----
	_, v3 := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "右键菜单", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"右键触发区: 含子元素, 每次弹出按状态重建", v3.Handle))
	contextTrigger(v3.Handle)
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "前置图标 / inset / 子菜单 / 勾选 / 单选 / 破坏性", v3.Handle).SetMuted())

	// ---- 第二行: 分页 ----
	_, v5 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "分页", v5.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"折叠采用恒定 7 槽位: 首段/中段/末段占位宽度一致, 连点同一坐标不会点空", v5.Handle))

	pagInfo := ui.NewLabel(0, 0, 0, 0, "", v5.Handle).SetMuted()
	xui.ItemFill(pagInfo)
	pag := ui.NewPagination(0, 0, 0, 0, v5.Handle,
		ui.WithPrevText("上一页"), ui.WithNextText("下一页"))
	xui.ItemFill(pag)
	pagInfo.SetText("共 120 条 · 每页 10 条 · 当前第 5 页")
	pag.SetTotal(120)
	pag.SetPageSize(10)
	pag.SetPage(5)
	pag.OnChange(func(p int) {
		pagInfo.SetText(fmt.Sprintf("共 120 条 · 每页 10 条 · 当前第 %d 页", p))
		pagInfo.Redraw(false)
	})

	// ---- 面包屑 ----
	_, vbc := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "面包屑", vbc.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Breadcrumb: 链接 muted → 悬停过渡到前景色; 分隔符 14, 条目间距 6, 列表间距 10 并自动换行", vbc.Handle))

	bcInfo := ui.NewLabel(0, 0, 0, 20, "点击任意链接查看回调", vbc.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(bcInfo)

	xui.ItemFill(hint(vbc.Handle, "完整路径"))
	bc := ui.NewBreadcrumb(0, 0, 0, 0, vbc.Handle)
	xui.ItemFill(bc)
	addBCLink := func(text string) {
		it := ui.NewBreadcrumbItem(bc.Handle)
		ui.NewBreadcrumbLink(it.Handle, text).OnClick(func() {
			bcInfo.SetText("点击了「" + text + "」").Redraw(false)
		})
	}
	addBCLink("首页")
	ui.NewBreadcrumbSeparator(bc.Handle)
	addBCLink("组件")
	ui.NewBreadcrumbSeparator(bc.Handle)
	itCur := ui.NewBreadcrumbItem(bc.Handle)
	ui.NewBreadcrumbPage(itCur.Handle, "面包屑")

	xui.ItemFill(hint(vbc.Handle, "折叠省略号(36×36, 可自行挂菜单事件)"))
	bc2 := ui.NewBreadcrumb(0, 0, 0, 0, vbc.Handle)
	xui.ItemFill(bc2)
	itHome := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbLink(itHome.Handle, "首页")
	ui.NewBreadcrumbSeparator(bc2.Handle)
	ui.NewBreadcrumbEllipsis(bc2.Handle)
	ui.NewBreadcrumbSeparator(bc2.Handle)
	itSet := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbLink(itSet.Handle, "设置")
	ui.NewBreadcrumbSeparator(bc2.Handle)
	itMe := ui.NewBreadcrumbItem(bc2.Handle)
	ui.NewBreadcrumbPage(itMe.Handle, "个人资料")

	// ---- 第三行: 可折叠面板 ----
	_, v4 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "可折叠面板", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"radix Collapsible: 触发区 + 内容区两种组合; 内容高度 200ms 过渡, 收起后不占布局", v4.Handle))
	ex := row(v4.Handle, 0, 24)
	left := col(ex.Handle, 1, 8)
	buildCollapsibleOrder(left.Handle)
	buildCollapsibleTextTrigger(left.Handle)
	buildCollapsibleTree(col(ex.Handle, 1, 8).Handle)
}

// buildCollapsibleTextTrigger 演示整宽文字触发区(justify-between: 文字靠左、箭头贴右)。
func buildCollapsibleTextTrigger(parent int) {
	xui.ItemFill(hint(parent, "整宽触发区 · 文字靠左 / 箭头贴右"))
	panel := ui.NewCollapsible(0, 0, 0, 0, parent)
	panel.SetSpace(8)
	xui.ItemFill(panel)
	btn := ui.NewButton(0, 0, 0, 0, "高级设置", panel.Handle,
		ui.WithVariant(ui.ButtonGhost),
		ui.WithButtonAlign(ui.ButtonAlignBetween),
		ui.WithButtonTrailingIcon(icons.IconChevronDown))
	xui.ItemFill(btn)
	panel.BindTrigger(btn)
	panel.OnOpenChange(func(open bool) {
		btn.SetTrailingIcon(common.Choose(open, icons.IconChevronUp, icons.IconChevronDown))
	})
	body := panel.ContentStack(8)
	desc := ui.NewLabel(0, 0, 0, 20, "通知与隐私相关的开关都放在这里; 展开/收起走同一个高度过渡。",
		body.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(desc)
	xui.ItemMargin(desc, 10, 0, 10, 0)
}

// buildCollapsibleOrder 是 shadcn collapsible-demo 的订单详情: 图标触发区 + 常显状态行 + 可折叠明细。
func buildCollapsibleOrder(parent int) {
	xui.ItemFill(hint(parent, "订单详情 · 图标触发区"))

	panel := ui.NewCollapsible(0, 0, 0, 0, parent)
	panel.SetSpace(8)
	xui.ItemFill(panel)

	head := widget.NewLayoutEle(0, 0, 0, 33, panel.Handle)
	head.EnableHorizon(true)
	head.EnableLayout(true)
	head.EnableBkTransparent(true)
	head.EnableAutoWrap(false)
	head.SetSpace(8)
	head.SetAlignV(xcc.Layout_Align_Center)
	head.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	head.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 33)
	xui.ItemMargin(head, 10, 0, 10, 0)
	title := ui.NewLabel(0, 0, 0, 20, "订单 #4189", head.Handle).SetFontSize(14).SetMedium()
	xui.ItemWeight(title, 1)
	toggle := ui.NewButton(0, 0, 0, 0, "", head.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeIcon),
		ui.WithIcon(icons.IconChevronsUpDown))
	xui.ItemWidth(toggle, 32)
	panel.BindTrigger(toggle)

	status := borderedBox(panel.Handle, 36, true)
	left := ui.NewLabel(0, 0, 0, 20, "状态", status.Handle).SetFontSize(14).SetMuted()
	xui.ItemWeight(left, 1)
	xui.ItemMarginLeft(left, 16)
	right := ui.NewLabel(0, 0, 0, 20, "已发货", status.Handle).SetFontSize(14).SetMedium()
	xui.ItemWidth(right, 72)
	xui.ItemMarginRight(right, 16)

	body := panel.ContentStack(8)
	addr := borderedBox(body.Handle, 56, false)
	addrTitle := ui.NewLabel(0, 0, 0, 20, "收货地址", addr.Handle).SetFontSize(14).SetMedium()
	xui.ItemFill(addrTitle)
	xui.ItemMargin(addrTitle, 16, 8, 16, 0)
	addrText := ui.NewLabel(0, 0, 0, 20, "100 Market St, San Francisco", addr.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(addrText)
	xui.ItemMargin(addrText, 16, 0, 16, 8)
	xui.ItemMargin(addr, 10, 0, 10, 0)

	items := borderedBox(body.Handle, 56, false)
	itemsTitle := ui.NewLabel(0, 0, 0, 20, "商品", items.Handle).SetFontSize(14).SetMedium()
	xui.ItemFill(itemsTitle)
	xui.ItemMargin(itemsTitle, 16, 8, 16, 0)
	itemsText := ui.NewLabel(0, 0, 0, 20, "2x 头戴式耳机", items.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(itemsText)
	xui.ItemMargin(itemsText, 16, 0, 16, 8)
	xui.ItemMargin(items, 10, 0, 10, 10)
}

// buildCollapsibleTree 演示嵌套折叠(文件夹树)与整宽触发区(文字靠左、图标靠右)。
func buildCollapsibleTree(parent int) {
	xui.ItemFill(hint(parent, "文件树 · 嵌套折叠"))

	tree := ui.NewCollapsible(0, 0, 0, 0, parent)
	tree.SetSpace(2)
	xui.ItemFill(tree)
	compBtn := folderTrigger(tree.Handle, "components")
	tree.BindTrigger(compBtn)
	tree.OnOpenChange(func(open bool) {
		compBtn.SetTrailingIcon(common.Choose(open, icons.IconChevronDown, icons.IconChevronRight))
	})

	compBody := tree.ContentStack(1)
	xui.ItemMarginLeft(compBody, 20)

	uiRoot := ui.NewCollapsible(0, 0, 0, 0, compBody.Handle)
	uiRoot.SetSpace(2)
	xui.ItemFill(uiRoot)
	uiBtn := folderTrigger(uiRoot.Handle, "ui")
	uiRoot.BindTrigger(uiBtn)
	uiRoot.OnOpenChange(func(open bool) {
		uiBtn.SetTrailingIcon(common.Choose(open, icons.IconChevronDown, icons.IconChevronRight))
	})
	uiBody := uiRoot.ContentStack(1)
	xui.ItemMarginLeft(uiBody, 20)
	fileRow(uiBody.Handle, "button.tsx")
	fileRow(uiBody.Handle, "card.tsx")
	fileRow(uiBody.Handle, "dialog.tsx")

	fileRow(compBody.Handle, "login-form.tsx")
	fileRow(compBody.Handle, "register-form.tsx")
}

// folderTrigger 建文件树的文件夹触发区: 幽灵按钮、整宽、内容靠左, 尾随方向箭头。
func folderTrigger(parent int, name string) *ui.Button {
	b := ui.NewButton(0, 0, 0, 0, name, parent,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm),
		ui.WithButtonAlign(ui.ButtonAlignStart),
		ui.WithButtonLeadingIcon(icons.IconFolder),
		ui.WithButtonTrailingIcon(icons.IconChevronRight))
	xui.ItemFill(b)
	return b
}

// fileRow 建文件树里的文件行(幽灵按钮, 内容靠左 + 文件图标)。
func fileRow(parent int, name string) *ui.Button {
	b := ui.NewButton(0, 0, 0, 0, name, parent,
		ui.WithVariant(ui.ButtonGhost), ui.WithSize(ui.ButtonSizeSm),
		ui.WithButtonAlign(ui.ButtonAlignStart),
		ui.WithButtonLeadingIcon(icons.IconFile))
	xui.ItemFill(b)
	return b
}

// borderedBox 建一个 1px Border + rounded-md 的盒子(对应 shadcn 示例里的
// `rounded-md border px-4 py-2`)。horizontal=true 时是横排(内容垂直居中)。
func borderedBox(parent int, h int32, horizontal bool) *widget.LayoutEle {
	box := widget.NewLayoutEle(0, 0, 0, h, parent)
	box.EnableHorizon(horizontal)
	box.EnableLayout(true)
	box.EnableBkTransparent(true)
	box.EnableAutoWrap(false)
	box.SetAlignV(xcc.Layout_Align_Center)
	if horizontal {
		// 横排(状态行)的左右项之间才需要间距; 竖向盒的上下内边距由子项 margin 表达,
		// 再叠一项间距会把底部内边距挤出盒子(文字贴到底边)。
		box.SetSpace(8)
	}
	box.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	box.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	box.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		// 描边沿路径居中, 贴边会被元素裁剪掉一半; 内缩 1px 让 1px 边框完整落在客户区内。
		rc := p.Rc
		rc.Left++
		rc.Top++
		rc.Right--
		rc.Bottom--
		p.StrokeRound(&rc, xui.Current().Border, xui.Current().RadiusMd(), 1)
		return 0
	})
	return box
}

// ===== 页 6: 反馈(对话框 / 提示气泡 / 进度) =====

func buildFeedPage(body *widget.LayoutEle, hWindow int) {
	r := row(body.Handle, 0, gap)

	// ---- 列 1: 对话框 ----
	_, v := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "对话框", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "窗口内覆盖元素: 全窗遮罩(backdrop-blur) + 居中面板 + Tab 焦点陷阱", v.Handle))
	rd := xui.HStack(v.Handle, 36, 8)
	editBtn := ui.NewButton(0, 0, 0, 0, "编辑资料", rd.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(editBtn, 104)
	delBtn := ui.NewButton(0, 0, 0, 0, "删除项目", rd.Handle, ui.WithVariant(ui.ButtonDestructive))
	xui.ItemWidth(delBtn, 104)
	scrollBtn := ui.NewButton(0, 0, 0, 0, "滚动内容", rd.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(scrollBtn, 104)
	buildDialogs(hWindow, editBtn, delBtn, scrollBtn)
	xui.ItemFill(desc(v.Handle, "ESC / 遮罩 / X 关闭, Tab 焦点循环;「滚动内容」头部/页脚固定。"))

	// ---- 列 2: 提示气泡 ----
	_, v2 := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "提示气泡", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "独立透明弹窗承载, 不抢焦点; 延迟 0, 进出场走淡入淡出", v2.Handle))
	rt := xui.HStack(v2.Handle, 36, 8)
	hoverBtn := ui.NewButton(0, 0, 0, 0, "悬浮我", rt.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(hoverBtn, 104)
	tip.Bind(hoverBtn.Handle, "我是提示气泡")
	xui.ItemFill(desc(v2.Handle, "绑定的元素自身有弹层时(如 ComboBox), 弹层打开要主动 HideTooltip。"))

	// ---- 第二行: 提示条 ----
	rAlert := row(body.Handle, 0, gap)

	_, va := autoCard(col(rAlert.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "提示条", va.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"图标 + 标题 + 描述 + 右上动作槽; 标题/描述随宽度折行, 高度按内容自撑", va.Handle))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconCircleCheck),
		ui.WithAlertTitle("Success! Your changes have been saved"),
		ui.WithAlertDescription("This is an alert with icon, title and description.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconInfo),
		ui.WithAlertTitle("This Alert has a title and an icon. No description.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertDescription("This one has a description only. No title. No icon.")))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, va.Handle,
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("This is a very long alert title that demonstrates how the component handles extended text content and potentially wraps across multiple lines"),
		ui.WithAlertDescription("This is an equally long description that contains detailed information about the alert. It wraps at the alert width, and the alert grows with it instead of clipping.")))
	xui.ItemFill(desc(va.Handle, "标题与描述用当前宽度量高: 窗口拉宽/收窄时会重新折行。"))

	_, vb := autoCard(col(rAlert.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "破坏性 / 动作", vb.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"destructive: 标题与图标用破坏性色, 描述 90%; 动作贴右上", vb.Handle))
	xui.ItemFill(ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertVariant(ui.AlertDestructive),
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("Unable to process your payment."),
		ui.WithAlertDescription("Please verify your billing information and try again.")))
	withAction := ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertIcon(icons.IconCircleAlert),
		ui.WithAlertTitle("The selected emails have been marked as spam."),
		ui.WithAlertDescription("This is a very long alert title that demonstrates how the component handles extended text content."))
	xui.ItemFill(withAction)
	ui.NewButton(0, 0, 0, 0, "Undo", withAction.Action().Handle,
		ui.WithSize(ui.ButtonSizeXs))
	withBadge := ui.NewAlert(0, 0, 0, 0, vb.Handle,
		ui.WithAlertIcon(icons.IconInfo),
		ui.WithAlertTitle("A new version is available."),
		ui.WithAlertDescription("Restart the app to apply the update."))
	xui.ItemFill(withBadge)
	badge := ui.NewBadge(0, 0, 0, 0, "New", withBadge.Action().Handle,
		ui.WithBadgeVariant(ui.BadgeSecondary))
	xui.ItemWidth(badge, 44)
	xui.ItemFill(desc(vb.Handle, "动作槽用 Action() 取; 可放 Button / Badge, 顶端对齐、内容自量宽。"))

	// ---- 第三行: 警示对话框 + 对话框变体 ----
	rDlg := row(body.Handle, 0, gap)

	_, vd := autoCard(col(rDlg.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "警示对话框", vd.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"无 X、点遮罩不关、ESC 关闭; sm 居中 + 按钮等宽双列", vd.Handle))
	rad := xui.HStack(vd.Handle, 36, 8)
	dfltBtn := ui.NewButton(0, 0, 0, 0, "默认", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(dfltBtn, 78)
	smBtn := ui.NewButton(0, 0, 0, 0, "小尺寸", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(smBtn, 78)
	mediaBtn := ui.NewButton(0, 0, 0, 0, "带图标", rad.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(mediaBtn, 78)
	destrBtn := ui.NewButton(0, 0, 0, 0, "破坏性", rad.Handle, ui.WithVariant(ui.ButtonDestructive))
	xui.ItemWidth(destrBtn, 78)
	buildAlertDialogs(hWindow, dfltBtn, smBtn, mediaBtn, destrBtn)
	xui.ItemFill(desc(vd.Handle, "打开时焦点落在 Cancel; Tab 只在当前对话框内循环; Action / Cancel 点击后关闭。"))

	_, vdv := autoCard(col(rDlg.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "对话框变体", vdv.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"无关闭钮; 嵌套时只有栈顶响应 ESC, 关闭上层后焦点回到下层", vdv.Handle))
	rv := xui.HStack(vdv.Handle, 36, 8)
	noCloseBtn := ui.NewButton(0, 0, 0, 0, "无关闭钮", rv.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(noCloseBtn, 104)
	nestedBtn := ui.NewButton(0, 0, 0, 0, "嵌套警示", rv.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(nestedBtn, 104)
	buildDialogVariants(hWindow, noCloseBtn, nestedBtn)
	xui.ItemFill(desc(vdv.Handle, "嵌套示例: 点「打开警示」在对话框上再叠一层; 连按 ESC 会先关上层。"))

	// ---- 第四行: 通知 + 进度 ----
	r2 := row(body.Handle, 0, gap)

	// 通知: 右下角堆叠, 4s 自动关闭, 悬停暂停
	_, v3 := autoCard(col(r2.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "通知", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"窗口内覆盖元素, 右下角堆叠; 4s 自动关闭, 悬停全部暂停; 关闭按钮默认隐藏", v3.Handle))
	toaster := ui.NewToaster(hWindow)
	rn1 := xui.HStack(v3.Handle, 36, 8)
	for _, it := range []struct {
		text    string
		variant ui.ToastVariant
	}{{"成功", ui.ToastSuccess}, {"错误", ui.ToastError}, {"警告", ui.ToastWarning}, {"信息", ui.ToastInfo}} {
		it := it
		b := ui.NewButton(0, 0, 0, 0, it.text, rn1.Handle, ui.WithVariant(ui.ButtonOutline))
		xui.ItemWidth(b, 78)
		b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			toaster.Show(ui.ToastOptions{
				Title:       it.text + "通知",
				Description: "这条通知会在 4 秒后自动关闭, 鼠标悬停可暂停倒计时。",
				Variant:     it.variant,
			})
			return 0
		})
	}
	rn2 := xui.HStack(v3.Handle, 36, 8)
	bAction := ui.NewButton(0, 0, 0, 0, "带动作", rn2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bAction, 78)
	bAction.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title:       "已删除项目",
			Description: "「xui-demo」已移入回收站。",
			Action:      "撤销",
			CloseButton: true,
			OnAction:    func() { fmt.Println("toast: 撤销删除") },
			OnClose:     func() { fmt.Println("toast: 关闭") },
		})
		return 0
	})
	bLong := ui.NewButton(0, 0, 0, 0, "长文本", rn2.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(bLong, 78)
	bLong.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		toaster.Show(ui.ToastOptions{
			Title: "同步完成",
			Description: "本次同步共处理 1,284 个文件, 其中 37 个发生冲突已自动合并, " +
				"3 个需要手动处理。",
			Variant:     ui.ToastInfo,
			Duration:    -1,
			CloseButton: true,
		})
		return 0
	})
	xui.ItemFill(desc(v3.Handle, "ToastOptions.Duration 传负数即常驻; Action 点击后默认关闭通知。"))

	_, v4 := autoCard(col(r2.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "进度", v4.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "h-1 圆角轨道 + 指示器(宽度按百分比)", v4.Handle))
	for _, val := range []float64{24, 64, 100} {
		pg := ui.NewProgress(0, 0, 0, 0, v4.Handle)
		xui.ItemFill(pg)
		pg.SetValue(val)
	}

	// ---- 第五行: 空状态 ----
	rEmpty := row(body.Handle, 0, gap)

	_, ve := autoCard(col(rEmpty.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "空状态(图标底板)", ve.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Empty: 图标块 + 标题 + 描述 + 动作区; p-6 / gap-6, 整块居中", ve.Handle))
	empty1 := ui.NewEmpty(0, 0, 0, 0, ve.Handle)
	xui.ItemFill(empty1)
	eh := empty1.Header()
	ui.NewEmptyMedia(eh.Handle, icons.IconFolderCode, ui.WithEmptyMediaVariant(ui.EmptyMediaIcon))
	ui.NewEmptyTitle(eh.Handle, "还没有项目")
	ui.NewEmptyDescription(eh.Handle, "你还没有创建任何项目, 从新建或导入开始。")
	emptyRow := xui.HStack(empty1.Content().Handle, 36, 8)
	xui.ItemWidth(emptyRow, 208) // 两枚 100 宽按钮 + 8 间距: 行宽按内容定, 才能被空状态居中
	createBtn := ui.NewButton(0, 0, 0, 0, "新建项目", emptyRow.Handle)
	xui.ItemWidth(createBtn, 100)
	importBtn := ui.NewButton(0, 0, 0, 0, "导入项目", emptyRow.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(importBtn, 100)

	_, ve2 := autoCard(col(rEmpty.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "空状态(默认媒体 / 尾部链接)", ve2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"默认媒体无底板(24 图标); 自定义元素直接挂 Body, 随整块居中", ve2.Handle))
	empty2 := ui.NewEmpty(0, 0, 0, 0, ve2.Handle)
	xui.ItemFill(empty2)
	eh2 := empty2.Header()
	ui.NewEmptyMedia(eh2.Handle, icons.IconInbox)
	ui.NewEmptyTitle(eh2.Handle, "收件箱为空")
	ui.NewEmptyDescription(eh2.Handle, "新消息会出现在这里。")
	learn := ui.NewButton(0, 0, 0, 0, "了解通知设置", empty2.Body().Handle,
		ui.WithVariant(ui.ButtonLink), ui.WithSize(ui.ButtonSizeSm))
	tip.Bind(learn.Handle, "Empty 不画边框(上游默认也没有可见边框); 需要边框时放进 Card")
}

// ===== 页 7: 数据(表格 / 树) =====

func buildDataPage(body *widget.LayoutEle) {
	r := rowStretch(body.Handle, gap) // 铺满型页面: 表格/树跟随窗口长高(内容区已 SetContentFill)

	// ---- 表格(整行占满剩余高度, 表格跟随窗口长高) ----
	_, v := stretchCard(colStretch(r.Handle, 3, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "数据表格", v.Handle))
	xui.ItemFill(desc(v.Handle, "原生 List 载体 + 全自绘皮肤: 表头 / 行底 / 行边框 / 滚动条。"+
		"行悬浮 hover:bg-muted/50, 选中的行 bg-muted, 最后一行不画底边。"))
	xui.ItemFillHeight(v) // 内容区占满卡片剩余高度
	table := ui.NewTable(0, 0, 0, 0, v.Handle, []ui.TableColumn{
		{Title: "客户", Width: 200},
		{Title: "状态", Width: 120},
		{Title: "金额", Width: 120},
		{Title: "日期"}, // Width<=0: 占满剩余宽度
	})
	xui.ItemFill(table)
	xui.ItemFillHeight(table) // 表格占满内容区里"标题/说明之外"的剩余高度
	for _, cells := range [][4]string{
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
		table.AddRow(cells[0], cells[1], cells[2], cells[3])
	}
	table.Select(1)
	table.OnSelect(func(row int32) {
		fmt.Printf("选中行: %d\n", row)
	})

	// ---- 树 ----
	_, v2 := stretchCard(colStretch(r.Handle, 2, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "树形视图", v2.Handle))
	xui.ItemFill(desc(v2.Handle, "原生 Tree 载体 + 全自绘皮肤: 缩进参考线 / 旋转 chevron / 行悬浮与选中 bg-accent。"+
		"点击 chevron 或文字选中。"))
	xui.ItemFillHeight(v2)
	demoTree = ui.NewTree(0, 0, 0, 0, v2.Handle)
	xui.ItemFill(demoTree)
	xui.ItemFillHeight(demoTree)
	demoTreeSel = buildDemoTree(demoTree)
	demoTree.OnSelect(func(id int32) {
		fmt.Printf("选中项: %s\n", demoTree.ItemText(id))
	})
}

// ===== 页 8: 图标(icons 资源子包) =====
// 网格是"一个格子一个自绘元素"(各自带 hover 态), 翻页只改格子里的名字 + 重绘,
// 不销毁元素(见 DESIGN 第八节 44)。格子默认只画图标: 悬浮出提示气泡显示名字,
// 点击把名字复制到剪贴板。

func buildIconPage(body *widget.LayoutEle) {
	_, v := autoCard(body.Handle, 14)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "图标", v.Handle))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 40,
		"运行时自带 lucide 图标资源(lucide-static "+icons.Version+", 共 "+
			strconv.Itoa(icons.Count())+" 个)。\n用法: p.Icon(icons.Path(\"star\"), &rc, color, 2); "+
			"悬浮图标看名字, 点击复制名字。", v.Handle).SetMuted().SetWrap())
	// 内容区水平居中: 固定尺寸的图标网格才会居中(Fill 宽的子项不受影响)
	v.SetAlignH(xcc.Layout_Align_Center)

	// 搜索行: 输入框(定宽) + 结果计数 + 复制反馈(两份等宽, 各占剩余宽度一半)
	searchRow := xui.HStack(v.Handle, 32, 8)
	iconSearch = ui.NewInput(0, 0, 0, 0, searchRow.Handle,
		ui.WithPlaceholder("搜索图标名, 如 arrow / star / chevron"))
	xui.ItemWidth(iconSearch, 260)
	iconInfo = ui.NewLabel(0, 0, 0, 0, "", searchRow.Handle).SetMuted()
	xui.ItemWeight(iconInfo, 1)
	iconCopied = ui.NewLabel(0, 0, 0, 0, "", searchRow.Handle).SetMuted()
	xui.ItemWeight(iconCopied, 1)

	// 分页(shadcn pagination: 上一页/页码/省略号/下一页, 内容居中)
	iconPag = ui.NewPagination(0, 0, 0, 0, v.Handle,
		ui.WithPrevText("上一页"), ui.WithNextText("下一页"))
	xui.ItemFill(iconPag)
	iconPag.OnChange(func(page int) { showIconPage(page - 1) })

	// 网格容器尺寸在创建时就给足(格子是固定尺寸), 不需要声明布局尺寸
	grid := widget.NewElement(0, 0, int32(iconCols*iconTileW), int32(iconRows*iconTileH), v.Handle)
	grid.EnableBkTransparent(true).EnableDrawBorder(false).EnableFocus(false) // 纯容器: 不画底(暗色下会是一块白)
	iconTiles = iconTiles[:0]
	for r := 0; r < iconRows; r++ {
		for c := 0; c < iconCols; c++ {
			iconTiles = append(iconTiles,
				newIconTile(int32(c)*iconTileW, int32(r)*iconTileH, grid.Handle))
		}
	}

	iconSearch.OnChange(func(text string) {
		filterIcons(text)
		iconCopied.SetText("")
		iconCopied.Redraw(false)
		showIconPage(0)
	})
	filterIcons("")
	showIconPage(0)
}

// ===== 图标总览页的状态与绘制 =====

const (
	iconCols  = 9
	iconRows  = 4
	iconTileW = 77 // 696 / 9
	iconTileH = 60
)

var (
	iconTiles    []*iconTile // 网格里复用的格子(翻页只换名字)
	iconFiltered []string    // 当前筛选结果(图标名, 升序)
	iconPageIdx  int
	iconInfo     *ui.Label      // 结果计数 / 当前页
	iconCopied   *ui.Label      // 复制反馈
	iconSearch   *ui.Input      // 搜索框
	iconPag      *ui.Pagination // 分页组件
)

// iconTile 是网格里的一格: 只画图标(名字走悬浮提示), 悬浮 bg-accent, 点击复制名字。
type iconTile struct {
	ele   *widget.Element
	name  string
	hover bool
}

func newIconTile(x, y int32, parent int) *iconTile {
	ele := widget.NewElement(x, y, iconTileW, iconTileH, parent)
	if ele == nil {
		return nil
	}
	t := &iconTile{ele: ele}
	ele.EnableBkTransparent(true)
	ele.AddEvent_Paint(t.onPaint)
	// 只在 hover 状态**变化**时重绘: MouseStay 是"鼠标在元素内移动"就会来一发的
	// (按下不判状态就会变成每个鼠标移动事件都触发一次重绘, 45 个格子扫一遍很难看)
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		if t.hover {
			return 0
		}
		t.hover = true
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		if !t.hover {
			return 0
		}
		t.hover = false
		ele.Redraw(false)
		return 0
	})
	// 点击把图标名复制到剪贴板(xcgui 自带的剪贴板封装)
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress()
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if t.name == "" {
			return 0
		}
		if err := wutil.SetClipboardText(t.name); err != nil {
			iconCopied.SetText("复制失败: " + err.Error())
			iconCopied.Redraw(false)
			return 0
		}
		iconCopied.SetText("已复制: " + t.name)
		iconCopied.Redraw(false)
		return 0
	})
	return t
}

// setName 换格子显示的图标(空串 = 本页没有这么多图标), 并同步悬浮提示的文字。
func (t *iconTile) setName(name string) {
	if t.name == name {
		return
	}
	t.name = name
	if tip != nil {
		// 悬浮气泡显示完整图标名(名字每次翻页/筛选都会变, Bind 可重复调用改文字)
		tip.Bind(t.ele.Handle, name)
	}
	t.ele.Redraw(false)
}

func (t *iconTile) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	*pbHandled = true
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	th := xui.Current()
	// 透明子元素重绘不会擦底: 先铺宿主底色(卡片), 否则 hover 离开后残留高亮块(见 DESIGN 第八节 28)
	p.FillRound(&p.Rc, th.Card, 0)
	if hv := xui.Hover(hEle, t.hover); hv > 0 {
		p.FillRound(&p.Rc, th.Accent.ScaleAlpha(hv), th.RadiusSm())
	}
	if t.name == "" {
		return 0 // 空格子(本页没有这么多图标)
	}
	// 只画图标(名字不画: 悬浮出提示气泡, 点击复制), 居中比无名字时大一号
	const iconSize = 28
	iconRc := xc.RECT{
		Left:   p.Rc.Left + (p.W()-iconSize)/2,
		Top:    p.Rc.Top + (p.H()-iconSize)/2,
		Right:  p.Rc.Left + (p.W()-iconSize)/2 + iconSize,
		Bottom: p.Rc.Top + (p.H()-iconSize)/2 + iconSize,
	}
	p.Icon(icons.Path(t.name), &iconRc, th.Foreground, 2)
	return 0
}

// filterIcons 按关键字过滤图标名(大小写不敏感, 空关键字 = 全部), 结果保持名字升序。
func filterIcons(q string) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		iconFiltered = icons.Names()
		return
	}
	out := make([]string, 0, 64)
	for _, n := range icons.Names() {
		if strings.Contains(strings.ToLower(n), q) {
			out = append(out, n)
		}
	}
	iconFiltered = out
}

// showIconPage 翻到第 idx 页(越界夹紧), 名字取自筛选结果(按名字升序)。
func showIconPage(idx int) {
	if len(iconTiles) == 0 {
		return
	}
	names := iconFiltered
	pages := (len(names) + len(iconTiles) - 1) / len(iconTiles)
	if pages < 1 {
		pages = 1
	}
	if idx < 0 {
		idx = 0
	}
	if idx > pages-1 {
		idx = pages - 1
	}
	iconPageIdx = idx
	base := idx * len(iconTiles)
	for i, t := range iconTiles {
		if base+i < len(names) {
			t.setName(names[base+i])
		} else {
			t.setName("")
		}
	}
	if iconInfo != nil {
		if len(names) == icons.Count() {
			iconInfo.SetText(fmt.Sprintf("共 %d 个图标 · 第 %d/%d 页", len(names), idx+1, pages))
		} else {
			iconInfo.SetText(fmt.Sprintf("筛选出 %d/%d 个 · 第 %d/%d 页",
				len(names), icons.Count(), idx+1, pages))
		}
		// 原生按钮改文字不会自己失效, 得手动重绘(与 SetIcon 后 w.Redraw(false) 同理)
		iconInfo.Redraw(false)
	}
	if iconPag != nil {
		// 分页组件不做数据分页: 调用方把总数/每页数同步进去, 再同步当前页
		// (SetPage 不触发 OnChange, 不会回流到 showIconPage 造成递归)
		iconPag.SetTotal(len(names))
		iconPag.SetPageSize(len(iconTiles))
		iconPag.SetPage(idx + 1)
	}
}

// buildDemoTree 填充演示树(文件树形态, 与 shadcn App.tsx 对照卡同构), 返回默认选中项。
func buildDemoTree(t *ui.Tree) int32 {
	src := t.AddRoot("src")
	comp := t.AddChild(src, "components")
	t.AddChild(comp, "button.go")
	t.AddChild(comp, "table.go")
	treeItem := t.AddChild(comp, "tree.go")
	demo := t.AddChild(src, "demo")
	t.AddChild(demo, "main.go")
	t.AddChild(src, "go.mod")
	docs := t.AddRoot("docs")
	t.AddChild(docs, "DESIGN.md")
	t.AddChild(docs, "HANDOFF.md")
	t.AddRoot("README.md")
	t.Expand(comp, true)
	t.Expand(src, true)
	t.Select(treeItem)
	return treeItem
}

// demo 菜单状态(菜单是一次性对象, 状态由调用方持有, 每次弹出重建)
var (
	demoLineNo    = true
	demoThemeG1   = true  // 单选组: 浅色
	demoThemeG2   = false // 单选组: 深色
	demoBookmarks = true
	demoURLs      = false
	demoPerson    = int32(1) // 右键菜单单选组: 1 张伟 / 2 李娜
)

// showDemoMenu 弹出演示菜单, 覆盖全部条目类型。
func showDemoMenu(hEle int) {
	m := ui.NewMenu()
	// 明确指定弹层宽度; 不加 Width 时组件默认 = max(触发区宽, 128)。
	m.Width(224)
	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuOnSelect(func() {}))
	m.Item("粘贴", ui.MenuShortcut("Ctrl+V"))
	m.Separator()
	m.Label("外观")
	m.Check("显示行号", demoLineNo, ui.MenuOnCheck(func(on bool) { demoLineNo = on }))
	m.Item("更多外观设置…", ui.MenuInset()) // 文字左缩进, 与勾选条目的文字对齐(shadcn inset)
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

// contextTrigger 建右键触发区: border-dashed 虚线框 + 居中两行提示
// (对应 shadcn ContextMenuDemo 的 "Right click here" 虚线区域)。
func contextTrigger(parent int) *widget.Element {
	var hovered, opened bool
	box := widget.NewElement(0, 0, 0, 150, parent)
	box.EnableBkTransparent(true)
	xui.ItemFill(box)
	xui.ItemHeight(box, 150)
	box.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		hovered = true
		box.Redraw(true) // 立即重绘一次, 让 xui.Hover 在 paint 里启动过渡(同 Badge/Slider)
		return 0
	})
	box.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		hovered = false
		box.Redraw(true)
		return 0
	})
	box.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		// 触发区给 hover / 展开两档反馈(对应按钮的 hover / aria-expanded):
		// 右键本身就是"点在一个区域上", 没有反馈的话菜单像凭空出现。
		hv := xui.Hover(hEle, hovered)
		ov := xui.HoverItem(hEle, 0, opened)
		// 透明元素重绘不擦底(见 DESIGN 28): 先铺宿主卡片底色, 否则悬停淡出会留旧像素。
		p.FillRound(&p.Rc, t.Card, t.RadiusMd())
		// hover = bg-accent(与 ghost 按钮同一档); 打开态继续满色, 靠边框变 Ring 区分。
		p.FillRound(&p.Rc, t.Accent.ScaleAlpha(hv+(1-hv)*ov), t.RadiusMd())
		rc := p.Rc
		// 边框线宽 1, 沿路径居中; 内缩 1px 让虚线完整落在客户区内(同 borderedBox)。
		rc.Left++
		rc.Top++
		rc.Right--
		rc.Bottom--
		d := p.Draw
		// 4px 线 + 3px 间隔的短划线(shadcn border-dashed 的观感;
		// xcgui 的 Dottedline 是 1px 点线, 在这个尺寸下几乎看不见)。
		d.SetLineWidth(1)
		mix := hv
		if ov > mix {
			mix = ov
		}
		d.SetBrushColor(uint32(xui.Fade(t.Border, t.Ring, mix)))
		for x := rc.Left; x < rc.Right; x += 7 {
			x2 := x + 4
			if x2 > rc.Right {
				x2 = rc.Right
			}
			d.DrawLine(x, rc.Top, x2, rc.Top)
			d.DrawLine(x, rc.Bottom, x2, rc.Bottom)
		}
		for y := rc.Top; y < rc.Bottom; y += 7 {
			y2 := y + 4
			if y2 > rc.Bottom {
				y2 = rc.Bottom
			}
			d.DrawLine(rc.Left, y, rc.Left, y2)
			d.DrawLine(rc.Right, y, rc.Right, y2)
		}

		mid := (rc.Top + rc.Bottom) / 2
		line1 := xc.RECT{Left: rc.Left, Top: mid - 22, Right: rc.Right, Bottom: mid}
		// 对齐必须在 SetFont 之后设置: XDraw_SetFont 会重置文本对齐状态,
		// 先设 align 再换字体会让这一行退回默认的左对齐(实测)。
		d.SetFont(xui.FontHandle(t.FontDefault))
		p.Text("在此处右键", &line1, t.Foreground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		line2 := xc.RECT{Left: rc.Left, Top: mid + 2, Right: rc.Right, Bottom: mid + 20}
		p.Text("ContextMenuTrigger · right click", &line2, t.MutedForeground,
			xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
		return 0
	})
	cm := ui.NewContextMenu(box.Handle, showDemoContextMenu)
	cm.OnOpenChange(func(open bool) {
		opened = open
		box.Redraw(true) // 阻塞式弹层: 先把这个展开态画出来
	})
	return box
}

// showDemoContextMenu 右键菜单示例, 与 shadcn ContextMenuDemo 同构:
// 条目 + 快捷键 + 前置图标 + SubTrigger inset + 勾选 + 单选组 + 破坏性。
func showDemoContextMenu(m *ui.Menu) {
	m.Width(208) // w-52
	m.Item("后退", ui.MenuShortcut("Ctrl+["), ui.MenuIcon(icons.IconArrowLeft))
	m.Item("前进", ui.MenuShortcut("Ctrl+]"), ui.MenuIcon(icons.IconArrowRight), ui.MenuDisabled())
	m.Item("重新加载", ui.MenuShortcut("Ctrl+R"), ui.MenuIcon(icons.IconRotateCw))
	m.Sub("更多工具", func(s *ui.Menu) {
		s.Item("保存页面…")
		s.Item("创建快捷方式…")
		s.Item("重命名窗口…")
		s.Separator()
		s.Item("开发者工具")
		s.Separator()
		s.Item("删除", ui.MenuDestructive())
	}, ui.MenuInset(), ui.MenuIcon(icons.IconWrench))
	m.Separator()
	m.Check("显示书签", demoBookmarks, ui.MenuOnCheck(func(on bool) { demoBookmarks = on }))
	m.Check("显示完整 URL", demoURLs, ui.MenuOnCheck(func(on bool) { demoURLs = on }))
	m.Separator()
	// 分组标签与条目文字同列(不 inset): 本菜单的 Check/Radio 勾选在右侧、条目文字
	// 从 6 起, 再给 Label 加 inset 会把它单独推出 22px, 看起来像没对齐。
	m.Label("人员")
	m.Radio("张伟", demoPerson == 1, ui.MenuGroup(2), ui.MenuOnCheck(func(on bool) {
		if on {
			demoPerson = 1
		}
	}))
	m.Radio("李娜", demoPerson == 2, ui.MenuGroup(2), ui.MenuOnCheck(func(on bool) {
		if on {
			demoPerson = 2
		}
	}))
}

// buildDialogs 三个对话框: 表单型(编辑资料)、确认型(删除项目) 与可滚动内容。
func buildDialogs(hWindow int, editTrigger, delTrigger, scrollTrigger *ui.Button) {
	// 表单对话框: 标题 + 描述 + 两个带标签的输入框 + 页脚
	edit := ui.NewDialog(hWindow)
	edit.SetTitle("编辑资料")
	edit.SetDescription("修改你的公开资料, 完成后点击保存。")
	edit.BindTrigger(editTrigger)

	// 内容容器是普通元素, 子元素按显式坐标摆放(对话框宽度固定, 高度由对话框自动量出)。
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

	// 可滚动内容: 对应 shadcn 文档的 Scrollable Content / With Sticky Footer ——
	// 头部与页脚固定, 中间内容超过 70vh(实际再夹到面板可用高度)后出滚动条。
	scroll := ui.NewDialog(hWindow, ui.WithDialogScroll(0))
	scroll.SetTitle("可滚动内容")
	scroll.SetDescription("头部与页脚固定, 中间内容区超过可用高度后滚动。")
	scroll.BindTrigger(scrollTrigger)
	body := scroll.ScrollArea().ContentStack(12)
	for i := 0; i < 10; i++ {
		l := ui.NewLabel(0, 0, 0, 100, fmt.Sprintf(
			"%d. 这是一段用于演示滚动的长文本。对话框的面板高度由头部、内容与页脚共同决定, "+
				"内容超过 max-h-[70vh] 时保持头部和页脚不动, 只让中间区域滚动。", i+1),
			body.Handle).SetMuted().SetWrap()
		xui.ItemFill(l)
	}
	scroll.AddFooter("关闭", ui.ButtonOutline, func() { scroll.Close() })
}

// buildAlertDialogs 四个 AlertDialog 示例: 默认 / 小尺寸 / 带媒体块 / 破坏性。
func buildAlertDialogs(hWindow int, defaultTrigger, smallTrigger, mediaTrigger, destructiveTrigger *ui.Button) {
	basic := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	basic.AddCancel("Cancel", nil)
	basic.AddAction("Continue", ui.ButtonPrimary, nil)
	basic.BindTrigger(defaultTrigger)

	small := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Allow accessory to connect?"),
		ui.WithAlertDialogDescription("Do you want to allow the USB accessory to connect to this device?"))
	small.AddCancel("Don't allow", nil)
	small.AddAction("Allow", ui.ButtonPrimary, nil)
	small.BindTrigger(smallTrigger)

	media := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogMedia(icons.IconBluetooth),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This will permanently delete your account and remove your data from our servers."))
	media.AddCancel("Cancel", nil)
	media.AddAction("Continue", ui.ButtonPrimary, nil)
	media.BindTrigger(mediaTrigger)

	destructive := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogMedia(icons.IconTrash2),
		ui.WithAlertDialogMediaTone(ui.AlertDialogMediaDestructive),
		ui.WithAlertDialogTitle("Delete chat?"),
		ui.WithAlertDialogDescription("This will permanently delete this chat conversation. You can delete any memories saved during this chat in Settings."))
	destructive.AddCancel("Cancel", nil) // AlertDialogCancel 默认就是 outline(有边框)样式
	destructive.AddAction("Delete", ui.ButtonDestructive, nil)
	destructive.BindTrigger(destructiveTrigger)
}

// buildDialogVariants 无右上角关闭钮的对话框 + 嵌套模态示例(对话框里再开 AlertDialog)。
func buildDialogVariants(hWindow int, noCloseTrigger, nestedTrigger *ui.Button) {
	noClose := ui.NewDialog(hWindow, ui.WithDialogCloseButton(false))
	noClose.SetTitle("No Close Button")
	noClose.SetDescription("This dialog doesn't have a close button in the top-right corner.")
	noClose.BindTrigger(noCloseTrigger)
	noClose.AddFooter("Close", ui.ButtonOutline, func() { noClose.Close() })

	// 嵌套: 下层 Dialog 开着一层 AlertDialog。模态栈保证只有栈顶响应 ESC/Tab,
	// 关闭上层后焦点回到下层对话框。
	host := ui.NewDialog(hWindow)
	host.SetTitle("Alert Dialog Example")
	host.SetDescription("Click the button below to open an alert dialog.")
	host.BindTrigger(nestedTrigger)
	nested := ui.NewAlertDialog(hWindow,
		ui.WithAlertDialogSize(ui.AlertDialogSizeSmall),
		ui.WithAlertDialogTitle("Are you absolutely sure?"),
		ui.WithAlertDialogDescription("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))
	nested.AddCancel("Cancel", nil)
	nested.AddAction("Continue", ui.ButtonPrimary, nil)
	openNested := ui.NewButton(0, 8, host.ContentWidth(), 0, "Open Alert Dialog", host.Content())
	openNested.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		nested.Show()
		return 0
	})
	host.AddFooter("Close", ui.ButtonOutline, func() { host.Close() })
}
