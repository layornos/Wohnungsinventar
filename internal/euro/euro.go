// Package euro formats amounts, which the app keeps as whole cents (spec 06).
package euro

import (
	"fmt"
	"strings"
)

// Format formats cents the German way: "15 €", "45,50 €", "1.200 €".
func Format(cents int64) string {
	euros := fmt.Sprint(cents / 100)
	var b strings.Builder
	for i, c := range euros {
		if i > 0 && (len(euros)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if rest := cents % 100; rest != 0 {
		fmt.Fprintf(&b, ",%02d", rest)
	}
	return b.String() + " €"
}
