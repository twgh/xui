#!/usr/bin/env python3
"""把 lucide-static 的 SVG 图标预处理成 xui 的图标资源(Go 源码)。

lucide 的 SVG 里除 <path> 还有 circle/ellipse/rect/line/polyline/polygon, 而 xui 的图标
渲染只吃一条 path 数据(见 icons.go 的 parseIconPath), 所以这里做四件事:

1. 每个图元按文档顺序转成 path 指令, 拼成一条数据;
2. **相对指令全部改写成绝对指令**: 同一个 SVG 里的每个图元是独立的路径(各自从 (0,0) 起),
   直接拼起来后, 后面图元开头的 `m`/`l` 会以前面图元的终点为基准 —— 原点错乱。
   改写后拼顺序、拼位置都无影响, 数据也能直接肉眼核对;
3. 数字做最简归一(命令后的空格去掉, `0.5`→`.5`), 但**绝不把相邻数字粘成一个** ——
   arc 的 large-arc/sweep 是裸数字, 粘连后会被解析器当成一个数;
4. lucide 里 `fill="currentColor"` 的小图形(色点)是"填充"不是描边, 给其子路径加
   `!` 前缀(xui 解析器据此走填充分支)。

用法:
    python scripts/gen_icons.py                       # 默认取 lucide-static **最新版**并生成
    python scripts/gen_icons.py --version 1.49.0      # 钉住某个版本(与参考实现对齐时用)
    python scripts/gen_icons.py --src <解包目录>       # 用已解包的 lucide-static(离线)
    python scripts/gen_icons.py --only eye,search,x   # 只生成这几个(调试用)

生成两样东西(都提交进仓库, 用户 go get 拿到的是它, 不再需要联网/Node):
  icons/lucide_gen.go        名字→数据的表(按名字升序, Path 二分查找)
  icons/lucide_icons_NNNN.go 每图标一个常量 IconChevronDown = "...", 每 400 个切一个文件
表里存的是**常量本身**(`{"chevron-down", IconChevronDown}`), 所以常量与表天生不会漂移;
常量按需链接 —— 只用几个图标的程序编译期内联, 一个字节都不多花(实测 +0),
只有调用 Path/Has/Names 的程序才会把整张表链进二进制(实测 +400KB 量级)。

**重新生成后请跑 `go test ./...`**: 资源完整性/名字映射都有断言。
"""

import argparse
import io
import json
import os
import re
import shutil
import subprocess
import tempfile
import xml.etree.ElementTree as ET

NS = "{http://www.w3.org/2000/svg}"

NUM_RE = re.compile(r"[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?")
CMD_CHARS = set("MmLlHhVvCcSsQqTtAaZz")
SEP_CHARS = " \t\r\n,"


# ---------------------------------------------------------------- 数值格式

def fmt(v):
    """坐标 → 最简文本(0.5 → .5, 1.0 → 1)。"""
    s = f"{v:.4f}".rstrip("0").rstrip(".")
    if s.startswith("0."):
        s = s[1:]
    elif s.startswith("-0."):
        s = "-" + s[2:]
    if s in ("", "-", "-0"):
        s = "0"
    return s


def emit(groups):
    """把 (命令, 参数) 序列拼成最简且安全的 path 文本。

    省空格的两种情况: 命令字母前(前一个数字已被字母终止)与下一个数字带负号时。
    其余位置留一个空格 —— 非负数字相邻时必须分开(arc 的 flag 是裸数字, 粘起来就变一个数)。
    """
    out, prev_is_cmd = [], True
    for cmd, args in groups:
        out.append(cmd)
        prev_is_cmd = True
        for a in args:
            tok = fmt(a)
            out.append(("" if (prev_is_cmd or tok.startswith("-")) else " ") + tok)
            prev_is_cmd = False
    return "".join(out)


# ---------------------------------------------------------------- 图元 → path

def circle_d(a):
    cx, cy, r = float(a["cx"]), float(a["cy"]), float(a["r"])
    return f"M{cx} {cy - r}a{r} {r} 0 1 1 0 {2 * r}a{r} {r} 0 1 1 0 {-2 * r}Z"


def ellipse_d(a):
    cx, cy, rx, ry = float(a["cx"]), float(a["cy"]), float(a["rx"]), float(a["ry"])
    return (f"M{cx - rx} {cy}a{rx} {ry} 0 1 1 {2 * rx} 0a{rx} {ry} 0 1 1 {-2 * rx} 0Z")


