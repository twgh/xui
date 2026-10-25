package ui

import (
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 对话框几何常量(与 shadcn dialog.tsx 的类名逐条对应, 单位逻辑像素):
const (
	dlgPad       int32 = 16  // p-4
	dlgGap       int32 = 16  // gap-4: 头部/内容/页脚三组之间
	dlgHeadGap   int32 = 8   // DialogHeader 的 gap-2
	dlgFooterGap int32 = 8   // DialogFooter 的 gap-2
	dlgDefaultW  int32 = 384 // sm:max-w-sm(24rem)
	dlgCloseOff  int32 = 8   // 关闭按钮 top-2 right-2
	dlgCloseSize int32 = 28  // size-7(按钮 icon-sm)
	dlgTitleH    int32 = 16  // text-base(16) + leading-none
	dlgDescH     int32 = 20  // text-sm(14) 默认行高 1.25rem
	dlgBtnH      int32 = 32  // 页脚按钮默认 h-8
	dlgBtnPadX   int32 = 10  // 按钮 px-2.5
	dlgFooterH   int32 = dlgPad*2 + dlgBtnH
	// dlgTextOver: 文本元素的上下外扩量。行盒按 CSS 的 leading-none/默认行高算,
	// 但元素内文字是垂直居中且被裁在元素内, 需要外扩几个像素才放得下字形
	dlgTextOver int32 = 3
	// 原生 Button 载体绘制文字时会再内缩一圈: 量折行高度要用收窄后的宽度,
	// 否则最后一行会被裁掉并出现省略号(同 alert.go, 见 DESIGN 第八节 90)
	dlgTextInsetX int32 = 12
	dlgTextPadY   int32 = 4
)

// AlertDialog 几何常量(对齐 alert-dialog.tsx (radix-nova), 单位逻辑像素):
const (
	alertDlgSmallW     int32 = 320 // max-w-xs(20rem)
	alertDlgDefaultW   int32 = 384 // data-[size=default]:sm:max-w-sm
	alertDlgHeadGap    int32 = 6   // AlertDialogHeader 的 gap-1.5
	alertDlgTitleH     int32 = 24  // text-base(16) 默认行高 1.5rem(没有 leading-none)
	alertDlgMediaSize  int32 = 40  // AlertDialogMedia 的 size-10
	alertDlgMediaIcon  int32 = 24  // media 里的 svg 是 size-6
	alertDlgMediaGapX  int32 = 16  // has-media 时的 gap-x-4
	alertDlgMediaGapY  int32 = 8   // media 的 mb-2(小尺寸上下堆叠时)
	alertDlgSmallBreak int32 = 640 // tailwind 的 sm 断点: 更窄时 default 也用 xs 宽
)

// AlertDialogSize 对应 alert-dialog.tsx 的 size prop(类型放在 dialog.go: 对话框面板
// 布局要按尺寸算宽度与页脚排列, dialog 单独安装时也必须能编译)。
type AlertDialogSize int

const (
	// AlertDialogSizeDefault: 宽 320/384(窄窗口 320, 桌面 384), 头部左对齐, 页脚右对齐。
	AlertDialogSizeDefault AlertDialogSize = iota
	// AlertDialogSizeSmall: 宽 320, 头部居中, 页脚两个按钮等宽双列(grid-cols-2)。
	AlertDialogSizeSmall
)

// AlertDialogMediaTone 控制 AlertDialogMedia 图标的配色。
type AlertDialogMediaTone int

const (
	// AlertDialogMediaMuted: bg-muted + text-muted-foreground(默认)。
	AlertDialogMediaMuted AlertDialogMediaTone = iota
	// AlertDialogMediaDestructive: bg-destructive/10 + text-destructive(暗色 /20)。
	AlertDialogMediaDestructive
)

// alertDialogPanelWidth 按尺寸与 Body 宽算 AlertDialog 面板宽:
// sm 恒 320; default 在 <640 断点为 320、否则 384; 都不超过 Body 宽 − 2rem。
func alertDialogPanelWidth(size AlertDialogSize, bodyW int32) int32 {
	pw := alertDlgDefaultW
	if size == AlertDialogSizeSmall || bodyW < alertDlgSmallBreak {
		pw = alertDlgSmallW
	}
	if max := bodyW - 2*dlgPad; pw > max {
		pw = max
	}
	return pw
}

// 对话框用到的虚拟键码。
const (
	vkDlgTab    uintptr = 9
	vkDlgEscape uintptr = 27
	vkDlgShift  int32   = 0x10
)

// Dialog 模态对话框: 一个铺满 Body(内容区)的遮罩元素 + 居中面板, 面板内容全部自绘。
//
// 与 shadcn dialog.tsx(radix-nova) 对齐:
//   - 遮罩: fixed inset-0 bg-black/10, 覆盖整个 Body(内容区)并拦截下层全部鼠标事件;
//     **不铺到阴影窗口的阴影带上** —— 铺上去会把窗口自身投影再压暗 10%, 四角最明显
//     (见 layout 里的说明);
//   - 面板: sm:max-w-sm(384) rounded-xl(半径 radius×1.4) bg-popover p-4,
//     组间距 gap-4, ring-1 ring-foreground/10;
//   - 标题 text-base / 描述 text-sm text-muted-foreground, 两者间距 gap-2;
//   - 关闭按钮 = 幽灵图标按钮 size-7(28×28), 位于右上角内缩 8px, 图标 lucide X;
//   - 页脚: -mx-4 -mb-4 贴到面板左右与底边(只有下面两个角是圆角),
//     border-t + bg-muted/50, 内边距 p-4, 按钮右对齐 gap-2;
//   - 交互: ESC / 点击遮罩 / 关闭按钮关闭; 打开时焦点进入第一个可聚焦控件,
//     Tab 在对话框内部循环(不会逃逸到窗口其它元素)。
//
// 与 shadcn 的差异: backdrop-blur-xs 用"抓窗快照 + Go 盒式模糊 + 元素 BkM 图片填充"实现
// (xcgui 没有背景模糊, 见 backdrop.go 与 DESIGN.md 第八节 47), 遮罩本身仍是固定黑色 10%。
type Dialog struct {
	hWnd int
	win  *window.Window

	overlay     *widget.Element // 遮罩(窗口子元素, 顶层 z 序)
	backdrop    *widget.Element // 背景模糊层(shadcn 的 backdrop-blur-xs, 见 backdrop.go)
	backdropImg int             // 模糊快照的图片句柄(每次 Show 刷新, 旧的释放)
	content     *widget.Element // 内容容器(用户往这里放输入框等)
	title       *Label
	desc        *Label
	closeBtn    *Button
	footers     []*Button

	titleText string
	descText  string
	showClose bool

	// AlertDialog 形态(由 alertdialog.go 配置, Dialog 自身不设置):
	//   - alert: 头部间距/行高换 alert-dialog.tsx 的一套, 页脚可按尺寸走双列;
	//   - alertSize: default = 左对齐头 + 右对齐页脚, sm = 居中头 + 等宽双列页脚;
	//   - media: 40×40 的图标块(可带破坏性配色);
	//   - dismissOutside = false: 点遮罩不关闭(radix AlertDialog 的 onInteractOutside preventDefault);
	//   - initialFocus: 打开时强制聚焦的元素(AlertDialog 聚焦 Cancel, 而不是内容区首项)。
	alert          bool
	alertSize      AlertDialogSize
	media          *widget.Element
	mediaIcon      string
	mediaTone      AlertDialogMediaTone
	dismissOutside bool
	initialFocus   int

	// 可滚动内容: 内容区外包一个 ScrollArea, 头部/页脚固定(对应文档的
	// "Scrollable Content / With Sticky Footer")。scrollMax <= 0 时用 70vh。
	scroll    *ScrollArea
	scrollOn  bool
	scrollMax int32

	// sheet/drawer 形态(由 sheet.go 配置): 面板贴边而非居中, 四向侧边 + 滑入。
	// 面板本身仍是 overlay 的自绘, 子元素按最终位置排好后整体平移(slideX/slideY)。
	sheet       bool
	drawer      bool
	sheetSide   SheetSide
	sheetW      int32
	sheetPanelW int32
	durOpen     time.Duration
	durClose    time.Duration
	slideX      int32
	slideY      int32
	appliedDX   int32
	appliedDY   int32
	dragger     *widget.Element // Drawer 顶部抓手(仅底部形态)
	dragging    bool
	dragStartY  int32
	dragBaseY   int32

	// fadeKey 是本实例的 presence 键(见 xui.NextPresenceKey): 所有对话框共用宿主窗口的
	// 键空间, 固定键会撞上别的弹层或上一个对话框已完成的过渡(过渡不启动 → 对话框全透明)。
	fadeKey   int32
	width     int32
	open      bool
	suspended bool // 最小化挂起中: 子树已同步隐藏, 还原后重新 Show(见构造函数注释)
	focusIdx  int  // 焦点序列的当前位置(Tab 循环的兜底基准)
	lastFocus int  // 最近一次真实聚焦的控件(关闭上层模态后还原焦点用)

	panel    xc.RECT // 面板矩形(遮罩局部坐标 —— 遮罩贴在 Body 上, 原点不再是窗口 (0,0))
	footerRc xc.RECT // 页脚带矩形(同上)

	onClose []func()
}

// 模态栈: 一个宿主窗口上可能叠多层(例如对话框里再开一个 AlertDialog)。
// 只有栈顶响应 ESC / Tab / 遮罩点击, 否则一次 ESC 会把两层一起关掉,
// 两层 KeyDown 回调还会互相抢焦点。
var modalStack []*Dialog

// DialogOption 对话框配置项, 构造时传入。
type DialogOption func(*Dialog)

// WithDialogWidth 覆盖面板宽度(默认 384 = sm:max-w-sm)。
func WithDialogWidth(w int32) DialogOption { return func(d *Dialog) { d.width = w } }

// WithDialogCloseButton 是否显示右上角关闭按钮(默认显示)。
func WithDialogCloseButton(b bool) DialogOption { return func(d *Dialog) { d.showClose = b } }

// WithDialogScroll 让内容区可以纵向滚动(头部与页脚固定, 对应 shadcn 文档的
// "Scrollable Content / With Sticky Footer"): 内容放进 ScrollArea(), 高度超过
// maxHeight 后出滚动条; maxHeight <= 0 时取窗口 Body 高的 70%。
func WithDialogScroll(maxHeight int32) DialogOption {
	return func(d *Dialog) { d.scrollOn = true; d.scrollMax = maxHeight }
}

// NewDialog 创建对话框(默认隐藏)。hWnd 为宿主窗口句柄, 对话框覆盖整个窗口。
func NewDialog(hWnd int, opts ...DialogOption) *Dialog {
	win := app.NewWindowByHandle(hWnd)
	if win == nil {
		return nil
	}
	d := &Dialog{hWnd: hWnd, win: win, width: dlgDefaultW, showClose: true,
		dismissOutside: true, fadeKey: xui.NextPresenceKey()}
	for _, o := range opts {
		o(d)
	}

	// 遮罩: 半透明黑 + 面板全部在 onPaint 里绘制; 元素本身透明, 因此只是
	// "一层画布", 同时天然拦截下层元素的鼠标事件(实测: 覆盖后下层按钮收不到点击)。
	d.overlay = widget.NewElement(0, 0, 10, 10, hWnd)
	if d.overlay == nil {
		return nil
	}
	d.overlay.EnableBkTransparent(true)
	// 原生焦点矩形必须灭掉: 点击遮罩/内容容器会聚焦到元素上, 原生会给它画一个
	// 蓝色虚线框(实测: 点面板内空白处 → 内容容器上出现蓝框, 压住输入框左右描边)
	d.overlay.EnableFocus(false)
	d.overlay.EnableDrawBorder(false)
	d.overlay.AddEvent_Paint(d.onPaint)
	d.overlay.AddEvent_LButtonDown(d.onOverlayDown)
	// modal 打开时阻止滚轮穿透到页面的 ScrollArea(浏览器里 body 不滚);
	// 对话框内的 ScrollArea / 原生列表在子元素上就把滚轮消费掉, 不会到这里
	d.overlay.EnableEvent_XE_MOUSEWHEEL(true)
	d.overlay.AddEvent_MouseWheel(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		*pbHandled = true
		return 0
	})

	// 背景模糊层: 抓窗快照 + 盒式模糊, 在遮罩之下、其余内容之上(见 backdrop.go)
	d.backdrop = newBackdrop(hWnd)

	// shadcn: text-base font-medium(500) —— 16px + 中等字重(不是 Bold 的 17px)
	d.title = NewLabel(0, 0, 0, dlgTitleH, "", d.overlay.Handle).SetMedium()
	d.title.SetFontSize(16)
	// 折行: 窄面板/长标题不会截断; 高度由 layout 按面板宽量出(见 textHeight)
	d.title.SetWrap()
	d.desc = NewLabel(0, 0, 0, dlgDescH, "", d.overlay.Handle).SetMuted().SetWrap()

	d.content = widget.NewElement(0, 0, 0, 0, d.overlay.Handle)
	d.content.EnableBkTransparent(true)
	d.content.EnableSwitchFocus(false) // 容器本身不进 Tab 序列
	// 原生焦点框与边框都要灭: 点击容器会聚焦到它, 框架会给它画蓝框/深色边框
	d.content.EnableFocus(false)
	d.content.EnableDrawBorder(false)

	if d.scrollOn {
		d.scroll = NewScrollArea(0, 0, 0, 0, d.content.Handle)
		if d.scroll != nil {
			d.scroll.EnableSwitchFocus(false)
			d.scroll.Content().EnableSwitchFocus(false)
		}
	}

	if d.showClose {
		d.closeBtn = NewButton(0, 0, dlgCloseSize, 0, "", d.overlay.Handle,
			WithVariant(ButtonGhost), WithSize(ButtonSizeSm), WithIcon(icons.IconX))
		if d.closeBtn != nil {
			d.closeBtn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
				d.Close()
				return 0
			})
		}
	}

	// 键盘与尺寸: 一个窗口下所有对话框各注册一次(AddEvent* 复用回调, 无 2000 上限风险)。
	win.AddEvent_KeyDown(d.onWindowKeyDown)
	win.AddEvent_Size(func(hWindow int, nFlags uint, pPt *xc.SIZE, pbHandled *bool) int {
		if d.open {
			d.layout()
			xc.XWnd_Redraw(d.hWnd, false)
		}
		return 0
	})
	// ⚠️ 异型阴影窗口的引擎级 bug: 开着对话框最小化再还原, 整窗元素绘制/命中全部
	// 失效(只剩 backdrop 的静态快照图; 元素树查询与绘制回调看似正常但输出不上屏、
	// 鼠标不到达), 组件层的"事后修复"都救不回来 —— 重抬 z 序/重显子树/整窗重绘/
	// 整套重物化(延迟到消息循环里做)全部实测无效; 对话框不可见时还原完全正常
	// (见 DESIGN 第八节 63)。规避 = 挂起/恢复: 最小化时把对话框整棵子树**同步隐藏**
	// (open 状态保留, 不触发 onClose), 还原后重新 Show —— 引擎全程只见"隐藏状态的
	// 还原", 走的是已验证可靠的路径, 用户还原后看到的还是原弹窗。
	// 注意两个时序坑: ①程序化最小化不发 WM_ACTIVATE(inactive), 只能靠
	// WM_SIZE/SIZE_MINIMIZED 抓时机; ②WM_ACTIVATE/WM_SIZE 里直接操作元素会把
	// 还原流程的合成打断(实测全窗白屏), 恢复动作放 AfterFunc 中延迟到消息循环。
	win.AddEvent_WindProc(func(hWindow int, message uint32, wParam, lParam uintptr, pbHandled *bool) int {
		if message == wapi.WM_DESTROY {
			dropModalStack(d.hWnd)
			return 0
		}
		if message == wapi.WM_SIZE {
			switch wParam {
			case 1: // SIZE_MINIMIZED: 挂起 —— 同步隐藏整组(Hide 的淡出完成回调是异步的, 最小化期间不会执行)
				if d.open && !d.suspended {
					d.suspended = true
					xc.XWnd_SetFocusEle(d.hWnd, 0)
					showEleTree(d.overlay.Handle, false)
					if d.backdrop != nil {
						d.backdrop.Show(false)
					}
				}
			case 0, 2: // SIZE_RESTORED / SIZE_MAXIMIZED: 预约恢复(AfterFunc(0) 不像 SetTimer 会去重,
				// 还原会连发多条 WM_SIZE → 调度多个回调, 靠回调内的 suspended 判重)
				if d.suspended && d.open {
					app.AfterFunc(0, func() {
						if !d.suspended {
							return
						}
						d.suspended = false
						if d.open {
							// 还原后不能走 Show() 的 Presence 淡入 —— 实测还原后立即启动的过渡
							// 帧不会被驱动(整组 alpha 卡在 0, 什么都看不见), 先无动画直接显示;
							// 之后用户再打开仍走正常淡入路径
							d.materialize()
							// 滚动条的 alpha 是它自己的悬停显隐状态, 不能被整组还原拉亮
							xui.SetSubtreeAlphaSkipScrollBars(1, d.overlay.Handle, d.backdropHandle())
							if d.scroll != nil {
								resetScrollBarHover(d.scroll.Handle)
							}
							d.overlay.Redraw(false)
							xc.XWnd_Redraw(d.hWnd, true)
							d.focusIdx = 0
							// 多层模态一起还原时, 只有栈顶抢焦点(各实例的 WM_SIZE 都会跑)
							if d.isTopModal() {
								if fs := d.focusables(); len(fs) > 0 {
									h := d.defaultFocus(fs)
									d.lastFocus = h
									d.setCaretWidthOf(h, 1) // 挂起时清过焦点, caret 已销毁, 重建即到位
									xc.XWnd_SetFocusEle(d.hWnd, h)
								}
							}
						}
					})
				}
			}
		}
		return 0
	})

	showEleTree(d.overlay.Handle, false)
	return d
}

