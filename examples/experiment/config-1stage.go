//go:build !stages

package main

const (
	TITLE  = "1stage exp."
	STAGES = 1
)

//go:generate tinygo flash -target=pico
