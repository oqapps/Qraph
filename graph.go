package main

import (
	"crypto/rand"
	"image/color"
	"math"
	"strings"
	"unsafe"

	"github.com/Knetic/govaluate"
)

type Graph func(x, y float64) (x1, y1 []float64)

var precision = 0.01

var graphs = make(map[color.Color][]Graph)

func parseMultiequationGraph(str string) (Graph, error) {
	z, err := parseMultiequationExpressions(str)
	if err != nil {
		return nil, err
	}

	return func(x, y float64) (x1, y1 []float64) {
		x1, y1 = make([]float64, len(z[0])), make([]float64, len(z[1]))

		for i, q := range z[0] {
			v, _ := q.Evaluate(map[string]interface{}{
				"x": x,
				"y": y,
			})
			x1[i], _ = v.(float64)
		}

		for i, q := range z[1] {
			v, _ := q.Evaluate(map[string]interface{}{
				"x": x,
				"y": y,
			})
			y1[i], _ = v.(float64)
		}

		return
	}, nil
}

func parseMultiequationExpressions(str string) ([2][]*govaluate.EvaluableExpression, error) {
	s := parseMultiequation(str)

	var eqs = [2][]*govaluate.EvaluableExpression{
		make([]*govaluate.EvaluableExpression, len(s[0])),
		make([]*govaluate.EvaluableExpression, len(s[1])),
	}

	var err error
	for i, eq := range s[0] {
		eqs[0][i], err = govaluate.NewEvaluableExpression(eq)
		if err != nil {
			return eqs, err
		}
	}
	for i, eq := range s[1] {
		eqs[1][i], err = govaluate.NewEvaluableExpression(eq)
		if err != nil {
			return eqs, err
		}
	}

	return eqs, nil
}

func parseMultiequation(str string) [2][]string {
	str = strings.TrimSpace(str)

	if i := strings.Index(str, "y="); i == 0 && i+2 < len(str) {
		return [2][]string{{"x"}, {str[i+2:]}}
	}
	if i := strings.Index(str, "x="); i == 0 && i+2 < len(str) {
		return [2][]string{{str[i+2:]}, {"y"}}
	}

	var open bool
	var currentString string

	var strs [2][]string

	var stringNumero int

	for _, char := range str {
		switch char {
		case '{':
			if open {
				currentString += string(char)
				continue
			}
			open = true
		case ',':
			if open {
				currentString += string(char)
				continue
			}
			open = true
			fallthrough
		case '}':
			if !open {
				currentString += string(char)
				continue
			}
			if stringNumero == len(strs) {
				return [2][]string{} //ERR
			}
			strs[stringNumero] = append(strs[stringNumero], currentString)
			open = false
			currentString = ""
			stringNumero++
		case ' ':
			continue
		default:
			currentString += string(char)
		}
	}

	if currentString != "" {
		if stringNumero == len(strs) {
			goto s
		}
		strs[stringNumero] = append(strs[stringNumero], currentString)
	}

s:

	if len(strs[0]) != 0 {
		strs[0] = strings.Split(strs[0][0], ",")
	}
	if len(strs[1]) != 0 {
		strs[1] = strings.Split(strs[1][0], ",")
	}

	return strs
}

func applyGraph(f Graph, c color.Color) {
	if c == nil {
		c = colorrand()
	}
	maxX, maxY := float64(graph.Rect.Max.X), float64(graph.Rect.Max.Y)

	dx, dy := maxX/2, maxY/2

	for x, y := -maxX, -maxY; x < maxX && y < maxY; x, y = x+precision, y+precision {
		xs, ys := f(x, y)

		for _, x1 := range xs {
			for _, y1 := range ys {
				graph.Set(int(dx+x1), int(dy-y1), c)
			}
		}
	}
}

func colorrand() color.Color {
	var color = color.RGBA{A: 255}
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

func oneXandOneY(x, y float64) ([]float64, []float64) {
	return []float64{x}, []float64{y}
}

// cubic function is defined as ax^3+bx^2+cx+d
func cubic(a, b, c, d float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, a*math.Pow(x, 3)+b*math.Pow(x, 2)+c*x+d)
	}
}

// a quartic function is defined as ax^4+bx^3+cx^2+dx+e
func quartic(a, b, c, d, e float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, a*math.Pow(x, 4)+b*math.Pow(x, 3)+c*math.Pow(x, 2)+d*x+e)
	}
}

// an exponential function is defined as e^x
func exponential() Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, math.Pow(math.E, x))
	}
}

// a quadratic function is defined as y=ax^2+bx+c
func quadratic(a, b, c float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, a*x*x+b*x+c)
	}
}

// a linear function is defined as y=mx+b
func linear(m, b float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, m*x+b)
	}
}

// constant x point
func constantX(cx float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(cx, y)
	}
}

// constant y point
func constantY(cy float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(x, cy)
	}
}

// constant point
func constant(cx, cy float64) Graph {
	return func(x, y float64) (x1, y1 []float64) {
		return oneXandOneY(cx, cy)
	}
}

// a circle function is defined as (x-h)^2 + (y-k)^2 = r^2
func circle(h, k, r float64) Graph {
	//(x-h)^2 + (y-k)^2 = r^2
	//(x-h)^2-r^2 = -(y-k)^2
	// r^2-(x-h)^2 = (y-k)^
	// sqrt(r^2-(x-h)^2) = y-k
	// sqrt(r^2-(x-h)^2)+k = y

	return func(x, y float64) (x1, y1 []float64) {
		d := math.Sqrt(math.Pow(r, 2) - math.Pow(x-h, 2))
		return []float64{x}, []float64{k + d, k - d}
	}
}
