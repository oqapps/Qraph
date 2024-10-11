package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"slices"
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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/aquilax/go-perlin"
	dialog2 "github.com/sqweek/dialog"
	"github.com/yeqown/go-qrcode/v2"
)

var graph = image.NewRGBA64(image.Rect(0, 0, 1200, 1200))

var p = perlin.NewPerlin(2, 2, 1, 39530)

func newWhiteBackground(w, h int) *image.Gray16 {
	var whiteBackground = image.NewGray16(image.Rect(0, 0, w, h))
	min := whiteBackground.Rect.Min
	max := whiteBackground.Rect.Max

	for x := min.X; x < max.X; x++ {
		for y := min.Y; y < max.Y; y++ {
			whiteBackground.Set(x, y, color.White)
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
			container.NewAppTabs(
				container.NewTabItem("Equations", equationsPage()),
				container.NewTabItem("Noise", perlinPage(w)),
				container.NewTabItem("QR-Code", qrPage()),
			),
		))
	w.Resize(fyne.NewSize(400, 600))
	w.ShowAndRun()
}

func equationsPage() fyne.CanvasObject {
	img := canvas.NewImageFromImage(graph)
	img.ScaleMode = canvas.ImageScaleSmooth

	precisionInput := widget.NewEntry()
	precisionInput.OnChanged = func(s string) {
		i, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		precision = i
	}
	precisionInput.SetText(strconv.FormatFloat(precision, 'f', 2, 64))

	renderingText := widget.NewLabel("Rendering...")
	renderingText.Hide()

	addGraph(constantX(0), color.White)
	addGraph(constantY(0), color.White)

	eqList := container.NewAdaptiveGrid(4)

	return container.NewBorder(container.NewVBox(container.NewHBox(widget.NewLabel("Precision"), precisionInput, widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		color := colorrand()

		entry := widget.NewEntry()

		entry.OnSubmitted = func(s string) {
			g, err := parseMultiequationGraph(s)
			if err != nil {
				return
			}

			graphs[color] = []Graph{g}
			renderingText.Show()
			reset()
			img.Refresh()
			renderingText.Hide()
		}

		circle := canvas.NewRectangle(color)
		circle.CornerRadius = 17
		circle.SetMinSize(fyne.NewSquareSize(entry.MinSize().Height))

		deleteButton := &widget.Button{
			Icon:       theme.ContentRemoveIcon(),
			Importance: widget.DangerImportance,
			OnTapped: func() {
				i := colorIndexOf(eqList, color)
				eqList.Objects = slices.Delete(eqList.Objects, i, i+1)
				eqList.Refresh()

				delete(graphs, color)
				reset()
				img.Refresh()
			},
		}

		eqList.Add(container.NewBorder(nil, nil, circle, deleteButton, entry))
	})), eqList), container.NewHBox(layout.NewSpacer(), renderingText), nil, nil, img)
}

