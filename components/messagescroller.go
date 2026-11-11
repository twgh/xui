package ui

import (
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 消息滚动容器的几何与过渡常量(逻辑像素 / 时长), 对齐 shadcn message-scroller 的默认值:
//   - scrollEdgeThreshold = 8(离边 8px 内算"在边上");
//   - scrollPreviousItemPeek = 64(把新锚点顶到视口顶部时, 上方留出上一条的 64px 尾巴);
//   - 内容区 gap-8 = 32;
//   - 浮动按钮 icon-sm = 28×28, 距底/顶 16(bottom-4 / top-4),
//     出现 200ms / 消失 400ms(组件类名里 data-active 的两档 duration)。
const (
	MessageScrollerEdgeThresholdDefault int32 = 8
	MessageScrollerPreviousItemPeek     int32 = 64
	messageScrollerGap                        = 32
	messageScrollerButtonSize                 = 28
	messageScrollerButtonInset                = 16
	messageScrollerButtonIconSize             = 16
	// 滚动单位设为 1px 后滚轮每格只剩 3px, 由组件自己按原生默认的 3 行(3×20)折算。
	messageScrollerWheelStep = 60
)

const (
	messageScrollerShowDur = 200 * time.Millisecond
	messageScrollerHideDur = 400 * time.Millisecond
	messageScrollerScrollD = 220 * time.Millisecond
)

// MessageScrollerDefaultScrollPosition 首次有内容时滚到哪里, 对应上游 provider 的
// defaultScrollPosition。
type MessageScrollerDefaultScrollPosition int

const (
	MessageScrollerScrollEnd        MessageScrollerDefaultScrollPosition = iota // 末尾(默认, 聊天默认停在最新一条)
	MessageScrollerScrollStart                                                  // 顶部
	MessageScrollerScrollLastAnchor                                             // 最后一个轮次锚点; 装不下时把它顶到视口顶部
)

// MessageScrollerButtonDirection 浮动按钮的方向: end 在底部(跳到最新), start 在顶部(跳到开头)。
type MessageScrollerButtonDirection int

const (
	MessageScrollerButtonEnd   MessageScrollerButtonDirection = iota // 底部(默认)
	MessageScrollerButtonStart                                       // 顶部, 图标旋转 180°
)

// MessageScrollerAlign 对齐方式, 对应上游 scrollToMessage 的 align。
type MessageScrollerAlign int

const (
	MessageScrollerAlignStart   MessageScrollerAlign = iota // 顶部(默认)
	MessageScrollerAlignCenter                              // 垂直居中
	MessageScrollerAlignEnd                                 // 底部
	MessageScrollerAlignNearest                             // 已完整可见就不动, 否则就近对齐
)

// MessageScrollerOption 消息滚动容器配置项。
type MessageScrollerOption func(*MessageScroller)

// WithMessageScrollerAutoScroll 只在贴底时跟随新内容(默认 false)。
func WithMessageScrollerAutoScroll(auto bool) MessageScrollerOption {
	return func(s *MessageScroller) { s.autoScroll = auto }
}

// WithMessageScrollerDefaultScrollPosition 设置首次有内容时的落点(默认 ScrollEnd)。
func WithMessageScrollerDefaultScrollPosition(pos MessageScrollerDefaultScrollPosition) MessageScrollerOption {
	return func(s *MessageScroller) { s.defaultPos = pos }
}

// WithMessageScrollerEdgeThreshold 设置"在边上"的判定距离(默认 8)。
func WithMessageScrollerEdgeThreshold(px int32) MessageScrollerOption {
	return func(s *MessageScroller) {
		if px >= 0 {
			s.edgeThreshold = px
		}
	}
}

// WithMessageScrollerPreviousItemPeek 设置轮次锚点上方的上一条尾巴高度(默认 64)。
func WithMessageScrollerPreviousItemPeek(px int32) MessageScrollerOption {
	return func(s *MessageScroller) {
		if px >= 0 {
			s.peek = px
		}
	}
}

// WithMessageScrollerScrollMargin 设置锚点时额外让出的上边距(默认 0)。
func WithMessageScrollerScrollMargin(px int32) MessageScrollerOption {
	return func(s *MessageScroller) {
		if px >= 0 {
			s.scrollMargin = px
		}
	}
}

// WithMessageScrollerButtonDirection 设置浮动按钮的方向(默认底部)。
func WithMessageScrollerButtonDirection(d MessageScrollerButtonDirection) MessageScrollerOption {
	return func(s *MessageScroller) { s.buttonDirection = d }
}

// MessageScroller 会话滚动容器: 轮次锚定 + 只在贴底时跟随 + 前置历史不跳动 + 跳到任意消息。
//
// 用法:
//
//	ms := ui.NewMessageScroller(0, 0, 0, 0, card.Content().Handle,
//		ui.WithMessageScrollerAutoScroll(true))
//	xui.ItemFill(ms)
//	ms.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 520) // 或放进 Weight 链
//	item := ms.AddItem("m1", true)                      // 第二条参数 = 轮次锚点
//	ui.NewBubble(item.Handle, "Hello")
//
// 行为(对齐 shadcn message-scroller):
//   - 追加新内容时, 若读者贴在底部且 autoScroll 打开 → 跟到末尾; 否则把新轮次的
//     第一个锚点项顶到视口顶部, 并保留上一条的 peek 像素;
//   - 前置历史(prepend)默认保持视口里的内容不动;
//   - 用户在任意滚动之后离开底部 → 取消跟随; 滚回底部 → 重新跟随;
//   - 轮次锚点正在被流式追加内容时, 锚点保持在视口顶部;
//   - 浮动按钮只在对应方向还能滚时出现, 点击平滑滚动到端点; 不占用布局。
//
// 高度: 默认 Weight 1(吃父级剩余), 和 ScrollArea 一样需要父级高度确定;
// 放在 Auto 高的卡片里时请显式 LayoutItem_SetHeight(Fixed, h)。
type MessageScroller struct {
	*widget.Element

	scroll  *ScrollArea
	content *widget.LayoutEle
	spacer  *widget.LayoutEle
	button  *Button

	autoScroll      bool
	defaultPos      MessageScrollerDefaultScrollPosition
	edgeThreshold   int32
	peek            int32
	scrollMargin    int32
	buttonDirection MessageScrollerButtonDirection

	items []*MessageScrollerItem
	byID  map[string]*MessageScrollerItem

	defaultApplied bool
	defaultPending bool
	syncQueued     bool
	following      bool
	anchoredID     string

	// 程序自己发起的滚动: 滚回事件里不能当成"用户离开底部"。
	selfScroll bool
	adjusting  bool

	// Add/Prepend/RemoveItem 的合并处理: 一次消息循环只跑一遍。
	mutPending   bool
	mutPrepend   bool
	mutFirstH    int
	mutFirstTop  int32
	mutScrollPos int32

	// 外部内容变高(流式文字)合并处理。
	resizePending bool

	// 待跳转的消息(先于内容创建时调用 ScrollToMessage)。
	pendingMessageID string
	pendingAlign     MessageScrollerAlign

	buttonKey   int32
	buttonV     float64
	buttonShown bool
	buttonWant  bool
	animGen     int
	animating   bool

	onScrollState func(start, end bool)
	onVisibility  func(currentAnchorID string, visible []string)
	lastStart     bool
	lastEnd       int8 // -1 未初始化, 0 false, 1 true
	lastAnchor    string
	lastVisible   string
	visPending    bool
	statePending  bool
	destroyed     bool
}

// MessageScrollerItem 会话里的一条消息边界: 宽度铺满、高度自撑, 记录消息 id 与是否轮次锚点。
type MessageScrollerItem struct {
	*widget.LayoutEle
	scroller  *MessageScroller
	messageID string
	anchor    bool
	handled   bool // 已参与过落点决策(追加时只认新出现的锚点)
}

func messageScrollerButtonIcon(d MessageScrollerButtonDirection) string {
	if d == MessageScrollerButtonStart {
		return icons.IconArrowUp
	}
	return icons.IconArrowDown
}

// NewMessageScroller 创建消息滚动容器。尺寸/布局项语义同普通容器(默认宽 Fill、高 Weight 1)。
func NewMessageScroller(x, y, cx, cy int32, hParent int, opts ...MessageScrollerOption) *MessageScroller {
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	s := &MessageScroller{
		Element:       ele,
		edgeThreshold: MessageScrollerEdgeThresholdDefault,
		peek:          MessageScrollerPreviousItemPeek,
		byID:          make(map[string]*MessageScrollerItem),
		lastEnd:       -1,
	}
	for _, o := range opts {
		o(s)
	}

	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	ele.EnableFocus(false)
	ele.EnableSwitchFocus(false)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Weight, 1)

	s.scroll = NewScrollArea(0, 0, cx, cy, ele.Handle)
	if s.scroll == nil {
		return nil
	}
	s.content = s.scroll.Content()
	s.content.SetSpace(messageScrollerGap)
	// ScrollPosYV 会按滚动单位(默认 20px/行)向下吸附: 388 → 380。锚点要求像素级落位,
	// 所以滚动单位设为 1px; 滚轮步进随滚动单位一起变成 3px, 下面单独按 60px/格处理。
	s.scroll.SetLineSize(1, 1)
	s.scroll.AddEvent_MouseWheel(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		delta := int32(int16(uint16(uint32(nFlags) >> 16)))
		if delta == 0 {
			return 0
		}
		step := -delta * messageScrollerWheelStep / 120
		s.cancelScrollAnim()
		s.scrollToSilent(s.scroll.GetViewPosV() + step)
		s.noteUserScroll(s.atEnd())
		s.syncState()
		*pbHandled = true
		return 0
	})

	// 尾部留白: 把"最后一个轮次"顶到视口顶部时, 内容末端可能还没有视口高,
	// 需要一段可滚动的空白才能让目标位置可达(对应上游 content 里的 spacer)。
	s.spacer = widget.NewLayoutEle(0, 0, 0, 0, s.content.Handle)
	s.spacer.EnableBkTransparent(true)
	s.spacer.EnableFocus(false)
	s.spacer.EnableDrawBorder(false)
	s.spacer.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	s.spacer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
	s.spacer.Show(false)

	s.button = NewButton(0, 0, messageScrollerButtonSize, messageScrollerButtonSize, "", ele.Handle,
		WithVariant(ButtonOutline), WithSize(ButtonSizeIconSm),
		WithButtonRounded(), WithIconSize(messageScrollerButtonIconSize),
		WithIcon(messageScrollerButtonIcon(s.buttonDirection)))
	if s.button != nil {
		// 不调 SetZOrder(0): 那会把它放到最底层。按钮在滚动视口之后创建, 天然在上层。
		s.button.Show(false)
		s.button.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
			s.onButtonClick()
			return 0
		})
	}
	s.buttonKey = xui.NextPresenceKey()

	ele.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		s.layoutViewport()
		s.scheduleSync()
		return 0
	})
	s.scroll.AddEvent_ScrollView_Scroll_V(func(hEle int, pos int32, pbHandled *bool) int {
		s.onScroll()
		return 0
	})
	s.content.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		if s.adjusting {
			return 0
		}
		s.scheduleResize()
		return 0
	})
	ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		s.destroyed = true
		xui.DropPresence(ele.Handle, s.buttonKey)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	s.layoutViewport()
	return s
}

