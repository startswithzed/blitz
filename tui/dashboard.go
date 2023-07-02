package tui

import (
	"context"
	"fmt"
	"github.com/startswithzed/web-ruckus/core"
	"log"
	"strconv"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Dashboard struct {
	testDuration   time.Duration
	durationTicker *time.Ticker
	outputs        *[]ui.Drawable
	uiMutex        sync.Mutex
	cancel         context.CancelFunc

	// refresh channel
	refreshReqChan chan struct{}

	// data channels
	reqPS        <-chan uint64
	resPS        <-chan uint64
	resTimes     <-chan uint64
	resStats     <-chan core.ResponseTimeStats
	errorStream  <-chan interface{}
	errCountChan <-chan uint64
}

type widgetPosition struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

type DashboardConfig struct {
	Duration    time.Duration
	Ticker      *time.Ticker
	Cancel      context.CancelFunc
	ReqPS       <-chan uint64
	ResPS       <-chan uint64
	ResTimes    <-chan uint64
	ResStats    <-chan core.ResponseTimeStats
	ErrorStream <-chan interface{}
	ErrorCount  <-chan uint64
}

func NewDashboard(dc DashboardConfig) *Dashboard {
	return &Dashboard{
		testDuration:   dc.Duration,
		durationTicker: dc.Ticker,
		outputs:        &[]ui.Drawable{},
		uiMutex:        sync.Mutex{},
		cancel:         dc.Cancel,
		refreshReqChan: make(chan struct{}, 1),
		reqPS:          dc.ReqPS,
		resPS:          dc.ResPS,
		resTimes:       dc.ResTimes,
		resStats:       dc.ResStats,
		errorStream:    dc.ErrorStream,
		errCountChan:   dc.ErrorCount,
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func logsToString(logs []interface{}) string {
	str := ""
	for _, errLog := range logs {
		switch l := errLog.(type) {
		case core.ResponseError:
			str += fmt.Sprintf("%d  [%d](fg:red)  %s  [%s](fg:blue)\n", l.Timestamp, l.StatusCode, l.Verb, l.URL)
		case core.NetworkError:
			str += fmt.Sprintf("%d  [%s](fg:red)\n", l.Timestamp, l.Error)
		default:
		}
	}

	return str
}

func uint64ToFloat64Chan(in <-chan uint64) <-chan float64 {
	out := make(chan float64)

	go func() {
		for {
			select {
			case val, ok := <-in:
				if !ok {
					close(out)
					return
				}
				out <- float64(val)
			}
		}
	}()

	return out
}

func (d *Dashboard) refreshUI() {
	d.uiMutex.Lock()
	defer d.uiMutex.Unlock()

	ui.Clear()
	ui.Render(*d.outputs...)
}

func (d *Dashboard) launchRefreshWorker() {
	go func() {
		for {
			select {
			case _, ok := <-d.refreshReqChan:
				if !ok {
					return
				}
				d.refreshUI()
			}
		}
	}()
}

func (d *Dashboard) drawLineGraph(title string, pos widgetPosition, dataChan <-chan float64) {
	const MarkingsBuffer = 7
	lengthXAxis := pos.x2 - pos.x1 - MarkingsBuffer

	dataArr := make([][]float64, 1)
	dataArr[0] = make([]float64, lengthXAxis)
	var data []float64

	p := widgets.NewPlot()
	p.Title = title
	p.Data = dataArr
	p.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
	p.LineColors[0] = ui.ColorYellow
	p.DrawDirection = widgets.DrawRight

	*d.outputs = append(*d.outputs, p)

	go func() {
		for {
			select {
			case val, ok := <-dataChan:
				if !ok {
					return
				}
				data = append(data, val)
				if len(data) > lengthXAxis {
					data = data[1:]
				}
				copy(dataArr[0][:], data)
				select {
				case d.refreshReqChan <- struct{}{}:
				default:
				}
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
					return // TODO: Cancel all other goroutines
				}

				percent = int(elapsed * 100 / d.testDuration)

				g.Percent = percent
				g.Label = fmt.Sprintf("%v%% %v/%v", g.Percent, formatDuration(elapsed), formatDuration(d.testDuration))
				select {
				case d.refreshReqChan <- struct{}{}:
				default:
				}
			}
		}
	}()
}

func (d *Dashboard) drawTable(title string, pos widgetPosition) {
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
			"0",
			"0",
			"0",
			"0",
		},
	}
	t.RowSeparator = true
	t.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)
	t.RowStyles[0] = ui.NewStyle(ui.ColorWhite, ui.ColorClear, ui.ModifierBold)
	t.TextAlignment = ui.AlignCenter

	*d.outputs = append(*d.outputs, t)

	go func() {
		for {
			select {
			case c, ok := <-d.errCountChan:
				if !ok {
					return
				}
				t.Rows[1][3] = strconv.FormatUint(c, 10)
				select {
				case d.refreshReqChan <- struct{}{}:
				default:
				}
			case stats, ok := <-d.resStats:
				if !ok {
					return
				}
				t.Rows[1][0] = strconv.FormatUint(stats.AverageTime, 10)
				t.Rows[1][1] = strconv.FormatUint(stats.MaxTime, 10)
				t.Rows[1][2] = strconv.FormatUint(stats.MinTime, 10)
				select {
				case d.refreshReqChan <- struct{}{}:
				default:
				}
			default:
			}
		}
	}()
}

