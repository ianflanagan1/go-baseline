// Package pseudorandom is a conformance fixture: a driven adapter shuffles a
// list with the standard library's pseudo-random generator.
package pseudorandom

import "math/rand/v2"

// Shuffle puts names in a random order.
func Shuffle(names []string) {
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
}
