package ui

// 菜单条目模型: 原生 Menu(menu.go) 与自绘 Menubar(menubar.go) 共用同一套
// 条目类型与选项构造器(MenuShortcut / MenuDisabled / MenuOnSelect ...)。
// 模型只描述数据, 不依赖任何绘制或原生控件。

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
	inset       bool   // 文字左缩进到与图标/勾选条目对齐(shadcn 的 data-inset:pl-7)
	icon        string // 前置图标(path/SVG, 与 Button.WithIcon 同口径)
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

// MenuInset 让条目文字左缩进到 28(shadcn 的 `inset` prop = `data-inset:pl-7`):
// 与带前置图标(MenuIcon)/前置指示器的条目文字对齐。普通条目、Label、Sub 都可加;
// 绘制只挪文字与右侧图标, 高亮底仍是整行。
func MenuInset() MenuOpt { return func(it *menuItem) { it.inset = true } }

// MenuIcon 给条目(普通/勾选/单选/子菜单)画前置图标, 图标 16px 贴着条目左缘
// left-1.5, 文字从 28 起(shadcn 条目内联 SVG 的 gap-1.5 口径)。
// 图标与文字/勾选/子菜单箭头可同时存在; path 与 Button.WithIcon 同口径
// (lucide path 或整段 SVG, 见 icons 子包)。
func MenuIcon(path string) MenuOpt { return func(it *menuItem) { it.icon = path } }

// MenuOnSelect 普通条目的点击回调。
func MenuOnSelect(fn func()) MenuOpt { return func(it *menuItem) { it.onSelect = fn } }

// MenuOnCheck 勾选/单选条目的状态回调, 参数为点击后的新状态。
func MenuOnCheck(fn func(checked bool)) MenuOpt { return func(it *menuItem) { it.onCheck = fn } }
