package ui

import (
	"strings"

	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/drawx"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Menu 下拉菜单: 载体为原生 Menu(弹出/键盘导航/子菜单/边缘翻转全部原生),
// 皮肤通过窗口级 Menu_DrawBackground / Menu_DrawItem 事件接管 —— 所有颜色
// 绘制时实时取令牌, 换主题零成本(同一窗口弹出的所有菜单共用这套皮肤)。
//
// 与 shadcn dropdown-menu.tsx(radix-nova) 对齐:
//   - 弹层: bg-popover rounded-lg p-1 ring-1 ring-foreground/10 shadow-md。
//     载体是阴影弹窗(Window_Transparent_Shadow, 官方 DrawMenu 示例同款路线):
//     圆角(ShadowInfo 的 nAngeleSize)、投影、整窗 alpha(进场淡入)一次解决,
//     不再用 Win32 区域裁剪; p-1 用菜单 SetBorderSize(4) 落实(默认 3);
//   - 进场动画: data-open:fade-in-0(duration-100), 整窗 alpha 由 xui.Presence
//     挂宿主窗口驱动; 无位移(slide-in 会让 demo 崩溃, 见 menuOnPopupWnd 注释);
//     退场原生即关即销毁, 无退场动画;
//   - 条目: h-7(28) rounded-md px-1.5, hover/focus 高亮 bg-accent + 文字变
//     accent-foreground; destructive 文字红、高亮 bg-destructive/10(dark /20);
//   - 禁用: opacity-50(组级透明 → 文字用 50% alpha 合成, 且不响应);
//   - 勾选/单选条目: 选中项右侧 Check 图标(right-2 size-4), 单选组内互斥;
//   - 分组标签: text-xs muted; 分隔栏: 1px bg-border 全宽(-mx-1; 原生槽高仅
//     4px, shadcn my-1 的 9px 不可达, 已知差异);
//   - 快捷键: 右对齐 text-xs muted(借助原生 "\t" 解析预留宽度);
//   - 宽度: 原生按内容自适应(shadcn 是 trigger 宽 / min-w-32; XMenu_SetItemWidth
//     实测不改变布局, 触发区对齐宽度不可达, 已知差异);
//   - 定位: 触发元素下方左对齐 sideOffset=4(radix Content 默认), 下方放不下
//     翻到上方, 屏幕边缘夹紧。
//
// 用法(一次性对象, 每次弹出新建):
//
//	m := ui.NewMenu()
//	m.Item("复制", ui.MenuShortcut("Ctrl+C"), ui.MenuOnSelect(func() {...}))
//	m.Check("显示行号", checked, ui.MenuOnCheck(func(on bool) {...}))
//	m.PopupAtElement(btn.Handle) // 或 m.PopupAtCursor(w.Handle)
type Menu struct {
	m      *widget.Menu
	items  []*menuItem
	byID   map[int32]*menuItem
	nextID int32
	parent int32 // Sub() 构建期间的父项 ID

	// 弹出锚点(物理屏幕像素); 翻转/夹紧由原生菜单自带, 不需也不应手动校正
	ax, yBelow int32
}

type menuKind int8

const (
	menuKindNormal menuKind = iota
	menuKindCheck
	menuKindRadio
	menuKindLabel
	menuKindSeparator
	menuKindSub
)

type menuItem struct {
	id          int32
	text        string
	shortcut    string
	kind        menuKind
	parent      int32
	checked     bool
	group       int32
	disabled    bool
	destructive bool
	inset       bool // 文字左缩进到与勾选条目对齐(shadcn 的 data-inset:pl-8)
	onSelect    func()
	onCheck     func(checked bool)
}

// MenuOpt 菜单项配置。
type MenuOpt func(*menuItem)

// MenuShortcut 菜单项右侧的快捷键文本。
func MenuShortcut(s string) MenuOpt { return func(it *menuItem) { it.shortcut = s } }

// MenuDisabled 禁用该项(灰显且不可点)。
func MenuDisabled() MenuOpt { return func(it *menuItem) { it.disabled = true } }

// MenuDestructive 破坏性操作(红字, 高亮为红色淡底)。
func MenuDestructive() MenuOpt { return func(it *menuItem) { it.destructive = true } }

// MenuGroup 单选组号: 同组 Radio 条目选中互斥(不设组号时单选条目独立切换)。
func MenuGroup(group int32) MenuOpt { return func(it *menuItem) { it.group = group } }

// MenuInset 让条目文字左缩进到与勾选/单选条目对齐(shadcn dropdown-menu 的 `inset`
// prop = `data-inset:pl-8`): 勾选条目的文字要为右侧 Check 之外的左侧留位, 不勾选的
// 普通条目用 inset 才不会看起来往左凸出去。绘制只挪文字, 高亮底仍是整行。
func MenuInset() MenuOpt { return func(it *menuItem) { it.inset = true } }

// MenuOnSelect 普通条目的点击回调。
func MenuOnSelect(fn func()) MenuOpt { return func(it *menuItem) { it.onSelect = fn } }

// MenuOnCheck 勾选/单选条目的状态回调, 参数为点击后的新状态。
func MenuOnCheck(fn func(checked bool)) MenuOpt { return func(it *menuItem) { it.onCheck = fn } }

// NewMenu 创建下拉菜单。菜单是一次性对象: 弹出关闭后原生菜单即销毁,
// 状态(如勾选)由调用方持有并在下次弹出时传入。
func NewMenu() *Menu {
	return &Menu{byID: map[int32]*menuItem{}, nextID: 1}
}

func (m *Menu) add(it *menuItem, opts []MenuOpt) *Menu {
	for _, o := range opts {
		o(it)
	}
	it.id = m.nextID
	m.nextID++
	it.parent = m.parent
	m.items = append(m.items, it)
	m.byID[it.id] = it
	return m
}

// Item 添加普通菜单项。
func (m *Menu) Item(text string, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindNormal, text: text}, opts)
}

