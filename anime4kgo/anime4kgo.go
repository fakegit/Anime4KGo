package anime4kgo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"

	"github.com/TianZerL/Anime4KGo/options"

	"github.com/disintegration/gift"
)

//Img defined the image class
type Img struct {
	W       int
	H       int
	FmtType string
	data    image.Image
}

//LoadImg read a image file and return
func LoadImg(src string) *Img {
	f, err := ioutil.ReadFile(src)
	if err != nil {
		log.Fatalln(err)
	}
	r := bytes.NewReader(f)
	img, fmtType, err := image.Decode(r)
	if err != nil {
		log.Fatalln(err)
	}
	b := img.Bounds()
	return &Img{W: b.Dx(), H: b.Dy(), FmtType: fmtType, data: img}
}

//ShowInfo will show the basic infomation of the image
func (img *Img) ShowInfo(opt *options.Opt) {
	fmt.Fprintf(os.Stderr, "Width: %v, Height: %v, Type: %v\n", img.W, img.H, img.FmtType)
	fmt.Println("----------------------------------------------")
	fmt.Fprintf(os.Stderr, "Input: \"%v\"\nOutput: \"%v\"\nPasses: %v\nFast mode: %v\nStrength color: %v\nStrength gradient: %v\n", opt.InputFile, opt.OutputFile, opt.Passes, opt.FastMode, opt.StrengthColor, opt.StrengthGradient)
	fmt.Println("----------------------------------------------")
}

//Process anime4k
func (img *Img) Process(passes int, sc, sg float64, fastMode bool) {
	//resize
	g := gift.New(gift.Resize(img.W*2, img.H*2, gift.CubicResampling))
	dstImg := image.NewRGBA(g.Bounds(img.data.Bounds()))
	g.Draw(dstImg, img.data)
	//debug("./debug/resize.png", dstImg)
	for i := 0; i < passes; i++ {
		getGray(dstImg)
		//debug("./debug/get_gray.jpg", dstImg)
		pushColor(dstImg, sc)
		//debug("./debug/push_color.jpg", dstImg)
		getGradient(dstImg, fastMode)
		//debug("./debug/get_gradient.jpg", dstImg)
		pushGradient(dstImg, sg)
		//debug("./debug/out.jpg", dstImg)
	}
	//debug("./debug/rst_out.png", dstImg)
	img.data = dstImg
}

//getGray compute the grayscale of the image and store it to the Alpha channel
func getGray(img *image.RGBA) {
	changeEachPixel(img, func(x, y int, p *color.RGBA) color.RGBA {
		g := 0.299*float64(p.R) + 0.587*float64(p.G) + 0.114*float64(p.B)
		return color.RGBA{p.R, p.G, p.B, uint8(g)}
	})
}

//getGradient compute the gradient of the image and store it to the Alpha channel
func getGradient(img *image.RGBA, fastMode bool) {
	maxX := img.Bounds().Dx() - 1
	maxY := img.Bounds().Dy() - 1

	if fastMode == true {
		changeEachPixel(img, func(x, y int, p *color.RGBA) color.RGBA {
			if x == 0 || x == maxX || y == 0 || y == maxY {
				return *p
			}

			G := (math.Abs(float64(img.RGBAAt(x-1, y+1).A)+2*float64(img.RGBAAt(x, y+1).A)+float64(img.RGBAAt(x+1, y+1).A)-float64(img.RGBAAt(x-1, y-1).A)-2*float64(img.RGBAAt(x, y-1).A)-float64(img.RGBAAt(x+1, y-1).A)) +
				math.Abs(float64(img.RGBAAt(x-1, y-1).A)+2*float64(img.RGBAAt(x-1, y).A)+float64(img.RGBAAt(x-1, y+1).A)-float64(img.RGBAAt(x+1, y-1).A)-2*float64(img.RGBAAt(x+1, y).A)-float64(img.RGBAAt(x+1, y+1).A)))

			rst := unFloat(G / 2)
			return color.RGBA{p.R, p.G, p.B, 255 - rst}
		})
	} else {
		changeEachPixel(img, func(x, y int, p *color.RGBA) color.RGBA {
			if x == 0 || x == maxX || y == 0 || y == maxY {
				return *p
			}

			G := math.Sqrt((float64(img.RGBAAt(x-1, y+1).A)+2*float64(img.RGBAAt(x, y+1).A)+float64(img.RGBAAt(x+1, y+1).A)-float64(img.RGBAAt(x-1, y-1).A)-2*float64(img.RGBAAt(x, y-1).A)-float64(img.RGBAAt(x+1, y-1).A))*(float64(img.RGBAAt(x-1, y+1).A)+2*float64(img.RGBAAt(x, y+1).A)+float64(img.RGBAAt(x+1, y+1).A)-float64(img.RGBAAt(x-1, y-1).A)-2*float64(img.RGBAAt(x, y-1).A)-float64(img.RGBAAt(x+1, y-1).A)) +
				(float64(img.RGBAAt(x-1, y-1).A)+2*float64(img.RGBAAt(x-1, y).A)+float64(img.RGBAAt(x-1, y+1).A)-float64(img.RGBAAt(x+1, y-1).A)-2*float64(img.RGBAAt(x+1, y).A)-float64(img.RGBAAt(x+1, y+1).A))*(float64(img.RGBAAt(x-1, y-1).A)+2*float64(img.RGBAAt(x-1, y).A)+float64(img.RGBAAt(x-1, y+1).A)-float64(img.RGBAAt(x+1, y-1).A)-2*float64(img.RGBAAt(x+1, y).A)-float64(img.RGBAAt(x+1, y+1).A)))

			rst := unFloat(G)
			return color.RGBA{p.R, p.G, p.B, 255 - rst}
		})
	}
}

