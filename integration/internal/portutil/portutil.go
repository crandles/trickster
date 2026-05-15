/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

// Package portutil reserves free TCP ports for integration tests that need
// to write port numbers into config templates before booting trickster.
package portutil

import (
	"net"
	"testing"
)

// Reserve binds n ephemeral TCP ports, releases them, and returns the port
// numbers in the order acquired. There is a small race window between
// release and trickster's bind, but for test-suite use it eliminates the
// deterministic-port-collision class of failure.
func Reserve(t testing.TB, n int) []int {
	t.Helper()
	ls := make([]*net.TCPListener, n)
	out := make([]int, n)
	for i := range n {
		l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			for _, x := range ls[:i] {
				x.Close()
			}
			t.Fatalf("listen :0: %v", err)
		}
		ls[i] = l
		out[i] = l.Addr().(*net.TCPAddr).Port
	}
	for _, l := range ls {
		l.Close()
	}
	return out
}
