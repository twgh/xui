// xui 组件演示: light/dark 一键换肤 + 全部组件。
//
// 布局: 窗口 → [标题栏行][内容区: 页签行 + 页面宿主][底部状态栏], 页面与卡片全部走**流式布局**,
// 不写"算出来的绝对坐标" —— 窗口拉大/最大化时整页自适应:
//   - 页面 = 铺满宿主的竖向流式容器; 隐藏的页会被布局跳过(实测 0x0), 所以不需要叠放定位;
//   - 卡片 = 宽度铺满 + 高度按内容自撑(Auto); 表格/树这类"越高越好"的内容声明成撑满剩余高度;
//   - 页面留白与间距一律用布局项外边距 / SetSpace 表达。
//
// 主题用环境变量 XUI_THEME=dark 启动即暗色, 运行中可用右上角按钮切换;
// 启动页 XUI_HOME_PAGE = 0..24 = 基础1..4 / 表单1..6 / 导航1..6 / 反馈1..3 / 图表 / 数据 / 数据2 / 对话 / 对话2 / 图标。
package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// 页面序号, 与 pageTitles 一一对应。
const (
	pageBase1  = iota // 基础1: 按钮 / 按钮组 / 切换按钮
	pageBase2         // 基础2: 徽章 / 分隔线 / 标签 / 快捷键 / 加载 / 骨架 / 头像
	pageBase3         // 基础3: 弹出层 / 轮播 / 分栏 / 卡片 / 比例容器
	pageBase4         // 基础4: 滚动容器 / 列表项 / 手风琴
	pageForm1         // 表单1: 输入 / 验证码 / 滑动条
	pageForm2         // 表单2: 复选 / 单选 / 开关 / 下拉框 / Select
	pageForm3         // 表单3: 字段与分组(Field / FieldSet) / 文件对话框(FileDialog)
	pageForm4         // 表单4: 日期与时间 / 月历 / 范围与预设
	pageForm5         // 表单5: 单选组 / 复选框三态与校验
	pageForm6         // 表单6: 多步问卷(Questionnaire)
	pageNav1          // 导航1: 命令面板 / 下拉菜单 / 右键菜单
	pageNav2          // 导航2: 标签页 / 面包屑 / 分页
	pageNav3          // 导航3: 步骤条 / 可折叠面板
	pageNav4          // 导航4: 时间轴
	pageNav5          // 导航5: 导航菜单(NavigationMenu) + 菜单栏(Menubar)
	pageNav6          // 导航6: 侧边栏(Sidebar)
	pageFeed1         // 反馈1: 对话框 / 提示气泡 / 警示对话框 / 对话框变体
	pageFeed2         // 反馈2: 提示条 / 通知 / 进度 / 空状态
	pageFeed3         // 反馈3: 侧边 Sheet / 底部 Drawer
	pageCharts        // 图表: 柱状 / 折线 / 面积 / 饼环
	pageData          // 数据: 数据表格(DataTable) / 树
	pageData2         // 数据2: 虚拟滚动数据表格(10 万行)
	pageChat          // 对话: 消息 / 气泡 / 标记 / 轨迹 / 附件
	pageChat2         // 对话2: MessageScroller(轮次锚定 / 跟随 / 前置历史 / 跳转)
	pageIcons         // 图标: 图标总览
	pageCount
)

// pageTitles 的个数必须与 pageCount 一致(页签与页一一对应)。
var pageTitles = [...]string{
	"基础1", "基础2", "基础3", "基础4",
	"表单1", "表单2", "表单3", "表单4", "表单5", "表单6",
	"导航1", "导航2", "导航3", "导航4", "导航5", "导航6",
	"反馈1", "反馈2", "反馈3",
	"图表",
	"数据", "数据2", "对话", "对话2", "图标",
}

// tabW 是页签的初始宽(也是换行模式下的单页最小宽)。页签行按窗口宽自适应,
// 装不下时由 Tabs 自己换行; 这个值只决定首帧尺寸。
const tabW = 72

