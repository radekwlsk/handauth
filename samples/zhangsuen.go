package samples

import (
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/draw"
)

var WhiteGoCV = color.Gray{Y: 0xff}
var BlackGoCV = color.Gray{Y: 0x0}

func ZhangSuen(src gocv.Mat, dst *gocv.Mat) {
	s1Flag := true
	s2Flag := true
	img, err := src.ToImage()
	if err != nil {
		panic(err)
	}
	grayImg := image.NewGray(img.Bounds())
	draw.Draw(grayImg, grayImg.Bounds(), img, image.Point{}, draw.Over)
	rows := src.Rows()
	cols := src.Cols()
	for s1Flag || s2Flag {
		s1Marks := make([][2]int, 0)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if step1ConditionsMet(grayImg, r, c) {
					s1Marks = append(s1Marks, [2]int{r, c})
				}
			}
		}
		s1Flag = len(s1Marks) > 0
		if s1Flag {
			for _, rc := range s1Marks {
				grayImg.Set(rc[1], rc[0], BlackGoCV)
			}
		}

		s2Marks := make([][2]int, 0)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if step2ConditionsMet(grayImg, r, c) {
					s2Marks = append(s2Marks, [2]int{r, c})
				}
			}
		}
		s2Flag = len(s2Marks) > 0
		if s2Flag {
			for _, rc := range s2Marks {
				grayImg.Set(rc[1], rc[0], BlackGoCV)
			}
		}
	}
	mat, _ := gocv.ImageGrayToMatGray(grayImg)
	defer mat.Close()
	mat.CopyTo(dst)
}

func step1ConditionsMet(image image.Image, row, col int) bool {
	n := getNeighbours(image, row, col)
	return basicConditionsMet(n) && !(n[1] && n[3] && n[5]) && !(n[3] && n[5] && n[7])
}

func step2ConditionsMet(image image.Image, row, col int) bool {
	n := getNeighbours(image, row, col)
	return basicConditionsMet(n) && !(n[1] && n[3] && n[7]) && !(n[1] && n[5] && n[7])
}

func getNeighbours(image image.Image, row, col int) []bool {
	var x0, y0, x1, y1 int
	bounds := image.Bounds()
	rows := bounds.Max.Y
	cols := bounds.Max.X

	if y0 = row - 1; y0 < 0 {
		y0 = 0
	}
	if y1 = row + 1; y1 >= rows {
		y1 = rows - 1
	}
	if x0 = col - 1; x0 < 0 {
		x0 = 0
	}
	if x1 = col + 1; x1 >= cols {
		x1 = cols - 1
	}

	var p1, p2, p3, p4, p5, p6, p7, p8, p9 bool
	p1 = image.At(col, row) == WhiteGoCV
	if row != rows-1 {
		p6 = image.At(col, y1) == WhiteGoCV
		if col != 0 {
			p7 = image.At(x0, y1) == WhiteGoCV
			p8 = image.At(x0, row) == WhiteGoCV
		}
		if col != cols-1 {
			p5 = image.At(x1, y1) == WhiteGoCV
			p4 = image.At(x1, row) == WhiteGoCV
		}
	}
	if row != 0 {
		p2 = image.At(col, y0) == WhiteGoCV
		if col != 0 {
			p9 = image.At(x0, y0) == WhiteGoCV
			p8 = image.At(x0, row) == WhiteGoCV
		}
		if col != cols-1 {
			p3 = image.At(x1, y0) == WhiteGoCV
			p4 = image.At(x1, row) == WhiteGoCV
		}
	}
	return []bool{
		p1,
		p2,
		p3,
		p4,
		p5,
		p6,
		p7,
		p8,
		p9,
	}
}

func basicConditionsMet(n []bool) bool {
	a := transitions(n)
	b := nonZero(n)
	return n[0] && b >= 2 && b <= 6 && a == 1
}

func transitions(n []bool) int {
	p := n[1]
	count := 0
	for _, n := range []bool{n[2], n[3], n[4], n[5], n[6], n[7], n[8], n[1]} {
		if p && !n {
			count++
		}
		p = n
	}
	return count
}

func nonZero(n []bool) int {
	count := 0
	for _, n := range []bool{n[1], n[2], n[3], n[4], n[5], n[6], n[7], n[8]} {
		if !n {
			count++
		}
	}
	return count
}
