package main

//===================================================
// Variables you can adjust yourself
//===================================================
var (
	// This fixes it completely for me (all conns are healthy at the end)
	neverInterruptQueries = false

	// This lets you attach with a debugger before we touch connections
	facilitateDebugging = false
)