var (
	tip         *ui.Tooltip
	demoTree    *ui.Tree
	demoTreeSel int32

	demoWindow      *ui.AppWindow
	demoData2Load   func() // 「数据2」首次显示时才生成 10 万行(默认启动不占这份内存)
	demoScreen      *ui.ScreenToaster
	demoTray        *ui.TrayIcon
	demoTrayButton  *ui.Button
	demoPopup       *ui.ScreenPopup
	demoFocusPopup  *ui.ScreenPopup
	demoPopupToggle *ui.Button
	demoPopupStatus *ui.Label
	demoPopupLines  int

	demoDark            bool // 当前亮/暗
	demoThemeColor      int  // demoThemePresets 下标; -1 = 用 demoCustomPrimary
	demoThemeColorLabel *ui.Label
	demoCustomPrimary   xui.Color         // 颜色选择器挑的自定义主色(0 = 未用)
	demoSwatches        []*widget.Element // 主题弹窗里的色块, 切主题后要逐个重绘
	demoThemePop        *ui.Popover       // 标题栏"主题"按钮弹的卡片
	demoCustomPicker    *ui.ColorPicker   // 卡片里的取色器(切换预设色后同步显示)
)

// 主题色预设: 默认(内置主题原样) + 5 个彩色主色。亮/暗各一套, 前景色跟随对比度。
// 彩色主题只换"主色组"令牌(Primary/Ring/SidebarPrimary), 其余令牌保持内置主题。
type demoThemePreset struct {
	name            string
	light, dark     xui.Color
	lightFg, darkFg xui.Color
}

var demoThemePresets = []demoThemePreset{
	{name: "默认"},
	{name: "蓝色", light: xui.Hex("#2563eb"), dark: xui.Hex("#3b82f6"), lightFg: xui.Hex("#fafafa"), darkFg: xui.Hex("#0b1220")},
	{name: "绿色", light: xui.Hex("#16a34a"), dark: xui.Hex("#22c55e"), lightFg: xui.Hex("#fafafa"), darkFg: xui.Hex("#052e16")},
	{name: "紫色", light: xui.Hex("#7c3aed"), dark: xui.Hex("#a78bfa"), lightFg: xui.Hex("#fafafa"), darkFg: xui.Hex("#1e1b4b")},
	{name: "玫瑰", light: xui.Hex("#e11d48"), dark: xui.Hex("#fb7185"), lightFg: xui.Hex("#fafafa"), darkFg: xui.Hex("#4c0519")},
	{name: "橙色", light: xui.Hex("#ea580c"), dark: xui.Hex("#fb923c"), lightFg: xui.Hex("#fafafa"), darkFg: xui.Hex("#431407")},
}

// applyDemoTheme 按当前亮暗 + 主题色应用主题。克隆内置主题后只替换主色组,
// 其余令牌(包括暗色模式)保持原样; 组件都在绘制时实时取令牌, 所以无需重建窗口。
func applyDemoTheme() {
	base := xui.Light()
	if demoDark {
		base = xui.Dark()
	}
	// 自定义主色: 前景色按亮度自动取黑/白, 免得挑个浅色后按钮文字看不见
	if demoThemeColor < 0 && demoCustomPrimary != 0 {
		t := *base
		fg := demoContrastFg(demoCustomPrimary)
		t.Name = base.Name + "-custom"
		t.Primary = demoCustomPrimary
		t.PrimaryForeground = fg
		t.Ring = demoCustomPrimary
		t.SidebarPrimary = demoCustomPrimary
		t.SidebarPrimaryForeground = fg
		xui.SetTheme(&t)
		if demoThemeColorLabel != nil {
			demoThemeColorLabel.SetText("当前主题色: 自定义 " + demoCustomPrimary.String()).Redraw(false)
		}
		demoRedrawSwatches()
		return
	}
	p := demoThemePresets[demoThemeColor]
	if demoThemeColor == 0 {
		xui.SetTheme(base)
	} else {
		t := *base
		primary, fg := p.light, p.lightFg
		if demoDark {
			primary, fg = p.dark, p.darkFg
		}
		t.Name = base.Name + "-" + p.name
		t.Primary = primary
		t.PrimaryForeground = fg
		t.Ring = primary
		// SidebarPrimary* 当前没有组件消费(本库侧栏没有上游那个"logo 方块"区块, 只认
		// SidebarAccent*), 这里照样设上是为了演示"转译社区主题时整组令牌都填"的做法。
		t.SidebarPrimary = primary
		t.SidebarPrimaryForeground = fg
		xui.SetTheme(&t)
	}
	if demoThemeColorLabel != nil {
		demoThemeColorLabel.SetText("当前主题色: " + p.name).Redraw(false)
	}
	demoRedrawSwatches()
}

