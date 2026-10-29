package ui

import (
	"strconv"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// 分页组件的几何常量(逻辑像素), 对应 shadcn pagination.tsx + button.tsx 的尺寸类。
const (
	pagLinkH int32 = 32 // size-8: 页码链接 32×32(上一页/下一页也是 h-8)
	pagGap   int32 = 2  // PaginationContent 的 gap-0.5
	pagPadX  int32 = 6  // 上一页/下一页的水平内边距(pl-1.5 / pr-1.5)
	// pagPadIn: 页码数字到方角标边缘的呼吸量(按钮是 h-8 px-2.5, 但页码是 size-8 方形,
	// 文字居中即可, 只给一点内缩避免两位数贴边)
	pagPadIn int32 = 4
	pagIcon  int32 = 16 // size-4
	// pagIconGap 是图标与文字之间的间距(gap-1)。
	pagIconGap int32 = 4
	// pagPressShift 是按下位移量, 对应 shadcn 的 active:translate-y-px, 语义同 button.go。
	// 元素高度比视觉高度多这么多, 多出的部分留作位移空间(元素绘制被裁在自身客户区内)。
	pagPressShift int32 = 1
	// pagMaxLinks 是"不折叠"的最大页数: 超过它就只显示首/末页 + 当前页 ±1 + 省略号。
	pagMaxLinks = 7
)

// Pagination 分页组件(纯自绘: 一个元素承载全部链接, 鼠标驱动)。
//
// 与 shadcn pagination.tsx(radix-nova) 对齐:
//   - 条目序列 = [上一页][1][2]…[n][下一页], 内容整体水平居中
//     (nav 的 mx-auto flex w-full justify-center), 条目间距 gap-0.5;
//   - 页码链接是 Button size="icon"(32×32): 未选中为 ghost(悬停 bg-muted),
//     当前页为 outline(bg-background + border + hover:bg-muted);
//   - 上一页/下一页 = 图标(chevron-left/right, size-4) + 文字, 水平内边距 pl-1.5/pr-1.5;
//   - 省略号是 size-8 的 more-horizontal 图标(muted-foreground), 不可点击;
//   - 页数超过 pagMaxLinks(7) 时折叠成 7 个固定槽位(首段 / 中段 / 末段三种排布),
//     槽位数恒定 → 整行宽度不随页码变化, 条目的屏幕位置也不会跳(见 links);
//   - 悬停过渡走运行时 xui.HoverItem(150ms + cubic-bezier, 逐项键 = 下标 + 1),
//     按下 active:translate-y-px 下沉 1px;
//   - 到首页/末页时上一页/下一页按"禁用"绘制(50% 透明, 不响应鼠标)。
//
// 为什么整块自绘而不是拼 Button 子元素: 页码数量随数据变化, 拼子元素就得反复
// 创建/销毁元素并重排布局; 单元素自绘只需要一次命中测试(同 slider.go 的路线)。
//
// 形态对齐 shadcn 文档的三个示例: 完整(上一页/页码/下一页, 默认)、简洁(只有页码,
// WithPaginationSimple)、仅上一页/下一页(WithPaginationNavOnly, 配合"每页行数"选择器)。
// 上一页/下一页还可再隐藏文字只留 chevron(WithPaginationHideText), 对应 `hidden sm:block`。
//
// 键盘可达: 整块是一个 Tab 站点(链接级各占一个 Tab 站点需要每个链接一个元素, 与
// 单元素自绘冲突), 进入后用 ←/→ 在条目间移动、Home/End 跳到首末、Enter/Space 激活;
// 焦点环只在键盘获得焦点时绘制(focus-visible 语义)。
type Pagination struct {
	*widget.Element

	total    int // 总条目数
	pageSize int // 每页条目数
	page     int // 当前页(1 起)
	prevText string
	nextText string
	cb       func(page int)

	mode     pagMode // 显示形态(完整 / 只有页码 / 只有上一页下一页)
	hideText bool    // 上一页/下一页只显示 chevron(文字隐藏)

	hover   int // 鼠标悬停的条目下标(-1 = 无)
	pressed int // 按下的条目下标(-1 = 无)
	focus   int // 键盘焦点所在条目下标(-1 = 还没进过聚焦态)

	items []pagItem // 最近一次布局结果(绘制与命中测试共用)

	wFont int   // 上次量宽用的字体句柄(-1 = 还没量过)
	prevW int32 // 上一页文字宽
	nextW int32 // 下一页文字宽
}

// pagMode 是分页的显示形态。
type pagMode int

const (
	pagModeFull   pagMode = iota // [上一页][1][2]…[n][下一页]
	pagModeSimple                // 只有页码(shadcn 的"简洁"示例)
	pagModeNav                   // 只有上一页/下一页(shadcn 的"仅图标"示例, 配合每页行数选择器)
)

// PaginationOption 分页组件配置项, 构造时传入。
type PaginationOption func(*Pagination)

// WithPrevText 覆盖"上一页"的文字(shadcn 默认 "Previous")。
func WithPrevText(s string) PaginationOption { return func(g *Pagination) { g.prevText = s } }

// WithNextText 覆盖"下一页"的文字(shadcn 默认 "Next")。
func WithNextText(s string) PaginationOption { return func(g *Pagination) { g.nextText = s } }

// WithPaginationSimple 只要页码, 不显示上一页/下一页(对应文档的"简洁"示例)。
func WithPaginationSimple() PaginationOption {
	return func(g *Pagination) { g.mode = pagModeSimple }
}

// WithPaginationNavOnly 只要上一页/下一页, 不显示页码(对应文档的"仅图标"示例,
// 适合配上"每页行数"选择器的数据表)。
func WithPaginationNavOnly() PaginationOption {
	return func(g *Pagination) { g.mode = pagModeNav }
}

// WithPaginationHideText 上一页/下一页只显示 chevron 图标、不显示文字
// (对应 pagination.tsx 里文字上的 `hidden sm:block`, 窄容器里省地方)。
func WithPaginationHideText() PaginationOption {
	return func(g *Pagination) { g.hideText = true }
}

// NewPagination 创建分页组件。cx 是元素宽度(内容在其中居中, 可传布局填充后的宽度),
// cy 传 0 时使用默认高度(pagLinkH + 按下位移预留)。
func NewPagination(x, y, cx, cy int32, hParent int, opts ...PaginationOption) *Pagination {
	if cy <= 0 {
		cy = pagLinkH + pagPressShift
	}
	if cx <= 0 {
		cx = pagLinkH * 6
	}
	ele := widget.NewElement(x, y, cx, cy, hParent)
	if ele == nil {
		return nil
	}
	g := &Pagination{
		Element:  ele,
		page:     1,
		pageSize: 10,
		prevText: "Previous",
		nextText: "Next",
		hover:    -1,
		pressed:  -1,
		focus:    -1,
		wFont:    -1,
	}
	for _, o := range opts {
		o(g)
	}
	ele.EnableBkTransparent(true)
	ele.EnableDrawBorder(false)
	// 整块是一个 Tab 站点: 链接级焦点(每个链接一个 Tab 站点)需要每个链接一个元素,
	// 与单元素自绘冲突 —— 进入后用 ←/→ 在条目间移动(见类型注释)。
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(true)
	xc.XEle_EnableDrawFocus(ele.Handle, false) // 原生焦点框灭掉, 自己画 focus-visible 环

	ele.AddEvent_Paint(g.onPaint)
	// 悬停条目判定: MouseStay 不带坐标, 用 MouseMove 做命中测试
	// (只在命中项变化时重绘, 见 iconTile 的同款处理)
	ele.AddEvent_MouseMove(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		g.setHover(g.hit(pPt.X, pPt.Y))
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		g.pressed = -1
		g.setHover(-1)
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标操作标记焦点来源, 保持 focus-visible 语义
		g.pressed = g.hit(pPt.X, pPt.Y)
		g.setHover(g.pressed)
		ele.Redraw(true)
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		idx := g.pressed
		g.pressed = -1
		// 按下与抬起必须落在同一个条目上才算点击(与按钮语义一致)
		if idx >= 0 && idx < len(g.items) && idx == g.hit(pPt.X, pPt.Y) {
			g.activate(g.items[idx])
			g.focus = idx // 键盘操作从这里接着走(浏览器里点过的链接也保持焦点)
		}
		// 延迟重绘(**不要**用立即重绘): OnChange 里调用方通常还要更新自己的
		// 文本/列表, 立即重绘这一个元素会把窗口的待重绘区域一起清掉, 调用方改的
		// 文字要等下一次无关重绘才出现(实测: demo 的页码说明滞后一帧)。
		ele.Redraw(false)
		return 0
	})
	// 焦点进出: 只在"进入"时把焦点放到当前页(没有则第一个可点条目), 并重绘焦点环。
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		if g.focus < 0 || g.focus >= len(g.items) {
			g.focusIntoView()
		}
		ele.Redraw(false)
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		ele.Redraw(false)
		return 0
	})
	xui.AttachWindow(ele.Handle)
	// 键盘处理必须挂在**窗口级**: 元素开了 EnableSwitchFocus(Tab 才能进入)后, xcgui
	// 会把 ←/→ 当成"切换焦点"并在元素 KeyDown 之前就执行 —— 探针实测元素回调里焦点
	// 已经落到下一个控件上, 拦不住。窗口 KeyDown 先于元素派发且此时焦点还正确,
	// 在这里按"当前焦点是不是本元素"分发、置 pbHandled 即可(实测焦点保持不动)。
	if hWindow := xc.XC_hWindowFromHWnd(xc.XWidget_GetHWND(ele.Handle)); hWindow != 0 {
		if w := window.NewByHandle(hWindow); w != nil {
			id := w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
				if xc.XWnd_GetFocusEle(hWin) != ele.Handle {
					return 0
				}
				return g.onKeyDown(wParam, pbHandled)
			})
			// 回调挂在**窗口**上, 元素先于窗口销毁时不会自动摘掉(会留下一个引用死元素的
			// 闭包, 每次按键都白跑一遍) —— 元素销毁时按回调 ID 反注册。注意元素的
			// XE_DESTROY 在窗口销毁**之后**才派发(见 DESIGN 第八节 54), 窗口已经没了
			// 就不能再拿句柄去反注册。
			ele.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
				if id > 0 {
					if xc.XC_IsHXCGUI(hWindow, xcc.XC_WINDOW) {
						w.RemoveEvent(xcc.WM_KEYDOWN, id)
					}
					id = 0
				}
				return 0
			})
		}
	}
	return g
}

