# HANDOFF — 交接文档

> **本文只回答"现在到哪了"**:当前状态、待办、命令、仓库结构、验证工作流、去哪查。
> 设计原理 / xcgui 踩坑细节 / 交互语义**一律不在这里重复** —— 见 **DESIGN.md**(第八节 = 踩坑全集按编号;第四节末 = 语义权威汇总表)。agent 会话硬规则见 **AGENTS.md**,面向用户的用法见 **README.md**。
> ⚠️ **DESIGN.md 会随 `xui init` 原样拷进用户项目的 `ui/`**, 按使用者视角写:不写日期、报障时间线、待办、发布流程、仓库结构 —— 那些只放在本文。

## 项目一句话

`github.com/twgh/xui`:为 xcgui(炫彩界面库)提供 **shadcn-ui 风格**的自绘组件——主题令牌、变体系统、源码拷贝分发(copy-not-install)。交互语义以本地 shadcn 参考实现(`D:\GOProject\src\ui\shadcn`)的源码为唯一权威。

## 当前状态(截至 2026-10-19)

**阶段**:P0 运行时 + P1/P2/P3 组件全部完成并视觉验收(**P3 已收尾**, 2026-10-17 补上 combobox 弹层进场动画;CLI 远程 registry 已完成);**v0.1.0 已发布(2026-10-19, GitHub 公开 + tag 已推,依赖 xcgui v1.4.3)**;P4(文档站、主题画廊)未开始。组件覆盖与已知限制见 DESIGN 第七节。

**已完成(括号里是根因/踩坑所在的 DESIGN 第八节条目号)**:

- ✅ **P0 运行时**:Color/Theme 令牌、主题管理器(SetTheme 零成本换肤 + OnThemeChange)、Painter、字体缓存(单位坑见 31)、focus-visible 焦点来源跟踪(16)
- ✅ **P1 组件**:Button(6 变体×8 尺寸档)、Checkbox、Radio、Input、Label、Separator+Badge、Card
- ✅ **P2 组件**:Switch(2 尺寸)、Progress、Slider、Tabs(default/line)、Tooltip、ComboBox、图标系统
- ✅ **Slider 渲染缺陷彻底修复**:悬浮/拖动冒方块、两端 ring 被裁、灰环残留(根因与结论见 25/26/27/28)
- ✅ **CLI**:`init / add / list`(registry.json 驱动,组件源码拷入用户项目);`update / --all` **未实现**
- ✅ **demo + 用户项目 E2E**:拷贝出的组件在独立用户模块编译/运行/渲染全部正常
- ✅ **demo 布局化 + 布局 API 收口**(2026-10-04~08):改用 `widget.NewLayoutEle` 流式布局;布局项统一到 `xui.Item*`(只声明要改变的那条轴, 哨兵 `xui.ItemKeep=-1`), 容器骨架统一到 `xui.HStack` / `xui.VStackWeight` / `ui.Card.ContentStack`;语义与"纵向溢出换列"坑见 29/41
- ✅ **参考实现补齐**(本地 shadcn 对照页):switch/progress/slider/tabs/tooltip/select/table/dropdown-menu 源码落盘 + `App.tsx` 对照卡(含新第 6 页「图标」);`pnpm build` 通过
  - 2026-10-14 再补:`scroll-area.tsx`(`npx shadcn add scroll-area`, 交互语义权威)+ 基础页"滚动容器"对照卡(与 demo 同构)。
- ✅ **ComboBox 弹层对齐 shadcn**:ring 四边完整(画在 `XE_PAINT_END` 内缩 1px)、item-aligned 定位 + 屏幕边界截断滚动(30)
- ✅ **Dialog**(2026-10-04):路线 = 窗口内覆盖元素(全窗遮罩 + 居中面板 + 页脚 + 幽灵图标关闭钮), ESC / 点遮罩 / 关闭钮关闭 + Tab 焦点陷阱;backdrop-blur-xs(33/34/47)
- ✅ **Menu**(2026-10-05):原生菜单承载行为 + 窗口级 `XWM_MENU_DRAW_*` 自绘皮肤(36)
- ✅ **Table**(2026-10-05):原生 List 载体 + 外观全自绘(表头/行底/边框/滚动条)(40/41)
- ✅ **Tree**(2026-10-06):原生 Tree 载体 + `XE_TREE_DRAWITEM` 自绘;"隐藏容器首显空白"回归已改为组件内自愈(42/51)
- ✅ **hover 过渡动画**(2026-10-06):150ms cubic-bezier(0.4,0,0.2,1), 运行时 `anim.go` 驱动, 组件侧只用 `xui.Hover / HoverItem + Fade`(45)
- ✅ **重绘节流调优**:应用初始化 `a.SetPaintFrequency(10)`, 实测 30.9 → 59.0 fps;`XC_SetTextRenderingHint` 在 D2D 下是空操作(46)
- ✅ **弹层/对话框进出场动画**(2026-10-06):`xui.Presence(...)` + `SetElementsAlpha / FadeWindowAlpha / SlideWindowAlpha`;窗口 alpha 永远不精确 = 0(47)。⚠️ **异型透明窗上不要用整窗 alpha 做动画**(只认第一次调用, 见 57), 淡入淡出改成元素 alpha
- ✅ **backdrop-blur-xs**(2026-10-07):抓窗快照 + Go 盒式模糊 + BkM 图片填充;改物理像素管线后全屏弹窗 160ms → 61ms(47)
- ✅ **Pagination + demo 图标页改造**(2026-10-08):恒定 7 槽位折叠, 搜索 / 翻页 / 悬浮 Tooltip / 点击复制图标名(53)
- ✅ **图标系统**:抗锯齿改**直通 D2D 几何填充**(`d2dicon.go`);**lucide 资源预处理**(2130 个图标, `icons/` 子包)—— 原理 / 坑 / 性能见 43/44,速览见下方「图标系统」
- ✅ **与 shadcn 逐条核对变体/尺寸并补齐**(2026-10-09):核对本地 18 个 tsx 的 cva variants/props 后补齐 —— Button 补 Link 变体 + 8 档尺寸(xs/sm/default/lg + 4 个方形图标档)、Badge 补 ghost/link、Card 补 sm 尺寸与说明/页脚、ComboBox 补 sm 触发区、Slider 补 min/max/step、Separator/Tabs 补纵向取向;像素实测按钮高度 24/28/32/36 与 shadcn h-6/7/8/9 完全吻合。映射与**仍然存在的差异**(aria-invalid / CardAction / Menu inset / chart·sidebar 令牌)记在 DESIGN 第四节末「变体/尺寸与上游的映射、已知差异」;1px 细条的绘制结论见 10
- ✅ **demo 全面布局化 + 分类重规划 + `Card` 改布局容器**(2026-10-09):窗口 800×590 → **1200×860**(最小 960×660), 页面/卡片去掉全部"算出来的绝对坐标", 改为流式布局(窗口最大化即整页自适应, 像素实测: 卡片左边距 36 = 阴影 12 + 页边距 24 ✓, 最大化时 24 ✓, 数据页表格跟随长到 ~1000 逻辑高)。页签重规划为 **基础 / 表单 / 导航 / 反馈 / 数据 / 图标**(原"控件/结构/新增变体"散布的内容各归其位, 不再挤在一页)。`Card` 底层换 `LayoutEle`、内容区随卡片尺寸联动(见 DESIGN 第八节 56);布局尺寸语义(尤其"`Fill` ≠ 剩余空间"、"`Weight` 需父级确定尺寸")见第八节 55
- ✅ **页面宿主的"叠放换页"不需要绝对定位**(2026-10-09):布局会跳过 `Show(false)` 的子项(排成 0x0), 所以 6 个页是宿主里的 6 个 Fill/Weight 兄弟, 切页只 `Show` + `AdjustLayout`(实测, 见 55)
- ✅ **Tooltip 气泡"显示着却看不见"根因修复**(2026-10-09):症状是悬停后提示不出现(用户侧描述"点几次切换主题后 ToolTip 就不显示了"), 实际窗口**被显示出来了**(可见、矩形正确、位置还随动画滑动), 但 layered alpha 被永久钉在 3/255 —— 异型透明窗的整窗透明度**只认第一次调用**(探针复现, 见 57)。修法: 进出场改淡**元素** alpha, 整窗 alpha 只在显示后设一次 1; 验收=亮/暗双主题各 3 轮、连续 6 次切主题后每轮气泡像素增量完全一致
- ✅ **Tooltip 跟随主题切换**(2026-10-09):气泡挂在屏幕上时点"切换主题", 它过去会保持旧配色 —— 自建的独立置顶小窗不在 `RegisterWindow` 重绘清单里, 得自己 `OnThemeChangeFor` 订阅并重算+重绘(见 58); 顺带把窗口扩展样式改成 `window.NewEx` 一次传入。验收=光标停在主题按钮上直接点它, 气泡当场从深色块变浅色块

