package ui

import (
	"math"
	"sort"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xui"
)

const (
	// sliderRange 是内部范围: 千分比, 对外 API 是实际值域(默认 0~100, 保留 1 位小数)。
	sliderRange int32 = 1000
	// sliderThumb 是滑块直径, shadcn 为 size-3。
	sliderThumb int32 = 12
	// sliderTrackH 是轨道高度, shadcn 为 h-1。
	sliderTrackH int32 = 4
	// sliderRingPad 是滑块 hover/active/focus-visible 时 ring-3 的宽度。
	sliderRingPad int32 = 3
	// sliderHalo 是滑块中心到 ring 外缘的距离(滑块半径 + ring 宽度)。
	// 元素沿轨道方向的两端各留出这么多: 元素自绘被裁剪在自身客户区内, 滑块位于
	// 轨道两端时需要这段余量 ring 才不会被切掉(见 DESIGN.md 第八节 10、20)。
	sliderHalo int32 = sliderThumb/2 + sliderRingPad // 9
	// sliderVerticalDefault 是垂直滑块的默认轨道长度(radix 的 data-vertical:min-h-40 = 160)。
	sliderVerticalDefault int32 = 160
)

// Slider 纯自绘滑动条(不使用原生 SliderBar)。
//
// 之所以不用原生 SliderBar 做载体: 其滑块是原生 Button, 默认样式在 Stay/Down/
// Focus 态会画出方块背景(BkInfo), ClearBkInfo + EnableBkTransparent +
// EnableDrawBorder/DrawFocus + 空 paint 回调都压不住(实测悬浮出灰方块、
// 拖动后出白方块、点击聚焦后又一种), 且 EnableFocus(false) 会让按钮不再上报
// Stay 状态。改用普通元素 + 自实现交互后, 组件 100% 掌控绘制, 无原生残留。
//
// 与 shadcn slider.tsx(radix-nova) 对齐:
//   - 轨道 h-1 rounded-full bg-muted; 已完成区间 bg-primary;
//   - 滑块 size-3 rounded-full border-ring bg-white;
//   - 滑块 hover / 拖动中 / focus-visible 时显示 ring-3 ring-ring/50;
//   - 整体禁用 opacity-50; 支持点击跳转、拖动、方向键(一个步进、连续时 ±5%);
//     点击时显式 SetFocus —— 普通 Element 点击不会自动获得键盘焦点, 不设焦点方向键收不到;
//   - 值以**数组**表达(radix 的 value: number[]): 一维数组长度 = 滑块数量,
//     两个值 = 范围选择, 三个及以上 = 多滑块; 相邻滑块间距受 minStepsBetweenThumbs 约束;
//   - orientation=vertical / inverted / disabled / onValueCommit 均已支持。
//
// 几何: 与 radix 一致, 滑块**中心**在轨道两端之间移动(0 → 轨道端点),
// 因此滑块会向轨道端点外沿伸出半个身位, ring 再向外扩 3px。
// 由于元素自绘无法越出自身客户区, 元素实际比轨道沿轴方向多出 sliderHalo*2(18),
// 轨道绘制在元素内缩 sliderHalo 的位置——**视觉轨道长度仍等于入参 cx(垂直为 cy)**。
type Slider struct {
	*widget.Element
	values     []float64 // 升序, 长度 = 滑块数量(radix 的 value 数组)
	min        float64   // 值域下限(radix 的 min), 默认 0
	max        float64   // 值域上限(radix 的 max), 默认 100
	step       float64   // 步进(radix 的 step), 0 = 连续取值
	vertical   bool      // orientation=vertical: 值从下往上增长
	inverted   bool      // inverted: 反转取值方向(视觉与方向键一起反转)
	disabled   bool      // data-disabled
	minSteps   int       // minStepsBetweenThumbs: 相邻滑块至少相隔的步数
	active     int       // 当前操作/键盘调整的滑块索引(焦点环画在它身上)
	drag       int       // 拖动中的滑块索引, -1 = 未拖动
	hoverThumb int       // 指针落在哪个滑块上(热区), -1 = 不在任何滑块上
	changes    []func([]float64)
	commits    []func([]float64)
}