def rect_d(a):
    x, y = float(a.get("x", 0)), float(a.get("y", 0))
    w, h = float(a["width"]), float(a["height"])
    rx = float(a["rx"]) if a.get("rx") is not None else (float(a["ry"]) if a.get("ry") is not None else 0.0)
    ry = float(a["ry"]) if a.get("ry") is not None else rx
    if rx <= 0 or ry <= 0:
        return f"M{x} {y}h{w}v{h}h{-w}Z"
    rx, ry = min(rx, w / 2), min(ry, h / 2)
    return (f"M{x + rx} {y}h{w - 2 * rx}a{rx} {ry} 0 0 1 {rx} {ry}"
            f"v{h - 2 * ry}a{rx} {ry} 0 0 1 {-rx} {ry}"
            f"h{-(w - 2 * rx)}a{rx} {ry} 0 0 1 {-rx} {-ry}"
            f"v{-(h - 2 * ry)}a{rx} {ry} 0 0 1 {rx} {-ry}Z")


def line_d(a):
    return f"M{a['x1']} {a['y1']}L{a['x2']} {a['y2']}"


def points_d(points, close):
    nums = NUM_RE.findall(points.replace(",", " "))
    if len(nums) < 4:
        return ""
    d = f"M{nums[0]} {nums[1]}" + "".join(
        f"L{nums[i]} {nums[i + 1]}" for i in range(2, len(nums) - 1, 2))
    return d + ("Z" if close else "")


# ---------------------------------------------------------------- 相对 → 绝对

class Scanner:
    """SVG path 数据的游标扫描器。

    为什么不能简单按 token 切: **arc 的 large-arc/sweep 是单字符标志**,
    `a.25.25 0 00.407.195` 里 `00.407.195` 是 flag=0 / flag=0 / x=.407 / y=.195,
    按"数字"切会读成 `00.407` 一个数, 后面整段图形就被吃掉(运行时解析器同样要按这个规则)。
    """

    def __init__(self, s):
        self.s, self.i = s, 0

    def skip(self):
        while self.i < len(self.s) and self.s[self.i] in SEP_CHARS:
            self.i += 1

    def cmd(self):
        self.skip()
        if self.i < len(self.s) and self.s[self.i] in CMD_CHARS:
            c = self.s[self.i]
            self.i += 1
            return c
        return None

    def num(self):
        self.skip()
        m = NUM_RE.match(self.s, self.i)
        if not m or m.group() in ("", "+", "-"):
            raise ValueError(f"不是数字: {self.s[self.i:self.i + 12]!r}")
        self.i = m.end()
        return float(m.group())

    def flag(self):
        self.skip()
        if self.i >= len(self.s) or self.s[self.i] not in "01":
            raise ValueError(f"不是 arc 标志(0/1): {self.s[self.i:self.i + 12]!r}")
        c = self.s[self.i]
        self.i += 1
        return float(c)

    def eof(self):
        self.skip()
        return self.i >= len(self.s)


def absolutize(d):
    """把 path 数据改写成"全绝对 + 显式命令", 返回子路径列表(每个是 (命令, 参数) 的列表)。

    隐式重复组会展开成显式命令; M 后面的隐式组按 L 处理(与 SVG 语法一致);
    H/V 归一成 L。任何畸形数据直接报错 —— 生成器宁可崩, 也不要产坏数据。
    """
    sc = Scanner(d)
    subs, cur = [], None
    x = y = sx = sy = 0.0
    cmd = None
    while True:
        c = sc.cmd()
        if c is not None:
            cmd = c
            if cmd.upper() == "Z":
                if cur is None:
                    raise ValueError("Z 作为首条命令")
                cur.append(("Z", []))
                x, y = sx, sy
                cmd = None
            continue
        if sc.eof():
            break
        if cmd is None:
            raise ValueError(f"数字没有归属的命令: {d!r}")
        rel, C = cmd.islower(), cmd.upper()
        if C == "M":
            ex, ey = sc.num(), sc.num()
            x, y = (x + ex, y + ey) if rel else (ex, ey)
            sx, sy = x, y
            if cur:
                subs.append(cur)
            cur = [("M", [x, y])]
            cmd = "l" if rel else "L"
        elif C == "L":
            ex, ey = sc.num(), sc.num()
            x, y = (x + ex, y + ey) if rel else (ex, ey)
            cur.append(("L", [x, y]))
        elif C == "H":
            v = sc.num()
            x = x + v if rel else v
            cur.append(("L", [x, y]))
        elif C == "V":
            v = sc.num()
            y = y + v if rel else v
            cur.append(("L", [x, y]))
        elif C == "C":
            a = [sc.num() for _ in range(6)]
            if rel:
                a = [x + a[0], y + a[1], x + a[2], y + a[3], x + a[4], y + a[5]]
            cur.append(("C", a))
            x, y = a[4], a[5]
        elif C == "S":
            a = [sc.num() for _ in range(4)]
            if rel:
                a = [x + a[0], y + a[1], x + a[2], y + a[3]]
            cur.append(("S", a))
            x, y = a[2], a[3]
        elif C == "Q":
            a = [sc.num() for _ in range(4)]
            if rel:
                a = [x + a[0], y + a[1], x + a[2], y + a[3]]
            cur.append(("Q", a))
            x, y = a[2], a[3]
        elif C == "T":
            ex, ey = sc.num(), sc.num()
            x, y = (x + ex, y + ey) if rel else (ex, ey)
            cur.append(("T", [x, y]))
        elif C == "A":
            rx, ry, rot = sc.num(), sc.num(), sc.num()
            laf, sf = sc.flag(), sc.flag()      # 单字符标志, 逐字符取
            ex, ey = sc.num(), sc.num()
            x, y = (x + ex, y + ey) if rel else (ex, ey)
            cur.append(("A", [rx, ry, rot, laf, sf, x, y]))
        else:
            raise ValueError(f"未处理的命令 {cmd!r}")
    if cur:
        subs.append(cur)
    return subs


