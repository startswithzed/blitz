package tui

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type widgetPosition struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func drawLineGraph(title string, pos widgetPosition, dataChan chan float64, plots *[]ui.Drawable) {
	dataArr := make([][]float64, 1)
	dataArr[0] = make([]float64, pos.x2-pos.x1)
	var data []float64

	p := widgets.NewPlot()
	p.Title = title
	p.Data = dataArr
	p.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
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
				if len(data) > pos.x2-pos.x1 {
					data = data[1:]
				}
				copy(dataArr[0][:], data)
				ui.Clear()
				ui.Render(*plots...)
			}
		}
	}()
}

func drawGauge(title string, pos widgetPosition, duration time.Duration, ticker time.Ticker, plots *[]ui.Drawable) {
	startTime := time.Now()
	endTime := startTime.Add(duration)

	g := widgets.NewGauge()
	g.Title = title
	g.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
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

func drawTable(title string, pos widgetPosition, avgResTime time.Duration, maxResTime time.Duration, minResTime time.Duration, errCount int64, plots *[]ui.Drawable) {
	t := widgets.NewTable()
	t.Title = title
	t.Rows = [][]string{
		{
			"Average Response Time",
			"Max Response Time",
			"Min Response Time",
			"Error Count",
		},
		{
			strconv.FormatInt(avgResTime.Milliseconds(), 10),
			strconv.FormatInt(maxResTime.Milliseconds(), 10),
			strconv.FormatInt(minResTime.Milliseconds(), 10),
			strconv.FormatInt(errCount, 10),
		},
	}
	t.RowSeparator = true
	t.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
	t.RowStyles[0] = ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)
	t.TextAlignment = ui.AlignCenter

	*plots = append(*plots, t)
}

func DrawDashboard() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	var outputs []ui.Drawable

	const MaxWidth = 90

	const GaugeHeight = 3
	const GraphHeight = 10
	const TableHeight = 5

	duration := 1 * time.Minute

	durationTicker := time.NewTicker(1 * time.Second)
	defer durationTicker.Stop()

	durationGaugePos := widgetPosition{
		x1: 0,
		y1: 0,
		x2: MaxWidth,
		y2: GaugeHeight,
	}

	drawGauge("Test Duration", durationGaugePos, duration, *durationTicker, &outputs)

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

	resTimeGraphPos := widgetPosition{
		x1: 0,
		y1: GaugeHeight,
		x2: MaxWidth / 3,
		y2: GaugeHeight + GraphHeight,
	}

	drawLineGraph("Responses times", resTimeGraphPos, resTimeChan, &outputs)

	reqPSGraphPos := widgetPosition{
		x1: MaxWidth / 3,
		y1: GaugeHeight,
		x2: 2 * (MaxWidth / 3),
		y2: GaugeHeight + GraphHeight,
	}

	drawLineGraph("Requests per second", reqPSGraphPos, reqPSChan, &outputs)

	resPSGraphPos := widgetPosition{
		x1: 2 * (MaxWidth / 3),
		y1: GaugeHeight,
		x2: 3 * (MaxWidth / 3),
		y2: GaugeHeight + GraphHeight,
	}

	drawLineGraph("Responses per second", resPSGraphPos, resPSChan, &outputs)

	resStatTablePos := widgetPosition{
		x1: 0,
		y1: GaugeHeight + GraphHeight,
		x2: MaxWidth,
		y2: GaugeHeight + GraphHeight + TableHeight,
	}

	drawTable("Response Stats", resStatTablePos, 14*time.Millisecond, 18*time.Millisecond, 10*time.Millisecond, 42, &outputs)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