// onKeyDown 处理键盘: ←/→(以及 ↑/↓) 在条目间移动焦点, Home/End 首末, Enter/Space 激活。
//
// 跳过省略号与到边界的禁用条目(同 hit 的可点击口径), 到底后不循环 —— 分页是有序
// 列表, 循环会让人分不清走到了哪一端。
func (g *Pagination) onKeyDown(wParam uintptr, pbHandled *bool) int {
	switch wParam {
	case 0x25, 0x27, 0x26, 0x28: // VK_LEFT / VK_RIGHT / VK_UP / VK_DOWN
		// ↑/↓ 也当"上一项/下一项": 分页只有一条横向的条目序列, 且方向键被框架默认
		// 用来切焦点(见调用点), 不拦就会跑到别的控件上。
		g.focusMove(wParam == 0x27 || wParam == 0x28)
	case 0x24, 0x23: // VK_HOME / VK_END
		g.focusMoveToEdge(wParam == 0x23)
	case 0x0D, 0x20: // VK_RETURN / VK_SPACE
		if g.focus >= 0 && g.focus < len(g.items) {
			g.activate(g.items[g.focus])
		}
	default:
		return 0
	}
	*pbHandled = true
	return 0
}

// focusIntoView 把键盘焦点放到当前页(没有页码条目时退回第一个可点条目)。
func (g *Pagination) focusIntoView() {
	g.refreshItems()
	for i := range g.items {
		if g.items[i].kind == pagPage && g.items[i].page == g.page {
			g.focus = i
			return
		}
	}
	for i := range g.items {
		if g.clickable(i) {
			g.focus = i
			return
		}
	}
	g.focus = -1
}

