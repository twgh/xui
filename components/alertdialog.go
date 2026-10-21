package ui

import "github.com/twgh/xcgui/widget"

// AlertDialog 警示对话框: 打断用户并要求明确选择(取消 / 继续)。
//
// 与 shadcn alert-dialog.tsx(radix-nova) 对齐:
//   - 没有右上角关闭按钮, 点遮罩不关闭(radix 的 onInteractOutside preventDefault),
//     ESC 关闭; 打开时焦点强制落在 Cancel(radix 的 onOpenAutoFocus);
//   - default: 头部左对齐(gap-1.5)、页脚右对齐一行; sm: 头部居中、页脚等宽双列;
//   - media = size-10 rounded-md bg-muted + size-6 图标, destructive 变体用
//     bg-destructive/10(text-destructive); sm 下 media 堆在标题上方(mb-2);
//   - Action 用调用方给的变体(primary / destructive), Cancel 默认 outline, 两者点击后都关闭;
//   - 与 Dialog 共用遮罩、backdrop-blur、焦点陷阱与 100ms 淡入淡出。
//
// 与上游的差异: zoom-in-95 不做(元素级缩放对自绘元素无效, 见 DESIGN 第八节 47);
// text-balance / text-pretty 不做(自绘文字没有该排版能力)。
type AlertDialog struct {
	d      *Dialog
	cancel *Button
	action *Button
}

// AlertDialogOption 警示对话框配置项。
type AlertDialogOption func(*AlertDialog)

// WithAlertDialogSize 设置尺寸(默认 AlertDialogSizeDefault)。
func WithAlertDialogSize(s AlertDialogSize) AlertDialogOption {
	return func(a *AlertDialog) { a.SetSize(s) }
}

// WithAlertDialogTitle 设置标题(等价于构造后调 SetTitle)。
func WithAlertDialogTitle(s string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetTitle(s) }
}

// WithAlertDialogDescription 设置描述(等价于构造后调 SetDescription)。
func WithAlertDialogDescription(s string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetDescription(s) }
}

// WithAlertDialogMedia 设置头部媒体块图标(path 数据或整段 SVG; 空串 = 无媒体块)。
func WithAlertDialogMedia(icon string) AlertDialogOption {
	return func(a *AlertDialog) { a.SetMedia(icon) }
}

// WithAlertDialogMediaTone 设置媒体块配色(默认 AlertDialogMediaMuted)。
func WithAlertDialogMediaTone(tone AlertDialogMediaTone) AlertDialogOption {
	return func(a *AlertDialog) { a.SetMediaTone(tone) }
}

// NewAlertDialog 创建警示对话框(默认隐藏)。hWnd 为宿主窗口句柄。
func NewAlertDialog(hWnd int, opts ...AlertDialogOption) *AlertDialog {
	d := NewDialog(hWnd, WithDialogCloseButton(false))
	if d == nil {
		return nil
	}
	d.alert = true
	d.dismissOutside = false
	a := &AlertDialog{d: d}
	for _, o := range opts {
		o(a)
	}
	a.applyAlignment()
	return a
}

// SetTitle 设置标题; 传空串隐藏标题。
func (a *AlertDialog) SetTitle(s string) *AlertDialog {
	if a != nil {
		a.d.SetTitle(s)
	}
	return a
}

// SetDescription 设置描述(text-sm, muted; 按宽度自动折行); 传空串隐藏描述。
func (a *AlertDialog) SetDescription(s string) *AlertDialog {
	if a != nil {
		a.d.SetDescription(s)
	}
	return a
}

// SetSize 切换尺寸(同时切换头部对齐与页脚排布)。
func (a *AlertDialog) SetSize(s AlertDialogSize) *AlertDialog {
	if a == nil {
		return a
	}
	a.d.alertSize = s
	a.applyAlignment()
	a.d.relayoutIfOpen()
	return a
}

// Size 返回当前尺寸。
func (a *AlertDialog) Size() AlertDialogSize {
	if a == nil {
		return AlertDialogSizeDefault
	}
	return a.d.alertSize
}

// SetMedia 设置头部媒体块图标(path 数据或整段 SVG; 空串 = 隐藏)。
func (a *AlertDialog) SetMedia(icon string) *AlertDialog {
	if a == nil {
		return a
	}
	d := a.d
	if d.media == nil {
		e := widget.NewElement(0, 0, alertDlgMediaSize, alertDlgMediaSize, d.overlay.Handle)
		if e == nil {
			return a
		}
		e.EnableBkTransparent(true)
		e.EnableDrawBorder(false)
		e.EnableFocus(false)
		e.EnableSwitchFocus(false)
		e.AddEvent_Paint(d.onMediaPaint)
		d.media = e
	}
	d.mediaIcon = icon
	// 媒体块可能在构造函数末尾的整树隐藏之后创建, 显隐要跟对话框状态同步
	d.media.Show(icon != "")
	showEleTree(d.media.Handle, d.open && icon != "")
	d.media.Redraw(false)
	d.relayoutIfOpen()
	return a
}