// SliderOption 滑动条配置项。
type SliderOption func(*Slider)

// WithSliderRange 设置值域(对应 radix slider 的 min/max), 默认 0~100。
func WithSliderRange(min, max float64) SliderOption {
	return func(s *Slider) {
		if max > min {
			s.min, s.max = min, max
		}
	}
}

// WithSliderStep 设置步进(对应 radix 的 step), 默认 0 = 连续取值。
// 设置后取值吸附到 `min + n*step`, 键盘方向键也按一个步进移动。
func WithSliderStep(step float64) SliderOption {
	return func(s *Slider) {
		if step > 0 {
			s.step = step
		}
	}
}

// WithSliderValues 设置初始值(对应 radix 的 value / defaultValue 数组)。
// 数组长度就是滑块数量: 1 个 = 普通滑动条, 2 个 = 范围选择, 3 个及以上 = 多滑块。
func WithSliderValues(vals ...float64) SliderOption {
	return func(s *Slider) {
		s.values = append([]float64(nil), vals...)
	}
}

// WithSliderRangeValue 设置范围选择的初值(两个滑块, 便捷写法)。
func WithSliderRangeValue(from, to float64) SliderOption {
	return func(s *Slider) {
		s.values = []float64{from, to}
	}
}

// WithSliderVertical 垂直方向(对应 radix 的 orientation="vertical")。
// 此时 NewSlider 的 cy 是**可见轨道长度**, cx 是元素宽(传 0 用 18);
// cy 传 0 用 radix 的 min-h-40(160)。
func WithSliderVertical() SliderOption {
	return func(s *Slider) { s.vertical = true }
}

// WithSliderInverted 反转取值方向(对应 radix 的 inverted): 值越大越靠左/靠下。
func WithSliderInverted() SliderOption {
	return func(s *Slider) { s.inverted = true }
}

// WithSliderDisabled 初始即禁用(不可拖动/聚焦, 整体 opacity-50)。
func WithSliderDisabled() SliderOption {
	return func(s *Slider) { s.disabled = true }
}

// WithSliderMinStepsBetweenThumbs 相邻滑块之间至少相隔 n 个步进
// (对应 radix 的 minStepsBetweenThumbs, 只在多滑块/范围模式有意义)。
// step <= 0(连续)时按值域的千分之一算一格。
func WithSliderMinStepsBetweenThumbs(n int) SliderOption {
	return func(s *Slider) {
		if n > 0 {
			s.minSteps = n
		}
	}
}

// NewSlider 创建滑动条。cx 是**可见轨道长度**(垂直时 cy 才是轨道长度); cy 传 0
// 时使用滑块 + 焦点环所需的最小高度(18)。元素沿轨道方向会比轨道多出 18
// (滑块两端 ring 的绘制余量), 布局里需要精确宽度时按 `轨道 + 18` 给。
func NewSlider(x, y, cx, cy int32, hParent int, opts ...SliderOption) *Slider {
	s := &Slider{max: 100, drag: -1, hoverThumb: -1}
	for _, o := range opts {
		o(s)
	}
	if s.max <= s.min {
		s.min, s.max = 0, 100
	}
	if len(s.values) == 0 {
		s.values = []float64{s.min}
	}
	s.normalize()

	var w, h int32
	if s.vertical {
		w, h = cx, cy
		if w <= 0 {
			w = sliderThumb + sliderRingPad*2
		}
		if h <= 0 {
			h = sliderVerticalDefault
		}
		h += sliderHalo * 2
	} else {
		w, h = cx, cy
		if w <= 0 {
			w = 100
		}
		if h <= 0 {
			h = sliderThumb + sliderRingPad*2
		}
		w += sliderHalo * 2
	}
	e := widget.NewElement(x, y, w, h, hParent)
	if e == nil {
		return nil
	}
	s.Element = e
	e.EnableFocus(true) // 可聚焦, 支持键盘方向键
	if s.disabled {
		// 构造期直接落禁用: Enable(false) 禁掉鼠标, EnableFocus(false) 禁掉键盘
		// (XEle_Enable 不禁鼠标聚焦, 见 DESIGN 第八节 114)。
		e.EnableFocus(false)
		e.Enable(false)
	}

	e.AddEvent_Paint(s.onPaint)
	e.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		e.Redraw(true)
		return 0
	})
	e.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		// 元素级 MouseLeave 在指针离开元素时到达; 清掉滑块悬停态。
		if s.hoverThumb != -1 {
			s.hoverThumb = -1
			e.Redraw(true)
		}
		return 0
	})
	e.AddEvent_LButtonDown(s.onLButtonDown)
	e.AddEvent_MouseMove(s.onMouseMove)
	e.AddEvent_LButtonUp(s.onLButtonUp)
	e.AddEvent_KeyDown(s.onKeyDown)
	xui.AttachWindow(e.Handle)
	return s
}