//pushColor will make the linework of the image thinner guided by the grayscale in Alpha channel
//the range of strength from 0 to 1, greater for thinner
func pushColor(dst *image.RGBA, strength float64) {
	getLightest := func(mc, a, b, c *color.RGBA) {
		mc.R = unFloat(float64(mc.R)*(1.0-strength) + ((float64(a.R)+float64(b.R)+float64(c.R))/3.0)*strength)
		mc.G = unFloat(float64(mc.G)*(1.0-strength) + ((float64(a.G)+float64(b.G)+float64(c.G))/3.0)*strength)
		mc.B = unFloat(float64(mc.B)*(1.0-strength) + ((float64(a.B)+float64(b.B)+float64(c.B))/3.0)*strength)
		mc.A = unFloat(float64(mc.A)*(1.0-strength) + ((float64(a.A)+float64(b.A)+float64(c.A))/3.0)*strength)
	}

	changeEachPixel(dst, func(x, y int, p *color.RGBA) color.RGBA {
		xn, xp, yn, yp := -1, 1, -1, 1
		if x == 0 {
			xn = 0
		} else if x == dst.Bounds().Dx()-1 {
			xp = 0
		}
		if y == 0 {
			yn = 0
		} else if y == dst.Bounds().Dy()-1 {
			yp = 0
		}

		tl, tc, tr := dst.RGBAAt(x+xn, y+yn), dst.RGBAAt(x, y+yn), dst.RGBAAt(x+xp, y+yn)
		ml, mc, mr := dst.RGBAAt(x+xn, y), *p, dst.RGBAAt(x+xp, y)
		bl, bc, br := dst.RGBAAt(x+xn, y+yp), dst.RGBAAt(x, y+yp), dst.RGBAAt(x+xn, y+yp)

		//top and bottom
		maxD := max(bl.A, bc.A, br.A)
		minL := min(tl.A, tc.A, tr.A)
		if minL > mc.A && mc.A > maxD {
			getLightest(&mc, &tl, &tc, &tr)
		} else {
			maxD = max(tl.A, tc.A, tr.A)
			minL = min(bl.A, bc.A, br.A)
			if minL > mc.A && mc.A > maxD {
				getLightest(&mc, &bl, &bc, &br)
			}
		}

		//subdiagonal
		maxD = max(ml.A, mc.A, bc.A)
		minL = min(mr.A, tc.A, tr.A)
		if minL > maxD {
			getLightest(&mc, &mr, &tc, &tr)
		} else {
			maxD = max(mc.A, mr.A, tc.A)
			minL = min(bl.A, ml.A, bc.A)
			if minL > maxD {
				getLightest(&mc, &bl, &ml, &bc)
			}
		}

		//left and right
		maxD = max(tl.A, ml.A, bl.A)
		minL = min(tr.A, mr.A, br.A)
		if minL > mc.A && mc.A > maxD {
			getLightest(&mc, &tr, &mr, &br)
		} else {
			maxD = max(tr.A, mr.A, br.A)
			minL = min(tl.A, ml.A, bl.A)
			if minL > mc.A && mc.A > maxD {
				getLightest(&mc, &tl, &ml, &bl)
			}
		}

		//diagonal
		maxD = max(ml.A, mc.A, tc.A)
		minL = min(mr.A, br.A, bc.A)
		if minL > maxD {
			getLightest(&mc, &mr, &br, &tc)
		} else {
			maxD = max(mc.A, mr.A, bc.A)
			minL = min(tc.A, ml.A, tl.A)
			if minL > maxD {
				getLightest(&mc, &tc, &ml, &tl)
			}
		}

		return mc
	})
}

