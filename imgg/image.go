package imgg

import (
	"code.google.com/p/draw2d/draw2d"
	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/truetype"
	"code.google.com/p/graphics-go/graphics"
	"fmt"
	"github.com/vdobler/chart"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"math"
)

// ImageGraphics implements BasicGraphics and uses the generic implementations
type ImageGraphics struct {
	Image  *image.RGBA
	x0, y0 int
	w, h   int
	bg     color.RGBA
	gc     draw2d.GraphicContext
}

// New creates a new ImageGraphics of dimension w x h.
func New(width, height int, background color.RGBA) *ImageGraphics {
	_ = fmt.Sprintf("")
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	gc := draw2d.NewGraphicContext(img)
	gc.SetLineJoin(draw2d.MiterJoin)
	gc.SetLineCap(draw2d.SquareCap)
	gc.SetStrokeColor(image.Black)
	gc.SetFillColor(background)
	gc.Clear()
	return &ImageGraphics{Image: img, x0: 0, y0: 0, w: width, h: height,
		bg: background, gc: gc}
}

// AddTo returns a new ImageGraphics which will write to (width x height) sized
// area starting at (x,y) on the provided image img.
func AddTo(img *image.RGBA, x, y, width, height int, background color.RGBA) *ImageGraphics {
	gc := draw2d.NewGraphicContext(img)
	gc.SetStrokeColor(image.Black)
	gc.SetFillColor(background)
	gc.Translate(float64(x), float64(y))
	gc.ClearRect(x, y, x+width, y+height)

	return &ImageGraphics{Image: img, x0: x, y0: y, w: width, h: height, bg: background, gc: gc}
}

func (ig *ImageGraphics) Begin()                              {}
func (ig *ImageGraphics) End()                                {}
func (ig *ImageGraphics) Background() (r, g, b, a uint8)      { return ig.bg.R, ig.bg.G, ig.bg.B, ig.bg.A }
func (ig *ImageGraphics) Dimensions() (int, int)              { return ig.w, ig.h }
func (ig *ImageGraphics) fontheight(font chart.Font) (fh int) { return 15 }
func (ig *ImageGraphics) FontMetrics(font chart.Font) (fw float32, fh int, mono bool) {
	return 8, 15, true
}
func (ig *ImageGraphics) TextLen(t string, font chart.Font) int {
	return chart.GenericTextLen(ig, t, font)
}

func (ig *ImageGraphics) setStyle(style chart.Style) {
	ig.gc.SetStrokeColor(chart.Color2RGBA(style.LineColor, 0xff))
	ig.gc.SetLineWidth(float64(style.LineWidth))
	ig.gc.SetLineDash(dashPattern[style.LineStyle], 0)
}

func (ig *ImageGraphics) Line(x0, y0, x1, y1 int, style chart.Style) {
	if style.LineWidth <= 0 {
		style.LineWidth = 1
	}
	ig.setStyle(style)
	ig.gc.MoveTo(float64(x0)+0.5, float64(y0)+0.5)
	ig.gc.LineTo(float64(x1)+0.5, float64(y1)+0.5)
	ig.gc.Stroke()
}

var dashPattern map[int][]float64 = map[int][]float64{
	chart.SolidLine:      nil, // []float64{10},
	chart.DashedLine:     []float64{50, 20},
	chart.DottedLine:     []float64{20, 20},
	chart.DashDotDotLine: []float64{50, 20, 20, 20, 20, 20},
	chart.LongDashLine:   []float64{50, 50},
	chart.LongDotLine:    []float64{20, 50},
}

func (ig *ImageGraphics) Path(x, y []int, style chart.Style) {
	ig.setStyle(style)
	ig.gc.MoveTo(float64(x[0]), float64(y[0]))
	for i := 1; i < len(x); i++ {
		ig.gc.LineTo(float64(x[i]), float64(y[i]))
	}
	ig.gc.Stroke()
}