// onLButtonDown 起手: 按在滑块上就拖它(不改值), 按在轨道上就把最近的滑块
// 跳到点击位置并开始拖(radix 的点击跳转语义)。
func (s *Slider) onLButtonDown(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if s.disabled {
		return 0
	}
	xui.NotePointerPress() // 鼠标操作标记焦点来源, 保持 focus-visible 语义
	// 普通 Element 点击不会自动获得键盘焦点(原生只有按钮/编辑框等控件接管),
	// 不显式置焦点的话键盘方向键永远收不到。
	s.SetFocus()

	idx := s.thumbAt(pPt.X, pPt.Y)
	if idx >= 0 {
		s.active = idx
	} else {
		s.active = s.nearestThumb(pPt.X, pPt.Y)
		if s.setValueAt(s.active, s.pointValue(pPt.X, pPt.Y)) {
			s.notifyChange()
		}
	}
	s.drag = s.active
	xc.XEle_SetCapture(hEle, true) // 捕获鼠标, 拖出元素范围也能继续
	s.Redraw(true)
	return 0
}

func (s *Slider) onMouseMove(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if s.disabled {
		return 0
	}
	if s.drag >= 0 && s.drag < len(s.values) {
		if s.setValueAt(s.drag, s.pointValue(pPt.X, pPt.Y)) {
			s.notifyChange()
		}
		s.Redraw(true)
		return 0
	}
	// 未拖动时跟踪滑块 hover(只有指针落在滑块热区才亮光环, 同 shadcn 的 thumb:hover)
	if idx := s.thumbAt(pPt.X, pPt.Y); idx != s.hoverThumb {
		s.hoverThumb = idx
		s.Redraw(true)
	}
	return 0
}

func (s *Slider) onLButtonUp(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
	if s.drag >= 0 {
		s.drag = -1
		xc.XEle_SetCapture(hEle, false)
		s.Redraw(true)
		s.notifyCommit()
	}
	return 0
}

// onKeyDown 键盘交互(radix 语义): 水平 ←→、垂直 ↑↓ 移动一个步进(连续时 ±5%),
// PageUp/PageDown 移动十步, Home/End 到两端; inverted 时方向键反向。
func (s *Slider) onKeyDown(hEle int, wParam, lParam uintptr, pbHandled *bool) int {
	if s.disabled || len(s.values) == 0 {
		return 0
	}
	const (
		vkHome     = 0x24
		vkEnd      = 0x23
		vkLeft     = 0x25
		vkUp       = 0x26
		vkRight    = 0x27
		vkDown     = 0x28
		vkPageUp   = 0x21
		vkPageDown = 0x22
	)
	span := s.max - s.min
	// 连续模式沿用 ±5%(与旧版行为一致); 有 step 时一个步进 = step。
	step := span / 20
	if s.step > 0 {
		step = s.step
	}
	if step <= 0 {
		return 0
	}

	newV := math.NaN()
	switch wParam {
	case vkLeft:
		if !s.vertical {
			newV = s.values[s.active] - step
		}
	case vkRight:
		if !s.vertical {
			newV = s.values[s.active] + step
		}
	case vkUp:
		if s.vertical {
			newV = s.values[s.active] + step
		}
	case vkDown:
		if s.vertical {
			newV = s.values[s.active] - step
		}
	case vkPageUp:
		newV = s.values[s.active] + step*10
	case vkPageDown:
		newV = s.values[s.active] - step*10
	case vkHome:
		newV = s.min
	case vkEnd:
		newV = s.max
	}
	if math.IsNaN(newV) {
		return 0
	}
	if s.inverted && wParam != vkHome && wParam != vkEnd {
		newV = s.values[s.active] - (newV - s.values[s.active])
	}
	if s.setValueAt(s.active, newV) {
		s.Redraw(true)
		s.notifyChange()
		s.notifyCommit()
	}
	return 0
}