// demoContrastFg 按 sRGB 亮度挑前景色: 亮的底用深色字, 暗的底用浅色字。
func demoContrastFg(c xui.Color) xui.Color {
	r, g, b, _ := c.RGBA()
	if 0.299*float64(r)+0.587*float64(g)+0.114*float64(b) > 153 {
		return xui.Hex("#0b1220")
	}
	return xui.Hex("#fafafa")
}

// demoSwatchColor 返回第 i 个预设色在当前亮暗下的实际颜色(默认项 = 内置主题的主色)。
func demoSwatchColor(i int) xui.Color {
	if i == 0 {
		if demoDark {
			return xui.Dark().Primary
		}
		return xui.Light().Primary
	}
	if demoDark {
		return demoThemePresets[i].dark
	}
	return demoThemePresets[i].light
}

func demoRedrawSwatches() {
	for _, s := range demoSwatches {
		if s != nil {
			s.Redraw(false)
		}
	}
}

// buildThemePopover 是标题栏"主题设置"卡片的内容: 一排能看见颜色的主色块 + 取色器。
//
// 色块是自绘 Element(圆点 + 选中环): 原生菜单项画不出颜色, 所以这里改弹 Popover 卡片。
func buildThemePopover(content int) {
	demoSwatches = demoSwatches[:0]
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "主题色", content).SetFontSize(14))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20,
		"点色块换预设, 或用取色器挑任意颜色", content).SetFontSize(12).SetMuted())

	swRow := xui.HStack(content, 30, 8)
	for i := range demoThemePresets {
		i := i
		dot := widget.NewElement(0, 0, 26, 26, swRow.Handle)
		if dot == nil {
			continue
		}
		xui.ItemSize(dot, 26, 26)
		dot.EnableBkTransparent(true)
		dot.EnableDrawBorder(false)
		dot.EnableFocus(false)
		dot.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
			p := xui.BeginPaint(hEle, hDraw)
			if p == nil {
				return 0
			}
			*pbHandled = true
			t := xui.Current()
			cx := (p.Rc.Left + p.Rc.Right) / 2
			cy := (p.Rc.Top + p.Rc.Bottom) / 2
			const r = int32(9)
			swatch := demoSwatchColor(i)
			// 未选中: 外圈 border 色当 1px 描边(白色色块没边就"消失")
			if demoThemeColor != i {
				p.Draw.SetBrushColor(uint32(t.Border))
				p.Draw.FillEllipse(&xc.RECT{Left: cx - r, Top: cy - r, Right: cx + r, Bottom: cy + r})
			}
			p.Draw.SetBrushColor(uint32(swatch))
			p.Draw.FillEllipse(&xc.RECT{Left: cx - r + 1, Top: cy - r + 1, Right: cx + r - 1, Bottom: cy + r - 1})
			// 选中: 色块中央打对勾(按色块亮度取黑/白)。
			// 不用外圈环: 环色(Ring)与被选中的色块常常同色, 糊成"甜甜圈"; 靠 1px 的缝分开时,
			// 缝在 150% 缩放下是 1 或 2 个物理像素, 左右不等就成了"圆点没在环心"。
			if demoThemeColor == i {
				mark := int32(11)
				ic := xc.RECT{Left: cx - mark/2, Top: cy - mark/2, Right: cx + mark/2, Bottom: cy + mark/2}
				p.Icon(icons.IconCheck, &ic, demoContrastFg(swatch), 2)
			}
			return 0
		})
		dot.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			*pbHandled = true
			demoThemeColor = i
			demoCustomPrimary = 0
			applyDemoTheme()
			if demoCustomPicker != nil {
				// 只同步显示, 不走 OnChange(SetValue 不触发回调, 否则会绕回来再改一次主题)
				demoCustomPicker.SetValue(demoSwatchColor(i).String())
			}
			return 0
		})
		demoSwatches = append(demoSwatches, dot)
	}

	xui.ItemFill(ui.NewSeparator(0, 0, 0, content))
	xui.ItemFill(ui.NewLabel(0, 0, 0, 20, "自定义颜色", content).SetFontSize(12).SetMuted())
	val := "#3b82f6"
	if demoCustomPrimary != 0 {
		val = demoCustomPrimary.String()
	}
	cp := ui.NewColorPicker(0, 0, 0, 0, content, ui.WithColorPickerValue(val),
		ui.WithColorPickerOnChange(func(hex string) {
			demoThemeColor = -1
			demoCustomPrimary = xui.Hex(hex)
			applyDemoTheme()
		}))
	if cp != nil {
		xui.ItemFill(cp)
		demoCustomPicker = cp
	}
}

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
	demoDark = strings.EqualFold(os.Getenv("XUI_THEME"), "dark")
	applyDemoTheme()
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
	demoWindow = w
	// 窗口图标也要设: 只设托盘不会同步任务栏/Alt+Tab/任务栏预览窗。大图标管
	// 任务栏 + Alt+Tab, 小图标管任务栏预览窗左上角(也影响任务管理器), 见 DESIGN 180。
	if hIcon := demoAppIcon(); hIcon != 0 {
		w.SetBigIcon(hIcon)
	}
	if hIcon := demoAppSmallIcon(); hIcon != 0 {
		w.SetSmallIcon(hIcon)
	}
	tip = ui.NewTooltip()

	// 标题栏自定义按钮(排在最小化左边): 布局边界 / 主题设置, 都带 tooltip。
	// 布局边界按钮的图标显示点击后的动作 —— 隐藏时 eye(去看), 显示时 eye-off(去关)。
	showLayoutFrame := false
	var btnFrame *ui.Button
	btnFrame = w.AddCaptionButton(icons.IconEye, func() {
		showLayoutFrame = !showLayoutFrame
		app.ShowLayoutFrame(showLayoutFrame) // 显示布局对象边界
		btnFrame.SetIcon(common.Choose(showLayoutFrame, icons.IconEyeOff, icons.IconEye))
		w.Redraw(false)
	})
	tip.Bind(btnFrame.Handle, "显示/隐藏布局对象边界")

	// 外观(亮/暗)独立成一个按钮: 图标显示"点了会变成什么" —— 当前是亮色就画月亮。
	var btnDark *ui.Button
	btnDark = w.AddCaptionButton(common.Choose(demoDark, icons.IconSun, icons.IconMoon), func() {
		demoDark = !demoDark
		applyDemoTheme()
		btnDark.SetIcon(common.Choose(demoDark, icons.IconSun, icons.IconMoon))
		w.Redraw(false)
	})
	tip.Bind(btnDark.Handle, "切换亮色/暗色")

	// 主题设置: 弹卡片(Popover)而不是原生菜单 —— 菜单项画不出颜色块, 也塞不进取色器。
	// 卡片里点 ColorPicker 会再弹一层取色面板, 所以本卡片要 keepOnBlur(否则先因失焦关掉)。
	var btnTheme *ui.Button
	btnTheme = w.AddCaptionButton(icons.IconPalette, nil)
	if btnTheme != nil {
		demoThemePop = ui.NewPopover(btnTheme.Handle,
			ui.WithPopoverWidth(264),
			ui.WithPopoverKeepOnBlur(true),
			ui.WithPopoverContent(buildThemePopover))
		tip.Bind(btnTheme.Handle, "主题设置")
	}
	_ = demoThemePop

	// 内容区: AppWindow 的纵向流式容器(宽铺满、高占满标题栏以下); 页面自己管留白
	content := w.Content()
	content.SetSpace(0)
	buildPages(w.Handle, content.Handle)

	// 窗口底部状态栏: 直接挂在窗口的流式布局末尾(内容区是 Weight 高, 状态栏 Fixed 高),
	// 不写绝对坐标; 右下角抓手按住拖动改宿主窗口大小, 尺寸读数跟着刷新。
	status := ui.NewStatusBar(0, 0, 0, 0, w.Handle,
		ui.WithStatusBarSizeReadout(), ui.WithStatusBarTooltip(tip))
	status.AddLeft("就绪", ui.WithStatusBarItemIcon(icons.IconCircleCheck),
		ui.WithStatusBarItemTooltip("xui demo 已就绪"))
	status.AddLeft("main", ui.WithStatusBarItemIcon(icons.IconGitBranch),
		ui.WithStatusBarItemTooltip("当前分支"))
	status.AddCenter("xui · shadcn 风格自绘组件", ui.WithStatusBarItemTooltip("窗口标题"))
	status.AddRight("UTF-8", ui.WithStatusBarItemTooltip("文件编码"))
	status.AddRight("Go", ui.WithStatusBarItemActive(), ui.WithStatusBarItemTooltip("语言模式"))
	status.AddRight("Ln 1, Col 1", ui.WithStatusBarItemTooltip("光标位置"))
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

	// 页签行: 左右与窗口留白对齐; 宽度跟随窗口, 装不下时自动换行(WithTabsWrap)。
	// 下边距 0: 与页面的距离由页面体自己的上边距表达, 不两边叠加。
	tabs := ui.NewTabs(0, 0, int32(len(pageTitles))*tabW, host, pageTitles[:], ui.WithTabsWrap())
	xui.ItemFill(tabs)
	xui.ItemMargin(tabs, pagePad, 8, pagePad, 0)

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
	pages[pageData2].SetContentFill() // 同样铺满: 虚拟滚动表要跟随窗口长高

	buildBase1Page(bodies[pageBase1])
	buildBase2Page(bodies[pageBase2])
	buildBase3Page(bodies[pageBase3])
	buildBase4Page(bodies[pageBase4])
	buildForm1Page(bodies[pageForm1])
	buildForm2Page(bodies[pageForm2])
	buildForm3Page(bodies[pageForm3], hWindow)
	buildForm4Page(bodies[pageForm4])
	buildForm5Page(bodies[pageForm5])
	buildForm6Page(bodies[pageForm6])
	buildNav1Page(bodies[pageNav1])
	buildNav2Page(bodies[pageNav2])
	buildNav3Page(bodies[pageNav3])
	buildNav4Page(bodies[pageNav4])
	buildNav5Page(bodies[pageNav5])
	buildNav6Page(bodies[pageNav6])
	buildFeed1Page(bodies[pageFeed1], hWindow)
	buildFeed2Page(bodies[pageFeed2], hWindow)
	buildFeed3Page(bodies[pageFeed3], hWindow)
	buildChartPage(bodies[pageCharts])
	buildDataPage(bodies[pageData])
	buildDataPage2(bodies[pageData2])
	buildChatPage(bodies[pageChat])
	buildChatScrollerPage(bodies[pageChat2])
	buildIconPage(bodies[pageIcons])

	show := func(i int) {
		// 「数据2」的 10 万行是演示数据, 首次进入该页才生成并灌入虚表。
		if i == pageData2 && demoData2Load != nil {
			demoData2Load()
			demoData2Load = nil
		}
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
// 上边距 16(< pagePad): 页签行与首张卡片之间只留一行节奏的空隙, 不叠加页签行距。
func newPage(host int) (page *ui.ScrollArea, body *widget.LayoutEle) {
	page = ui.NewScrollArea(0, 0, 0, 0, host)
	page.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	xui.ItemFillHeight(page) // 高度 = 页面宿主的剩余空间(Weight)

	body = page.ContentStack(gap)
	body.LayoutItem_SetMargin(pagePad, 16, pagePad, pagePad)
	return page, body
}
