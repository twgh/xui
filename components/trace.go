package ui

import (
	"math"
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Trace 相关几何常量(逻辑像素), 对齐 shadcn trace.tsx:
//   - 行内 icon 16、gap-2(8)、text-sm(14) muted-foreground;
//   - 描述文字 text-xs(12)、muted-foreground/70;
//   - running 标题微光扫过, error 红色, pending 60% 不透明度;
//   - 可折叠触发条右侧 chevron 展开时旋转 90°(200ms)。
const (
	traceRowGap        int32 = 8
	traceIconSize      int32 = 16
	traceLineGap       int32 = 4
	traceContentIndent int32 = 24
	traceContentTop    int32 = 6
)

// TraceStatus 轨迹行的状态, 对应 shadcn trace.tsx 的 status。
type TraceStatus int

const (
	TraceStatusNone    TraceStatus = iota // 普通行(默认)
	TraceStatusRunning                    // 进行中: 标题微光
	TraceStatusError                      // 失败: 红色
	TraceStatusPending                    // 等待: 60% 不透明度
)

// TraceOption 轨迹列表配置项(保留给将来)。
type TraceOption func(*Trace)

// Trace 轨迹列表: 竖向排列的 TraceLine(推理过程 / 工具调用 / 来源)。
//
// 与 shadcn trace.tsx 对齐: 宽铺满、子项间距 4, 行内 icon + 标题 + 描述;
// 每行可点击(悬停变前景色), 也可用 TraceCollapsible 折叠更多内容。
type Trace struct {
	*widget.LayoutEle
}

// NewTrace 创建轨迹列表。宽铺满、高自撑。
func NewTrace(x, y, cx, cy int32, hParent int, opts ...TraceOption) *Trace {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	t := &Trace{LayoutEle: ele}
	for _, o := range opts {
		o(t)
	}
	ele.EnableHorizon(false)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(traceLineGap)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return t
}

// TraceLineOption 轨迹行配置项。
type TraceLineOption func(*TraceLine)

// WithTraceStatus 设置行状态(默认 TraceStatusNone)。
func WithTraceStatus(s TraceStatus) TraceLineOption {
	return func(l *TraceLine) { l.status = s }
}

// WithTraceIcon 在行首放一个 16px 图标(lucide path)。
func WithTraceIcon(icon string) TraceLineOption {
	return func(l *TraceLine) { l.icon = icon }
}

// WithTraceTitle 设置标题(text-sm, 吃剩余宽度, 超宽省略号)。
func WithTraceTitle(text string) TraceLineOption {
	return func(l *TraceLine) { l.titleText = text }
}

// WithTraceDescription 设置行尾描述(text-xs, 弱化 70%)。
func WithTraceDescription(text string) TraceLineOption {
	return func(l *TraceLine) { l.descText = text }
}

// WithTraceTrailingIcon 在行尾放一个图标(例如可点击行的手势提示 chevron)。
func WithTraceTrailingIcon(icon string) TraceLineOption {
	return func(l *TraceLine) { l.trailingIcon = icon }
}

// WithTraceOnClick 让整行可点击: 悬停文字变前景色, 松开左键回调。
func WithTraceOnClick(fn func()) TraceLineOption {
	return func(l *TraceLine) { l.onClick = fn }
}

// TraceLine 一条轨迹行: [图标][标题(吃剩余宽)][描述][尾部图标]。
type TraceLine struct {
	*widget.LayoutEle
	status       TraceStatus
	icon         string
	titleText    string
	descText     string
	trailingIcon string
	onClick      func()
	hover        bool
	title        *chatText
}

// NewTraceLine 创建轨迹行。宽铺满、高自撑、行内 gap-8、垂直居中。
func NewTraceLine(x, y, cx, cy int32, hParent int, opts ...TraceLineOption) *TraceLine {
	ele := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	l := &TraceLine{LayoutEle: ele}
	for _, o := range opts {
		o(l)
	}
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableAutoWrap(false)
	ele.SetSpace(traceRowGap)
	ele.SetAlignV(xcc.Layout_Align_Center)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)

	// 子项按固定顺序创建: 图标 / 标题 / 描述 / 尾部图标。
	// 子元素也要绑同一套悬停/点击: 点标题文字时事件落在子元素上, 不会冒泡到行容器。
	if l.icon != "" {
		ic := newChatIcon(ele.Handle, l.icon, traceIconSize)
		if ic != nil {
			ic.SetColorFunc(l.colorOf)
			l.bindChild(ic.Handle)
		}
	}
	if l.titleText != "" || l.descText != "" || l.trailingIcon != "" {
		if l.titleText != "" {
			c := newChatFillText(ele.Handle, l.titleText, chatTextSize)
			if c != nil {
				// 主轴上 Fill 会独占整行、把尾部图标挤出去(见 AGENTS 布局规则),
				// 标题吃"剩余空间"必须用 Weight。
				c.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
				c.SetColorFunc(l.colorOf)
				if l.status == TraceStatusRunning {
					c.SetShimmer(true)
				}
				l.title = c
				l.bindChild(c.Handle)
			}
		}
		if l.descText != "" {
			c := newChatFitText(ele.Handle, l.descText, chatMetaSize)
			if c != nil {
				c.SetColorFunc(func(t *xui.Theme) xui.Color {
					return t.MutedForeground.ScaleAlpha(0.7)
				})
				l.bindChild(c.Handle)
			}
		}
		if l.trailingIcon != "" {
			ic := newChatIcon(ele.Handle, l.trailingIcon, traceIconSize)
			if ic != nil {
				ic.SetColorFunc(l.colorOf)
				l.bindChild(ic.Handle)
			}
		}
	}
	xui.AttachWindow(ele.Handle)
	if l.onClick != nil {
		ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
			l.setHover(true)
			return 0
		})
		ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
			l.setHover(false)
			return 0
		})
		ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			return 0
		})
		ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
			if l.onClick != nil {
				l.onClick()
			}
			return 0
		})
	}
	return l
}

