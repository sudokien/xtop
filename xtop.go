package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
	"gopkg.in/alecthomas/kingpin.v1"
)

func main() {
	name := "xtop"
	help := `A top-like tool to monitor responses from a target URL. This tool periodically collects and prints out response statuses and a custom response header received from the url.`

	app := kingpin.New(name, help)
	app.Version("0.0.1")

	var (
		url        = app.Arg("url", "target URL").Required().URL()
		concurrent = app.Flag("concurrent", "number of concurrent requests sent in one batch").Short('c').Default("3").Int()
		header     = app.Flag("header", "custom header name to collect").Short('x').Default("X-Server").String()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	x := NewXTop((*url).String(), *concurrent, *header)
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
	URL        string
	Concurrent int
	Header     string

	TotalRequestsSent int
	StatusMap         map[string]int
	ServerMap         map[string]int
	G                 *gocui.Gui
}

// NewXTop returns a new XTop instance
func NewXTop(url string, concurrent int, header string) XTop {
	url = strings.ToLower(url)
	if strings.Index(url, "http://") == -1 && strings.Index(url, "https://") == -1 {
		url = "http://" + url
	}

	x := XTop{}
	x.URL = url
	x.Concurrent = concurrent
	x.Header = header
	x.StatusMap = make(map[string]int)
	x.ServerMap = make(map[string]int)

	var err error = nil
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		panic(err)
	}
	g.FgColor = gocui.ColorGreen
	g.SetLayout(x.layout)
	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrorQuit
	}); err != nil {
		log.Panicln(err)
	}

	x.G = g
	return x
}

// Run the app
func (r *XTop) Start() {
	defer r.G.Close()

	// One thread (that spawns several goroutines) to send requests and collect data
	go r.run()

	// One thread to periodically print out results to the screen
	go r.updateView()

	// Main loop of gocui
	if err := r.G.MainLoop(); err != nil && err != gocui.ErrorQuit {
		log.Panicln(err)
	}
}

// run has an infinite loop, spawning several goroutines to send http requests and collect data
func (r *XTop) run() {

	var (
		status, header string
		wg             sync.WaitGroup
	)

	for { // loop forever, send N requests each loop
		for i := 0; i < r.Concurrent; i++ {
			wg.Add(1)
			go func() {
				// decrement the counter when the goroutine completes.
				defer wg.Done()

				r.TotalRequestsSent++

				resp, err := http.Get(r.URL)
				if err != nil {
					// todo better error handling
					return
				}

				// must close body after finish
				defer resp.Body.Close()

				// collect response status code
				status = resp.Status
				_, exist := r.StatusMap[status]
				if exist {
					r.StatusMap[status]++
				} else {
					r.StatusMap[status] = 1
				}

				// collect custom response header
				header = resp.Header.Get(r.Header)
				_, exist = r.ServerMap[header]
				if exist {
					r.ServerMap[header]++
				} else {
					r.ServerMap[header] = 1
				}
			}()
		}
		wg.Wait()
	}
}

func (r *XTop) layout(*gocui.Gui) error {
	maxX, maxY := r.G.Size()
	if v, err := r.G.SetView("center", 3, 0, maxX, maxY); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Frame = false
		r.display(v)
	}
	return nil
}

// updateView runs in a goroutine to periodically print out results to the screen
// it calls display() to actually print to a view
func (r *XTop) updateView() {
	for {
		time.Sleep(time.Second)
		v, err := r.G.View("center")
		if err != nil {
			panic(err)
		}
		v.Clear()
		r.display(v)
		r.G.Flush()
	}
}

// display prints output to the view
func (r *XTop) display(v io.Writer) error {

	output := fmt.Sprintf("Target: %s\n", r.URL)
	output += fmt.Sprintf("Header to check: %s\n", r.Header)
	output += fmt.Sprintf("Concurrent requests: %d\n\n", r.Concurrent)

	// Statuses
	sorted := sortMapByValue(r.StatusMap)
	output += fmt.Sprintf("=== Response status ===\n")
	for _, v := range sorted {
		output += fmt.Sprintf("%6s %8s %s\n",
			fmt.Sprintf("[%d%s]", v.Value*100/r.TotalRequestsSent, "%%"),
			fmt.Sprintf("[%d/%d]", v.Value, r.TotalRequestsSent),
			v.Key)
	}
	output += fmt.Sprintf("\n")

	// Custom header
	a := []string{}

	for k := range r.ServerMap {
		a = append(a, k)
	}
	sort.Strings(a)

	output += fmt.Sprintf("=== Response header %s ===\n", r.Header)
	for i, v := range a {
		output += fmt.Sprintf("%6s %8s %3d %s\n",
			fmt.Sprintf("[%d%s]", r.ServerMap[v]*100/r.TotalRequestsSent, "%%"),
			fmt.Sprintf("[%d/%d]", r.ServerMap[v], r.TotalRequestsSent),
			i+1,
			v)
	}
	output += fmt.Sprintf("\n")
	fmt.Fprintf(v, output)
	return nil
}