// focusMove 把焦点往左/往右挪一个可点条目(到端点后不动)。
func (g *Pagination) focusMove(right bool) {
	g.refreshItems()
	if g.focus < 0 || g.focus >= len(g.items) {
		g.focusIntoView()
		g.Redraw(false)
		return
	}
	step := 1
	if !right {
		step = -1
	}
	if i := pagSeek(len(g.items), g.focus+step, step, g.clickable); i >= 0 {
		g.focus = i
		g.Redraw(false)
	}
}

// focusMoveToEdge 把焦点挪到第一个/最后一个可点条目(Home/End 语义, 不是"当前页")。
func (g *Pagination) focusMoveToEdge(last bool) {
	g.refreshItems()
	from, step := 0, 1
	if last {
		from, step = len(g.items)-1, -1
	}
	if i := pagSeek(len(g.items), from, step, g.clickable); i >= 0 {
		g.focus = i
		g.Redraw(false)
	}
}

// pagSeek 从 from 起按 step 方向(±1)找第一个满足 ok 的下标, 走完没找到返回 -1。
// 抽成纯函数是为了能单测键盘的"跳过省略号/到边界导航项"逻辑(不需要真实元素)。
func pagSeek(n, from, step int, ok func(int) bool) int {
	if step == 0 || n <= 0 {
		return -1
	}
	for i := from; i >= 0 && i < n; i += step {
		if ok(i) {
			return i
		}
	}
	return -1
}

