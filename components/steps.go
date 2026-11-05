package ui

import (
	"math"
	"strconv"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 步骤条几何常量(逻辑像素)。
//
// 没有 shadcn 官方 stepper, 语义按社区通行做法定死(见 DESIGN 第四节末):
//   - 横向: 每步等宽, 指示圆点居中, 标题/说明在圆点下方居中; 圆点之间画连接线;
//   - 竖向: 指示圆点在左、文字在右, 圆点之间画竖线; 行高按是否有说明固定;
//   - 四态: 已完成(主色实心 + 对勾) / 进行中(主色描边环 + 主色编号) / 未达(边框色环 + 弱化编号)
//     / 出错(破坏色实心 + X, 指向它的连接线也转破坏色);
//     步骤带 Icon 时圆点里改画图标, 代替序号/对勾/X, 状态仍由颜色区分;
//   - 默认只有已完成步可点回, 当前与未达步不响应鼠标(避免"点了没反应"的误导);
//     可点全部 / 单步禁用 / 进行中进度 / 尺寸档见 StepsOption 与 StepItem。
const (
	stepsCircle       int32 = 24               // 指示圆点直径(size-6)
	stepsBorder       int32 = 2                // 圆点环宽(border-2)
	stepsIcon         int32 = 12               // 对勾图标尺寸
	stepsItemIconSize int32 = 14               // 自定义图标尺寸(比对勾大一点, 复杂轮廓才不糊)
	stepsHGap         int32 = 8                // 圆点与标题的间距
	stepsTitleH       int32 = 20               // text-sm 行高
	stepsDescH        int32 = 16               // text-xs 行高
	stepsDescGap      int32 = 2                // 标题与说明的间距
	stepsRowGap       int32 = 12               // 竖向相邻行之间的连接线长度
	stepsTextPadX     int32 = 4                // 横向每列文字左右的呼吸量
	stepsTextLeft     int32 = stepsCircle + 12 // 竖向文字起点(圆点 + gap-3)
	stepsFocusPad     int32 = 3                // 焦点环外扩量: 元素四周留出的呼吸量

	// 小号(sm): 圆点 20、标题 18/说明 14, 字重与字号同步缩一档。
	stepsSmallCircle       int32 = 20
	stepsSmallIcon         int32 = 10
	stepsSmallItemIconSize int32 = 12
	stepsSmallHGap         int32 = 6
	stepsSmallTitleH       int32 = 18
	stepsSmallDescH        int32 = 14
	stepsSmallTextPadX     int32 = 3
	stepsSmallTextLeft     int32 = stepsSmallCircle + 10
	stepsSmallRowGap       int32 = 10
)

// StepStatus 是单个步骤的显示状态(对应 Ant Design 的 wait / process / finish / error)。
type StepStatus int

const (
	StepStatusAuto    StepStatus = iota // 未指定: 按 current 自动推导(< 当前=已完成, = 当前=进行中, > 当前=未达)
	StepStatusWait                      // 未达: 边框环 + 弱化编号
	StepStatusProcess                   // 进行中: 主色环 + 主色编号
	StepStatusFinish                    // 已完成: 主色实心 + 对勾
	StepStatusError                     // 出错: 破坏色实心 + X, 指向它的连接线也转破坏色
)

// StepsSize 是步骤条的尺寸档。
type StepsSize int

const (
	StepsMedium StepsSize = iota // 普通(默认): 圆点 24、标题 20/说明 16
	StepsSmall                   // 小号: 圆点 20、标题 18/说明 14, 字号各缩一档
)

// stepsMetrics 是单套尺寸下的几何与字号增量(供布局、绘制、测试共用)。
type stepsMetrics struct {
	circle     int32
	border     int32
	icon       int32
	itemIcon   int32
	hGap       int32
	titleH     int32
	descH      int32
	descGap    int32
	rowGap     int32
	textPadX   int32
	textLeft   int32
	focusPad   int32
	titleDelta int32 // 标题字号相对主题默认值的增量
	smallDelta int32 // 说明/编号字号的增量
}

func stepsMetricsFor(size StepsSize) stepsMetrics {
	if size == StepsSmall {
		return stepsMetrics{
			circle: stepsSmallCircle, border: stepsBorder, icon: stepsSmallIcon,
			itemIcon: stepsSmallItemIconSize, hGap: stepsSmallHGap,
			titleH: stepsSmallTitleH, descH: stepsSmallDescH, descGap: stepsDescGap,
			rowGap: stepsSmallRowGap, textPadX: stepsSmallTextPadX,
			textLeft: stepsSmallTextLeft, focusPad: stepsFocusPad,
			titleDelta: -1, smallDelta: -3,
		}
	}
	return stepsMetrics{
		circle: stepsCircle, border: stepsBorder, icon: stepsIcon,
		itemIcon: stepsItemIconSize, hGap: stepsHGap,
		titleH: stepsTitleH, descH: stepsDescH, descGap: stepsDescGap,
		rowGap: stepsRowGap, textPadX: stepsTextPadX,
		textLeft: stepsTextLeft, focusPad: stepsFocusPad,
		titleDelta: 0, smallDelta: -2,
	}
}

// StepItem 是单个步骤的内容。
type StepItem struct {
	Title       string // 步骤标题
	Description string // 步骤说明(可空; 任一非空时横向/竖向统一预留说明行)
	// Icon 是圆点内的自定义图标(见 xui.Painter.Icon: icons.IconXxx 常量、path 数据或整段 SVG)。
	// 非空时**四态都画它**, 代替序号、对勾与出错 X; 状态仍由圆点颜色
	// (主色实心 / 主色环 / 边框环 / 破坏色实心)与连接线表达 —— 同 Ant Design / Element Plus 的 step icon 口径。
	Icon string
	// Status 覆盖按 current 推导的自动状态(见 StepStatus); 零值 = 自动。
	// 典型用法: 当前步校验失败时设 StepStatusError(红点 + X, 上一段连接线变红)。
	Status StepStatus
	// Disabled 禁用点击(仍显示, 也可用方向键浏览); 已完成步设 true 后不能再点回。
	Disabled bool
	// Percent 进行中步骤的进度(1..100; 0 = 不画进度环)。
	// 只对 Status == StepStatusProcess(含自动推导的当前步)生效, 用主色弧覆盖在边框色圆环上。
	Percent int32
}

// StepsOrientation 是步骤条的取向。
type StepsOrientation int

const (
	StepsHorizontal StepsOrientation = iota // 横向: 圆点在上、文字在下
	StepsVertical                           // 竖向: 圆点在左、文字在右
)

// Steps 步骤条(纯自绘: 一个元素承载全部步骤, 鼠标/键盘驱动)。
//
// 与社区 stepper 的通行语义对齐:
//   - 状态: 已完成/进行中/未达按 current 推导, StepItem.Status 可单步覆盖;
//     出错步 = 破坏色实心 + X, 指向它的连接线也转破坏色;
//   - 横向等宽列、圆点居中, 竖向固定行高、圆点在左; 已完成段落的连接线取主色;
//   - 默认点已完成步回退(SetCurrent + OnChange), 当前/未达步不响应鼠标;
//     WithStepsClickable 放开任意非禁用、非当前步, StepItem.Disabled 单步禁用;
//   - StepItem.Percent(1..100) 在进行中步骤的圆点上画主色进度弧;
//   - 尺寸档 StepsMedium/StepsSmall, 编号可用 WithStepsInitial 整体偏移;
//   - focus-visible: 整块是一个 Tab 站点, ←/→/↑/↓ 在**所有步骤**间移动焦点
//     (含未达步, 只读浏览), Home/End 首末, Enter/Space 激活可点的聚焦步;
//     按任意键即切到键盘焦点态并绘制焦点环 —— 鼠标点过后直接按方向键也能看到
//     焦点在动, 不会出现"按了没反应"。
//
// 为什么整块自绘: 步骤数、两形态、连接线跨列都要参与布局, 拼子元素就得为每种取向
// 各写一套排版与重排(同 pagination.go 的路线)。
type Steps struct {
	*widget.Element

	items     []StepItem
	current   int
	vertical  bool
	size      StepsSize
	clickable bool       // 任意非禁用步都可点(默认只允许点回已完成步)
	initial   int        // 显示编号的起始值(仍按 0 起访问)
	curStatus StepStatus // 当前步的默认状态(默认 Process; 单步 Status 优先)
	backdrop  func(t *xui.Theme) xui.Color

	hover   int // 鼠标悬停的步骤下标(-1 = 无)
	pressed int // 按下的步骤下标(-1 = 无)
	focus   int // 键盘焦点所在步骤下标(-1 = 还没进过聚焦态)
	boxes   []stepBox

	keyID int
	cb    func(index int)
}

// StepsOption 步骤条配置项。
type StepsOption func(*Steps)

// WithStepsOrientation 设置取向(默认横向)。
func WithStepsOrientation(o StepsOrientation) StepsOption {
	return func(s *Steps) { s.vertical = o == StepsVertical }
}

// WithStepsCurrent 设置初始当前步(0 起; 越界夹紧)。
func WithStepsCurrent(i int) StepsOption {
	return func(s *Steps) { s.current = i }
}

// WithStepsStatus 指定当前步的默认状态(默认 Process; 传 Error 可整条流程标红)。
// 单步的 StepItem.Status 优先级更高。
func WithStepsStatus(st StepStatus) StepsOption {
	return func(s *Steps) {
		if st != StepStatusAuto {
			s.curStatus = st
		}
	}
}

// WithStepsSize 设置尺寸档(默认 StepsMedium)。
func WithStepsSize(size StepsSize) StepsOption {
	return func(s *Steps) {
		if size == StepsMedium || size == StepsSmall {
			s.size = size
		}
	}
}

// WithStepsInitial 设置显示编号的起始值(默认 0 起, 即第一步显示 1)。
// 只影响编号文本, 不影响索引/点击语义; 用于跨页向导从第 N 步接着编号。
func WithStepsInitial(n int) StepsOption {
	return func(s *Steps) {
		if n >= 0 {
			s.initial = n
		}
	}
}

// WithStepsClickable 允许点击任意非禁用、非当前的步骤(默认只能点回已完成步)。
// 配合 OnChange 可把步骤条当成向导页签; 默认行为不变, 避免误点前进造成流程跳步。
func WithStepsClickable() StepsOption {
	return func(s *Steps) { s.clickable = true }
}

// WithStepsBackdrop 指定铺在元素底下的宿主底色(默认为 t.Card)。
// 步骤条只画局部形状(悬停块/圆点/连接线), 每帧先用宿主底色擦掉上一帧;
// 放进非 Card 底色的容器时用它跟随宿主(同 Checkbox.SetBackdrop)。
func WithStepsBackdrop(fn func(t *xui.Theme) xui.Color) StepsOption {
	return func(s *Steps) { s.backdrop = fn }
}

// NewSteps 创建步骤条。cy 传 0 时按内容与尺寸档自动取高(含焦点环呼吸量):
// 横向 = 圆点 + 间距 + 标题(+ 说明), 竖向 = 行高 × 步数 + 行距 × (步数-1)。
func NewSteps(x, y, cx, cy int32, hParent int, items []StepItem, opts ...StepsOption) *Steps {
	s := &Steps{
		items: append([]StepItem(nil), items...),
		hover: -1, pressed: -1, focus: -1, curStatus: StepStatusProcess,
	}
	for _, o := range opts {
		o(s)
	}
	s.clampCurrent()
	if cx <= 0 {
		cx = 360
	}
	if cy <= 0 {
		cy = stepsNaturalHeight(s.items, s.vertical, stepsMetricsFor(s.size))
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	s.Element = ele
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	// 整块是一个 Tab 站点(同分页): 步骤级各占一个 Tab 站点需要每步一个元素,
	// 与单元素自绘冲突 —— 进入后用方向键在可达步骤间移动。
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(true)
	xc.XEle_EnableDrawFocus(ele.Handle, false)

	ele.AddEvent_Paint(s.onPaint)
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		s.setHover(s.hit(pPt.X, pPt.Y))
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		s.pressed = -1
		s.setHover(-1)
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标操作标记焦点来源, 保持 focus-visible 语义
		s.pressed = s.hit(pPt.X, pPt.Y)
		s.setHover(s.pressed)
		ele.Redraw(true)
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		idx := s.pressed
		s.pressed = -1
		if idx >= 0 && idx == s.hit(pPt.X, pPt.Y) {
			s.focus = idx // 点过的步骤保持键盘焦点, 方向键从这里接着走
			s.activate(idx)
		}
		// 延迟重绘: 调用方通常在 OnChange 里同步自己的说明文字, 立即重绘会把
		// 窗口待重绘区域一起清掉(同 pagination 的踩坑记录)。
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		if s.focus < 0 || s.focus >= len(s.items) {
			s.focus = s.current
		}
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	s.installKeys()
	return s
}

// SetBackdrop 指定宿主底色(默认按"放在 Card 上"用 t.Card)。
func (s *Steps) SetBackdrop(fn func(t *xui.Theme) xui.Color) *Steps {
	if s == nil {
		return s
	}
	s.backdrop = fn
	s.Redraw(false)
	return s
}

// Current 返回当前步下标(0 起)。
func (s *Steps) Current() int { return s.current }

// SetCurrent 设置当前步(越界夹紧)。**不触发** OnChange —— 供调用方把状态
// 同步到自己的向导流程(OnChange 只响应用户点击/键盘)。
func (s *Steps) SetCurrent(i int) *Steps {
	if s == nil {
		return s
	}
	s.setCurrent(i, false)
	return s
}

// OnChange 注册用户回退到某步的回调(index 为 0 起的新当前步)。
func (s *Steps) OnChange(fn func(index int)) *Steps {
	if s == nil {
		return s
	}
	s.cb = fn
	return s
}

// SetItemPercent 更新第 i 步的进行中进度(1..100; 0 = 关掉进度环; 越界忽略)。
// 只对进行中步骤可见; 供上传/安装这类流程在回调里推进进度。
func (s *Steps) SetItemPercent(i int, pct int32) *Steps {
	if s == nil || i < 0 || i >= len(s.items) {
		return s
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if s.items[i].Percent == pct {
		return s
	}
	s.items[i].Percent = pct
	s.Redraw(false)
	return s
}

// setCurrent 切当前步; notify=true 时触发 OnChange。
func (s *Steps) setCurrent(i int, notify bool) {
	if len(s.items) == 0 {
		s.current = 0
		return
	}
	if i < 0 {
		i = 0
	} else if i >= len(s.items) {
		i = len(s.items) - 1
	}
	if i == s.current {
		return
	}
	s.current = i
	if s.focus > i {
		s.focus = i
	}
	s.Element.Redraw(false)
	if notify && s.cb != nil {
		s.cb(i)
	}
}

// statusAt 返回第 i 步的实际显示状态: 单步 Status > 当前步默认状态 > 按 current 推导。
func (s *Steps) statusAt(i int) StepStatus {
	if i < 0 || i >= len(s.items) {
		return StepStatusWait
	}
	if st := s.items[i].Status; st != StepStatusAuto {
		return st
	}
	switch {
	case i < s.current:
		return StepStatusFinish
	case i > s.current:
		return StepStatusWait
	default:
		if s.curStatus == StepStatusAuto {
			return StepStatusProcess
		}
		return s.curStatus
	}
}

// canActivate 报告第 i 步是否可点/可回车激活: 默认只允许点回已完成步;
// WithStepsClickable 后任意非当前、非禁用步都可以。
func (s *Steps) canActivate(i int) bool {
	if i < 0 || i >= len(s.items) || i == s.current || s.items[i].Disabled {
		return false
	}
	return s.clickable || i < s.current
}

func (s *Steps) metrics() stepsMetrics { return stepsMetricsFor(s.size) }

// activate 响应鼠标/键盘: 按 canActivate 的规则切换当前步。
func (s *Steps) activate(i int) {
	if !s.canActivate(i) {
		return
	}
	s.setCurrent(i, true)
}

func (s *Steps) clampCurrent() {
	if len(s.items) == 0 {
		s.current = 0
		return
	}
	if s.current < 0 {
		s.current = 0
	} else if s.current >= len(s.items) {
		s.current = len(s.items) - 1
	}
}

// installKeys 挂键盘导航。必须挂窗口级: 元素开了 EnableSwitchFocus(Tab 才能进入)后,
// xcgui 会把方向键当成"切换焦点"并在元素 KeyDown 之前执行(见 DESIGN 第八节 152)。
func (s *Steps) installKeys() {
	hWindow := xui.HostWindowOf(s.Handle) // 步骤条通常在 ScrollArea 里, 直接映射 HWND 会拿到无效值
	if hWindow == 0 {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	s.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		if xc.XWnd_GetFocusEle(hWin) != s.Handle {
			return 0
		}
		return s.onKeyDown(wParam, pbHandled)
	})
	s.Element.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if s.keyID > 0 {
			if xc.XC_IsHXCGUI(hWindow, xcc.XC_WINDOW) {
				w.RemoveEvent(xcc.WM_KEYDOWN, s.keyID)
			}
			s.keyID = 0
		}
		return 0
	})
}