def element_subpaths(tag, a):
    """返回该元素的子路径文本列表(已带 `!` 填充标记)。

    注意: lucide 里 `fill="currentColor"` 的元素**同时也继承根节点的描边**
    (stroke=currentColor / stroke-width=2), 所以它在浏览器里是"填充 + 描边"的并集 ——
    例如 chart-scatter 的 `circle r=.5` 实际画出来是半径 1.5 的实心点, 不是半径 .5 的点。
    因此填充图元要同时输出填充与描边两条子路径, 否则点会明显偏小。
    """
    filled = (a.get("fill") or "none") != "none"
    if tag == "path":
        d = a["d"]
    elif tag == "circle":
        d = circle_d(a)
    elif tag == "ellipse":
        d = ellipse_d(a)
    elif tag == "rect":
        d = rect_d(a)
    elif tag == "line":
        d = line_d(a)
    elif tag == "polyline":
        d = points_d(a["points"], False)
    elif tag == "polygon":
        d = points_d(a["points"], True)
    else:
        raise SystemExit(f"未知图元 <{tag}>, 需要在 element_subpaths 里补转换规则")
    out = []
    for sub in absolutize(d):
        if not sub:
            continue
        s = emit(sub)
        if filled:  # 填充 + 描边并集(见函数注释)
            out.append("!" + s)
        out.append(s)
    return out


def icon_path(svg_file):
    root = ET.parse(svg_file).getroot()
    vb = [float(v) for v in root.get("viewBox", "0 0 24 24").split()]
    if vb != [0.0, 0.0, 24.0, 24.0]:
        raise SystemExit(f"{svg_file}: viewBox 不是 0 0 24 24, 本渲染管线按 24 网格")
    parts = []
    for ch in root:
        tag = ch.tag.replace(NS, "")
        if tag in ("title", "desc", "metadata", "defs", "g"):
            continue
        parts.extend(element_subpaths(tag, ch.attrib))
    return " ".join(parts)


# ---------------------------------------------------------------- 取包

def latest_version():
    """问 npm 要 lucide-static 的最新版本号(缺省行为: 生成最新版)。"""
    npm = shutil.which("npm")
    if not npm:
        raise SystemExit("找不到 npm; 请手动 `npm pack lucide-static@<版本>` 解包后用 --src 指定")
    out = subprocess.run([npm, "view", "lucide-static", "version"], check=True,
                         capture_output=True, text=True)
    return out.stdout.strip()


def read_version(pkg_dir):
    """离线(--src)模式: 从解包目录的 package.json 里读版本号(读不到就写 unknown)。"""
    try:
        with io.open(os.path.join(pkg_dir, "package.json"), encoding="utf-8") as f:
            return json.load(f).get("version", "unknown")
    except (OSError, ValueError):
        return "unknown"