// refreshItems 在没有绘制过(刚创建就被 Tab 到)时先算一次条目表。
func (g *Pagination) refreshItems() {
	if len(g.items) > 0 {
		return
	}
	var rc xc.RECT
	xc.XEle_GetClientRect(g.Handle, &rc)
	g.build(rc)
}

// clickable 报告条目下标是否可点(省略号与到边界的上一页/下一页不可点)。
func (g *Pagination) clickable(i int) bool {
	if i < 0 || i >= len(g.items) {
		return false
	}
	it := g.items[i]
	pages := g.PageCount()
	if it.kind == pagEllipsis {
		return false
	}
	if it.kind == pagPrev && g.page <= 1 {
		return false
	}
	if it.kind == pagNext && g.page >= pages {
		return false
	}
	return true
}

// SetTotal 设置总条目数(0 也占一页)。
func (g *Pagination) SetTotal(total int) *Pagination {
	if total < 0 {
		total = 0
	}
	g.total = total
	g.clampPage()
	g.Redraw(false)
	return g
}

// SetPageSize 设置每页条目数(至少 1)。
func (g *Pagination) SetPageSize(size int) *Pagination {
	if size < 1 {
		size = 1
	}
	g.pageSize = size
	g.clampPage()
	g.Redraw(false)
	return g
}

// SetPage 设置当前页(1 起, 越界夹紧)。**不触发** OnChange —— 供调用方把
// 组件状态同步到自己的数据游标(OnChange 只响应鼠标点击)。
func (g *Pagination) SetPage(page int) *Pagination {
	g.page = page
	g.clampPage()
	g.Redraw(false)
	return g
}

// Page 返回当前页(1 起)。
func (g *Pagination) Page() int { return g.page }

// PageCount 返回总页数(至少 1)。
func (g *Pagination) PageCount() int {
	if g.pageSize < 1 {
		return 1
	}
	n := (g.total + g.pageSize - 1) / g.pageSize
	if n < 1 {
		n = 1
	}
	return n
}

// OnChange 注册页码变化回调(page 为 1 起的新页码; SetPage 不会触发)。
func (g *Pagination) OnChange(fn func(page int)) *Pagination {
	g.cb = fn
	return g
}

