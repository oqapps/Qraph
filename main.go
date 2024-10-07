package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/aquilax/go-perlin"
	dialog2 "github.com/sqweek/dialog"
)

var graph = image.NewRGBA64(image.Rect(0, 0, 1200, 1200))

var p = perlin.NewPerlin(2, 2, 1, 39530)

func newWhiteBackground(w, h int) *image.RGBA {
	var whiteBackground = image.NewRGBA(image.Rect(0, 0, w, h))
	min := whiteBackground.Rect.Min
	max := whiteBackground.Rect.Max

	for x := min.X; x < max.X; x++ {
		for y := min.Y; y < max.Y; y++ {
			whiteBackground.Set(x, y, white)
		}
	}

	return whiteBackground
}

func main() {
	a := app.New()
	w := a.NewWindow("Qraph")

	w.SetContent(
		container.NewBorder(
			container.NewCenter(widget.NewRichTextFromMarkdown("# Qraph")), nil, nil, nil,
			container.NewAppTabs(container.NewTabItem("Equations", equationsPage()), container.NewTabItem("Noise", perlinPage(w))),
		))
	w.Resize(fyne.NewSize(400, 600))
	w.ShowAndRun()
}

var white = color.RGBA{255, 255, 255, 255}

func equationsPage() fyne.CanvasObject {
	img := canvas.NewImageFromImage(graph)
	img.ScaleMode = canvas.ImageScaleFastest
	img.FillMode = canvas.ImageFillContain

	addGraph(constantX(0), color.White)
	addGraph(constantY(0), color.White)

	top := widget.NewButton("Add Quadratic", func() {
		addGraph(quadratic(rand.Float64()*0.05, rand.Float64()*6, rand.Float64()*6), nil)

		img.Refresh()
	})

	return container.NewBorder(top, nil, nil, nil, img)
}

