package main

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"math/rand"
	"time"
)

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	n := 500

	rps := make([][]float64, 1)
	rps[0] = make([]float64, n)

	ticker := time.NewTicker(200 * time.Millisecond)
	i := 0

	p0 := widgets.NewPlot()
	p0.Title = "Response Time"
	p0.Data = rps
	p0.SetRect(0, 0, 100, 25)
	p0.AxesColor = ui.ColorWhite
	p0.LineColors[0] = ui.ColorGreen

	go func() {
		for {
			select {
			case <-ticker.C:
				rps[0][i] = float64(rand.Intn(20))
				i++
				ui.Clear()
				ui.Render(p0)
			}
		}
	}()

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
