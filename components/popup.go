package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// monitorWorkArea 返回窗口所在显示器的工作区(物理像素); 取不到时 ok=false。
//
// 用工作区而不是整屏: 工作区已经扣掉任务栏 / 停靠栏。弹层夹在整屏高度里时, 只要触发区
// 靠近屏幕底部(例如页面已滚到底), 卡片下半截就会被任务栏盖住 —— 任务栏永远在最上层,
// 弹层是 topmost 也盖不过它。
//
// ⚠️ hWnd 必须是 **Win32 HWND**(xc.XWnd_GetHWND 转出来的), 不能喂 xcgui 的窗口句柄:
// MonitorFromWindow 对无效句柄返回 0, 调用方会静默退回"整屏"兜底分支, 任务栏又会盖上来。
func monitorWorkArea(hWnd uintptr) (xc.RECT, bool) {
	hMon := wapi.MonitorFromWindow(hWnd, wapi.MONITOR_DEFAULTTONEAREST)
	if hMon == 0 { // 句柄无效时退回桌面窗口: 至少仍是"工作区", 不会漏掉任务栏
		hMon = wapi.MonitorFromWindow(wapi.GetDesktopWindow(), wapi.MONITOR_DEFAULTTONEAREST)
	}
	if hMon == 0 {
		return xc.RECT{}, false
	}
	var mi wapi.MONITORINFO
	if !wapi.GetMonitorInfo(hMon, &mi) {
		return xc.RECT{}, false
	}
	return xc.RECT{
		Left:   mi.RcWork.Left,
		Top:    mi.RcWork.Top,
		Right:  mi.RcWork.Right,
		Bottom: mi.RcWork.Bottom,
	}, true
}

// PopoverSide 弹层相对触发区的方位, 对应 radix Popover 的 side。
type PopoverSide int

const (
	PopoverSideBottom PopoverSide = iota // 下方(默认)
	PopoverSideTop                       // 上方
	PopoverSideLeft                      // 左侧
	PopoverSideRight                     // 右侧
)

// PopoverAlign 弹层与触发区在交叉轴上的对齐方式, 对应 radix Popover 的 align。
type PopoverAlign int

const (
	PopoverAlignCenter PopoverAlign = iota // 居中对齐(默认)
	PopoverAlignStart                      // 起始边对齐
	PopoverAlignEnd                        // 结束边对齐
)

// popupPosition 计算锚定弹层的位置: 先按 side/align 摆放, 放不下翻到对侧,
// 最后夹进显示器工作区。hTrigger 只用来取触发元素与宿主窗口。
func popupPosition(hTrigger int, w, h int32, side PopoverSide, align PopoverAlign, gap int32) (int32, int32) {
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(hTrigger, &rc)
	hWnd := xc.XWidget_GetHWINDOW(hTrigger)
	if hWnd <= 0 {
		return 0, 0
	}
	xc.XWnd_GetRect(hWnd, &wr)
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96

	aL := wr.Left + rc.Left
	aT := wr.Top + rc.Top
	aR := wr.Left + rc.Right
	aB := wr.Top + rc.Bottom
	x, y := popupPlaceIn(hTrigger, w, h, aL, aT, aR, aB, side, align, gap)

	if wa, ok := monitorWorkArea(xc.XWnd_GetHWND(hWnd)); ok {
		waL := int32(float64(wa.Left)/sc + 0.5)
		waT := int32(float64(wa.Top)/sc + 0.5)
		waR := int32(float64(wa.Right)/sc + 0.5)
		waB := int32(float64(wa.Bottom)/sc + 0.5)
		if x+w > waR {
			x = waR - w
		}
		if x < waL {
			x = waL
		}
		if y+h > waB {
			y = waB - h
		}
		if y < waT {
			y = waT
		}
	}
	return x, y
}

