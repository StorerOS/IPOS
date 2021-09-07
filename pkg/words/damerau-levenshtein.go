package words

import "math"

func minimum(integers []int) (minVal int) {
	minVal = math.MaxInt32
	for _, v := range integers {
		if v < minVal {
			minVal = v
		}
	}
	return
}

func DamerauLevenshteinDistance(a string, b string) int {
	var cost int
	d := make([][]int, len(a)+1)
	for i := 1; i <= len(a)+1; i++ {
		d[i-1] = make([]int, len(b)+1)
	}
	for i := 0; i <= len(a); i++ {
		d[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		d[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			if a[i-1] == b[j-1] {
				cost = 0
			} else {
				cost = 1
			}
			d[i][j] = minimum([]int{
				d[i-1][j] + 1,
				d[i][j-1] + 1,
				d[i-1][j-1] + cost,
			})
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d[i][j] = minimum([]int{d[i][j], d[i-2][j-2] + cost})
			}
		}
	}
	return d[len(a)][len(b)]
}
