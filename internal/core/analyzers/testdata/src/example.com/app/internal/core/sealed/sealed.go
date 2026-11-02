// Package sealed holds sealed interfaces, marked and unmarked.
package sealed

type Unmarked interface{ isUnmarked() } // want `Unmarked is sealed, so it's a closed family; add a //sumtype:decl comment`

// Marked is a closed family.
//
//sumtype:decl
type Marked interface{ isMarked() }

type (
	// InGroup is a closed family declared in a group.
	//
	//sumtype:decl
	InGroup interface{ isInGroup() }

	UnmarkedInGroup interface{ isUnmarkedInGroup() } // want `UnmarkedInGroup is sealed`
)

// Open is an interface any package can implement.
type Open interface{ Area() float64 }

type Struct struct{}