// SetTitle 设置标题(text-base), 传空串隐藏标题。
func (d *Dialog) SetTitle(s string) *Dialog {
	if d == nil {
		return d
	}
	d.titleText = s
	d.title.SetText(s) // 先落地: layout 里的 WrappedHeight 读的是 Label 当前文本
	d.title.Show(s != "")
	d.relayoutIfOpen()
	return d
}

// SetDescription 设置描述(text-sm, 弱化色), 传空串隐藏描述。
func (d *Dialog) SetDescription(s string) *Dialog {
	if d == nil {
		return d
	}
	d.descText = s
	d.desc.SetText(s)
	d.desc.Show(s != "")
	d.relayoutIfOpen()
	return d
}

// relayoutIfOpen 打开状态下内容变化时重排并整窗重绘。
func (d *Dialog) relayoutIfOpen() {
	if d == nil || !d.open {
		return
	}
	d.layout()
	d.overlay.Redraw(false)
	xc.XWnd_Redraw(d.hWnd, true)
}

// Content 返回内容容器句柄: 把输入框等元素建在它里面(坐标相对容器左上角),
// 容器宽度 = ContentWidth()。面板高度由容器内子元素的实际占位自动量出。
func (d *Dialog) Content() int {
	if d == nil || d.content == nil {
		return 0
	}
	if d.scroll != nil {
		return d.scroll.Content().Handle
	}
	return d.content.Handle
}

