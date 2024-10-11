package main

import (
	"crypto/rand"
	"fmt"
	"image/color"
	"math"
	r2 "math/rand"
	"strings"
	"unsafe"

	"github.com/Knetic/govaluate"
	"github.com/aquilax/go-perlin"
)

type Graph func(x, y float64) (x1, y1 []float64)

var precision = 0.01

var graphs = make(map[color.Color][]Graph)

type pc1 struct {
	a, b, x float64
	n       int32
	seed    int64
}

var perlinCache = map[pc1]float64{}

func parseMultiequationGraph(str string) (Graph, error) {
	z, err := parseMultiequationExpressions(str)
	if err != nil {
		return nil, err
	}

	return func(x, y float64) (x1, y1 []float64) {
		x1, y1 = make([]float64, len(z[0])), make([]float64, len(z[1]))

		params := map[string]interface{}{
			"x":     x,
			"y":     y,
			"π":     math.Pi,
			"e":     math.E,
			"max64": math.MaxFloat64,
			"min64": math.SmallestNonzeroFloat64,
		}

		for i, q := range z[0] {
			v, _ := q.Evaluate(params)
			x1[i], _ = v.(float64)
		}

		for i, q := range z[1] {
			v, _ := q.Evaluate(params)
			y1[i], _ = v.(float64)
		}

		return
	}, nil
}

var functions = map[string]govaluate.ExpressionFunction{
	"sqrt": newFloat64Func(math.Sqrt),
	"abs":  newFloat64Func(math.Abs),
	"rnd": func(arguments ...interface{}) (interface{}, error) {
		return r2.Float64(), nil
	},
	"acos":  newFloat64Func(math.Acos),
	"acosh": newFloat64Func(math.Acosh),
	"asin":  newFloat64Func(math.Asin),
	"asinh": newFloat64Func(math.Asinh),
	"atan":  newFloat64Func(math.Atan),
	"atanh": newFloat64Func(math.Atanh),
	"cbrt":  newFloat64Func(math.Cbrt),
	"ceil":  newFloat64Func(math.Ceil),
	"cos":   newFloat64Func(math.Cos),
	"cosh":  newFloat64Func(math.Cosh),
	"floor": newFloat64Func(math.Floor),
	"sin":   newFloat64Func(math.Sin),
	"sinh":  newFloat64Func(math.Sinh),
	"tan":   newFloat64Func(math.Tan),
	"tanh":  newFloat64Func(math.Tanh),
	"p1": func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 5 {
			return 0, fmt.Errorf("must have 5 arguments: alpha, beta, n, seed, x")
		}
		pc := pc1{arguments[0].(float64), arguments[1].(float64), arguments[4].(float64), int32(arguments[2].(float64)), int64(arguments[3].(float64))}
		v, ok := perlinCache[pc]
		if ok {
			return v, nil
		}

		p := perlin.NewPerlin(arguments[0].(float64), arguments[1].(float64), int32(arguments[2].(float64)), int64(arguments[3].(float64)))
		v = p.Noise1D(arguments[4].(float64))
		perlinCache[pc] = v

		return v, nil
	},
	"p2": func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 6 {
			return 0, fmt.Errorf("must have 6 arguments: alpha, beta, n, seed, x, y")
		}
		p := perlin.NewPerlin(arguments[0].(float64), arguments[1].(float64), int32(arguments[2].(float64)), int64(arguments[3].(float64)))

		return p.Noise2D(arguments[4].(float64), arguments[5].(float64)), nil
	},
	"p3": func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 7 {
			return 0, fmt.Errorf("must have 7 arguments: alpha, beta, n, seed, x, y, z")
		}
		p := perlin.NewPerlin(arguments[0].(float64), arguments[1].(float64), int32(arguments[2].(float64)), int64(arguments[3].(float64)))

		return p.Noise3D(arguments[4].(float64), arguments[5].(float64), arguments[6].(float64)), nil
	},
	"min":       new2Float64Func(math.Min),
	"max":       new2Float64Func(math.Max),
	"atan2":     new2Float64Func(math.Atan2),
	"dim":       new2Float64Func(math.Dim),
	"mod":       new2Float64Func(math.Mod),
	"remainder": new2Float64Func(math.Remainder),
	"copysign":  new2Float64Func(math.Copysign),
	"hypot":     new2Float64Func(math.Hypot),
}

func new2Float64Func(g func(float64, float64) float64) govaluate.ExpressionFunction {
	return func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 2 {
			return nil, fmt.Errorf("expected x,y")
		}
		x, _ := arguments[0].(float64)
		y, _ := arguments[1].(float64)

		return g(x, y), nil
	}
}

func newFloat64Func(g func(float64) float64) govaluate.ExpressionFunction {
	return func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 1 {
			return nil, fmt.Errorf("expected an argument")
		}
		v, _ := arguments[0].(float64)

		return g(v), nil
	}
}

func parseMultiequationExpressions(str string) ([2][]*govaluate.EvaluableExpression, error) {
	s := parseMultiequation(str)

	var eqs = [2][]*govaluate.EvaluableExpression{
		make([]*govaluate.EvaluableExpression, len(s[0])),
		make([]*govaluate.EvaluableExpression, len(s[1])),
	}

	var err error
	for i, eq := range s[0] {
		eqs[0][i], err = govaluate.NewEvaluableExpressionWithFunctions(eq, functions)
		if err != nil {
			return eqs, err
		}
	}
	for i, eq := range s[1] {
		eqs[1][i], err = govaluate.NewEvaluableExpressionWithFunctions(eq, functions)
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

	var strs0 []string
	var strs1 []string
	currentString = ""
	open = false

	if len(strs[0]) != 0 {
		for _, char := range strs[0][0] {
			switch char {
			case ',':
				if open {
					currentString += string(char)
					continue
				}
				strs0 = append(strs0, currentString)
				currentString = ""
				open = false
			case '(', ')':
				open = char == '('
				fallthrough
			default:
				currentString += string(char)
			}
		}
		strs0 = append(strs0, currentString)
		currentString = ""
		open = false
	}

	if len(strs[1]) != 0 {
		for _, char := range strs[1][0] {
			switch char {
			case ',':
				if open {
					currentString += string(char)
					continue
				}
				strs1 = append(strs1, currentString)
				currentString = ""
				open = false
			case '(', ')':
				open = char == '('
				fallthrough
			default:
				currentString += string(char)
			}
		}
		strs1 = append(strs1, currentString)
	}
	strs[0], strs[1] = strs0, strs1

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
				setpix(dx+x1, dy-y1, c)
			}
		}
	}
}

func setpix(x, y float64, c color.Color) {
	graph.Set(int(x), int(y), c)

	graph.Set(int(x+1), int(y), c)
	graph.Set(int(x), int(y+1), c)

	graph.Set(int(x+2), int(y), c)
	graph.Set(int(x), int(y+2), c)
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
	maxX, maxY := float64(graph.Rect.Max.X), float64(graph.Rect.Max.Y)

	dx, dy := maxX/2, maxY/2

	for x, y := -maxX, -maxY; x < maxX && y < maxY; x, y = x+precision, y+precision {
		for c, graphs := range graphs {
			for _, g := range graphs {
				xs, ys := g(x, y)

				for _, x1 := range xs {
					for _, y1 := range ys {
						setpix(dx+x1, dy-y1, c)
					}
				}
			}
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
