// xui 组件演示: light/dark 一键换肤 + 全部组件。
//
// 布局: 窗口 → [标题栏行][内容区: 页签行 + 页面宿主], 页面与卡片全部走**流式布局**,
// 不写"算出来的绝对坐标" —— 窗口拉大/最大化时整页自适应:
//   - 页面 = 铺满宿主的竖向流式容器; 隐藏的页会被布局跳过(实测 0x0), 所以不需要叠放定位;
//   - 卡片 = 宽度铺满 + 高度按内容自撑(Auto); 表格/树这类"越高越好"的内容声明成撑满剩余高度;
//   - 页面留白与间距一律用布局项外边距 / SetSpace 表达。
//
// 主题用环境变量 XUI_THEME=dark 启动即暗色, 运行中可用右上角按钮切换;
// 启动页 XUI_HOME_PAGE = 0..5 = 基础 / 表单 / 导航 / 反馈 / 数据 / 图标。
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// 页面序号, 与 pageTitles 一一对应。
const (
	pageBase  = iota // 基础: 按钮 / 徽章 / 分隔线 / 标签 / 卡片
	pageForm         // 表单: 输入 / 复选 / 单选 / 开关 / 滑条 / 下拉框
	pageNav          // 导航: 标签页 / 分页 / 菜单
	pageFeed         // 反馈: 对话框 / 提示气泡 / 进度
	pageData         // 数据: 表格 / 树
	pageIcons        // 图标: 图标总览
	pageCount
)

// pageTitles 的个数必须与 pageCount 一致(页签与页一一对应)。
var pageTitles = [...]string{"基础", "表单", "导航", "反馈", "数据", "图标"}

var (
	tip         *ui.Tooltip
	demoTree    *ui.Tree
	demoTreeSel int32
)

// 标题栏行高度与窗口阴影边带宽度(逻辑像素)。
// 二者必须分开用: 窗口/事件坐标的原点在**整窗**左上角, 非最大化时四周各压着 shadowSize
// 宽的阴影带; 最大化时窗口没有阴影, 原点就是标题栏行左上角 —— 双击标题栏的 y 判定要按
// 这个规则算(见下面的 LButtonDBClick)。都别写成裸数字。
const (
	captionH   int32 = 30 // 标题栏行高度(BorderSize=0 时即 cap 布局项高度)
	shadowSize int32 = 12 // Window_Transparent_Shadow 的阴影大小(SetShadowInfo 的 nSize)
	pagePad    int32 = 24 // 页面内容四周留白
	gap        int32 = 16 // 页面内行/列间距
)