// onKeyDown 处理键盘: 方向键在所有步骤间移动焦点(未达步也可以浏览), Home/End 首末,
// Enter/Space 激活可点的聚焦步(默认只有已完成步可点)。步骤条是有序的, 到端点后不循环。
func (s *Steps) onKeyDown(wParam uintptr, pbHandled *bool) int {
	// 只要用了键盘, 就把焦点来源切成键盘: 鼠标点过组件再按方向键也会立即画出焦点环。
	// 否则 focus-visible 仍是鼠标来源, 方向键挪了焦点却看不见, 像"按了没反应"。
	xui.NoteKeyboard()
	switch wParam {
	case 0x25, 0x26: // VK_LEFT / VK_UP
		s.focusMove(-1)
	case 0x27, 0x28: // VK_RIGHT / VK_DOWN
		s.focusMove(1)
	case 0x24, 0x23: // VK_HOME / VK_END
		s.focusEdge(wParam == 0x23)
	case 0x0D, 0x20: // VK_RETURN / VK_SPACE
		s.activate(s.focus)
	default:
		return 0
	}
	*pbHandled = true
	return 0
}

// focusMove 把焦点往前后挪一步; 到端点不动。
func (s *Steps) focusMove(step int) {
	if len(s.items) == 0 {
		s.focus = -1
		return
	}
	if s.focus < 0 || s.focus >= len(s.items) {
		s.focus = s.current
		s.Redraw(false)
		return
	}
	next := s.focus + step
	if next < 0 || next >= len(s.items) {
		return
	}
	s.focus = next
	s.Redraw(false)
}