- ✅ **表格"首屏自适应列宽塌成最小值"修复**(2026-10-10):数据页刚进来时最后一列只有 20px(表头裁成"日"、单元格全显示 `...`), 拉一下窗口才正常 —— 根因是"列宽按元素宽度算, 而 `Show()` 之前元素宽度是 0"; 现在组件自己订阅 `List.AddEvent_Size` 自愈(`lastW` 防震荡), 调用方不必再记得调 `FitColumns`(见 60)
- ✅ **Menu 弹层阴影 + p-1 边距 + 进场淡入**(2026-10-10):弹窗转 `Window_Transparent_Shadow`(官方 DrawMenu 同款, 圆角+投影+整窗 alpha 一次解决); `SetBorderSize(4)` 落实 shadcn p-1(高亮距弹层左右边缘 4 逻辑像素对称, 修掉"与参考差 1px"); 分隔栏改全宽(-mx-1); 进场 fade-in 100ms(Presence 挂宿主窗驱动)。三处深坑: 过渡闭包捕获 `Menu_PopupWnd_` **指针**会读到 0/垃圾(0=淡入冻结, 垃圾=崩进程), 必须拷贝句柄值 + `XC_IsHXCGUI` 守卫; 对菜单弹窗 `SetWindowPos` 位移直接崩进程 → slide-in 不可实现, 只做淡入; `XMenu_SetItemWidth` 实测有效(早期"无效"结论已修正, 见 64)
- ✅ **暗色禁用/未选态配色按 shadcn 逐像素对齐**(2026-10-10):复选框未选态(暗色 `bg-input/30`)、复选框/单选框/开关的禁用态、禁用输入框过去都错用 `WithAlpha` 覆盖半透明令牌(暗色下画出发亮灰块 / 亮白框线 / 全亮白滑块)。现在统一为"**先铺宿主底色(Card) + 每个图层各乘 50%**":形状全部跟随, 原生文字保持原色, 输入框禁用文字预混变淡(见 61); 顺带修掉透明元素重绘**不擦底**留下的旧像素(焦点环 / 上一个状态)。验收=让"切换后的渲染"与"直接建成目标状态"逐像素比, 亮/暗各一轮都是 **0 像素差**(色值与 shadcn 的整组 50% 合成理论值逐条吻合)
- ✅ **本地 shadcn 参照页(界面展示)与新 demo 同构**(2026-10-10):`shadcn/src/App.tsx` 重整为与 demo 一致的六页(**基础 / 表单 / 导航 / 反馈 / 数据 / 图标**), 内容逐条对应(6 变体 × 4 档尺寸 + 4 方形图标档、徽章 6 变体、CardFooter、多行与禁用输入、滑动条两档、开关 default/sm、Tabs default/line/纵向、恒 7 槽分页、表格 12 行 + 树); 新增 `?tab=form` 这类初始页签参数(对应 demo 的 `XUI_HOME_PAGE`), 便于分页截图对照; `pnpm build` 通过

- ✅ **SVG 解析层上移到上游 `xcgui/svgp`**(2026-10-11): path 语法解析、SVG 文档遍历(`<g>` 继承/viewBox)、
  子路径分组与绕向统一、d2d 几何组装全在那边(纯 Go, 14 个单测, 不经过 DLL —— 而 xcgui 自带的 svg 是
  DLL 内 C++ 且不支持svg的一些特性, 纯 Go 是唯一可继续改进的路径); xui 只留图标语义(24 网格归一化、
  `!` 填充前缀、SVG 原文缓存、GDI 折线回落), 删掉本地解析器约 500 行。验收: 改动前后各编一份 demo,
  图标页 light 148/2322000(0.0064%) / dark 185 / 基础页 **4** 像素, 全是 `Z` 闭合语义的 AA 级微差;
  墨迹量 143083 → 143082。

- ✅ **图标支持整段 SVG 字符串**(2026-10-11): 与 path 数据**共用 `Painter.Icon` 入口**(自动识别: 首字符
  是不是 `<`), 另有 `ParseIconSVG(svg)` 供"先校验/自己缓存" ——
  提取 `<path>`(多个, 支持 `<g>` 继承 fill/stroke/stroke-width)、按 viewBox 归一到 24 网格(任意网格)、
  按 fill/stroke 判定填充子路径、建议线宽取 stroke-width; 解析结果按 SVG 原文缓存, 几何照旧走图标几何缓存。
  于是 heroicons / feather / iconfont 的 SVG 可以直接贴, "只吃 24 网格 path"这条限制解除。
  验收见 DESIGN 第八节 44 末尾; 顺带修了 `parseIconPathEx` 对尾部空白的误判。

- ✅ **CLI 远程 registry(v2)+ `update` / `add --all`**(2026-10-11):数据源四态 —— 默认源
  `raw.githubusercontent.com/twgh/xui/main`(**自动选路:直连 → Gitee 镜像 `gitee.com/twgh/xui/raw/main`
  → GitCode 镜像 `api.gitcode.com/api/v5/repos/twgh/xui/contents`(v5 API, JSON+base64;
  raw 直链对 `.go` 匿名访问只回 403; 匿名 **50 次/分钟**, 实测 0.4~0.8s/请求) → 加速代理 ghfast / gh-proxy**,
  探测 = 拉一次 registry.json, 6s 超时, 第一个通的跑完本次命令, 选路
  过程逐行打 stderr; **拷到一半单文件失败(限速/挂起)按候选顺序换下一个源并把新源粘住**
  (`source.rest`, 免得留下半套 `ui/`); 组件跟 main 永远最新)、`XUI_REGISTRY` 指向本地目录或任意 `http(s)://` 基址
  (离线 / 锁组件版本; **写什么用什么, 不套镜像/代理**)、全不通时降级 `moduleRoot()` 内嵌源码树
  (显式设了 `XUI_REGISTRY` 则报错不降级);`update` 无 manifest
  语义 = 缺则建 / 一致跳过 / 用户改过警告跳过(`-o/--overwrite` 覆盖),省略组件名 = 更新 `ui/` 里已装的全部;
  新增 `-o/--overwrite` / `--all` 两个旗标。方案与验收清单见 [REGISTRY_V2.md](REGISTRY_V2.md);离线验收全绿
  (本地路径 / 本地 HTTP / 404 / 不可达 / 降级 / update 四态 / 逐组件单独安装后编译 / init 后编译),
  行为已固化为 `cmd/xui/main_test.go`(候选顺序 / 数据源解析 / httptest 端到端 / 显式源失败退出 / GitCode API 解码 / 单文件换源)。
  **顺带修掉 registry.json 的依赖缺陷**:`dialog` 漏 `backdrop.go`、`combobox`/`menu` 漏 `tooltip` 依赖
  —— 过去 `xui add dialog` 拷出来的组件根本编译不过。