// Values 返回全部滑块的值(升序副本)。
func (s *Slider) Values() []float64 {
	return append([]float64(nil), s.values...)
}

// Value 返回第一个滑块的值(单滑块场景就是唯一值; 默认值域 0~100)。
func (s *Slider) Value() float64 {
	if len(s.values) == 0 {
		return s.min
	}
	return s.values[0]
}

// ValueAt 返回第 i 个滑块的值(越界返回 s.min)。
func (s *Slider) ValueAt(i int) float64 {
	if i < 0 || i >= len(s.values) {
		return s.min
	}
	return s.values[i]
}

// SetValue 设置第一个滑块的值, v 会被夹取到值域/相邻滑块之间并按步进吸附。
func (s *Slider) SetValue(v float64) *Slider {
	if len(s.values) == 0 {
		s.values = []float64{v}
	}
	if s.setValueAt(0, v) {
		s.Redraw(true)
	}
	return s
}

// SetValues 设置全部滑块的值(数量可以变化, 数量变化即重建滑块)。
// 传入值会被排序、夹取到值域并按 minStepsBetweenThumbs 拉开间距; 不触发回调。
func (s *Slider) SetValues(vals ...float64) *Slider {
	if len(vals) == 0 {
		return s
	}
	s.values = append([]float64(nil), vals...)
	s.normalize()
	if s.active >= len(s.values) {
		s.active = len(s.values) - 1
	}
	if s.active < 0 {
		s.active = 0
	}
	if s.drag >= len(s.values) {
		s.drag = -1
	}
	if s.hoverThumb >= len(s.values) {
		s.hoverThumb = -1
	}
	s.Redraw(true)
	return s
}

// SetValueAt 设置第 i 个滑块的值(越界忽略); 不触发回调。
func (s *Slider) SetValueAt(i int, v float64) *Slider {
	if i < 0 || i >= len(s.values) {
		return s
	}
	if s.setValueAt(i, v) {
		s.Redraw(true)
	}
	return s
}

// SetDisabled 设置禁用态(对应 radix 的 disabled): 不可拖动、不可聚焦, 整体 opacity-50。
func (s *Slider) SetDisabled(v bool) *Slider {
	if s.disabled == v {
		return s
	}
	s.disabled = v
	s.Enable(!v)
	s.EnableFocus(!v)
	if v {
		s.drag = -1
		s.hoverThumb = -1
	}
	s.Redraw(false)
	return s
}

// Disabled 返回是否禁用。
func (s *Slider) Disabled() bool { return s.disabled }

// OnChange 注册值变化回调(v 为**第一个**滑块的值), 拖动过程中会连续触发。
// 多滑块请用 OnValuesChange。为兼容旧 API 保留单值签名。
func (s *Slider) OnChange(fn func(v float64)) *Slider {
	if fn == nil {
		return s
	}
	s.changes = append(s.changes, func(vals []float64) { fn(vals[0]) })
	return s
}

// OnValuesChange 注册值变化回调(对应 radix 的 onValueChange), 拖动/键盘过程中连续触发。
func (s *Slider) OnValuesChange(fn func(vals []float64)) *Slider {
	if fn == nil {
		return s
	}
	s.changes = append(s.changes, fn)
	return s
}

// OnValuesCommit 注册"操作结束"回调(对应 radix 的 onValueCommit):
// 拖拽松手、键盘调整后各触发一次, 适合做提交/请求。
func (s *Slider) OnValuesCommit(fn func(vals []float64)) *Slider {
	if fn == nil {
		return s
	}
	s.commits = append(s.commits, fn)
	return s
}

