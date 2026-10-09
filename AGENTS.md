# AGENTS.md — xui 项目会话规则

shadcn 风格的 xcgui 自绘组件库(copy-not-install 分发)。**本文件只放硬规则与"去哪查",不做内容副本**:

- **docs/HANDOFF.md** — 当前进度 / 待办 / 仓库结构 / 常用命令 / 视觉验证工作流
- **docs/DESIGN.md** — 知识库:架构原理(一~四节) + **xcgui 踩坑全集(第八节, 按编号)** + API 清单(第九节);**交互语义权威汇总在第四节末**
- **README.md** — 面向用户的安装与用法
- 本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn\src\components\ui\*.tsx`,令牌在 `src/index.css`)是交互语义的**唯一权威**

## 硬规则

- **绘制坐标是逻辑像素**:`XEle_GetClientRect` 返回未缩放矩形,框架渲染时统一 DPI 变换;自绘**禁止**再按 DPI 预缩放(见 DESIGN 6/10)
- **组件禁止缓存颜色**:paint 回调里 `xui.Current()` 实时取令牌;需要同步原生 API 的颜色用 `xui.OnThemeChangeFor(元素句柄, fn)`(元素销毁时自动反注册;仅非元素的长生命周期订阅才用裸 `OnThemeChange`)(见 DESIGN 3.2/3.3)
- **事件一律 `AddEvent*` 系列**(有 2000 回调上限问题);不用背景管理器管理组件外观(见 DESIGN 2/3)
- **渲染固定 Direct2D**:`app.New(true)`,demo 是标准样板(含无图标无三键标题栏样式,见 DESIGN 38)
- **改交互语义前**:先查 DESIGN 第四节末的语义汇总表,再读本地 shadcn 源码 —— 勿凭记忆改动
- **界面布局禁止绝对坐标**:窗口/页面/卡片全部走流式布局。三条会踩的语义 —— ① `Fill` 是"父级整尺寸(减自身边距)"**不是剩余空间**, 竖向堆叠用它必溢出;② 要"剩余"用 `Weight`, 但它只在**父级尺寸确定**(窗口直属 / Fixed / Weight 解析值)时生效, 父级是 `Fill`/`Auto` 时恒为 0(所以竖向主链从窗口开始一路 `Weight`, 中间别垫 `Fill` 容器);③ 卡片高度要随内容就用 `Auto` 自撑, 要长高的面板则整条链全 `Weight`(见 DESIGN 8.55/8.56, 样板 `demo/main.go`)
- **Auto 高的卡片, 外面那一层(行里的"列")高度也必须是 `Auto`**: 列高用 `Fill`/`Weight`/`Fixed` 时, 卡片内容区里的整宽控件(原生 `Edit`/`ComboBox`/分隔线)会按**卡片宽**排版, 右边缘压到卡片边框上;中间也**不要**再垫一层, 行高也别用 `Auto`(Auto 行里 `Weight` 宽的列会塌成 0 宽)。栅格一律"一行 + `Weight` 宽的 `Auto` 高列 + 卡片竖排", 见 DESIGN 8.59

## 运行时基建(踩过坑, 照做)

- **hover 过渡**一律用运行时 `xui.Hover(hEle, 悬停中)` / `xui.HoverItem(hOwner, 索引, 悬停中)` 取值 + `xui.Fade(静止色, 悬停色, hv)` 取色(叠加层用 `ScaleAlpha(hv)`);**不要在组件里自己起定时器或缓存过渡值**(见 DESIGN 45)
- **应用初始化 `a.SetPaintFrequency(10)`**:xcgui 默认有 30ms 重绘节流(hover 过渡只有约 5 帧的根因),设 10 后实测 30.9 → 59.0 fps。`XC_SetTextRenderingHint` 在 D2D 下是空操作,不要用(见 DESIGN 46)
- **弹层/对话框进出场**用 `xui.Presence(...)` + `SetElementsAlpha / FadeWindowAlpha / SlideWindowAlpha`,**不要自己起定时器**;过渡挂**宿主窗口**、别挂弹层小窗;**异型透明窗(`Window_Transparent_Shaped`)上不要用整窗 alpha 做动画** —— 它只认第一次调用、之后永远冻结, 淡入淡出改成元素 alpha(见 DESIGN 8.57);**组件自建的弹层小窗不在主题管理器的重绘清单里, 要自己 `OnThemeChangeFor` 订阅并重绘**(否则切主题时挂在屏幕上的弹层还是旧配色, 见 DESIGN 8.58);窗口 alpha 永远不精确 = 0(0 会让 xcgui 停止合成);要 Win32 HWND 用窗口 wrapper 的 `GetHWND()`,别拿 xcgui 窗口句柄喂 `XWidget_GetHWND`(弹模态错误框,表现为"点了没反应")(见 DESIGN 47)

## 常用命令

```bash
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe   # demo = 全组件视觉验证目标
cd D:\GOProject\src\ui\shadcn && pnpm dev    # shadcn 参考实现, localhost:5173
go run ./cmd/xui list                        # CLI 本地测试
go test ./...                                # 图标解析器 + 资源完整性 + 缓动曲线
```

## 工作流惯例

- 视觉改动必须截图放大验证:主模型有视觉就直接看 + crop 放大;像素级数值用脚本(PIL)测量,不靠目测;无视觉的模型用 vision CLI(`C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`)(完整流程见 HANDOFF)
- 疑难渲染问题写探针程序二分定位,完事删除
- 每次发现 xcgui 未文档化的行为,补进 `docs/DESIGN.md` 第八节(新条目编号往后排;交互语义类补进第四节末的表)