- ✅ **页面滚动容器 `ui.ScrollArea`**(2026-10-14):载体是原生 **`LayoutFrame`(布局框架 = 布局盒子 + 滚动视图)**, 内容总大小**按子项自动撑开**、溢出自动出纵向滚动条(不需要手动 `SetTotalSize`)。探针实测定死一批语义(子项 `Fill` = 视口高且内层溢出**不算**滚动范围、`Weight` 恒为 0、`EnableOverflowHide` 是"布局时把子项高度裁掉"**不是**裁剪显示、首帧内容宽少一个滚动条宽要补排一次、`ScrollPosYV` 才是像素)与滚动条皮肤(轨道透明 + `bg-Border` 圆头滑块 + `OnThemeChangeFor` 换肤跟随)—— 全部记进 **DESIGN 第八节 66** + API 清单。demo 六个页全部换成滚动容器(不再被 `EnableOverflowHide` 裁掉): **自撑型**(基础/表单/导航/反馈/图标, 内容区 Auto 高, 窗口小于内容时滚)与**铺满型**(数据页 `SetContentFill`, 表格/树仍跟随窗口长高、整页不滚);demo 的 `row` 骨架同步改成行高 Auto(`rowStretch` 给数据页)。验收: 980×700 下表单页出滚动条 + 真实滚轮滚动(先点一下页内空白让窗口拿到焦点)+ 滚动条滑块随主题亮(#e5e5e5)→暗(#232323)现场切换;大窗(1500×880)无滚动条且卡片右缘 = 视口右缘 − 24 页边距(无"12px 空白"回归);六页 + 对话框浮层在两种尺寸/两种主题下逐页抓图核对;基础页加了一张"滚动容器"卡片(组件形态, 嵌套滚动)。⚠️ 滚动条**滑块拖动**在合成输入下未复现(裸 LayoutFrame 同样拖不动、表格 List 的能拖 —— 非组件引入的差异, 见 DESIGN 66 末尾)。

- ✅ **D2D COM 直通底座上游化到 xcgui**(2026-10-11):xcgui "交给 Go 侧裸 `ID2D1Factory` /
  `ID2D1RenderTarget` 指针却不给封装"留下的空白, 现在收进 xcgui 新子包 **`github.com/twgh/xcgui/wapi/d2d`**,
  COM 基础件(`ComProc`/`IUnknown`/`IUnknownVtbl`/`InterfacePtr`)抽到 **`wapi/com`** 子包共用, d2d 以
  类型别名引用(edge 自带一份同源实现, 迁移面大暂未动)。该包还提供 **amd64 的汇编转发器**(`CallWithFloat32`/
  `CallWithFloat64`/`CallWithFloat32RetFloat`)—— 让"形参含浮点"的 COM 方法也能调(SyscallN 设不了 XMM);
  三个实测坑(必须经 SyscallN 借 cgocall 切系统栈、调用点 16 字节对齐 + 32 字节 shadow、浮点槽号要按接口
  实测)与 386/amd64 的参数规则都写在包的注释里, 见 DESIGN 第八节 65。封装形式照 **`edge` 包(WebView2)**那套:每个 COM 接口一个文件、**按 SDK 声明顺序完整列出**的虚表结构体
  (索引由字段顺序决定, 不写魔数) + `ComProc.Call`(带 `//go:uintptrescapes`) + 中文注释与 MSDN 链接;
  接口结构体按 COM 继承链**值嵌入**(`IUnknown` → `ID2D1Resource` → …), 三件套与 `GetFactory` 各只实现
  一次, 自己的方法用 `vtbl()` 取 `i.Vtbl` 重解释虚表。含 ABI 三条硬规则、几何/画刷的创建释放、几何自检
  (`Usable`)、`NewGeometry`/`FillGeometryColor`。8 个单测:颜色换算 / 点打包 / **虚表槽位**
  (`TestVtblIndexLayout`, `unsafe.Offsetof` 钉住 FillGeometry=23、IsSupported=50 等) / **虚表解析**
  (`TestVtblResolution` —— 曾把"对象地址"当虚表, demo 启动即 panic PC=0x989680) / 结构体布局 / 继承复用 /
  nil 安全 / 空句柄。
  顺手把 `XDraw_GetD2dRenderTarget` / `XDraw_ConvRect` / `XC_GetD2dFactory` 的注释补齐了"直通调用拿的是
  父级坐标系, 必须先 `ConvRect`"这条坑。本仓库 `d2dicon.go` 只剩图标语义(描边展开 + 盒计算 + 缓存 +
  元素 alpha), 少约 200 行重复底座。**验收**:图标页/基础页 × 亮/暗三组 PrintWindow 抓图逐像素 **0 差**;
  图标页鼠标扫动 CPU 交替多轮无可辨差异(±10% 噪声); 细节见 DESIGN 第八节 43。**联调方式**: 本仓库用
  **`go.work`** 把本地 xcgui 工作副本接进来(该文件已在 `.gitignore`), `go.mod` 保持干净的 `require v1.4.2`。
  ⚠️ 在 xcgui 工作副本里**只跑 `go test ./d2d/ ./wapi/com/`**, 别跑 `go test ./...` —— 那是 GUI 库,
  全仓测试会不停弹真窗口(实测弹到没法干活); `go build ./...` / `go vet ./d2d/ ./wapi/com/` 是安全的。
  ⚠️ **本仓库 `go.mod` 的 go 指令是 1.18(用户明确不升级)**: 即使工具链是 Go 1.22+, `for range` 的循环
  变量也按**旧语义共享** —— 在循环里注册回调/起 goroutine 必须 `v := v` 拷一份(探针踩过: 三个元素全画成
  最后一个)。xui 自身的循环注册点已扫过(vet + 人工, 只有两处"for opts"是误报), 新代码保持这个习惯。
  后来又补了 `SetTransform` / `GetTransform`(矩阵按**指针**传, 不受"按值浮点走 XMM"的限制)——
  矩阵往返一致(单位矩阵 → 平移 12,34 → 设回单位矩阵)。
  借它把图标**位置从几何缓存里外置**了: 几何只按 (path, size, strokeW) 缓存、绘制时用平移矩阵移过去 ——
  窗口 resize 不再重建几何(实测尺寸来回切 10 次的 CPU 78/93ms → 62/62ms), 像素验收两组 0 差、图标页
  6/2322000 像素(AA 边缘); 前提是**成对恢复变换**(实测 XDraw 用的是绘制当时的 RT 变换, 见 DESIGN 43/65)。
  **再进一步(2026-10-11): 描边也交给 D2D 了** —— `wapi/com` 新增 `CallWithFloat32Stack`(最多 4 个栈参数)
  后, `ID2D1RenderTarget::DrawGeometry`(线宽在 XMM3、style 压栈)可用, 图标折线不再在 CPU 侧展开:
  `d2dicon.go` 删掉 `strokeOutline/addArc/miterPoint` 约 300 行, 缓存 key 精简成 (path, size), 角/端点由
  D2D 按 SVG 的 round 语义画(实测"角变圆"、墨迹量 +0.09%)。**性能: 微观 4 倍(2 万次 20ms vs 80ms)、
  真实场景在噪声内** —— 收益是质量与代码量, 不是帧率(详见 DESIGN 43 末尾)。