// Check 添加勾选菜单项: 点击后状态翻转, 新状态通过 MenuOnCheck 回调告知。
func (m *Menu) Check(text string, checked bool, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindCheck, text: text, checked: checked}, opts)
}

// Radio 添加单选菜单项: 与同组(MenuGroup)条目互斥, 行为同 shadcn RadioItem
// (视觉上同样是右侧 Check 图标)。组号不同或未设组号时独立切换。
func (m *Menu) Radio(text string, checked bool, opts ...MenuOpt) *Menu {
	return m.add(&menuItem{kind: menuKindRadio, text: text, checked: checked}, opts)
}

// Label 添加分组标签(不可聚焦、不可点)。
func (m *Menu) Label(text string) *Menu {
	return m.add(&menuItem{kind: menuKindLabel, text: text, disabled: true}, nil)
}

// Separator 添加分隔栏。
func (m *Menu) Separator() *Menu {
	return m.add(&menuItem{kind: menuKindSeparator}, nil)
}

// Sub 添加子菜单, build 里添加的条目都挂在该子菜单下(可嵌套)。
func (m *Menu) Sub(text string, build func(s *Menu)) *Menu {
	p := &menuItem{kind: menuKindSub, text: text}
	m.add(p, nil)
	saved := m.parent
	m.parent = p.id
	build(m)
	m.parent = saved
	return m
}

// PopupAtElement 在触发元素下方弹出(radix DropdownMenuContent 默认定位:
// align=start, sideOffset=4), 下方放不下翻到上方, 屏幕边缘夹紧。
func (m *Menu) PopupAtElement(hEle int) {
	HideTooltip(hEle) // 弹层接管鼠标后触发元素收不到 MouseLeave(见 tooltip.go)
	var trig xc.RECT
	xc.XEle_GetWndClientRect(hEle, &trig)
	hw := xc.XWidget_GetHWINDOW(hEle)
	if hw <= 0 {
		return
	}
	dpi := xc.XWnd_GetDPI(hw)
	if dpi <= 0 {
		dpi = 96
	}
	sc := float64(dpi) / 96
	var mrc xc.RECT
	if !wapi.GetWindowRect(xc.XWnd_GetHWND(hw), &mrc) {
		return
	}
	const sideOffset = 4
	m.ax = mrc.Left + int32(float64(trig.Left)*sc)
	m.yBelow = mrc.Top + int32(float64(trig.Bottom)*sc) + int32(sideOffset*sc)
	m.popup(hw)
}

// PopupAtCursor 在鼠标位置弹出(右键菜单用法)。hWindow 是承载菜单事件的
// 炫彩窗口句柄 —— 皮肤绘制事件注册在它上面。
func (m *Menu) PopupAtCursor(hWindow int) {
	if hWindow <= 0 {
		return
	}
	var pt wapi.POINT
	wapi.GetCursorPos(&pt)
	m.ax, m.yBelow = pt.X, pt.Y
	m.popup(hWindow)
}

