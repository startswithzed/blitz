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

	n := 45

	data := make([][]float64, 1)
	data[0] = make([]float64, n)
	var rps []float64

	ticker := time.NewTicker(200 * time.Millisecond)

	p0 := widgets.NewPlot()
	p0.Title = "Response Time"
	p0.Data = data
	p0.SetRect(0, 0, n+5, n/2)
	p0.AxesColor = ui.ColorBlue
	p0.LineColors[0] = ui.ColorMagenta

	go func() {
		for {
			select {
			case <-ticker.C:
				rps = append(rps, float64(rand.Intn(20)))
				if len(rps) > n {
					rps = rps[1:]
				}
				copy(data[0][:], rps)
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
