package main

import (
	"image/color"

	ui "github.com/itohio/tinygui"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

type Label struct {
	ui.WidgetBase
	text  func() string
	color color.RGBA
}

func NewLabel(w, h uint16, text func() string, color color.RGBA) *Label {
	return &Label{
		WidgetBase: ui.NewWidgetBase(w, h),
		text:       text,
		color:      color,
	}
}

func (l *Label) Draw(ctx ui.Context) {
	x, y := ctx.DisplayPos()
	tinyfont.WriteLine(ctx.D(), &proggy.TinySZ8pt7b, x, y+int16(l.Height), l.text(), l.color)
}