func (m *Menu) popup(hWindow int) {
	m.m = m.build(hWindow)
	// 初始位置即锚点(元素矩形是精确的)。
	// ⚠️ 不要在这里做任何位置校正: XMenu_Popup 是阻塞的(菜单关闭后才返回),
	// 校正会拿到已销毁的弹窗句柄(报「输入句柄可能无效」)。
	// Menu_PopupWnd 回调里的窗口矩形外扩/进场动画位移是官方 DrawMenu 示例
	// 同款模式, 探针实测不影响子菜单锚点; 但"把弹窗挪去别的位置"这类校正
	// 仍然不要做(历史踩坑: 二级菜单弹到父菜单原点盖住它)。
	// 翻转/夹紧/子菜单定位原生自带, 锚点给对即可。
	m.m.Popup(xc.XWnd_GetHWND(hWindow), m.ax, m.yBelow, 0, xcc.Menu_Popup_Position_Left_Top)
}

// build 创建原生菜单并同步条目。
// 快捷键拼进文本("\t" 分隔): 原生按整串计算自适应宽度并为快捷键预留区域
// (探针实测 NShortcutKeyWidth=37), 绘制时再拆开分别渲染。
func (m *Menu) build(hWindow int) *widget.Menu {
	menu := widget.NewMenu()
	menu.SetAutoDestroy(true)
	menu.SetItemHeight(28) // 条目 = text-sm(20) + py-1(4×2)
	menu.SetLeftWidth(0)   // shadcn 条目没有左侧图标条, 图标(勾)画在右侧
	// p-1(4px): 默认边距是 3, 条目高亮距弹层左右边缘原差 1px; 设 4 后条目盒
	// 与 shadcn 完全一致(内容内缩 4, 文字再 px-1.5)。
	menu.SetBorderSize(4, 4, 4, 4)
	menu.EnableDrawBackground(true)
	menu.EnableDrawItem(true)
	for _, it := range m.items {
		text := it.text
		if it.shortcut != "" {
			text += "\t" + it.shortcut
		}
		flags := xcc.Menu_Item_Flag_Normal
		switch {
		case it.kind == menuKindSeparator:
			flags = xcc.Menu_Item_Flag_Separator
		case it.disabled: // Label 借用禁用态: 不高亮、不可点
			flags = xcc.Menu_Item_Flag_Disable
		case (it.kind == menuKindCheck || it.kind == menuKindRadio) && it.checked:
			flags = xcc.Menu_Item_Flag_Check
		}
		menu.AddItem(it.id, text, it.parent, flags)
	}
	ensureMenuSkin(hWindow)
	menuSkin.active = m
	menu.AddEvent_Menu_Select(hWindow, func(_ int, nID int32, _ *bool) int {
		m.dispatch(nID)
		return 0
	}, false)
	menu.AddEvent_Menu_Exit(hWindow, func(_ int, _ *bool) int {
		if menuSkin.active == m {
			menuSkin.active = nil
		}
		return 0
	}, false)
	return menu
}

// dispatch 分发选择事件: Check 翻转, Radio 置真并清同组, 其余走 onSelect。
// 菜单关闭后原生对象销毁, 状态只保留在 Go 侧条目上, 由调用方回调承接。
func (m *Menu) dispatch(nID int32) {
	it := m.byID[nID]
	if it == nil {
		return
	}
	switch it.kind {
	case menuKindCheck:
		it.checked = !it.checked
		if it.onCheck != nil {
			it.onCheck(it.checked)
		}
	case menuKindRadio:
		it.checked = true
		for _, other := range m.items {
			if other != it && other.kind == menuKindRadio && other.group == it.group {
				other.checked = false
			}
		}
		if it.onCheck != nil {
			it.onCheck(true)
		}
	default:
		if it.onSelect != nil {
			it.onSelect()
		}
	}
}

// ===== 皮肤(窗口级, 同一窗口弹出的所有菜单共用) =====

var menuSkin = struct {
	windows map[int]bool
	active  *Menu
}{windows: map[int]bool{}}

// ensureMenuSkin 在承载窗口上注册菜单绘制事件(每窗口一次)。
// 绘制回调全部实时取令牌, 换主题后下一次弹出自动是新皮肤。
func ensureMenuSkin(hWindow int) {
	if menuSkin.windows[hWindow] {
		return
	}
	w := window.NewByHandle(hWindow)
	if w == nil {
		return
	}
	menuSkin.windows[hWindow] = true
	w.AddEvent_Menu_PopupWnd(menuOnPopupWnd)
	w.AddEvent_Menu_DrawBackground(menuDrawBackground)
	w.AddEvent_Menu_DrawItem(menuDrawItem)
	w.AddEvent_Menu_Exit(func(_ int, _ *bool) int {
		menuSkin.active = nil
		return 0
	})
}

