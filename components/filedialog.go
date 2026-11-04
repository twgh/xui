package ui

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/wapi/wutil"
	"github.com/twgh/xcgui/xc"
)

// FileFilter 是文件对话框的类型过滤器。
//
// Name 显示在类型下拉框里, Patterns 是分号分隔的通配符(如 "*.png;*.jpg"),
// 对话框里显示为 "图片 (*.png;*.jpg)"。Name 留空时直接显示 Patterns。
type FileFilter struct {
	Name     string
	Patterns string
}

// FileDialog 是系统原生文件对话框的门面: 打开单个/多个文件、保存文件、选择文件夹。
//
// 对话框由系统绘制(不套 xui 主题), 只负责取路径, 不做读写; 取消时返回空串 / nil。
// 必须在 UI 线程(事件回调)里调用 —— 原生模态对话框会自己跑消息循环。
//
//	fd := ui.NewFileDialog(w.Handle,
//		ui.WithFileDialogTitle("打开图片"),
//		ui.WithFileDialogFilters(
//			ui.FileFilter{Name: "图片", Patterns: "*.png;*.jpg"},
//			ui.FileFilter{Name: "所有文件", Patterns: "*.*"}))
//	if path := fd.Open(); path != "" {
//		// 使用 path
//	}
type FileDialog struct {
	hWindow     int
	title       string
	filters     []FileFilter
	initialDir  string
	defaultName string
	defaultExt  string
	maxFiles    int
}

// FileDialogOption 是 FileDialog 的配置项。
type FileDialogOption func(*FileDialog)

// WithFileDialogTitle 设置对话框标题(默认"打开文件"/"保存文件"/"选择文件夹")。
func WithFileDialogTitle(title string) FileDialogOption {
	return func(d *FileDialog) { d.title = title }
}

// WithFileDialogFilters 设置类型过滤器; 不传则下拉框里没有类型选项。
func WithFileDialogFilters(filters ...FileFilter) FileDialogOption {
	return func(d *FileDialog) { d.filters = append([]FileFilter(nil), filters...) }
}

// WithFileDialogInitialDir 设置初始目录(文件对话框用; 选择文件夹由系统决定初始位置)。
func WithFileDialogInitialDir(dir string) FileDialogOption {
	return func(d *FileDialog) { d.initialDir = dir }
}

// WithFileDialogDefaultName 设置保存对话框里的默认文件名。
func WithFileDialogDefaultName(name string) FileDialogOption {
	return func(d *FileDialog) { d.defaultName = name }
}

// WithFileDialogDefaultExt 设置保存时用户没写扩展名则自动补上的扩展名(可带点)。
func WithFileDialogDefaultExt(ext string) FileDialogOption {
	return func(d *FileDialog) { d.defaultExt = strings.TrimPrefix(ext, ".") }
}

// WithFileDialogMaxFiles 设置多选时最多返回几个文件(默认 10)。
//
// 这个值只决定缓冲区大小, 不能限制用户在对话框里勾选多少个; 勾选超过时只返回前 maxFiles 个。
// 勾选特别多、超出缓冲区时系统会返回空(与 wutil.OpenFiles 一致)。
func WithFileDialogMaxFiles(n int) FileDialogOption {
	return func(d *FileDialog) { d.maxFiles = n }
}

// NewFileDialog 创建文件对话框门面。hWindow 是父窗口句柄(0 = 无父窗口)。
func NewFileDialog(hWindow int, opts ...FileDialogOption) *FileDialog {
	d := &FileDialog{hWindow: hWindow, maxFiles: 10}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Open 打开单个文件, 返回完整路径; 取消返回空串。
func (d *FileDialog) Open() string {
	opt := d.options()
	if paths := wutil.OpenFileEx(opt); len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// OpenMulti 打开多个文件, 返回完整路径; 取消返回 nil。
func (d *FileDialog) OpenMulti() []string {
	opt := d.options()
	maxFiles := d.maxFiles
	if maxFiles <= 0 {
		maxFiles = 10
	}
	opt.MaxOpenFiles = uint32(maxFiles)
	opt.Flags = wapi.OFN_ALLOWMULTISELECT | wapi.OFN_EXPLORER | wapi.OFN_PATHMUTEXIST
	return wutil.OpenFileEx(opt)
}

// Save 打开另存为对话框, 返回完整路径; 取消返回空串。同名文件已存在时系统会提示覆盖。
func (d *FileDialog) Save() string {
	return wutil.SaveFileEx(d.options())
}

// PickDir 打开"选择文件夹"对话框, 返回完整路径; 取消返回空串。
func (d *FileDialog) PickDir() string {
	var hwnd uintptr
	if d.hWindow > 0 {
		hwnd = xc.XWnd_GetHWND(d.hWindow)
	}

	displayName := make([]uint16, syscall.MAX_PATH)
	title := d.title
	if title == "" {
		title = "选择文件夹"
	}
	lpszTitle, _ := syscall.UTF16PtrFromString(title)

	bi := wapi.BrowseInfoW{
		HwndOwner:      hwnd,
		PszDisplayName: &displayName[0],
		LpszTitle:      lpszTitle,
		UlFlags:        wapi.BIF_USENEWUI,
	}
	pidl := wapi.SHBrowseForFolderW(&bi)
	if pidl == 0 {
		return ""
	}
	var path string
	if !wapi.SHGetPathFromIDListW(pidl, &path) {
		return ""
	}
	return path
}

// options 组装 wutil 的通用参数; Flags 留给各方法自己定(0 = wutil 的默认值)。
func (d *FileDialog) options() wutil.OpenFileOption {
	opt := wutil.OpenFileOption{
		Title:       d.title,
		DefDir:      d.initialDir,
		DefFileName: d.defaultName,
		DefExt:      d.defaultExt,
		Filters:     d.filterPairs(),
	}
	if d.hWindow > 0 {
		opt.HwndOwner = xc.XWnd_GetHWND(d.hWindow)
	}
	return opt
}

// filterPairs 把 FileFilter 翻成 wutil 要的"描述, 通配符"扁平数组。
func (d *FileDialog) filterPairs() []string {
	if len(d.filters) == 0 {
		return nil
	}
	pairs := make([]string, 0, len(d.filters)*2)
	for _, f := range d.filters {
		patterns := strings.TrimSpace(f.Patterns)
		if patterns == "" {
			continue
		}
		name := strings.TrimSpace(f.Name)
		if name == "" {
			name = patterns
		}
		pairs = append(pairs, fmt.Sprintf("%s (%s)", name, patterns), patterns)
	}
	return pairs
}
