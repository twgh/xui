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

// TestIconGeometryFigureCount 守住"几何规模"这条性能红线:
// D2D 填充自重叠路径是 stencil-and-cover, 代价随 figure 数近似平方增长 ——
// 一旦把手写的图元汤(每段一个四边形 + 每个顶点一个圆帽)塞回来, 45 个图标同框的
// 总览页会从 4ms/帧 掉到 700ms/帧(见 DESIGN 第八节 43)。
//
// 现在描边不再展开(交给 D2D 的 DrawGeometry), 所以"子路径数"就等于 path 自己的子路径数;
// 这条红线照样要守。
func TestIconGeometryFigureCount(t *testing.T) {
	names := icons.Names()
	worst, maxFig, sum := "", 0, 0
	for _, n := range names {
		stroke, fill := iconFigures(parseIconPath(icons.Path(n)), 1)
		figs := len(stroke) + len(fill)
		if figs > maxFig {
			maxFig, worst = figs, n
		}
		sum += figs
	}
	t.Logf("%d 个图标: 平均 %.2f 条子路径/图标, 最多 %d 条(%s)",
		len(names), float64(sum)/float64(len(names)), maxFig, worst)
	// 实测: 平均 4.11 条/图标, 最多 20 条(virus, 尖刺多)。图元汤写法是几百条。
	if maxFig > 32 {
		t.Errorf("%s 有 %d 条子路径, 太多了(D2D 的代价随 figure 数增长)", worst, maxFig)
	}
}

// TestIconFigures 确认两组几何的硬语义:
//   - 描边子路径**保持开口**(Closed=false)—— 闭合会多画一段收尾线(三角形描边变回形针);
//   - 填充子路径闭合, 且点数不被展开(描边展开已交给 D2D);
//   - 描边/填充分开成两组, 各自进自己的几何(描边交给 DrawGeometry, 填充交给 FillGeometry)。
func TestIconFigures(t *testing.T) {
	subs := []iconSub{
		{pts: []iconPt{{x: 2, y: 2}, {x: 4, y: 2}, {x: 4, y: 4}, {x: 2, y: 4}}, filled: true},
		{pts: []iconPt{{x: 10, y: 10}, {x: 12, y: 10}, {x: 12, y: 12}}, filled: false},
	}
	stroke, fill := iconFigures(subs, 1)
	if len(fill) != 1 {
		t.Fatalf("填充子路径数 = %d, 想要 1", len(fill))
	}
	if !fill[0].Closed {
		t.Error("填充子路径必须闭合")
	}
	if len(fill[0].Points) != 8 {
		t.Errorf("填充点数 = %d, 想要 4 个点(不展开)", len(fill[0].Points)/2)
	}
	if len(stroke) != 1 {
		t.Fatalf("描边子路径数 = %d, 想要 1", len(stroke))
	}
	if stroke[0].Closed {
		t.Error("描边子路径必须开口(闭合会多画一段收尾线)")
	}
	if len(stroke[0].Points) != 6 {
		t.Errorf("描边点数 = %d, 想要 3 个点(不展开)", len(stroke[0].Points)/2)
	}
}
