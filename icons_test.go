package xui

import (
	"testing"

	"github.com/twgh/xcgui/svgp"
	"github.com/twgh/xui/icons"
)

// TestGeneratedIcons 是资源完整性测试: 生成的每个图标都必须被解析器完整吃下,
// 且点都落在 24 网格附近(预处理把各图元转成 path, 转错了这里就会炸)。
//
// path 语法与容错的用例都在 github.com/twgh/xcgui/svgp(那边有 syntax 表与紧凑写法用例);
// 这里守的是**资源与解析器的契约**: 两千多条生成数据条条可解析、坐标不越界、填充标记还在。
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
		subs, err := svgp.ParsePath(d)
		if err != nil {
			t.Errorf("%s: 数据未被完整解析: %v (%s)", name, err, d)
			continue
		}
		if len(subs) == 0 {
			t.Errorf("%s: 没有子路径: %s", name, d)
			continue
		}
		for _, s := range subs {
			if s.Filled {
				filled++
			}
			for _, p := range s.Points {
				if p.X < -1 || p.X > 25 || p.Y < -1 || p.Y > 25 {
					t.Errorf("%s: 点 (%v,%v) 跑出 24 网格", name, p.X, p.Y)
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
		subs, _ := svgp.ParsePath(icons.Path(n))
		if len(subs) > maxFig {
			maxFig, worst = len(subs), n
		}
		sum += len(subs)
	}
	t.Logf("%d 个图标: 平均 %.2f 条子路径/图标, 最多 %d 条(%s)",
		len(names), float64(sum)/float64(len(names)), maxFig, worst)
	// 实测: 平均 4.11 条/图标, 最多 20 条(virus, 尖刺多)。图元汤写法是几百条。
	if maxFig > 32 {
		t.Errorf("%s 有 %d 条子路径, 太多了(D2D 的代价随 figure 数增长)", worst, maxFig)
	}
}