func (s *Slider) notifyChange() {
	if len(s.changes) == 0 {
		return
	}
	vals := append([]float64(nil), s.values...)
	for _, fn := range s.changes {
		fn(vals)
	}
}

func (s *Slider) notifyCommit() {
	if len(s.commits) == 0 {
		return
	}
	vals := append([]float64(nil), s.values...)
	for _, fn := range s.commits {
		fn(vals)
	}
}

// unitStep 是一格的大小: 有 step 用 step, 连续时用值域的千分之一。
func (s *Slider) unitStep() float64 {
	if s.step > 0 {
		return s.step
	}
	return (s.max - s.min) / float64(sliderRange)
}

// minGap 是相邻滑块之间允许的最小差值(minStepsBetweenThumbs)。
func (s *Slider) minGap() float64 {
	if s.minSteps <= 0 {
		return 0
	}
	return s.unitStep() * float64(s.minSteps)
}

// normalize 排序 + 夹取到值域 + 按最小间距拉开(构造与 SetValues 共用)。
func (s *Slider) normalize() {
	if len(s.values) == 0 {
		s.values = []float64{s.min}
	}
	for i := range s.values {
		s.values[i] = s.clampRange(s.values[i])
	}
	sort.Float64s(s.values)
	gap := s.minGap()
	for i := 1; i < len(s.values); i++ {
		if minV := s.values[i-1] + gap; s.values[i] < minV {
			s.values[i] = minV
		}
	}
	// 末端被顶出值域时从后往前回推
	for i := len(s.values) - 1; i >= 0; i-- {
		upper := s.max
		if i < len(s.values)-1 {
			upper = s.values[i+1] - gap
		}
		if s.values[i] > upper {
			s.values[i] = upper
		}
		if s.values[i] < s.min {
			s.values[i] = s.min
		}
	}
}

func (s *Slider) clampRange(v float64) float64 {
	if v < s.min {
		return s.min
	}
	if v > s.max {
		return s.max
	}
	return v
}

// setValueAt 写入第 i 个滑块的值: 夹进"相邻滑块的间距"之间并吸附到步进。
// 返回是否真的发生了变化。
func (s *Slider) setValueAt(i int, v float64) bool {
	if i < 0 || i >= len(s.values) {
		return false
	}
	v = s.clampRange(v)
	gap := s.minGap()
	lower, upper := s.min, s.max
	if i > 0 {
		lower = s.values[i-1] + gap
	}
	if i < len(s.values)-1 {
		upper = s.values[i+1] - gap
	}
	if lower > upper { // 空间不足(值域比间距还小): 保持原值
		return false
	}
	if v < lower {
		v = lower
	} else if v > upper {
		v = upper
	}
	v = s.snap(v)
	if v < lower {
		v = lower
	} else if v > upper {
		v = upper
	}
	if v == s.values[i] {
		return false
	}
	s.values[i] = v
	return true
}

// snap 按步进吸附(step <= 0 时原样返回)。
func (s *Slider) snap(v float64) float64 {
	if s.step <= 0 {
		return v
	}
	v = s.min + math.Round((v-s.min)/s.step)*s.step
	return s.clampRange(v)
}

// geoFrac 把值换算成**几何比例**(水平: 从左到右; 垂直: 从下到上), 已处理 inverted。
func (s *Slider) geoFrac(v float64) float64 {
	span := s.max - s.min
	if span <= 0 {
		return 0
	}
	f := (v - s.min) / span
	if s.inverted {
		f = 1 - f
	}
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	return f
}

// valueOfGeo 是 geoFrac 的反变换。
func (s *Slider) valueOfGeo(f float64) float64 {
	if s.inverted {
		f = 1 - f
	}
	return s.min + f*(s.max-s.min)
}

// trackLen 返回元素内可见轨道的长度(沿轴, 逻辑像素)。
func (s *Slider) trackLen() int32 {
	if s.vertical {
		return s.GetHeight() - sliderHalo*2
	}
	return s.GetWidth() - sliderHalo*2
}

