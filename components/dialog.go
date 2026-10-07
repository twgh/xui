package ui

import (
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
)

// 对话框用到的虚拟键码。
const (
	vkDlgTab    uintptr = 9
	vkDlgEscape uintptr = 27
	vkDlgShift  int32   = 0x10
)

// dlgFadeKey 是对话框淡入淡出用的 presence 键(见 presence.go, 与 hover 键空间隔离)。
const dlgFadeKey int32 = 0

// Dialog 模态对话框: 一个铺满整窗的遮罩元素 + 居中面板, 面板内容全部自绘。
//
// 与 shadcn dialog.tsx(radix-nova) 对齐:
//   - 遮罩: fixed inset-0 bg-black/10, 覆盖整个窗口并拦截下层全部鼠标事件;
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

	width    int32
	open     bool
	focusIdx int // 焦点序列的当前位置(Tab 循环的兜底基准)

	panel    xc.RECT // 面板矩形(遮罩局部坐标 = 窗口坐标)
	footerRc xc.RECT // 页脚带矩形

	onClose []func()
}

// DialogOption 对话框配置项, 构造时传入。
type DialogOption func(*Dialog)

// WithDialogWidth 覆盖面板宽度(默认 384 = sm:max-w-sm)。
func WithDialogWidth(w int32) DialogOption { return func(d *Dialog) { d.width = w } }

// WithDialogCloseButton 是否显示右上角关闭按钮(默认显示)。
func WithDialogCloseButton(b bool) DialogOption { return func(d *Dialog) { d.showClose = b } }

// NewDialog 创建对话框(默认隐藏)。hWnd 为宿主窗口句柄, 对话框覆盖整个窗口。
func NewDialog(hWnd int, opts ...DialogOption) *Dialog {
	win := app.NewWindowByHandle(hWnd)
	if win == nil {
		return nil
	}
	d := &Dialog{hWnd: hWnd, win: win, width: dlgDefaultW, showClose: true}
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

	// 背景模糊层: 抓窗快照 + 盒式模糊, 在遮罩之下、其余内容之上(见 backdrop.go)
	d.backdrop = newBackdrop(hWnd)

	// shadcn: text-base font-medium(500) —— 16px + 中等字重(不是 Bold 的 17px)
	d.title = NewLabel(0, 0, 0, dlgTitleH, "", d.overlay.Handle).SetMedium()
	d.title.SetFontSize(16)
	d.desc = NewLabel(0, 0, 0, dlgDescH, "", d.overlay.Handle).SetMuted()

	d.content = widget.NewElement(0, 0, 0, 0, d.overlay.Handle)
	d.content.EnableBkTransparent(true)
	d.content.EnableSwitchFocus(false) // 容器本身不进 Tab 序列
	// 原生焦点框与边框都要灭: 点击容器会聚焦到它, 框架会给它画蓝框/深色边框
	d.content.EnableFocus(false)
	d.content.EnableDrawBorder(false)

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

	showEleTree(d.overlay.Handle, false)
	return d
}

// SetTitle 设置标题(text-base), 传空串隐藏标题。
func (d *Dialog) SetTitle(s string) *Dialog {
	if d == nil {
		return d
	}
	d.titleText = s
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
	return d.content.Handle
}

// ContentWidth 内容容器宽度(面板宽 - 左右 padding), 创建内容元素时作参考宽度。
func (d *Dialog) ContentWidth() int32 {
	w := d.width
	if w <= 0 {
		w = dlgDefaultW
	}
	return w - 2*dlgPad
}