// ScrollArea 返回可滚动内容容器(仅在 NewDialog 传入 WithDialogScroll 后非 nil)。
// 往它的 Content()/ContentStack() 里放内容; Content() 也直接返回这个内容区。
func (d *Dialog) ScrollArea() *ScrollArea {
	if d == nil {
		return nil
	}
	return d.scroll
}

// ContentWidth 内容容器宽度(面板宽 - 左右 padding), 创建内容元素时作参考宽度。
func (d *Dialog) ContentWidth() int32 {
	w := d.width
	if w <= 0 {
		w = dlgDefaultW
	}
	if d.alert {
		w = alertDlgDefaultW
		if d.alertSize == AlertDialogSizeSmall {
			w = alertDlgSmallW
		}
	}
	return w - 2*dlgPad
}

// addFooterButton 创建页脚按钮并登记(不绑定点击、不重排); AlertDialog 与 AddFooter 共用。
func (d *Dialog) addFooterButton(text string, variant ButtonVariant) *Button {
	// 用运行时 TextWidth 量单行宽(GetTextShowRect 传 width=0 会按"一个字符宽折行",
	// 返回的 CX 只是一列宽 —— 英语/多字文本的按钮会被挤成两行, 见 DESIGN 第八节 48)
	w := int32(0)
	if f := xui.FontHandle(xui.FontSpec{
		Family: xui.Current().FontDefault.Family,
		Size:   buttonSpecs[ButtonSizeDefault].font,
	}); f != 0 {
		w = xui.TextWidth(text, f)
	}
	if w <= 0 {
		w = 40
	}
	w += 2 * dlgBtnPadX // px-2.5
	btn := NewButton(0, 0, w, 0, text, d.overlay.Handle, WithVariant(variant))
	if btn == nil {
		return nil
	}
	d.footers = append(d.footers, btn)
	showEleTree(btn.Handle, d.open)
	return btn
}