// bindChild 把行内子元素的悬停/点击接到行容器上:
// 点击标题文字时事件不会冒泡到行, 必须让子元素自己转发。
func (l *TraceLine) bindChild(h int) {
	if l == nil || h == 0 || l.onClick == nil {
		return
	}
	ele := widget.NewElementByHandle(h)
	if ele == nil {
		return
	}
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		l.setHover(true)
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		l.setHover(false)
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if l.onClick != nil {
			l.onClick()
		}
		return 0
	})
}

// colorOf 返回行的基础文字色: 普通 muted、error 红、pending 60%。
// 可点击行悬停时淡入前景色(error 保持红色, 与上游一致)。
func (l *TraceLine) colorOf(t *xui.Theme) xui.Color {
	c := t.MutedForeground
	switch l.status {
	case TraceStatusError:
		c = t.Destructive
	case TraceStatusPending:
		c = t.MutedForeground.ScaleAlpha(0.6)
	}
	if l.onClick != nil {
		target := t.Foreground
		if l.status == TraceStatusError {
			target = t.Destructive
		}
		c = xui.Fade(c, target, xui.Hover(l.Handle, l.hover))
	}
	return c
}

// setHover 更新悬停态并重绘行内所有子项(颜色在各自 paint 里实时算)。
func (l *TraceLine) setHover(hover bool) {
	if l == nil || l.hover == hover {
		return
	}
	l.hover = hover
	n := xc.XEle_GetChildCount(l.Handle)
	for i := int32(0); i < n; i++ {
		if h := xc.XEle_GetChildByIndex(l.Handle, i); h != 0 {
			xc.XEle_Redraw(h, true)
		}
	}
}

// Title 返回标题文本元素(可能为 nil, 取决于是否配置了标题)。
func (l *TraceLine) Title() *chatText {
	if l == nil {
		return nil
	}
	return l.title
}

// TraceCollapsibleOption 可折叠轨迹段配置项。
type TraceCollapsibleOption func(*traceCollapsibleConfig)

type traceCollapsibleConfig struct {
	open     bool
	lineOpts []TraceLineOption
	onOpen   func(open bool)
}