// Content 返回内容区(竖向流容器): 宽铺满视口、高按内容自撑。一般用 AddItem 加条目,
// 也可以直接往里放其它元素(它们不参与消息 id / 锚点跟踪)。
func (s *MessageScroller) Content() *widget.LayoutEle {
	if s == nil {
		return nil
	}
	return s.content
}

// ContentStack 设置内容区子项间距并返回内容区。
func (s *MessageScroller) ContentStack(space int32) *widget.LayoutEle {
	if s == nil {
		return nil
	}
	s.content.SetSpace(space)
	return s.content
}

// AddItem 在末尾追加一条消息边界并返回它(往里放 Message / Bubble 等内容)。
// 同一个 id 重复调用返回已有条目, 不会重复创建。
func (s *MessageScroller) AddItem(messageID string, scrollAnchor bool) *MessageScrollerItem {
	if s == nil {
		return nil
	}
	if it := s.byID[messageID]; it != nil {
		return it
	}
	it := s.newItem(messageID, scrollAnchor)
	if it == nil {
		return nil
	}
	// 条目创建时不挂父级(InsertChild 要求游离对象), 插到 spacer 前面。
	if !s.content.InsertChild(it.Handle, int32(xc.XEle_GetChildCount(s.content.Handle)-1)) {
		it.Destroy()
		return nil
	}
	xui.AttachWindow(it.Handle)
	s.items = append(s.items, it)
	s.byID[messageID] = it
	s.recordMutation(false)
	return it
}