// AddFooter 在页脚追加一个按钮(右对齐排列, 高度用默认 h-8, 宽度按文字量出)。
// onClick 为 nil 时只画按钮不绑事件。
func (d *Dialog) AddFooter(text string, variant ButtonVariant, onClick func()) *Dialog {
	if d == nil {
		return d
	}
	w := int32(0)
	if f := xui.FontHandle(xui.FontSpec{
		Family: xui.Current().FontDefault.Family,
		Size:   buttonSpecs[ButtonSizeDefault].font,
	}); f != 0 {
		var sz xc.SIZE
		app.GetTextShowRect(text, -1, f, 0, 0, &sz)
		// 该 API 的返回值约为实际绘制宽度的一半(144 DPI 实测: "保存" 返回 14.7 逻辑,
		// 实际绘制 29.3), 直接拿它算宽度会导致按钮挤到文字上, 这里 ×2 校正
		w = sz.CX * 2
	}
	if w <= 0 {
		w = 40
	}
	w += 2 * dlgBtnPadX // px-2.5
	btn := NewButton(0, 0, w, 0, text, d.overlay.Handle, WithVariant(variant))
	if btn == nil {
		return d
	}
	if onClick != nil {
		btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			onClick()
			return 0
		})
	}
	d.footers = append(d.footers, btn)
	showEleTree(btn.Handle, d.open)
	d.relayoutIfOpen()
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
	d.open = true
	// 先抓背景快照(此刻遮罩/模糊层都还藏着, 抓到的是无对话框的画面), 再显示再布局:
	// 内容高度量的是"可见子元素"的占位, 隐藏状态下量出来是 0
	d.refreshBackdrop()
	showEleTree(d.overlay.Handle, true)
	d.layout()
	// z 序: 模糊层先抬顶, 遮罩再抬顶压住它(两者都在其余内容之上)
	if d.backdrop != nil {
		d.backdrop.SetZOrderEx(0, xcc.Zorder_Top)
	}
	d.overlay.SetZOrderEx(0, xcc.Zorder_Top)
	// 进场: 整组(模糊层 + 遮罩 + 面板 + 全部子元素)从全透明淡入 —— 对齐 shadcn dialog 的
	// data-open:animate-in fade-in-0 zoom-in-95(时长 100ms); 缩放在 xcgui 里做不到
	// (元素级 ani 对自绘元素无效, 见 DESIGN 第八节 47), 这里只做淡入。
	// 注意必须整棵子树逐个设 alpha: 元素透明度不级联, 只设 overlay 的话标题/输入框
	// 等子元素会全不透明压在还没模糊的背景上(实测踩坑)
	xui.SetSubtreeAlpha(0, d.overlay.Handle, d.backdropHandle())
	d.overlay.Redraw(false)
	xc.XWnd_Redraw(d.hWnd, true)
	d.focusIdx = 0
	fs := d.focusables()
	if len(fs) > 0 {
		// 插入符宽度只在 caret 创建(获得焦点)时生效 —— 对已显示的 caret 改宽度无效,
		// 清焦点再聚焦才生效(实测)。所以聚焦前先把宽度置 0, 淡入期间插入符隐身;
		// 到位后恢复宽度并重铸焦点让插入符回来(焦点环则始终随整组淡入, 不闪)
		d.setCaretWidthOf(fs[0], 0)
		xc.XWnd_SetFocusEle(d.hWnd, fs[0])
	}
	xui.Presence(d.hWnd, dlgFadeKey, true, xui.PopupDuration, d.applyFade, func(float64) {
		if len(fs) > 0 {
			d.setCaretWidthOf(fs[0], 1)
			if xc.XWnd_GetFocusEle(d.hWnd) == fs[0] {
				xc.XWnd_SetFocusEle(d.hWnd, 0)
				xc.XWnd_SetFocusEle(d.hWnd, fs[0])
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
	d.open = false
	// 清焦点: 插入符随焦点消失, 淡出的 100ms 里不会闪现全亮度的插入符,
	// 也避免键盘输入继续打进正在消失的输入框
	xc.XWnd_SetFocusEle(d.hWnd, 0)
	xui.Presence(d.hWnd, dlgFadeKey, false, xui.PopupDuration, d.applyFade, func(v float64) {
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
// 实测有效; 但**不级联到子元素**, 所以用 SetSubtreeAlpha 对整棵子树逐个设置 ——
// 遮罩、面板、模糊层与标题/输入框等内容一起淡; 焦点环也因此从第一帧就随整组淡入,
// 不会在最后闪一下, 对齐 shadcn)。
// 输入框插入符竖条不随元素 alpha 淡(原生绘制, 实测) —— 由 setCaretWidth 在进出场
// 期间置 0 隐身, 不在这里处理。
func (d *Dialog) applyFade(v float64) {
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

// layout 计算面板与全部子元素的位置。元素坐标原点 = 窗口左上角(含标题栏),
// 遮罩因此铺满整个窗口; 面板按窗口居中。
func (d *Dialog) layout() {
	var wr xc.RECT
	xc.XWnd_GetRect(d.hWnd, &wr)
	ow, oh := wr.Right-wr.Left, wr.Bottom-wr.Top
	if ow <= 0 || oh <= 0 {
		return
	}
	xc.XEle_SetRect(d.overlay.Handle, &xc.RECT{Right: ow, Bottom: oh}, false, xcc.AdjustLayout_No)
	if d.backdrop != nil && d.backdropImg != 0 {
		// 模糊层跟 Body 区走, 不铺整窗(元素坐标系含阴影带, 见 backdrop.go backdropPlacement)
		bx, by, bw, bh := d.backdropPlacement()
		if bw > 0 && bh > 0 {
			setEleRect(d.backdrop.Handle, bx, by, bw, bh)
		}
	}

	// 面板宽度: w-full max-w-sm, 同时不超过 窗口宽 - 2rem
	pw := d.width
	if pw <= 0 {
		pw = dlgDefaultW
	}
	if max := ow - 2*dlgPad; pw > max {
		pw = max
	}

	// 头部高度: 标题(text-base) + gap-2 + 描述(text-sm)
	headerH := int32(0)
	if d.titleText != "" {
		headerH = dlgTitleH
	}
	if d.descText != "" {
		if headerH > 0 {
			headerH += dlgHeadGap
		}
		headerH += dlgDescH
	}
	contentH := d.contentHeight()

	// 面板高度: 三组按 gap-4 排列; 页脚 -mb-4 抵消面板的下 padding, 所以贴到底边
	body := headerH
	if contentH > 0 {
		if body > 0 {
			body += dlgGap
		}
		body += contentH
	}
	if len(d.footers) > 0 {
		if body > 0 {
			body += dlgGap
		}
		// 页脚 -mb-4 抵消面板的下 padding, 上下都归零后它的高度全量计入
		body += dlgFooterH
	}
	ph := dlgPad + body

	px := (ow - pw) / 2
	py := (oh - ph) / 2
	d.panel = xc.RECT{Left: px, Top: py, Right: px + pw, Bottom: py + ph}
	d.footerRc = xc.RECT{}
	if len(d.footers) > 0 {
		d.footerRc = xc.RECT{
			Left:   d.panel.Left,
			Top:    d.panel.Bottom - dlgFooterH,
			Right:  d.panel.Right,
			Bottom: d.panel.Bottom,
		}
	}

	// 标题/描述/内容: 纵向游标, 组间 gap-4
	innerX := px + dlgPad
	innerW := pw - 2*dlgPad
	textW := innerW - dlgCloseSize - dlgCloseOff // 给右上角关闭按钮让位
	y := py + dlgPad
	if d.closeBtn != nil {
		h := d.closeBtn.GetHeight()
		setEleRect(d.closeBtn.Handle, px+pw-dlgCloseOff-dlgCloseSize, py+dlgCloseOff, dlgCloseSize, h)
	}
	if d.titleText != "" {
		// 元素高度比行盒大 dlgTextOver: 文字在元素内垂直居中, 行盒按 leading-none 计,
		// 元素在行盒上下各外扩一点才能既放得下字形又不占布局高度
		setEleRect(d.title.Handle, innerX, y-dlgTextOver, textW, dlgTitleH+2*dlgTextOver)
		d.title.SetText(d.titleText)
		y += dlgTitleH
	}
	if d.descText != "" {
		if d.titleText != "" {
			y += dlgHeadGap
		}
		setEleRect(d.desc.Handle, innerX, y-dlgTextOver, textW, dlgDescH+2*dlgTextOver)
		d.desc.SetText(d.descText)
		y += dlgDescH
	}
	if contentH > 0 {
		if y > py+dlgPad {
			y += dlgGap
		}
		setEleRect(d.content.Handle, innerX, y, innerW, contentH)
	}

	// 页脚按钮: 右对齐, 间距 gap-2
	if len(d.footers) > 0 {
		total := int32(0)
		for i, b := range d.footers {
			if i > 0 {
				total += dlgFooterGap
			}
			total += b.GetWidth()
		}
		x := d.footerRc.Right - dlgPad - total
		by := d.footerRc.Top + dlgPad
		for _, b := range d.footers {
			setEleRect(b.Handle, x, by, b.GetWidth(), b.GetHeight())
			x += b.GetWidth() + dlgFooterGap
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
	var maxB int32
	n := d.content.GetChildCount()
	for i := int32(0); i < n; i++ {
		h := d.content.GetChildByIndex(i)
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

// onPaint 绘制遮罩、面板、页脚带与描边。
func (d *Dialog) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// 遮罩: bg-black/10(shadcn 的遮罩是固定黑色 10%, 不随主题变化)
	p.FillRound(&p.Rc, xui.RGBA(0, 0, 0, 26), 0)
	if d.panel.Right <= d.panel.Left || d.panel.Bottom <= d.panel.Top {
		return 0
	}
	radius := t.RadiusXl() // rounded-xl

	// 面板: bg-popover
	p.FillRound(&d.panel, t.Popover, radius)

	// 页脚带: bg-muted/50 + border-t, 只有下面两个角随面板圆角
	if d.footerRc.Bottom > d.footerRc.Top {
		fr := d.footerRc
		band := t.Muted.WithAlpha(128)
		p.FillRound(&fr, band, radius)
		// 上两个角补平(-mx-4 的页脚带左右贴面板边, 上方与面板内容同面)
		sq := xc.RECT{Left: fr.Left, Top: fr.Top, Right: fr.Right, Bottom: fr.Top + radius}
		p.FillRound(&sq, band, 0)
		p.Line(fr.Left, fr.Top, fr.Right, fr.Top, t.Border, 1)
	}

	// ring-1 ring-foreground/10: 贴面板边缘的描边, 画在最后(压住底色边缘)。
	// 注意 tailwind 的 ring 是 box-shadow, 画在盒子**外侧**(压在遮罩灰底上才形成可见的
	// 深色描边线); xcgui 的圆角矩形描边落在路径内侧, 直接描 d.panel 会把 10% 黑压在
	// 面板白底边上(230), 与遮罩灰(229)几乎同值而不可见 —— 实测后外扩 1px 对齐浏览器
	ringRc := d.panel
	ringRc.Left--
	ringRc.Top--
	ringRc.Right++
	ringRc.Bottom++
	p.StrokeRound(&ringRc, t.Foreground.WithAlpha(26), radius+1, 1)
	return 0
}

// onOverlayDown 点击遮罩(面板之外)关闭对话框。
func (d *Dialog) onOverlayDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if !d.open || pPt == nil {
		return 0
	}
	if pPt.X < d.panel.Left || pPt.X >= d.panel.Right || pPt.Y < d.panel.Top || pPt.Y >= d.panel.Bottom {
		d.Close()
	}
	return 0
}

// onWindowKeyDown 处理 ESC 与 Tab: 键盘事件统一走窗口(实测: 元素收不到键盘事件,
// 编辑框聚焦时窗口也照常收到), 因此在这里做模态键盘拦截。
func (d *Dialog) onWindowKeyDown(hWindow int, wParam, lParam uintptr, pbHandled *bool) int {
	if !d.open {
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
		showEleTree(xc.XEle_GetChildByIndex(hEle, i), show)
	}
}