// AddFooter 在页脚追加一个按钮(右对齐排列, 高度用默认 h-8, 宽度按文字量出)。
// onClick 为 nil 时只画按钮不绑事件。
func (d *Dialog) AddFooter(text string, variant ButtonVariant, onClick func()) *Dialog {
	if d == nil {
		return d
	}
	btn := d.addFooterButton(text, variant)
	if btn == nil {
		return d
	}
	if onClick != nil {
		btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			onClick()
			return 0
		})
	}
	d.relayoutIfOpen()
	return d
}

// BindClose 把任意可点击控件绑成"点击即关闭"(= shadcn 的 DialogClose)。
func (d *Dialog) BindClose(t clickBinder) *Dialog {
	if d == nil || t == nil {
		return d
	}
	t.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		d.Close()
		return 0
	})
	return d
}

// Refresh 重新量内容并重排重绘: 打开状态下改过内容(增删内容元素、换文本、调宽度)后调用。
func (d *Dialog) Refresh() *Dialog {
	if d == nil {
		return d
	}
	d.relayoutIfOpen()
	return d
}

// OnClose 注册关闭回调(ESC / 点击遮罩 / 关闭按钮 / 调 Close 都会触发)。
func (d *Dialog) OnClose(fn func()) *Dialog {
	if d == nil {
		return d
	}
	d.onClose = append(d.onClose, fn)
	return d
}

// materialize 重拍背景快照 + 显示整组 + 布局 + 抬 z 序(Show 与最小化恢复共用)。
// 先抓快照再显示再布局: 内容高度量的是"可见子元素"的占位, 隐藏状态下量出来是 0;
// z 序是模糊层先抬顶、遮罩再抬顶压住它(两者都在其余内容之上)。
func (d *Dialog) materialize() {
	if !d.sheet {
		d.refreshBackdrop()
	}
	showEleTree(d.overlay.Handle, true)
	d.layout()
	raiseModalStack(d.hWnd)
}

// raiseModalStack 按打开顺序重抬所有对话框的 z 序: 后打开的(栈顶)必须压在
// 先打开的上面。逐个"backdrop 再 overlay"地抬, 最后一个对话框的 overlay 在最高层。
func raiseModalStack(hWnd int) {
	for _, x := range modalStack {
		if x == nil || x.hWnd != hWnd || !x.open || x.overlay == nil {
			continue
		}
		if x.backdrop != nil {
			x.backdrop.SetZOrderEx(0, xcc.Zorder_Top)
		}
		x.overlay.SetZOrderEx(0, xcc.Zorder_Top)
	}
}

// pushModal 把对话框压入模态栈(重复打开时移到栈顶)。
func (d *Dialog) pushModal() {
	d.popModal()
	modalStack = append(modalStack, d)
}

// popModal 从模态栈移除自己。
func (d *Dialog) popModal() {
	for i := len(modalStack) - 1; i >= 0; i-- {
		if modalStack[i] == d {
			modalStack = append(modalStack[:i], modalStack[i+1:]...)
			return
		}
	}
}

// dropModalStack 丢弃某个宿主窗口上的模态记录(窗口销毁时调用, 避免句柄复用后
// 旧对话框还在栈里冒充"栈顶")。
func dropModalStack(hWnd int) {
	out := modalStack[:0]
	for _, x := range modalStack {
		if x == nil || x.hWnd != hWnd {
			out = append(out, x)
		}
	}
	modalStack = out
}

// topModal 返回指定宿主窗口上的栈顶对话框(没有则 nil)。
func topModal(hWnd int) *Dialog {
	for i := len(modalStack) - 1; i >= 0; i-- {
		if x := modalStack[i]; x != nil && x.hWnd == hWnd && x.open {
			return x
		}
	}
	return nil
}

// isTopModal 报告自己是不是宿主窗口上的栈顶模态。
func (d *Dialog) isTopModal() bool {
	return topModal(d.hWnd) == d
}

// clickBinder 任何带原生点击事件的控件(Button / Label / Checkbox / Radio / Switch ...)。
type clickBinder interface {
	AddEvent_BnClick(fn xc.XE_BNCLICK1, allowAddingMultiple ...bool) int
}

// BindTrigger 把控件绑成"点击即打开对话框"(= shadcn 的 DialogTrigger), 例如:
//
//	ui.NewDialog(w.Handle).BindTrigger(ui.NewButton(24, 24, 120, 0, "编辑资料", w.Handle))
func (d *Dialog) BindTrigger(t clickBinder) *Dialog {
	if d == nil || t == nil {
		return d
	}
	t.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		d.Show()
		return 0
	})
	return d
}

// Show 打开对话框: 居中定位、抬到最顶层、焦点进入第一个可聚焦控件。
func (d *Dialog) Show() *Dialog {
	if d == nil || d.overlay == nil {
		return d
	}
	if d.sheet {
		return d.showSheet()
	}
	d.open = true
	d.pushModal()
	d.materialize()
	if d.scroll != nil {
		// shadcn 每次打开都重新挂载内容, 滚动位置回到顶部; 本实现复用元素, 手动归零
		d.scroll.ScrollPosYV(0)
	}
	// 进场: 整组(模糊层 + 遮罩 + 面板 + 全部子元素)从全透明淡入 —— 对齐 shadcn dialog 的
	// data-open:animate-in fade-in-0 zoom-in-95(时长 100ms); 缩放在 xcgui 里做不到
	// (元素级 ani 对自绘元素无效, 见 DESIGN 第八节 47), 这里只做淡入。
	// 注意必须整棵子树逐个设 alpha: 元素透明度不级联, 只设 overlay 的话标题/输入框
	// 等子元素会全不透明压在还没模糊的背景上(实测踩坑)
	// 滚动条不参与整组淡入(它的 alpha 是悬停显隐状态); 每次打开先归零,
	// 否则上次关闭时亮着的滚动条会带到这次(快速重开时宿主可能没收到 Show(false))。
	xui.SetSubtreeAlphaSkipScrollBars(0, d.overlay.Handle, d.backdropHandle())
	if d.scroll != nil {
		resetScrollBarHover(d.scroll.Handle)
	}
	d.overlay.Redraw(false)
	xc.XWnd_Redraw(d.hWnd, true)
	d.focusIdx = 0
	fs := d.focusables()
	focus := d.defaultFocus(fs)
	if focus != 0 {
		// 插入符宽度只在 caret 创建(获得焦点)时生效 —— 对已显示的 caret 改宽度无效,
		// 清焦点再聚焦才生效(实测)。所以聚焦前先把宽度置 0, 淡入期间插入符隐身;
		// 到位后恢复宽度并重铸焦点让插入符回来(焦点环则始终随整组淡入, 不闪)
		d.setCaretWidthOf(focus, 0)
		xc.XWnd_SetFocusEle(d.hWnd, focus)
		d.lastFocus = focus
	}
	xui.Presence(d.hWnd, d.fadeKey, true, xui.PopupDuration, d.applyFade, func(float64) {
		if focus != 0 {
			d.setCaretWidthOf(focus, 1)
			if xc.XWnd_GetFocusEle(d.hWnd) == focus {
				xc.XWnd_SetFocusEle(d.hWnd, 0)
				xc.XWnd_SetFocusEle(d.hWnd, focus)
			}
		}
	})
	return d
}