// pointValue 把元素客户区坐标换算成值(拖动/点击轨道用)。
func (s *Slider) pointValue(px, py int32) float64 {
	length := s.trackLen()
	if length <= 0 {
		return s.Value()
	}
	var f float64
	if s.vertical {
		// 底部 = 0, 顶部 = 1
		f = float64(s.GetHeight()-sliderHalo-py) / float64(length)
	} else {
		f = float64(px-sliderHalo) / float64(length)
	}
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	return s.snap(s.clampRange(s.valueOfGeo(f)))
}

// thumbAxis 返回第 i 个滑块中心沿轴的坐标(水平 = x, 垂直 = y)。
func (s *Slider) thumbAxis(i int) int32 {
	length := s.trackLen()
	var f float64
	if i >= 0 && i < len(s.values) {
		f = s.geoFrac(s.values[i])
	}
	if s.vertical {
		return s.GetHeight() - sliderHalo - int32(math.Round(f*float64(length)))
	}
	return sliderHalo + int32(math.Round(f*float64(length)))
}

// thumbAt 返回指针命中的滑块索引(-1 = 未命中)。热区 = 滑块半径 + 2px,
// 对应 shadcn 滑块上的 `after:-inset-2`。
func (s *Slider) thumbAt(px, py int32) int {
	const pad = sliderThumb/2 + 2
	best, bestD := -1, pad+1
	for i := range s.values {
		var d int32
		if s.vertical {
			// 垂直时还要落在元素横向范围内(元素宽本身就窄)
			if px < 0 || px > s.GetWidth() {
				continue
			}
			d = py - s.thumbAxis(i)
		} else {
			d = px - s.thumbAxis(i)
		}
		if d < 0 {
			d = -d
		}
		if d <= pad && (best < 0 || d < bestD) {
			best, bestD = i, d
		}
	}
	return best
}