// setPage 切页: notify=true(鼠标点击)时触发 OnChange。
func (g *Pagination) setPage(page int, notify bool) {
	if page < 1 {
		page = 1
	}
	if n := g.PageCount(); page > n {
		page = n
	}
	if page == g.page {
		return
	}
	g.page = page
	// 延迟重绘: 见 LButtonUp 里的说明(调用方常要在 OnChange 里同步更新自己的界面)
	g.Element.Redraw(false)
	if notify && g.cb != nil {
		g.cb(page)
	}
}

// clampPage 把当前页夹回 [1, PageCount]。
func (g *Pagination) clampPage() {
	if n := g.PageCount(); g.page > n {
		g.page = n
	}
	if g.page < 1 {
		g.page = 1
	}
}

// activate 响应点击。
func (g *Pagination) activate(it pagItem) {
	switch it.kind {
	case pagPrev:
		g.setPage(g.page-1, true)
	case pagNext:
		g.setPage(g.page+1, true)
	case pagPage:
		g.setPage(it.page, true)
	}
}

// setHover 更新悬停条目(变化才重绘)。
func (g *Pagination) setHover(i int) {
	if g.hover == i {
		return
	}
	g.hover = i
	g.Element.Redraw(true)
}

// links 返回要显示的页码序列(0 表示省略号)。
//
// 三段式取自 shadcn 文档里 generatePagination 的写法(首段 / 中段 / 末段), 但**三段
// 都固定占 7 个槽位**: 官方那版首段 6 个、中段 7 个, 翻页跨段时整行会宽出一个页码位,
// 居中排布下"下一页"会跟着往右跳(实测反馈: 点着点着就点不到了)。槽位数恒定 → 条目
// 总宽恒定 → 每个条目的位置都不随页码变化(点击位置稳定)。
func (g *Pagination) links() []int {
	n := g.PageCount()
	if n <= pagMaxLinks {
		out := make([]int, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, i)
		}
		return out
	}
	p := g.page
	switch {
	case p <= 4:
		// [1][2][3][4][5][…][n]
		return []int{1, 2, 3, 4, 5, 0, n}
	case p >= n-3:
		// [1][…][n-4][n-3][n-2][n-1][n]
		return []int{1, 0, n - 4, n - 3, n - 2, n - 1, n}
	default:
		// [1][…][p-1][p][p+1][…][n]
		return []int{1, 0, p - 1, p, p + 1, 0, n}
	}
}

// hit 返回客户区内 (x,y) 命中的条目下标, 未命中(或落在不可点击条目上)返回 -1。
func (g *Pagination) hit(x, y int32) int {
	var rc xc.RECT
	xc.XEle_GetClientRect(g.Handle, &rc)
	g.build(rc)
	for i := range g.items {
		if !g.clickable(i) {
			continue // 省略号(同 shadcn 的 sr-only span)/ 到边界的上一页下一页
		}
		it := g.items[i]
		if x >= it.rc.Left && x < it.rc.Right && y >= it.rc.Top && y < it.rc.Bottom {
			return i
		}
	}
	return -1
}

// sequence 返回条目序列(只有 kind/page, 没有矩形), 供 build 与单测共用。
//
// 完整 = [上一页][1][2]…[n][下一页] / 简洁 = [1][2]…[n] / 仅导航 = [上一页][下一页]。
func (g *Pagination) sequence() []pagItem {
	var out []pagItem
	if g.mode != pagModeSimple {
		out = append(out, pagItem{kind: pagPrev})
	}
	if g.mode != pagModeNav {
		for _, v := range g.links() {
			if v == 0 {
				out = append(out, pagItem{kind: pagEllipsis})
				continue
			}
			out = append(out, pagItem{kind: pagPage, page: v})
		}
	}
	if g.mode != pagModeSimple {
		out = append(out, pagItem{kind: pagNext})
	}
	return out
}

