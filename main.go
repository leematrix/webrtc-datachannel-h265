package main

import (
	"github.com/leematrix/webrtc-datachannel-h265/http"
	"github.com/leematrix/webrtc-datachannel-h265/websocket"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(1)

	go http.ListenAndServe()

	go websocket.ListenAndServe()

	wg.Wait()
}
