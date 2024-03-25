package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/watch"
)

type WebSocketHandler interface {
	WithRequest(*http.Request, http.ResponseWriter) WebSocketHandler
	WithWatchInterface(func() (watch.Interface, error)) WebSocketHandler
	WithUpgrader(websocket.Upgrader) WebSocketHandler
	Handle(<-chan struct{}) (bool, error)
}

func NewDefaultWebSocketHandler() WebSocketHandler {
	return &defaultWebSocketHandler{}
}

type defaultWebSocketHandler struct {
	request     *http.Request
	writer      http.ResponseWriter
	watchGetter func() (watch.Interface, error)
	upgrader    websocket.Upgrader
}

func (h *defaultWebSocketHandler) WithRequest(request *http.Request, writer http.ResponseWriter) WebSocketHandler {
	h.request = request
	h.writer = writer
	return h
}

func (h *defaultWebSocketHandler) WithWatchInterface(watchGetter func() (watch.Interface, error)) WebSocketHandler {
	h.watchGetter = watchGetter
	return h
}

func (h *defaultWebSocketHandler) WithUpgrader(upgrader websocket.Upgrader) WebSocketHandler {
	h.upgrader = upgrader
	return h
}

func (h *defaultWebSocketHandler) Handle(done <-chan struct{}) (ok bool, err error) {
	if !h.isWebSoket() {
		return
	}

	var conn *websocket.Conn
	conn, err = h.upgrader.Upgrade(h.writer, h.request, nil)
	if err != nil {
		return
	}

	ok = true
	var watchInter watch.Interface
	if watchInter, err = h.watchGetter(); err != nil {
		return
	}

	go func() {
		for {
			select {
			case e := <-watchInter.ResultChan():
				if e.Type != "" {
					conn.WriteJSON(e)
				}
				if e.Type == watch.Error {
					watchInter.Stop()
					return
				}
			case <-done:
				watchInter.Stop()
				fmt.Println("canceled")
				return
			}
		}
	}()
	return
}

func (h *defaultWebSocketHandler) isWebSoket() bool {
	return h.request.Header.Get("Connection") == "Upgrade"
}
