package samples

import (
	"gocv.io/x/gocv"
)

var WhiteGoCV uint8 = 255
var BlackGoCV uint8 = 0

func ZhangSuen(src gocv.Mat, dst *gocv.Mat) {
	s1Flag := true
	s2Flag := true
	data := src.DataPtrUint8()
	src.CopyTo(dst)
	rows := src.Rows()
	cols := src.Cols()
	for s1Flag || s2Flag {
		s1Marks := make([][2]int, 0)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if data[r*(cols)+c] == WhiteGoCV && step1ConditionsMet(data, r, c, rows, cols) {
					s1Marks = append(s1Marks, [2]int{r, c})
				}
			}
		}
		s1Flag = len(s1Marks) > 0
		if s1Flag {
			for _, rc := range s1Marks {
				r := rc[0]
				c := rc[1]
				data[r*(cols)+c] = BlackGoCV
				dst.SetUCharAt(r, c, BlackGoCV)
			}
		}

		s2Marks := make([][2]int, 0)
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if data[r*(cols)+c] == WhiteGoCV && step2ConditionsMet(data, r, c, rows, cols) {
					s2Marks = append(s2Marks, [2]int{r, c})
				}
			}
		}
		s2Flag = len(s2Marks) > 0
		if s2Flag {
			for _, rc := range s2Marks {
				r := rc[0]
				c := rc[1]
				data[r*(cols)+c] = BlackGoCV
				dst.SetUCharAt(r, c, BlackGoCV)
			}
		}
	}
}

func step1ConditionsMet(image []uint8, row, col, rows, cols int) bool {
	ns := getNeighbours(image, row, col, rows, cols)
	return basicConditionsMet(ns) && !(ns[0] && ns[2] && ns[4]) && !(ns[2] && ns[4] && ns[6])
}

func step2ConditionsMet(image []uint8, row, col, rows, cols int) bool {
	ns := getNeighbours(image, row, col, rows, cols)
	return basicConditionsMet(ns) && !(ns[0] && ns[2] && ns[6]) && !(ns[0] && ns[4] && ns[6])
}

func getNeighbours(image []uint8, row, col, rows, cols int) []bool {
	var x0, y0, x1, y1 int

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

	var p2, p3, p4, p5, p6, p7, p8, p9 bool
	if row != rows-1 {
		p6 = image[y1*(cols)+col] == WhiteGoCV
		if col != 0 {
			p7 = image[y1*(cols)+x0] == WhiteGoCV
			p8 = image[row*(cols)+x0] == WhiteGoCV
		}
		if col != cols-1 {
			p5 = image[y1*(cols)+x1] == WhiteGoCV
			p4 = image[row*(cols)+x1] == WhiteGoCV
		}
	}
	if row != 0 {
		p2 = image[y0*(cols)+col] == WhiteGoCV
		if col != 0 {
			p9 = image[y0*(cols)+x0] == WhiteGoCV
			p8 = image[row*(cols)+x0] == WhiteGoCV
		}
		if col != cols-1 {
			p3 = image[y0*(cols)+x1] == WhiteGoCV
			p4 = image[row*(cols)+x1] == WhiteGoCV
		}
	}
	return []bool{
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

func basicConditionsMet(ns []bool) bool {
	b := nonZero(ns)
	return b >= 2 && b <= 6 && transitions(ns) == 1
}

func transitions(ns []bool) int {
	count := 0
	for i, n := range ns {
		if n && !ns[(i+1)%8] {
			count++
		}
	}
	return count
}

func nonZero(ns []bool) int {
	count := 0
	for _, n := range ns {
		if !n {
			count++
		}
	}
	return count
}
