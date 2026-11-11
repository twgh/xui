package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
	"github.com/twgh/xui/icons"
)

// ===== 页 5: 表单1(文本输入 / 验证码 / 滑动条) =====
//
// 骨架是"一行两列、每列把卡片竖着排", 不是"两行各两列":
// 行高只有 Weight(按剩余空间等分)可用, 卡片比行高时下一行就会压到上一行的卡片上
// (实测: 输入框卡 430 高, 行只分到 363, 第二行的卡片上沿 379 直接盖住它 —— 用户报的
// "下方两个卡片盖住上方卡片")。竖排在同一列里则天然按内容顺序往下走, 不重叠。
func buildForm1Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 1, gap)
	colR := xui.VStack(r.Handle, 1, gap)

	// ---- 左列 1: 输入框 ----
	_, v := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "输入框", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0, "原生 Edit 载体: 只画边框与焦点环, 光标/输入法/选区都是原生的", v.Handle))

	xui.ItemFill(hint(v.Handle, "邮箱地址"))
	xui.ItemFill(ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithPlaceholder("m@example.com")))
	xui.ItemFill(hint(v.Handle, "带初始值"))
	filled := ui.NewInput(0, 0, 0, 0, v.Handle)
	xui.ItemFill(filled)
	filled.SetValue("twgh")
	xui.ItemFill(hint(v.Handle, "文本域(随内容长高, 最少 4 行; 要固定高度/滚动条用 Input + WithMultiline)"))
	ta := ui.NewTextarea(0, 0, 0, 0, v.Handle, ui.WithPlaceholder("介绍一下自己…"))
	xui.ItemFill(ta)
	ta.SetValue("测试文本域:\n输入内容后高度自动增长。")
	xui.ItemFill(hint(v.Handle, "禁用"))
	dis := ui.NewInput(0, 0, 0, 0, v.Handle)
	xui.ItemFill(dis)
	dis.SetValue("不可编辑")
	dis.Enable(false)
	xui.ItemFill(hint(v.Handle, "校验失败(aria-invalid)"))
	inv := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithInputInvalid())
	xui.ItemFill(inv)
	inv.SetValue("not-an-email")
	xui.ItemFill(hint(v.Handle, "步进器(−/+ 或 ↑↓ 调整, 失焦自动夹取/吸附; 步长可配 1 / 10 / 0.1)"))
	step := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithStepper(0, 100, 1))
	xui.ItemFill(step)
	step.SetNumber(42)
	xui.ItemFill(hint(v.Handle, "上下三角步进器(悬停/聚焦才出现, 步长 10)"))
	arrows := ui.NewInput(0, 0, 0, 0, v.Handle,
		ui.WithStepper(0, 1000, 10), ui.WithStepperArrows(), ui.WithStepperAutoHide())
	xui.ItemFill(arrows)
	arrows.SetNumber(320)
	xui.ItemFill(hint(v.Handle, "前置图标 + 前后缀(input-group)"))
	url := ui.NewInput(0, 0, 0, 0, v.Handle,
		ui.WithLeadingIcon(icons.IconAtSign), ui.WithPrefixText("https://"), ui.WithSuffixText(".com"))
	xui.ItemFill(url)
	url.SetValue("twgh")
	xui.ItemFill(hint(v.Handle, "清除按钮 / 密码显隐 / 字符计数"))
	search := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithLeadingIcon(icons.IconSearch), ui.WithClearButton())
	xui.ItemFill(search)
	search.SetValue("shadcn")
	pwd := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithPasswordToggle())
	xui.ItemFill(pwd)
	pwd.SetValue("secret-123")
	counter := ui.NewInput(0, 0, 0, 0, v.Handle, ui.WithCounter(120))
	xui.ItemFill(counter)
	counter.SetValue("这是一段用于演示字数统计的文本")

	// ---- 左列 2: 输入组(Input Group) ----
	_, vg := autoCard(colL.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "输入组", vg.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"InputGroup: 输入框与附加件共用一层边框与焦点环; addon 四向, 可放图标/文字/按钮/Kbd/下拉菜单", vg.Handle))

	xui.ItemFill(hint(vg.Handle, "前置图标 / 后置按钮"))
	g1 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g1.Input(ui.WithPlaceholder("搜索组件…"))
	g1.Addon(ui.InputGroupInlineStart).Icon(icons.IconSearch)
	xui.ItemFill(g1)

	g2 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g2.Input(ui.WithPlaceholder("输入 GitHub 用户名"))
	g2.Addon(ui.InputGroupInlineEnd).Button("查询")
	xui.ItemFill(g2)

	xui.ItemFill(hint(vg.Handle, "前后缀文字 + Kbd"))
	g3 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g3.Addon(ui.InputGroupInlineStart).Text("https://")
	g3.Input(ui.WithPlaceholder("example.com"))
	g3.Addon(ui.InputGroupInlineEnd).Kbd("⌘K")
	xui.ItemFill(g3)

	xui.ItemFill(hint(vg.Handle, "block-start 标题 + block-end 说明/按钮"))
	g4 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g4.Addon(ui.InputGroupBlockStart).Text("邮箱地址")
	g4.Input(ui.WithPlaceholder("m@example.com"))
	g4.Addon(ui.InputGroupBlockEnd).Text("我们会发送验证码").Button("发送")
	xui.ItemFill(g4)

	xui.ItemFill(hint(vg.Handle, "文本域 + 字符计数"))
	g5 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g5.Textarea(80, ui.WithPlaceholder("输入留言…"))
	g5.Addon(ui.InputGroupBlockEnd).Text("0/280").Button("发送")
	xui.ItemFill(g5)

	xui.ItemFill(hint(vg.Handle, "复合 addon: 下拉菜单 / 多控件 + 尺寸档"))
	g8 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g8.Input(ui.WithPlaceholder("Search In..."))
	g8.Addon(ui.InputGroupInlineEnd).MenuButton("", icons.IconChevronDown, func(m *ui.Menu) {
		m.Item("All")
		m.Item("Files")
		m.Item("Code")
		m.Item("Web")
	})
	xui.ItemFill(g8)

	g9 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle)
	g9.Input(ui.WithPlaceholder("一个 addon 里放两个按钮(sm / icon-sm)"))
	end9 := g9.Addon(ui.InputGroupInlineEnd)
	end9.Button("发送", ui.WithSize(ui.ButtonSizeSm))
	end9.IconButton(icons.IconEllipsis, ui.WithSize(ui.ButtonSizeIconSm))
	xui.ItemFill(g9)

	xui.ItemFill(hint(vg.Handle, "校验失败 / 禁用"))
	g6 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle, ui.WithInputGroupInvalid())
	g6.Input(ui.WithPlaceholder("必填项"))
	xui.ItemFill(g6)
	g7 := ui.NewInputGroup(0, 0, 0, 0, vg.Handle, ui.WithInputGroupDisabled())
	g7.Input(ui.WithPlaceholder("不可编辑"))
	g7.Addon(ui.InputGroupInlineEnd).IconButton(icons.IconCopy)
	xui.ItemFill(g7)

	// ---- 右列 1: 验证码输入 ----
	_, v6 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "验证码输入", v6.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"InputOTP: 逐格输入(打进已填格 = 替换), ←→/Home/End 移动, Backspace 空格退格, Ctrl+V 分发粘贴; 全满触发 OnComplete", v6.Handle))

	otpVal := ui.NewLabel(0, 0, 0, 0, "值:", v6.Handle).SetMuted()
	xui.ItemFill(otpVal)
	otp := ui.NewInputOTP(0, 0, 0, 0, v6.Handle,
		ui.WithInputOTPLength(6), ui.WithInputOTPPattern(ui.InputOTPPatternDigit),
		ui.WithInputOTPSeparatorAfter(3))
	xui.ItemFill(otp)
	otp.OnChange(func(v string) { otpVal.SetText("值: " + v).Redraw() })
	otp.OnComplete(func(v string) { otpVal.SetText("已完成: " + v).Redraw() })

	xui.ItemFill(hint(v6.Handle, "禁用 / 校验失败(aria-invalid)"))
	xui.ItemFill(ui.NewInputOTP(0, 0, 0, 0, v6.Handle,
		ui.WithInputOTPLength(4), ui.WithInputOTPDisabled()))
	xui.ItemFill(ui.NewInputOTP(0, 0, 0, 0, v6.Handle,
		ui.WithInputOTPLength(4), ui.WithInputOTPInvalid()))

	// ---- 右列 2: 滑动条 ----
	_, v3 := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "滑动条", v3.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"纯自绘: 点击跳转 / 拖拽 / 方向键, hover 与拖动显示光环; 单值 / 范围 / 多滑块 / 垂直 / 禁用", v3.Handle))

	xui.ItemFill(hint(v3.Handle, "连续(0~100)"))
	rowS := xui.HStack(v3.Handle, 24, 8)
	sl := ui.NewSlider(0, 0, 200, 0, rowS.Handle)
	xui.ItemWidth(sl, 218) // 220 = 可见轨道 200 + 两端各 9(滑块 ring 的绘制余量)
	slVal := ui.NewLabel(0, 0, 40, 0, "40", rowS.Handle).SetMuted()
	xui.ItemWidth(slVal, 40)
	sl.SetValue(40)
	sl.OnValuesChange(func(vals []float64) { slVal.SetText(strconv.Itoa(int(vals[0] + 0.5))).Redraw() })

	xui.ItemFill(hint(v3.Handle, "范围双滑块(20~80, step 1, 最小间距 5 步)"))
	rowR := xui.HStack(v3.Handle, 24, 8)
	rng := ui.NewSlider(0, 0, 200, 0, rowR.Handle,
		ui.WithSliderValues(20, 80), ui.WithSliderStep(1), ui.WithSliderMinStepsBetweenThumbs(5))
	xui.ItemWidth(rng, 218)
	rngVal := ui.NewLabel(0, 0, 80, 0, "20 - 80", rowR.Handle).SetMuted()
	xui.ItemWidth(rngVal, 80)
	rng.OnValuesChange(func(vals []float64) {
		rngVal.SetText(fmt.Sprintf("%.0f - %.0f", vals[0], vals[1])).Redraw()
	})

	xui.ItemFill(hint(v3.Handle, "三个滑块(10 / 40 / 70, step 10)"))
	rowM := xui.HStack(v3.Handle, 24, 8)
	multi := ui.NewSlider(0, 0, 200, 0, rowM.Handle,
		ui.WithSliderValues(10, 40, 70), ui.WithSliderStep(10))
	xui.ItemWidth(multi, 218)
	multiVal := ui.NewLabel(0, 0, 96, 0, "10/40/70", rowM.Handle).SetMuted()
	xui.ItemWidth(multiVal, 96)
	multi.OnValuesChange(func(vals []float64) {
		multiVal.SetText(fmt.Sprintf("%.0f/%.0f/%.0f", vals[0], vals[1], vals[2])).Redraw()
	})

	xui.ItemFill(hint(v3.Handle, "垂直(轨道 120) + 提交回调 / 禁用"))
	rowV := xui.HStack(v3.Handle, 138, 16)
	vs := ui.NewSlider(0, 0, 0, 120, rowV.Handle, ui.WithSliderVertical(), ui.WithSliderValues(40))
	vsVal := ui.NewLabel(0, 0, 48, 0, "40", rowV.Handle).SetMuted()
	xui.ItemWidth(vsVal, 48)
	vs.OnValuesChange(func(vals []float64) { vsVal.SetText(strconv.Itoa(int(vals[0] + 0.5))).Redraw() })
	commitVal := ui.NewLabel(0, 0, 140, 0, "commit: -", rowV.Handle).SetMuted()
	xui.ItemWidth(commitVal, 140)
	vs.OnValuesCommit(func(vals []float64) {
		commitVal.SetText(fmt.Sprintf("commit: %.0f", vals[0])).Redraw()
	})
	sDis := ui.NewSlider(0, 0, 200, 0, rowV.Handle,
		ui.WithSliderDisabled(), ui.WithSliderValues(30))
	xui.ItemWidth(sDis, 218)

	xui.ItemFill(hint(v3.Handle, "值域 0~10, step 1"))
	rowS2 := xui.HStack(v3.Handle, 24, 8)
	st := ui.NewSlider(0, 0, 200, 0, rowS2.Handle,
		ui.WithSliderRange(0, 10), ui.WithSliderStep(1))
	xui.ItemWidth(st, 218)
	stVal := ui.NewLabel(0, 0, 40, 0, "3", rowS2.Handle).SetMuted()
	xui.ItemWidth(stVal, 40)
	st.SetValue(3)
	st.OnValuesChange(func(vals []float64) { stVal.SetText(strconv.Itoa(int(vals[0] + 0.5))).Redraw() })

	// ---- 右列 3: 标签输入 ----
	_, vt := autoCard(colR.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "标签输入", vt.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"TagInput: 回车成标签 / 逗号分隔 / 退格删末个 / 点 × 删除; 胶囊排满自动换行并撑高", vt.Handle))

	tagVal := ui.NewLabel(0, 0, 0, 0, "标签: -", vt.Handle).SetMuted()
	xui.ItemFill(tagVal)
	tags := ui.NewTagInput(0, 0, 0, 0, vt.Handle,
		ui.WithTagInputPlaceholder("输入后回车…"),
		ui.WithTagInputTags("Go", "xcgui"))
	xui.ItemFill(tags)
	tagVal.SetText("标签: " + strings.Join(tags.Tags(), " / ")).Redraw()
	tags.OnChange(func(v []string) {
		tagVal.SetText("标签: " + strings.Join(v, " / ")).Redraw()
	})

	xui.ItemFill(hint(vt.Handle, "上限 3 个(达到上限后提交被忽略, 输入文字保留)"))
	xui.ItemFill(ui.NewTagInput(0, 0, 0, 0, vt.Handle,
		ui.WithTagInputMax(3), ui.WithTagInputTags("设计", "开发", "发布"),
		ui.WithTagInputPlaceholder("已达上限")))

	xui.ItemFill(hint(vt.Handle, "禁用 / 校验失败"))
	xui.ItemFill(ui.NewTagInput(0, 0, 0, 0, vt.Handle,
		ui.WithTagInputDisabled(), ui.WithTagInputTags("不可编辑")))
	xui.ItemFill(ui.NewTagInput(0, 0, 0, 0, vt.Handle,
		ui.WithTagInputInvalid(), ui.WithTagInputTags("待校验")))
}