func colorIndexOf(cont *fyne.Container, c color.Color) int {
	return slices.IndexFunc(cont.Objects, func(w fyne.CanvasObject) bool {
		return w.(*fyne.Container).Objects[1].(*canvas.Rectangle).FillColor == c
	})
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
	var renderProgress = true
	var useWhiteBackground = true

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

	var codeBlock = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

	var rtc int32

	var resetButton = widget.NewButton("Render", func() {
		t := time.Now()
		p := perlin.NewPerlin(a, b, n, seed)
		min := graph.Rect.Min
		max := graph.Rect.Max

		totalPixels := max.X * max.Y

		codeBlock.SetText(genCode(a, b, n, seed, xDivide, zDivide, intensify, max.X, max.Y))

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
				if renderProgress {
					atomic.StoreInt32(&rtc, int32(i*100/totalPixels))
				}
			}
		}

		if !individualRefresh {
			img.Refresh()
		}

		timeTaken := time.Since(t)

		pixelPerSecond.SetText(fmt.Sprintf("%d px/s (%ds)", int(float64(totalPixels)/math.Max(1, math.Ceil(timeTaken.Seconds()))), int(timeTaken.Seconds())))
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
	rM := widget.NewCheck("Real time reload", func(b bool) {
		individualRefresh = b
	})
	pM := widget.NewCheck("Render progress", func(b bool) {
		renderProgress = b
	})
	wB := widget.NewCheck("White background", func(b bool) {
		useWhiteBackground = b
		if !b {
			whiteBackground.Hide()
		} else {
			if graph.Rect.Max != whiteBackground.Image.Bounds().Max {
				whiteBackground.Image = newWhiteBackground(graph.Rect.Dx(), graph.Rect.Dy())
			}
			whiteBackground.Show()
		}
	})
	wB.SetChecked(useWhiteBackground)
	pM.SetChecked(renderProgress)

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
		container.NewGridWithColumns(3, widget.NewLabel("Focus"), xDivideInput, zDivideInput),
		container.NewGridWithColumns(2, widget.NewLabel("Intensity"), intensifyInput),
		wB,
		rM,
		pM,
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
					if useWhiteBackground {
						whiteBackground.Image = newWhiteBackground(newWidth, newHeight)
						whiteBackground.Refresh()
					}

					graph = image.NewNRGBA64(image.Rect(0, 0, newWidth, newHeight))
					img.Image = graph
					img.Refresh()
					dialog.Hide()
				}),
			})

			dialog.Show()
		}),
		widget.NewButton("Export", func() {
			p, err := dialog2.File().Filter("PNG (.png)", "png").Filter("JPEG (.jpg/.jpeg/.jfif)", ".jpg", ".jpeg", ".jfif").Save()
			if err != nil {
				return
			}
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

	return container.NewBorder(top, container.NewHBox(container.NewBorder(nil, pixelPerSecond, nil, nil,
		container.NewBorder(widget.NewLabel("Code: "), nil, nil, nil, codeBlock),
	), layout.NewSpacer(), container.NewVBox(layout.NewSpacer(), rendering)), nil, nil, container.NewStack(whiteBackground, img))
}

func genCode(a, b float64, i int32, seed int64, xd, zd, is float64, w, h int) string {
	return fmt.Sprintf("import \"github.com/aquilax/go-perlin\"\n\nvar p = perlin.NewPerlin(%f, %f, %d, %d)\nfor x := 0; x < %d; x++ {\n\tfor y := 0; y < %d; y++ {\n\t\tvar value = p.Perlin2D(x/%f, y/%f)*%f\n\t}\n}", a, b, i, seed, w, h, xd, zd, is)
}

func qrPage() fyne.CanvasObject {
	textEntry := widget.NewEntry()
	genButton := widget.NewButton("Generate", nil)
	img := canvas.NewImageFromImage(nil)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScalePixels

	saveButton := widget.NewButton("Export", func() {
		p, err := dialog2.File().Filter("PNG (.png)", "png").Filter("JPEG (.jpg/.jpeg/.jfif)", ".jpg", ".jpeg", ".jfif").Save()
		if err != nil {
			return
		}
		extI := strings.LastIndex(p, ".")
		var ext string
		if extI != -1 {
			ext = p[extI:]
		}

		file, _ := os.Create(p)

		switch ext {
		case "jpeg", "jpg", "jfif":
			jpeg.Encode(file, img.Image, nil)
		default:
			png.Encode(file, img.Image)
		}

		file.Close()
	})

	top := container.NewBorder(nil, container.NewHBox(layout.NewSpacer(), saveButton), widget.NewLabel("Text:"), genButton, textEntry)

	var imgw = imageWriter{
		img,
	}

	genButton.OnTapped = func() {
		c, _ := qrcode.New(textEntry.Text)

		c.Save(imgw)
	}

	return container.NewBorder(top, nil, nil, nil, img)
}

type imageWriter struct {
	img *canvas.Image
}

func (i imageWriter) Write(mat qrcode.Matrix) error {
	image := image.NewRGBA(image.Rect(0, 0, mat.Width(), mat.Height()))
	i.img.Image = image

	mat.Iterate(qrcode.IterDirection_ROW, func(x, y int, s qrcode.QRValue) {
		var c = color.White
		if s.IsSet() {
			c = color.Black
		}

		image.Set(x, y, c)
	})

	i.img.Refresh()

	return nil
}

func (imageWriter) Close() error {
	return nil
}