func (ig *ImageGraphics) Text(x, y int, t string, align string, rot int, f chart.Font) {
	if len(align) == 1 {
		align = "c" + align
	}
	// fw, fh, _ := ig.FontMetrics(f)
	//fmt.Printf("Text '%s' at (%d,%d) %s\n", t, x,y, align)
	// TODO: handle rot

	size := 12
	if f.Size < 0 {
		size = 10
	} else if f.Size > 0 {
		size = 14
	}
	textImage := textBox(t, size)
	bounds := textImage.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	var centerX, centerY int

	if rot != 0 {
		alpha := float64(rot) / 180 * math.Pi
		cos := math.Cos(alpha)
		sin := math.Sin(alpha)
		hs, hc := float64(h)*sin, float64(h)*cos
		ws, wc := float64(w)*sin, float64(w)*cos
		W := int(math.Ceil(hs + wc))
		H := int(math.Ceil(hc + ws))
		rotated := image.NewAlpha(image.Rect(0, 0, W, H))
		graphics.Rotate(rotated, textImage, &graphics.RotateOptions{-alpha})
		textImage = rotated
		centerX, centerY = W/2, H/2

		switch align {
		case "bl":
			centerX, centerY = int(hs), H
		case "bc":
			centerX, centerY = W-int(wc/2), int(ws/2)
		case "br":
			centerX, centerY = W, int(hc)
		case "tl":
			centerX, centerY = 0, H-int(hc)
		case "tc":
			centerX, centerY = int(ws/2), H-int(ws/2)
		case "tr":
			centerX, centerY = W-int(hs), 0
		case "cl":
			centerX, centerY = int(hs/2), H-int(hc/2)
		case "cr":
			centerX, centerY = W-int(hs/2), int(hc/2)
		}
	} else {
		centerX, centerY = w/2, h/2
		switch align[0] {
		case 'b':
			centerY = h
		case 't':
			centerY = 0
		}
		switch align[1] {
		case 'l':
			centerX = 0
		case 'r':
			centerX = w
		}
	}

	bounds = textImage.Bounds()
	w, h = bounds.Dx(), bounds.Dy()
	x -= centerX
	y -= centerY
	x += ig.x0
	y += ig.y0

	col := "#000000"
	if f.Color != "" {
		col = f.Color
	}
	r, g, b := chart.Color2rgb(col)
	tcol := image.NewUniform(color.RGBA{uint8(r), uint8(g), uint8(b), 255})

	draw.DrawMask(ig.Image, image.Rect(x, y, x+w, y+h), tcol, image.ZP,
		textImage, textImage.Bounds().Min, draw.Over)
}

// textBox renders t into a tight fitting image
func textBox(t string, size int) image.Image {
	// Initialize the context.
	fg := image.NewUniform(color.Alpha{0xff})
	bg := image.NewUniform(color.Alpha{0x00})
	canvas := image.NewAlpha(image.Rect(0, 0, 400, 40))
	draw.Draw(canvas, canvas.Bounds(), bg, image.ZP, draw.Src)

	c := freetype.NewContext()
	c.SetDPI(dpi)
	c.SetFont(theFont)
	c.SetFontSize(float64(size))
	c.SetClip(canvas.Bounds())
	c.SetDst(canvas)
	c.SetSrc(fg)

	// Draw the text.
	h := c.FUnitToPixelRU(theFont.UnitsPerEm())
	pt := freetype.Pt(0, h)
	extent, err := c.DrawString(t, pt)
	if err != nil {
		log.Println(err)
		return nil
	}
	// log.Printf("text %q, extent: %v", t, extent)
	return canvas.SubImage(image.Rect(0, 0, int(extent.X/256), h*5/4))
}

func (ig *ImageGraphics) paint(x, y int, R, G, B uint32, alpha uint32) {
	r, g, b, a := ig.Image.At(x, y).RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	a >>= 8
	r *= alpha
	g *= alpha
	b *= alpha
	a *= alpha
	r += R * (0xff - alpha)
	g += G * (0xff - alpha)
	b += B * (0xff - alpha)
	r >>= 8
	g >>= 8
	b >>= 8
	a >>= 8
	ig.Image.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
}

func (ig *ImageGraphics) Symbol(x, y int, style chart.Style) {
	chart.GenericSymbol(ig, x, y, style)
}