// Hide 关闭对话框(不触发 OnClose 回调): 先淡出(100ms), 到位后再隐藏并整窗重绘。
// d.open 立刻置 false, 淡出期间键盘/遮罩点击都已不再生效。
func (d *Dialog) Hide() *Dialog {
	if d == nil || d.overlay == nil || !d.open {
		return d
	}
	if d.sheet {
		return d.hideSheet()
	}
	d.open = false
	d.popModal()
	// 清焦点: 插入符随焦点消失, 淡出的 100ms 里不会闪现全亮度的插入符,
	// 也避免键盘输入继续打进正在消失的输入框
	d.suspended = false // 挂起中被隐藏(如还原前按了 ESC): 清标记, 否则下次最小化不会挂起
	xc.XWnd_SetFocusEle(d.hWnd, 0)
	// 叠在别的对话框上时(如对话框里再开 AlertDialog), 关闭后把焦点还给下层
	restoreModalFocus(d.hWnd)
	xui.Presence(d.hWnd, d.fadeKey, false, xui.PopupDuration, d.applyFade, func(v float64) {
		showEleTree(d.overlay.Handle, false)
		if d.backdrop != nil {
			d.backdrop.Show(false)
		}
		// 遮罩是半透明合成, 必须以整窗重绘还原下方内容(局部重绘会留下变暗残影)
		xc.XWnd_Redraw(d.hWnd, true)
	})
	return d
}

// backdropHandle 返回模糊层句柄(未创建时为 0, SetElementsAlpha 对 0 忽略)。
func (d *Dialog) backdropHandle() int {
	if d == nil || d.backdrop == nil {
		return 0
	}
	return d.backdrop.Handle
}

// applyFade 把进出场进度落到对话框整组元素的透明度上(元素透明度会向下作用于自绘内容,
// 实测有效; 但**不级联到子元素**, 所以对整棵子树逐个设置 —— 遮罩、面板、模糊层与
// 标题/输入框等内容一起淡; 焦点环也因此从第一帧就随整组淡入, 不会在最后闪一下,
// 对齐 shadcn)。打开时跳过滚动条: 滚动条的 alpha 是"悬停显隐"状态, 由
// showScrollBarOnHover 自理, 跟整组淡入一起设 1 会让它打开即亮(见 DESIGN 98)。
// 输入框插入符竖条不随元素 alpha 淡(原生绘制, 实测) —— 由 setCaretWidth 在进出场
// 期间置 0 隐身, 不在这里处理。
func (d *Dialog) applyFade(v float64) {
	if d.open {
		// 打开: 跳过滚动条, 等鼠标真正移入内容区再由 showScrollBarOnHover 淡入。
		xui.SetSubtreeAlphaSkipScrollBars(v, d.overlay.Handle, d.backdropHandle())
		return
	}
	// 关闭: 滚动条跟着一起淡出(不做例外, 否则半透明的面板上会留一条亮滚动条)。
	xui.SetSubtreeAlpha(v, d.overlay.Handle, d.backdropHandle())
}

// setCaretWidthOf 把指定编辑框的插入符宽度设为 w(0 = 隐身)。只处理 XC_EDIT。
// 注意宽度仅在 caret 创建(获得焦点)时生效, 改已显示 caret 的宽度要清焦点再重聚焦。
func (d *Dialog) setCaretWidthOf(hEle int, w int32) {
	if hEle != 0 && xc.XObj_GetType(hEle) == xcc.XC_EDIT {
		xc.XEdit_SetCaretWidth(hEle, w)
	}
}

// Close 关闭对话框并触发关闭回调。
func (d *Dialog) Close() *Dialog {
	if d == nil || !d.open {
		return d
	}
	d.Hide()
	for _, fn := range d.onClose {
		fn()
	}
	return d
}

// IsOpen 报告对话框是否打开。
func (d *Dialog) IsOpen() bool {
	return d != nil && d.open
}

