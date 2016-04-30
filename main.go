package main

import (
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

func echoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func incomingHandler(ws *websocket.Conn) {

}

func reportingHandler(ws *websocket.Conn) {

}

func main() {
	http.Handle("/echo", websocket.Handler(echoHandler))
	http.Handle("/incoming", websocket.Handler(incomingHandler))
	http.Handle("/reporting", websocket.Handler(reportingHandler))
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
