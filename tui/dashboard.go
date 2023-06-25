package tui

import (
	"log"
	"math/rand"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func drawLineGraph(title string, x1 int, y1 int, x2 int, y2 int, dataChan chan float64, plots *[]ui.Drawable) {
	dataArr := make([][]float64, 1)
	dataArr[0] = make([]float64, x2-x1)
	var data []float64

	p := widgets.NewPlot()
	p.Title = title
	p.Data = dataArr
	p.SetRect(x1, y1, x2, y2)
	p.AxesColor = ui.ColorBlue
	p.LineColors[0] = ui.ColorMagenta

	*plots = append(*plots, p)

	go func() {
		for {
			select {
			case val, ok := <-dataChan:
				if !ok {
					return
				}
				data = append(data, val)
				if len(data) > x2-x1 {
					data = data[1:]
				}
				copy(dataArr[0][:], data)
				ui.Clear()
				ui.Render(*plots...)
			}
		}
	}()
}

func DrawDashboard() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	var outputs []ui.Drawable

	ticker := time.NewTicker(200 * time.Millisecond)

	resTimeChan := make(chan float64)
	defer close(resTimeChan) // TODO: check for graceful exit

	rpsChan := make(chan float64)
	defer close(rpsChan)

	go func() {
		for {
			select {
			case <-ticker.C:
				resTimeChan <- float64(rand.Intn(20))
				rpsChan <- float64(rand.Intn(20))
			}
		}
	}()

	width := 45
	margin := 2
	height := width / 3

	drawLineGraph("Responses times", 0, 0, width, height, resTimeChan, &outputs)

	drawLineGraph("Requests per second", width+margin, 0, 2*width+margin, height, rpsChan, &outputs)

	drawLineGraph("Responses per second", 2*width+2*margin, 0, 3*width+2*margin, height, resTimeChan, &outputs)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
