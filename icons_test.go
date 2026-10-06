package xui

import (
	"math"
	"testing"

	"github.com/twgh/xui/icons"
)

// TestIconParseArcFlags 覆盖 lucide 数据里最容易踩的写法: arc 的 large-arc / sweep
// 是**单字符标志**, lucide 会连写成 "a.25.25 0 00.407.195"(= flag 0, flag 0, x .407, y .195)。
// 按"数字"贪婪读会得到 00.407 一个数, 于是参数不足 → 后面整段图形被丢掉(图标只画一半)。
func TestIconParseArcFlags(t *testing.T) {
	d := "M11 3v7.751a.25.25 0 00.407.195l2.28-1.834a.5.5 0 01.627 0l2.28 1.834a.25.25 0 00.406-.195V3" // lucide album 原样
	subs, complete := parseIconPathEx(d)
	if !complete {
		t.Fatalf("album 数据未被完整解析")
	}
	if len(subs) != 1 {
		t.Fatalf("子路径数 = %d, 想要 1", len(subs))
	}
	last := subs[0].pts[len(subs[0].pts)-1]
	if math.Abs(float64(last.y-3)) > 0.01 { // V3: 末点必须落在 y=3
		t.Errorf("末点 = (%.3f,%.3f), 想要落在 V3 的 y=3 上", last.x, last.y)
	}
}

// TestIconParseSyntax 覆盖相对指令、隐式重复组、H/V/S/T、粘连负号与填充标记。
func TestIconParseSyntax(t *testing.T) {
	cases := []struct {
		name     string
		d        string
		subs     int
		filled   int
		lastX    float32
		lastY    float32
		complete bool
	}{
		{"相对折线", "m2 2 4 0 0 4", 1, 0, 6, 6, true},                        // m 之后的隐式组按 l 处理
		{"粘连负号", "M10 10l-2-2", 1, 0, 8, 8, true},                         // "l-2-2" 是两个数
		{"hv", "M2 2h4v4", 1, 0, 6, 6, true},                              // H/V 保持当前另一轴
		{"多子路径", "M2 2L3 3M5 5L6 6", 2, 0, 6, 6, true},                    // M 起新子路径
		{"闭合", "M2 2L6 2L6 6Z", 1, 0, 2, 2, true},                         // Z 回到子路径起点
		{"填充标记", "!M2 2A2 2 0 1 1 6 2Z M10 10L12 12", 2, 1, 12, 12, true}, // 第一条子路径填充
		{"不完整", "M2 2L", 0, 0, 0, 0, false},                               // 参数不足必须如实报告(只有一个点画不出东西)
		{"未知指令", "M2 2Q", 0, 0, 0, 0, false},
	}
	for _, c := range cases {
		subs, complete := parseIconPathEx(c.d)
		if complete != c.complete {
			t.Errorf("%s: complete = %v, 想要 %v", c.name, complete, c.complete)
		}
		if len(subs) != c.subs {
			t.Errorf("%s: 子路径 %d 条, 想要 %d", c.name, len(subs), c.subs)
			continue
		}
		nFill := 0
		for _, s := range subs {
			for _, p := range s.pts {
				if math.IsNaN(float64(p.x)) || math.IsInf(float64(p.x), 0) {
					t.Fatalf("%s: 点坐标非有限值", c.name)
				}
			}
			if s.filled {
				nFill++
			}
		}
		if nFill != c.filled {
			t.Errorf("%s: 填充子路径 %d 条, 想要 %d", c.name, nFill, c.filled)
		}
		if len(subs) == 0 {
			continue
		}
		lastSub := subs[len(subs)-1]
		last := lastSub.pts[len(lastSub.pts)-1]
		if math.Abs(float64(last.x-c.lastX)) > 0.01 || math.Abs(float64(last.y-c.lastY)) > 0.01 {
			t.Errorf("%s: 末点 (%.2f,%.2f), 想要 (%.2f,%.2f)", c.name, last.x, last.y, c.lastX, c.lastY)
		}
	}
}

// TestGeneratedIcons 是资源完整性测试: 生成的每个图标都必须被解析器完整吃下,
// 且点都落在 24 网格附近(预处理把各图元转成 path, 转错了这里就会炸)。
func TestGeneratedIcons(t *testing.T) {
	names := icons.Names()
	if len(names) < 2000 {
		t.Fatalf("图标只有 %d 个, 生成资源不完整", len(names))
	}
	filled := 0
	for _, name := range names {
		d := icons.Path(name)
		if d == "" {
			t.Errorf("%s: 数据为空", name)
			continue
		}
		subs, complete := parseIconPathEx(d)
		if !complete {
			t.Errorf("%s: 数据未被完整解析: %s", name, d)
			continue
		}
		if len(subs) == 0 {
			t.Errorf("%s: 没有子路径: %s", name, d)
			continue
		}
		for _, s := range subs {
			if s.filled {
				filled++
			}
			for _, p := range s.pts {
				if p.x < -1 || p.x > 25 || p.y < -1 || p.y > 25 {
					t.Errorf("%s: 点 (%v,%v) 跑出 24 网格", name, p.x, p.y)
					break
				}
			}
		}
	}
	if filled < 20 { // lucide 里 fill="currentColor" 的色点(11 个图标)
		t.Errorf("填充子路径只有 %d 条, 预处理可能丢了 fill 标记", filled)
	}
}

