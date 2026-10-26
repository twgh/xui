package ui

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
)

// ScrollArea 滚动容器: 内容比容器高时纵向滚动(而不是被裁掉)。
//
// 载体是原生 **LayoutFrame(布局框架)** —— 它 = 布局盒子 + 滚动视图: 和其他容器一样排子项,
// 内容总大小**按子项自动撑开**、内容溢出时自动出现纵向滚动条, 不需要手动 SetTotalSize
// (那是 ScrollView 的用法, 见 xcgui 的 Basic/ScrollView 示例)。探针实测(见 DESIGN 第八节 66):
//
//   - 子项 Fixed / Auto 高都拿到自然值, 不会被压扁; 内容比视口高就出滚动条;
//   - 滚动偏移由框架自己作用于绘制, 子元素矩形读出来仍是"内容坐标";
//   - 鼠标滚轮: 悬停在**子元素上**也会冒泡到框架并滚动(子元素的 XE_MOUSEWHEEL 默认禁用);
//   - 代码滚动用 `ScrollPosYV(像素)`; `ScrollPosV` 的单位是"行"(默认 20px 一行), 不是像素;
//   - 滚出视口的内容不会画出来(视口裁剪), **不要**再开 EnableOverflowHide —— 那是"布局时把
//     超出视口的子项高度裁掉", 卡片会被切短(实测 400 高的子项被压成 10)。
//
// 用法(内容高度按内容自撑, 超出视口即滚动):
//
//	sa := ui.NewScrollArea(0, 0, 0, 0, pageHost)
//	sa.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0) // 宽铺满
//	xui.ItemFillHeight(sa)                          // 高吃父级剩余空间(Weight 1)
//	body := sa.ContentStack(16)                     // 内容区: 宽铺满视口, 高按内容自撑
//	body.LayoutItem_SetMargin(0, 24, 0, 24)         // 上下留白(外边距计入滚动范围)
//	body.SetPadding(24, 0, 24, 0)                   // 左右用元素 padding, 约束子项 Fill 宽
//	... 往 body 里塞卡片/控件
//
// 与 shadcn / radix ScrollArea 的差异(有意为之):
//   - 滚动条"鼠标移入容器才显示, 离开淡出"(同 radix 的 type=hover); 隐藏用元素透明度
//     而非 Show(false) —— 布局宽度始终保留, 内容不会跳变; 几何/配色统一到 scrollbar.go
//     (轨道 10px + 滑块 8px 圆头 bg-border, 与 Table/Tree/ComboBox/多行 Input 同规格);
//   - 只做纵向: 内容区永远铺满视口宽, 没有横向滚动条。
type ScrollArea struct {
	*widget.LayoutFrame

	content *widget.LayoutEle

	// 首帧、以及滚动条显隐翻转的那一帧, 编排会把滚动条宽度多减一次 —— 子项右缘因此比视口
	// 窄一个滚动条宽度(探针实测: 视口 772 时子项只有 760, 右侧多出 12px 空白)。再排一次
	// 就正确, 这里用"上次看到的尺寸"做闸门自愈: 同一对尺寸只补排一次, 不会来回震荡。
	lastFW, lastViewW int32
	healing           bool
}