func perlinPage(w fyne.Window) fyne.CanvasObject {
	whiteBackground := canvas.NewImageFromImage(newWhiteBackground(1200, 1200))
	whiteBackground.ScaleMode = canvas.ImageScaleFastest
	whiteBackground.FillMode = canvas.ImageFillContain

	var graph = image.NewNRGBA64(image.Rect(0, 0, 1200, 1200))

	var a, b = 2.0, 2.0
	var n int32 = 1
	var seed int64 = 123456
	var colorMode = "NRGBA64"
	var individualRefresh = false

	var xDivide = 15.0
	var zDivide = 15.0
	var intensify = 1.0

	var alphaInput = widget.NewEntry()
	var betaInput = widget.NewEntry()
	var iterationsInput = widget.NewEntry()
	var seedInput = widget.NewEntry()

	alphaInput.SetText(strconv.FormatFloat(a, 'f', 2, 64))
	betaInput.SetText(strconv.FormatFloat(b, 'f', 2, 64))
	iterationsInput.SetText(strconv.FormatInt(int64(n), 10))
	seedInput.SetText(strconv.FormatInt(int64(seed), 10))

	alphaInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		a = i
	}
	betaInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		b = i
	}
	iterationsInput.OnChanged = func(s string) {
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return
		}
		n = int32(i)
	}
	seedInput.OnChanged = func(s string) {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
		seed = i
	}

	img := canvas.NewImageFromImage(graph)
	img.ScaleMode = canvas.ImageScaleFastest
	img.FillMode = canvas.ImageFillContain

	var pixelPerSecond = widget.NewLabel("N/A pps")
	var rendering = widget.NewLabel("Rendering... N/A%")

	var rtc int32

	var resetButton = widget.NewButton("Apply", func() {
		t := time.Now()
		p := perlin.NewPerlin(a, b, n, seed)
		min := graph.Rect.Min
		max := graph.Rect.Max

		totalPixels := max.X * max.Y

		var i int
		for x := min.X; x < max.X; x++ {
			for y := min.Y; y < max.Y; y++ {
				l := p.Noise2D(float64(x)/xDivide, float64(y)/zDivide) * intensify

				switch colorMode {
				case "CMYK":
					r := uint64(l*1000) + (math.Float64bits(l)>>52)*1000
					z := uint32(r) + uint32(r>>32)

					graph.Set(x, y, *(*color.CMYK)(unsafe.Pointer(&z)))
				case "Gray":
					graph.Set(x, y, color.Gray{Y: uint8(l * 255)})
				case "Gray-16":
					graph.Set(x, y, color.Gray16{Y: uint16(l * 65535)})
				case "NRGBA":
					graph.Set(x, y, color.NRGBA{A: uint8(l * 255)})
				case "NRGBA64":
					graph.Set(x, y, color.NRGBA64{A: uint16(l * 65535)})
				case "NYCbCrA":
					r := uint64(l*1000) + (math.Float64bits(l)>>52)*1000
					z := uint32(r) + uint32(r>>32)
					c := *(*color.NYCbCrA)(unsafe.Pointer(&z))
					c.A = 255

					graph.Set(x, y, c)
				case "RGBA":
					graph.Set(x, y, color.RGBA{A: uint8(l * 255)})
				case "RGBA64":
					graph.Set(x, y, color.RGBA64{A: uint16(l * 65535)})
				case "YCbCr":
					r := uint64(l*1000) + (math.Float64bits(l)>>52)*1000
					z := uint32(r) + uint32(r>>32)

					graph.Set(x, y, *(*color.YCbCr)(unsafe.Pointer(&z)))
				}
				if individualRefresh {
					img.Refresh()
				}
				i++
				atomic.StoreInt32(&rtc, int32(i*100/totalPixels))
			}
		}

		if !individualRefresh {
			img.Refresh()
		}

		timeTaken := time.Since(t)
		pixelPerSecond.SetText(fmt.Sprintf("%d pps", int(float64(totalPixels)/timeTaken.Seconds())))
	})

	go func() {
		var oldRtc int32
		for {
			r := atomic.LoadInt32(&rtc)
			if r == oldRtc {
				continue
			}
			rendering.SetText(fmt.Sprintf("Rendering... %d%%", r))
			oldRtc = r
		}
	}()

	var colorModeSelect = widget.NewSelect([]string{"CMYK", "Gray", "Gray-16", "NRGBA", "NRGBA64", "NYCbCrA", "RGBA", "RGBA64", "YCbCr"}, func(s string) {
		colorMode = s
	})
	colorModeSelect.SetSelected(colorMode)
	cM := container.NewBorder(nil, nil, widget.NewLabel("Color"), nil, colorModeSelect)
	rM := widget.NewCheck("Individual Refresh", func(b bool) {
		individualRefresh = b
	})

	var xDivideInput = widget.NewEntry()
	xDivideInput.SetText(strconv.FormatFloat(xDivide, 'f', 2, 64))
	xDivideInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		xDivide = i
	}

	var zDivideInput = widget.NewEntry()
	zDivideInput.SetText(strconv.FormatFloat(zDivide, 'f', 2, 64))
	zDivideInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		zDivide = i
	}

	var intensifyInput = widget.NewEntry()
	intensifyInput.SetText(strconv.FormatFloat(intensify, 'f', 2, 64))
	intensifyInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		intensify = i
	}

	topBottom := container.NewHBox(
		cM,
		container.NewGridWithColumns(2, widget.NewLabel("X-div"), xDivideInput),
		container.NewGridWithColumns(2, widget.NewLabel("Z-div"), zDivideInput),
		container.NewGridWithColumns(2, widget.NewLabel("Intensity"), intensifyInput),
		rM,
		widget.NewButton("Resize", func() {
			var newWidth = graph.Rect.Dx()
			var newHeight = graph.Rect.Dy()

			var widthInput = widget.NewEntry()
			widthInput.SetText(strconv.FormatInt(int64(newWidth), 10))
			widthInput.OnChanged = func(s string) {
				i, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return
				}
				newWidth = int(i)
			}

			var heightInput = widget.NewEntry()
			heightInput.SetText(strconv.FormatInt(int64(newHeight), 10))
			heightInput.OnChanged = func(s string) {
				i, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return
				}
				newHeight = int(i)
			}

			width := container.NewGridWithColumns(2, widget.NewLabel("Width"), widthInput)
			height := container.NewGridWithColumns(2, widget.NewLabel("Height"), heightInput)

			dialog := dialog.NewCustomWithoutButtons("Resize", container.NewCenter(container.NewVBox(width, height)), w)
			dialog.SetButtons([]fyne.CanvasObject{
				widget.NewButton("Resize", func() {
					whiteBackground.Image = newWhiteBackground(newWidth, newHeight)
					whiteBackground.Refresh()

					graph = image.NewNRGBA64(image.Rect(0, 0, newWidth, newHeight))
					img.Image = graph
					img.Refresh()
					dialog.Hide()
				}),
			})

			dialog.Show()
		}),
		widget.NewButton("Save As", func() {
			p, _ := dialog2.File().Filter("PNG (.png)", "png").Filter("JPEG (.jpg/.jpeg/.jfif)", ".jpg", ".jpeg", ".jfif").Save()
			extI := strings.LastIndex(p, ".")
			var ext string
			if extI != -1 {
				ext = p[extI:]
			}

			file, _ := os.Create(p)

			switch ext {
			case "jpeg", "jpg", "jfif":
				jpeg.Encode(file, graph, nil)
			default:
				png.Encode(file, graph)
			}

			file.Close()
		}),
	)

	var top = container.NewBorder(nil, topBottom, nil, resetButton, container.NewGridWithColumns(4,
		container.NewBorder(nil, nil, widget.NewLabel("Alpha"), nil, alphaInput),
		container.NewBorder(nil, nil, widget.NewLabel("Beta"), nil, betaInput),
		container.NewBorder(nil, nil, widget.NewLabel("Iterations"), nil, iterationsInput),
		container.NewBorder(nil, nil, widget.NewLabel("Seed"), nil, seedInput),
	))

	return container.NewBorder(top, container.NewHBox(pixelPerSecond, layout.NewSpacer(), rendering), nil, nil, container.NewStack(whiteBackground, img))
}
