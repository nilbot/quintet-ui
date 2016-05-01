package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/vdobler/chart"
	"github.com/vdobler/chart/imgg"
	"golang.org/x/net/websocket"
)

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", home)
	if *debug {
		http.HandleFunc("/resend", resend)
	}

	http.Handle("/echo", websocket.Handler(echoHandler))
	http.Handle("/incoming", websocket.Handler(incomingHandler))
	http.Handle("/watch", websocket.Handler(faithfulAudience))

	log.Fatal(http.ListenAndServe(*webListen, nil))
}

///////////////////////////////// VARIABLE /////////////////////////////////////
var (
	webListen = flag.String("listen", ":8080",
		"address to listen for HTTP/WebSockets on")
	domain = flag.String("domain", "demo.nilbot.net",
		"quintet-ui frontend")
	wsAddr = flag.String("ws", "demo.nilbot.net",
		"websocket endpoint, as seen by Quintet")
	debug = flag.Bool("debug", false,
		"enable debug features")
)

var (
	mu        sync.Mutex // guards clientMap
	clientMap = map[Client]bool{}
)

var uiTemplate = template.Must(template.ParseFiles("ui.html"))

type uiTemplateData struct {
	WSAddr string
	Domain string
}

// Client is browser user who will listen to result and fancy drawings
type Client chan *Message

var backlog []*Message

///////////////////////////////// HANDLER //////////////////////////////////////

func resend(w http.ResponseWriter, r *http.Request) {
	l := len(backlog)
	if l == 0 {
		return
	}
	m := backlog[l-1]
	for _, c := range clients() {
		c.Deliver(m)
	}
}
func echoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func incomingHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func home(w http.ResponseWriter, r *http.Request) {
	var err error
	if *debug {
		uiTemplate, err = template.ParseFiles("ui.html")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
	}
	err = uiTemplate.Execute(w, uiTemplateData{
		WSAddr: *wsAddr,
		Domain: *domain,
	})
	if err != nil {
		log.Println(err)
	}
}

///////////////////////////////// STRUCTS //////////////////////////////////////

// Message implements smtpd.Envelope by streaming the message to all
// connected websocket clients.
type Message struct {
	// HTML-escaped fields sent to the client
	Undefined    string
	UndefinedToo string
	Body         string // includes images (via data URLs)

	// internal state
	images []png
	bodies []string
	buf    bytes.Buffer // for accumulating email as it comes in
	msg    interface{}  // alternate message to send
}

type png struct {
	Type string
	Data []byte
}

// Stat is a JSON status message sent to clients when the number
// of connected WebSocket clients change.
type Stat struct {
	NumClients int
}

// ResultStat is a JSON status message sent to clients when the number
// of connected SMTP clients change.
type ResultStat struct {
	NumProjects int
	NumStudents int
}

// InputMeta has meta about input
type InputMeta struct {
	MessageType        string         `json:"MessageType"`
	NumberOfStudents   int            `json:"NumberOfStudents"`
	NumberOfProjects   int            `json:"NumberOfProjects"`
	HottestProject     string         `json:"hottestProject"`
	FreqListSorted     []Project      `json:"freqListSorted"`
	ProjectFrequencies map[string]int `json:"projectFrequencies"`
}

// Project mirrors quintet project type
type Project struct {
	ProjectName string `json:"projectName"`
}

// Student mirrors quintet student type
type Student struct {
	Name string `json:"Name"`
}

// Assignment mirrors quintet assignment type
type Assignment struct {
	Student         Student `json:"student"`
	AssignedProject Project `json:"assignedProject"`
}

// Result is the quintet result type
type Result struct {
	Assignments        []Assignment `json:"assignments"`
	Fitness            float64      `json:"fitness"`
	EnergyScore        int          `json:"energyScore"`
	IterationPerformed int          `json:"iterationPerformed,omitempty"`
	SolvingStrategy    string       `json:"solvingStrategy"`
	MessageType        string       `json:"MessageType"`
}

///////////////////////////////// FUNCTION /////////////////////////////////////

// clients returns all connected clients.
func clients() (cs []Client) {
	mu.Lock()
	defer mu.Unlock()
	for c := range clientMap {
		cs = append(cs, c)
	}
	return
}

