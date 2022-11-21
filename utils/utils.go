package utils

import (
	"fmt"
)

// Convert month days to ordinal indicators
func Ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		suffix = "st"
	case 2:
		suffix = "nd"
	case 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%v%s", n, suffix)
}