func popupPlaceIn(hTrigger int, w, h, aL, aT, aR, aB int32, side PopoverSide, align PopoverAlign, gap int32) (int32, int32) {
	x, y := aL, aT
	switch side {
	case PopoverSideTop:
		x = popupCrossH(aL, aR, w, align)
		y = aT - h - gap
		if y < 0 && aB+gap+h <= screenBottom(hTrigger) {
			y = aB + gap
		}
	case PopoverSideLeft:
		x = aL - w - gap
		y = popupCrossV(aT, aB, h, align)
		if x < 0 && aR+gap+w <= screenRight(hTrigger) {
			x = aR + gap
		}
	case PopoverSideRight:
		x = aR + gap
		y = popupCrossV(aT, aB, h, align)
		if x+w > screenRight(hTrigger) && aL-gap-w >= 0 {
			x = aL - w - gap
		}
	default:
		x = popupCrossH(aL, aR, w, align)
		y = aB + gap
		if y+h > screenBottom(hTrigger) && aT-h-gap >= 0 {
			y = aT - h - gap
		}
	}
	return x, y
}

func popupCrossH(aL, aR, w int32, align PopoverAlign) int32 {
	switch align {
	case PopoverAlignStart:
		return aL
	case PopoverAlignEnd:
		return aR - w
	default:
		return aL + (aR-aL)/2 - w/2
	}
}

func popupCrossV(aT, aB, h int32, align PopoverAlign) int32 {
	switch align {
	case PopoverAlignStart:
		return aT
	case PopoverAlignEnd:
		return aB - h
	default:
		return aT + (aB-aT)/2 - h/2
	}
}

// screenBottom/screenRight 返回触发区所在显示器的逻辑工作区边界(取不到时用整屏兜底)。
func screenBottom(hTrigger int) int32 {
	if hWnd := xc.XWidget_GetHWINDOW(hTrigger); hWnd > 0 {
		if wa, ok := monitorWorkArea(xc.XWnd_GetHWND(hWnd)); ok {
			dpi := xc.XWnd_GetDPI(hWnd)
			if dpi <= 0 {
				dpi = 96
			}
			return int32(float64(wa.Bottom)/float64(dpi)*96 + 0.5)
		}
	}
	return 1 << 30
}

func screenRight(hTrigger int) int32 {
	if hWnd := xc.XWidget_GetHWINDOW(hTrigger); hWnd > 0 {
		if wa, ok := monitorWorkArea(xc.XWnd_GetHWND(hWnd)); ok {
			dpi := xc.XWnd_GetDPI(hWnd)
			if dpi <= 0 {
				dpi = 96
			}
			return int32(float64(wa.Right)/float64(dpi)*96 + 0.5)
		}
	}
	return 1 << 30
}

// ===== 组件内建弹层的公共基座 =====
//
// 日期框 / 时间框这类组件需要一个"贴着触发区弹出的卡片", 原生控件里没有可直接复用的
// 容器(ComboBox 的弹层是它私有的 ListBox 窗口, Menu 只装菜单项), 所以自建:
//
//   - 窗口: Window_Transparent_Shadow —— 圆角、投影一次解决, 客户区自动收缩出
//     阴影带(实测同 menu, 见 DESIGN 第八节 64); 背景必须垫 alpha=0 填充压掉
//     xcgui 的白色默认底(见 Show 的说明);
//   - body: 一个铺满卡片区的自绘元素, 画 bg-popover 卡片 + ring-1 ring-foreground/10,
//     子控件(月历 / 时间列表)直接建在它上面;
//   - 关闭: 窗口失焦(KillFocus) —— 与原生弹层同一种机制, 点外部即关; ESC 也关;
//   - 进场: **元素 alpha**(SetSubtreeAlpha)淡入 —— 整窗 alpha 对透明窗只认第一次
//     调用(57), 复用弹窗整段失效; 过渡挂**弹窗自己**, 挂宿主窗会每帧全量重绘
//     主窗口(快速点击时 UI 卡顿, 见 DESIGN 73)。
//
// 为什么不用"点击外部轮询/全局钩子": 原生弹层就是靠激活窗口 + 失焦关闭, 这套机制在
// 多显示器 / 输入法 / 模态场景下都由系统处理, 自己判定鼠标位置会漏掉键盘切窗等路径。

// popoverShadow 是弹层的阴影带宽(逻辑像素), 与 menu 的 shadow-md 同一口径。
// 曾经取 8, 弹层四周会露出一圈过宽的柔光(用户反馈"阴影区域怎么那么外面"),
// 收紧到 5 后与菜单弹层的阴影观感一致。
const popoverShadow int32 = 5