// build 计算条目矩形(绘制与命中测试共用)。origin 是元素客户区矩形。
func (g *Pagination) build(origin xc.RECT) {
	g.refresh()
	g.items = append(g.items[:0], g.sequence()...)

	// 宽度: 页码/省略号是 32 方形, 上一页/下一页 = 内边距 + 图标 + 间距 + 文字 + 内边距
	prevW, nextW := g.prevW, g.nextW
	if g.hideText {
		prevW, nextW = 0, 0
	}
	widths := make([]int32, len(g.items))
	content := pagGap * int32(len(g.items)-1)
	for i := range g.items {
		w := pagLinkH
		switch g.items[i].kind {
		case pagPrev:
			w = pagPadX + pagIcon + prevW + pagPadX
			if prevW > 0 {
				w += pagIconGap
			}
			if g.hideText {
				w = pagLinkH // 文字隐藏时 = 32 方块(上游 pl-1.5/pr-2.5 + size-4 图标)
			}
		case pagNext:
			w = pagPadX + nextW + pagIcon + pagPadX
			if nextW > 0 {
				w += pagIconGap
			}
			if g.hideText {
				w = pagLinkH
			}
		}
		widths[i] = w
		content += w
	}

	// 定位: 内容整体水平居中(justify-center), 垂直居中
	itemH := pagLinkH + pagPressShift
	h := origin.Bottom - origin.Top
	x := origin.Left + (origin.Right-origin.Left-content)/2
	if x < origin.Left {
		x = origin.Left // 放不下时左对齐(窄容器)
	}
	y := origin.Top + (h-itemH)/2
	for i := range g.items {
		g.items[i].rc = xc.RECT{Left: x, Top: y, Right: x + widths[i], Bottom: y + itemH}
		x += widths[i] + pagGap
	}

	g.normalizeFocus()
}

// normalizeFocus 在条目表重建后收敛焦点下标: 页码变化的折叠可能让它越界, 或者
// 落到省略号/到边界的导航条目上。清零后下一次按方向键会重新落到当前页。
func (g *Pagination) normalizeFocus() {
	if g.focus >= 0 && (g.focus >= len(g.items) || !g.clickable(g.focus)) {
		g.focus = -1
	}
}

// refresh 在字体变化时重新量上一页/下一页的文字宽度。
func (g *Pagination) refresh() {
	f := xui.FontHandle(xui.Current().FontDefault) // 链接文字 = text-sm = 主题字体
	if f == g.wFont {
		return
	}
	g.wFont = f
	g.prevW = pagTextWidth(g.prevText, f)
	g.nextW = pagTextWidth(g.nextText, f)
}

// pagTextWidth 量一段文字的逻辑宽度(单行)。
//
// 坑(探针实测): `app.GetTextShowRect` 的 width 参数传 0 时**不是"不限宽"**,
// 而是按一个字符宽折行 —— 返回的 CX 只是单行宽度、CY 变成行数×行高(3 个汉字 = CX 15/CY 59)。
// 所以量宽必须给足 width 并加 NoWrap, 此时 CX 就是真实逻辑宽度(汉字 15px/字 @14px 字号),
// 不需要任何倍率校正。
func pagTextWidth(text string, hFont int) int32 {
	return xui.TextWidth(text, hFont) // 量宽口径收在运行时(见 xui/text.go)
}

// pagItemKind 是条目类型。
type pagItemKind int

const (
	pagPrev pagItemKind = iota
	pagPage
	pagEllipsis
	pagNext
)

// pagItem 是分页条目: kind 决定绘制与可点击性, rc 是命中矩形(含按下位移预留的全高)。
type pagItem struct {
	kind pagItemKind
	page int // 1 起(pagPage)
	rc   xc.RECT
}

