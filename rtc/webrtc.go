package rtc

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"os"
	"time"
)

const (
	independent    = 0
	fragmentBegin  = 1
	fragmentMid    = 2
	fragmentEnd    = 3
	maxMessageSize = 65535
)

type ChannelMessageType struct {
	Type    uint8  `json:"type"`
	Payload []byte `json:"payload"`
}

type SessionWebrtc struct {
	peerConn *webrtc.PeerConnection
}

func (s *SessionWebrtc) Init() {
	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. H265 stream will now be sent to any connected DataChannels.\n", d.Label(), d.ID())
			open, err := os.Open("/Users/lzx/Documents/nginx/html/ipc.h265")
			if err != nil {
				fmt.Printf("Open h265 file failed, err:%v", err)
				return
			}
			buf := make([]byte, 2*1024*1024)
			n, err := open.Read(buf)
			if err != nil {
				fmt.Printf("Read h265 buf failed, err:%v", err)
				return
			}
			if n < 4 {
				fmt.Printf("len h265 too short, len:%d", n)
				return
			}
			frameCount := 0
			start := 0
			for i := 4; i < n; i++ {
				if buf[i-1] == 0x01 && buf[i-2] == 0 && buf[i-3] == 0 && buf[i-4] == 0 &&
					(buf[i] == 0x26 || buf[i] == 0x02) && i != 4 {
					dataLen := i - 4 - start
					//fmt.Printf("dataLen:%d\n", dataLen)
					if dataLen > maxMessageSize {
						partNumber := dataLen / (maxMessageSize - 1)
						for j := 0; j < partNumber; j++ {
							newStart := start + j*(maxMessageSize-1)
							var newBuf []byte
							if j == 0 {
								newBuf = make([]byte, maxMessageSize)
								newBuf[0] = 1
								copy(newBuf[1:], buf[newStart:newStart+maxMessageSize-1])
							} else if j == partNumber {
								newBuf = make([]byte, dataLen%(maxMessageSize-1)+1)
								newBuf[0] = 3
								copy(newBuf[1:], buf[newStart:dataLen%(maxMessageSize-1)])
							} else {
								newBuf = make([]byte, maxMessageSize)
								newBuf[0] = 2
								copy(newBuf[1:], buf[newStart:newStart+(maxMessageSize-1)])
							}
							//fmt.Printf("newBuf len:%d\n", len(newBuf))
							sendErr := d.Send(newBuf)
							if sendErr != nil {
								fmt.Println(sendErr)
								return
							}
						}
					} else {
						newBuf := make([]byte, dataLen+1)
						newBuf[0] = 0
						copy(newBuf[1:], buf[start:start+dataLen])
						//fmt.Printf("newBuf len:%d\n", len(newBuf))
						sendErr := d.Send(newBuf)
						if sendErr != nil {
							fmt.Println(sendErr)
							return
						}
					}

					fmt.Printf("send %d frame.\n", frameCount)
					frameCount++
					start = i - 3
					time.Sleep(40 * time.Millisecond)
				}
			}
			fmt.Printf("Send h265 stream done.")
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})
	s.peerConn = peerConnection
}

func (s *SessionWebrtc) SetOffer(data string) (error, string) {
	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	err := json.Unmarshal([]byte(data), &offer)
	if err != nil {
		panic(err)
	}
	fmt.Println(offer)
	// Set the remote SessionDescription
	err = s.peerConn.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := s.peerConn.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(s.peerConn)

	// Sets the LocalDescription, and starts our UDP listeners
	err = s.peerConn.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	localDesc := *s.peerConn.LocalDescription()

	fmt.Println(localDesc.SDP)
	return nil, localDesc.SDP
}

func (s *SessionWebrtc) Close() {
	if s.peerConn != nil {
		s.peerConn.Close()
	}
}