// nearestThumb 返回值方向上离指针最近的滑块索引(点击轨道跳转用)。
func (s *Slider) nearestThumb(px, py int32) int {
	if len(s.values) == 0 {
		return 0
	}
	length := s.trackLen()
	if length <= 0 {
		return 0
	}
	var f float64
	if s.vertical {
		f = float64(s.GetHeight()-sliderHalo-py) / float64(length)
	} else {
		f = float64(px-sliderHalo) / float64(length)
	}
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	v := s.valueOfGeo(f)
	best := 0
	bestD := math.Abs(s.values[0] - v)
	for i := 1; i < len(s.values); i++ {
		if d := math.Abs(s.values[i] - v); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func (s *Slider) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true // 全部自绘
	t := xui.Current()
	st := xui.StateOf(hEle)

	// 先铺宿主底色: 本组件不整体填充(轨道/滑块都是局部形状), 若不铺底,
	// 上一帧的 ring/滑块像素会残留在画布上(实测 hover 离开/点击聚焦后灰环不消失)。
	// 默认用 Card(组件通常放在卡片上); 放在其他底色上请改窗口主题或自行调整。
	p.FillRound(&p.Rc, t.Card, 0)

	disabled := s.disabled || st.Disabled
	alpha := byte(255)
	if disabled {
		alpha = 128 // data-disabled:opacity-50
	}

	// 轨道几何: 沿轴两端内缩 sliderHalo(与滑块中心的活动范围一致)
	var track xc.RECT
	var trackLen int32
	if s.vertical {
		x := p.Rc.Left + p.W()/2 - sliderTrackH/2
		track = xc.RECT{
			Left:   x,
			Top:    p.Rc.Top + sliderHalo,
			Right:  x + sliderTrackH,
			Bottom: p.Rc.Bottom - sliderHalo,
		}
		trackLen = track.Bottom - track.Top
	} else {
		y := p.Rc.Top + p.H()/2 - sliderTrackH/2
		track = xc.RECT{
			Left:   p.Rc.Left + sliderHalo,
			Top:    y,
			Right:  p.Rc.Right - sliderHalo,
			Bottom: y + sliderTrackH,
		}
		trackLen = track.Right - track.Left
	}
	if trackLen < 0 {
		trackLen = 0
	}

	// 轨道: h-1 rounded-full bg-muted
	p.FillRound(&track, t.Muted.WithAlpha(alpha), sliderTrackH/2)

	// 已完成区间: 单滑块 = 值域起点到滑块; 多滑块 = 第一个到最后一个(radix 的 Range 语义)。
	// 起点要用**值域下限的几何位置**, inverted 时它在右端(直接写 0 会把整条点亮)。
	if len(s.values) > 0 && trackLen > 0 {
		var f0, f1 float64
		if len(s.values) == 1 {
			f0, f1 = s.geoFrac(s.min), s.geoFrac(s.values[0])
		} else {
			f0, f1 = s.geoFrac(s.values[0]), s.geoFrac(s.values[len(s.values)-1])
		}
		if f0 > f1 {
			f0, f1 = f1, f0
		}
		if f1 > f0 {
			var rangeRc xc.RECT
			if s.vertical {
				top := track.Bottom - int32(math.Round(f1*float64(trackLen)))
				bottom := track.Bottom - int32(math.Round(f0*float64(trackLen)))
				rangeRc = xc.RECT{Left: track.Left, Top: top, Right: track.Right, Bottom: bottom}
			} else {
				left := track.Left + int32(math.Round(f0*float64(trackLen)))
				right := track.Left + int32(math.Round(f1*float64(trackLen)))
				rangeRc = xc.RECT{Left: left, Top: track.Top, Right: right, Bottom: track.Bottom}
			}
			p.FillRound(&rangeRc, t.Primary.WithAlpha(alpha), sliderTrackH/2)
		}
	}

	// 逐滑块绘制(hover / 拖动 / 键盘焦点各自带 ring, 过渡走运行时 HoverItem)
	focusedByKey := st.Focused && !disabled && xui.FocusedByKeyboard()
	for i := range s.values {
		center := s.thumbPoint(p.Rc, i)
		haloOn := !disabled && (s.drag == i || s.hoverThumb == i || (focusedByKey && s.active == i))
		if hv := xui.HoverItem(hEle, int32(i), haloOn); hv > 0 {
			halo := xc.RECT{
				Left:   center.X - sliderHalo,
				Top:    center.Y - sliderHalo,
				Right:  center.X + sliderHalo,
				Bottom: center.Y + sliderHalo,
			}
			p.FillRound(&halo, t.Ring.WithAlpha(128).ScaleAlpha(hv), sliderHalo)
		}

		// 滑块本体: border-ring bg-white(shadcn 里滑块底色是字面量 white, 两个主题都一样)
		thumb := xc.RECT{
			Left:   center.X - sliderThumb/2,
			Top:    center.Y - sliderThumb/2,
			Right:  center.X + sliderThumb/2,
			Bottom: center.Y + sliderThumb/2,
		}
		p.Draw.SetLineWidth(1)
		// 滑块是"内容", 禁用(组级透明)下核心保持原色, 只有轨道/完成区间变淡
		p.Draw.SetBrushColor(uint32(xui.RGBA(255, 255, 255, 255)))
		p.Draw.FillEllipse(&thumb)
		p.Draw.SetBrushColor(uint32(t.Ring))
		p.Draw.DrawEllipse(&thumb)
	}
	return 0
}

// thumbPoint 返回第 i 个滑块中心的元素坐标(水平 = 轨道中线, 垂直 = 轨道中线)。
func (s *Slider) thumbPoint(rc xc.RECT, i int) xc.POINT {
	var f float64
	if i >= 0 && i < len(s.values) {
		f = s.geoFrac(s.values[i])
	}
	if s.vertical {
		length := (rc.Bottom - rc.Top) - sliderHalo*2
		return xc.POINT{
			X: rc.Left + (rc.Right-rc.Left)/2,
			Y: rc.Bottom - sliderHalo - int32(math.Round(f*float64(length))),
		}
	}
	length := (rc.Right - rc.Left) - sliderHalo*2
	return xc.POINT{
		X: rc.Left + sliderHalo + int32(math.Round(f*float64(length))),
		Y: rc.Top + (rc.Bottom-rc.Top)/2,
	}
}