func (g *Pagination) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()

	// 先铺宿主底色: 本组件只画局部形状(悬停底色/描边), 不铺底的话上一帧的像素会残留
	// (透明元素重绘不擦底, 见 DESIGN 第八节 28)。默认用 Card —— 分页通常放在卡片上。
	p.FillRound(&p.Rc, t.Card, 0)

	g.build(p.Rc)
	fontH := xui.FontHandle(t.FontDefault)
	if fontH != 0 {
		p.Draw.SetFont(fontH)
	}
	// focus-visible: 只有键盘获得的焦点才画环(鼠标点击不画, 同 shadcn 的 :focus-visible)。
	// 环画在**当前条目**上而不是整块元素上 —— 分页是多链接并列, 整块环会看不出是哪一项。
	focusedByKey := xui.StateOf(hEle).Focused && xui.FocusedByKeyboard()
	pages := g.PageCount()

	for i := range g.items {
		it := g.items[i]
		hv := xui.HoverItem(hEle, int32(i), i == g.hover)
		// 按下 active:translate-y-px: 壳体下沉; 内容跟随壳体在**物理像素**上的实际
		// 位移(1 逻辑像素在 150% DPI 下会被取整成 1 或 2 物理像素, 文字与图标各自
		// 取整相位不同, 会显得没一起动, 见 button.go 的 shellPressShift / DESIGN 88)。
		baseShell := it.rc
		baseShell.Bottom -= pagPressShift
		shell := baseShell
		contentDy := int32(0)
		if i == g.pressed {
			shell.Top += pagPressShift
			shell.Bottom += pagPressShift
			contentDy = shellPressShift(p, baseShell, shell)
		}

		// 焦点项: 先铺 3px 环带, 内芯再铺回宿主底色(同 Button 的画法), 本体视觉
		// 画在内芯上 —— 直接整块铺环色会盖掉条目自己的底色/描边。
		bgRc := shell
		if focusedByKey && i == g.focus {
			p.FillRound(&shell, t.Ring.WithAlpha(128), t.RadiusLg())
			inner := pagInset(shell, 3)
			if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
				p.FillRound(&inner, t.Card, t.RadiusLg()-3)
				bgRc = inner
			}
		}
		radius := t.RadiusLg()
		if bgRc != shell {
			radius -= 3
		}

		switch it.kind {
		case pagEllipsis:
			// size-8 + more-horizontal(muted-foreground), 无悬停反馈
			g.drawIcon(p, icons.IconMoreHorizontal, baseShell, t.MutedForeground, contentDy)

		case pagPage:
			if it.page == g.page {
				// 当前页 = outline 变体(bg-background + border-border hover:bg-muted)
				p.FillRound(&bgRc, xui.Fade(t.Background, t.Muted, hv), radius)
				p.StrokeRound(&bgRc, t.Border, radius, 1)
			} else if bg := xui.Fade(xui.None, t.Muted, hv); !bg.IsNone() {
				// 其它页 = ghost 变体(悬停才有底色)
				p.FillRound(&bgRc, bg, radius)
			}
			g.drawText(p, strconv.Itoa(it.page), baseShell, t.Foreground, xcc.TextAlignFlag_Center, pagPadIn, contentDy)

		case pagPrev, pagNext:
			dis := (it.kind == pagPrev && g.page <= 1) || (it.kind == pagNext && g.page >= pages)
			if !dis {
				if bg := xui.Fade(xui.None, t.Muted, hv); !bg.IsNone() {
					p.FillRound(&bgRc, bg, radius)
				}
			}
			// 到边界: 按禁用绘制(50% 透明, 不响应鼠标)
			col := t.Foreground
			if dis {
				col = t.Foreground.WithAlpha(128)
			}
			midY := (baseShell.Top + baseShell.Bottom) / 2
			midX := (baseShell.Left + baseShell.Right) / 2
			if it.kind == pagPrev {
				iconRc := xc.RECT{
					Left:   baseShell.Left + pagPadX,
					Top:    midY - pagIcon/2,
					Right:  baseShell.Left + pagPadX + pagIcon,
					Bottom: midY + pagIcon/2,
				}
				if g.hideText {
					// 只留 chevron: 图标在整个链接里居中(而不是贴左)
					iconRc = xc.RECT{
						Left:   midX - pagIcon/2,
						Top:    midY - pagIcon/2,
						Right:  midX + pagIcon/2,
						Bottom: midY + pagIcon/2,
					}
				}
				p.IconRotatedOffset(icons.IconChevronLeft, &iconRc, col, 2, 0, 0, contentDy)
				// 文字矩形右边界取到壳体右缘: 量宽偏小时也不会截字
				// (尾部留给 pr-1.5 的余量, 文字左对齐所以观感不变)
				if !g.hideText { // 只留 chevron 时不能画文字: 退化的矩形会留下一条竖线残影
					textRc := xc.RECT{
						Left:   iconRc.Right + pagIconGap,
						Top:    baseShell.Top,
						Right:  baseShell.Right,
						Bottom: baseShell.Bottom,
					}
					g.drawText(p, g.prevText, textRc, col, xcc.TextAlignFlag_Left, 0, contentDy)
				}
			} else {
				iconRc := xc.RECT{
					Left:   baseShell.Right - pagPadX - pagIcon,
					Top:    midY - pagIcon/2,
					Right:  baseShell.Right - pagPadX,
					Bottom: midY + pagIcon/2,
				}
				if g.hideText {
					iconRc = xc.RECT{
						Left:   midX - pagIcon/2,
						Top:    midY - pagIcon/2,
						Right:  midX + pagIcon/2,
						Bottom: midY + pagIcon/2,
					}
				}
				p.IconRotatedOffset(icons.IconChevronRight, &iconRc, col, 2, 0, 0, contentDy)
				// 同上: 左边界取到壳体左缘, 右对齐文字不会被截头
				if !g.hideText {
					textRc := xc.RECT{
						Left:   baseShell.Left,
						Top:    baseShell.Top,
						Right:  iconRc.Left - pagIconGap,
						Bottom: baseShell.Bottom,
					}
					g.drawText(p, g.nextText, textRc, col, xcc.TextAlignFlag_Right, 0, contentDy)
				}
			}
		}
	}
	return 0
}

