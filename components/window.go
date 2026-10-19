package ui

import (
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// AppWindow 的默认几何(逻辑像素): 标题栏 30 高, 右侧三个系统按钮各 36x28
// (hover 是圆角幽灵按钮, 与 Win11 的无间距平铺不同 —— 我们要的是和 shadcn 卡片搭)。
const (
	appWndCaptionH   int32 = 30
	appWndShadowSize int32 = 12
	appWndBtnW       int32 = 36
	appWndBtnH       int32 = 28
	appWndIconSize   int32 = 16
	// appWndIconBig 是最小化/关闭按钮的图标尺寸: minus/x 这类"一根横线/一个叉"的图形,
	// 16 在 36 宽的按钮里观感太小。
	appWndIconBig int32 = 20
)

// AppWindow 是开箱即用的主窗口: 圆角阴影 + 自绘标题栏(最小化/最大化/关闭) + 主题化底色。
//
// 为什么需要它: xcgui 的默认标题栏是系统配色/系统按钮, 与 shadcn 风格的卡片、按钮放一起
// 明显不是一套。这里把整套"无系统标题栏 + 自绘三键"的样板固化下来 —— 窗口样式、
// 阴影参数、拖动、双击最大化、最大化图标切换、主题跟随全部配好, 用户只需要往里塞内容:
//
//	w := ui.NewAppWindow("我的应用", 960, 640)
//	ui.ItemFill(ui.NewLabel(0, 0, 0, 28, "标题", w.Content().Handle))
//	body := ui.NewCard(0, 0, 0, 0, w.Content().Handle)
//	w.Show(true)
//
// 内容区 Content() 是一个纵向流式容器(宽铺满、高占满剩余), 页面/卡片按布局往里放。
type AppWindow struct {
	*window.Window

	caption   *widget.LayoutEle // 标题栏行
	titleRc   *widget.LayoutEle // 标题文字(Weight 宽, 把按钮组顶到右边)
	customBox *widget.LayoutEle // 自定义按钮区(在三个系统按钮左侧)
	content   *widget.LayoutEle // 内容区(纵向流式, Weight 高)
	titleText string
	btnMax    *Button
}

// AppWindowOption 是 AppWindow 的配置项。
type AppWindowOption func(*appWindowConfig)

type appWindowConfig struct {
	minW, minH int32
	captionH   int32
	noMinMax   bool
	icon       string
	pad        [4]int32
	padSet     bool
}

// WithAppWindowMinSize 设置窗口最小尺寸(默认 480x320)。
func WithAppWindowMinSize(w, h int32) AppWindowOption {
	return func(c *appWindowConfig) {
		if w > 0 && h > 0 {
			c.minW, c.minH = w, h
		}
	}
}

// WithAppWindowCaptionHeight 覆盖标题栏高度(默认 30)。
func WithAppWindowCaptionHeight(h int32) AppWindowOption {
	return func(c *appWindowConfig) {
		if h >= 24 {
			c.captionH = h
		}
	}
}

// WithAppWindowNoMinMax 只保留关闭按钮(做单窗口小工具/对话框时用)。
func WithAppWindowNoMinMax() AppWindowOption {
	return func(c *appWindowConfig) { c.noMinMax = true }
}

// WithAppWindowIcon 在标题左侧显示一个图标(path 数据或整段 SVG, 见 Painter.Icon)。
func WithAppWindowIcon(path string) AppWindowOption {
	return func(c *appWindowConfig) { c.icon = path }
}

// WithAppWindowContentPadding 给内容区加内边距(默认 0, 由页面自己管留白)。
func WithAppWindowContentPadding(left, top, right, bottom int32) AppWindowOption {
	return func(c *appWindowConfig) {
		c.pad = [4]int32{left, top, right, bottom}
		c.padSet = true
	}
}

// NewAppWindow 创建主窗口。标题栏与内容区都已按布局挂好, 直接往 w.Content() 里放组件。
func NewAppWindow(title string, cx, cy int32, opts ...AppWindowOption) *AppWindow {
	cfg := appWindowConfig{minW: 480, minH: 320, captionH: appWndCaptionH}
	for _, o := range opts {
		o(&cfg)
	}

	w := &AppWindow{Window: window.New(0, 0, cx, cy, title, 0,
		// 无 Icon / 无系统三键: 标题栏完全自绘(样式与 shadcn 一套)
		xcc.Window_Style_Center|xcc.Window_Style_Border|xcc.Window_Style_Drag_Window|
			xcc.Window_Style_Drag_Border|xcc.Window_Style_Allow_MaxWindow)}
	if w.Window == nil {
		return nil
	}
	w.titleText = title

	// 阴影/圆角/透明: 12px 阴影带、圆角 8
	w.SetTransparentType(xcc.Window_Transparent_Shadow)
	w.SetShadowInfo(appWndShadowSize, 128, 8, false, xc.RGBA(0, 0, 0, 128))
	w.SetTransparentAlpha(255)
	w.SetBorderSize(0, 0, 0, 0)
	w.SetDragBorderSize(1, 1, 1, 1) // 边缘拖动区收窄, 否则鼠标贴边描边会误触发改大小
	w.SetMinimumSize(cfg.minW, cfg.minH)
	w.SetFont(xui.FontHandle(xui.Current().FontDefault))

	xui.RegisterWindow(w.Handle)
	xui.BindWindowBackground(w.Handle) // 窗口底色随主题(标题栏与内容区共用)

	// 窗口本身就是布局容器: [标题栏 Fixed][内容区 Weight 1]
	w.EnableLayout(true)
	w.buildCaption(cfg)
	w.buildContent(cfg)

	// 窗口销毁时从主题注册表摘掉(句柄会被系统复用, 留着会让主题切换操作到别的窗口)
	w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			xui.UnregisterWindow(hWindow)
		}
		return 0
	})

	// 双击标题栏最大化/还原。y 判定要按窗口实际坐标系算: 非最大化时整窗矩形四周含
	// shadowSize 宽的阴影带, 标题栏落在 [shadow, shadow+captionH]; 最大化时没有阴影。
	// 点在标题栏的按钮上不触发(GetStayEle != 0)。
	w.AddEvent_LButtonDBClick(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		top := cfg.captionH + appWndShadowSize
		if w.IsMaxWindow() {
			top = cfg.captionH
		}
		if pPt.Y > top {
			return 0
		}
		// 点在标题栏的按钮上不触发。注意不能写成"StayEle != 0 就跳过": 标题/图标是
		// MouseThrough 的, 鼠标停在它们上面时 StayEle 也非 0(实测双击会整段失效) ——
		// 只有停在真正的按钮上才该拦。
		if h := w.GetStayEle(); h != 0 && xc.XC_GetObjectType(h) == xcc.XC_BUTTON {
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
	return w
}

// buildCaption 挂标题栏: [图标][标题 Weight][自定义按钮区][最小化/最大化/关闭]。
func (w *AppWindow) buildCaption(cfg appWindowConfig) {
	h := cfg.captionH

	cap := widget.NewLayoutEle(0, 0, 0, h, w.Handle)
	cap.EnableHorizon(true)
	cap.EnableLayout(true)
	cap.EnableBkTransparent(true)
	cap.SetSpace(4)
	cap.SetAlignV(xcc.Layout_Align_Center)
	cap.EnableMouseThrough(true) // 行本身穿透, 不挡窗口拖动; 子按钮不受影响
	cap.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	cap.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
	w.caption = cap

	// 标题图标(可选): 16x16, 左缩进 12
	if cfg.icon != "" {
		icon := widget.NewLayoutEle(0, 0, appWndIconSize, appWndIconSize, cap.Handle)
		icon.EnableBkTransparent(true)
		icon.AddEvent_Paint(func(hEle, hDraw int, pbHandled *bool) int {
			p := xui.BeginPaint(hEle, hDraw)
			if p == nil {
				return 0
			}
			*pbHandled = true
			p.Icon(cfg.icon, &p.Rc, xui.Current().Foreground, 2)
			return 0
		})
		icon.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, appWndIconSize)
		icon.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, appWndIconSize)
		xui.ItemMarginLeft(icon, 12)
	}

	// 标题文字: Weight 宽占满剩余空间, 把右侧按钮组顶到行尾。
	// (横向行里 Fill 会独占整行把兄弟挤下去, 剩余空间必须用 Weight)
	title := widget.NewLayoutEle(0, 0, 0, h, cap.Handle)
	title.EnableBkTransparent(true)
	title.EnableMouseThrough(true)
	title.AddEvent_Paint(func(hEle, hDraw int, pbHandled *bool) int {
		p := xui.BeginPaint(hEle, hDraw)
		if p == nil {
			return 0
		}
		*pbHandled = true
		if f := xui.FontHandle(xui.Current().FontDefault); f != 0 {
			p.Draw.SetFont(f)
		}
		p.Text(w.titleText, &p.Rc, xui.Current().Foreground,
			xcc.TextAlignFlag_Left|xcc.TextAlignFlag_Vcenter|
				xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter)
		return 0
	})
	title.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	title.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	if cfg.icon == "" {
		xui.ItemMarginLeft(title, 12)
	}
	w.titleRc = title

	// 自定义按钮区(主题切换、布局边界…): 宽度按内容自撑, 排在系统按钮左侧
	custom := widget.NewLayoutEle(0, 0, 0, h, cap.Handle)
	custom.EnableHorizon(true)
	custom.EnableLayout(true)
	custom.EnableBkTransparent(true)
	custom.SetSpace(2)
	custom.SetAlignV(xcc.Layout_Align_Center)
	custom.LayoutItem_SetWidth(xcc.Layout_Size_Auto, 0)
	custom.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	w.customBox = custom

	// 系统按钮: 顺序固定 [最小化][最大化][关闭], 不加布局项 —— 尺寸由创建参数给足,
	// 布局"不接管尺寸"就原样保留(见 xui/layout.go 的约定)。
	if !cfg.noMinMax {
		btnMin := NewButton(0, 0, appWndBtnW, appWndBtnH, "", cap.Handle,
			WithVariant(ButtonGhost), WithIcon(icons.IconMinus), WithIconSize(appWndIconBig))
		btnMin.SetTypeEx(xcc.Button_Type_Min)

		w.btnMax = NewButton(0, 0, appWndBtnW, appWndBtnH, "", cap.Handle,
			WithVariant(ButtonGhost), WithIcon(icons.IconSquare), WithIconSize(appWndIconSize))
		w.btnMax.SetTypeEx(xcc.Button_Type_Max)

		// 最大化/还原换图标: 按钮类型只管行为, 自绘图标要跟窗口状态走。
		// 挂 WM_SIZE 而不是只挂点击回调 —— 双击标题栏 / Win+↑ / 系统菜单改状态都经过它;
		// 延后到消息处理之后(XC_CallUT)再读 IsMaxWindow, 拿到的才是新状态。
		w.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
			if message == wapi.WM_SIZE && (wParam == wapi.SIZE_MAXIMIZED || wParam == wapi.SIZE_RESTORED) {
				btnMax := w.btnMax
				win := w.Window
				xc.XC_CallUT(func() {
					if btnMax != nil {
						btnMax.SetIcon(chooseIcon(win.IsMaxWindow(), icons.IconCopy, icons.IconSquare))
					}
				})
			}
			return 0
		})
	}

	btnClose := NewButton(0, 0, appWndBtnW, appWndBtnH, "", cap.Handle,
		WithVariant(ButtonGhost), WithIcon(icons.IconX), WithIconSize(appWndIconBig))
	btnClose.SetTypeEx(xcc.Button_Type_Close)
	xui.ItemMarginRight(btnClose, 4) // 关闭钮与圆角之间留一点, 避免 hover 底色贴边
}

