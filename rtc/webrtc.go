package rtc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"os"
	"time"
)

const (
	independent    = 0
	start          = 1
	mid            = 2
	end            = 3
	maxMessageSize = 65000
)

const (
	ADD_VIDEO    = 0
	ADD_AUDIO    = 1
	INIT_SEGMENT = 2
	SAVE_SEGMENT = 3
	VIDEO_FRAME  = 4
	AUDIO_FRAME  = 5
)

type ChannelMessage struct {
	Type    uint8  `json:"type"`
	Payload []byte `json:"payload"`
}

func SpliceChannelMessage(data []byte, dts uint32) (msgList []ChannelMessage) {
	dataLen := len(data)
	count := dataLen / maxMessageSize
	for j := 0; j <= count; j++ {
		msg := ChannelMessage{}
		if j == 0 {
			msg.Payload = make([]byte, 10+maxMessageSize)
			msg.Payload[0] = VIDEO_FRAME
			msg.Payload[1] = start
			binary.BigEndian.PutUint32(msg.Payload[2:], dts)
			copy(msg.Payload[6:], data[j*maxMessageSize:])
		} else if j == count {
			msg.Payload = make([]byte, 2+dataLen%maxMessageSize)
			msg.Payload[0] = VIDEO_FRAME
			msg.Payload[1] = end
			copy(msg.Payload[2:], data[j*maxMessageSize:])
		} else {
			msg.Payload = make([]byte, 2+maxMessageSize)
			msg.Payload[0] = VIDEO_FRAME
			msg.Payload[1] = mid
			copy(msg.Payload[2:], data[j*maxMessageSize:])
		}
		msgList = append(msgList, msg)
	}
	return
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
			open, err := os.Open("/Users/lzx/Movies/AV/h265/ipc.h265")
			if err != nil {
				fmt.Printf("Open h265 file failed, err:%v", err)
				return
			}
			buf := make([]byte, 4*1024*1024)
			n, err := open.Read(buf)
			if err != nil {
				fmt.Printf("Read h265 buf failed, err:%v", err)
				return
			}
			hevc := H265Parser(buf[:n])
			//1.添加video track
			sendBuf := make([]byte, maxMessageSize)
			sendBuf[0] = ADD_VIDEO
			binary.BigEndian.PutUint16(sendBuf[1:], 1920)
			binary.BigEndian.PutUint16(sendBuf[3:], 1080)
			copy(sendBuf[5:], hevc.ps)
			sendErr := d.Send(sendBuf[:5+len(hevc.ps)])
			if sendErr != nil {
				fmt.Println("add video track", sendErr)
				return
			}

			//2.init segment
			sendBuf[0] = INIT_SEGMENT
			sendErr = d.Send(sendBuf[:1])
			if sendErr != nil {
				fmt.Println("init segment", sendErr)
				return
			}
			time.Sleep(500 * time.Millisecond)

			//3.send video frame
			for i := 0; i < len(hevc.naluList); i++ {
				if hevc.naluList[i].Type == HEVC_NAL_VPS {
					//save segment
					sendBuf[0] = SAVE_SEGMENT
					sendErr = d.Send(sendBuf[:1])
					if sendErr != nil {
						fmt.Println("save segment", sendErr)
						return
					}
					time.Sleep(500 * time.Millisecond)
				}

				payloadLen := len(hevc.naluList[i].Payload)
				sendLen := 0
				if payloadLen > maxMessageSize {
					msgList := SpliceChannelMessage(hevc.naluList[i].Payload, hevc.naluList[i].Dts)
					for _, msg := range msgList {
						sendErr = d.Send(msg.Payload)
						if sendErr != nil {
							fmt.Println("send un independent video frame", sendErr)
							return
						}
					}
				} else {
					sendBuf[0] = VIDEO_FRAME
					sendBuf[1] = independent
					binary.BigEndian.PutUint32(sendBuf[2:], hevc.naluList[i].Dts)
					copy(sendBuf[6:], hevc.naluList[i].Payload)
					sendLen = 6 + payloadLen

					sendErr = d.Send(sendBuf[:sendLen])
					if sendErr != nil {
						fmt.Println("send independent video frame", sendErr)
						return
					}
				}
			}

			//4.save segment
			sendBuf[0] = SAVE_SEGMENT
			sendErr = d.Send(sendBuf[:1])
			if sendErr != nil {
				fmt.Println("save segment", sendErr)
				return
			}
			time.Sleep(500 * time.Millisecond)
			fmt.Printf("Send h265 stream done.")
			return
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
