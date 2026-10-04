import re
p = r"d:\GOProject\src\ui\xui\icons.go"
s = open(p, encoding="utf-8").read()
s = s.replace('\t"strconv"', '\t"math"\n\t"strconv"', 1)
s = s.replace("lineW := int32(strokeWidth*s + 0.5)", "lineW := int32(math.Ceil(float64(strokeWidth * s))) // 线宽是物理像素: 取整会让 16px 图标(2*16/24=1.33)退化成 1px 细线")
open(p, "w", encoding="utf-8", newline="").write(s)
print("patched", s.count("math.Ceil"))
