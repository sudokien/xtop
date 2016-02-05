package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/solidfoxrock/xtop/Godeps/_workspace/src/github.com/jroimartin/gocui"
	"github.com/solidfoxrock/xtop/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v1"
)

func main() {
	name := "xtop"
	help := `A top-like tool to monitor responses from a target URL. This tool periodically collects and prints out response statuses and a custom response header received from the url.`

	app := kingpin.New(name, help)
	app.Version("0.0.2")

	var (
		url         = app.Arg("url", "target URL").Required().URL()
		concurrency = app.Flag("concurrency", "max number of concurrent requests").Short('c').Default("10").Int()
		header      = app.Flag("header", "custom header name to collect").Short('x').Default("X-Server").String()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	x := NewXTop((*url).String(), *concurrency, *header)
	x.Start()
}

// Pair is a data structure to hold a key/value pair.
type Pair struct {
	Key   string
	Value int
}

// PairList is a slice of Pairs that implements sort.Interface to sort by Value.
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

// sortMapByValue turns a map into a PairList, then sort and return it.
func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(p)
	return p
}

// appendIfMissing appends a string into a slice if it does not exist yet
func appendIfMissing(a []string, item string) []string {
	for _, v := range a {
		if item == v {
			return a
		}
	}
	return append(a, item)
}

// XTop is the main struct
type XTop struct {
	URL         string
	Concurrency int
	Header      string

	TotalRequestsSent int
	StatusMap         map[string]int
	HeaderMap         map[string]int
	G                 *gocui.Gui
}

// Data is the transfer object for communication from our background worker threads
type Data struct {
	RespStatus string
	RespHeader string
	Error      error
}

// NewXTop returns a new XTop instance
func NewXTop(url string, concurrency int, header string) XTop {
	url = strings.ToLower(url)
	if strings.Index(url, "http://") == -1 && strings.Index(url, "https://") == -1 {
		url = "http://" + url
	}

	x := XTop{}
	x.URL = url
	x.Concurrency = concurrency
	x.Header = header
	x.StatusMap = make(map[string]int)
	x.HeaderMap = make(map[string]int)

	var err error
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		panic(err)
	}
	g.FgColor = gocui.ColorGreen
	g.SetLayout(x.layout)
	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.Quit
	}); err != nil {
		log.Panicln(err)
	}

	x.G = g
	return x
}

// Start starts everything
func (x *XTop) Start() {
	defer x.G.Close()

	// Collect data
	go x.run()

	// Periodically print out results to the screen
	go x.updateView()

	// Main loop of gocui
	if err := x.G.MainLoop(); err != nil && err != gocui.Quit {
		log.Panicln(err)
	}
}

// run spawns multiple worker goroutines making requests to target URL in background
// then it keeps listening for results sent back from those workers
func (x *XTop) run() {
	N := x.Concurrency       // max concurrency
	ch := make(chan Data, N) // channel for workers to send back results

	// Spawn worker goroutines
	for i := 0; i < N; i++ {
		go func(ch chan Data) {
			for {
				resp, err := http.Get(x.URL)
				if err != nil {
					ch <- Data{Error: err}
					continue
				}
				ch <- Data{resp.Status, resp.Header.Get(x.Header), nil}
			}
		}(ch)
	}

	// Collecting results from workers
	for {
		data := <-ch
		x.TotalRequestsSent++

		if data.Error != nil {
			// TODO collect errors
			continue
		}

		s := data.RespStatus
		if _, exist := x.StatusMap[s]; exist {
			x.StatusMap[s]++
		} else {
			x.StatusMap[s] = 1
		}

		h := data.RespHeader
		if _, exist := x.HeaderMap[h]; exist {
			x.HeaderMap[h]++
		} else {
			x.HeaderMap[h] = 1
		}
	}
}

func (x *XTop) layout(*gocui.Gui) error {
	maxX, maxY := x.G.Size()
	if v, err := x.G.SetView("center", 3, 0, maxX, maxY); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Frame = false
		x.display(v)
	}
	return nil
}

// updateView runs in a goroutine to periodically print out results to the screen
// it calls display() to actually print to a view
func (x *XTop) updateView() {
	for {
		time.Sleep(time.Second)
		v, err := x.G.View("center")
		if err != nil {
			panic(err)
		}
		v.Clear()
		x.display(v)
		x.G.Flush()
	}
}

// display prints output to the view
func (x *XTop) display(v io.Writer) error {

	output := fmt.Sprintf("Target: %s\n", x.URL)
	output += fmt.Sprintf("Header: %s\n", x.Header)
	output += fmt.Sprintf("Max concurrency: %d\n\n", x.Concurrency)

	// Response status
	sorted := sortMapByValue(x.StatusMap)
	output += fmt.Sprintf("=== Response status ===\n")
	for _, v := range sorted {
		output += fmt.Sprintf("%6s %8s %s\n",
			fmt.Sprintf("[%d%s]", v.Value*100/x.TotalRequestsSent, "%%"),
			fmt.Sprintf("[%d/%d]", v.Value, x.TotalRequestsSent),
			v.Key)
	}
	output += fmt.Sprintf("\n")

	// Response header
	a := []string{}

	for k := range x.HeaderMap {
		a = append(a, k)
	}
	sort.Strings(a)

	output += fmt.Sprintf("=== Response header %s ===\n", x.Header)
	for i, v := range a {
		output += fmt.Sprintf("%6s %8s %3d %s\n",
			fmt.Sprintf("[%d%s]", x.HeaderMap[v]*100/x.TotalRequestsSent, "%%"),
			fmt.Sprintf("[%d/%d]", x.HeaderMap[v], x.TotalRequestsSent),
			i+1,
			v)
	}
	output += fmt.Sprintf("\n")
	fmt.Fprintf(v, output)
	return nil
}
