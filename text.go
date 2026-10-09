package xui

import (
	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// TextWidth 量测**单行**文本的逻辑像素宽度(与绘制同字体; hFont 用 FontHandle 取)。
//
// 坑(探针实测, 见 docs/DESIGN.md 第八节 48): `app.GetTextShowRect` 的 width 参数传 0
// **不是"不限宽"**, 而是"按一个字符宽折行"——此时返回的 CX 只是一列宽, CY 变成"行数×行高"
// (3 个汉字会得到 CX=15 / CY=59)。量宽必须给足 width 并加 NoWrap, 此时 CX 就是真实单行宽度
// (14px 字号下汉字恰好 15/字, 与绘制结果一致), **不需要任何倍率校正**。
func TextWidth(text string, hFont int) int32 {
	if text == "" || hFont == 0 {
		return 0
	}
	const enough = 4096 // 给足宽度: 只影响折行判定, 单行时不影响 CX
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, hFont,
		xcc.TextAlignFlag_Left|xcc.TextFormatFlag_NoWrap, enough, &sz)
	return sz.CX
}

// TextHeight 量测单行文本的行高(逻辑像素)。同 TextWidth 的坑, 必须给足宽度 + NoWrap。
func TextHeight(text string, hFont int) int32 {
	if text == "" || hFont == 0 {
		return 0
	}
	var sz xc.SIZE
	app.GetTextShowRect(text, -1, hFont,
		xcc.TextAlignFlag_Left|xcc.TextFormatFlag_NoWrap, 4096, &sz)
	return sz.CY
}