def fetch_lucide(version, workdir):
    npm = shutil.which("npm")
    if not npm:
        raise SystemExit(f"找不到 npm; 请手动 `npm pack lucide-static@{version}` 解包后用 --src 指定")
    print(f"→ npm pack lucide-static@{version}(联网)")
    subprocess.run([npm, "pack", f"lucide-static@{version}"], cwd=workdir, check=True,
                   stdout=subprocess.DEVNULL)
    tgz = [f for f in os.listdir(workdir) if f.endswith(".tgz")][0]
    subprocess.run(["tar", "-xzf", tgz], cwd=workdir, check=True)
    return os.path.join(workdir, "package")


# ---------------------------------------------------------------- 名字 → Go 标识符

# 每文件多少个图标常量(生成的源文件会很大, 切开对编辑器/gopls 友好)
CHUNK = 400

# 名字映射的抽查: 生成器只顾自己那套转换规则很容易"自洽但错了", 所以钉几个已知结果。
# 下面这几对名字在 lucide 里**同时存在**, 是"规则必须单射"的活样本:
#   arrow-down-0-1 / arrow-down-01、axis-3-d / axis-3d、grid-2-x-2 / grid-2x2
SPOT = {
    "chevron-down": "IconChevronDown",
    "eye-off": "IconEyeOff",
    "x": "IconX",
    "arrow-down-0-1": "IconArrowDown0_1",
    "arrow-down-01": "IconArrowDown01",
    "axis-3-d": "IconAxis3D",
    "axis-3d": "IconAxis3d",
    "grid-2-x-2": "IconGrid2X2",
    "grid-2x2": "IconGrid2x2",
}


def go_const(name):
    """lucide kebab 名 → Go 常量名: chevron-down → IconChevronDown。

    逐段首字母大写; **只有"数字段紧跟数字段"时补一个下划线**: lucide 里
    `arrow-down-0-1` 与 `arrow-down-01` 都存在, 不在那里分开就会撞成同一个标识符。
    其余连字符边界靠大小写区分(axis-3-d → IconAxis3D vs axis-3d → IconAxis3d),
    所以整体单射 —— 生成器另有冲突检查兜底, 撞了就报错而不是静默覆盖。
    加 `Icon` 前缀是为了和包里的 Path/Has/Names/Count/Version 以及将来的函数名互不干扰。
    """
    out, prev_digit = [], False
    for p in name.split("-"):
        if not re.fullmatch(r"[a-z0-9]+", p):
            raise SystemExit(f"图标名 {name!r} 含意外字符(只支持小写字母/数字/连字符)")
        chunk = p[:1].upper() + p[1:]
        if prev_digit and chunk[0].isdigit():
            out.append("_")
        out.append(chunk)
        prev_digit = chunk[-1].isdigit()
    return "Icon" + "".join(out)


def check_names(data):
    """标识符必须单射: 两个名字撞同一个常量名就直接报错(不许静默覆盖)。"""
    seen = {}
    for name in sorted(data):
        c = go_const(name)
        if c in seen:
            raise SystemExit(f"图标名 {name!r} 与 {seen[c]!r} 都映射到常量 {c}, 冲突")
        if c in ("Icon", "Version"):
            raise SystemExit(f"图标名 {name!r} 的常量名 {c} 与包内标识符冲突")
        seen[c] = name
    for name, want in SPOT.items():
        if name in data and go_const(name) != want:
            raise SystemExit(f"名字映射抽查失败: {name!r} → {go_const(name)}, 期望 {want}")


# ---------------------------------------------------------------- 生成 Go

DOC = """// Code generated by scripts/gen_icons.py from lucide-static v{version}; DO NOT EDIT.
//
// lucide 图标资源(24 网格 path 数据; 数据来源 npm pack lucide-static@{version} 的
// package/icons/*.svg, 含 lucide 保留的废弃别名图标)。
//
// 每个图标是一条完整的 path 数据(lucide 约定 stroke-width=2、round cap/join):
//   - circle/ellipse/rect/line/polyline/polygon 都已转成等价的 path 指令;
//   - 相对指令已全部改写成绝对指令(跨图元拼接的前提), 隐式重复组已展开;
//   - 数字是最简写法, 但相邻数字之间保留必要空格(arc 的 large-arc/sweep 是裸数字);
//   - `!` 前缀的子路径表示**填充**; lucide 里 fill="currentColor" 的色点还同时继承根节点的
//     描边, 所以那类图元输出"填充 + 描边"两条子路径(与浏览器/lucide-react 的渲染一致)。
//
// 两种用法:
//
//	p.Icon(icons.IconChevronDown, &rc, color, 2) // 常量: 编译期内联, 零体积成本(推荐)
//	p.Icon(icons.Path("star"), &rc, color, 2)    // 按名字取: 会把整张表链进二进制
package icons
"""

