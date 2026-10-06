# PLAN: 图标抗锯齿渲染 — 方案二(直接 D2D 几何填充)

> **状态: 已完成(2026-10-06)。** 实现落在 `d2dicon.go` + `icons.go`, 结论已浓缩进
> `docs/DESIGN.md` 第八节 43, 本文件仅作计划存档。执行中发现两处本计划没覆盖的硬坑,
> 以 DESIGN.md 第 43 条为准:
> ① 直通 D2D 调用拿不到"当前元素"的变换/裁剪(炫彩只在刷新自己的 XDraw 命令时才施加),
>    必须先用 `XDraw_ConvRect` 把客户区坐标换算成画布对齐物理坐标;
> ② `ID2D1SimplifiedGeometrySink` 除 `Close` 外全是 void, 把 `EndFigure` 的垃圾返回值
>    当 HRESULT 检查会每次都静默回落折线。另外"XDraw 线条无 AA"这一前提不准确:
>    实测线条也有 AA, 旧图标难看的真因是折线点取整到逻辑像素 + 多边形棱角。

> 交接计划。上一轮已用探针(`probe/main.go`, 保留勿删, 实现完成后再删)把整条 D2D
> 管线验证通了, 本文件自包含全部已验证事实与实现步骤, 按此执行即可, 无需重新探路。

## 0. 背景与根因

- 现状: `icons.go` 把 lucide path 解析成折线, 用 `XDraw_DrawLine/DrawLineF` 描边 → 锯齿。
- 根因: **XDraw 的线条图元是无抗锯齿的**。`BeginPaint` 已调用 `EnableSmoothingMode(true)`,
  对线条无效(实测); `FillRound` 等填充平滑是因为走另一条带 AA 的管线。
- 平台约定: xcgui 只用于 Windows; 渲染固定 Direct2D(`app.New(true)`); 允许写汇编。
- lucide 是描边图标(stroke-width 2, round cap/join, 24x24 viewBox)。

## 1. 探针已实证的事实(全部可信, 勿再猜)

### 1.1 获取 COM 对象(xcgui 层, 无坑)

```go
rt      := xc.XDraw_GetD2dRenderTarget(hDraw) // paint 回调内, 真 *ID2D1RenderTarget
factory := xc.XC_GetD2dFactory()
// 另有 XC_GetWicFactory / XC_GetDWriteFactory / XDraw_GetD2dBitmap, 本方案用不到
```

- 已用 `rt.GetFactory()`(见下)反查验证: rt 是真 COM 对象且 factory 一致。
- paint 回调内可以: 建画刷、建几何、FillGeometry, 坐标系 = 元素逻辑像素
  (与 XDraw_* 相同, DPI 由 RT 自身变换承担, 实测 150% 缩放下位置尺寸正确)。
- RT 的 AA 模式已是 PER_PRIMITIVE(GetAntialiasMode=0)。

### 1.2 纯 Go COM 调用(vcall)

```go
func vcall(obj uintptr, index int, args ...uintptr) uintptr {
    vtbl := *(*uintptr)(unsafe.Pointer(obj))
    fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(vtbl)))
    r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
    return r
}
```

**ABI 硬规则**(违反任一条即轻则帧被毁/窗口隐形, 重则崩溃):
1. Win64 浮点参数走 XMM 寄存器, `SyscallN` 传不了 → **带 float 形参的方法一律禁调**
   (DrawGeometry 的 strokeWidth、DrawLine 全家、SetTransform 的矩阵是按指针传所以没事)。
2. 8 字节结构体按值传 = 打包成一个 64 位整数放 GP 寄存器:
   `D2D1_POINT_2F` → `uintptr(f32bits(x)) | uintptr(f32bits(y))<<32`。
3. 其余全用指针/整数 → SyscallN 安全。

### 1.3 已验证的 vtable 索引

ID2D1RenderTarget(基链: IUnknown 0-2, **ID2D1Resource::GetFactory 占 3**, 方法从 4 起;
注意 Create* 之后 Draw/Fill 混排, 别按字母序想当然):

