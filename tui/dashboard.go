package tui

import (
	"fmt"
	"log"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Dashboard struct {
	testDuration   time.Duration
	durationTicker time.Ticker
	outputs        *[]ui.Drawable
}

type widgetPosition struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

func NewDashboard(testDuration time.Duration, durationTicker time.Ticker) *Dashboard {
	return &Dashboard{
		testDuration:   testDuration,
		durationTicker: durationTicker,
		outputs:        &[]ui.Drawable{},
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

//func logsToString(logs []interface{}) string {
//	str := ""
//	for _, errLog := range logs {
//		switch l := errLog.(type) {
//		case core.ErrorLog:
//			str += fmt.Sprintf("%d  [%d](fg:red)  %s  [%s](fg:blue)\n", l.Timestamp, l.StatusCode, l.Verb, l.URL)
//		case error:
//			str += fmt.Sprintf("[%s](fg:red)\n", l)
//		default:
//		}
//	}
//
//	return str
//}

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

func (d *Dashboard) drawGauge(title string, pos widgetPosition) {
	startTime := time.Now()
	endTime := startTime.Add(d.testDuration)

	g := widgets.NewGauge()
	g.Title = title
	g.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
	g.BarColor = ui.ColorGreen
	g.TitleStyle.Fg = ui.ColorCyan
	g.Percent = 0

	*d.outputs = append(*d.outputs, g)

	go func() {
		percent := 0
		for {
			select {
			case now := <-d.durationTicker.C:
				elapsed := now.Sub(startTime)
				remaining := endTime.Sub(now)

				if remaining <= 0 {
					break
				}

				percent = int(elapsed * 100 / d.testDuration)

				g.Percent = percent
				g.Label = fmt.Sprintf("%v%% %v/%v", g.Percent, formatDuration(elapsed), formatDuration(d.testDuration))
				ui.Clear()
				ui.Render(*d.outputs...)
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

//func drawLogs(title string, pos widgetPosition, errStream chan interface{}, plots *[]ui.Drawable) {
//	logs := make([]interface{}, 11)
//
//	p := widgets.NewParagraph()
//	p.Title = title
//	p.Text = logsToString(logs)
//	p.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
//
//	*plots = append(*plots, p)
//
//	go func() {
//		for {
//			select {
//			case val, ok := <-errStream:
//				if !ok {
//					return
//				}
//				switch l := val.(type) {
//				case core.ErrorLog, error:
//					logs = append(logs, l)
//					if len(logs) > 10 {
//						logs = logs[1:]
//					}
//					p.Text = logsToString(logs)
//					ui.Clear()
//					ui.Render(*plots...)
//				default:
//				}
//			}
//		}
//	}()
//}

func (d *Dashboard) DrawDashboard() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	const MaxWidth = 90

	const GaugeHeight = 3
	const GraphHeight = 10
	const TableHeight = 5
	const LogsHeight = 12

	//durationTicker := time.NewTicker(1 * time.Second)
	//defer durationTicker.Stop()

	durationGaugePos := widgetPosition{
		x1: 0,
		y1: 0,
		x2: MaxWidth,
		y2: GaugeHeight,
	}

	d.drawGauge("Test Duration", durationGaugePos)

	//ticker := time.NewTicker(200 * time.Millisecond)
	//
	//resTimeChan := make(chan float64)
	//defer close(resTimeChan) // TODO: check for graceful exit
	//
	//reqPSChan := make(chan float64)
	//defer close(reqPSChan)
	//
	//resPSChan := make(chan float64)
	//defer close(resPSChan)
	//
	//errStream := make(chan interface{})
	//defer close(errStream)

	//go func() {
	//	rand.Seed(time.Now().UnixNano())
	//	for {
	//		select {
	//		case <-ticker.C:
	//			resTimeChan <- float64(rand.Intn(20))
	//			reqPSChan <- float64(rand.Intn(50))
	//			resPSChan <- float64(rand.Intn(50))
	//			idx := rand.Intn(2)
	//			switch idx {
	//			case 0:
	//				errStream <- core.ErrorLog{Timestamp: 1687770075, Verb: "GET", URL: "http://dummywebserver.startswithzed.repl.co", StatusCode: 501}
	//			case 1:
	//				errStream <- errors.New("something went wrong")
	//			}
	//		}
	//	}
	//}()

	//resTimeGraphPos := widgetPosition{
	//	x1: 0,
	//	y1: GaugeHeight,
	//	x2: MaxWidth / 3,
	//	y2: GaugeHeight + GraphHeight,
	//}
	//drawLineGraph("Responses times (ms)", resTimeGraphPos, resTimeChan, &outputs)
	//
	//reqPSGraphPos := widgetPosition{
	//	x1: MaxWidth / 3,
	//	y1: GaugeHeight,
	//	x2: 2 * (MaxWidth / 3),
	//	y2: GaugeHeight + GraphHeight,
	//}
	//drawLineGraph("Requests per second", reqPSGraphPos, reqPSChan, &outputs)
	//
	//resPSGraphPos := widgetPosition{
	//	x1: 2 * (MaxWidth / 3),
	//	y1: GaugeHeight,
	//	x2: 3 * (MaxWidth / 3),
	//	y2: GaugeHeight + GraphHeight,
	//}
	//drawLineGraph("Responses per second", resPSGraphPos, resPSChan, &outputs)
	//
	//resStatTablePos := widgetPosition{
	//	x1: 0,
	//	y1: GaugeHeight + GraphHeight,
	//	x2: MaxWidth,
	//	y2: GaugeHeight + GraphHeight + TableHeight,
	//}
	//drawTable("Response Stats", resStatTablePos, 14*time.Millisecond, 18*time.Millisecond, 10*time.Millisecond, 42, &outputs)
	//
	//errorLogsPos := widgetPosition{
	//	x1: 0,
	//	y1: GaugeHeight + GraphHeight + TableHeight,
	//	x2: MaxWidth,
	//	y2: GaugeHeight + GraphHeight + TableHeight + LogsHeight,
	//}
	//drawLogs("Error Logs", errorLogsPos, errStream, &outputs)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		}
	}
}
