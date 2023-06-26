package tui

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

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

func drawGauge(title string, duration time.Duration, ticker time.Ticker, width int, margin int, height int, plots *[]ui.Drawable) {
	startTime := time.Now()
	endTime := startTime.Add(duration)

	g := widgets.NewGauge()
	g.Title = title
	g.SetRect(0, 0, 3*width+2*margin, height)
	g.BarColor = ui.ColorGreen
	g.TitleStyle.Fg = ui.ColorCyan
	g.Percent = 0

	*plots = append(*plots, g)

	go func() {
		percent := 0
		for {
			select {
			case now := <-ticker.C:
				elapsed := now.Sub(startTime)
				remaining := endTime.Sub(now)

				if remaining <= 0 {
					break
				}

				percent = int(elapsed * 100 / duration)

				g.Percent = percent
				g.Label = fmt.Sprintf("%v%% %v/%v", g.Percent, formatDuration(elapsed), formatDuration(duration))
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

	width := 45
	margin := 2
	durationGaugeHeight := 3
	height := width / 3

	duration := 1 * time.Minute

	durationTicker := time.NewTicker(1 * time.Second)
	defer durationTicker.Stop()

	drawGauge("Test Duration", duration, *durationTicker, width, margin, durationGaugeHeight, &outputs)

	ticker := time.NewTicker(200 * time.Millisecond)

	resTimeChan := make(chan float64)
	defer close(resTimeChan) // TODO: check for graceful exit

	reqPSChan := make(chan float64)
	defer close(reqPSChan)

	resPSChan := make(chan float64)
	defer close(resPSChan)

	go func() {
		rand.Seed(time.Now().UnixNano())
		for {
			select {
			case <-ticker.C:
				resTimeChan <- float64(rand.Intn(20))
				reqPSChan <- float64(rand.Intn(50))
				resPSChan <- float64(rand.Intn(50))
			}
		}
	}()

	drawLineGraph("Responses times", 0, durationGaugeHeight+margin, width, durationGaugeHeight+margin+height, resTimeChan, &outputs)

	drawLineGraph("Requests per second", width+margin, durationGaugeHeight+margin, 2*width+margin, durationGaugeHeight+margin+height, reqPSChan, &outputs)

	drawLineGraph("Responses per second", 2*width+2*margin, durationGaugeHeight+margin, 3*width+2*margin, durationGaugeHeight+margin+height, resPSChan, &outputs)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
