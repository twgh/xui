package main

import (
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 14: 导航5(导航菜单 / 菜单栏) =====

func buildNav5Page(body *widget.LayoutEle) {
	// ---- 导航菜单(共享视口) ----
	_, vn := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "导航菜单", vn.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"共享视口贴在列表下方: 悬停 200ms 打开、离开 150ms 收起; 切换菜单时视口随内容收放, 新页面从切换方向滑入", vn.Handle))
	navStatus := ui.NewLabel(0, 0, 0, 20, "尚未选择链接", vn.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(navStatus)
	setNavStatus := func(text string) { navStatus.SetText(text).Redraw(false) }
	navState := ui.NewLabel(0, 0, 0, 20, "菜单: 已收起", vn.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(navState)

	nav := ui.NewNavigationMenu(vn.Handle)
	nav.AddMenu("开始使用", func(c *ui.NavigationMenuContent) {
		c.Link("组件介绍", "可复用的自绘组件, 基于 xcgui 构建", func() {
			setNavStatus("开始使用 / 组件介绍")
		})
		c.Link("安装", "把组件源码拷进你的项目", func() {
			setNavStatus("开始使用 / 安装")
		})
		c.Link("排版", "标题、正文与间距的基本规则", func() {
			setNavStatus("开始使用 / 排版")
		})
	}, ui.WithNavigationMenuWidth(400))
	nav.AddMenu("组件", func(c *ui.NavigationMenuContent) {
		grid := [][2][2]string{
			{{"按钮", "6 个变体 × 8 档尺寸"}, {"对话框", "窗口内覆盖元素 + 焦点陷阱"}},
			{{"下拉菜单", "自绘弹层, 支持子菜单"}, {"表格", "原生 List 载体, 外观全自绘"}},
			{{"月历", "范围选择 / 多月并排"}, {"通知", "堆叠、倒计时与富文本配色"}},
		}
		for _, pair := range grid {
			rh := xui.HStack(c.Handle(), 68, 8)
			for _, cell := range pair {
				cell := cell // go.mod 还是 1.18: 循环变量按旧语义共享, 回调里必须拷一份
				l := c.LinkIn(rh.Handle, cell[0], cell[1], func() {
					setNavStatus("组件 / " + cell[0])
				})
				xui.ItemWeight(l, 1)
			}
		}
	}, ui.WithNavigationMenuWidth(520))
	nav.AddLink("文档", func() {
		setNavStatus("文档")
	})
	nav.OnValueChange(func(value string) {
		if value == "" {
			navState.SetText("菜单: 已收起").Redraw(false)
		} else {
			navState.SetText("菜单: " + value).Redraw(false)
		}
	})
	xui.ItemFill(desc(vn.Handle,
		"悬停 200ms 打开 · 离开 150ms 收起 · 关闭后 300ms 内切到其它触发区立即打开; ←→ 移动焦点, ↓ 进入内容, ↑↓ 选择, Enter 激活, ESC 关闭。"))

	_, v := autoCard(body.Handle, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "菜单栏", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"Menubar: 打开任一菜单后, 鼠标移到其它触发区即切换; 勾选/单选/子菜单/快捷键", v.Handle))

	status := ui.NewLabel(0, 0, 0, 20, "尚未选择条目", v.Handle).SetFontSize(14).SetMuted()
	xui.ItemFill(status)
	setStatus := func(text string) { status.SetText(text).Redraw(false) }

	mb := ui.NewMenubar(v.Handle)

	file := mb.AddMenu("文件")
	file.Item("新建标签页", ui.MenuShortcut("Ctrl+T"), ui.MenuOnSelect(func() {
		setStatus("文件 / 新建标签页")
	}))
	file.Item("新建窗口", ui.MenuShortcut("Ctrl+N"), ui.MenuOnSelect(func() {
		setStatus("文件 / 新建窗口")
	}))
	file.Item("新建无痕窗口", ui.MenuDisabled())
	file.Separator()
	file.Sub("分享", func(s *ui.MenubarMenu) {
		s.Item("邮件链接", ui.MenuOnSelect(func() { setStatus("文件 / 分享 / 邮件链接") }))
		s.Item("消息", ui.MenuOnSelect(func() { setStatus("文件 / 分享 / 消息") }))
		s.Item("备忘录", ui.MenuOnSelect(func() { setStatus("文件 / 分享 / 备忘录") }))
	})
	file.Separator()
	file.Item("打印…", ui.MenuShortcut("Ctrl+P"), ui.MenuOnSelect(func() {
		setStatus("文件 / 打印…")
	}))

	edit := mb.AddMenu("编辑")
	edit.Item("撤销", ui.MenuShortcut("Ctrl+Z"), ui.MenuOnSelect(func() { setStatus("编辑 / 撤销") }))
	edit.Item("重做", ui.MenuShortcut("Ctrl+Y"), ui.MenuOnSelect(func() { setStatus("编辑 / 重做") }))
	edit.Separator()
	edit.Item("剪切", ui.MenuOnSelect(func() { setStatus("编辑 / 剪切") }))
	edit.Item("复制", ui.MenuOnSelect(func() { setStatus("编辑 / 复制") }))
	edit.Item("粘贴", ui.MenuOnSelect(func() { setStatus("编辑 / 粘贴") }))

	view := mb.AddMenu("视图")
	view.Check("显示行号", true, ui.MenuOnCheck(func(on bool) {
		if on {
			setStatus("视图 / 显示行号: 开")
		} else {
			setStatus("视图 / 显示行号: 关")
		}
	}))
	view.Check("显示书签栏", false, ui.MenuOnCheck(func(on bool) {
		if on {
			setStatus("视图 / 显示书签栏: 开")
		} else {
			setStatus("视图 / 显示书签栏: 关")
		}
	}))
	view.Separator()
	view.Item("重新加载", ui.MenuShortcut("F5"), ui.MenuOnSelect(func() { setStatus("视图 / 重新加载") }))
	view.Item("强制重新加载", ui.MenuDisabled())

	profile := mb.AddMenu("用户")
	profile.Label("账号")
	profile.Radio("浅色主题", true, ui.MenuGroup(1), ui.MenuOnCheck(func(bool) {
		setStatus("用户 / 主题: 浅色")
	}))
	profile.Radio("深色主题", false, ui.MenuGroup(1), ui.MenuOnCheck(func(bool) {
		setStatus("用户 / 主题: 深色")
	}))
	profile.Separator()
	profile.Item("退出登录", ui.MenuDestructive(), ui.MenuOnSelect(func() { setStatus("用户 / 退出登录") }))

	xui.ItemFill(desc(v.Handle,
		"↑↓ 移动 · Enter 选中 · ←→ 切换顶级菜单 · ESC 关闭并把焦点还给触发区; 子菜单悬停展开(非激活弹层, 不抢焦点)。"))
}