// menuShadowSize 是顶级菜单弹窗阴影带的宽度(逻辑像素) —— Window_Transparent_Shadow
// 会把客户区四周各收缩这么多, 窗口矩形要同步外扩让 Body(菜单内容区)回到原位。
// 取 5 对齐 shadcn shadow-md(0 4px 6px -1px black/10%: 侧向可见约 5px)。
const menuShadowSize int32 = 5

// menuShadowSizeSub 是子菜单弹窗阴影带宽: shadcn 子菜单用 shadow-lg
// (0 10px 15px -3px black/10%, 侧向可见约 12px), 均匀环带引擎下取 8 折中。
const menuShadowSizeSub int32 = 8

// menuShadowDepth 是阴影深度(0-255), 与 shadow-md/lg 的 10% 峰值黑度对齐
// (阴影引擎在白底上的边缘浓度约 depth/1000)。
const menuShadowDepth int32 = 50

// menuOnPopupWnd 弹窗出现: 转阴影弹窗(圆角+投影+可整窗 alpha), 顶级弹窗播
// 进场淡入, 子菜单即时出现(见下方说明)。
// 窗口矩形外扩在本回调里做(官方 DrawMenu 示例同款), 探针实测子菜单锚点不受影响。
//
// ⚠️ 只有淡入, 没有 radix 的 slide-in 位移: 对菜单弹窗做 SetWindowPos 移动
// (哪怕只在进场那一瞬、SWP_NOSIZE|NOZORDER|NOACTIVATE)会让 demo 直接崩溃
// (DLL 内部访问违例, 静默退出; 官方示例的 SetRect 外扩没事, 单独的"移动"不行,
// 探针+demo 双向验证)。100ms 的纯淡入与 radix fade-in-0 观感一致, 位移省略。
func menuOnPopupWnd(hWindow, hMenu int, pInfo *xc.Menu_PopupWnd_, pbHandled *bool) int {
	if pInfo == nil || pInfo.HWindow <= 0 {
		return 0
	}
	mw := window.NewByHandle(pInfo.HWindow)
	if mw == nil {
		return 0
	}
	t := xui.Current()

	// 阴影弹窗: nAngeleSize = rounded-lg, 圆角由阴影窗口自身按逐像素 alpha 画出
	// (背景/条目照常填充, 框架自动裁到圆角 Body), 不再需要 SetWindowRound 区域裁剪。
	// 阴影带宽: 顶级 = shadow-md(5), 子菜单 = shadow-lg(8), 对齐 shadcn。
	band := menuShadowSize
	if pInfo.NParentID != 0 {
		band = menuShadowSizeSub
	}
	mw.SetTransparentType(xcc.Window_Transparent_Shadow)
	mw.SetShadowInfo(band, menuShadowDepth, t.RadiusLg(), false, xc.RGBA(0, 0, 0, 128))
	mw.SetTransparentAlpha(255)
	// 客户区已收缩出阴影带: 窗口矩形四周外扩同尺寸, Body 对齐原生给定的菜单位置。
	rc := mw.GetRectEx()
	mw.SetRect(&xc.RECT{
		Left:   rc.Left - band,
		Top:    rc.Top - band,
		Right:  rc.Right + band,
		Bottom: rc.Bottom + band,
	})

	// 进场淡入(fade-in, 100ms):
	//   - 整窗 alpha 只在透明类型窗口上有效(阴影窗是), 且永远不能精确 = 0
	//     (0 会让 xcgui 停止合成该窗口), 先压到下限防首帧闪现;
	//   - 过渡挂宿主窗口驱动 —— 弹层小窗自己做重绘目标时定时器掉帧(DESIGN 47);
	//   - ⚠️ apply 每帧必须先验证弹窗还活着: 菜单可能在过渡途中被关闭(弹窗
	//     随之销毁), 拿失效句柄调 X API 会让整个进程静默崩溃(demo 实测)。
	//     官方 DrawMenu 示例在 KillFocus 里同样用 XC_IsHXCGUI 做守卫。
	//
	// 子菜单(NParentID != 0)不做淡入: 悬停到展开本身有 xcgui 固化的 ~330ms
	// 内部延迟(纯净菜单探针实测同值, 无配置 API, 输入注入也无法提前触发),
	// 再叠 100ms 动画会明显比 shadcn 慢一截 —— 子菜单直接全 alpha 出现。
	if pInfo.NParentID == 0 {
		xui.SetWindowAlpha(pInfo.HWindow, xui.AlphaFloor())
		// ⚠️ 必须拷贝句柄值而不是捕获 pInfo 指针: 该结构体是原生栈对象, 回调返回后
		// 会被复用/清空, 过渡帧里再解引用拿到的是 0 或垃圾(0 → 淡入永远不动,
		// 垃圾 → 拿无效句柄调 X API 直接崩进程, demo 实测)。
		hw := pInfo.HWindow
		// 键取全局唯一(不要用本组件自己的计数器: 各组件都从 1 开始会撞车, 见
		// xui.NextPresenceKey 的说明 —— 撞上已完成的过渡时新菜单窗会停在 alpha 下限)。
		key := xui.NextPresenceKey()
		xui.Presence(hWindow, key, true, xui.PopupDuration, func(v float64) {
			// 菜单可能在过渡途中被关闭(弹窗销毁): 剩余帧必须跳过。
			// 官方 DrawMenu 示例在 KillFocus 里同样用 XC_IsHXCGUI 做守卫。
			if !xc.XC_IsHXCGUI(hw, xcc.XC_WINDOW) {
				return
			}
			xui.SetWindowAlpha(hw, v)
			xc.XWnd_Redraw(hw, true)
		}, nil)
	}
	return 0
}