```
CreateSolidColorBrush = 8   // (color *D2D1_COLOR_F, props *D2D1_BRUSH_PROPERTIES(NULL 可), out **brush)
DrawGeometry          = 22  // 带 float 线宽 → 禁调
FillGeometry          = 23  // (geometry, brush, layer=NULL) ← 本方案的唯一出图口
SetTransform          = 30  // (matrix *D2D1_MATRIX_3X2_F) 指针传参可用, 但别乱调!
GetAntialiasMode      = 33  // 返回 0=PER_PRIMITIVE(实测)
```

ID2D1Factory(直接继承 IUnknown, 方法从 3 起):

```
CreateEllipseGeometry = 7   // (ellipse *D2D1_ELLIPSE, out **geom) 24 字节结构按指针
CreatePathGeometry    = 10  // (out **geom)
```

ID2D1PathGeometry(继承 ID2D1Geometry 的 13 个方法, 所以):

```
Open = 17  // (out **sink) → HRESULT; 新几何必须 Open, 用完必须 sink.Close()
GetBounds = 4  // (transform=NULL, out *D2D1_RECT_F) — 验证几何合法性的手段:
               // 未 Close 的几何返回 D2DERR_WRONG_STATE(0x88990001)
```

ID2D1GeometrySink(**注意: Simplified 部分只有 7 个自有方法**, EndFigure=8/Close=9;
GeometrySink 追加的 AddBezier(10)/AddQuadraticBezier(11)/AddQuadraticBeziers(12)/AddArc(13)
都带指针参数, 别乱调):

```
SetFillMode  = 3  // (D2D1_FILL_MODE) ← 描边展开叠加轮廓必须设 WINDING(1), 否则
                  //   默认 ALTERNATE 会让重叠的段四边形/圆帽互相镂空!
BeginFigure  = 5  // (POINT_2F 按 64 位打包, D2D1_FIGURE_BEGIN: FILLED=0/HOLLOW=1)
AddLines     = 6  // (points *[]D2D1_POINT_2F(即 *float32 扁平数组), count)
EndFigure    = 8  // (D2D1_FIGURE_END: OPEN=0/CLOSED=1)
Close        = 9  // () HRESULT
```

### 1.4 血泪教训(每条都是实测踩出来的)

- vtable 索引错位 = 调到邻居方法。曾把 30 当 GetAntialiasMode 实为 **SetTransform**,
  用垃圾指针设了变换 → EndDraw 报错 → xcgui 整窗不再呈现 → **窗口彻底隐形(连边框都没)**,
  且进程不崩、日志正常, 极难察觉。改索引必须像探针那样用 GetBounds/返回值交叉验证。
- 几何未 Close(索引错导致 EndFigure/Close 调错) → FillGeometry 用它 → 同样整帧错误态。
- 用 `GetBounds`(几何 idx 4)当"几何是否合法"的探针, hr=0 且 bounds 合理才可用。
- 验证窗口可见性别信"窗口矩形/IsWindowVisible/前台"——帧毁了这些全正常, 只信截图。
  另: 新起进程的窗口可能被前台竞争压在下面, 置前用
  `(New-Object -ComObject WScript.Shell).AppActivate('窗口标题')`, 实测可靠。
- 探针还发现: sink 索引错时 BeginFigure 的 figureBegin 会读到浮点位模式当枚举,
  静默污染几何——所以**每次 FillGeometry 前置校验是必要的**(见步骤 4)。

## 2. 实现步骤

### 步骤 1: 新建 `d2dicon.go`(xui 根目录, 与 icons.go 同包)

内容 = 探针里验证过的部分搬正 + 缓存:

```go
// vcall / f32 / colorF{R,G,B,A float32} / 1.3 节全部索引常量
// createSolidBrush(rt, colorF) uintptr
// makeGeometry(factory, figures [][]float32扁平点) — SetFillMode(WINDING) →
//   每条 figure: BeginFigure(打包首点, FILLED) → AddLines → EndFigure(CLOSED) → 最后 Close
//   全部 hr 校验, 任一非 0 → 返回 0(调用方回落折线渲染)
// 几何缓存: map[iconKey]uintptr, key = 图标path串哈希 + iconSize(int32) + strokeW
//   附加 rt 指针字段: 换 RT(设备重建)即整体作废重建 — 简单做法:
//   cache struct{ rt, key uintptr; geom uintptr }, rt 不同就全部 Release/重建
//   ( Release = vcall(geom, 2) — IUnknown::Release, 可选; 不 Release 泄漏有限可接受,
//     图标集合有限, 但换 RT 必须清 )
```