func (d *Dashboard) drawLogs(title string, pos widgetPosition) {
	logs := make([]interface{}, 11)

	p := widgets.NewParagraph()
	p.Title = title
	p.Text = logsToString(logs)
	p.SetRect(pos.x1, pos.y1, pos.x2, pos.y2)

	*d.outputs = append(*d.outputs, p)

	go func() {
		for {
			select {
			case val, ok := <-d.errorStream:
				if !ok {
					return
				}
				switch l := val.(type) {
				case core.ResponseError, core.NetworkError:
					logs = append(logs, l)
					if len(logs) > 10 {
						logs = logs[1:]
					}
					p.Text = logsToString(logs)
					select {
					case d.refreshReqChan <- struct{}{}:
					default:
					}
				default:
				}
			}
		}
	}()
}

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

	d.launchRefreshWorker()

	durationGaugePos := widgetPosition{
		x1: 0,
		y1: 0,
		x2: MaxWidth,
		y2: GaugeHeight,
	}

	d.drawGauge("Test Duration", durationGaugePos)

	resTimeGraphPos := widgetPosition{
		x1: 0,
		y1: GaugeHeight,
		x2: MaxWidth / 3,
		y2: GaugeHeight + GraphHeight,
	}
	d.drawLineGraph("Responses times (ms)", resTimeGraphPos, uint64ToFloat64Chan(d.resTimes))

	reqPSGraphPos := widgetPosition{
		x1: MaxWidth / 3,
		y1: GaugeHeight,
		x2: 2 * (MaxWidth / 3),
		y2: GaugeHeight + GraphHeight,
	}
	d.drawLineGraph("Requests per second", reqPSGraphPos, uint64ToFloat64Chan(d.reqPS))

	resPSGraphPos := widgetPosition{
		x1: 2 * (MaxWidth / 3),
		y1: GaugeHeight,
		x2: 3 * (MaxWidth / 3),
		y2: GaugeHeight + GraphHeight,
	}
	d.drawLineGraph("Responses per second", resPSGraphPos, uint64ToFloat64Chan(d.resPS))

	resStatTablePos := widgetPosition{
		x1: 0,
		y1: GaugeHeight + GraphHeight,
		x2: MaxWidth,
		y2: GaugeHeight + GraphHeight + TableHeight,
	}
	d.drawTable("Response Stats", resStatTablePos)

	errorLogsPos := widgetPosition{
		x1: 0,
		y1: GaugeHeight + GraphHeight + TableHeight,
		x2: MaxWidth,
		y2: GaugeHeight + GraphHeight + TableHeight + LogsHeight,
	}
	d.drawLogs("Error Logs", errorLogsPos)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			d.cancel()
			close(d.refreshReqChan)
			return
		}
	}
}