// focusEdge 把焦点挪到第一个/最后一个步骤(Home/End)。
func (s *Steps) focusEdge(last bool) {
	if len(s.items) == 0 {
		s.focus = -1
		return
	}
	if last {
		s.focus = len(s.items) - 1
	} else {
		s.focus = 0
	}
	s.Redraw(false)
}

// setHover 更新悬停步骤(变化才重绘)。
func (s *Steps) setHover(i int) {
	if s.hover == i {
		return
	}
	s.hover = i
	s.Element.Redraw(true)
}

// refreshBoxes 在没有绘制过(刚创建就被点到/Tab 到)时先算一次几何。
func (s *Steps) refreshBoxes() {
	if len(s.boxes) > 0 || len(s.items) == 0 {
		return
	}
	var rc xc.RECT
	xc.XEle_GetClientRect(s.Handle, &rc)
	s.boxes = stepsLayout(rc, s.items, s.vertical, s.metrics())
}

// hit 返回客户区内 (x,y) 命中的**可点击**步骤下标, 否则 -1。
func (s *Steps) hit(x, y int32) int {
	s.refreshBoxes()
	return stepsHitIndex(s.boxes, x, y, s.canActivate)
}

// stepsHitIndex 是命中测试的纯函数版本(单测用): canActivate 决定哪些步骤可点。
func stepsHitIndex(boxes []stepBox, x, y int32, canActivate func(int) bool) int {
	for i := range boxes {
		if canActivate != nil && !canActivate(i) {
			continue
		}
		rc := boxes[i].cell
		if x >= rc.Left && x < rc.Right && y >= rc.Top && y < rc.Bottom {
			return i
		}
	}
	return -1
}

