package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/leematrix/webrtc-datachannel-h265/rtc"
	"net/http"
	"os"
	"strconv"
)

// WsMessage WebSocket通道收发消息的顶层结构体
type WsMessage struct {
	Type    string          `json:"type"`              // 消息类型，为offer、candidate、answer
	Payload string          `json:"payload,omitempty"` // WebSocket承载的消息内容
	Binary  json.RawMessage `json:"binary,omitempty"`

	Success bool `json:"success,omitempty"` // 标志WebSocket请求是否成功，仅给Web客户端回复时有效
}

// 此Sample下WebSocket传输的是json，Message Type为1(Text)
// 此Sample关闭了请求源地址检查，生产环境中应该开启
var upgrader = websocket.Upgrader{
	Subprotocols: []string{"json"},
	CheckOrigin:  checkOrigin,
}

// 关闭请求源地址检查
func checkOrigin(r *http.Request) bool {
	return true
}

// ListenAndServe 提供WebSocket服务入口/webrtc，由HTTP协议升级到WebSocket
func ListenAndServe() {
	http.HandleFunc("/webrtc", webrtc)

	fmt.Print("websocket server listen on :8888...\n")

	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		fmt.Printf("websocket serve fail: %s", err.Error())
	}
}

var fmp4Index = 0

// WebSocket的连接处理函数，在Golang中每个连接独享自己的协程（类似C++/Java中线程，更轻量化）
func webrtc(w http.ResponseWriter, r *http.Request) {
	// 升级连接协议到WebSocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("upgrade to websocket fail: %s", err.Error())

		return
	}
	defer c.Close()

	fmt.Printf("new ws client, addr: %s", r.RemoteAddr)

	wRtc := rtc.SessionWebrtc{}
	defer wRtc.Close()

	// 从WebSocket连接轮询消息
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Printf("ws read fail: %s", err.Error())

			break
		}

		//fmt.Printf("ws recv: %s", string(message))

		msg := &WsMessage{}
		err = json.Unmarshal(message, msg)
		if err != nil {
			fmt.Printf("unmarshal ws message fail: %s", err.Error())

			name := strconv.Itoa(fmp4Index) + ".mp4"
			err := os.WriteFile(name, message, 0666)
			if err != nil {
				fmt.Printf("write fmp4 file[%s], err:%v\n", name, err)
				break
			}
			fmp4Index++
			fmt.Printf("write fmp4 file[%s] ok\n", name)
			continue
		}
		if msg.Type == "offer" {
			wRtc.Close()
			wRtc.Init()
			err, answer := wRtc.SetOffer(msg.Payload)
			if err != nil {
				fmt.Printf("set offer fail: %s", err.Error())
				break
			}

			resp := &WsMessage{
				Type:    "answer",
				Payload: answer,
				Success: true,
			}
			sendBytes, err := json.Marshal(resp)
			if err != nil {
				fmt.Printf("marshal WsMessage fail: %s", err.Error())
				return
			}

			err = c.WriteMessage(websocket.TextMessage, sendBytes)
			if err != nil {
				fmt.Printf("ws write fail: %s", err.Error())
				break
			}
		}
	}
}
