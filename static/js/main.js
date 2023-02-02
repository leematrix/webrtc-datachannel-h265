'use strict'
//webrtc方式
const callButton = document.getElementById('callButton')

callButton.addEventListener('click', call)

// Web端WebSocket连接对象
let ws = null

let pc = null
let sendChannel = null

// webRTC的启动时间
let startTime

// 网页加载后连接WebSocket服务
window.onload = connectWS()

const remoteVideo = document.getElementById('remoteVideo')
const remoteAudio = document.getElementById('remoteAudio')

const log = msg => {
    console.log(msg)
}

remoteVideo.addEventListener('loadedmetadata', function () {
    console.log(`Remote video videoWidth: ${this.videoWidth}px,  videoHeight: ${this.videoHeight}px`)
})

remoteVideo.addEventListener('resize', () => {
    console.log(`Remote video size changed to ${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`)
    // We'll use the first onsize callback as an indication that video has started playing out
    if (startTime) {
        const elapsedTime = window.performance.now() - startTime
        console.log('Setup time: ' + elapsedTime.toFixed(3) + 'ms')
        startTime = null
    }
})

// 连接WebSocket服务
async function connectWS() {
    if (ws == null) {
        ws = new WebSocket("ws://127.0.0.1:8888/webrtc", "json")
        console.log('***CREATED WEBSOCKET')
    }

    ws.onopen = function (evt) {
        console.log('***ONOPEN')
    }

    // 注册WebSocket的消息回调处理函数
    ws.onmessage = function (evt) {
        console.log('***ONMESSAGE')
        console.log(evt.data)
        console.log(JSON.parse(evt.data))

        parseResponse(JSON.parse(evt.data))
    }
}

// 发送数据到WebSocket服务
function sendWS(type, payload) {
    console.log('***SEND')

    var data = {}
    data["type"] = type
    data["payload"] = payload

    console.log(JSON.stringify(data))
    ws.send(JSON.stringify(data))
}

// 关闭到WebSocket服务的连接
function closeWS() {
    console.log('***CLOSE')
    ws.close()
}

// 启动webRTC业务
async function call() {
    console.log('call')

    startTime = window.performance.now()

    if (pc !== null){
        pc.close()
    }

    pc = new RTCPeerConnection()
    console.log('Created remote peer connection object pc')

    pc.oniceconnectionstatechange = e => log(pc.iceConnectionState)
    pc.onicecandidate = event => {
        if (event.candidate === null) {
            sendOffer(JSON.stringify(pc.localDescription))
        }
    }
    pc.onnegotiationneeded = e =>
        pc.createOffer().then(d => pc.setLocalDescription(d)).catch(log)

    if (sendChannel !== null){
        sendChannel.close()
    }
    sendChannel = pc.createDataChannel('video')
    sendChannel.binaryType = "arraybuffer"
    //sendChannel.bufferedAmountLowThreshold = 65535
    //sendChannel.ordered = false

    sendChannel.onclose = () => console.log('sendChannel has closed')
    sendChannel.onopen = () => console.log('sendChannel has opened')
    sendChannel.onmessage = e => {
        console.log(`Message from DataChannel '${sendChannel.label}' payload '${e.data}'`)

    }
}

// WebSocket消息回调处理函数
function parseResponse(response) {
    console.log("Response:", response)

    if (response.success !== true) {
        console.log("response not success")
        return
    }

    if (response.type === 'answer') {
        console.log("get answer from OpenAPI")

        var answer = {}
        answer["type"] = "answer"
        answer["sdp"] = response.payload
        pc.setRemoteDescription(answer)
    }  else if (response.type === 'disconnect') {
        console.log("get disconnect from OpenAPI")

        sendDisconnect()
    }
}

async function sendOffer(sdp) {
    // shorter sdp, remove a=extmap... line, device ONLY allow 8KB json payload
    sdp = sdp.replace(/\r\na=extmap[^\r\n]*/g, '')

    console.log("send offer: " + sdp)

    try {
        sendWS("offer", sdp)
    } catch (e) {
        console.log("send offer via WebSocket fail: " + e.name)
    }
}

async function sendDisconnect() {
    console.log("hangup")

    pc.close()

    try {
        sendWS("disconnect", "")
    } catch (e) {
        console.log("hangup the call fail: " + e.name)
    }
}