// menuDrawBackground 弹层底色 + ring-1 ring-foreground/10。
//
// 坐标系(标定实验实测): 绘制上下文原点 = 弹窗窗口左上角(逻辑像素),
// XWnd_GetClientRect 返回的 {8,8,168,276} 就是阴影窗的 Body(可见白区)
// 在窗口内的矩形 —— 它与窗口矩形差一圈阴影带, 但 fill/ring 都画在它上面
// 就是正确的(探针色块标定验证)。ring 直接画在 Body 边界上: 线宽的一半
// 落在体外被阴影 alpha 蒙版裁掉, 肉眼是一条贴着窗口边缘的线(shadcn 边框
// 贴边), 圆角与 Body 的 nAngeleSize(RadiusLg) 一致, 四角不脱开。
// ⚠️ 画在 Body 矩形之外(阴影带里)的任何像素都会被蒙版压暗至不可见。
func menuDrawBackground(_ int, hDraw int, pInfo *xc.Menu_DrawBackground_, pbHandled *bool) int {
	*pbHandled = true
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return 0
	}
	d.EnableSmoothingMode(true)
	t := xui.Current()
	var rc xc.RECT
	xc.XWnd_GetClientRect(pInfo.HWindow, &rc)
	d.SetBrushColor(uint32(t.Popover))
	d.FillRect(&rc)
	d.SetLineWidth(1)
	d.SetBrushColor(uint32(t.Foreground.WithAlpha(26)))
	d.DrawRoundRect(&rc, t.RadiusLg(), t.RadiusLg())
	return 0
}

