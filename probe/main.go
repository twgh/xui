// 临时探针: 对比图标的抗锯齿管线。验证完删除。
//
// 一个窗口三组对照:
//   A. 采样圆弧折线 + XDraw_DrawLineF(icons.go 同款管线)
//   B. 同一份折线点喂 D2D PathGeometry, RT::DrawGeometry 描边(D2D AA 测试)
//   C. 斜线对照: XDraw_DrawLineF vs D2D RT::DrawLine
// 控制台打印 RT GetAntialiasMode。
package main

import (
	"log"
	"math"
	"os"
	"syscall"
	"unsafe"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
)

// ---------- COM vtable ----------

func vcall(obj uintptr, index int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(vtbl)))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func f32(v float32) uintptr { return uintptr(math.Float32bits(v)) }

type colorF struct{ R, G, B, A float32 }

// ID2D1RenderTarget vtable(d2d1.h 顺序; IUnknown(0-2) → Resource.GetFactory(3) →
// Create* 4-14 → Draw/Fill* 15-29 → SetTransform(30)... GetAntialiasMode(33))
const (
	rtCreateSolidColorBrush = 8
	rtFillGeometry          = 23
	rtGetAntialiasMode      = 33
)

// ID2D1Factory / ID2D1PathGeometry(继承 ID2D1Geometry 13 个方法, Open 在 17) / ID2D1GeometrySink
// Simplified 部分只有 7 个方法: SetFillMode(3) SetSegmentFlags(4) BeginFigure(5)
//   AddLines(6) AddBeziers(7) EndFigure(8) Close(9);
// GeometrySink 追加: AddBezier(10) AddQuadraticBezier(11) AddQuadraticBeziers(12) AddArc(13)
const (
	facCreatePathGeometry = 10
	pgOpen                = 17
	sinkBeginFigure       = 5
	sinkAddLines          = 6
	sinkEndFigure         = 8
	sinkClose             = 9
)

func createSolidBrush(rt uintptr, c colorF) uintptr {
	var brush uintptr
	vcall(rt, rtCreateSolidColorBrush,
		uintptr(unsafe.Pointer(&c)), 0, uintptr(unsafe.Pointer(&brush)))
	return brush
}

func circlePts(cx, cy, r float32, n int) []float32 {
	pts := make([]float32, 0, (n+1)*2)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n) * 2 * math.Pi
		pts = append(pts, cx+r*float32(math.Cos(t)), cy+r*float32(math.Sin(t)))
	}
	return pts
}

// 二分开关: 0=只跑 XDraw; 1=取RT+查AA; 2=+建画刷; 3=+建几何; 4=+FillGeometry
var enableD2D = 4

func init() {
	if v := os.Getenv("PROBE_D2D"); v != "" {
		n := 0
		for _, c := range v {
			n = n*10 + int(c-'0')
		}
		enableD2D = n
	}
}

// 折线点(扁平 x,y)喂给 D2D PathGeometry(单条开放折线)
func makeGeometry(factory uintptr, pts []float32) uintptr {
	return makeGeom(factory, pts, 0 /*OPEN*/)
}

// 闭合折线(用于填充)
func makeGeometryClosed(factory uintptr, pts []float32) uintptr {
	return makeGeom(factory, pts, 1 /*CLOSED*/)
}

func makeGeom(factory uintptr, pts []float32, figureEnd uintptr) uintptr {
	var geom uintptr
	hr := vcall(factory, facCreatePathGeometry, uintptr(unsafe.Pointer(&geom)))
	log.Printf("CreatePathGeometry hr=%x geom=%x", hr&0xffffffff, geom)
	if geom == 0 {
		panic("CreatePathGeometry failed")
	}
	var sink uintptr
	hr = vcall(geom, pgOpen, uintptr(unsafe.Pointer(&sink)))
	log.Printf("Open hr=%x sink=%x", hr&0xffffffff, sink)
	if sink == 0 {
		panic("Open failed")
	}
	log.Println("BeginFigure")
	vcall(sink, sinkBeginFigure,
		uintptr(f32(pts[0]))|uintptr(f32(pts[1]))<<32, 0)
	log.Println("AddLines")
	vcall(sink, sinkAddLines,
		uintptr(unsafe.Pointer(&pts[2])), uintptr(len(pts)/2-1))
	log.Println("EndFigure")
	vcall(sink, sinkEndFigure, figureEnd)
	log.Println("Close")
	hrC := vcall(sink, sinkClose)
	log.Printf("sink.Close hr=%x", hrC&0xffffffff)
	return geom
}