// popoverReopenFadeSkip 是"关闭后多久内重开可以跳过进场淡入":
// 用户在快速交互(连点开关)时, 每次都从透明渐显会让弹层看起来在"眨眼/抖动";
// 对齐 CSS 过渡的中断语义 —— 刚关就开, 直接以完成态出现, 不重播动画。
const popoverReopenFadeSkip = 300 * time.Millisecond

// fadeSubtreeAlpha 遍历设置子树元素透明度, 有两个例外:
//   - **跳过滚动条子树**: 时间列表的滚动条显隐由 showScrollBarOnHover 自理(初始
//     隐藏、悬停列表才显示), 若跟随弹层淡入一起设置, 每帧 apply 都会把初始隐藏的
//     滚动条拉亮, 淡入完成后才被 OnFaded 藏回去 —— 表现为"弹层刚显示时滚动条没隐藏"。
//   - **跳过无效句柄**: 原生 ListBox 的子元素里有延迟创建的内部对象(实测 type=62、
//     `XC_IsHXCGUI(h, XC_ELE)=false` 但 GetChildByIndex 照样返回非 0 句柄), 对它调
//     XEle_SetAlpha 会触发 xcgui 的 DLL 错误弹窗("SetAlpha 句柄无效")且不产生 Go
//     panic(见 DESIGN 73)。
func fadeSubtreeAlpha(hEle int, a byte) {
	if hEle == 0 || !xc.XC_IsHXCGUI(hEle, xcc.XC_ELE) {
		return
	}
	if xc.XObj_GetType(hEle) == xcc.XC_SCROLLBAR {
		return
	}
	xc.XEle_SetAlpha(hEle, a)
	n := xc.XEle_GetChildCount(hEle)
	for i := int32(0); i < n; i++ {
		if h := xc.XEle_GetChildByIndex(hEle, i); h != 0 {
			fadeSubtreeAlpha(h, a)
		}
	}
}

// popoverPanel 是一个已建好的锚定弹层。
type popoverPanel struct {
	win  *window.Window
	body *widget.Element
	cw   int32
	ch   int32
	open bool

	// activate 为 false 时不抢焦点、不因失焦关闭(悬停卡片这类"鼠标移入内容"的弹层)。
	activate bool
	// keepOnBlur 为 true 时暂时忽略失焦关闭: Menubar 打开子菜单时焦点会转移到
	// 子菜单窗口, 父弹层必须保持打开(否则子菜单在鼠标按下时被父层一起关掉)。
	keepOnBlur bool
	// paintRadius / paintBorder 覆盖卡片圆角与描边(0 = 默认 RadiusLg / foreground 10%)。
	// Menubar 的弹层是 rounded-md + border, 与 Popover 的 rounded-lg + ring 不同。
	paintRadius int32
	paintBorder xui.Color
	// bodyLayout 是启用布局的内容宿主(layoutBody = true 时非空); 公开 Popover /
	// HoverCard 的内容区建在它上面, 这样才能量出内容的自然高度。
	bodyLayout *widget.LayoutEle
	content    *widget.LayoutEle

	// OnKeyDown 键盘事件钩子: ESC 之外的自定义键位(方向键/回车等)。返回 true 表示
	// 已处理(事件标记为已吃掉, 不再继续分发)。
	OnKeyDown func(wParam uintptr) bool

	// OnEsc ESC 关闭弹层后的回调(如把焦点还给触发区)。
	OnEsc func()

	// OnHide 弹层关闭后的回调(OnOpenChange(false) 用)。ESC / 失焦 / 主动 Hide 都会走。
	OnHide func()

	// OnFaded 进场淡入到位后的回调(可选)。淡入的 SetSubtreeAlpha 会把 body 子树里
	// 所有元素的 alpha 一并拉满 —— 子控件里如果有"自己管理元素 alpha"的东西
	// (TimePicker 列表滚动条的显隐), 在这里恢复它的初始态。
	OnFaded func()

	// lastClose 记录最近一次关闭的时刻: 点触发区关闭弹层时, 弹层的 KillFocus 先于
	// 触发区的 LButtonUp 到达, 没有这个时间窗会在关闭后立刻又打开(表现为"关不掉");
	// 同时它也是"快速重开跳过淡入"的判定依据。
	lastClose time.Time
}

