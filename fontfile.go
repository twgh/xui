package xui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// xcgui 在字号 < 10pt 时会把**常规字重**替换成宋体(见 DESIGN 第八节 149):
// 不论请求的是微软雅黑还是 Arial, 9pt 常规都画成宋体, 10pt 及以上才正常;
// 9pt 粗体也正常。绕法是绕开 XFont_CreateEx 的字族解析 —— 直接按字体文件
// 创建(font.NewByFile), 字号与字族都保持不变。这里只负责把字族名解析成
// 系统字体文件: GDI+ 取英文族名(配置里的 "微软雅黑" 在注册表里登记为
// "Microsoft YaHei"), 再查 Fonts 注册表(当前用户 + 本机)拿文件。
//
// 解析失败(字体不存在、注册表里没有对应表项等)时返回空串, 调用方回退到
// 原来的 font.NewEX, 也就是接受 xcgui 的小字号替换。
const (
	hkeyLocalMachine = 0x80000002
	hkeyCurrentUser  = 0x80000001
	keyRead          = 0x20019
	langEnglish      = 0x0409 // GdipGetFamilyName 的英文族名
	fontsRegPath     = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`
)

var (
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegEnumValueW    = advapi32.NewProc("RegEnumValueW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")

	gdiplus            = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup = gdiplus.NewProc("GdiplusStartup")
	procGdipCreateFam  = gdiplus.NewProc("GdipCreateFontFamilyFromName")
	procGdipGetFamName = gdiplus.NewProc("GdipGetFamilyName")
	procGdipDeleteFam  = gdiplus.NewProc("GdipDeleteFontFamily")
	gdiplusOnce        sync.Once
	gdiplusReady       bool

	fontFilesMu sync.Mutex
	fontFiles   = map[string]string{} // 字族名 -> 字体文件(空串 = 解析失败)
)

// gdiplusStartupInput 对应 GdiplusStartupInput(4 + 对齐 + 指针 + 两个 BOOL)。
type gdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

// fontFileForFamily 返回字族常规字重对应的字体文件路径; 解析失败返回空串。
// 结果按字族缓存(包括失败), 不会在绘制路径上反复读注册表。
func fontFileForFamily(family string) string {
	if family == "" {
		return ""
	}
	fontFilesMu.Lock()
	defer fontFilesMu.Unlock()
	if p, ok := fontFiles[family]; ok {
		return p
	}
	p := lookupFontFile(family)
	fontFiles[family] = p
	return p
}

// lookupFontFile 按"配置里的族名"和"英文族名"在 Fonts 注册表里找文件。
// 只认 " & " 分隔的第一段: .ttc 以文件里的第 0 号字形为准(msyh.ttc 第 0 个
// 是 Microsoft YaHei, UI 变体在后面), 其他段名的文件会加载错字形。
func lookupFontFile(family string) string {
	names := []string{family}
	if en := englishFamilyName(family); en != "" && !strings.EqualFold(en, family) {
		names = append(names, en)
	}
	for _, root := range []uintptr{hkeyLocalMachine, hkeyCurrentUser} {
		if f := findFontFileInRoot(root, names); f != "" {
			return f
		}
	}
	return ""
}

// englishFamilyName 用 GDI+ 取字族的英文名(GDI+ 能吃本地化族名, 且
// GdipGetFamilyName 带 LANG_ENGLISH 时返回注册表用的英文名)。
func englishFamilyName(family string) string {
	gdiplusOnce.Do(func() {
		var token uintptr
		in := gdiplusStartupInput{GdiplusVersion: 1}
		if r, _, _ := procGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)),
			uintptr(unsafe.Pointer(&in)), 0); r != 0 {
			return
		}
		gdiplusReady = true // token 不回收: 进程生命周期内保持 GDI+ 可用
	})
	if !gdiplusReady {
		return ""
	}
	p, err := syscall.UTF16PtrFromString(family)
	if err != nil {
		return ""
	}
	var fam uintptr
	if r, _, _ := procGdipCreateFam.Call(uintptr(unsafe.Pointer(p)), 0,
		uintptr(unsafe.Pointer(&fam))); r != 0 || fam == 0 {
		return ""
	}
	defer procGdipDeleteFam.Call(fam)
	var buf [64]uint16
	if r, _, _ := procGdipGetFamName.Call(fam, uintptr(unsafe.Pointer(&buf[0])),
		langEnglish); r != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}

// findFontFileInRoot 枚举一个注册表根下的字体表项, 返回第一个匹配族名的文件。
func findFontFileInRoot(root uintptr, names []string) string {
	sub, err := syscall.UTF16PtrFromString(fontsRegPath)
	if err != nil {
		return ""
	}
	var key uintptr
	if r, _, _ := procRegOpenKeyExW.Call(root, uintptr(unsafe.Pointer(sub)), 0,
		keyRead, uintptr(unsafe.Pointer(&key))); r != 0 {
		return ""
	}
	defer procRegCloseKey.Call(key)

	for i := uint32(0); ; i++ {
		nameBuf := make([]uint16, 512)
		n := uint32(len(nameBuf)) // 返回时是字符数(不含结尾 0)
		var typ uint32
		if r, _, _ := procRegEnumValueW.Call(key, uintptr(i),
			uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(unsafe.Pointer(&n)),
			0, uintptr(unsafe.Pointer(&typ)), 0, 0); r != 0 {
			return "" // ERROR_NO_MORE_ITEMS 或读失败: 没有匹配项
		}
		display := syscall.UTF16ToString(nameBuf[:n])
		fam := firstFamilyName(display)
		if !matchFamily(fam, names) {
			continue
		}
		file := regStringValue(key, nameBuf[:n])
		if file == "" {
			continue
		}
		if !filepath.IsAbs(file) {
			file = filepath.Join(os.Getenv("WINDIR"), "Fonts", file)
		}
		if st, err := os.Stat(file); err == nil && !st.IsDir() {
			return file
		}
	}
}

// firstFamilyName 从注册表表项名("Microsoft YaHei & Microsoft YaHei UI
// (TrueType)")里取出第一段族名。带空格的多族名以 " & " 分隔; 单族名只是
// 去掉 "(TrueType)"/"(OpenType)" 后缀。
func firstFamilyName(display string) string {
	s := display
	for _, suffix := range []string{" (TrueType)", " (OpenType)"} {
		if strings.HasSuffix(s, suffix) {
			s = strings.TrimSuffix(s, suffix)
			break
		}
	}
	if i := strings.Index(s, " & "); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// matchFamily 族名是否匹配(忽略大小写)。
func matchFamily(fam string, names []string) bool {
	for _, n := range names {
		if strings.EqualFold(fam, n) {
			return true
		}
	}
	return false
}

// regStringValue 读取注册表字符串值。name 是值名(不含结尾 0)。
func regStringValue(key uintptr, name []uint16) string {
	p := make([]uint16, len(name)+1) // 注册表 API 要求名字以 0 结尾
	copy(p, name)
	var size uint32
	if r, _, _ := procRegQueryValueExW.Call(key, uintptr(unsafe.Pointer(&p[0])),
		0, 0, 0, uintptr(unsafe.Pointer(&size))); r != 0 || size == 0 {
		return ""
	}
	buf := make([]uint16, (size+1)/2)
	if r, _, _ := procRegQueryValueExW.Call(key, uintptr(unsafe.Pointer(&p[0])),
		0, 0, uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size))); r != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