### 步骤 2: 描边展开器(纯 Go, `icons.go` 或 d2dicon.go)

输入: parseIconPath 的产物 `[][]iconPt`(24 网格坐标, 已含圆弧转贝塞尔离散点),
线宽 w(lucide 2), 目标边长 s(iconSize 逻辑像素)。**先缩放后展开**:
把点乘 `s/24` 得到最终逻辑坐标, 再按 w_logical = w*s/24 展开 — 这样缓存 key 只需
(图标, s), 几何里就是最终坐标, 一次成型。

展开规则(lucide strokeLinecap/join 都是 round):
- 每条折线段的两侧各偏移 w/2 → 一个四边形(4 点 AddLines, CLOSED);
- 每个顶点(段与段的连接处)补一个圆帽多边形: 半径 w/2, 12~16 边形, CLOSED;
- 每条子路径的两个端点也补圆帽(round cap);
- 所有 figure 进同一 geometry, **必须 SetFillMode(WINDING)**, 重叠区才会实心。
- 自交风险: 无 — 全部形状是凸多边形, WINDING 下并集即笔画形状。

### 步骤 3: `Painter.Icon` 改造(icons.go)

```go
func (p *Painter) Icon(paths string, rc *xc.RECT, color Color, strokeWidth float32) {
    // 尝试 D2D:
    rt := xc.XDraw_GetD2dRenderTarget(hDraw)  // 需要拿到 hDraw — Painter 已有
    if rt != 0 && d2dOK {
        geom := cachedGeometry(paths, iconSize, strokeWidth)
        brush := createSolidBrush(rt, colorToColorF(color))
        vcall(rt, rtFillGeometry, geom, brush, 0)
        return
    }
    // 回落: GDI 模式(app.New(false))或 D2D 不可用时, 保留现有折线 DrawLine 渲染
}
```

- `IconSearch`(icons.go 里的椭圆特例)改成 ellipse geometry 或折线化, 走同一管线。
- brush 每帧新建(探针实测代价可忽略); 想省就按 (rt,color) 缓存。
- 图标墨迹尺寸: 现有 button.go 用 iconSize=18(18 而非 16 的理由见其注释), 不动。

### 步骤 4: 验证清单(全部要过)

1. `go build ./... && go vet ./...`
2. demo 四个图标(eye/eye-off/sun/moon)放大截图: 边缘平滑无锯齿, 与浏览器 lucide
   对照(dev 工具 18px 图标); 像素级检查可用 PIL 采样边缘灰度过渡。
3. 主题切换(亮↔暗)图标颜色随令牌; hover 变色正常(Button 的 textColor 随状态)。
4. DPI: 100%/125%/150% 各截一次, 图标大小与位置一致(几何是逻辑像素)。
5. GDI 回落: 临时 `app.New(false)` 跑 demo, 图标应为旧折线渲染且不崩。
6. 回归: checkbox/radio/slider/switch 等所有走 Painter 的组件不变形(BeginPaint 没动)。
7. 结尾: 删 `probe/`、`probe/log.txt`; **把 1.x 节的 D2D 行为结论浓缩后补进
   docs/DESIGN.md 第八节**(xcgui 暴露真 COM 指针 + 索引表 + ABI 三规则 + 隐形窗口坑);
   更新 docs/HANDOFF.md; git 提交。

## 3. 风险与备注

- COM 引用计数: geometry/brush 不 Release 只是小量泄漏(图标集合有限); 但缓存必须绑定
  RT 指针, 设备重建后旧几何不可用(见步骤 1)。
- 若将来需要真描边(非 round cap 的图形), 路线: 小段 Go 汇编 shim 加载 XMM 寄存器调
  DrawGeometry(用户已允许写汇编; gonutz/d2d1 有现成参考实现)。当前 round-cap 展开
  方案不需要它。
- 探针残留状态: probe/main.go 里部分 fmt/日志是调试残留, 只作参考, 不照抄结构。
