package ui

import (
	"time"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// 滚动条几何: 逐条对齐 shadcn scroll-area.tsx 的滚动条 ——
//
//	轨道 = data-vertical:w-2.5 → 10 逻辑像素;
//	滑块 = p-px 内缩 → 8 逻辑像素宽;
//	滑块圆角 = rounded-full(8px 宽的两端全圆) → 半径 4。
//
// 各载体(ScrollArea/Table/Tree/ComboBox 弹层/多行 Input)构造时统一
// SetScrollBarSize(scrollBarW), 外观统一走 styleScrollBar, 显隐统一走
// showScrollBarOnHover。
const (
	scrollBarW      int32 = 10
	scrollBarInset  int32 = 1
	scrollBarThumbR       = int32(4)
)

// scrollBarFadeDur 是滚动条"随宿主悬停淡入 / 离开淡出"的时长(与 hover 过渡同速)。
const scrollBarFadeDur = 150 * time.Millisecond

// scrollBarSkin 是一条滚动条的配色: 滑块的两级色(轨道色由调用方随 styleScrollBar 传入)。
//
// 颜色必须是**不透明值** —— 见 flatOn 的说明(半透明色会与所在表面再合成一次,
// 同一个滑块在页面与卡片上会呈现两种灰度)。thumbHover == thumb 的皮肤(容器型)
// 悬停不换色, 换色回调会自动成为空操作。
type scrollBarSkin struct {
	thumb      uint32 // 滑块静止色
	thumbHover uint32 // 滑块悬停色(按下同色)
}

// 滑块三个状态的背景对象 id(AddFill 时登记, hover 换色时按 id 取回改色)。
const (
	thumbIDLeave int32 = 1
	thumbIDStay  int32 = 2
	thumbIDDown  int32 = 3
)

// thumbSkinReg 记录每条滚动条当前生效的皮肤(hover 换色时按它取两级色)。
// 键是滚动条元素句柄; 都是 UI 线程读写(样式刷新与鼠标事件同线程), 不需要加锁。
var thumbSkinReg = map[int]scrollBarSkin{}

// scrollBarHoverResets 按宿主句柄登记"把滚动条恢复到未悬停初始态"的闭包,
// 供弹窗在每次打开时主动归零(快速重开时宿主未必触发过 Show(false))。
var scrollBarHoverResets = map[int]func(){}

// resetScrollBarHover 把某个宿主上的滚动条恢复成"未悬停"初始态:
// 清掉显隐过渡、alpha 归零、滑块换回静止色。宿主隐藏时由 Show 事件自动调用。
func resetScrollBarHover(hHost int) {
	if fn := scrollBarHoverResets[hHost]; fn != nil {
		fn()
	}
}

// containerScrollBarSkin 是"容器型"滚动条的皮肤(ScrollArea 的页面与卡片内滚动容器):
// 对齐 shadcn scroll-area.tsx —— 滑块 bg-border、悬停不换色。
func containerScrollBarSkin(t *xui.Theme) scrollBarSkin {
	thumb := uint32(flatOn(t.Border, t.Card))
	return scrollBarSkin{thumb: thumb, thumbHover: thumb}
}

// embedScrollBarSkin 是"嵌入式"滚动条的皮肤(多行输入框 / 表格 / 树 / 下拉弹层)。
// 这几个载体在 shadcn 原版里就是浏览器原生滚动条, 观感对齐 Windows 的 Fluent
// 滚动条: 前景色的淡化版, 悬停加深(参考: 亮色 #8b8b8b → 悬停 #636363)。
func embedScrollBarSkin(t *xui.Theme) scrollBarSkin {
	// 前景色的不透明度: 亮色主题 45%(静止) / 61%(悬停), 暗色主题压到 22% / 35% ——
	// 同一 alpha 在暗色底上会明显偏亮(对比度差一倍)。
	leave, hover := 0.45, 0.61
	if isDarkTheme(t) {
		leave, hover = 0.22, 0.35
	}
	return scrollBarSkin{
		thumb:      uint32(flatOn(t.Foreground.ScaleAlpha(leave), t.Card)),
		thumbHover: uint32(flatOn(t.Foreground.ScaleAlpha(hover), t.Card)),
	}
}

// isDarkTheme 按主题底色亮度判断极性(不依赖主题名, 自定义主题同样适用)。
func isDarkTheme(t *xui.Theme) bool {
	r, g, b, _ := t.Background.RGBA()
	return int(r)*299+int(g)*587+int(b)*114 < 128*1000
}

// flatOn 把可能带透明度的颜色 c 合成到不透明的底 base 上, 得到不透明色。
//
// 滚动条滑块必须用不透明色: 半透明色(t.Border 在暗色主题下是 10% 白)会与
// "所在表面"再合成一次 —— 同一个滑块在页面(background)上是 #232323、在卡片
// (card)上是 #2f2f2f, 两个滚动条颜色肉眼可见地不一致。展平到卡片色后,
// 与表面的对比观感保持不变、处处一致。
func flatOn(c, base xui.Color) xui.Color {
	cr, cg, cb, ca := c.RGBA()
	if ca == 255 {
		return c
	}
	br, bg, bb, _ := base.RGBA()
	f := float64(ca) / 255
	mixv := func(a, b byte) byte { return byte(float64(a)*f + float64(b)*(1-f) + 0.5) }
	return xui.RGBA(mixv(cr, br), mixv(cg, bg), mixv(cb, bb), 255)
}

// styleScrollBar 把原生滚动条收敛成 shadcn 观感, 供各组件在 applyTheme 里调用:
// 去掉上下箭头、轨道铺 track 色(0 = 透明, 露出宿主底色)、滑块 = skin 的两级色
// 8px 圆头细条(rounded-full), 悬停/按下加深。
//
// ⚠️ 原生滚动条的配色只能经背景管理器下发, 换肤/重入必须 Clear 后重加, 否则累积
// 超过炫彩的 20 个背景对象上限(见 DESIGN 第八节 21); ClearBkInfo 后还必须垫一层
// 填充 —— 留空会回退"系统默认内容"(白底方角), 透明轨道用 alpha=0 的填充表达
// (见 DESIGN 第八节 30 的同类坑)。
func styleScrollBar(sb *widget.ScrollBar, track uint32, skin scrollBarSkin) {
	if sb == nil || sb.Handle <= 0 {
		return
	}
	sb.ShowButton(false)
	sb.SetSliderPadding(scrollBarInset)
	sb.EnableBkTransparent(true)
	sb.ClearBkInfo()
	sb.AddBkFill(xcc.Element_State_Flag_Leave, track)
	sb.AddBkFill(xcc.Element_State_Flag_Stay, track)

	sl := sb.GetButtonSliderObj()
	if sl == nil || sl.Handle <= 0 {
		return
	}
	sl.ClearBkInfo()
	bkm := sl.GetBkManagerObj()
	if bkm == nil {
		return
	}
	// 圆角只能设在**背景对象**上(bkmanager 的 id 版 AddFill + GetObjectObj)。
	// ⚠️ 每个状态必须用**各自的 id** 添加: 三个状态共用一个 id 时只有第一次生效,
	// hover(Stay) 时找不到背景对象会回退默认绘制 —— 表现为"悬停时滑块变宽且
	// 圆角消失"(实测, 见 DESIGN 第八节 21)。
	bkm.AddFill(xcc.Element_State_Flag_Leave, skin.thumb, thumbIDLeave)
	bkm.AddFill(xcc.Element_State_Flag_Stay, skin.thumbHover, thumbIDStay)
	bkm.AddFill(xcc.Element_State_Flag_Down, skin.thumbHover, thumbIDDown)
	for _, id := range []int32{thumbIDLeave, thumbIDStay, thumbIDDown} {
		if o := bkm.GetObjectObj(id); o != nil && o.Handle > 0 {
			o.SetRectRoundAngle(scrollBarThumbR, scrollBarThumbR, scrollBarThumbR, scrollBarThumbR)
		}
	}
	thumbSkinReg[sb.Handle] = skin
}

// setThumbSkinColor 就地把滑块三态的填充色改成 color(复用已登记的背景对象,
// 不重建 —— SetFillColor 是 BkObj 的属性, 改完立即重绘滚动条)。
func setThumbSkinColor(sb *widget.ScrollBar, color uint32) {
	if color == 0 {
		return // 皮肤还没登记(未 style 过), 别把滑块刷成透明
	}
	sl := sb.GetButtonSliderObj()
	if sl == nil || sl.Handle <= 0 {
		return
	}
	bkm := sl.GetBkManagerObj()
	if bkm == nil {
		return
	}
	for _, id := range []int32{thumbIDLeave, thumbIDStay, thumbIDDown} {
		if o := bkm.GetObjectObj(id); o != nil && o.Handle > 0 {
			o.SetFillColor(color)
		}
	}
	sl.Redraw(true)
}

// showScrollBarOnHover 把滚动条绑成"鼠标移入宿主(容器)才显示, 离开即淡出":
// 悬停在宿主内容或滚动条自身上时保持显示, 淡入淡出 150ms。
//
// 为什么用元素透明度而不是 Show(false): 原生滚动条的显隐翻转会改变视口宽度 ——
// 表格列宽、ScrollArea 内容宽都会跟着跳变(内容重排)。alpha 只是让滚动条"看不见",
// 布局宽度始终保留, 与 radix 的 overlay 滚动条同观感(见 DESIGN 第八节 47)。
//
// 显隐判定走**窗口级**鼠标移动 + `XWnd_GetStayEle`(鼠标当前停留元素):
// 鼠标落在宿主子树(含内容、滚动条、滑块)里就显示。不用元素级 MouseStay/MouseLeave
// 是因为原生控件的 leave 不可靠 —— 实测鼠标移出多行 Edit 后收不到 MouseLeave
// (滚动条会一直挂着), 而 Edit 内部的移动也未必逐级上报。窗口级每次移动重算一次,
// 顺带修正"从滑块移回内容区"这类跨子元素的过渡。
func showScrollBarOnHover(hHost int, sb *widget.ScrollBar) {
	if hHost <= 0 || sb == nil || sb.Handle <= 0 {
		return
	}
	hSB := sb.Handle
	xui.SetSubtreeAlpha(0, hSB) // 初始不可见(鼠标尚未进入)

	set := func(visible bool) {
		xui.Presence(hSB, 0, visible, scrollBarFadeDur, func(v float64) {
			xui.SetSubtreeAlpha(v, hSB)
		}, func(float64) {
			// 到位那一帧 anim 不再重绘(它对 hover 色差无感), 但滚动条要"完全消失"
			// 就得把最后一帧(v=0)真正画出来, 否则屏幕上留一层淡残影。
			xc.XEle_Redraw(hSB, true)
		})
	}

	hWindow := xc.XWidget_GetHWINDOW(hHost)
	if hWindow == 0 {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	// onBar 记录"鼠标是否停在滚动条上", 只在翻转时换一次颜色。
	onBar := false
	w.AddEvent_MouseMove(func(hWindow int, nFlags uint, pPt *xc.POINT, pbHandled *bool) int {
		stay := xc.XWnd_GetStayEle(hWindow)
		set(stay == hHost || (stay > 0 && xc.XEle_IsChildEle(hHost, stay)))
		// 滑块"悬停加深": 鼠标停在滚动条子树里(实测: 悬停时 stay 返回滚动条或
		// 滑块句柄, 与渲染位置完全一致)时换 hover 色。不用滚动条自身的 MouseStay
		// —— 原生 List 不把鼠标悬停路由给它的滚动条。容器型皮肤两级同色, 这里是空操作。
		inBar := stay == hSB || (stay > 0 && xc.XEle_IsChildEle(hSB, stay))
		if inBar != onBar {
			onBar = inBar
			skin := thumbSkinReg[hSB]
			if inBar {
				setThumbSkinColor(sb, skin.thumbHover)
			} else {
				setThumbSkinColor(sb, skin.thumb)
			}
		}
		return 0
	})
	// 鼠标离开整个窗口: 收不到 MouseMove, 单独收口。
	w.AddEvent_MouseLeave(func(hWindow int, pbHandled *bool) int {
		set(false)
		if onBar {
			onBar = false
			setThumbSkinColor(sb, thumbSkinReg[hSB].thumb)
		}
		return 0
	})

	// reset 把滚动条回到"未悬停"初始态。宿主隐藏(页面切换/弹窗关闭/最小化)时调用:
	// 滚动条的 alpha 是显隐状态, 不归零的话下次显示会带着上一次的可见状态出现
	// (弹窗的整组淡入已跳过滚动条, 更依赖这里复位)。
	reset := func() {
		xui.DropTransitions(hSB)
		xui.SetSubtreeAlpha(0, hSB)
		onBar = false
		setThumbSkinColor(sb, thumbSkinReg[hSB].thumb)
		xc.XEle_Redraw(hSB, true)
	}
	scrollBarHoverResets[hHost] = reset
	if host := widget.NewElementByHandle(hHost); host != nil {
		host.AddEvent_Show(func(hEle int, bShow bool, pbHandled *bool) int {
			if !bShow {
				reset()
			}
			return 0
		})
		host.AddEvent_Destroy_End(func(hEle int, pbHandled *bool) int {
			delete(scrollBarHoverResets, hHost)
			return 0
		})
	}

	// 滚动条销毁(弹层 ListBox 每次都新建)时清掉登记, 句柄会被复用。
	if bar := widget.NewElementByHandle(hSB); bar != nil {
		bar.AddEvent_Destroy_End(func(hEle int, pbHandled *bool) int {
			delete(thumbSkinReg, hSB)
			return 0
		})
	}
}