func main() {
	f, _ := os.Create("D:/GOProject/src/ui/xui/probe/log.txt")
	defer f.Close()
	log.SetOutput(f)
	log.Println("step: init")
	app.InitOrExit()
	a := app.New(true) // D2D 渲染
	a.EnableDPI()
	log.Println("step: app created")

	w := window.New(0, 0, 420, 280, "AA probe", 0, xcc.Window_Style_Default)
	ele := widget.NewElement(0, 0, 400, 240, w.Handle)
	log.Println("step: window created")

	arcPts := circlePts(60, 70, 22, 48) // 48 段采样圆(半径 22)

	ele.AddEvent_Paint(func(hEle int, hDraw int, pbHandled *bool) int {
		*pbHandled = true
		log.Printf("paint enter, level=%d", enableD2D)
		var rt, factory, brush uintptr
		if enableD2D >= 1 {
			rt = xc.XDraw_GetD2dRenderTarget(hDraw)
			if rt == 0 {
				log.Println("no D2D RT")
				return 0
			}
			log.Printf("idx32(SetAA?)=%x idx33(GetAA?)=%x idx34=%x idx35=%x",
				vcall(rt, 32)&0xffffffff, vcall(rt, 33)&0xffffffff,
				vcall(rt, 34)&0xffffffff, vcall(rt, 35)&0xffffffff)
		}
		if enableD2D >= 2 {
			factory = xc.XC_GetD2dFactory()
			log.Printf("factory=%x", factory)
		}
		if enableD2D >= 3 {
			brush = createSolidBrush(rt, colorF{0, 0, 0, 1})
			log.Printf("brush=%x", brush)
		}

		// A: 折线 + XDraw_DrawLineF(icons.go 同款), 行高 -30 平移到 y=40
		xc.XDraw_SetLineWidth(hDraw, 2)
		for i := 2; i < len(arcPts); i += 2 {
			xc.XDraw_DrawLineF(hDraw, arcPts[i-2], arcPts[i-1]-30, arcPts[i], arcPts[i+1]-30)
		}

		// B: 同折线点右移 150 → D2D FillGeometry(闭合填充, 无 float 参数)
		if enableD2D >= 4 {
			var geom uintptr
			if enableD2D == 6 {
				// 对照: 用 CreateEllipseGeometry(factory idx 7), 排除手搓 path 几何的问题
				// D2D1_ELLIPSE = {POINT_2F point; float rx, ry} = 24 字节, 指针传递
				var ellipse [6]float32
				ellipse[0], ellipse[1] = 210, 70 // 圆心
				ellipse[2], ellipse[3] = 22, 22  // 半径
				hrE := vcall(factory, 7, uintptr(unsafe.Pointer(&ellipse)), uintptr(unsafe.Pointer(&geom)))
				log.Printf("CreateEllipseGeometry hr=%x geom=%x", hrE&0xffffffff, geom)
			} else if enableD2D == 7 {
				// 最小 path: 三角形(FILLED+CLOSED), 验证 sink 流程本身
				tri := []float32{200, 40, 280, 100, 200, 100}
				geom = makeGeom(factory, tri, 1)
				log.Printf("triangle geom=%x", geom)
			} else {
				off := make([]float32, len(arcPts))
				for i := 0; i < len(arcPts); i += 2 {
					off[i], off[i+1] = arcPts[i]+150, arcPts[i+1]-30
				}
				geom = makeGeometryClosed(factory, off)
				log.Printf("geom=%x", geom)
			}
			// GetBounds(geom idx 4, transform=NULL) 验证几何合法性
			var bounds [4]float32
			hrB := vcall(geom, 4, 0, uintptr(unsafe.Pointer(&bounds)))
			log.Printf("GetBounds hr=%x bounds=(%v,%v)-(%v,%v)", hrB&0xffffffff,
				bounds[0], bounds[1], bounds[2], bounds[3])
			if enableD2D >= 5 {
				hr := vcall(rt, rtFillGeometry, geom, brush, 0)
				log.Printf("FillGeometry called hr=%x", hr&0xffffffff)
			}
		}

		// C: 斜线对照: 左 XDraw
		xc.XDraw_DrawLineF(hDraw, 30, 150, 90, 210)

		log.Println("paint done")
		return 0
	})

	w.Show()
	log.Println("step: shown, entering run loop")
	a.Run()
	log.Println("step: run loop exited")
	a.Exit()
}