func main() {
	app.InitOrExit()
	a := app.New(true)
	a.EnableAutoDPI(true).EnableDPI(true)
	// 重绘节流默认 30ms(hover 过渡只有约 5 帧); 设 10ms 让过渡/动画跑到 ~60fps。
	// 10 而不是 16: 节流按两次绘制的间隔算, 与运行时 16ms 的过渡定时器会周期性 beat 掉帧;
	// 低于 16 也不再涨(实测 ~16.6ms 硬地板)。见 docs/DESIGN.md 第八节 46。
	a.SetPaintFrequency(10)

	w := window.New(0, 0, 1200, 860, "xui · shadcn 风格自绘组件", 0,
		xcc.Window_Style_Center|xcc.Window_Style_Border|xcc.Window_Style_Drag_Border|xcc.Window_Style_Drag_Window|xcc.Window_Style_Allow_MaxWindow)
	w.SetTransparentType(xcc.Window_Transparent_Shadow)
	w.SetShadowInfo(shadowSize, 128, 8, false, xc.RGBA(0, 0, 0, 128))
	w.SetTransparentAlpha(255)
	w.SetBorderSize(0, 0, 0, 0)
	w.SetDragBorderSize(1, 1, 1, 1) // 让可拖动区域变小, 不然鼠标移到窗口边缘滚动条时会触发拖动改变窗口大小
	w.SetMinimumSize(960, 660)
	w.SetFont(xui.FontHandle(xui.Current().FontDefault))

	xui.RegisterWindow(w.Handle)
	xui.BindWindowBackground(w.Handle) // 窗口底色随主题切换
	tip = ui.NewTooltip()

	// ===== 布局骨架: 直接挂在窗口上 —— [标题栏行 captionH][内容区 Weight 1] =====
	// 为什么不套一层 root: 竖轴上 "占满剩余空间" 只有 `Weight` 能做到, 而 **Weight 只在
	// 父级尺寸确定时才生效**(窗口直属 / Fixed / Weight 父级都算; 父级是 Fill 或 Auto 时
	// 恒为 0, 实测见 DESIGN 第八节 55)。窗口是唯一"天然确定"的父级, 所以这一层必须直接
	// 是窗口的子项 —— 中间垫一个 Fill 容器, 下面的 Weight 就全废了。
	w.EnableLayout(true)

	// 标题栏行: logo/标题居左, 按钮组浮到最右(中间标题 Fill 撑开)。
	// 行自身鼠标穿透 —— 不挡 Drag_Window 的拖动命中, 子元素(按钮)不受影响。
	cap := widget.NewLayoutEle(0, 0, 0, captionH, w.Handle)
	cap.EnableHorizon(true)
	cap.EnableBkTransparent(true)
	cap.SetSpace(6)
	cap.SetAlignV(xcc.Layout_Align_Center)
	cap.EnableMouseThrough(true)
	cap.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	cap.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, captionH)

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
		p.Icon(icons.IconAppWindow, &p.Rc, t.Foreground, 2)
		return 0
	})
	logo.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, 18)
	logo.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 18)
	xui.ItemMarginLeft(logo, 6)

	// 标题文字: Weight 宽度占满剩余空间把按钮组顶到行尾
	// (横向行里主轴 Fill 会独占整行把兄弟挤到下一行, 剩余空间必须用 Weight)
	title := widget.NewLayoutEle(0, 0, 0, captionH, cap.Handle)
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

	showLayoutFrame := false
	// 标题栏这排按钮都不声明布局尺寸: 尺寸在创建参数里给足(36 宽, 高度含 button.go 的
	// 1px 按压预留), 布局默认"不接管尺寸"就会原样保留 —— 见 xui/layout.go 的约定。
	// 显示布局对象边界按钮(ghost 图标按钮, 与最小化/关闭同排同款):
	// 图标显示点击后的动作 —— 隐藏时 eye(去看), 显示时 eye-off(去关)
	btnShowLayoutFrame := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconEye))
	tip.Bind(btnShowLayoutFrame.Handle, "显示/隐藏布局对象边界")
	btnShowLayoutFrame.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		showLayoutFrame = !showLayoutFrame
		app.ShowLayoutFrame(showLayoutFrame) // 显示布局对象边界
		btnShowLayoutFrame.SetIcon(common.Choose(showLayoutFrame, icons.IconEyeOff, icons.IconEye))
		w.Redraw(false)
		return 0
	})

	// 主题切换按钮: 图标显示切换目标 —— 亮色时 moon(去暗色), 暗色时 sun(去亮色)
	toggle := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconMoon))
	tip.Bind(toggle.Handle, "切换主题")
	if dark {
		toggle.SetIcon(icons.IconSun)
	}
	toggle.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		dark = !dark
		toggle.SetIcon(common.Choose(dark, icons.IconSun, icons.IconMoon))
		xui.SetTheme(common.Choose(dark, xui.Dark(), xui.Light()))
		// 切换主题会重绘窗口, 所以 toggle 不用单独重绘
		return 0
	})

	// 控制按钮: 最小化 / 最大化(还原) / 关闭, 行为全部交给 xcgui 的按钮类型。
	// 最小化和关闭按钮图标放大了些
	btnMin := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconMinus), ui.WithIconSize(22))
	btnMin.SetTypeEx(xcc.Button_Type_Min)

	btnMax := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconSquare))
	btnMax.SetTypeEx(xcc.Button_Type_Max)

	btnClose := ui.NewButton(0, 0, capBtnW, capBtnH, "", cap.Handle,
		ui.WithVariant(ui.ButtonGhost), ui.WithIcon(icons.IconX), ui.WithIconSize(22))
	btnClose.SetTypeEx(xcc.Button_Type_Close)

	// 最大化/还原换图标: 按钮类型只管行为, 自绘图标不会跟窗口状态变。
	// 挂 WM_SIZE 而不是只挂点击回调 —— 双击标题栏、Win+↑、系统菜单等改状态的路径
	// 都会经过它; 用 a.UI 推迟到消息处理之后, IsMaxWindow 已是新状态。
	syncMaxIcon := func() {
		btnMax.SetIcon(common.Choose(w.IsMaxWindow(), icons.IconCopy, icons.IconSquare))
	}
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_SIZE && (wParam == wapi.SIZE_MAXIMIZED || wParam == wapi.SIZE_RESTORED) {
			a.UI(syncMaxIcon)
		}
		return 0
	})

	// 内容区: 竖向流式 [页签行][页面宿主 Weight 1]。它自己是窗口的 Weight 布局项;
	// 打开布局后, 页签行与页面宿主就交给它排, 不再需要绝对定位。
	content := widget.NewLayoutEle(0, 0, 0, 0, w.Handle)
	content.EnableHorizon(false)
	content.EnableLayout(true)
	content.EnableBkTransparent(true)
	content.EnableAutoWrap(false)
	content.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	content.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)

	buildPages(w.Handle, content.Handle)

	// 双击标题栏行最大化/还原。
	// y 判定必须按窗口**实际**坐标系算: 非最大化时整窗矩形四周含 shadowSize 宽的阴影带,
	// 而事件坐标原点是整窗左上角, 标题栏行落在 [shadowSize, shadowSize+captionH]; 最大化时
	// 窗口没有阴影, 行落在 [0, captionH]。只写 captionH 会让非最大化时的双击判定少一段。
	w.AddEvent_LButtonDBClick(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		top := captionH + shadowSize
		if w.IsMaxWindow() {
			top = captionH
		}
		if pPt.Y > top {
			return 0
		} else if w.GetStayEle() != 0 { // 防止点到标题栏上的按钮
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

	// 页签行: 左右与窗口留白对齐
	tabs := ui.NewTabs(0, 0, 520, host, pageTitles[:])
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

	buildBasePage(bodies[pageBase])
	buildFormPage(bodies[pageForm])
	buildNavPage(bodies[pageNav])
	buildFeedPage(bodies[pageFeed], hWindow)
	buildDataPage(bodies[pageData])
	buildIconPage(bodies[pageIcons])

	show := func(i int) {
		for j, p := range pages {
			p.Show(j == i)
		}
		// 可见性变了必须重排宿主: 隐藏的页不参与排版, 显示的那页才会拿到整个高度。
		// 页面自身也要重排(布局元素在父容器隐藏期间不排版, 恢复可见后不重排就是空白)。
		pageHost.AdjustLayout()
		pages[i].AdjustLayout()
		pages[i].Redraw(false)
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

// ===== 页 1: 基础(按钮 / 徽章 / 分隔线 / 标签 / 卡片) =====

func buildBasePage(body *widget.LayoutEle) {
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

	// ---- 第二行: 卡片本身(直接以真卡片呈现, 不套外壳) ----
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

	footer := ui.NewCardFooter(0, 0, 0, 0, cardDemo.Handle)
	xui.ItemFill(footer)
	ui.NewLabel(16, 10, 0, 24, "CardFooter · border-t + bg-muted/50, 只随卡片下两角圆角", footer.Handle)

	// ---- 第三行: 滚动容器(组件形态; 页面本身也是它, 见 newPage) ----
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
}

// ===== 页 2: 表单(输入 / 复选 / 单选 / 开关 / 滑条 / 下拉框) =====

// 骨架是"一行两列、每列把两张卡竖着排", 不是"两行各两列":
// 行高只有 Weight(按剩余空间等分)可用, 卡片比行高时下一行就会压到上一行的卡片上
// (实测: 输入框卡 430 高, 行只分到 363, 第二行的卡片上沿 379 直接盖住它 —— 用户报的
// "下方两个卡片盖住上方卡片")。竖排在同一列里则天然按内容顺序往下走, 不重叠。
func buildFormPage(body *widget.LayoutEle) {
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
	ml := ui.NewInput(0, 0, 0, 72, v.Handle,
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

	// ---- 左列 3: 日期与时间 ----
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
	rowT := xui.HStack(v5.Handle, 32, 8)
	tp := ui.NewTimePicker(0, 0, 0, 0, rowT.Handle, ui.WithTimePickerValue(9, 30))
	xui.ItemWidth(tp, 160)
	tpVal := ui.NewLabel(0, 0, 0, 0, "09:30", rowT.Handle).SetMuted()
	xui.ItemWidth(tpVal, 80)
	tp.OnChange(func(h, m int) {
		tpVal.SetText(tp.Text()).Redraw()
	})

	// ---- 右列 3: 月历卡片 ----
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
}

// ===== 页 3: 导航(标签页 / 分页 / 菜单) =====

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
	menuBtn := ui.NewButton(0, 0, 0, 0, "打开菜单", rm.Handle, ui.WithVariant(ui.ButtonOutline))
	xui.ItemWidth(menuBtn, 104)
	menuBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		showDemoMenu(hEle)
		return 0
	})
	xui.ItemFill(ui.NewLabel(0, 0, 0, 0, "条目 hover 走 bg-accent, 与表格行 / 树行同一口径", v2.Handle).SetMuted())

	// ---- 第二行: 分页 ----
	_, v3 := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "分页", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"折叠采用恒定 7 槽位: 首段/中段/末段占位宽度一致, 连点同一坐标不会点空", v3.Handle))

	pagInfo := ui.NewLabel(0, 0, 0, 0, "", v3.Handle).SetMuted()
	xui.ItemFill(pagInfo)
	pag := ui.NewPagination(0, 0, 0, 0, v3.Handle,
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
}

