package ui

// textVDy 返回 Vcenter 文本的光学上移量(逻辑像素)。
//
// 150% DPI 实测: 同一套 Vcenter 下, 拉丁大写字母的墨迹中心在行盒中心上方
// 约 1 物理像素, 中文字形几乎占满字身、墨迹中心在行盒中心下方约 1 物理像素;
// 14px 规格又被取整成 11pt(14.67px), 中文再低一点。胶囊/按钮一量就看得出来。
// 因此只对含中文的文本上移半个逻辑像素(整数矩形做不到半步, 用 TextF 承载),
// 纯拉丁文本保持原位。见 DESIGN 第八节 151。
func textVDy(s string) float32 {
	for _, r := range s {
		if isCJK(r) {
			return 0.5
		}
	}
	return 0
}

// isCJK 判断是否中日韩文字/全角字符。
func isCJK(r rune) bool {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF: // 部首、假名、统一汉字
		return true
	case r >= 0xF900 && r <= 0xFAFF: // 兼容汉字
		return true
	case r >= 0xFE30 && r <= 0xFE4F: // 兼容形式
		return true
	case r >= 0xFF00 && r <= 0xFFEF: // 全角/半角形式
		return true
	}
	return false
}