// PrependItem 在开头插入一条消息边界(加载更早的历史); 默认保持当前视口内容不跳动。
func (s *MessageScroller) PrependItem(messageID string, scrollAnchor bool) *MessageScrollerItem {
	if s == nil {
		return nil
	}
	if it := s.byID[messageID]; it != nil {
		return it
	}
	it := s.newItem(messageID, scrollAnchor)
	if it == nil {
		return nil
	}
	if s.mutFirstH == 0 && len(s.items) > 0 {
		s.mutFirstH = s.items[0].Handle
		s.mutFirstTop = s.items[0].rect().Top
		s.mutScrollPos = s.scroll.GetViewPosV()
	}
	if !s.content.InsertChild(it.Handle, 0) {
		it.Destroy()
		return nil
	}
	xui.AttachWindow(it.Handle)
	s.items = append([]*MessageScrollerItem{it}, s.items...)
	s.byID[messageID] = it
	s.recordMutation(true)
	return it
}

// RemoveItem 移除一条消息边界并销毁它。
func (s *MessageScroller) RemoveItem(messageID string) bool {
	if s == nil {
		return false
	}
	it := s.byID[messageID]
	if it == nil {
		return false
	}
	delete(s.byID, messageID)
	for i, x := range s.items {
		if x == it {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	if s.anchoredID == messageID {
		s.anchoredID = ""
	}
	if s.mutFirstH == it.Handle {
		s.mutFirstH = 0
	}
	it.LayoutEle.Remove()
	it.Destroy()
	s.recordMutation(false)
	return true
}

// Item 按 id 取条目, 不存在返回 nil。
func (s *MessageScroller) Item(messageID string) *MessageScrollerItem {
	if s == nil {
		return nil
	}
	return s.byID[messageID]
}

func (s *MessageScroller) newItem(messageID string, scrollAnchor bool) *MessageScrollerItem {
	// hParent = 0: 先建游离元素, 由 AddItem/PrependItem 用 InsertChild 放到指定位置。
	l := widget.NewLayoutEle(0, 0, 0, 0, 0)
	if l == nil {
		return nil
	}
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableAutoWrap(false)
	l.EnableBkTransparent(true)
	l.EnableDrawBorder(false)
	l.EnableFocus(false)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	it := &MessageScrollerItem{LayoutEle: l, scroller: s, messageID: messageID, anchor: scrollAnchor}
	// 条目内部改高(折叠展开 / 流式文本)也要走同一套"跟随或重锚"处理。
	l.AddEvent_Size(func(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
		if s.adjusting {
			return 0
		}
		s.scheduleResize()
		return 0
	})
	return it
}

// rect 返回条目在内容坐标系里的矩形(滚动偏移不参与, 见 DESIGN 第八节 66)。
func (it *MessageScrollerItem) rect() xc.RECT {
	var r xc.RECT
	if it != nil && it.Handle != 0 {
		xc.XEle_GetRect(it.Handle, &r)
	}
	return r
}

// MessageID 返回条目的消息 id。
func (it *MessageScrollerItem) MessageID() string {
	if it == nil {
		return ""
	}
	return it.messageID
}

// ScrollAnchor 返回条目是否是轮次起点。
func (it *MessageScrollerItem) ScrollAnchor() bool {
	if it == nil {
		return false
	}
	return it.anchor
}

// SetScrollAnchor 运行时修改轮次锚点标记(影响之后追加内容时的锚定选择)。
func (it *MessageScrollerItem) SetScrollAnchor(anchor bool) *MessageScrollerItem {
	if it == nil {
		return it
	}
	it.anchor = anchor
	return it
}

// Remove 从滚动容器里移除并销毁这条消息。
func (it *MessageScrollerItem) Remove() {
	if it == nil || it.scroller == nil {
		return
	}
	it.scroller.RemoveItem(it.messageID)
}

// SetAutoScroll 运行时开关"贴底跟随"。
func (s *MessageScroller) SetAutoScroll(auto bool) *MessageScroller {
	if s == nil {
		return s
	}
	s.autoScroll = auto
	if auto && s.atEnd() {
		s.following = true
	}
	if !auto {
		s.following = false
	}
	s.syncState()
	return s
}

// ScrollState 返回当前是否还能向两端滚动(对应上游 useMessageScrollerScrollable)。
func (s *MessageScroller) ScrollState() (start, end bool) {
	if s == nil {
		return false, false
	}
	return s.edges()
}

// CurrentAnchorID 返回视口顶部所在轮次的锚点消息 id(对应上游 currentAnchorId)。
func (s *MessageScroller) CurrentAnchorID() string {
	if s == nil {
		return ""
	}
	return s.currentAnchorID()
}

// VisibleMessageIDs 返回当前视口里可见的消息 id(按内容顺序)。
func (s *MessageScroller) VisibleMessageIDs() []string {
	if s == nil {
		return nil
	}
	return s.visibleIDs()
}

// OnScrollStateChange 注册滚动可达状态变化回调(start/end 任一翻转时触发)。
func (s *MessageScroller) OnScrollStateChange(fn func(start, end bool)) *MessageScroller {
	if s == nil {
		return s
	}
	s.onScrollState = fn
	if fn != nil {
		s.lastEnd = -1
		s.syncState()
	}
	return s
}

// OnVisibilityChange 注册视口可见消息 / 当前锚点变化回调。
func (s *MessageScroller) OnVisibilityChange(fn func(currentAnchorID string, visible []string)) *MessageScroller {
	if s == nil {
		return s
	}
	s.onVisibility = fn
	if fn != nil {
		s.lastAnchor = "\x00"
		s.lastVisible = "\x00"
		s.syncState()
	}
	return s
}

// ScrollToEnd 滚到末尾(取消轮次锚定; autoScroll 打开时重新进入跟随)。
func (s *MessageScroller) ScrollToEnd() bool {
	if s == nil {
		return false
	}
	return s.scrollToEnd(false)
}

// ScrollToStart 滚到开头。
func (s *MessageScroller) ScrollToStart() bool {
	if s == nil {
		return false
	}
	return s.scrollToStart(false)
}

// ScrollToMessage 把指定消息滚到视口顶部; 内容尚未创建时挂起, 等它出现后再生效。
// 返回 false 表示消息已存在但当前不可滚(例如容器还没有尺寸)。
func (s *MessageScroller) ScrollToMessage(messageID string) bool {
	return s.ScrollToMessageAligned(messageID, MessageScrollerAlignStart)
}

// ScrollToMessageAligned 同 ScrollToMessage, 可指定对齐方式。
func (s *MessageScroller) ScrollToMessageAligned(messageID string, align MessageScrollerAlign) bool {
	if s == nil {
		return false
	}
	if it := s.byID[messageID]; it != nil {
		return s.scrollToItem(it, align, s.scrollMargin, false, false)
	}
	if len(s.items) == 0 {
		s.pendingMessageID, s.pendingAlign = messageID, align
		return true
	}
	return false
}

// SetViewPadding 设置滚动内容四边的内边距(左右用元素 padding, 上下计入滚动范围)。
func (s *MessageScroller) SetViewPadding(left, top, right, bottom int32) *MessageScroller {
	if s == nil {
		return s
	}
	s.content.SetPadding(left, top, right, bottom)
	s.relayout()
	return s
}

// ===== 内部: 布局 / 状态 =====

// layoutViewport 把滚动视口铺满根元素; 浮动按钮是根元素的子项, 不参与流式布局。
func (s *MessageScroller) layoutViewport() {
	if s == nil || s.scroll == nil {
		return
	}
	w, h := s.GetWidth(), s.GetHeight()
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	s.scroll.SetRectEx(0, 0, w, h, false, xcc.AdjustLayout_No)
	s.adjusting = true
	s.scroll.AdjustLayout()
	s.adjusting = false
	if s.buttonShown || s.buttonWant {
		s.positionButton(s.buttonV)
	}
}

// positionButton 按当前显隐进度摆放按钮(v=0 完全滑出根元素, v=1 贴边 16px)。
func (s *MessageScroller) positionButton(v float64) {
	if s.button == nil {
		return
	}
	w, h := s.GetWidth(), s.GetHeight()
	bw, bh := int32(messageScrollerButtonSize), int32(messageScrollerButtonSize)
	x := (w - bw) / 2
	off := int32((1 - v) * float64(bh))
	var y int32
	if s.buttonDirection == MessageScrollerButtonStart {
		y = messageScrollerButtonInset - off
	} else {
		y = h - messageScrollerButtonInset - bh + off
	}
	s.button.SetRectEx(x, y, bw, bh, false, xcc.AdjustLayout_No)
}

// setButtonActive 淡入/滑入或淡出/滑出浮动按钮。
func (s *MessageScroller) setButtonActive(active bool) {
	if s == nil || s.button == nil || active == s.buttonWant {
		return
	}
	s.buttonWant = active
	if active {
		s.button.Show(true)
		s.buttonShown = true
	}
	dur := messageScrollerShowDur
	if !active {
		dur = messageScrollerHideDur
	}
	xui.Presence(s.Handle, s.buttonKey, active, dur, func(v float64) {
		if s.destroyed {
			return
		}
		s.buttonV = v
		s.positionButton(v)
		xui.SetElementsAlpha(v, s.button.Handle)
	}, func(float64) {
		if s.destroyed {
			return
		}
		if !active {
			s.button.Show(false)
			s.buttonShown = false
		}
	})
}

func (s *MessageScroller) onButtonClick() {
	if s.button == nil {
		return
	}
	// 按钮获得焦点后会画出焦点框, 点击后立即交还焦点(上游是 currentTarget.blur())。
	if hwnd := xc.XWidget_GetHWINDOW(s.Handle); hwnd != 0 {
		xc.XWnd_SetFocusEle(hwnd, 0)
	}
	if s.buttonDirection == MessageScrollerButtonStart {
		s.scrollToStart(true)
	} else {
		s.scrollToEnd(true)
	}
}

// onScroll 滚动事件: 事件参数 pos 是"行"单位(见 DESIGN 第八节), 像素位置必须用 GetViewPosV。
func (s *MessageScroller) onScroll() {
	if s == nil || s.scroll == nil {
		return
	}
	if !s.selfScroll {
		s.noteUserScroll(s.atEnd())
	}
	s.syncState()
}

// noteUserScroll 更新"用户滚动"后的跟随状态: 回到末尾重新跟随, 离开末尾立即放手。
func (s *MessageScroller) noteUserScroll(atEnd bool) {
	s.cancelScrollAnim()
	if atEnd {
		if s.autoScroll {
			s.following = true
			s.anchoredID = ""
		}
		return
	}
	s.following = false
	s.anchoredID = ""
}

// edges 返回两端是否还能滚(上游 Scrollable 语义: end = 还能向末尾滚, 不是"已在末尾")。
func (s *MessageScroller) edges() (start, end bool) {
	if s == nil || s.scroll == nil {
		return false, false
	}
	pos := s.scroll.GetViewPosV()
	max := s.maxScroll()
	start = pos > s.edgeThreshold
	end = max-pos > s.edgeThreshold
	return
}

func (s *MessageScroller) atEnd() bool {
	return !s.canScrollEnd()
}

func (s *MessageScroller) canScrollEnd() bool {
	_, end := s.edges()
	return end
}

// maxScroll 返回内容区(含顶/底内边距与 spacer)的最大像素滚动位。
func (s *MessageScroller) maxScroll() int32 {
	var sz xc.SIZE
	xc.XSView_GetTotalSize(s.scroll.Handle, &sz)
	max := sz.CY - s.scroll.GetViewHeight()
	if max < 0 {
		return 0
	}
	return max
}

// metrics 返回内容总高与视口高(先做一次布局, 保证总高是最新的)。
// relayout 重排内容与视口。期间压掉内容/条目的尺寸事件, 避免自己的调整触发
// "外部内容变高"那条重锚链路(会与当前操作互相触发)。
func (s *MessageScroller) relayout() {
	if s == nil || s.content == nil || s.scroll == nil {
		return
	}
	s.adjusting = true
	s.content.AdjustLayout()
	s.scroll.AdjustLayout()
	s.adjusting = false
}

func (s *MessageScroller) metrics() (total, viewH int32) {
	s.relayout()
	var sz xc.SIZE
	xc.XSView_GetTotalSize(s.scroll.Handle, &sz)
	return sz.CY, s.scroll.GetViewHeight()
}

func (s *MessageScroller) syncState() {
	if s == nil || s.scroll == nil {
		return
	}
	start, end := s.edges()
	if s.onScrollState != nil {
		if s.lastEnd < 0 || start != s.lastStart || end != (s.lastEnd == 1) {
			s.lastStart = start
			if end {
				s.lastEnd = 1
			} else {
				s.lastEnd = 0
			}
			// 滚动事件里同步改 Label 文本时, 界面可能正在滚动/绘制这棵树, 重绘请求会被吞掉
			// (实测: 首次落底后状态文字停在中间态)。和可见性一样合并到下一轮消息循环再发。
			if !s.statePending {
				s.statePending = true
				app.AfterFunc(0, func() {
					s.statePending = false
					if s.destroyed || s.onScrollState == nil {
						return
					}
					curStart, curEnd := s.edges()
					s.onScrollState(curStart, curEnd)
				})
			}
		}
	}
	if s.buttonDirection == MessageScrollerButtonStart {
		s.setButtonActive(start)
	} else {
		s.setButtonActive(end)
	}
	s.emitVisibility()
}

func (s *MessageScroller) visibleIDs() []string {
	if s == nil || s.scroll == nil {
		return nil
	}
	pos := s.scroll.GetViewPosV()
	bottom := pos + s.scroll.GetViewHeight()
	var out []string
	for _, it := range s.items {
		if it.messageID == "" {
			continue
		}
		r := it.rect()
		if r.Bottom <= pos || r.Top >= bottom {
			continue
		}
		out = append(out, it.messageID)
	}
	return out
}

func (s *MessageScroller) currentAnchorID() string {
	if s == nil || s.scroll == nil {
		return ""
	}
	limit := s.scroll.GetViewPosV() + s.scrollMargin + s.peek
	id := ""
	for _, it := range s.items {
		if !it.anchor || it.messageID == "" {
			continue
		}
		if it.rect().Top <= limit {
			id = it.messageID
		}
	}
	return id
}

func (s *MessageScroller) emitVisibility() {
	if s.onVisibility == nil || s.visPending || s.destroyed {
		return
	}
	// 滚动事件很密, 可见集按消息条数线性量测; 合并到下一轮消息循环再算。
	s.visPending = true
	app.AfterFunc(0, func() {
		s.visPending = false
		if s.destroyed || s.onVisibility == nil {
			return
		}
		ids := s.visibleIDs()
		anchor := s.currentAnchorID()
		joined := ""
		for _, id := range ids {
			joined += id + "\x00"
		}
		if anchor == s.lastAnchor && joined == s.lastVisible {
			return
		}
		s.lastAnchor, s.lastVisible = anchor, joined
		s.onVisibility(anchor, ids)
	})
}

// ===== 内部: 内容变更 =====

// recordMutation 合并同一轮消息循环里的多次增删, 在布局落定后再决定落点。
func (s *MessageScroller) recordMutation(prepend bool) {
	if s == nil {
		return
	}
	if prepend {
		s.mutPrepend = true
	}
	if s.mutPending {
		return
	}
	s.mutPending = true
	app.AfterFunc(0, func() {
		if s.destroyed {
			return
		}
		s.mutPending = false
		s.applyMutation()
	})
}

// scheduleSync 根元素尺寸变化: 首次上屏时补落默认位置, 之后按"跟随 / 重锚"处理。
func (s *MessageScroller) scheduleSync() {
	if s == nil || s.syncQueued {
		return
	}
	s.syncQueued = true
	app.AfterFunc(0, func() {
		if s.destroyed {
			return
		}
		s.syncQueued = false
		if s.mutPending {
			return
		}
		if !s.defaultApplied || s.defaultPending {
			if s.scroll.GetViewHeight() > 0 {
				s.applyDefault()
			}
			return
		}
		s.afterResize()
	})
}

func (s *MessageScroller) scheduleResize() {
	if s == nil || s.resizePending {
		return
	}
	s.resizePending = true
	app.AfterFunc(0, func() {
		if s.destroyed {
			return
		}
		s.resizePending = false
		if s.mutPending {
			return
		}
		s.afterResize()
	})
}

// applyMutation 一次增删结束后的落点决策(对应上游 handleContentChange)。
func (s *MessageScroller) applyMutation() {
	if s == nil || s.scroll == nil {
		return
	}
	s.relayout()

	if s.scroll.GetViewHeight() <= 0 {
		// 容器还没上屏(隐藏页), 等尺寸事件再落默认位置。
		if !s.defaultApplied {
			s.defaultPending = true
		}
		return
	}
	if !s.defaultApplied {
		if s.pendingMessageID != "" {
			if it := s.byID[s.pendingMessageID]; it != nil {
				align := s.pendingAlign
				s.pendingMessageID = ""
				s.scrollToItem(it, align, s.scrollMargin, false, false)
				s.defaultApplied = true
				s.defaultPending = false
				return
			}
		}
		s.applyDefault()
		return
	}
	if s.pendingMessageID != "" {
		if it := s.byID[s.pendingMessageID]; it != nil {
			align := s.pendingAlign
			s.pendingMessageID = ""
			s.scrollToItem(it, align, s.scrollMargin, false, false)
			return
		}
	}

	// 前置历史: 把"原来的第一条"滚回原来的视口位置。
	if s.mutPrepend && s.mutFirstH != 0 {
		if h := s.mutFirstH; h != 0 {
			for _, it := range s.items {
				if it.Handle == h {
					if r := it.rect(); r.Top != s.mutFirstTop {
						s.scrollToSilent(s.mutScrollPos + (r.Top - s.mutFirstTop))
					}
					break
				}
			}
		}
		s.mutPrepend = false
		s.mutFirstH = 0
		s.mutFirstTop, s.mutScrollPos = 0, 0
		s.markAllHandled()
		s.syncState()
		return
	}

	// 追加: 新轮次的第一个锚点顶到视口顶部(跟随时改为贴底)。
	s.applyAppend()
}

func (s *MessageScroller) applyAppend() {
	if s == nil {
		return
	}
	end := s.atEnd()
	if s.following && s.autoScroll {
		if s.countNewAnchors() <= 1 {
			// 单轮次开始: 即使贴底也把这一轮顶到顶部, 让回复在下方流式长出来。
			if it := s.firstNewAnchor(); it != nil {
				s.scrollToItem(it, MessageScrollerAlignStart, s.scrollMargin, true, false)
				return
			}
		}
		s.scrollToEnd(false)
		return
	}
	if it := s.firstNewAnchor(); it != nil {
		s.anchoredID = it.messageID
		s.scrollToItem(it, MessageScrollerAlignStart, s.scrollMargin, true, false)
		return
	}
	if end && s.autoScroll {
		s.following = true
	}
	s.syncState()
}

// 下面两个辅助靠"上次已处理到的条数"来识别新增项; 增删都较稀疏, 直接扫一遍即可。

func (s *MessageScroller) firstNewAnchor() *MessageScrollerItem {
	for _, it := range s.items {
		if it.anchor && !it.handled {
			return it
		}
	}
	return nil
}

// countNewAnchors 统计从上次处理位置起新增的锚点数(上游用"追加了多个轮次"才贴底)。
func (s *MessageScroller) countNewAnchors() int {
	n := 0
	for _, it := range s.items {
		if it.anchor && !it.handled {
			n++
		}
	}
	return n
}

func (s *MessageScroller) markHandledThrough(it *MessageScrollerItem) {
	for _, x := range s.items {
		if x.anchor {
			x.handled = true
		}
		if x == it {
			return
		}
	}
}

func (s *MessageScroller) markAllHandled() {
	for _, it := range s.items {
		it.handled = true
	}
}

// afterResize 内容自身高度变化(流式文本 / 折叠展开)但不增删条目时的处理。
func (s *MessageScroller) afterResize() {
	if s == nil || s.scroll == nil {
		return
	}
	if s.defaultPending || !s.defaultApplied {
		return
	}
	if s.following && s.autoScroll {
		s.scrollToEnd(false)
		return
	}
	if s.anchoredID != "" {
		if it := s.byID[s.anchoredID]; it != nil {
			s.scrollToItem(it, MessageScrollerAlignStart, s.scrollMargin, true, false)
			return
		}
		s.anchoredID = ""
	}
	s.syncState()
}

// applyDefault 首次内容落点: end / start / last-anchor。
func (s *MessageScroller) applyDefault() {
	if s == nil || s.scroll == nil || s.scroll.GetViewHeight() <= 0 {
		s.defaultPending = true
		return
	}
	s.defaultApplied = true
	s.defaultPending = false
	s.markAllHandled()
	switch s.defaultPos {
	case MessageScrollerScrollStart:
		s.scrollToStart(false)
	case MessageScrollerScrollLastAnchor:
		var last *MessageScrollerItem
		for _, it := range s.items {
			if it.anchor {
				last = it
			}
		}
		if last == nil {
			s.scrollToEnd(false)
			return
		}
		total, viewH := s.metrics()
		if total-last.rect().Top <= viewH {
			s.scrollToEnd(false)
			return
		}
		s.scrollToItem(last, MessageScrollerAlignStart, s.scrollMargin, true, false)
	default:
		s.scrollToEnd(false)
	}
}

// ===== 内部: 滚动 =====

func (s *MessageScroller) scrollToSilent(pos int32) {
	if s == nil || s.scroll == nil {
		return
	}
	if pos < 0 {
		pos = 0
	}
	if max := s.maxScroll(); pos > max {
		pos = max
	}
	s.selfScroll = true
	s.scroll.ScrollPosYV(pos)
	s.selfScroll = false
}

// scrollTo 程序滚动(animate = 平滑, 用于浮动按钮点击)。
func (s *MessageScroller) scrollTo(pos int32, animate bool) {
	if s == nil || s.scroll == nil {
		return
	}
	if pos < 0 {
		pos = 0
	}
	if max := s.maxScroll(); pos > max {
		pos = max
	}
	start := s.scroll.GetViewPosV()
	if !animate || start == pos {
		s.cancelScrollAnim()
		s.scrollToSilent(pos)
		s.syncState()
		return
	}
	// 每次点击都要一段新过渡: PresenceOnce 到位即回收, 不会像同键 Presence 那样常驻表里。
	s.animGen++
	gen := s.animGen
	s.animating = true
	xui.PresenceOnce(s.Handle, true, messageScrollerScrollD, func(v float64) {
		if s.destroyed || s.animGen != gen {
			return
		}
		p := start + int32(float64(pos-start)*v)
		s.scrollToSilent(p)
	}, func(float64) {
		if s.destroyed || s.animGen != gen {
			return
		}
		s.animating = false
		s.scrollToSilent(pos)
		s.syncState()
	})
}

// cancelScrollAnim 让正在跑的平滑滚动失效(用户滚轮 / 拖滚动条时立即接手)。
func (s *MessageScroller) cancelScrollAnim() {
	if s == nil || !s.animating {
		return
	}
	s.animating = false
	s.animGen++
}

func (s *MessageScroller) scrollToEnd(animate bool) bool {
	if s == nil || s.scroll == nil {
		return false
	}
	s.setSpacer(0)
	s.relayout()
	s.anchoredID = ""
	s.markAllHandled()
	if s.autoScroll {
		s.following = true
	} else {
		s.following = false
	}
	s.scrollTo(s.maxScroll(), animate)
	return true
}

func (s *MessageScroller) scrollToStart(animate bool) bool {
	if s == nil || s.scroll == nil {
		return false
	}
	s.setSpacer(0)
	s.relayout()
	s.anchoredID = ""
	s.following = false
	s.markAllHandled()
	s.scrollTo(0, animate)
	return true
}

// scrollToItem 把条目按 align 放到视口里。keepPeek 时上方保留上一条的 peek 像素,
// 并把该条目记为"流式锚点"(内容继续长高时保持它不动)。
func (s *MessageScroller) scrollToItem(it *MessageScrollerItem, align MessageScrollerAlign,
	margin int32, keepPeek, animate bool) bool {
	if s == nil || s.scroll == nil || it == nil || it.Handle == 0 {
		return false
	}
	viewH := s.scroll.GetViewHeight()
	if viewH <= 0 {
		return false
	}
	if keepPeek {
		margin += s.peek
	}

	// 先清 spacer, 量到"自然内容"的滚动范围。
	s.setSpacer(0)
	s.relayout()
	total, _ := s.metrics()

	r := it.rect()
	itemH := r.Bottom - r.Top
	pos := s.scroll.GetViewPosV()
	var target int32
	switch align {
	case MessageScrollerAlignCenter:
		target = r.Top - (viewH-itemH)/2 - margin
	case MessageScrollerAlignEnd:
		target = r.Top - viewH + itemH + margin
	case MessageScrollerAlignNearest:
		top := r.Top - margin
		bottom := r.Bottom + margin
		if top >= pos && bottom <= pos+viewH {
			target = pos
		} else if top < pos {
			target = top
		} else {
			target = bottom - viewH
		}
	default:
		target = r.Top - margin
	}
	if target < 0 {
		target = 0
	}

	// 目标在自然内容末端之外(最后一条轮次被顶到顶部): 补一段尾部空白让它可达。
	maxNatural := total - viewH
	if maxNatural < 0 {
		maxNatural = 0
	}
	if over := target - maxNatural; over > 0 {
		spacer := over
		s.setSpacer(spacer)
		s.relayout()
		// spacer 显示后会额外带上一段内容间距, 按实测总高把差值补回去。
		total2, _ := s.metrics()
		if d := target + viewH - total2; d != 0 {
			spacer += d
			if spacer < 1 {
				spacer = 1
			}
			s.setSpacer(spacer)
			s.relayout()
		}
	}

	if keepPeek {
		s.anchoredID = it.messageID
		s.following = false
	} else {
		s.anchoredID = ""
		s.following = false
	}
	s.markHandledThrough(it)
	s.scrollTo(target, animate)
	return true
}

// setSpacer 调整尾部空白高度; 0 时隐藏, 不占内容区的间距。
func (s *MessageScroller) setSpacer(h int32) {
	if s == nil || s.spacer == nil {
		return
	}
	if h < 0 {
		h = 0
	}
	if h == 0 {
		s.spacer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 0)
		s.spacer.Show(false)
		return
	}
	s.spacer.Show(true)
	s.spacer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, h)
}