// newPopoverPanel 创建弹层(cw/ch 是卡片尺寸, 不含阴影带), 失败返回 nil。
// hAnchor 是触发元素句柄, 只用来取宿主窗口与 DPI。
func newPopoverPanel(cw, ch int32, hAnchor int) *popoverPanel {
	return newPopoverPanelEx(cw, ch, hAnchor, false, true)
}

// newPopoverPanelEx 是带形态参数的构造器:
//   - layoutBody: body 换成 LayoutEle, 内容区可以用布局量出自然高度(公开 Popover);
//   - activate: false = 不抢焦点、不因失焦关闭(HoverCard; 窗口加 WS_EX_NOACTIVATE)。
func newPopoverPanelEx(cw, ch int32, hAnchor int, layoutBody, activate bool) *popoverPanel {
	host := xc.XWidget_GetHWINDOW(hAnchor)
	if host <= 0 {
		return nil
	}
	band := popoverShadow
	exStyle := xcc.WS_EX_TOOLWINDOW
	if !activate {
		exStyle |= xcc.WS_EX_NOACTIVATE
	}
	win := window.NewEx(exStyle, 0, "", 0, 0, cw+band*2, ch+band*2, "", 0,
		xcc.Window_Style_Nothing)
	if win == nil {
		return nil
	}
	win.SetBorderSize(0, 0, 0, 0)
	win.SetTransparentType(xcc.Window_Transparent_Shadow)
	win.SetShadowInfo(band, 50, xui.Current().RadiusLg(), false, xc.RGBA(0, 0, 0, 128))
	win.SetTransparentAlpha(255)
	// ⚠️ 必须垫一层 alpha=0 的窗口背景填充: 弹窗默认背景是**白色**, 淡入用元素 alpha,
	// 元素全透明时白底就从卡片下面露出来 —— 深色主题下表现为"每次弹出闪一下白"
	// (逐帧抓图可见第二次显示的首帧 mean=228, 就是这层白底)。
	// 留空(不设置)会回退系统默认内容, 透明必须显式表达, 同 DESIGN 第八节 30。
	if bkm := xc.XWnd_GetBkManager(win.Handle); bkm != 0 {
		xc.XBkM_Clear(bkm)
		xc.XBkM_AddFill(bkm, xcc.CombinedState_Window_State_Flag_Leave, 0, 0)
	}

	var body *widget.Element
	var bodyLayout *widget.LayoutEle
	if layoutBody {
		bodyLayout = widget.NewLayoutEle(band, band, cw, ch, win.Handle)
		if bodyLayout == nil {
			win.DestroyWindow()
			return nil
		}
		body = &bodyLayout.Element
		bodyLayout.EnableHorizon(false)
		bodyLayout.EnableLayout(true)
		bodyLayout.EnableAutoWrap(false)
		bodyLayout.SetAlignV(xcc.Layout_Align_Top)
	} else {
		body = widget.NewElement(band, band, cw, ch, win.Handle)
		if body == nil {
			win.DestroyWindow()
			return nil
		}
	}
	body.EnableBkTransparent(true)
	p := &popoverPanel{win: win, body: body, bodyLayout: bodyLayout, cw: cw, ch: ch, activate: activate}
	body.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		paint := xui.BeginPaint(hEle, hDraw)
		if paint == nil {
			return 0
		}
		*pbHandled = true
		t := xui.Current()
		radius := p.paintRadius
		if radius <= 0 {
			radius = t.RadiusLg()
		}
		border := p.paintBorder
		if border == 0 {
			border = t.Foreground.WithAlpha(26)
		}
		paint.FillRound(&paint.Rc, t.Popover, radius)
		paint.StrokeRound(&paint.Rc, border, radius, 1)
		return 0
	})

	win.AddEvent_KillFocus(func(hWindow int, pbHandled *bool) int {
		if p.activate && p.open && !p.keepOnBlur {
			p.Hide()
		}
		return 0
	})
	win.AddEvent_KeyDown(func(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
		if wParam == xcc.VK_Esc && p.open {
			p.Hide()
			if p.OnEsc != nil {
				p.OnEsc()
			}
			*pbHandled = true
			return 0
		}
		if p.OnKeyDown != nil && p.open && p.OnKeyDown(wParam) {
			*pbHandled = true
		}
		return 0
	})
	// 弹层是独立窗口, 不在主题管理器的重绘清单里: 切主题时正挂着的那层要自己重绘
	// (与 tooltip 同一条, 见 DESIGN 第八节 58)。
	xui.OnThemeChangeFor(body.Handle, func(*xui.Theme) {
		if p.alive() {
			p.win.Redraw(true)
		}
	})
	return p
}