//pushGradient will make the linework of the image sharper guided by the gradient in Alpha channel
//the range of strength from 0 to 1, greater for sharper
func pushGradient(dst *image.RGBA, strength float64) {
	getLightest := func(mc, a, b, c *color.RGBA) color.RGBA {
		mc.R = unFloat(float64(mc.R)*(1.0-strength) + ((float64(a.R)+float64(b.R)+float64(c.R))/3.0)*strength)
		mc.G = unFloat(float64(mc.G)*(1.0-strength) + ((float64(a.G)+float64(b.G)+float64(c.G))/3.0)*strength)
		mc.B = unFloat(float64(mc.B)*(1.0-strength) + ((float64(a.B)+float64(b.B)+float64(c.B))/3.0)*strength)
		mc.A = 255

		return *mc
	}

	changeEachPixel(dst, func(x, y int, p *color.RGBA) color.RGBA {
		xn, xp, yn, yp := -1, 1, -1, 1
		if x == 0 {
			xn = 0
		} else if x == dst.Bounds().Dx()-1 {
			xp = 0
		}
		if y == 0 {
			yn = 0
		} else if y == dst.Bounds().Dy()-1 {
			yp = 0
		}

		tl, tc, tr := dst.RGBAAt(x+xn, y+yn), dst.RGBAAt(x, y+yn), dst.RGBAAt(x+xp, y+yn)
		ml, mc, mr := dst.RGBAAt(x+xn, y), *p, dst.RGBAAt(x+xp, y)
		bl, bc, br := dst.RGBAAt(x+xn, y+yp), dst.RGBAAt(x, y+yp), dst.RGBAAt(x+xn, y+yp)

		//top and right
		maxD := max(bl.A, bc.A, br.A)
		minL := min(tl.A, tc.A, tr.A)
		if minL > mc.A && mc.A > maxD {
			return getLightest(&mc, &tl, &tc, &tr)
		}
		maxD = max(tl.A, tc.A, tr.A)
		minL = min(bl.A, bc.A, br.A)
		if minL > mc.A && mc.A > maxD {
			return getLightest(&mc, &bl, &bc, &br)
		}

		//subdiagonal
		maxD = max(ml.A, mc.A, bc.A)
		minL = min(mr.A, tc.A, tr.A)
		if minL > maxD {
			return getLightest(&mc, &mr, &tc, &tr)
		}
		maxD = max(mc.A, mr.A, tc.A)
		minL = min(bl.A, ml.A, bc.A)
		if minL > maxD {
			return getLightest(&mc, &bl, &ml, &bc)
		}

		//left and right
		maxD = max(tl.A, ml.A, bl.A)
		minL = min(tr.A, mr.A, br.A)
		if minL > mc.A && mc.A > maxD {
			return getLightest(&mc, &tr, &mr, &br)
		}
		maxD = max(tr.A, mr.A, br.A)
		minL = min(tl.A, ml.A, bl.A)
		if minL > mc.A && mc.A > maxD {
			return getLightest(&mc, &tl, &ml, &bl)
		}

		//diagonal
		maxD = max(ml.A, mc.A, tc.A)
		minL = min(mr.A, br.A, bc.A)
		if minL > maxD {
			return getLightest(&mc, &mr, &br, &tc)
		}
		maxD = max(mc.A, mr.A, bc.A)
		minL = min(tc.A, ml.A, tl.A)
		if minL > maxD {
			return getLightest(&mc, &tc, &ml, &tl)
		}

		mc.A = 255
		return mc
	})
}

func max(a, b, c uint8) uint8 {
	if a > b && a > c {
		return a
	} else if b > c {
		return b
	}
	return c
}

func min(a, b, c uint8) uint8 {
	if a < b && a < c {
		return a
	} else if b < c {
		return b
	}
	return c
}

//changeEachPixel will traverse all the pixel of the image, and change it by fun, all the change will be applied after traversing
func changeEachPixel(img *image.RGBA, fun func(x, y int, p *color.RGBA) color.RGBA) {
	imgInfo := img.Bounds()
	temp := image.NewRGBA(imgInfo)
	dx, dy := imgInfo.Dx(), imgInfo.Dy()
	for i := 0; i < dx; i++ {
		for j := 0; j < dy; j++ {
			p := img.RGBAAt(i, j)
			temp.SetRGBA(i, j, fun(i, j, &p))
		}
	}
	*img = *temp
}

//unFloat convert float64 to uint8,range from 0-255
func unFloat(n float64) uint8 {
	n += 0.5

	if n >= 255 {
		return 255
	} else if n <= 0 {
		return 0
	}
	return uint8(n)
}

//debug will save a image to debug folder, which need to be created manually before using
func debug(filename string, img image.Image) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("os.Create failed: %v", err)
	}
	defer f.Close()
	//err = jpeg.Encode(f, img, nil)
	err = png.Encode(f, img)
	if err != nil {
		log.Fatalf("png.Encode failed: %v", err)
	}
}

//SaveImg will save a target image to disk
func (img *Img) SaveImg(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("os.Create failed: %v", err)
	}
	defer f.Close()
	err = png.Encode(f, img.data)
	if err != nil {
		log.Fatalf("png.Encode failed: %v", err)
	}
}
