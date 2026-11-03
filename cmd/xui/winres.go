package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// projectIconICO 是新建项目的默认图标(xcgui 品牌图标)。
//
//go:embed assets/app.ico
var projectIconICO []byte

// projectWinresJSONTemplate 是 `xc res init` 生成的完整骨架
// (RT_GROUP_ICON / RT_MANIFEST / RT_VERSION 三段全在, 不是只放图标的精简版),
// 只把版本信息里的占位文案换成项目名。资源名 APP 与文档示例统一。
const projectWinresJSONTemplate = `{
  "RT_GROUP_ICON": {
    "APP": {
      "0000": "icon.ico"
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0804": {
        "identity": {
          "name": "",
          "version": ""
        },
        "description": "",
        "minimum-os": "win7",
        "execution-level": "as invoker",
        "ui-access": false,
        "auto-elevate": false,
        "dpi-awareness": "per monitor v2",
        "disable-theming": false,
        "disable-window-filtering": false,
        "high-resolution-scrolling-aware": false,
        "ultra-high-resolution-scrolling-aware": false,
        "long-path-aware": true,
        "printer-driver-isolation": false,
        "gdi-scaling": false,
        "segment-heap": true,
        "use-common-controls-v6": false
      }
    }
  },
  "RT_VERSION": {
    "#1": {
      "0000": {
        "fixed": {
          "file_version": "0.0.0.1",
          "product_version": "0.0.0.1"
        },
        "info": {
          "0804": {
            "Comments": "由 xui 创建",
            "CompanyName": "",
            "FileDescription": {{NAME}},
            "FileVersion": "0.0.0.1",
            "InternalName": {{NAME}},
            "LegalCopyright": "",
            "LegalTrademarks": "",
            "OriginalFilename": {{EXE}},
            "PrivateBuild": "",
            "ProductName": {{NAME}},
            "ProductVersion": "0.0.0.1",
            "SpecialBuild": ""
          }
        }
      }
    }
  }
}
`

// projectWinresJSON 用项目名填充模板; 项目名可能含 JSON 特殊字符, 统一走 json.Marshal 转义。
func projectWinresJSON(name string) string {
	return strings.NewReplacer(
		"{{NAME}}", jsonString(name),
		"{{EXE}}", jsonString(name+".exe"),
	).Replace(projectWinresJSONTemplate)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// scaffoldProjectWinres 为新建项目准备 winres/(icon.ico + winres.json)并生成 .syso。
// 已存在的文件跳过(重复 init 不覆盖用户改过的图标/版本信息)。
// allowInstall=false(--no-install)时只用 PATH 里现成的 xc, 不联网安装。
func scaffoldProjectWinres(dir, name string, allowInstall, dryRun bool) {
	winresDir := filepath.Join(dir, "winres")
	iconPath := filepath.Join(winresDir, "icon.ico")
	jsonPath := filepath.Join(winresDir, "winres.json")
	if dryRun {
		// 预览: 连目录都不建, 也不叫 xc 生成 syso(它会联网 go install)。
		for _, p := range []string{iconPath, jsonPath} {
			if fileExists(p) {
				info("跳过 %s (已存在)", p)
			} else {
				info("将创建 %s", p)
			}
		}
		if !fileExists(iconPath) || !fileExists(jsonPath) {
			info("将生成 %s", filepath.Join(dir, "rsrc_windows_{386,amd64}.syso"))
		}
		return
	}
	if err := os.MkdirAll(winresDir, 0o755); err != nil {
		warn("创建 winres 目录失败: %v", err)
		return
	}
	if _, err := os.Stat(iconPath); err == nil {
		info("跳过 %s (已存在)", iconPath)
	} else if err := os.WriteFile(iconPath, projectIconICO, 0o644); err != nil {
		warn("写入 %s 失败: %v", iconPath, err)
		return
	} else {
		info("创建 %s", iconPath)
	}
	if _, err := os.Stat(jsonPath); err == nil {
		info("跳过 %s (已存在)", jsonPath)
	} else if err := os.WriteFile(jsonPath, []byte(projectWinresJSON(name)), 0o644); err != nil {
		warn("写入 %s 失败: %v", jsonPath, err)
		return
	} else {
		info("创建 %s", jsonPath)
	}
	if err := winresMake(dir, allowInstall); err != nil {
		warn("生成 .syso 失败: %v", err)
	}
}

// winresMake 是测试缝: 单测替换成桩, 不依赖本机 xc。
var winresMake = makeWinres

// makeWinres 在项目根执行 `xc res make --arch amd64,386`。
// 只出 386/amd64: xcgui 不支持 ARM, 生成 arm64 syso 只会在交叉编译时误导。
func makeWinres(dir string, allowInstall bool) error {
	xc, err := findXc(allowInstall)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, xc, "res", "make", "--arch", "amd64,386")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%v\n%s", err, msg)
		}
		return err
	}
	info("生成 %s", filepath.Join(dir, "rsrc_windows_{386,amd64}.syso"))
	return nil
}

// findXc 定位 xc 可执行文件; 不在 PATH 且允许安装时执行
// `go install github.com/twgh/xc@latest`, 再到 GOBIN / GOPATH/bin 下找。
func findXc(allowInstall bool) (string, error) {
	if p, err := exec.LookPath("xc"); err == nil {
		return p, nil
	}
	if !allowInstall {
		return "", fmt.Errorf("未找到 xc, 请先 go install github.com/twgh/xc@latest, 再在项目根执行 xc res make --arch amd64,386")
	}
	info("未找到 xc, 正在安装 (go install github.com/twgh/xc@latest)...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "install", "github.com/twgh/xc@latest")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return "", fmt.Errorf("安装 xc 失败: %v\n%s", err, msg)
		}
		return "", fmt.Errorf("安装 xc 失败: %v", err)
	}
	name := "xc"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, binDir := range goBinDirs() {
		p := filepath.Join(binDir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("已执行 go install, 但 GOBIN/GOPATH/bin 下没有找到 %s", name)
}

// goBinDirs 返回 go install 可能写入的目录(GOBIN 优先, 然后是 GOPATH/bin)。
func goBinDirs() []string {
	var dirs []string
	if out, err := exec.Command("go", "env", "GOBIN").Output(); err == nil {
		if d := strings.TrimSpace(string(out)); d != "" {
			dirs = append(dirs, d)
		}
	}
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		// GOPATH 可以是多目录(Windows 用 ; 分隔): 必须逐条 Join, 整串拼 "/bin"
		// 会得到一个不存在的路径, go install 之后照样找不到 xc。
		for _, p := range filepath.SplitList(strings.TrimSpace(string(out))) {
			if p != "" {
				dirs = append(dirs, filepath.Join(p, "bin"))
			}
		}
	}
	return dirs
}