// WithTraceCollapsibleOpen 设置初始展开状态(不播动画)。
func WithTraceCollapsibleOpen(open bool) TraceCollapsibleOption {
	return func(c *traceCollapsibleConfig) { c.open = open }
}

// WithTraceCollapsibleTrigger 设置触发条的行配置(图标/标题/描述/状态/点击)。
func WithTraceCollapsibleTrigger(opts ...TraceLineOption) TraceCollapsibleOption {
	return func(c *traceCollapsibleConfig) { c.lineOpts = append(c.lineOpts, opts...) }
}

// WithTraceCollapsibleOnOpen 监听展开/收起。
func WithTraceCollapsibleOnOpen(fn func(open bool)) TraceCollapsibleOption {
	return func(c *traceCollapsibleConfig) { c.onOpen = fn }
}

// TraceCollapsible 可折叠的轨迹段: 触发条 + 缩进内容。
//
// 高度过渡复用 Collapsible(200ms); 触发条右侧的 chevron 展开时旋转 90°,
// 旋转用 Presence 过渡, 不自己起定时器。
type TraceCollapsible struct {
	*Collapsible
	trigger *TraceLine
	arrow   *chatIcon
	presKey int32
}

// NewTraceCollapsible 创建可折叠轨迹段。触发条立即建好(含尾部 chevron),
// 内容用 Content()/ContentStack() 追加在触发条下方。
func NewTraceCollapsible(x, y, cx, cy int32, hParent int, opts ...TraceCollapsibleOption) *TraceCollapsible {
	cfg := traceCollapsibleConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	c := NewCollapsible(x, y, cx, cy, hParent, WithCollapsibleOpen(cfg.open))
	if c == nil {
		return nil
	}
	_ = c
	tc := &TraceCollapsible{Collapsible: c, presKey: xui.NextPresenceKey()}
	lineOpts := append([]TraceLineOption{}, cfg.lineOpts...)
	lineOpts = append(lineOpts, WithTraceOnClick(func() {
		c.Toggle()
	}))
	tc.trigger = NewTraceLine(0, 0, 0, 0, c.Handle, lineOpts...)
	if tc.trigger != nil {
		tc.arrow = newChatIcon(tc.trigger.Handle, icons.IconChevronRight, traceIconSize)
		if tc.arrow != nil {
			tc.arrow.SetColorFunc(func(t *xui.Theme) xui.Color { return t.MutedForeground })
			if c.Open() {
				tc.arrow.SetAngle(math.Pi / 2)
			}
		}
	}
	c.OnOpenChange(func(open bool) {
		tc.animateArrow(open)
		if cfg.onOpen != nil {
			cfg.onOpen(open)
		}
	})
	return tc
}

// Trigger 返回触发条。
func (tc *TraceCollapsible) Trigger() *TraceLine {
	if tc == nil {
		return nil
	}
	return tc.trigger
}

// Content 返回缩进的内容区(左侧 24、顶部 6)。第一次调用会接在触发条后面。
func (tc *TraceCollapsible) Content() *widget.LayoutEle {
	if tc == nil {
		return nil
	}
	body := tc.Collapsible.Content()
	if body == nil {
		return nil
	}
	xc.XEle_SetPadding(body.Handle, traceContentIndent, traceContentTop, 0, 0)
	return body
}

// ContentStack 设置内容区子项间距并返回它。
func (tc *TraceCollapsible) ContentStack(space int32) *widget.LayoutEle {
	body := tc.Content()
	if body != nil {
		body.SetSpace(space)
	}
	return body
}

// animateArrow 用 Presence 把 chevron 转到打开/收起角度。
func (tc *TraceCollapsible) animateArrow(open bool) {
	if tc == nil || tc.arrow == nil {
		return
	}
	host := xui.HostWindowOf(tc.arrow.Handle)
	if host == 0 {
		if open {
			tc.arrow.SetAngle(math.Pi / 2)
		} else {
			tc.arrow.SetAngle(0)
		}
		return
	}
	xui.Presence(host, tc.presKey, open, 200*time.Millisecond, func(v float64) {
		tc.arrow.SetAngle(v * math.Pi / 2)
	}, nil)
}