func (ig *ImageGraphics) Rect(x, y, w, h int, style chart.Style) {
	ig.setStyle(style)
	stroke := func() { ig.gc.Stroke() }
	if style.FillColor != "" {
		ig.gc.SetFillColor(chart.Color2RGBA(style.FillColor, 0x0ff /*uint8(style.Alpha*255)*/))
		stroke = func() { ig.gc.FillStroke() }
	}
	ig.gc.MoveTo(float64(x), float64(y))
	ig.gc.LineTo(float64(x+w), float64(y))
	ig.gc.LineTo(float64(x+w), float64(y+h))
	ig.gc.LineTo(float64(x), float64(y+h))
	ig.gc.LineTo(float64(x), float64(y))
	stroke()
}

func (ig *ImageGraphics) Wedge(ix, iy, iro, iri int, phi, psi float64, style chart.Style) {
	ig.setStyle(style)
	stroke := func() { ig.gc.Stroke() }
	if style.FillColor != "" {
		ig.gc.SetFillColor(chart.Color2RGBA(style.FillColor, 0x0ff /*uint8(style.Alpha*255)*/))
		stroke = func() { ig.gc.FillStroke() }
	}

	iri = 0
	ecc := 1.0                           // eccentricity
	x, y := float64(ix), float64(iy)     // center as float
	ro, ri := float64(iro), float64(iri) // radius outer and inner as float
	roe, rie := ro*ecc, ri*ecc           // inner and outer radius corrected by ecc

	xao, yao := math.Cos(phi)*roe+x, y+math.Sin(phi)*ro
	// xco, yco := math.Cos(psi)*roe+x, y-math.Sin(psi)*ro
	xai, yai := math.Cos(phi)*rie+x, y+math.Sin(phi)*ri
	xci, yci := math.Cos(psi)*rie+x, y+math.Sin(psi)*ri

	// outbound straight line
	if ri > 0 {
		ig.gc.MoveTo(xai, yai)
	} else {
		ig.gc.MoveTo(x, y)
	}
	ig.gc.LineTo(xao, yao)

	// outer arc
	ig.gc.ArcTo(x, y, ro, roe, phi, psi-phi)

	// inbound straight line
	if ri > 0 {
		ig.gc.LineTo(xci, yci)
		ig.gc.ArcTo(x, y, ri, rie, psi, phi-psi)
	} else {
		ig.gc.LineTo(x, y)
	}
	stroke()
}

func (ig *ImageGraphics) Title(text string) {
	font := chart.DefaultFont["title"]
	_, fh, _ := ig.FontMetrics(font)
	x, y := ig.w/2, fh/2
	ig.Text(x, y, text, "tc", 0, font)
}

func (ig *ImageGraphics) XAxis(xr chart.Range, ys, yms int) {
	chart.GenericXAxis(ig, xr, ys, yms)
}
func (ig *ImageGraphics) YAxis(yr chart.Range, xs, xms int) {
	chart.GenericYAxis(ig, yr, xs, xms)
}

func (ig *ImageGraphics) Scatter(points []chart.EPoint, plotstyle chart.PlotStyle, style chart.Style) {
	chart.GenericScatter(ig, points, plotstyle, style)
}

func (ig *ImageGraphics) Boxes(boxes []chart.Box, width int, style chart.Style) {
	chart.GenericBoxes(ig, boxes, width, style)
}

func (ig *ImageGraphics) Key(x, y int, key chart.Key) {
	chart.GenericKey(ig, x, y, key)
}

func (ig *ImageGraphics) Bars(bars []chart.Barinfo, style chart.Style) {
	chart.GenericBars(ig, bars, style)
}

func (ig *ImageGraphics) Rings(wedges []chart.Wedgeinfo, x, y, ro, ri int) {
	chart.GenericRings(ig, wedges, x, y, ro, ri, 1)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func sign(a int) int {
	if a < 0 {
		return -1
	}
	if a == 0 {
		return 0
	}
	return 1
}

var (
	dpi      = 72
	fontfile = "../../../../code.google.com/p/freetype-go/luxi-fonts/luximr.ttf"
	size     = 14
	theFont  *truetype.Font
)

func init() {
	// Read the font data.
	fontBytes, err := ioutil.ReadFile(fontfile)
	if err != nil {
		log.Println(err)
	}
	theFont, err = freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
	}
	log.Println("Loaded Font")
}