- ✅ **滚动条统一: shadcn 几何 + 两套配色 + 悬停显隐**(2026-10-15, 两轮):五处原生滚动条载体(ScrollArea
  页面与卡片内 / Table / Tree / ComboBox 弹层 / **多行 Input, 原先是系统滚动条**)的几何、配色、显隐
  全部收敛到 `components/scrollbar.go` 的 `styleScrollBar` + `showScrollBarOnHover`。
  **几何**: 轨道 10(`w-2.5`)、滑块内缩 1(`p-px`) → 8 逻辑像素宽、`rounded-full`(圆角 4);
  **配色两套**(用户对照浏览器原版提出): 容器型 = `bg-border` 悬停不换色(radix 语义), 嵌入式
  (Input/Table/Tree/弹层, 原版里是浏览器原生滚动条) = 前景色淡化 + 悬停加深(对齐 Win11 Fluent,
  亮 #8b8b8b→#636363 的观感); 颜色一律 `flatOn` **展平成不透明**(暗色 Border 是半透明白, 不展平
  会"同一滑块在页面 #232323、卡片 #2f2f2f"地不一致 —— 用户报的 bug);
  **显隐**: 初始 `SetSubtreeAlpha(0)`, 窗口 MouseMove + `XWnd_GetStayEle` 判定(宿主子树 → `Presence`
  淡入/淡出 150ms; 滚动条子树 → 滑块换悬停色), 不用 Show(false)(会改视口宽、内容跳变)。
  三个踩坑(详见 DESIGN 第八节 21): ① 滑块三态的背景对象必须**各用各的 id** —— 共用一个 id 时只有
  第一个生效, hover 会回退默认绘制(表现为"悬停时变宽且圆角消失"); ② 元素级 MouseStay/MouseLeave
  不可靠(多行 Edit 收不到 leave、原生 List 不把悬停路由给滚动条) → 一律走窗口级 GetStayEle;
  ③ 淡出到位那帧 anim 不重绘, 要 done 回调补 `XEle_Redraw`, 否则留淡残影。Table 的自适应列让宽
  12→14; registry 的 5 个组件已登记 `scrollbar.go`(逐组件安装+编译复验过), demo 多行输入框加了
  初值。**验收**: 五处载体 × 亮/暗, 隐藏/显示/悬停加深三态全部像素扫描通过(滑块物理 12px @150%;
  亮色静止 #919191 / 悬停 #6a6a6a, 暗色两容器统一 #2f2f2f)。

- ✅ **combobox 弹层进场动画(P3 收尾, 2026-10-17)**:下拉弹层是原生自建普通窗口, `SetTransparentAlpha` 对它无效;
  在 `XE_COMBOBOX_POPUP_LIST`(此时窗口**还没显示**)里转 `Window_Transparent_Shaped` 后整窗 alpha 即可播放
  100ms 淡入, 且**连续改 alpha 每帧生效**(与 57 的 tooltip 结论不同, 三档抓图
  #40A0FF/#A0D0FF/#FFFFFF), 区域裁剪的圆角与 ring/条目绘制全部照常。过渡挂宿主窗驱动 + `XC_IsHXCGUI` 守卫 +
  每次弹出用 `xui.NextPresenceKey()` 取键(各组件自起计数器会撞车, 见下条修复); 真实组件逐帧抓图看到
  5 帧淡入曲线。原理与坑见 DESIGN 第八节 30/69/70。

- ✅ **与 shadcn 的边角补齐(2026-10-17)**:
  - **aria-invalid**(Button / Input / ComboBox): `WithButtonInvalid()` / `WithInputInvalid()` / `WithComboInvalid()`
    + 运行时 `SetInvalid(bool)`; 红边框 + ring-3 destructive/20(暗色 border /50、ring /40), 常驻不要求焦点,
    与 focus-visible 同时成立时红色优先。demo 基础页加了三个按钮、表单页加了输入框, 亮/暗各验一轮。
  - **CardHeader / CardAction**: `ui.NewCardHeader(parent, space)` + `TitleStack()` / `ActionStack()`, 左列标题+说明
    占 `1fr`、右列动作 `auto` 且跨两行顶端对齐; demo 的「卡片」示例改成头 + 省略号动作钮。
  - **`MenuInset()`**: 对应 shadcn dropdown-menu 的 `inset` prop(`data-inset:pl-7`), 条目文字左缘 = 28,
    普通条目借此与带勾选/图标的条目对齐;demo 菜单加了一项。
  - **`--chart-1..5` / `--sidebar-*` 令牌进 `xui.Theme`**(Chart1..5 / Sidebar*): 组件库不用, 供用户应用取用;
    `manager_test.go` 有守卫防漏填。DESIGN 第七节已无菜单宽度差异(原"`XMenu_SetItemWidth` 无效"结论有误, 已修正, 见 DESIGN 64)。

- ✅ **修复: 弹层键撞车导致"点击没反应"(2026-10-17, 用户报障)**: 选过 sm 下拉框后第一次点"打开菜单"无反应(第二次才开),
  反过来先开菜单再点下拉框也无反应。事件日志证明 BN_CLICK 已触发 —— 菜单确实弹了, 但 alpha 停在 `AlphaFloor`(全透明)且处于模态,
  把下一次点击吃掉。根因是 `xui.Presence` 的幂等语义 + menu/combobox **各自的计数器都从 1 开始**(键撞上已完成的过渡 →
  动画不跑、apply 不调)。修法: 新增 `xui.NextPresenceKey()` 全局分配键(menu/combobox/dialog 全改用), `Presence` 立即
  apply 一次兜底。详见 DESIGN 第八节 70。

- ✅ **逐组件单独安装 + 编译回归(2026-10-17)**:20 个组件各建临时项目 `xui init` + `xui add <组件>` + `go build ./ui`
  (本地 `XUI_REGISTRY` + 模块 replace), 另加 `add --all` 一项, 21/21 通过。

- ✅ **Calendar / DatePicker / TimePicker 三个新组件(2026-10-18, 参考 HUMUI HouTai017)**:
  - **`ui.Calendar` 月历卡片**(`components/calendar.go`): 全自绘(不用原生 MonthCal —— 其格子/比例不可控), 248×292、
    6×7 固定网格(高度不跳), 格 32 圆角 6; 选中实心/今天浅底/悬停过渡/非本月弱化; 周一开头可配(`WithCalendarWeekStart`);
    可独立摆放, 弹层内 `WithCalendarFlat` 免双层边框。
  - **`ui.DatePicker` 日期框**(`components/datepicker.go` + 弹层骨架 `popup.go`): 触发区 = outline 按钮(calendar 图标 +
    "2006年1月2日"/占位符), 弹层 = 自建阴影窗(卡片由 body 画, 月历 flat) + item 左下对齐(sideOffset=4)、
    下方放不下翻上方、夹屏幕内; 淡入 100ms(Presence 挂宿主窗 + `NextPresenceKey`); 选中即回写+关闭+回调;
    点外部(KillFocus)/ESC 关闭; **点触发区关闭后 250ms 内不重开**(KillFocus 先于 LButtonUp, 否则"点两次才关掉");
    Open/Close/IsOpen 程序化控制, 销毁时弹窗跟着销毁, `OnThemeChangeFor` 跟主题重绘。
  - **`ui.TimePicker` 时间框**(`components/timepicker.go`): 触发区同款(clock 图标 + "09:30"); 弹层两列原生 ListBox
    (时/分, 64 宽、8 项可见), 条目背景/文字自绘(选中 bg-primary 反色、悬停 bg-accent 过渡), 滚动条统一皮肤;
    选择即生效、弹层保持打开; 打开时选中项滚到中间(行步进 = 项高+1 + 首项固有内缩, 见 DESIGN 72)。
    ⚠️ ListBox 手动创建的一批语义(适配器先行/两套索引/模板文字静态色)记 DESIGN 第八节 72。
  - **弹层骨架 `popup.go`**(阴影窗 + 自绘卡片 body + KillFocus/ESC 关闭 + 定位翻转 + 淡入): datepicker/timepicker 的
    files 里各带一份, registry 已登记。
  - **验收**: 亮/暗、打开/选中回写/外部关闭/重开/翻月逐项通过、demo 表单页两张新卡片(深色滚动截图);
    逐组件安装编译 calendar/datepicker/timepicker + `add --all` 全部通过; registry.json 已登记 3 个组件。
  - 新坑记 DESIGN 第八节 **71**(阴影窗元素坐标原点 = 窗口矩形原点, 卡片要放 (band,band))、**72**(手动 ListBox 自绘列表语义)。

- ✅ **修复: 日期框/时间框弹层"闪烁/抖动"(2026-10-18, 用户报障 + 60fps 录屏, 四个根因全修)**:
  - **根因 A(白闪)**: 弹窗不设背景时 xcgui 画**白色默认底**, 淡入期间白底从半透明卡片下透出(帧级抓图: 第二次
    显示 mean=228 白帧)。修法: 弹窗创建时 `XBkM_Clear + XBkM_AddFill(alpha=0)` 垫透明填充(同第八节 30)。
  - **根因 B(淡入路线)**: 整窗 alpha 对透明窗只认第一次调用(57), 复用弹窗整段失效(两次显示都瞬间全亮)。
    修法: 淡入改**元素 alpha**(SetSubtreeAlpha(body) + 每帧重绘, tooltip 同款), 显示前压 0。
  - **根因 C**: `hoverTick` 对已到位的旧 presence 条目仍每帧调 apply, 二次弹出时互相打架(实测 12↔85 交替)。
    修法(anim.go): settled 条目不再调 apply; menu/combobox/dialog 同表受益。
  - **根因 D(用户 60fps 录屏"抖动", 快速点击更明显)**: 过渡挂**宿主主窗口** → 淡入 100ms 里每帧全量重绘
    整个表单页, 快速点击时重绘排队、UI 线程卡到出系统忙碌光标(用户录屏可见蓝圈)。修法: 整窗 alpha 时代挂宿主窗
    是被迫的(DWM 合成要宿主带动), 改元素 alpha 后**过渡挂弹窗自己**(400×500 物理像素, 每 tick 重绘无压力)。
  - **根因 E(用户逐帧"直接显示 → 消失 → 再淡入")**: 复用窗口再次 ShowWindow 首帧合成的是**表面残留的上一次
    内容**(关闭前全亮卡片), 元素 alpha=0 要等首次重绘才落地。修法: **屏幕外预显示** —— SetWindowPos(-30000) +
    Redraw 画好透明首帧再挪到目标位置。
  - **根因 F(时间框点击闪退 + "SetAlpha 句柄无效"弹窗)**: `GetChildByIndex` 对原生 ListBox 返回**非 0 但无效**
    的子句柄(type=62 延迟创建对象, 显示后依然存在, 每列约 8-16 个) → `XEle_SetAlpha(无效句柄)` 触发 DLL 错误
    弹窗 —— DLL 错误不产生 Go panic, 被吞掉后可能 access violation 闪退。修法: `fadeSubtreeAlpha` 与
    `setAlphaTree` 对每个句柄加 `XC_IsHXCGUI(h, XC_ELE)` 守卫 + fadeSubtreeAlpha 跳过滚动条子树(滚动条显隐自理)。
  - **根因 G(概率性半透明透字 + 滚动条不隐藏)**: 淡入中途被 hide(快速点击/选日期/点外部)时 alpha 停在中间值、
    条目未 settled —— ①300ms 内 fast 重开只设属性**没有重绘**, 屏幕上还是中间值的残留帧(弹层半透明透出页面文字,
    且滚动条随子树一起可见); ②未 settled 的旧条目每帧 apply(v=1.0) 与新状态互相覆盖。修法: Hide 时
    `xui.DropTransitions(弹窗)`(新 API, 清条目+关残留定时器) + `SetSubtreeAlpha(1)` 归位终态 + OnFaded 恢复
    滚动条隐藏; fast 路径补 `Redraw(true)` 让属性落地。验证: 60fps 录屏半透明卡住段 0, 快速交互进程存活。
  - **交互补充**: 关闭后 300ms 内重开(`popoverReopenFadeSkip`)**跳过淡入直接完成态** —— 快速连点时弹层反复从
    透明渐显会"眨眼", 对齐 CSS 过渡的中断语义(不重播动画)。
  - **滚动对齐(用户截图"两列选中项不在同一高度")**: 两个叠加原因 —— ①`SetSelectItem` 触发 xcgui 内建"选中项
    滚入视口"的异步滚动, 布局完成后执行并**覆盖**紧跟的 ScrollPosYV; ②`ScrollPosYV` 的落点被吸附到**滚动单位
    网格(默认 20px)**, 实测 170→160、779→760(都是 floor(p/20)×20), 两列请求值除 20 的余量不同 → 吸附后错位。
    修法: `syncLists` 不再 SetSelectItem(选中态由 DrawItem 按组件值绘制, 不依赖原生 Select) + 滚动定位挪到
    `OnFaded`(首帧绘制前) + **`SetLineSize(20, 行步进)`** 把滚动单位设为行步进(29), 两列 pos 相差整行数,
    吸附后仍差整行 → 选中项必然同高。PrintWindow 抓弹窗验证: 两列选中块同一水平线、滚动条隐藏。
  - 详见 DESIGN 第八节 73(含排查方法论: 帧级抓图 > 亮度采样; 弹窗进程内自抓 > 外部轮询; 录屏坐标要按实际
    内容校准; DLL 错误弹窗 ≠ Go panic)。验证: 60fps 录屏逐帧分析 —— 白帧 0, 弹层出现为纯渐显(19→31 无全亮相),
    快速连点段无爆闪、无忙碌圈; 时间框连点两次进程存活。

- ✅ **Calendar 三块新能力(2026-10-19, 参考用户截图)**:
  - **年月可选**: 标题行原来是一整块"2026年10月", 现在拆成「2026年 ⌄」「10月 ⌄」两个热区, 点开是贴着标题的自绘下拉
    (bg-popover + ring, 行高按 12 行塞满可用高度自适应, 当前值右侧 check, 超出高度滚轮滚动 + 2px 细滚动条); 面板内点击=取值、
    面板外=关闭、点另一个热区=直接切换; 只改显示月份不动选中日期。年份范围默认"视图年 ±50", `WithCalendarYearRange` 可配。
  - **禁用日期**: `WithCalendarDisabled(fn)` + 运行时 `SetDisabled`, 内置 `CalendarDisableBefore/Weekend/Dates/Any` 四个 helper;
    禁用格 text-muted-foreground opacity-50, **不可点、无悬停、不触发 OnSelect**(探针点周六验证无选中变化)。
  - **页脚**: `WithCalendarFooter(items...)` / `WithCalendarFooterToday()` / 构造后 `AddFooterItem/Button/Text` / `ClearFooter`;
    页面全宽 1px 分隔线 + 居中按钮行(ghost 化: 悬停 bg-accent), 元素未指定高度时自动长高 `CalFooterHeight()`(37);
    要放原生控件时用 `FooterRect()` 取内容区矩形自行摆放(子元素盖在自绘内容之上, 点击不穿透)。
  - **验收**: 探针(亮色, 两张卡片: 周末禁用 / 页脚)逐项截图 —— 年列表开合与滚轮滚动、选年切月、月份列表、页脚「今天」回本月并选中、
    点禁用日期无响应; demo 表单页「月历卡片」改为 周末禁用 + 页脚(今天/清除)。`go test ./...`、`go vet` 通过。
  - **注意**: 下拉是**元素内自绘**(不是独立弹层窗口), 因此受卡片尺寸限制 —— 行高会自动压缩以塞下 12 行; 鼠标离开卡片即收起
    (自绘下拉没有原生弹层的 KillFocus 失焦机制); 滚轮事件只在打开时 `EnableEvent_XE_MOUSEWHEEL(true)`, 关闭即还回外层 ScrollArea。

- ✅ **月历等比缩放 + 弹层不压任务栏(2026-10-19 用户报障)**:
  - **月历内部随尺寸缩放**: 原来几何是写死的 32px 格子/12px 内边距, demo 把卡片放大 1.12 倍只有空白变大。现在
    `Calendar.geom()` 按**元素宽度**算一整套几何(内边距/格子/标题行/星期行/页脚/字号全部等比), 高度只决定 6 行之间的
    行距(多余空间均摊到行间), 绘制与命中判定共用同一套值; 下拉面板的行高/字号/内边距同样跟着缩放。
  - **默认尺寸改为 1.12 倍**: `CalWidth/CalHeight/CalFooterHeight` = 278×327 / 41(设计基准 248×292 的比例), 即用户
    截图里"乘 1.12"的效果; demo 去掉手写的 ×1.12, 直接用 `NewCalendar(0,0,0,0,...)`。
  - **新增 `CalHeightFor(width)` / `CalFooterHeightFor(width)`**: 只想改宽度时 `cy` 传 0(NewCalendar 自动配平),
    或显式 `CalHeightFor(w)+CalFooterHeightFor(w)` —— 手写 `CalHeight()*1.5+CalFooterHeight()` 会少算页脚的缩放, 底行被页脚压住。
  - **弹层夹边界改用显示器工作区**(`MonitorFromWindow`+`GetMonitorInfo` 的 `rcWork`, 见 DESIGN 第八节 74): 最大化窗口 +
    页面滚到末尾时, 贴着任务栏的日期框向下弹会被任务栏盖住; 现在下方放不下先翻上方, 上方也放不下就贴工作区底边。
    ⚠️ **必须传 Win32 HWND**(`xc.XWnd_GetHWND`), 传 xcgui 窗口句柄会让 `MonitorFromWindow` 返回 0 并静默退回整屏 —— 这是
    第一次修完仍然"被任务栏遮挡"的根因(用户第二次报障); 现在取不到显示器时还会退回 `GetDesktopWindow()` 再查一次。
  - **验收**: 探针(默认 + 1.5 倍宽两张卡片, 高度传 0 自动)截图确认内部等比、6 行不裁切; **真实复现路径**——demo 表单页
    最大化 + 滚到最底部点日期框, 枚举弹层矩形 (48,531)-(489,1046), 底边 1046 < 任务栏 1520, 弹层翻到触发区上方完整可见;
    demo 表单页月历卡片改用默认尺寸。
  - **注意**: 探针里别在 goroutine 里 `dp.Open()` —— xcgui 创建窗口/元素不是线程安全的, 会出现"弹层白板不画月历"的假象(实测)。
  - **上游化(同一日)**: xui 里散落的 Win32 声明全部收进 xcgui 的 wapi —— 新增 `GetWindowDC / ReleaseDC /
    PrintWindow / MonitorFromWindow / GetMonitorInfo` + `PW_` / `MONITOR_` 常量 + `wapi.MONITORINFO`(types 包),
    gdi 侧复用已有的 `gdi.CreateCompatibleDC / CreateDIBSection / SelectObject / DeleteObject / DeleteDC`;
    `components/backdrop.go` 的 `modUser32/modGdi32` 与 `components/popup.go` 的 MONITORINFO 声明都已删除,
    xui 中**不再有裸 `syscall.NewLazyDLL`**。上游补了 `TestGetMonitorInfo / TestGetWindowDC`(实测工作区
    1707×1019 vs 整屏 1707×1067, 任务栏 48px 被扣掉)。

- ✅ **Toast 通知(2026-10-19, `components/toast.go`)**:
  - **载体**: 窗口内覆盖元素(每条通知一个自绘 Element, 无子元素), 右下角堆叠、跟随窗口移动/缩放/最小化;
    不占用原生控件, 关闭按钮/动作按钮的命中在卡片局部坐标里算。
  - **API**: `ui.NewToaster(w.Handle)` + `Show(ToastOptions{...})` / `Success / Error / Warning / Info`;
    `Duration` 传负数 = 常驻, `Action` + `OnAction` 动作按钮, `CloseButton` 显示关闭钮;
    选项 `WithToastPosition/Width/Offset/Gap/Duration/MaxVisible`, 实例 `Update/Close/Handle`。
  - **交互(对齐 sonner 默认值)**: 自动关闭 4s; **悬停任意一条暂停全部倒计时**(窗口级 MouseMove +
    `GetStayEle`, 元素级 Stay/Leave 不可靠, 见 DESIGN 21/75); 超出 3 条移除最旧;
    进场淡入 + 滑入 12px(200ms), 退场淡出 + 侧滑 24px(180ms); 主题切换重新量高重排。
  - **验收**: demo 反馈页「通知」卡片(成功/错误/警告/信息/带动作/长文本), 亮/暗两主题抓图:
    单条与三条堆叠、图标语义色、长描述折行、动作按钮与关闭按钮点击、悬停 6s 不消失、移开后自动关闭;
    `go build ./...` / `go vet ./...` / `go test ./...` 通过。
  - **内边距对称性(像素实测)**: 卡片排版坐标必须用"卡片矩形"(元素比卡片四周大 6px 画投影),
    否则卡片可见高度少 12、下内边距被吃掉 6(用户报障"底部空白比顶部小")。修后两行卡片
    topGap=botGap=28 物理像素(@150% DPI)、单行卡片 26/27; `GetTextShowRect` 的 CY 已含行距
    (13px: 单行 18 / 两行 36), 别再补偿(见 DESIGN 75)。
  - 原理与坑见 DESIGN 第八节 **75**(单元素自绘 / 窗口级悬停 / 倒计时用元素 XCTimer / 位置动画挂窗口整窗重绘)。

- ✅ **DESIGN.md 改造为"随源码分发的自定义指南"**(2026-10-19):
  - `xui init` 现在把 `docs/DESIGN.md` 拷到用户 `ui/DESIGN.md`(已存在则跳过, 读不到不阻断) —— 组件注释里
    "见 DESIGN 第 N 节"在用户项目里可达。
  - 内容改写: 去掉 73 处日期、10 处"用户报障"、43 处 demo 页签引用;
    `components/` 路径改 `ui/`; 第五~七节从"CLI 实现现状 / 仓库结构 / 路线图"改为
    "安装与更新 / ui 目录与组件依赖 / 组件覆盖与已知限制"。**章节与第八节条目编号未动**(源码注释靠编号引用)。
  - 开头加"按需查"导航 + 两条说明(示例程序 = 仓库 `demo/`;探针 = 临时验证小程序)。
  - AGENTS/HANDOFF 补规则: DESIGN 会拷给用户, 不写日期/报障时间线/待办/发布流程。
  - 验收: 临时项目 `xui init` → `ui/DESIGN.md` 存在且与源文件 MD5 一致; 重复 init 跳过;
    `go build ./...` / `go vet ./...` / `go test ./...` 通过。

- ✅ **AppWindow 主窗口封装**(2026-10-19, `components/window.go`):
  - `ui.NewAppWindow("标题", w, h)` = 圆角阴影 + 无系统标题栏 + 自绘三键(最小化/最大化/关闭) + 主题化底色;
    内容放进 `w.Content()`(纵向流式 Weight 1), `AddCaptionButton` 在标题栏加自定义按钮(排在最小化左边)。
  - 选项: `WithAppWindowMinSize/CaptionHeight/NoMinMax/Icon/ContentPadding`。
  - 封装期踩到两个坑(细节见 DESIGN 76): ① 窗口 `EnableLayout` 漏调 → 内容区高 0, 界面只剩标题栏;
    ② 双击判定用 `GetStayEle != 0` → 因标题/图标 MouseThrough 而整段失效, 改成只拦 `XC_BUTTON`。
  - 验收: 亮/暗两主题、最大化(图标自动切"还原")、双击最大化/还原、拖动、关闭;
    `ui/window.go` 单组件安装编译通过(依赖 button)。
  - **demo 已整体迁移**(标题栏的 logo / 布局边界 / 主题按钮走 `WithAppWindowIcon` + `AddCaptionButton`,
    tooltip 照常绑定): 六页切换、亮/暗、最大化/还原、双击标题栏、拖动全部回归通过; demo 自己那套
    窗口/标题栏/双击代码已删(`captionH` / `shadowSize` 常量一并移除)。
- ✅ **发布 GitHub + 首个版本 v0.1.0**(2026-10-19):
  - 仓库公开(https://github.com/twgh/xui),`main` 已推;tag **v0.1.0** 已打并推送(指向 `7d2e7a7`)。
  - 依赖: `go.mod` 升到 `github.com/twgh/xcgui v1.4.3`(含 `svgp` / `wapi/d2d`), `go.sum` 已更新。
  - 验收(`GOWORK=off`, 即不依赖本地 xcgui 工作副本): `go build/vet/test ./...` 全过;
    全新项目 `go run .../cmd/xui@v0.1.0 init` 生成 theme.go + 基础组件 + DESIGN.md;
    `GOPROXY=direct go list -m ...@latest` → v0.1.0。
  - ⚠️ goproxy.cn 的 `@latest` 元数据有 CDN 缓存延迟, 刚发版时可能返回旧伪版本(内容同样是 main 最新),
    显式 `@v0.1.0` 或 `GOPROXY=direct` 不受影响; 其 zip 下载偶发一次 `stream error`(重试即好)。
- ✅ **xcgui 已发 v1.4.3(含 `d2d`/`svgp`), `go.mod` 已升级**(2026-10-19): 不再依赖 `go.work` 联调,
  `GOWORK=off go build ./...` 通过;发布前必须保持"公共模块可编译" —— 改回本地联调时记得再验证一次。
- ✅ **D2D 直通能力的上游化, 一次做齐**(2026-10-11)(趁上游发版窗口把原待办三项做完, 细节见 DESIGN 第八节 43 末 / 65):
  ① **"按值浮点形参"的封装扩到基本图元**: `DrawLine`(线宽在**栈[0]** —— 5 个参数, 位置 4 已超出前 4 个 XMM 槽)/ `DrawRectangle` / `DrawEllipse` / `FillRectangle` / `FillEllipse`(各带 `*Color` 便利版); 转发器升级为"最多 4 个栈参数"的 `wapi/com.CallWithFloat32Stack`(栈上的 float 占 8 字节槽、值放低 4 字节)。
  ② **形状几何与参数上移**: `ID2D1Factory` 补 `Create{Rectangle,RoundedRectangle,Ellipse}Geometry`(配 `D2D1_ELLIPSE`/`D2D1_ROUNDED_RECT`); 绘制/填充口参数从 `*ID2D1PathGeometry` 上移到 `*ID2D1Geometry`(各几何加 `Base()`, `Usable()` 随之上移), 于是"任意几何"都能画。
  ③ **几何缓存策略**做成通用的 `d2d.GeometryCache[K comparable]`(按工厂作废 + 失败也记 + 统一 Release), 本仓库的图标几何改用它 —— 重构后墨迹量逐像素不变(143083), 鼠标扫动 734/765ms(此前 828/836ms)。
  ④ 原来的"描边展开算法上游化"**作废**: 描边走 D2D 真描边(见上一条), CPU 展开代码已从本仓库删除。
  8 种图元/几何一次抓图全对; `wapi/com`/`d2d` 单测(槽位/布局/nil 安全)在 CI 与离线均可跑。

- ✅ **CLI 参数对齐 shadcn + 新项目脚手架**(2026-10-20):
  - `xui init -n <项目名>` 创建完整 Go 项目: `go.mod`(依赖版本从 CLI build info 取,
    本地 devel 构建回退 `v0.2.0`)、`main.go`(AppWindow + 示例卡片)、`README.md`、
    `.gitignore`、`ui/`(theme.go + DESIGN.md + 基础组件); 非空目录需 `-f/--force`;
    生成后**自动在项目目录执行 `go mod tidy`**（`GOWORK=off`、2 分钟超时，失败只告警不阻断），
    离线可加 `--no-install` 跳过。
  - `add/update`: `-o/--overwrite`(移除旧 `--force` 命名)、`-p/--path`、`-a/--all`、
    `--dry-run`、`--view[=文件]`、`--diff[=文件]`; 新增 `view` / `info` 命令; `list/search` 支持
    `-q/-l/-o`; 全局 `-c/--cwd`、`-s/--silent`。
  - `xui init [ui目录]` 旧行为不变; 参数解析、脚手架、dry-run、diff、过滤/分页均有单测。
  - 验收: 临时目录生成(自动生成 go.sum) → `go build` + 启动截图 + 关闭进程正常;
    手动跑过 list/view/dry-run/path/overwrite/diff/silent; `go build/vet/test ./...` 通过。
- ✅ **Menu 弹层宽度对齐 shadcn**(2026-10-20, trigger 宽 + min-w-32 下限):
  - 依据: `XMenu_SetItemWidth` 逐项设"条目文本区宽度"实测有效 —— 弹窗宽 = max(各条目有效宽) + 固定开销
    (BorderSize(4)/条目高 28 配置下 48), 且**设定值完全覆盖自然宽**(自然文本宽 229 的长条目设 60 后弹窗仍 108,
    长条目不会撑宽弹层)。早期"实测无效"的结论已证伪修正(详见 DESIGN 64)。
  - 实现(`components/menu.go`): `PopupAtElement` 记录触发区宽(逻辑像素), build 时给每个**顶级**条目设
    `max(触发区宽, 128) − 48` → 弹层宽 = max(触发区宽, 128), shadcn `w-(--radix-dropdown-menu-trigger-width)
    min-w-32` 语义齐了; 右键菜单(PopupAtCursor)无触发区 → 内容自适应 + 128 下限; 子菜单条目不设宽
    (shadcn SubContent 是内容自适应 + min-w-96, 不跟触发区); 分隔栏不设。长文本由自绘 `XDraw_SetClipRect`
    硬裁剪(= overflow-x-hidden; 原生条目文本同样裁在弹层边缘, 截图验证)。显式覆盖 = `Menu.Width(w)`
    (对应 shadcn 在 Content 上加 `className="w-56"` 这类宽度类 —— **shadcn 参考卡 App.tsx:815 就是 w-56=224**,
    所以参考页里看到的"默认"其实带了这个覆盖; demo 已同步 `Width(224)` 对齐)。
  - 验收: 探针(直接用 `ui.Menu` + 真实触发钮)断言弹窗客户区宽 —— 触发钮 240 → 240 ✓, 触发钮 60 → 128 ✓,
    `Width(224)` → 224 ✓; 窄触发 + 超长条目截图放大: 文本裁在弹层边缘内、勾选/快捷键/destructive/label 全部正常;
    `go build/vet/test ./...` 通过。

- ✅ **关窗后进程不退修复**(2026-10-20, 用户报障):
  - 现象: demo 点关闭钮后主窗口消失, 进程仍在(枚举到一个 `XCGUI` 残留小窗,
    悬停过标题栏按钮时还可能直接留在屏幕上)。
  - 根因: 消息循环的退出条件是**所有用户创建的窗口对象都销毁**, 不看可见性。Tooltip 的置顶小窗只 `SW_HIDE`
    不销毁, 且宿主销毁后挂在宿主上的退场过渡不再执行 —— 于是主窗口关了, `a.Run()` 还在跑。
  - 修法(`components/tooltip.go`): 气泡首次在某宿主窗口显示时给宿主挂 `WM_DESTROY`, 销毁气泡窗并清状态
    (先清状态再 `DestroyWindow`; 宿主销毁期间 `dying` 集合拒绝迟到的 `show()`, 避免 `XEle_SetRect` 无效句柄错误框);
    气泡属于其他仍存活宿主时保留。
  - 验收: ①气泡可见时关主窗 → 进程退出; ②气泡已隐藏时关 → 进程退出; ③标题栏"布局边界/主题"两个按钮各来一轮,
    均无残留窗口; `go build/vet/test ./...` 通过。详见 DESIGN 77。

## 已知限制(不是待办, 是 xcgui 的能力边界)

- ℹ️ Input 的文字/光标/选区/IME 由原生 Edit 绘制(只开放字体/字色/占位符色/插入符色/内边距, 见 DESIGN 三 第 4 条 / 23 / 61)。
- ℹ️ Toast 只支持四角堆叠(无顶部/底部居中、无 richColors, 见 DESIGN 75)。
- 完整清单见 DESIGN 第七节。

## 待办

- ⬜ **每加组件/跨文件助手就跑「逐组件单独安装 + 编译」**(方法见 REGISTRY_V2.md 五 末尾;2026-10-17 全量 21 项、2026-10-18 新增 calendar/datepicker/timepicker + add --all 通过; 注意 `xui init` 前临时项目必须已有 go.mod, 否则 `go run` 起不来)
- ⬜ **P4(发布后, 非阻塞)**: 文档站、主题画廊(把 shadcn 社区主题批量转译成 Go 令牌)

## 仓库结构

```
D:\GOProject\src\ui\xui\          ← 本仓库(模块 github.com/twgh/xui)
├── *.go                          运行时: color / theme / themes_builtin / manager
│                                 draw(Painter) / icons(图标几何+D2D 直通出图) / focus / font / state
│                                 anim(hover 过渡) / presence(进出场) / container·layout(布局骨架)
├── components\                   组件源码 = registry 内容(扁平单包 package ui)
│                                 button checkbox radio input label separator(含 Badge) card
│                                 switch progress slider tabs tooltip toast window combobox dialog
│                                 menu table tree pagination scrollarea
│                                 calendar datepicker timepicker(弹层骨架 popup.go)
├── icons\                        图标资源子包: lucide 预处理数据(2130 个) + 每图标一个
│                                 IconXxx 常量 + Path/Has/Names/Count/Version
├── scripts\gen_icons.py          lucide-static SVG → icons\lucide_gen.go(名字表)
│                                 + icons\lucide_icons_*.go(图标常量, 每 400 个一个文件)
├── cmd\xui\                      CLI(init / add / add --all / update / list; 数据源默认
│                                 远程 main 分支, XUI_REGISTRY 可覆盖, 失败降级内嵌)
├── icons_test.go                 解析器语法 + 资源完整性测试(go test ./...)
├── demo\                         演示应用(也是全组件编译测试目标)
├── registry.json                 组件清单(files + deps)
├── docs\DESIGN.md                知识库: 架构设计 + xcgui 踩坑全集(第八节必读)
├── docs\HANDOFF.md               本文
├── README.md                     用法
└── AGENTS.md                     agent 会话自动加载的入口规则

D:\GOProject\src\ui\shadcn\       ← 本地部署的 shadcn-ui 参考实现(语义权威)
├── src\components\ui\*.tsx       组件语义的唯一来源(hover/focus-visible/disabled 等)
├── src\index.css                 主题令牌(:root/.dark, nova 预设 neutral 色板)
└── src\App.tsx                   与 demo 同构的对照页
```

## 常用命令

```bash
# xui demo(全部组件的编译+视觉验证目标)
cd D:\GOProject\src\ui\xui
go build -ldflags="-s -w -H windowsgui" -trimpath -o demo/demo.exe ./demo && demo\demo.exe

# shadcn 参考实现 dev server(可能已在运行: netstat -ano | findstr 5173)
cd D:\GOProject\src\ui\shadcn && pnpm dev    # http://localhost:5173

# CLI 本地测试(默认源:直连→Gitee 镜像→GitCode 镜像→代理, 全不通才降级内嵌源码)
go run ./cmd/xui list / init <目录> / add <组件...> / add --all / update [组件...]
go test ./cmd/xui    # 候选顺序 / 数据源解析 / httptest 端到端 / 显式源失败退出
# 想验远程链路(无需 GitHub): 仓库根起 python -m http.server 8123,
# 再 XUI_REGISTRY=http://127.0.0.1:8123 go run ./cmd/xui add dialog
# 数据源也可指本地目录(完全离线): XUI_REGISTRY=D:\GOProject\src\ui\xui
# 依赖回归: 逐组件单独装到临时项目再 go build ./ui(方法见 REGISTRY_V2.md 五)

# 暗色对照验收: 环境变量决定启动主题, 不用改代码
#     XUI_THEME=dark  启动 demo 即暗色(cmd: set XUI_THEME=dark && demo\demo.exe)
#     运行中仍可用右上角按钮切换(参考页 shadcn 同样支持按 d 键切换)
# 直接启动到指定页签(验收某一页时省得点): XUI_HOME_PAGE=0..5 = 基础/表单/导航/反馈/数据/图标

# 全部测试(图标解析器语法 + 资源完整性 + 缓动曲线 + CLI 选路/数据源/端到端)
go test ./...
```

## 关键设计决策(一句话版, 详见 DESIGN.md)

1. **copy-not-install**:组件源码由 CLI 拷进用户 `ui/` 目录,归用户;运行时(theme/icons/helpers)走 go.mod
2. **绘制时取令牌**:组件永不缓存颜色,paint 回调里 `xui.Current()` 现取 → 切主题 = 换指针 + Redraw
3. **行为/视觉分层**:xcgui 原生控件 = 无头行为层,`AddEvent_Paint` + `drawx` 只管视觉
4. **交互语义逐条对齐 shadcn 源码**:改交互前先在 DESIGN 第四节末的语义汇总表确认,再读 `shadcn/src/components/ui/*.tsx`
5. **组件文件写薄**:绘制细节下沉运行时助手,降低拷贝模式下的升级冲突面
6. **界面用流式布局表达,不写绝对坐标**:窗口是唯一"天然确定尺寸"的父级,竖向主链从窗口开始一路 `Weight`(每级确定尺寸,下一级的 `Weight` 才算得出剩余空间);卡片高度用 `Auto` 自撑,要长高的面板整条链全 `Weight`。**别在中间垫 `Fill` 容器,也别用 `Fill` 当"剩余空间"**(尺寸语义见 DESIGN 第八节 55,样板在 `demo/main.go` 的 `row`/`colStretch`/`autoCard`)。页面本身是 `ui.ScrollArea` 滚动容器: 自撑型页面(内容 Auto 高,行高也必须 Auto)窗口小了就滚, 铺满型页面(数据页 `SetContentFill`)整链 Weight、面板跟随窗口长高(见第八节 66)

## 文档导航(去哪查什么)

| 想了解 | 去哪 |
|---|---|
| 当前进度 / 待办 / 命令 / 仓库结构 / 验证工作流 | **本文** |
| 架构原理、分层边界、主题令牌、组件解剖 | DESIGN.md 一~四节 |
| **交互语义(改交互前必看)** | DESIGN.md 第四节末「与 shadcn 对齐的语义(权威汇总)」+ 本地 shadcn 源码 |
| **我们与 shadcn 的差异 / 变体尺寸映射** | DESIGN.md 第四节末「变体/尺寸与上游的映射、已知差异」 |
| 某个 xcgui 行为 / 为什么这么写(踩坑) | DESIGN.md 第八节(按编号, 1~70) |
| 界面布局怎么写(容器/尺寸/随窗口自适应/滚动) | DESIGN.md 第八节 55(尺寸语义) + 56(Card) + 66(滚动容器), 样板在 `demo/main.go` |
| 用到的 xcgui API 清单 | DESIGN.md 第九节 |
| agent 会话硬规则 | AGENTS.md |
| 面向用户的安装与用法 | README.md |

## 视觉验证工作流(本项目惯例)

按模型能力分两条路径,像素级事实(颜色值/位置/尺寸/占比)在两条路径里都**用脚本测量、不靠目测**——"看起来的颜色"经常是错的,本项目多次靠像素测量纠偏:

**主模型有视觉**:

1. 构建 + 运行 demo → computer-use 截图(返回图片直接看)
2. `crop`(vision-skills 工具)或 computer-use zoom 放大细部目检
3. 精确数值用像素扫描脚本 `python scripts/pixel_scan.py <png> v|h|box|mix ...` 测量(如圆盘/白点直径占比、hover 前后色值)

**无视觉能力的模型**:

- vision CLI 在 `C:\Users\twgh\AppData\Local\agent-vision-toolkit\bin`:glance 看图/ground 定位/crop 放大
- 或主模型自己看完,把坐标与结论转述给子智能体

**通用**:

- 疑难渲染问题写探针程序(`.verify/probeN` 独立 main 包, `go build -o .verify/probeN/probeN.exe ./.verify/probeN` 后运行;`.verify/` 已被 go 工具链忽略),完事删掉
- 布局类问题**用 `xc.XEle_GetRectLogic` 打印各元素矩形**判定, 比截图量像素快且准(探针里 `AdjustLayout()` 之后读;注意窗口必须 `Show` 过才有非零尺寸, 见 DESIGN 8.55)
- 要按**物理/逻辑像素精确量测**时, 自己用 Win32 抓窗口(GetWindowRect + CopyFromScreen)比 MCP 截图可靠: MCP 截图会被缩放到模型尺寸。**抓之前必须把抓图进程设为 per-monitor DPI-aware**(`SetProcessDpiAwarenessContext(-4)`), 否则 GetWindowRect 给虚拟化坐标、CopyFromScreen 抓物理像素, 两者差 150% —— 抓出来的图左边会多出一条桌面(踩过)
- 需要"真实鼠标"验证 hover/点击时, 探针里用 `wapi.SetCursorPos` 比外部脚本可靠;从 PowerShell 移动鼠标要注意它是 DPI-unaware 进程, 坐标会被虚拟化/偏移, 别拿截图坐标直接当屏幕坐标用(踩过坑)。脚本里出现中文时**必须存成 UTF-8 **带 BOM**** —— PowerShell 5.1 按 ANSI 读 `.ps1`, 无 BOM 的中文串会把语法读坏(报一堆莫名其妙的 "Unexpected token")
- ⚠️ **定位窗口一律用"枚举进程顶层窗口 + 取可见且面积最大"**, 不要用 `Process.MainWindowHandle`: 弹层/提示这类小窗(带 `WS_EX_TOOLWINDOW`)一旦创建, 它就会返回**小窗**, 之后用它算出来的坐标、抓出来的图全是错的(踩过: 整个复现流程因此偏了半天)。诊断脚本见 DESIGN 第九节的 Win32 诊断 API 一行
- **"窗口可见 ≠ 屏幕上有东西"**: 判断弹层/提示这类异型透明窗有没有真的显示, 不能只看 `IsWindowVisible`/矩形 —— 用 `PrintWindow` 抓它自己的内容(证明"画了")与 `CopyFromScreen` 抓同一块屏幕(证明"屏幕上有没有")对照, 再读 `GetLayeredWindowAttributes` 的 alpha;`可见 + 矩形正确 + 像素为空` ⇒ 整窗 alpha/形状问题(见 57)
- **验收探针必须跑 `a.Run()` 消息循环**(无泵时 Presence/定时器全不工作, 结论作废)
- 浏览器对照:shadcn 参考页 + xui demo 同屏对比;同一张截图里量两边(ours vs ref)是最可靠的对照法
- **弹层类组件的暗色验收一律 zoom/像素测量**:全屏缩略截图里暗色弹层会假显亮底(36)

## 环境 / 注意事项

- shadcn dev server 由会话后台启动,机器重启后需按上文命令重启
- hover 过渡用运行时 `anim.go` 的炫彩定时器(不是 ani 包), 时长可用 `xui.SetHoverDuration` 覆盖(传 0 = 关掉过渡);**新增 hover 反馈只用 `xui.Hover`/`HoverItem` + `Fade`, 别自己起定时器**
- 用户项目须 `app.New(true)`(D2D)+ `EnableAutoDPI`,demo 是标准样板
- 弹层/对话框过渡挂**宿主窗口**,别挂弹层小窗;要 Win32 HWND 用窗口 wrapper 的 `GetHWND()`,别拿 xcgui 窗口句柄喂 `XWidget_GetHWND`(弹模态错误框)

## 图标系统(速览)

- `icons/` 子包内置 **2130 个** lucide 图标(lucide-static **1.52.0**), **每图标一个 `IconXxx` 常量**;按名字取用 `icons.Path("eye")`(给"名字来自用户输入/配置"的场合)。数据由 `scripts/gen_icons.py` 生成(默认取最新版, `--version` 可钉版)
- 出图走**直通 D2D 几何填充**(`d2dicon.go`), GDI 模式(`app.New(false)`)自动回落折线描边
- 体积(实测, 同基准 `-s -w`): 只用常量 **+0 字节**;调 `Path/Names` 引用整张表 **+426KB**
- demo 第 6 页「图标」是浏览入口(`XUI_HOME_PAGE=5`):9×4 格 + 搜索 + 翻页, 悬浮出 Tooltip 显示图标名、点击复制
- **原理、坑、性能结论见 DESIGN 第八节 43(抗锯齿 / figure 数)与 44(资源预处理 / 名字单射 / arc 标志位)**