TABLE_HEAD = DOC + '''
// Version 是数据来源的 lucide-static 版本(与 shadcn 参考实现的 lucide-react 同版本)。
const Version = "{version}"

// entry 是一条图标数据; all 按 name 升序排列, Path 用二分查找(避免 map 初始化开销)。
// 表里存的是**常量本身**, 所以"按名字取"和"直接引用常量"永远是同一份数据。
type entry struct {{
	name string
	d    string
}}

var all = []entry{{
'''

CONSTS_HEAD = DOC + '''
// 本文件是图标常量的一部分(全部常量见同目录 lucide_icons_*.go, 按名字升序切块):
// {first} .. {last}
//
// 一行一个独立 const 声明(而不是一个 const 块): 块内 gofmt 会给整块补齐 `=` 对齐,
// 2000 多行平均多出几十个空格, 只是白占源码体积。
'''


def write_go(out_dir, data, version):
    """写 lucide_gen.go(表) + lucide_icons_*.go(常量, 分块)。返回 (常量文件数, 常量数)。"""
    names = sorted(data)
    check_names(data)

    # 表: 名字 → 常量
    out = [TABLE_HEAD.format(version=version)]
    for name in names:
        out.append(f'\t{{"{name}", {go_const(name)}}},\n')
    out.append("}\n")
    with io.open(os.path.join(out_dir, "lucide_gen.go"), "w", encoding="utf-8", newline="\n") as f:
        f.write("".join(out))

    # 常量: 分块(先删掉上一轮的块文件, 免得图标变少后留下孤儿)
    for old in os.listdir(out_dir):
        if re.fullmatch(r"lucide_icons_\d+\.go", old):
            os.remove(os.path.join(out_dir, old))
    nchunk = 0
    for start in range(0, len(names), CHUNK):
        chunk = names[start:start + CHUNK]
        nchunk += 1
        body = [CONSTS_HEAD.format(version=version, first=chunk[0], last=chunk[-1])]
        for name in chunk:
            d = data[name].replace("\\", "\\\\").replace('"', '\\"')
            body.append(f'const {go_const(name)} = "{d}"\n')
        path = os.path.join(out_dir, f"lucide_icons_{start:04d}.go")
        with io.open(path, "w", encoding="utf-8", newline="\n") as f:
            f.write("".join(body))
    return nchunk, len(names)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", help="lucide-static 解包目录(含 icons/*.svg); 缺省自动 npm pack")
    ap.add_argument("--out-dir", default="icons")
    ap.add_argument("--version", default="", help="lucide-static 版本, 缺省用最新版")
    ap.add_argument("--only", help="只生成这些图标(逗号分隔, 调试用)")
    args = ap.parse_args()

    tmp = None
    pkg = args.src
    if not pkg:
        if not args.version:
            args.version = latest_version()
            print(f"→ lucide-static 最新版 {args.version}(需要钉版用 --version)")
        tmp = tempfile.mkdtemp(prefix="lucide-static-")
        pkg = fetch_lucide(args.version, tmp)
    icons_dir = os.path.join(pkg, "icons")
    if not os.path.isdir(icons_dir):
        raise SystemExit(f"{icons_dir} 不存在")
    if not args.version:  # --src 离线模式: 从包信息里读版本号
        args.version = read_version(pkg)

    only = set(args.only.split(",")) if args.only else None
    data, skipped = {}, []
    for f in sorted(os.listdir(icons_dir)):
        if not f.endswith(".svg"):
            continue
        name = f[:-4]
        if only and name not in only:
            continue
        try:
            d = icon_path(os.path.join(icons_dir, f))
        except Exception as e:
            raise SystemExit(f"{f}: {e}")
        if d:
            data[name] = d
        else:
            skipped.append(name)

    os.makedirs(args.out_dir, exist_ok=True)
    nchunk, nconst = write_go(args.out_dir, data, args.version)
    if tmp:
        shutil.rmtree(tmp, ignore_errors=True)

    total = sum(len(v) for v in data.values())
    tab = os.path.join(args.out_dir, "lucide_gen.go")
    print(f"→ {nconst} 个图标: 表 {tab} ({os.path.getsize(tab) / 1024:.0f} KB; path 数据 "
          f"{total / 1024:.0f} KB, 平均 {total / max(len(data), 1):.0f} 字符/图标); "
          f"常量 {nchunk} 个文件(每 {CHUNK} 个)")
    if skipped:
        print(f"   空图标 {len(skipped)} 个: {skipped[:5]}")


if __name__ == "__main__":
    main()