// menuDrawItem 自绘菜单项。
// 项矩形几何(SetBorderSize(4) 后探针实测): 四周各 4px 边距(= shadcn p-1),
// 高度即 SetItemHeight(28), 分隔栏槽高固定 4px。
func menuDrawItem(_ int, hDraw int, pInfo *xc.Menu_DrawItem_, pbHandled *bool) int {
	*pbHandled = true
	d := drawx.NewByHandle(hDraw)
	if d == nil {
		return 0
	}
	d.EnableSmoothingMode(true)
	p := &xui.Painter{Draw: d}
	t := xui.Current()
	st := pInfo.NState
	rc := pInfo.RcItem

	var it *menuItem
	if am := menuSkin.active; am != nil {
		it = am.byID[pInfo.NID]
	}

	// 分隔栏: 1px bg-border 全宽(shadcn -mx-1: 越过 p-1 边距铺满弹层),
	// 居中画在原生固定的 4px 槽内
	if st&xcc.MenuItem_State_Flag_Separator > 0 {
		d.SetLineWidth(1)
		d.SetBrushColor(uint32(t.Border))
		d.DrawLine(rc.Left-4, rc.Top+2, rc.Right+4, rc.Top+2)
		return 0
	}

	disabled := st&xcc.MenuItem_State_Flag_Disable > 0
	hot := st&xcc.MenuItem_State_Flag_Stay > 0

	// 文字与高亮配色(dropdown-menu.tsx):
	//   普通: text-popover-foreground, hot: bg-accent + text-accent-foreground
	//   destructive: text-destructive, hot: bg-destructive/10(dark /20), 文字不变
	//   disabled: opacity-50(组级透明), 不参与高亮
	//   Label: text-xs muted, 永不高亮(原生已置禁用)
	fore := t.PopoverForeground
	hotBg := t.Accent
	isDestructive := it != nil && it.destructive
	if isDestructive {
		fore = t.Destructive
		hotBg = t.Destructive.WithAlpha(26)
		if t.IsDark() {
			hotBg = t.Destructive.WithAlpha(51)
		}
	}
	if disabled {
		fore = fore.WithAlpha(128)
		hot = false
	}
	isLabel := it != nil && it.kind == menuKindLabel
	// 悬停底与字色按过渡值淡入淡出。载体是原生菜单弹窗(pInfo.HWindow): 鼠标在条目
	// 间移动时原生自己会重绘, 过渡值由 xui 按"窗口 + 条目 ID"保管并驱动补帧。
	hv := xui.HoverItem(pInfo.HWindow, pInfo.NID, hot && !isLabel)
	if hv > 0 {
		p.FillRound(&rc, hotBg.ScaleAlpha(hv), t.RadiusSm())
		if !isDestructive {
			fore = xui.Fade(fore, t.AccentForeground, hv)
		}
	}

	name, shortcut := menuItemText(it, pInfo)
	hasIcon := it != nil && ((it.kind == menuKindCheck || it.kind == menuKindRadio) && it.checked ||
		it.kind == menuKindSub)

	// 文本: px-1.5; 右侧有图标(勾/子菜单箭头)时按 pr-8 让位;
	// inset 条目按 shadcn 的 `data-inset:pl-7` 左缘 = 条目矩形 + 28(shadcn 用它把
	// 普通条目与"带前置图标/勾选"的条目文字对齐)。
	textRc := rc
	textRc.Left += 6
	if it != nil && it.inset {
		textRc.Left = rc.Left + 28
	}
	textRc.Right -= 6
	if hasIcon {
		textRc.Right -= 24
	}
	if isLabel {
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		d.SetBrushColor(uint32(t.MutedForeground))
	} else {
		d.SetFont(xui.FontHandle(t.FontDefault))
		d.SetBrushColor(uint32(fore))
	}
	d.SetTextAlign(xcc.TextAlignFlag_Left | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap)
	d.DrawText(name, &textRc)

	// 快捷键: 右对齐 text-xs muted(hot 时跟随 accent-foreground)
	if shortcut != "" {
		sc := t.MutedForeground
		if hv > 0 {
			sc = xui.Fade(sc, t.AccentForeground, hv)
		}
		if disabled {
			sc = sc.WithAlpha(128)
		}
		sr := rc
		sr.Right -= 6
		if hasIcon {
			sr.Right -= 24
		}
		d.SetFont(xui.FontHandle(xui.FontSpec{
			Family: t.FontDefault.Family,
			Size:   t.FontDefault.Size - 2, // text-xs
		}))
		d.SetBrushColor(uint32(sc))
		d.SetTextAlign(xcc.TextAlignFlag_Right | xcc.TextAlignFlag_Vcenter | xcc.TextFormatFlag_NoWrap)
		d.DrawText(shortcut, &sr)
	}

	// 右侧图标: 勾选标记 / 子菜单箭头(right-2, size-4)
	if hasIcon {
		iconRc := xc.RECT{
			Left:   rc.Right - 24,
			Top:    rc.Top + (rc.Bottom-rc.Top-16)/2,
			Right:  rc.Right - 8,
			Bottom: rc.Top + (rc.Bottom-rc.Top+16)/2,
		}
		if it.kind == menuKindSub {
			p.Icon(icons.IconChevronRight, &iconRc, fore, 2)
		} else {
			p.Icon(icons.IconCheck, &iconRc, fore, 2)
		}
	}
	return 0
}

// menuItemText 取条目文本: 优先用 Go 侧条目定义(名字/快捷键分开);
// 未知条目(如外部代码弹的原生菜单)从 PText 按 "\t" 拆分兜底。
func menuItemText(it *menuItem, pInfo *xc.Menu_DrawItem_) (name, shortcut string) {
	if it != nil {
		return it.text, it.shortcut
	}
	s := ""
	if pInfo.PText != 0 {
		s = common.UintPtrToString(pInfo.PText)
	}
	if i := strings.IndexByte(s, '\t'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