// Media 返回媒体块图标(空串 = 没有)。
func (a *AlertDialog) Media() string {
	if a == nil {
		return ""
	}
	return a.d.mediaIcon
}

// SetMediaTone 设置媒体块配色。
func (a *AlertDialog) SetMediaTone(tone AlertDialogMediaTone) *AlertDialog {
	if a == nil {
		return a
	}
	a.d.mediaTone = tone
	if a.d.media != nil {
		a.d.media.Redraw(false)
	}
	return a
}

// AddCancel 追加取消按钮(outline, 默认聚焦)。点击后先执行 onClick(可为 nil)再关闭;
// 对应 AlertDialogCancel。可以在 AddAction 前后调用, 打开时焦点总是 Cancel。
func (a *AlertDialog) AddCancel(text string, onClick func()) *AlertDialog {
	if a == nil {
		return a
	}
	btn := a.d.addFooterButton(text, ButtonOutline)
	if btn == nil {
		return a
	}
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if onClick != nil {
			onClick()
		}
		a.Close()
		return 0
	})
	a.cancel = btn
	a.d.initialFocus = btn.Handle
	a.d.relayoutIfOpen()
	return a
}

// AddAction 追加确认按钮(变体由调用方给, 通常是 ButtonPrimary / ButtonDestructive)。
// 点击后先执行 onClick(可为 nil)再关闭; 对应 AlertDialogAction。
func (a *AlertDialog) AddAction(text string, variant ButtonVariant, onClick func()) *AlertDialog {
	if a == nil {
		return a
	}
	btn := a.d.addFooterButton(text, variant)
	if btn == nil {
		return a
	}
	btn.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		if onClick != nil {
			onClick()
		}
		a.Close()
		return 0
	})
	a.action = btn
	if a.d.initialFocus == 0 {
		a.d.initialFocus = btn.Handle
	}
	a.d.relayoutIfOpen()
	return a
}

// CancelButton 返回 Cancel 按钮(可在 AddCancel 后改变体/图标); 未添加时 nil。
func (a *AlertDialog) CancelButton() *Button {
	if a == nil {
		return nil
	}
	return a.cancel
}

// ActionButton 返回 Action 按钮(可在 AddAction 后改变体/图标); 未添加时 nil。
func (a *AlertDialog) ActionButton() *Button {
	if a == nil {
		return nil
	}
	return a.action
}

// BindTrigger 把控件绑成"点击即打开"(= AlertDialogTrigger)。
func (a *AlertDialog) BindTrigger(t clickBinder) *AlertDialog {
	if a == nil || t == nil {
		return a
	}
	t.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		a.Show()
		return 0
	})
	return a
}

// Content 返回内容容器句柄(可放额外内容; AlertDialog 通常只有头部 + 页脚)。
func (a *AlertDialog) Content() int {
	if a == nil {
		return 0
	}
	return a.d.Content()
}

// ContentWidth 内容容器宽度(面板宽 - 左右 padding)。
func (a *AlertDialog) ContentWidth() int32 {
	if a == nil {
		return 0
	}
	return a.d.ContentWidth()
}

// Show 打开警示对话框。
func (a *AlertDialog) Show() *AlertDialog {
	if a != nil {
		a.d.Show()
	}
	return a
}

// Hide 关闭(不触发 OnClose 回调)。
func (a *AlertDialog) Hide() *AlertDialog {
	if a != nil {
		a.d.Hide()
	}
	return a
}

// Close 关闭并触发关闭回调。
func (a *AlertDialog) Close() *AlertDialog {
	if a != nil {
		a.d.Close()
	}
	return a
}

// IsOpen 报告是否打开。
func (a *AlertDialog) IsOpen() bool {
	return a != nil && a.d.IsOpen()
}

// OnClose 注册关闭回调(ESC / Cancel / Action / Close 都会触发; 点遮罩不会)。
func (a *AlertDialog) OnClose(fn func()) *AlertDialog {
	if a != nil {
		a.d.OnClose(fn)
	}
	return a
}

// Refresh 重新量内容并重排(打开状态下改过内容后调用)。
func (a *AlertDialog) Refresh() *AlertDialog {
	if a != nil {
		a.d.Refresh()
	}
	return a
}

// applyAlignment 按尺寸切换标题/描述对齐: default 左对齐, sm 居中。
func (a *AlertDialog) applyAlignment() {
	if a == nil || a.d == nil {
		return
	}
	if a.d.alertSize == AlertDialogSizeSmall {
		a.d.title.SetAlignCenter()
		a.d.desc.SetAlignCenter()
	} else {
		a.d.title.SetAlignLeft()
		a.d.desc.SetAlignLeft()
	}
}