// alive 报告弹层窗口是否仍然存在。
func (p *popoverPanel) alive() bool {
	return p != nil && p.win != nil && xc.XC_IsHXCGUI(p.win.Handle, xcc.XC_WINDOW)
}

// Content 返回卡片内容元素的句柄(子控件建在它上面)。
func (p *popoverPanel) Content() int {
	if p == nil || p.body == nil {
		return 0
	}
	return p.body.Handle
}

// LayoutContent 在启用布局的弹层里建(或返回)内容区: 宽铺满卡片、高按内容自撑,
// padding 为四周内边距。只有 newPopoverPanelEx(..., layoutBody=true) 创建的弹层可用。
func (p *popoverPanel) LayoutContent(padding, gap int32) *widget.LayoutEle {
	if p == nil || p.bodyLayout == nil {
		return nil
	}
	if p.content != nil {
		return p.content
	}
	c := widget.NewLayoutEle(0, 0, p.cw-padding*2, 0, p.body.Handle)
	if c == nil {
		return nil
	}
	c.EnableHorizon(false)
	c.EnableLayout(true)
	c.EnableBkTransparent(true)
	c.EnableAutoWrap(false)
	c.SetAlignV(xcc.Layout_Align_Top)
	c.SetSpace(gap)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	// 左右用元素 padding 约束子项 Fill 宽, 上下用外边距计入高度 —— 四边外边距会让
	// 内容区子项按弹层全宽排版, 行容器/整宽控件向右溢出(同 card.ContentStack,
	// 见 DESIGN 第八节 129)。
	c.LayoutItem_SetMargin(0, padding, 0, padding)
	c.SetPadding(padding, 0, padding, 0)
	p.content = c
	return c
}