// NewScrollArea 创建滚动容器。尺寸/布局项语义与普通容器相同(默认不接管尺寸, 用 LayoutItem_* 声明)。
func NewScrollArea(x, y, cx, cy int32, hParent int) *ScrollArea {
	lf := widget.NewLayoutFrame(x, y, cx, cy, hParent)
	if lf == nil {
		return nil
	}
	s := &ScrollArea{LayoutFrame: lf, lastFW: -1, lastViewW: -1}

	// 框架自己: 竖向流 + 关掉自动换列(换列会把溢出的控件甩到容器右边, 见 DESIGN 第八节 41)。
	lf.EnableHorizon(false)
	lf.EnableAutoWrap(false)
	// 容器被点一下会获得焦点并画出原生蓝框(实测), 灭掉(同 DESIGN 第八节 34 的容器处理)。
	lf.EnableFocus(false)
	lf.EnableDrawBorder(false)

	lf.SetScrollBarSize(scrollBarW)
	lf.EnableAutoShowScrollBar(true)
	lf.ShowSBarH(false) // 内容区永远铺满视口宽, 横向滚动条不需要

	// 内容区: 宽铺满**视口**, 高按内容自撑 —— 这正是"内容比视口高就滚动"的形态。
	//
	// 高度不能写作 Fill / Weight(探针实测, 见 DESIGN 第八节 66):
	//   - Fill = 视口高: 内容溢出视口后**不算进滚动范围**(8×100 的卡片在 662 高的框架里
	//     total 仍是 662, 滚不动) —— 那是"铺满视口 + 内层用 Weight 吃剩余"的形态, 见 SetContentFill;
	//   - Weight 在框架里恒为 0(内容轴按内容自撑, 父级尺寸不确定)。
	c := widget.NewLayoutEle(0, 0, 0, 0, lf.Handle)
	c.EnableHorizon(false)
	c.EnableAutoWrap(false)
	c.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	c.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	s.content = c

	lf.AddEvent_Size(s.onSize)
	xui.OnThemeChangeFor(lf.Handle, s.applyTheme)
	s.applyTheme(xui.Current())
	xui.AttachWindow(lf.Handle)
	showScrollBarOnHover(lf.Handle, lf.GetScrollBarVObj())
	return s
}

// Content 返回内容区(竖向流容器): 宽铺满视口, 高按内容自撑, 超出视口即滚动。
// 往它里面塞内容即可; 要设子项间距用 ContentStack 或 SetSpace。
func (s *ScrollArea) Content() *widget.LayoutEle {
	return s.content
}

// ContentStack 设置内容区的子项间距并返回它(与 Card.ContentStack 同名同义)。
func (s *ScrollArea) ContentStack(space int32) *widget.LayoutEle {
	s.content.SetSpace(space)
	return s.content
}

// SetContentFill 把内容区改成"铺满视口"形态: 内容高度 = 视口高度, 内层子项可以用
// xui.ItemFillHeight 吃满剩余高度(表格 / 树这类要跟随窗口长高的面板)。
//
// 代价是页面**不再滚动**: 真实内容比视口高时会被裁掉(见构造函数注释里的实测), 只适合
// "内容都由 Weight 自适应、永远装得下"的页面; 要滚动就用默认的 Auto。
func (s *ScrollArea) SetContentFill() *ScrollArea {
	s.content.LayoutItem_SetHeight(xcc.Layout_Size_Fill, 0)
	s.AdjustLayout()
	s.Redraw(false)
	return s
}

// ScrollToTop 滚回顶部。
func (s *ScrollArea) ScrollToTop() *ScrollArea {
	s.ScrollPosYV(0)
	return s
}

// onSize 元素尺寸事件: 首帧与滚动条显隐翻转的那一帧补排一次(见 lastFW 注释)。
func (s *ScrollArea) onSize(hEle int, nFlags xcc.AdjustLayout_, nAdjustNo uint32, pbHandled *bool) int {
	if s.healing {
		return 0
	}
	fw, vw := s.GetWidth(), s.GetViewWidth()
	if fw == s.lastFW && vw == s.lastViewW {
		return 0
	}
	s.lastFW, s.lastViewW = fw, vw
	s.healing = true
	s.AdjustLayout()
	s.healing = false
	return 0
}

// applyTheme 把滚动条收成主题细条(几何/圆角/配色见 scrollbar.go): 轨道透明(露出页面
// 底色), 滑块 = 容器型皮肤(bg-border 圆头, 悬停不换色, 同 shadcn)。滚动条颜色只能经
// 背景管理器下发给原生对象, 换肤要重刷一遍(Clear 后重加, 不会累积到炫彩的 20 个背景
// 对象上限)。
func (s *ScrollArea) applyTheme(th *xui.Theme) {
	styleScrollBar(s.GetScrollBarVObj(), 0, containerScrollBarSkin(th))
}