// Deliver sends Message to clients
func (c Client) Deliver(m *Message) {
	select {
	case c <- m:
	default:
		// Client is too backlogged. They don't get this message.
	}
}

// remember client (for current session) as subscriber
func register(c Client) {
	mu.Lock()
	clientMap[c] = true
	n := len(clientMap)
	mu.Unlock()
	broadcast(&Message{msg: &Stat{NumClients: n}})
}

// remove client from session
func unregister(c Client) {
	mu.Lock()
	delete(clientMap, c)
	n := len(clientMap)
	mu.Unlock()
	broadcast(&Message{msg: &Stat{NumClients: n}})
}

// broadcast to all subscribers
func broadcast(m *Message) {
	for _, c := range clients() {
		c.Deliver(m)
	}
}

/////////////////////////////// CORE ///////////////////////////////////////////
func faithfulAudience(ws *websocket.Conn) {
	log.Printf("websocket connection from %v", ws.RemoteAddr())
	client := Client(make(chan *Message, 100))
	register(client)
	defer unregister(client)

	deadc := make(chan bool, 1)

	// Wait for incoming messages. Don't really care about them, but
	// use this to find out if client goes away.
	go func() {
		var msg Message
		for {
			err := websocket.JSON.Receive(ws, &msg)
			switch err {
			case nil:
				log.Printf("Unexpected message from %v: %+v",
					ws.RemoteAddr(), msg)
				continue
			case io.EOF:
			default:
				log.Printf("Receive error from %v: %v",
					ws.RemoteAddr(), err)
			}
			deadc <- true
		}
	}()

	for {
		select {
		case <-deadc:
			return
		case m := <-client:
			var err error
			if m.msg != nil {
				err = websocket.JSON.Send(ws, m.msg)
			} else {
				err = websocket.JSON.Send(ws, m)
			}
			if err != nil {
				return
			}
		}
	}
}

/////////////////////////////// GRAPH //////////////////////////////////////////

type imageBuilder struct {
	N, M, W, H, Cnt int
	I               *image.RGBA
}

func (im *InputMeta) getChart(name string, n, m, w, h int) (*image.RGBA, error) {
	if im.MessageType != "InputMeta" {
		return nil, errors.New("content mismatch: input is not InputMeta")
	}

	d := newImageBuilder(name, n, m, w, h)

	wuc := chart.BarChart{Title: "Input Metadata: Project Popularity"}
	p := im.FreqListSorted
	wuc.YRange.ShowZero = true
	wuc.XRange.Label, wuc.YRange.Label = "Project (name)", "Popularity"
	wuc.Key.Pos = "otc"
	var x []string
	var y []float64
	var xIdx []float64
	for i, v := range p {
		x = append(x, v.ProjectName)
		xIdx = append(xIdx, float64(i))
		y = append(y, float64(im.ProjectFrequencies[v.ProjectName]))
	}

	wuc.XRange.Category = x
	blue := chart.Style{
		Symbol: '#', LineColor: color.NRGBA{
			0x00, 0x00, 0xff, 0xff,
		},
		LineWidth: 4,
		FillColor: color.NRGBA{
			0x40, 0x40, 0xff, 0xff,
		},
	}
	wuc.AddDataPair("Popularity", xIdx, y, blue)
	return d.plot(&wuc), nil
}

func newImageBuilder(name string, n, m, w, h int) *imageBuilder {
	rst := imageBuilder{N: n, M: m, W: w, H: h}

	rst.I = image.NewRGBA(image.Rect(0, 0, n*w, m*h))
	bg := image.NewUniform(color.RGBA{0xff, 0xff, 0xff, 0xff})
	draw.Draw(rst.I, rst.I.Bounds(), bg, image.ZP, draw.Src)

	return &rst
}

func (d *imageBuilder) plot(c chart.Chart) *image.RGBA {
	row, col := d.Cnt/d.N, d.Cnt%d.N
	igr := imgg.AddTo(d.I, col*d.W, row*d.H, d.W, d.H, color.RGBA{
		0xff, 0xff, 0xff, 0xff}, nil, nil)
	c.Plot(igr)
	return d.I
}
