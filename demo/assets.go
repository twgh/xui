package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/twgh/xcgui/common"
	"github.com/twgh/xcgui/wapi"
)

// demoAvatarPNG 是演示头像; 用 embed 而不是运行时读 demo/assets, 这样
// `go run github.com/twgh/xui/demo@latest` 在任意工作目录下也能显示图片。
//
//go:embed assets/avatar.png
var demoAvatarPNG []byte

func demoAvatar() image.Image {
	img, _, err := image.Decode(bytes.NewReader(demoAvatarPNG))
	if err != nil {
		return nil
	}
	return img
}

// demoAppIcon 从 exe 的 APP 图标资源加载大图标: 任务栏 / Alt+Tab / 托盘共用。
// 资源由 winres/winres.json 经 `xc res make` 生成 .syso 编进程序, 不用运行时写临时文件。
// LR_SHARED 加载的图标归系统共享管理, 不应 DestroyIcon。
func demoAppIcon() uintptr {
	return wapi.LoadImageW(wapi.GetModuleHandleW(""), common.StrPtr("APP"),
		wapi.IMAGE_ICON, 0, 0, wapi.LR_SHARED|wapi.LR_DEFAULTSIZE)
}

// demoAppSmallIcon 加载任务栏预览窗口左上角的小图标(也会影响任务管理器)。
func demoAppSmallIcon() uintptr {
	return wapi.LoadImageW(wapi.GetModuleHandleW(""), common.StrPtr("APP"),
		wapi.IMAGE_ICON, 24, 24, wapi.LR_SHARED)
}
