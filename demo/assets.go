package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
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
