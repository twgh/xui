# AGENTS.md — xui 项目会话规则

shadcn 风格的 xcgui 自绘组件库(copy-not-install 分发)。当前 P0 运行时 + P1 组件已完成并视觉验收,详见 **docs/HANDOFF.md**(交接状态)、**docs/DESIGN.md**(架构 + xcgui 踩坑清单,第八节必读)、**README.md**(用法)。

## 硬规则

- **绘制坐标是逻辑像素**:`XEle_GetClientRect` 返回未缩放矩形,框架渲染时统一 DPI 变换;自绘**禁止**再按 DPI 预缩放
- **组件禁止缓存颜色**:paint 回调里 `xui.Current()` 实时取令牌;需要同步原生 API 的颜色用 `xui.OnThemeChange`
- **事件一律 `AddEvent*` 系列**(有 2000 回调上限问题);不使用背景管理器管理组件外观
- **渲染固定 Direct2D**:`app.New(true)`,demo 是标准样板(含无图标无三键标题栏样式)
- 修改组件交互语义前,**先读 `D:\GOProject\src\ui\shadcn\src\components\ui\*.tsx`**——本地 shadcn 参考实现是语义的唯一权威(令牌在 `src/index.css`)

## 交互语义速查(与 shadcn 对齐,勿凭记忆改动)

hover 透明度(/80 等)、focus-visible 键盘才显示焦点框(ring-3 ring-ring/50 + 边框变 Ring 色;文本输入框点击也显示)、disabled 原色 opacity-50、active:translate-y-px 按压下沉 1px(文字需手动绘制)、destructive 软色调。焦点来源跟踪在 `focus.go`。

## 常用命令

```bash
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe   # demo = 全组件视觉验证目标
cd D:\GOProject\src\ui\shadcn && pnpm dev    # shadcn 参考实现, localhost:5173
go run ./cmd/xui list                        # CLI 本地测试
```

## 工作流惯例

- 视觉改动必须截图放大验证:主模型有视觉就直接看 + crop 放大;像素级数值用 PIL 脚本测量,不靠目测;无视觉的模型用 vision CLI(`C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`)
- 疑难渲染问题写探针程序二分定位,完事删除
- 每次发现 xcgui 未文档化的行为,补进 docs/DESIGN.md 第八节
