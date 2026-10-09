package xui

// FontSpec 描述主题中的一套字体方案。组件按需从中选取字段。
type FontSpec struct {
	Family string // 字体名, 如 "微软雅黑"、"Segoe UI"
	Size   int32  // 字号(逻辑像素; xcgui 字体单位是 pt, FontHandle 内部按 px×0.75 换算)
	Bold   bool
	Italic bool
}

// Theme 是一套完整的设计令牌, 字段与 shadcn-ui 的 CSS 变量一一对应。
// 任何 shadcn 社区主题都可以机械转译成本结构体。
//
// 组件绘制时必须通过 Current() 实时取令牌, 禁止把颜色缓存到组件字段,
// 这是主题系统零成本切换的前提。
type Theme struct {
	Name string // 主题名, 用于调试与主题切换菜单

	// 表面
	Background Color // 窗口底色
	Foreground Color // 默认前景(正文文字)

	// 卡片 / 弹层
	Card        Color
	CardForeground    Color
	Popover     Color // 弹出层(菜单、下拉)底色
	PopoverForeground Color

	// 语义色
	Primary         Color // 主操作色
	PrimaryForeground     Color // 主操作色上的文字
	Secondary       Color // 次要填充
	SecondaryForeground   Color
	Muted           Color // 弱化填充(输入框底、禁用底)
	MutedForeground       Color // 弱化文字(说明、占位符)
	Accent          Color // 悬停高亮
	AccentForeground      Color
	Destructive     Color // 破坏性操作
	DestructiveForeground Color

	// 边框与焦点环
	Border Color
	Input  Color // 输入控件边框, shadcn 中通常与 Border 相同
	Ring   Color // 焦点环

	// 圆角(逻辑像素)。shadcn 用 --radius 派生 sm/md/lg/xl, 这里由助手函数计算。
	Radius int32

	// 字体方案
	FontDefault FontSpec
	FontMono    FontSpec
}

// RadiusSm/Md/Lg/Xl 按 shadcn 的比例从基准 Radius 派生各级圆角。
// 圆角比例与 shadcn/tailwind v4 的 radius 派生一致(基准 --radius, nova 预设为 10):
// sm=0.6x, md=0.8x, lg=1x, xl=1.4x。
func (t *Theme) RadiusSm() int32 {
	return int32(float32(t.Radius) * 0.6)
}

func (t *Theme) RadiusMd() int32 { return int32(float32(t.Radius) * 0.8) }

func (t *Theme) RadiusLg() int32 { return t.Radius }

func (t *Theme) RadiusXl() int32 { return int32(float32(t.Radius) * 1.4) }

// Pill 返回高度 h 的胶囊圆角(徽章 rounded-4xl)。
func (t *Theme) Pill(h int32) int32 { return h / 2 }

// Disabled 返回禁用态下的文字色(约 50% 透明度的前景色)。
func (t *Theme) Disabled() Color { return t.Foreground.WithAlpha(128) }

// IsDark 报告主题底色是否偏暗。
// 组件需要按 shadcn 的 dark:* 规则换色时用它——shadcn 允许对同一个部件在
// 亮/暗主题下指定不同令牌(如 switch 的滑块), 而主题结构体里没有"亮暗"标记,
// 只能按底色亮度推断。
func (t *Theme) IsDark() bool {
	r, g, b, _ := t.Background.RGBA()
	// Rec.601 感知亮度
	return float64(r)*0.299+float64(g)*0.587+float64(b)*0.114 < 127.5
}

// Overlay 返回弹层遮罩色(黑色 55% 透明)。
func (t *Theme) Overlay() Color { return RGBA(0, 0, 0, 140) }
