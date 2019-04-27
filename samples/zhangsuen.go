package samples

func Step1ConditionsMet(sample *Sample, row, col int) bool {
	n := getNeighbours(sample, row, col)
	return basicConditionsMet(n) && !(n[1] && n[3] && n[5]) && !(n[3] && n[5] && n[7])
}

func Step2ConditionsMet(sample *Sample, row, col int) bool {
	n := getNeighbours(sample, row, col)
	return basicConditionsMet(n) && !(n[1] && n[3] && n[7]) && !(n[1] && n[5] && n[7])
}

func getNeighbours(sample *Sample, row, col int) []bool {
	var x0, y0, x1, y1 int
	if y0 = row - 1; y0 < 0 {
		y0 = 0
	}
	if y1 = row + 1; y1 >= sample.Height() {
		y1 = sample.Height() - 1
	}
	if x0 = col - 1; x0 < 0 {
		x0 = 0
	}
	if x1 = col + 1; x1 >= sample.Width() {
		x1 = sample.Width() - 1
	}

	var p1, p2, p3, p4, p5, p6, p7, p8, p9 bool
	p1 = sample.mat.GetUCharAt(row, col) > 0
	if row != sample.Height()-1 {
		p6 = sample.mat.GetUCharAt(y1, col) > 0
		if col != 0 {
			p7 = sample.mat.GetUCharAt(y1, x0) > 0
			p8 = sample.mat.GetUCharAt(row, x0) > 0
		}
		if col != sample.Width()-1 {
			p5 = sample.mat.GetUCharAt(y1, x1) > 0
			p4 = sample.mat.GetUCharAt(row, x1) > 0
		}
	}
	if row != 0 {
		p2 = sample.mat.GetUCharAt(y0, col) > 0
		if col != 0 {
			p9 = sample.mat.GetUCharAt(y0, x0) > 0
			p8 = sample.mat.GetUCharAt(row, x0) > 0
		}
		if col != sample.Width()-1 {
			p3 = sample.mat.GetUCharAt(y0, x1) > 0
			p4 = sample.mat.GetUCharAt(row, x1) > 0
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
