package main

import (
	"crypto/rand"
	"image/color"
	"unsafe"
)

type Graph func(x, y float64) (x1, y1 float64)

var precision = 0.01

var graphs = make(map[color.Color][]Graph)

func applyGraph(f Graph, c color.Color) {
	if c == nil {
		c = colorrand()
	}
	maxX, maxY := float64(graph.Rect.Max.X), float64(graph.Rect.Max.Y)

	dx, dy := maxX/2, maxY/2

	for x, y := -maxX, -maxY; x < maxX && y < maxY; x, y = x+precision, y+precision {
		x1, y1 := f(x, y)

		graph.Set(int(dx+x1), int(dy-y1), c)
	}
}

func colorrand() color.Color {
	var color = color.RGBA{A: 1}
	for {
		rand.Read(unsafe.Slice((*byte)(unsafe.Pointer(&color)), 3))
		if _, ok := graphs[color]; !ok {
			break
		}
	}

	return color
}

func addGraph(f Graph, c color.Color) {
	applyGraph(f, c)
	graphs[c] = append(graphs[c], f)
}

func reset() {
	clear(graph.Pix)
	for c, graphs := range graphs {
		for _, g := range graphs {
			applyGraph(g, c)
		}
	}
}

func quadratic(a, b, c float64) func(x, y float64) (x1, y1 float64) {
	return func(x, y float64) (x1 float64, y1 float64) {
		return x, a*x*x + b*x + c
	}
}
func linear(m, b float64) func(x, y float64) (x1, y1 float64) {
	return func(x, y float64) (x1 float64, y1 float64) {
		return x, m*x + b
	}
}
func constantX(cx float64) func(x, y float64) (x1, y1 float64) {
	return func(x, y float64) (x1 float64, y1 float64) {
		return cx, y
	}
}
func constantY(cy float64) func(x, y float64) (x1, y1 float64) {
	return func(x, y float64) (x1 float64, y1 float64) {
		return x, cy
	}
}
func constant(cx, cy float64) func(x, y float64) (x1, y1 float64) {
	return func(x, y float64) (x1 float64, y1 float64) {
		return cx, cy
	}
}