// MeasureContent 在屏幕外把弹层显示出来, 让内容区完成一次布局, 再把卡片高度收成
// 内容自然高度 + 上下内边距。返回内容区高度(0 表示测量失败)。
//
// 为什么必须显示: 布局只在元素真正可见时才有尺寸(探针实测, 隐藏窗口里
// LayoutEle 的 Auto 高始终是 0, GetContentSize 也不出结果)。窗口在 -30000 处
// 显示, 用户看不到; 用的是不激活的显示方式, 不会抢走触发区焦点。
func (p *popoverPanel) MeasureContent(padding int32) int32 {
	if p == nil || !p.alive() || p.content == nil {
		return 0
	}
	// ⚠️ 弹层窗口必须用 TOPMOST 抬 z 序: 宿主开了"窗口总在最前"(SetTop(true))时,
	// HWND_TOP 只把弹层放到**非顶层** band 的顶部, 仍会被 TOPMOST 的主窗口整个盖住 ——
	// 弹层"打开成功但完全看不见"(实测)。tooltip 一直用 HWND_TOPMOST; 两套弹层统一。
	p.win.SetWindowPos(xcc.HWND_TOPMOST, -30000, -30000, 0, 0,
		xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
	if p.activate {
		p.win.ShowWindow(xcc.SW_SHOW)
	} else {
		wapi.ShowWindow(p.win.GetHWND(), 4) // SW_SHOWNOACTIVATE
	}
	p.bodyLayout.AdjustLayout()
	h := p.content.GetHeight()
	p.hideNative()
	if h > 0 {
		p.resize(p.cw, h+padding*2)
	}
	return h
}

// resize 调整卡片尺寸(不含阴影带), 并同步窗口/内容宿主; 窗口尺寸传逻辑像素, 由 xcgui 做 DPI 适配。
func (p *popoverPanel) resize(cw, ch int32) {
	if p == nil || cw <= 0 || ch <= 0 {
		return
	}
	p.cw, p.ch = cw, ch
	if !p.alive() {
		return
	}
	band := popoverShadow
	xc.XEle_SetRect(p.body.Handle,
		&xc.RECT{Left: band, Top: band, Right: band + cw, Bottom: band + ch},
		false, xcc.AdjustLayout_No)
	// ⚠️ SetWindowPos 的尺寸参数是**逻辑像素**(xcgui 内部再做 DPI 缩放);
	// 曾经把尺寸乘了 sc 再传, 窗口实际比卡片大 1.5 倍, 四周多出一大片"阴影".
	p.win.SetWindowPos(xcc.HWND_TOP, 0, 0, cw+band*2, ch+band*2,
		xcc.SWP_NOMOVE|xcc.SWP_NOZORDER|xcc.SWP_NOACTIVATE)
	if p.bodyLayout != nil {
		p.bodyLayout.AdjustLayout()
	}
}

// hideNative 只隐藏原生窗口, 不改 open / lastClose 状态(测量阶段用)。
func (p *popoverPanel) hideNative() {
	if p.alive() {
		p.win.ShowWindow(xcc.SW_HIDE)
	}
}

// IsOpen 报告弹层是否打开。
func (p *popoverPanel) IsOpen() bool { return p != nil && p.open }

// RecentlyClosed 报告弹层是否刚刚关闭(用于触发区的点击防抖)。
func (p *popoverPanel) RecentlyClosed() bool {
	return p != nil && !p.open && !p.lastClose.IsZero() &&
		time.Since(p.lastClose) < 250*time.Millisecond
}

// Show 在触发元素下方弹出弹层(gap = 与元素的间距); 下方放不下翻到上方, 并夹在显示器工作区内。
func (p *popoverPanel) Show(hAnchor int, gap int32) {
	if p == nil || p.open || !p.alive() {
		return
	}
	hWnd := xc.XWidget_GetHWINDOW(hAnchor)
	if hWnd <= 0 {
		return
	}
	var rc, wr xc.RECT
	xc.XEle_GetWndClientRect(hAnchor, &rc)
	xc.XWnd_GetRect(hWnd, &wr)
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96

	x := wr.Left + rc.Left
	y := wr.Top + rc.Bottom + gap

	// 夹在**工作区**里(扣掉任务栏): 只用屏幕高度会把弹层下半截压到任务栏底下。
	// 注意传 Win32 HWND(xcgui 句柄对 Win32 无效, 会让工作区查询静默失败)。
	wa, hasWA := monitorWorkArea(xc.XWnd_GetHWND(hWnd))
	if hasWA {
		waL := int32(math.Round(float64(wa.Left) / sc))
		waT := int32(math.Round(float64(wa.Top) / sc))
		waR := int32(math.Round(float64(wa.Right) / sc))
		waB := int32(math.Round(float64(wa.Bottom) / sc))
		if y+p.ch > waB { // 下方放不下(含被任务栏挡住) → 翻到上方
			y = wr.Top + rc.Top - gap - p.ch
		}
		if y+p.ch > waB { // 上方也放不下: 贴着工作区底边
			y = waB - p.ch
		}
		if y < waT {
			y = waT
		}
		if x+p.cw > waR {
			x = waR - p.cw
		}
		if x < waL {
			x = waL
		}
	} else { // 取不到显示器信息: 退回整屏(旧行为)
		screenH := int32(float64(wapi.GetSystemMetrics(wapi.SM_CYSCREEN)) / sc)
		screenW := int32(float64(wapi.GetSystemMetrics(wapi.SM_CXVIRTUALSCREEN)) / sc)
		if y+p.ch > screenH {
			y = wr.Top + rc.Top - gap - p.ch
		}
		if y < 0 {
			y = 0
		}
		if x+p.cw > screenW {
			x = screenW - p.cw
		}
		if x < 0 {
			x = 0
		}
	}
	p.showAt(hAnchor, x, y)
}

// ShowAt 在调用方算好的位置弹出弹层(x/y 为**屏幕逻辑像素**, 与 Show 内部
// wr.Left+rc.Left 同口径), 同样夹在显示器工作区内。Select 的 item-aligned 定位用:
// 弹层盖住触发区、选中项正对触发区, 位置不是"贴着触发区下方"能表达的。
//
// 与 Show 的差异只有"位置怎么来": 钳制、屏幕外预显示、淡入全部共用。
func (p *popoverPanel) ShowAt(hAnchor int, x, y int32) {
	if p == nil || p.open || !p.alive() {
		return
	}
	hWnd := xc.XWidget_GetHWINDOW(hAnchor)
	if hWnd <= 0 {
		return
	}
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	wa, hasWA := monitorWorkArea(xc.XWnd_GetHWND(hWnd))
	if hasWA {
		waL := int32(math.Round(float64(wa.Left) / sc))
		waT := int32(math.Round(float64(wa.Top) / sc))
		waR := int32(math.Round(float64(wa.Right) / sc))
		waB := int32(math.Round(float64(wa.Bottom) / sc))
		if y+p.ch > waB {
			y = waB - p.ch
		}
		if y < waT {
			y = waT
		}
		if x+p.cw > waR {
			x = waR - p.cw
		}
		if x < waL {
			x = waL
		}
	}
	p.showAt(hAnchor, x, y)
}

// showAt 是 Show/ShowAt 的公共落位与进场路径(x/y 已夹好, 逻辑像素)。
func (p *popoverPanel) showAt(hAnchor int, x, y int32) {
	if p == nil || p.open || !p.alive() {
		return
	}
	hWnd := xc.XWidget_GetHWINDOW(hAnchor)
	if hWnd <= 0 {
		return
	}
	dpi := xc.XWnd_GetDPI(hWnd)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96

	// ⚠️ SetWindowPos 的 **x/y 是物理像素**(窗口位置要自己乘 DPI), 但 **cx/cy 是逻辑像素**
	// (xcgui 会再做一次 DPI 缩放, 见 DESIGN 第八节 119); 窗口左上角含阴影带。
	band := float64(popoverShadow)
	wx := int32((float64(x)-band)*sc + 0.5)
	wy := int32((float64(y)-band)*sc + 0.5)
	p.open = true

	// 淡入用**元素 alpha**(SetSubtreeAlpha), 不是整窗 alpha —— 整窗透明度对透明窗
	// "只认第一次调用"(创建时的 255), 之后无论设什么、怎么重绘都冻结在那个值
	// (DESIGN 57/73; menu/combobox 的弹窗淡入能用是因为它们每次弹出都重建窗口,
	// 每次都是"第一次调用"; 本面板是复用的, 整窗 alpha 路线完全不可行 —— 逐帧抓图
	// 实测两次显示都是瞬间全亮)。元素 alpha 是框架合成时乘上去的, 每帧都生效。
	//
	// 关闭后 300ms 内重开(快速连点)直接以完成态出现, 不重播动画 —— 否则弹层反复
	// "眨眼"(CSS 过渡的中断语义: 刚中断就重启, 直接终态)。
	fast := !p.lastClose.IsZero() && time.Since(p.lastClose) < popoverReopenFadeSkip
	if fast {
		fadeSubtreeAlpha(p.body.Handle, 255)
	} else {
		fadeSubtreeAlpha(p.body.Handle, 0)
	}

	// ⚠️ 屏幕外预显示: 复用窗口再次 ShowWindow 时, 首帧合成的是**窗口表面残留的
	// 上一次内容**(关闭前 alpha=1 的全亮卡片) —— 元素 alpha=0 要等第一次重绘才落地,
	// 于是屏幕上"直接显示 → 消失 → 再淡入"(逐帧抓图确认的闪烁)。
	// 先在屏幕外显示并立即重绘出"透明首帧", 再挪到目标位置: 挪动只改 DWM 合成位置,
	// 表面内容已是透明帧。Win32 窗口坐标范围 ±32768, -30000 安全。
	p.win.SetWindowPos(xcc.HWND_TOPMOST, -30000, -30000, 0, 0,
		xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE|xcc.SWP_SHOWWINDOW)
	if p.activate {
		p.win.ShowWindow(xcc.SW_SHOW)
	} else {
		// 悬停卡片不能抢焦点: 一抢, 用户正在操作的原窗口就丢焦点(输入框光标消失),
		// 而且弹层会收到 KillFocus 立刻关掉自己。
		wapi.ShowWindow(p.win.GetHWND(), 4) // SW_SHOWNOACTIVATE
	}
	// ⚠️ OnFaded 必须在这次重绘**之前**调用: 子控件的"初始隐藏态"(时间列表的滚动条
	// alpha=0)要落进首帧 —— 时机错了滚动条会随首帧一起亮出来, 直到鼠标进入列表再
	// 离开才被显隐逻辑藏掉 —— 表现为"刚弹出时滚动条是亮着的"。窗口已显示(虽在屏幕外), 子元素句柄
	// 此时全部有效, setAlphaTree 的句柄守卫不会跳过。
	if p.OnFaded != nil {
		p.OnFaded()
	}
	p.win.Redraw(true) // 在屏幕外把 alpha=0 的透明帧画好(含滚动条隐藏)
	p.win.SetWindowPos(xcc.HWND_TOPMOST, wx, wy, 0, 0,
		xcc.SWP_NOSIZE|xcc.SWP_NOACTIVATE)
	if p.activate {
		wapi.SetForegroundWindow(p.win.GetHWND()) // 拿到焦点才会在点击外部时收到 KillFocus
	}

	if fast { // 跳过淡入: 直接完成态 —— 属性已设 1, 再重绘一次落地
		p.win.Redraw(true)
		return
	}

	// 进场: fade-in-0(duration-100)。Presence 会立即以当前进度 apply 一次,
	// 之后每帧由定时器驱动。
	//
	// ⚠️ 过渡必须挂**弹窗自己**(p.win), 不能挂宿主窗口: hoverTick 每个 tick 都会
	// 立即重绘 hOwner —— 挂宿主窗 = 淡入的 100ms 里每帧全量重绘整个主窗口(demo 表单页
	// 上千个元素), 快速点击时重绘排队, UI 线程卡到出现系统忙碌光标, 弹层开关肉眼可见
	// 地"抖动/掉帧"(录屏逐帧可见)。整窗 alpha 时代挂宿主窗是被迫的
	// (DWM 合成要宿主带动, 见 DESIGN 第八节 47/73), 改元素 alpha 后弹窗自己重绘就落地,
	// 弹窗只有 400×500 物理像素, 每 tick 重绘毫无压力。
	hBody := p.body.Handle
	hw := p.win.Handle
	key := xui.NextPresenceKey()
	xui.Presence(p.win.Handle, key, true, xui.PopupDuration, func(v float64) {
		if !xc.XC_IsHXCGUI(hw, xcc.XC_WINDOW) {
			return
		}
		fadeSubtreeAlpha(hBody, byte(v*255+0.5))
		// 重绘由 hoverTick 对 hOwner(=弹窗) 的立即重绘完成, 这里不重复
	}, func(v float64) {
		if p.OnFaded != nil {
			p.OnFaded()
		}
	})
}

// Hide 关闭弹层。
func (p *popoverPanel) Hide() {
	if p == nil || !p.open {
		return
	}
	p.open = false
	p.lastClose = time.Now()
	// ① 清掉本弹窗上全部过渡条目: 淡入中途被隐藏时, 未到位条目(settled=false)会
	//    随残留定时器继续执行 apply, 与下一次进场(尤其 fast 路径的终态)互相覆盖 ——
	//    表现为弹层概率性停在半透明、透出后方文字。
	xui.DropTransitions(p.win.Handle)
	// ② 元素 alpha 归位到终态(属性): 潜在的 surface 残留帧由下次 Show 的
	//    屏幕外预显示(fast 路径的重绘)兜底。
	fadeSubtreeAlpha(p.body.Handle, 255)
	if p.OnFaded != nil { // 滚动条等子元素的初始态照常恢复
		p.OnFaded()
	}
	if p.alive() {
		p.win.ShowWindow(xcc.SW_HIDE)
	}
	if p.OnHide != nil {
		p.OnHide()
	}
}

// Destroy 销毁弹层窗口(触发元素销毁时调用)。
func (p *popoverPanel) Destroy() {
	if !p.alive() {
		return
	}
	p.open = false
	p.win.DestroyWindow()
}