// layout 计算面板与全部子元素的位置。遮罩元素铺满 Body(内容区, 见下), 面板在遮罩内居中;
// **面板与子元素一律用"遮罩局部坐标"** —— 子元素坐标是相对父元素原点的, 遮罩本体被摆到
// Body 的 (mx,my) 后, 局部坐标就从 Body 左上角起算, 不要再叠加 mx/my。
func (d *Dialog) layout() {
	if d.sheet {
		d.layoutSheet()
		return
	}
	var wr xc.RECT
	xc.XWnd_GetRect(d.hWnd, &wr)
	ow, oh := wr.Right-wr.Left, wr.Bottom-wr.Top
	if ow <= 0 || oh <= 0 {
		return
	}
	// 遮罩铺 **Body 区**, 不铺整窗: 阴影窗口(Window_Transparent_Shadow)的窗口矩形四周含
	// 阴影带, 遮罩盖上去会把窗口自身的投影再压暗 10% —— Body 是圆角的, 四角露出的阴影带
	// 最多, 肉眼就是"遮罩四角多出黑色块"(实测: 铺满屏未最大化时四角 #11111b、Body 内
	// #ffffff)。浏览器里 overlay 盖的是 viewport(= 内容区)、窗口外框仍归系统, 语义一致。
	// 非阴影窗口 Body = 整窗, 行为不变。
	mx, my := int32(0), int32(0)
	var bodyRc xc.RECT
	xc.XWnd_GetBodyRect(d.hWnd, &bodyRc)
	if bodyRc.Right > bodyRc.Left && bodyRc.Bottom > bodyRc.Top &&
		(bodyRc.Left > 0 || bodyRc.Top > 0 || bodyRc.Right < ow || bodyRc.Bottom < oh) {
		mx, my = bodyRc.Left, bodyRc.Top
		ow, oh = bodyRc.Right-bodyRc.Left, bodyRc.Bottom-bodyRc.Top
	}
	xc.XEle_SetRect(d.overlay.Handle, &xc.RECT{Left: mx, Top: my, Right: mx + ow, Bottom: my + oh},
		false, xcc.AdjustLayout_No)
	if d.backdrop != nil && d.backdropImg != 0 {
		// 模糊层与遮罩同域(见 backdrop.go backdropPlacement)
		bx, by, bw, bh := d.backdropPlacement()
		if bw > 0 && bh > 0 {
			setEleRect(d.backdrop.Handle, bx, by, bw, bh)
		}
	}

	// 面板宽度:
	//   - Dialog: w-full max-w-sm(384), 不超过 窗口宽 - 2rem;
	//   - AlertDialog: data-[size=sm] 恒为 max-w-xs(320); default 在 <sm 断点时也是 320,
	//     宽屏才是 384(tailwind `data-[size=default]:max-w-xs data-[size=default]:sm:max-w-sm`)。
	pw := d.width
	if pw <= 0 {
		pw = dlgDefaultW
	}
	if d.alert {
		pw = alertDialogPanelWidth(d.alertSize, ow)
	} else if max := ow - 2*dlgPad; pw > max {
		pw = max
	}
	innerW := pw - 2*dlgPad

	// 文本列宽度: 对话框给右上角关闭钮让位; AlertDialog 的 media 在 default 尺寸下
	// 与文本并排(占 40 + gap-4), 在 sm 尺寸下堆在文本上方。
	// 用 mediaIcon 而不是 IsShow: showEleTree 会无条件显示整棵子树, 空图标不能占位
	mediaOn := d.media != nil && d.mediaIcon != ""
	mediaStacked := d.alert && mediaOn && d.alertSize == AlertDialogSizeSmall
	textW := innerW
	if d.alert {
		if mediaOn && !mediaStacked {
			textW -= alertDlgMediaSize + alertDlgMediaGapX
		}
	} else if d.closeBtn != nil {
		textW -= dlgCloseSize + dlgCloseOff
	}

	// 头部: 标题/描述按可用宽折行量高。两套行高/间距:
	//   - Dialog: text-base leading-none(16) / gap-2(8);
	//   - AlertDialog: text-base 默认行高(24) / gap-1.5(6)。
	titleLineH, headGap := dlgTitleH, dlgHeadGap
	if d.alert {
		titleLineH, headGap = alertDlgTitleH, alertDlgHeadGap
	}
	titleH := d.textHeight(d.title, d.titleText, textW, titleLineH)
	descH := d.textHeight(d.desc, d.descText, textW, dlgDescH)
	headerH := int32(0)
	if titleH > 0 {
		headerH = titleH
	}
	if descH > 0 {
		if headerH > 0 {
			headerH += headGap
		}
		headerH += descH
	}
	if d.alert && mediaOn {
		if mediaStacked {
			headerH += alertDlgMediaSize + alertDlgMediaGapY
		} else if headerH < alertDlgMediaSize {
			headerH = alertDlgMediaSize // media 与文本列同一网格行, 行高取两者较大值
		}
	}

	footerH := int32(0)
	if len(d.footers) > 0 {
		footerH = dlgFooterH
	}

	// 内容高度: 普通内容量子元素占位; 可滚动内容取 min(内容自然高, 上限)。
	contentH := int32(0)
	if d.scroll != nil {
		contentH = d.scrollContentHeight(innerW, headerH, footerH, oh)
	} else {
		contentH = d.contentHeight()
	}

	// 面板高度: 三组按 gap-4 排列; 页脚 -mb-4 抵消面板的下 padding, 所以贴到底边。
	// 没有页脚时要补下 padding —— 旧实现漏掉这 16px, 纯头部/纯内容的对话框底部会贴边。
	body := headerH
	if contentH > 0 {
		if body > 0 {
			body += dlgGap
		}
		body += contentH
	}
	if footerH > 0 {
		if body > 0 {
			body += dlgGap
		}
		body += footerH
	}
	ph := dlgPad + body
	if footerH == 0 {
		ph += dlgPad
	}

	px := (ow - pw) / 2
	py := (oh - ph) / 2
	d.panel = xc.RECT{Left: px, Top: py, Right: px + pw, Bottom: py + ph}
	d.footerRc = xc.RECT{}
	if footerH > 0 {
		d.footerRc = xc.RECT{
			Left:   d.panel.Left,
			Top:    d.panel.Bottom - footerH,
			Right:  d.panel.Right,
			Bottom: d.panel.Bottom,
		}
	}

	// 头部/内容: 纵向游标
	innerX := px + dlgPad
	y := py + dlgPad
	if d.closeBtn != nil {
		h := d.closeBtn.GetHeight()
		setEleRect(d.closeBtn.Handle, px+pw-dlgCloseOff-dlgCloseSize, py+dlgCloseOff, dlgCloseSize, h)
	}
	textX := innerX
	if d.alert && mediaOn {
		if mediaStacked {
			mx := px + (pw-alertDlgMediaSize)/2
			setEleRect(d.media.Handle, mx, y, alertDlgMediaSize, alertDlgMediaSize)
			y += alertDlgMediaSize + alertDlgMediaGapY
		} else {
			setEleRect(d.media.Handle, innerX, y, alertDlgMediaSize, alertDlgMediaSize)
			textX = innerX + alertDlgMediaSize + alertDlgMediaGapX
		}
	}
	if titleH > 0 {
		// 元素高度比行盒大 dlgTextOver: 文字在元素内垂直居中, 元素上下各外扩一点
		// 才能既放得下字形(含折行时的行距)又不占布局高度
		setEleRect(d.title.Handle, textX, y-dlgTextOver, textW, titleH+2*dlgTextOver)
		d.title.SetText(d.titleText)
		y += titleH
	}
	if descH > 0 {
		if titleH > 0 {
			y += headGap
		}
		setEleRect(d.desc.Handle, textX, y-dlgTextOver, textW, descH+2*dlgTextOver)
		d.desc.SetText(d.descText)
		y += descH
	}
	if contentH > 0 {
		cy := py + dlgPad + headerH
		if headerH > 0 {
			cy += dlgGap
		}
		setEleRect(d.content.Handle, innerX, cy, innerW, contentH)
		if d.scroll != nil {
			// ScrollArea 是内容容器里的子元素, 用容器局部坐标
			setEleRect(d.scroll.Handle, 0, 0, innerW, contentH)
			xc.XEle_AdjustLayout(d.scroll.Handle)
		}
	}

	// 页脚按钮:
	//   - Dialog / AlertDialog(default): 右对齐一行, gap-2;
	//   - AlertDialog(sm): grid-cols-2, 两个按钮等宽各占一半。
	if footerH > 0 {
		by := d.footerRc.Top + dlgPad
		if d.alert && d.alertSize == AlertDialogSizeSmall {
			cellW := (innerW - dlgFooterGap) / 2
			for i, b := range d.footers {
				if i >= 2 {
					break // AlertDialogFooter 只有 Cancel/Action 两个
				}
				x := innerX + int32(i)*(cellW+dlgFooterGap)
				setEleRect(b.Handle, x, by, cellW, b.GetHeight())
			}
		} else {
			total := int32(0)
			for i, b := range d.footers {
				if i > 0 {
					total += dlgFooterGap
				}
				total += b.GetWidth()
			}
			x := d.footerRc.Right - dlgPad - total
			for _, b := range d.footers {
				setEleRect(b.Handle, x, by, b.GetWidth(), b.GetHeight())
				x += b.GetWidth() + dlgFooterGap
			}
		}
	}
}

