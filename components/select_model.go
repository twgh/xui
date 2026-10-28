package ui

// Select / Combobox 共享的数据模型与行几何。
//
// 单独成文件是因为 Combobox(自动补全) 与 Select(下拉选择) 用同一套条目模型:
// 两者的弹层行高、分组标签高一致, 用户也可以把同一份 []SelectGroup 传给任意一个。
// registry 里 select 与 combobox-search 都引用本文件。

// Select 系几何(逻辑像素), 对齐 select.tsx(radix-nova):
// 条目 text-sm + py-1 = 28; label text-xs + py-1 = 24;
// 分隔线 my-1 + 1px = 17; 内容四周 p-1; 弹层 min-w-36。
const (
	selectItemH  int32 = 28
	selectLabelH int32 = 24
	selectSepH   int32 = 17
	selectPad    int32 = 4
	selectMinW   int32 = 144 // min-w-36
	selectMaxH   int32 = 320 // 内容高上限(约 11 项), 超出滚动
)

// SelectSize 触发区尺寸, 同 select.tsx 的 size(default h-8 / sm h-7)。
type SelectSize int32

const (
	SelectSizeDefault SelectSize = iota // 高 32
	SelectSizeSm                        // 高 28, 圆角 min(--radius-md, 10)
)

// SelectItem 下拉条目。Value 是程序侧标识(onChange 回传), Text 是显示文字。
type SelectItem struct {
	Value    string
	Text     string
	Icon     string // 前置图标(24 网格 path / icons 常量), 可空
	Disabled bool
}

// SelectGroup 带标签的条目分组(同 radix SelectGroup + SelectLabel)。Label 为空时只有分隔线。
type SelectGroup struct {
	Label string
	Items []SelectItem
}
