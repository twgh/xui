package xui

import (
	"strings"
	"testing"
)

// 一段"标准图标站导出"的 SVG: viewBox 24、fill=none、stroke=currentColor、stroke-width=2。
const testSVG24 = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M12 3 L21 21 L3 21 Z"/>
</svg>`

// 16 网格 + <g> 继承 + 多 path + 一个填充 path。
const testSVG16 = `<svg viewBox="0 0 16 16">
  <g fill="none" stroke="currentColor" stroke-width="1.5">
    <path d="M2 2 L14 2"/>
    <path d="M8 2 L8 14"/>
  </g>
  <path fill="currentColor" stroke="none" d="M6 6 L10 6 L10 10 L6 10 Z"/>
</svg>`

func TestParseIconSVG24(t *testing.T) {
	paths, sw, err := ParseIconSVG(testSVG24)
	if err != nil {
		t.Fatal(err)
	}
	if sw != 2 {
		t.Errorf("建议线宽 = %v, 想要 2", sw)
	}
	if strings.HasPrefix(paths, "!") {
		t.Errorf("fill=none 的子路径不该被标成填充: %s", paths)
	}
	t.Logf("归一化输出(前 120 字符): %.120q", paths)
	// 输出必须能被现有解析器完整吃下(闭环)
	subs, complete := parseIconPathEx(paths)
	if !complete || len(subs) != 1 {
		t.Fatalf("归一化结果解析不完整: complete=%v 子路径=%d", complete, len(subs))
	}
	// 坐标应原样(24 网格不缩放), 首点 (12,3)
	if got := subs[0].pts[0]; got.x < 11.99 || got.x > 12.01 || got.y < 2.99 || got.y > 3.01 {
		t.Errorf("首点 = (%v,%v), 想要 (12,3)", got.x, got.y)
	}
}

func TestParseIconSVGScaleAndFill(t *testing.T) {
	paths, sw, err := ParseIconSVG(testSVG16)
	if err != nil {
		t.Fatal(err)
	}
	// 16 网格 ×1.5 → 24; stroke-width 1.5 ×1.5 = 2.25
	if sw < 2.24 || sw > 2.26 {
		t.Errorf("建议线宽 = %v, 想要 2.25(源 1.5 × 1.5)", sw)
	}
	// 三条子路径: 两条描边(无 `!`) + 一条填充(有 `!`)
	if n := strings.Count(paths, "!"); n != 1 {
		t.Errorf("填充前缀数 = %d, 想要 1: %s", n, paths)
	}
	subs, complete := parseIconPathEx(paths)
	if !complete || len(subs) != 3 {
		t.Fatalf("归一化结果解析不完整: complete=%v 子路径=%d", complete, len(subs))
	}
	if !subs[2].filled {
		t.Error("第三条子路径应是填充")
	}
	if subs[0].filled || subs[1].filled {
		t.Error("<g fill=none> 里的两条应是描边")
	}
	// 缩放: 源 (2,2) → (3,3)
	if got := subs[0].pts[0]; got.x < 2.99 || got.x > 3.01 {
		t.Errorf("首点 x = %v, 想要 3(源 2 × 1.5)", got.x)
	}
}

func TestParseIconSVGViewBoxOffset(t *testing.T) {
	// 非零 min-x/min-y 要平掉
	paths, _, err := ParseIconSVG(`<svg viewBox="10 20 24 24"><path d="M10 20 L12 20"/></svg>`)
	if err != nil {
		t.Fatal(err)
	}
	subs, _ := parseIconPathEx(paths)
	if len(subs) != 1 || len(subs[0].pts) != 2 {
		t.Fatalf("解析结果不对: %s", paths)
	}
	if p := subs[0].pts[0]; p.x != 0 || p.y != 0 {
		t.Errorf("viewBox 偏移没平掉: 首点 = (%v,%v), 想要 (0,0)", p.x, p.y)
	}
}

func TestParseIconSVGErrors(t *testing.T) {
	if _, _, err := ParseIconSVG(`<svg viewBox="0 0 24 24"></svg>`); err == nil {
		t.Error("没有 path 应报错")
	}
	if _, _, err := ParseIconSVG(`不是 svg`); err == nil {
		t.Error("不是 SVG 应报错")
	}
}

// 用 heroicons 官方原文(整段 SVG)走一遍闭环 —— 这是"用户直接贴图标站代码"的典型形态。
func TestParseIconSVGHeroicons(t *testing.T) {
	const hero = `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
  <path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.562.562 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z"/>
</svg>`
	paths, sw, err := ParseIconSVG(hero)
	if err != nil {
		t.Fatal(err)
	}
	if sw != 1.5 {
		t.Errorf("建议线宽 = %v, 想要 1.5", sw)
	}
	subs, complete := parseIconPathEx(paths)
	if !complete || len(subs) != 1 {
		t.Fatalf("heroicons 归一化后解析不完整: complete=%v", complete)
	}
	if len(subs[0].pts) < 50 {
		t.Errorf("点数 = %d, 太少了(弧线没展开?)", len(subs[0].pts))
	}
}

// 画在 Painter 上的入口: 解析失败要安静地什么都不画(不 panic)。
func TestIconSVGNilSafety(t *testing.T) {
	var p *Painter
	p.IconSVG(testSVG24, nil, RGBA(0, 0, 0, 255), 2) // nil Painter + 坏参数不该 panic
	if v := parseIconSVGCached("不是 svg"); v.ok {
		t.Error("坏 SVG 不该被标记为 ok")
	}
}