// contentHeight 量内容区高度: 取容器内可见子元素占位并集的底边。
// 用户用布局容器时先重排一次(AdjustLayout 对普通元素是空操作)。
func (d *Dialog) contentHeight() int32 {
	if d.content == nil {
		return 0
	}
	xc.XEle_AdjustLayout(d.content.Handle)
	return childrenBottom(d.content.Handle)
}

// childrenBottom 返回某容器内可见子元素占位并集的底边(局部坐标)。
func childrenBottom(hParent int) int32 {
	var maxB int32
	n := xc.XEle_GetChildCount(hParent)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(hParent, i)
		if h == 0 || !xc.XWidget_IsShow(h) {
			continue
		}
		var rc xc.RECT
		xc.XEle_GetRect(h, &rc)
		if rc.Bottom > maxB {
			maxB = rc.Bottom
		}
	}
	return maxB
}

// textHeight 量标题/描述在当前宽度下的折行高度(逻辑像素)。
//
// 量宽要收 dlgTextInsetX: 原生 Button 载体的文字排版区比元素窄一圈, 按元素宽量会
// 少算一行(同 alert.go 的踩坑)。单行高度直接用行盒尺寸; 多行时取整到整行并补 4px,
// 否则最后一行会被裁掉并出现省略号。
func (d *Dialog) textHeight(l *Label, text string, width, lineH int32) int32 {
	if l == nil || text == "" || width <= 0 || lineH <= 0 {
		return 0
	}
	measureW := width - dlgTextInsetX
	if measureW < lineH {
		measureW = width
	}
	h := l.WrappedHeight(measureW)
	// GetTextShowRect 的单行返回值比行盒略高(16px 字约 19), 容差内仍按单行算
	if h <= lineH+dlgTextPadY {
		return lineH
	}
	return (h+lineH-1)/lineH*lineH + dlgTextPadY
}

// scrollContentHeight 量可滚动内容的自然高度, 夹到 70vh / 面板可用高度以内。
// 先给 ScrollArea 一个临时高度并重排, 才能拿到内容区子项的真实占位。
func (d *Dialog) scrollContentHeight(innerW, headerH, footerH, bodyH int32) int32 {
	if d.scroll == nil {
		return 0
	}
	maxH := d.scrollMax
	if maxH <= 0 {
		maxH = bodyH * 7 / 10 // shadcn 示例的 max-h-[70vh]
	}
	avail := bodyH - 2*dlgPad - headerH - footerH
	if headerH > 0 {
		avail -= dlgGap
	}
	if footerH > 0 {
		avail -= dlgGap
	}
	if avail < 80 {
		avail = 80
	}
	if maxH > avail {
		maxH = avail
	}
	setEleRect(d.scroll.Handle, 0, 0, innerW, maxH)
	xc.XEle_AdjustLayout(d.scroll.Handle)
	h := childrenBottom(d.scroll.Content().Handle)
	if h > maxH {
		h = maxH
	}
	if h < 1 {
		h = 1
	}
	return h
}

// onPaint 绘制遮罩、面板、页脚带与描边。
func (d *Dialog) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	if d.sheet {
		d.paintSheet(p)
		return 0
	}
	t := xui.Current()

	// 遮罩: bg-black/10(shadcn 的遮罩是固定黑色 10%, 不随主题变化)
	p.FillRound(&p.Rc, xui.RGBA(0, 0, 0, 26), 0)

	panel := d.panel // 面板/页脚都存遮罩局部坐标, 与绘制坐标系一致
	if panel.Right <= panel.Left || panel.Bottom <= panel.Top {
		return 0
	}
	radius := t.RadiusXl() // rounded-xl

	// 面板: bg-popover
	p.FillRound(&panel, t.Popover, radius)

	// 页脚带: bg-muted/50 + border-t, 只有下面两个角随面板圆角。
	// 必须**一次**画出"上平下圆"的整块: 先画圆角矩形再用方角块补上边缘会让半透明色
	// 在交叠区被合成两次(实测顶部 radius 高的一条比页脚其余部分深一档, 呈 #f7f7f7 条带)。
	if d.footerRc.Bottom > d.footerRc.Top {
		fr := d.footerRc
		band := t.Muted.ScaleAlpha(0.5) // bg-muted/50: 在令牌自身 alpha 上乘, 不是覆盖
		p.FillRoundCorners(&fr, band, xui.Corners{BottomRight: radius, BottomLeft: radius})
		p.Line(fr.Left, fr.Top, fr.Right, fr.Top, t.Border, 1)
	}

	// ring-1 ring-foreground/10: 贴面板边缘的描边, 画在最后(压住底色边缘)。
	// 注意 tailwind 的 ring 是 box-shadow, 画在盒子**外侧**(压在遮罩灰底上才形成可见的
	// 深色描边线); xcgui 的圆角矩形描边落在路径内侧, 直接描 d.panel 会把 10% 黑压在
	// 面板白底边上(230), 与遮罩灰(229)几乎同值而不可见 —— 实测后外扩 1px 对齐浏览器
	ringRc := panel
	ringRc.Left--
	ringRc.Top--
	ringRc.Right++
	ringRc.Bottom++
	p.StrokeRound(&ringRc, t.Foreground.WithAlpha(26), radius+1, 1)
	return 0
}

// onMediaPaint 画 AlertDialogMedia: size-10 rounded-md bg-muted + size-6 图标;
// 破坏性变体是 bg-destructive/10 text-destructive(暗色 /20)。
func (d *Dialog) onMediaPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil || d.mediaIcon == "" {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	bg, fg := t.Muted, t.MutedForeground
	if d.mediaTone == AlertDialogMediaDestructive {
		frac := 0.1
		if t.IsDark() {
			frac = 0.2
		}
		bg, fg = t.Destructive.ScaleAlpha(frac), t.Destructive
	}
	p.FillRound(&p.Rc, bg, t.RadiusMd())
	ir := p.Rc
	inset := (alertDlgMediaSize - alertDlgMediaIcon) / 2
	ir.Left += inset
	ir.Top += inset
	ir.Right -= inset
	ir.Bottom -= inset
	p.Icon(d.mediaIcon, &ir, fg, 2)
	return 0
}

// onOverlayDown 点击遮罩(面板之外)关闭对话框。
func (d *Dialog) onOverlayDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !d.open || !d.isTopModal() || pPt == nil {
		return 0
	}
	// 对齐 radix AlertDialog: onInteractOutside preventDefault, 点遮罩不关闭
	if !d.dismissOutside {
		return 0
	}
	// pPt 与 d.panel 都是遮罩局部坐标(遮罩原点 = Body 左上角), 直接比
	if pPt.X < d.panel.Left || pPt.X >= d.panel.Right || pPt.Y < d.panel.Top || pPt.Y >= d.panel.Bottom {
		d.Close()
	}
	return 0
}