// buildContent 挂内容区: 纵向流式容器, 宽铺满、高占满标题栏以下的剩余空间。
func (w *AppWindow) buildContent(cfg appWindowConfig) {
	content := widget.NewLayoutEle(0, 0, 0, 0, w.Handle)
	content.EnableHorizon(false)
	content.EnableLayout(true)
	content.EnableBkTransparent(true)
	content.EnableAutoWrap(false) // 纵向溢出换列会让控件整块消失(DESIGN 41)
	content.SetSpace(12)
	content.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	content.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)
	if cfg.padSet {
		content.SetPadding(cfg.pad[0], cfg.pad[1], cfg.pad[2], cfg.pad[3])
	}
	w.content = content
}

// Content 返回内容区容器(纵向流式): 页面/卡片按布局往里放, 不写绝对坐标。
func (w *AppWindow) Content() *widget.LayoutEle {
	return w.content
}

// Caption 返回标题栏行容器(高 = 标题栏高), 想自己往标题栏塞东西时用。
// 常规用法是 AddCaptionButton; 自己建控件时要补布局项尺寸(Fixed/Auto 等)。
func (w *AppWindow) Caption() *widget.LayoutEle {
	return w.caption
}

// AddCaptionButton 在标题栏右侧加一个图标按钮(排在最小化按钮左边),
// 用来放"切换主题 / 设置 / 布局边界"这类按钮。返回按钮便于继续绑事件。
func (w *AppWindow) AddCaptionButton(icon string, onClick func()) *Button {
	if w.customBox == nil {
		return nil
	}
	b := NewButton(0, 0, appWndBtnW, appWndBtnH, "", w.customBox.Handle,
		WithVariant(ButtonGhost), WithIcon(icon), WithIconSize(appWndIconSize))
	if onClick != nil {
		b.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			onClick()
			return 0
		})
	}
	w.customBox.AdjustLayout()
	w.caption.AdjustLayout()
	return b
}

// SetTitle 同时更新自绘标题栏与窗口标题(任务栏/Alt+Tab 用)。
func (w *AppWindow) SetTitle(title string) {
	w.titleText = title
	w.Window.SetTitle(title)
	if w.titleRc != nil {
		w.titleRc.Redraw(false)
	}
}

// chooseIcon 是三元选择的空串安全版(选中 a, 否则 b)。
func chooseIcon(on bool, a, b string) string {
	if on {
		return a
	}
	return b
}