// ===== 页 4: 反馈(对话框 / 提示气泡 / 进度) =====

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
	buildDialogs(hWindow, editBtn, delBtn)
	xui.ItemFill(desc(v.Handle, "ESC / 点击遮罩 / 右上角 X 关闭; Tab 与 Shift+Tab 在对话框内循环。"))

	// ---- 列 2: 提示气泡 ----
	_, v2 := autoCard(col(r.Handle, 1, gap).Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "提示气泡", v2.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "独立透明弹窗承载, 不抢焦点; 延迟 0, 进出场走淡入淡出", v2.Handle))
	rt := xui.HStack(v2.Handle, 36, 8)
	hoverBtn := ui.NewButton(0, 0, 0, 0, "悬浮我", rt.Handle, ui.WithVariant(ui.ButtonSecondary))
	xui.ItemWidth(hoverBtn, 104)
	tip.Bind(hoverBtn.Handle, "我是提示气泡")
	xui.ItemFill(desc(v2.Handle, "绑定的元素自身有弹层时(如 ComboBox), 弹层打开要主动 HideTooltip。"))

	// ---- 第二行: 通知 + 进度 ----
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
}

// ===== 页 5: 数据(表格 / 树) =====

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

// ===== 页 6: 图标(icons 资源子包) =====
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

// buildDialogs 两个对话框: 表单型(编辑资料) 与 确认型(删除项目)。
func buildDialogs(hWindow int, editTrigger, delTrigger *ui.Button) {
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
}