// builtinIcons 是运行时内联的图标常量(组件自用, 见 icons.go 顶部注释)。
var builtinIcons = map[string]string{
	"chevron-down":  IconChevronDown,
	"chevron-up":    IconChevronUp,
	"chevron-left":  IconChevronLeft,
	"chevron-right": IconChevronRight,
	"check":         IconCheck,
	"x":             IconX,
	"minus":         IconMinus,
	"eye":           IconEye,
	"eye-off":       IconEyeOff,
	"sun":           IconSun,
	"moon":          IconMoon,
}

// TestBuiltinIconsMatchResource 断言内联常量与资源子包里的同名图标画的是同一个图形。
//
// 为什么需要: 常量是手抄的 lucide 数据(为了不把 350KB 资源表链进每个用户程序), 手抄就会漂移 ——
// 资源重新生成(或 lucide 改图形)后这里立刻报出来。两边文本不同(资源是绝对化 + 两位小数),
// 所以比的是**解析后的几何**: 子路径数、填充标记、逐点距离。
func TestBuiltinIconsMatchResource(t *testing.T) {
	for name, d := range builtinIcons {
		if !icons.Has(name) {
			t.Errorf("资源里没有 %q", name)
			continue
		}
		want, okW := parseIconPathEx(icons.Path(name))
		got, okG := parseIconPathEx(d)
		if !okW || !okG {
			t.Errorf("%s: 未被完整解析(资源 %v / 常量 %v)", name, okW, okG)
			continue
		}
		if len(want) != len(got) {
			t.Errorf("%s: 子路径数 资源 %d / 常量 %d", name, len(want), len(got))
			continue
		}
		for i := range want {
			a, b := want[i], got[i]
			if a.filled != b.filled {
				t.Errorf("%s 子路径 %d: 填充标记 资源 %v / 常量 %v", name, i, a.filled, b.filled)
			}
			// 两端写法可能不同(圆弧的分段数与起点不同), 所以不能逐点比 —— 比的是"两条折线贴不贴"。
			// 常量就是从资源抄来的, 阈值只留浮点取整的余量。
			if d := plDist(a.pts, b.pts); d > 0.02 {
				t.Errorf("%s 子路径 %d: 与资源的形状距离 %.3f 网格单位(>0.02) —— 常量与资源不一致(手抄漂移或 lucide 改了图形)",
					name, i, d)
			}
		}
	}
}

// plDist 量两条折线的最大偏差(单向 Hausdorff 的对称化)。
// 弧线离散方式不同会导致采样点错位, 但同一图形的偏差只有离散步长的量级(<< 0.1 网格单位)。
func plDist(a, b []iconPt) float64 {
	return math.Max(directedDist(a, b), directedDist(b, a))
}

func directedDist(from, to []iconPt) float64 {
	worst := 0.0
	for _, p := range from {
		best := math.MaxFloat64
		for i := 0; i+1 < len(to); i++ {
			if d := ptSegDist(p, to[i], to[i+1]); d < best {
				best = d
			}
		}
		if best > worst {
			worst = best
		}
	}
	return worst
}

func ptSegDist(p, a, b iconPt) float64 {
	px, py := float64(p.x), float64(p.y)
	ax, ay := float64(a.x), float64(a.y)
	bx, by := float64(b.x), float64(b.y)
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = ((px-ax)*dx + (py-ay)*dy) / l2
		t = math.Max(0, math.Min(1, t))
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

// TestIconGeomFilled 确认填充子路径是"原样入几何", 而不是被描边展开成胖环。
func TestIconGeomFilled(t *testing.T) {
	subs := []iconSub{
		{pts: []iconPt{{x: 2, y: 2}, {x: 4, y: 2}, {x: 4, y: 4}, {x: 2, y: 4}}, filled: true},
		{pts: []iconPt{{x: 10, y: 10}, {x: 12, y: 10}}, filled: false},
	}
	figs := iconStrokeFigures(subs, 0, 0, 1, 2)
	if len(figs) != 1+1+2 { // 填充 1 条 + 描边 1 段(1 个四边形)+ 2 个顶点圆帽
		t.Fatalf("figure 数 = %d, 想要 %d", len(figs), 1+1+2)
	}
	if len(figs[0]) != 8 { // 4 个点 = 8 个 float, 未被展开
		t.Errorf("填充 figure 点数 = %d, 想要 4", len(figs[0])/2)
	}
}
