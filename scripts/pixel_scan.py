#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""像素扫描/测量助手(视觉验收用): 沿一条线把截图切成颜色分段, 或统计区域主色。

为什么需要它: 目测颜色/尺寸经常是错的 —— 本仓库多次靠像素测量纠偏
(例如"暗色弹层在全屏缩略图里假显亮底"、"hover 到底有没有画上去")。

用法:
    python scripts/pixel_scan.py <png> v <x> <y0> <y1>              # 纵向扫描
    python scripts/pixel_scan.py <png> h <y> <x0> <x1>              # 横向扫描
    python scripts/pixel_scan.py <png> mix <n>                      # 全图出现最多的 n 种颜色
    python scripts/pixel_scan.py <png> box <x0> <y0> <x1> <y1> [n]  # 区域内主色统计

注意: 屏幕截图工具输出的坐标系未必与 Win32 屏幕坐标一致(实测 utools 截图在 y 方向
有整体偏移), 定位元素时先用扫描找到已知边界(如卡片描边)再换算。
"""

import sys
from collections import Counter

from PIL import Image


def runs(pixels):
    """把像素序列切成 (start, end, color) 分段。"""
    out = []
    prev = None
    start = 0
    for i, c in enumerate(pixels):
        if prev is None:
            prev, start = c, 0
            continue
        if c != prev:
            out.append((start, i - 1, prev))
            prev, start = c, i
    if prev is not None:
        out.append((start, len(pixels) - 1, prev))
    return out


def dump(items, offset, axis):
    for a, b, c in items:
        print("%s %5d..%5d (%4dpx) #%02x%02x%02x" % (axis, offset + a, offset + b, b - a + 1, c[0], c[1], c[2]))


def top_colors(px, n):
    for c, k in Counter(px).most_common(n):
        print("#%02x%02x%02x  %d" % (c[0], c[1], c[2], k))


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        return 1
    path, mode = sys.argv[1], sys.argv[2]
    im = Image.open(path).convert("RGB")
    w, h = im.size
    print("image %dx%d" % (w, h))

    if mode == "v":
        x, y0, y1 = int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5])
        dump(runs([im.getpixel((x, y)) for y in range(y0, y1)]), y0, "y")
    elif mode == "h":
        y, x0, x1 = int(sys.argv[3]), int(sys.argv[4]), int(sys.argv[5])
        dump(runs([im.getpixel((x, y)) for x in range(x0, x1)]), x0, "x")
    elif mode == "mix":
        top_colors([im.getpixel((x, y)) for y in range(h) for x in range(w)], int(sys.argv[3]))
    elif mode == "box":
        x0, y0, x1, y1 = (int(v) for v in sys.argv[3:7])
        n = int(sys.argv[7]) if len(sys.argv) > 7 else 10
        top_colors([im.getpixel((x, y)) for y in range(y0, y1) for x in range(x0, x1)], n)
    else:
        print(__doc__)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