// backdropColor 返回宿主底色(默认 Card)。
func (s *Steps) backdropColor(t *xui.Theme) xui.Color {
	if s.backdrop != nil {
		return s.backdrop(t)
	}
	return t.Card
}

func (s *Steps) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	m := s.metrics()
	bg := s.backdropColor(t)

	// 先铺宿主底色: 本组件只画局部形状, 不擦底的话上一帧的悬停块/连接线会残留
	// (透明元素重绘不擦底, 见 DESIGN 第八节 28)。
	p.FillRound(&p.Rc, bg, 0)

	s.boxes = stepsLayout(p.Rc, s.items, s.vertical, m)
	if len(s.boxes) == 0 {
		return 0
	}

	titleFont := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family, Size: t.FontDefault.Size + m.titleDelta, Bold: true,
	})
	descFont := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family, Size: t.FontDefault.Size + m.smallDelta,
	})
	numFont := xui.FontHandle(xui.FontSpec{
		Family: t.FontDefault.Family, Size: t.FontDefault.Size + m.smallDelta, Bold: true,
	})
	focusedByKey := xui.StateOf(hEle).Focused && xui.FocusedByKeyboard()

	// 悬停块最先画(在连接线与圆点下面): 只给可激活的步骤反馈, 当前/禁用步
	// 不响应鼠标, 不给"点了没反应"的悬停暗示。
	for i := range s.boxes {
		if !s.canActivate(i) {
			continue
		}
		hv := xui.HoverItem(hEle, int32(i), i == s.hover)
		if bg := xui.Fade(xui.None, t.Muted, hv); !bg.IsNone() {
			cell := s.boxes[i].cell
			if !s.vertical {
				cell.Left += m.textPadX
				cell.Right -= m.textPadX
			}
			p.FillRound(&cell, bg, t.RadiusLg())
		}
	}

	// 连接线画在圆点之前(圆点盖住线头), 但要在悬停块之后(否则被悬停底色盖掉)。
	// 语义: 下一步出错时整段转破坏色; 已完成段取主色; 其余取边框色。
	// 进度只画在圆点上(见下方 Percent 弧), 连接线不再重复表达进度。
	for i := 0; i+1 < len(s.boxes); i++ {
		a, b := s.boxes[i].circle, s.boxes[i+1].circle
		color := t.Border
		switch s.statusAt(i + 1) {
		case StepStatusError:
			color = t.Destructive
		case StepStatusFinish:
			color = t.Primary
		default:
			if s.statusAt(i) == StepStatusFinish {
				color = t.Primary
			}
		}
		var x1, y1, x2, y2 int32
		if s.vertical {
			x1 = (a.Left + a.Right) / 2
			x2 = x1
			y1, y2 = a.Bottom, b.Top
			if y2 <= y1 {
				continue
			}
		} else {
			y1 = (a.Top + a.Bottom) / 2
			y2 = y1
			x1, x2 = a.Right, b.Left
			if x2 <= x1 {
				continue
			}
		}
		p.Line(x1, y1, x2, y2, color, 2)
	}

	for i := range s.boxes {
		b := s.boxes[i]
		st := s.statusAt(i)

		// focus-visible: 只在键盘获得的焦点上画环(鼠标点击不画, 同 shadcn 的 :focus-visible)。
		// 环画在圆点外侧(元素四周已留 focusPad), 不能画在内侧 —— 实心圆会盖掉内圈。
		if focusedByKey && i == s.focus {
			stepsFill(p, stepsInset(b.circle, -m.focusPad), t.Ring.WithAlpha(128))
		}

		icon := s.items[i].Icon
		switch st {
		case StepStatusFinish:
			stepsFill(p, b.circle, t.Primary)
			if icon != "" {
				stepsItemIcon(p, b.circle, icon, t.PrimaryForeground, m.itemIcon)
			} else {
				check := stepsInset(b.circle, (m.circle-m.icon)/2)
				p.IconRotatedOffset(icons.IconCheck, &check, t.PrimaryForeground, 2, 0, 0, 0)
			}
		case StepStatusProcess:
			if pct := s.items[i].Percent; pct > 0 {
				if pct > 100 {
					pct = 100
				}
				stepsRing(p, b.circle, t.Border, bg, m.border)
				stepsArc(p, s.Handle,
					(b.circle.Left+b.circle.Right)/2, (b.circle.Top+b.circle.Bottom)/2,
					m.circle/2, m.circle/2-m.border,
					0, 360*float64(pct)/100, t.Primary)
			} else {
				stepsRing(p, b.circle, t.Primary, bg, m.border)
			}
			if icon != "" {
				stepsItemIcon(p, b.circle, icon, t.Primary, m.itemIcon)
			} else {
				stepsNumber(p, b.circle, strconv.Itoa(s.initial+i+1), numFont, t.Primary)
			}
		case StepStatusError:
			stepsFill(p, b.circle, t.Destructive)
			if icon != "" {
				stepsItemIcon(p, b.circle, icon, t.DestructiveForeground, m.itemIcon)
			} else {
				cross := stepsInset(b.circle, (m.circle-m.icon)/2)
				p.IconRotatedOffset(icons.IconX, &cross, t.DestructiveForeground, 2, 0, 0, 0)
			}
		default: // StepStatusWait
			stepsRing(p, b.circle, t.Border, bg, m.border)
			if icon != "" {
				stepsItemIcon(p, b.circle, icon, t.MutedForeground, m.itemIcon)
			} else {
				stepsNumber(p, b.circle, strconv.Itoa(s.initial+i+1), numFont, t.MutedForeground)
			}
		}

		titleColor := t.Foreground
		descColor := t.MutedForeground
		switch st {
		case StepStatusWait:
			titleColor = t.MutedForeground
		case StepStatusError:
			titleColor = t.Destructive
			descColor = t.Destructive
		}
		align := xcc.TextAlignFlag_Center
		if s.vertical {
			align = xcc.TextAlignFlag_Left
		}
		if titleFont != 0 {
			p.Draw.SetFont(titleFont)
		}
		stepsText(p, s.items[i].Title, b.title, titleColor, align|xcc.TextAlignFlag_Vcenter)
		if s.items[i].Description != "" {
			if descFont != 0 {
				p.Draw.SetFont(descFont)
			}
			stepsText(p, s.items[i].Description, b.desc, descColor, align|xcc.TextAlignFlag_Vcenter)
		}
	}
	return 0
}

