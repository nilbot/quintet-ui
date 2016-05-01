package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
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
	wsAddr = flag.String("ws", "ws.nilbot.net",
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

///////////////////////////////// LOGGER ///////////////////////////////////////

// Logger logs
func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inner.ServeHTTP(w, r)
		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

///////////////////////////////// ERRORS ///////////////////////////////////////

type jsonErr struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

///////////////////////////////// ROUTES ///////////////////////////////////////

// Route struct
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes lots of route
type Routes []Route

var routes = Routes{
	Route{
		"Home",
		"GET",
		"/",
		home,
	},
}

// NewRouter gives a nice mux.Router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	return router
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
	images []image
	bodies []string
	buf    bytes.Buffer // for accumulating email as it comes in
	msg    interface{}  // alternate message to send
}

type image struct {
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
