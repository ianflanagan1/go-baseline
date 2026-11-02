// Package network is a conformance fixture.
package network

import (
	"net"
)

// F dials.
func F() (net.Conn, error) { return net.Dial("tcp", "example.com:80") }