// pagInset 把矩形四边各内缩 d。
func pagInset(rc xc.RECT, d int32) xc.RECT {
	rc.Left += d
	rc.Top += d
	rc.Right -= d
	rc.Bottom -= d
	return rc
}

// drawText 在 rc 内水平方向按 align 对齐、垂直居中绘制一段文字, 左右各内缩 inset。
// dy 是按下时内容跟随壳体的**物理像素**位移(见 shellPressShift), 用 TextF 承载,
// 避免 1 逻辑像素在 150% DPI 下被取整到与图标不同的物理行。
func (g *Pagination) drawText(p *xui.Painter, text string, rc xc.RECT, color xui.Color,
	align xcc.TextFormatFlag_, inset, dy int32) {
	if text == "" {
		return
	}
	rc.Left += inset
	rc.Right -= inset
	scale := p.Scale
	if scale <= 0 {
		scale = 1
	}
	dyF := float32(dy) / float32(scale)
	dyF -= textVDy(text)
	rcF := xc.RECTF{
		Left:   float32(rc.Left),
		Top:    float32(rc.Top) + dyF,
		Right:  float32(rc.Right),
		Bottom: float32(rc.Bottom) + dyF,
	}
	p.TextF(text, &rcF, color,
		align|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
}

// drawIcon 在 rc 内居中绘制一枚 size-4 图标, dy 语义同 drawText(物理像素)。
func (g *Pagination) drawIcon(p *xui.Painter, path string, rc xc.RECT, color xui.Color, dy int32) {
	midX := (rc.Left + rc.Right) / 2
	midY := (rc.Top + rc.Bottom) / 2
	iconRc := xc.RECT{
		Left:   midX - pagIcon/2,
		Top:    midY - pagIcon/2,
		Right:  midX + pagIcon/2,
		Bottom: midY + pagIcon/2,
	}
	p.IconRotatedOffset(path, &iconRc, color, 2, 0, 0, dy)
}
