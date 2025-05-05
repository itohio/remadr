//go:build stages

package main

const (
	TITLE  = "2stages exp."
	STAGES = 2
)

//go:generate tinygo flash -tags=stages -target=pico