// onWindowKeyDown 处理 ESC 与 Tab: 键盘事件统一走窗口(实测: 元素收不到键盘事件,
// 编辑框聚焦时窗口也照常收到), 因此在这里做模态键盘拦截。
func (d *Dialog) onWindowKeyDown(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
	if !d.open || !d.isTopModal() {
		return 0
	}
	switch wParam {
	case vkDlgEscape:
		*pbHandled = true
		d.Close()
	case vkDlgTab:
		*pbHandled = true
		d.moveFocus(wapi.GetAsyncKeyState(vkDlgShift) < 0) // 最高位 = 当前按下
	}
	return 0
}

// moveFocus 在对话框内部循环移动焦点(Tab / Shift+Tab)。
func (d *Dialog) moveFocus(back bool) {
	fs := d.focusables()
	if len(fs) == 0 {
		return
	}
	idx := d.currentIndex(fs)
	if idx < 0 {
		idx = d.focusIdx
	}
	// 先标记"焦点来自键盘", 否则搬完焦点才被 AttachWindow 的 Tab 钩子标记,
	// 组件已经重绘完, 焦点框不会出现
	xui.NoteKeyboard()
	// 逐个尝试: 焦点可能落在复合控件的内部元素上, SetFocusEle 未必一次成功
	old := xc.XWnd_GetFocusEle(d.hWnd)
	for i := 0; i < len(fs); i++ {
		var next int
		if back {
			next = (idx - 1 - i + 2*len(fs)) % len(fs)
		} else {
			next = (idx + 1 + i) % len(fs)
		}
		if xc.XWnd_SetFocusEle(d.hWnd, fs[next]) {
			d.focusIdx = next
			d.lastFocus = fs[next]
			// 焦点状态变化不一定触发重绘, 手动刷新新旧元素(否则按钮上的焦点框不出现)
			if old != 0 && old != fs[next] {
				xc.XEle_Redraw(old, false)
			}
			xc.XEle_Redraw(fs[next], false)
			return
		}
	}
}

// currentIndex 反查当前焦点元素在焦点序列中的下标。焦点可能落在控件的内部元素上
// (复合控件的子元素), 因此沿父链向上找; 找不到返回 -1。
func (d *Dialog) currentIndex(fs []int) int {
	cur := xc.XWnd_GetFocusEle(d.hWnd)
	for h := cur; h != 0; h = xc.XWidget_GetParentEle(h) {
		for i, f := range fs {
			if f == h {
				return i
			}
		}
		if h == d.overlay.Handle {
			break
		}
	}
	return -1
}

// focusables 收集对话框内的焦点顺序: 内容子元素 → 页脚按钮 → 关闭按钮
// (与 radix 打开时自动聚焦第一个可聚焦元素一致)。
func (d *Dialog) focusables() []int {
	var list []int
	if d.content != nil {
		collectFocusable(d.content.Handle, &list)
	}
	for _, b := range d.footers {
		if b.IsShow() {
			list = append(list, b.Handle)
		}
	}
	if d.closeBtn != nil && d.closeBtn.IsShow() {
		list = append(list, d.closeBtn.Handle)
	}
	return list
}

// defaultFocus 返回打开时应聚焦的元素:
//   - AlertDialog 强制聚焦 Cancel(radix 的 onOpenAutoFocus → cancelRef.focus), 没有
//     Cancel 时才退到第一个可聚焦控件;
//   - Dialog 聚焦第一个可聚焦控件。
func (d *Dialog) defaultFocus(fs []int) int {
	if len(fs) == 0 {
		return 0
	}
	if d.initialFocus != 0 {
		for _, h := range fs {
			if h == d.initialFocus {
				return h
			}
		}
	}
	return fs[0]
}

// restoreModalFocus 关闭栈顶后, 把焦点还给下层对话框最近一次聚焦的控件
// (没有记录/已不可见时退回它的默认焦点)。没有下层时什么都不做。
func restoreModalFocus(hWnd int) {
	top := topModal(hWnd)
	if top == nil {
		return
	}
	fs := top.focusables()
	if len(fs) == 0 {
		return
	}
	h := top.lastFocus
	if h == 0 {
		h = top.defaultFocus(fs)
	}
	for _, f := range fs {
		if f == h {
			top.setCaretWidthOf(h, 1)
			xc.XWnd_SetFocusEle(hWnd, h)
			xc.XEle_Redraw(h, false)
			return
		}
	}
	h = top.defaultFocus(fs)
	top.lastFocus = h
	xc.XWnd_SetFocusEle(hWnd, h)
	xc.XEle_Redraw(h, false)
}

// collectFocusable 深度优先收集可聚焦的交互控件(只认按钮/编辑框这类原生控件,
// 容器元素(普通元素/布局元素)不进 Tab 序列)。
func collectFocusable(hParent int, out *[]int) {
	n := xc.XEle_GetChildCount(hParent)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(hParent, i)
		if h == 0 || !xc.XWidget_IsShow(h) {
			continue
		}
		if isFocusControl(h) {
			*out = append(*out, h)
		}
		collectFocusable(h, out)
	}
}

// isFocusControl 判断元素是否是值得进 Tab 序列的交互控件。
func isFocusControl(h int) bool {
	switch xc.XObj_GetType(h) {
	case xcc.XC_BUTTON, xcc.XC_EDIT, xcc.XC_COMBOBOX, xcc.XC_SLIDERBAR,
		xcc.XC_EDITOR, xcc.XC_RICHEDIT, xcc.XC_LISTBOX, xcc.XC_TABBAR, xcc.XC_SCROLLBAR:
		return xc.XEle_IsEnableFocus(h) && xc.XEle_IsSwitchFocus(h)
	}
	return false
}

// setEleRect 设置元素矩形(不触发重绘, 由调用方统一重绘)。
func setEleRect(hEle int, x, y, cx, cy int32) {
	xc.XEle_SetRect(hEle, &xc.RECT{Left: x, Top: y, Right: x + cx, Bottom: y + cy},
		false, xcc.AdjustLayout_No)
}

// showEleTree 递归显隐元素及其子树。xcgui 的 Show 只改元素自身状态,
// 这里显式下钻, 保证对话框隐藏时其内部控件不会残留在窗口上。
func showEleTree(hEle int, show bool) {
	if hEle == 0 {
		return
	}
	xc.XWidget_Show(hEle, show)
	n := xc.XEle_GetChildCount(hEle)
	for i := int32(0); i < n; i++ {
		h := xc.XEle_GetChildByIndex(hEle, i)
		// 滚动条不参与整树显隐: 原生 Edit 一显示就会**补建**自己的滚动条子元素,
		// 第二次整树 Show(true) 会把它们一起点亮 —— 而单行输入框的那两个滚动条是
		// 被刻意按住的(默认矩形 (0,0,100,12)/(0,0,12,100), 点亮后画在控件左上角,
		// 表现就是"第一次打开正常, 第二次打开输入框左上角多出一条灰条")。
		// 滚动条的显隐本来就由控件/皮肤自己管(见 scrollbar.go 与第八节 98)。
		if h != 0 && xc.XObj_GetType(h) == xcc.XC_SCROLLBAR {
			continue
		}
		showEleTree(h, show)
	}
}
