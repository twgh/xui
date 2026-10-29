package xui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFontFileForFamily 校验小字号字体的文件解析链路(DESIGN 第八节 149):
// 配置里的中文族名要被解析成实际存在的字体文件, 不存在的族名必须返回空串
// (调用方据此回退 font.NewEX)。
func TestFontFileForFamily(t *testing.T) {
	path := fontFileForFamily("微软雅黑")
	if path == "" {
		t.Skip("系统未安装微软雅黑, 跳过")
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".ttc", ".otf":
	default:
		t.Fatalf("解析出的不是字体文件: %q", path)
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		t.Fatalf("字体文件不可用: %q, err=%v", path, err)
	}
	// 结果要缓存: 第二次调用必须是同一个路径
	if again := fontFileForFamily("微软雅黑"); again != path {
		t.Fatalf("缓存未命中: %q != %q", again, path)
	}
	if p := fontFileForFamily("xui-no-such-font-family"); p != "" {
		t.Fatalf("不存在的字族不应解析出文件: %q", p)
	}
}