// stepsArc 在圆心 (cx,cy) 的 [rInner, rOuter] 扇环上画一段圆弧。
// 角度制: 12 点方向为 0、顺时针为正; 用扇环多边形而不是描边, 避免椭圆描边的半像素毛边。
// 填充必须走 D2D 直通(fillPolygonSmooth): FillPolygonF 在 D2D 下没有抗锯齿,
// 小圆点上的弧会是一圈硬台阶(见 DESIGN 第八节 160)。
func stepsArc(p *xui.Painter, hEle int, cx, cy, rOuter, rInner int32, from, to float64, color xui.Color) {
	span := to - from
	if span <= 0 || rOuter <= rInner || color.IsNone() {
		return
	}
	steps := int(math.Ceil(span * 2))
	if steps < 2 {
		steps = 2
	}
	pt := func(r int32, a float64) xc.POINTF {
		rad := a * math.Pi / 180
		return xc.POINTF{
			X: float32(cx) + float32(float64(r)*math.Sin(rad)),
			Y: float32(cy) - float32(float64(r)*math.Cos(rad)),
		}
	}
	pts := make([]xc.POINTF, 0, steps*2+2)
	for i := 0; i <= steps; i++ {
		pts = append(pts, pt(rOuter, from+span*float64(i)/float64(steps)))
	}
	for i := steps; i >= 0; i-- {
		pts = append(pts, pt(rInner, from+span*float64(i)/float64(steps)))
	}
	if fillPolygonSmooth(p, hEle, color, pts) {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.FillPolygonF(pts)
}

// stepsRowH 返回竖向每一步的行高: 有说明 = 标题相对圆点下移 (圆点-标题)/2 +
// 标题行 + 间距 + 说明行; 没说明 = 圆点高(圆点与标题行中心对齐)。
func stepsRowH(items []StepItem, m stepsMetrics) int32 {
	if stepsHasDesc(items) {
		offset := (m.circle - m.titleH) / 2
		if offset < 0 {
			offset = 0
		}
		return offset + m.titleH + m.descGap + m.descH
	}
	return m.circle
}

// stepsNaturalHeight 返回 cy 传 0 时元素应取的高度。
func stepsNaturalHeight(items []StepItem, vertical bool, m stepsMetrics) int32 {
	n := len(items)
	if n <= 0 {
		return m.circle
	}
	if vertical {
		return 2*m.focusPad + int32(n)*stepsRowH(items, m) + int32(n-1)*m.rowGap
	}
	h := m.circle + m.hGap + m.titleH
	if stepsHasDesc(items) {
		h += m.descGap + m.descH
	}
	return h + 2*m.focusPad
}

// stepsHasDesc 报告是否有任一步带说明(有则横/竖统一预留说明行, 避免各行高度参差)。
func stepsHasDesc(items []StepItem) bool {
	for _, it := range items {
		if it.Description != "" {
			return true
		}
	}
	return false
}

// stepsLayout 计算每一步的矩形(绘制、命中测试、单测共用)。
//
// 横向: 列宽按累计比例切分(最后列贴右缘, 避免整除余数累积); 圆点在各列居中,
// 标题/说明在圆点下方居中。竖向: 圆点在左、文字在右; 元素高于自然高度时把
// 多出的空间均摊到行距(连接线变长), 而不是把内容整体居中或留一大块空白。
func stepsLayout(origin xc.RECT, items []StepItem, vertical bool, m stepsMetrics) []stepBox {
	n := len(items)
	if n <= 0 {
		return nil
	}
	hasDesc := stepsHasDesc(items)
	w := origin.Right - origin.Left
	h := origin.Bottom - origin.Top
	out := make([]stepBox, n)

	if vertical {
		rowH := stepsRowH(items, m)
		gap := m.rowGap
		natural := 2*m.focusPad + int32(n)*rowH + int32(n-1)*gap
		if n > 1 && h > natural {
			gap += (h - natural) / int32(n-1)
		}
		textLeft := origin.Left + m.focusPad + m.textLeft
		textRight := origin.Right - m.textPadX
		titleOffset := (m.circle - m.titleH) / 2
		if titleOffset < 0 {
			titleOffset = 0
		}
		for i := range items {
			top := origin.Top + m.focusPad + int32(i)*(rowH+gap)
			titleTop := top + titleOffset
			out[i] = stepBox{
				cell:   xc.RECT{Left: origin.Left, Top: top - m.focusPad, Right: origin.Right, Bottom: top + rowH + m.focusPad},
				circle: xc.RECT{Left: origin.Left + m.focusPad, Top: top, Right: origin.Left + m.focusPad + m.circle, Bottom: top + m.circle},
				title:  xc.RECT{Left: textLeft, Top: titleTop, Right: textRight, Bottom: titleTop + m.titleH},
			}
			if hasDesc {
				out[i].desc = xc.RECT{
					Left: textLeft, Top: titleTop + m.titleH + m.descGap,
					Right: textRight, Bottom: titleTop + m.titleH + m.descGap + m.descH,
				}
			} else {
				out[i].desc = xc.RECT{Left: textLeft, Top: top + m.circle, Right: textRight, Bottom: top + m.circle}
			}
		}
		return out
	}

	contentH := m.circle + m.hGap + m.titleH
	if hasDesc {
		contentH += m.descGap + m.descH
	}
	top := origin.Top + m.focusPad
	if centered := origin.Top + (h-contentH)/2; centered > top {
		top = centered
	}
	for i := range items {
		left := origin.Left + int32(int64(w)*int64(i)/int64(n))
		right := origin.Left + int32(int64(w)*int64(i+1)/int64(n))
		cx := (left + right) / 2
		titleTop := top + m.circle + m.hGap
		out[i] = stepBox{
			cell:   xc.RECT{Left: left, Top: top - m.focusPad, Right: right, Bottom: top + contentH + m.focusPad},
			circle: xc.RECT{Left: cx - m.circle/2, Top: top, Right: cx + m.circle/2, Bottom: top + m.circle},
			title: xc.RECT{
				Left: left + m.textPadX, Top: titleTop,
				Right: right - m.textPadX, Bottom: titleTop + m.titleH,
			},
		}
		if hasDesc {
			out[i].desc = xc.RECT{
				Left: left + m.textPadX, Top: titleTop + m.titleH + m.descGap,
				Right: right - m.textPadX, Bottom: top + contentH,
			}
		} else {
			out[i].desc = xc.RECT{
				Left: left + m.textPadX, Top: top + contentH,
				Right: right - m.textPadX, Bottom: top + contentH,
			}
		}
	}
	return out
}

// stepBox 是单步的几何: cell 用于悬停/命中, 其余是绘制矩形。
type stepBox struct {
	cell   xc.RECT
	circle xc.RECT
	title  xc.RECT
	desc   xc.RECT
}

// stepsInset 把矩形四边内缩 d(负值外扩)。
func stepsInset(rc xc.RECT, d int32) xc.RECT {
	return xc.RECT{Left: rc.Left + d, Top: rc.Top + d, Right: rc.Right - d, Bottom: rc.Bottom - d}
}

// stepsFill 填充一个实心圆。
func stepsFill(p *xui.Painter, rc xc.RECT, color xui.Color) {
	if color.IsNone() {
		return
	}
	p.Draw.SetBrushColor(uint32(color))
	p.Draw.FillEllipse(&rc)
}

// stepsRing 用两层实心圆造出"环 + 挖洞": 椭圆描边落在半像素上会毛糙,
// 叠实心圆则天然平滑(同 colorpicker 的圆点/环画法)。
func stepsRing(p *xui.Painter, rc xc.RECT, ring, hole xui.Color, width int32) {
	stepsFill(p, rc, ring)
	if hole.IsNone() {
		return
	}
	inner := stepsInset(rc, width)
	if inner.Right <= inner.Left || inner.Bottom <= inner.Top {
		return
	}
	stepsFill(p, inner, hole)
}

// stepsItemIcon 在圆点内居中绘制自定义图标。
func stepsItemIcon(p *xui.Painter, rc xc.RECT, data string, color xui.Color, size int32) {
	box := stepsInset(rc, (rc.Right-rc.Left-size)/2)
	p.Icon(data, &box, color, 2)
}

// stepsNumber 在圆点内居中绘制编号。
func stepsNumber(p *xui.Painter, rc xc.RECT, text string, font int, color xui.Color) {
	if font == 0 {
		return
	}
	p.Draw.SetFont(font)
	rcF := xc.RECTF{
		Left: float32(rc.Left), Top: float32(rc.Top) - textVDy(text),
		Right: float32(rc.Right), Bottom: float32(rc.Bottom) - textVDy(text),
	}
	p.TextF(text, &rcF, color,
		xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
}

// stepsText 在 rc 内按 align 对齐绘制单行文字(超宽省略号)。
func stepsText(p *xui.Painter, text string, rc xc.RECT, color xui.Color, align xcc.TextFormatFlag_) {
	if text == "" || color.IsNone() {
		return
	}
	rcF := xc.RECTF{
		Left: float32(rc.Left), Top: float32(rc.Top) - textVDy(text),
		Right: float32(rc.Right), Bottom: float32(rc.Bottom) - textVDy(text),
	}
	p.TextF(text, &rcF, color,
		align|xcc.TextFormatFlag_NoWrap|xcc.TextTrimming_EllipsisCharacter)
}
