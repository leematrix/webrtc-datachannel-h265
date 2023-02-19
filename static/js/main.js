'use strict'
import { Canvas2DRenderer } from './renderer_2d.js'
import { Module } from '../fmp4-muxer.js'

//webrtc方式
const callButton = document.getElementById('callButton')

callButton.addEventListener('click', call)

let ws = null
let pc = null
let sendChannel = null

// 网页加载后连接WebSocket服务
window.onload = connectWS()

const log = msg => {
    console.log(msg)
}

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
function sendWS(type, payload, binary) {
    console.log('***SEND')

    var data = {}
    data["type"] = type
    data["payload"] = payload
    data["binary"] = binary

    //console.log(JSON.stringify(data))
    ws.send(JSON.stringify(data))
}

// 发送数据到WebSocket服务
function sendWSBinary(binary) {
    console.log('***SEND')

    //console.log(JSON.stringify(data))
    ws.send(binary)
}

// 关闭到WebSocket服务的连接
function closeWS() {
    console.log('***CLOSE')
    ws.close()
}

// Rendering. Drawing is limited to once per animation frame.
const canvas = document.querySelector("canvas").transferControlToOffscreen();
let renderer = new Canvas2DRenderer(canvas);
let pendingFrame = null;
let startTime = null;
let frameCount = 0;

function renderFrame(frame) {
    if (!pendingFrame) {
        // Schedule rendering in the next animation frame.
        requestAnimationFrame(renderAnimationFrame);
    } else {
        // Close the current pending frame before replacing it.
        pendingFrame.close();
    }
    // Set or replace the pending frame.
    pendingFrame = frame;
}

function renderAnimationFrame() {
    renderer.draw(pendingFrame);
    pendingFrame = null;
}

let pendingStatus = null;

function setStatus(type, message) {
    if (pendingStatus) {
        pendingStatus[type] = message;
    } else {
        pendingStatus = {[type]: message};
        self.requestAnimationFrame(statusAnimationFrame);
    }
}

function statusAnimationFrame() {
    self.postMessage(pendingStatus);
    pendingStatus = null;
}

// Set up a VideoDecoer.
const decoder = new VideoDecoder({
    output(frame) {
        // Update statistics.
        if (startTime == null) {
            startTime = performance.now();
        } else {
            const elapsed = (performance.now() - startTime) / 1000;
            const fps = ++frameCount / elapsed;
            setStatus("render", `${fps.toFixed(0)} fps`);
        }

        // Schedule the frame to be rendered.
        renderFrame(frame);
    },
    error(e) {
        setStatus("decode", e);
    }
});

const description = [1, 1, 96, 0, 0, 0, 176, 0, 0, 0, 0, 0, 123, 240, 0, 252, 253, 248, 248, 0, 0, 3, 3, 32, 0, 1, 0, 23, 64, 1, 12, 1, 255, 255, 1, 96, 0, 0, 3, 0, 176, 0, 0, 3, 0, 0, 3, 0, 123, 172, 9, 33, 0, 1, 0, 34, 66, 1, 1, 1, 96, 0, 0, 3, 0, 176, 0, 0, 3, 0, 0, 3, 0, 123, 160, 3, 192, 128, 16, 229, 141, 174, 73, 50, 244, 220, 4, 4, 4, 2, 34, 0, 1, 0, 7, 68, 1, 192, 242, 240, 60, 144]

const config = {
    codec: "hvc1.1.6.L123.b0",
    codedWidth: 1920,
    codedHeight: 1080,
    description: new Uint8Array(description),
};
decoder.configure(config);
console.log("decoder init done.")

let buffer = [];
let bufferLen = 0
let index = 0
let bytesToInt2 = function(bytes, off) {
    let b0 = bytes[off] & 0xFF;
    let b1 = bytes[off + 1] & 0xFF;
    let b2 = bytes[off + 2] & 0xFF;
    let b3 = bytes[off + 3] & 0xFF;
    return (b0 << 24) | (b1 << 16) | (b2 << 8) | b3;
}

if (!window.MediaSource) {
    console.error('No Media Source API available');
}

let ms = new MediaSource();
let video = document.querySelector('video');
video.src = window.URL.createObjectURL(ms);
ms.addEventListener('sourceopen', onMediaSourceOpen);
let sourceBuffer
function onMediaSourceOpen() {
    sourceBuffer = ms.addSourceBuffer('video/mp4; codecs="hev1.1.2.L153"')
    //sourceBuffer = ms.addSourceBuffer('video/mp4; codecs="hvc1.4.10.L90.9d.8,mp4a.40.2"')
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
    //sendChannel.binaryType = "blob"
    //sendChannel.bufferedAmountLowThreshold = 65535
    //sendChannel.ordered = false

    var videoCallback = Module.addFunction(function (d, d_size){
        console.log("received fmp4 data.")
        var data = new Uint8Array(d_size)
        for(var i=0; i<d_size; i++){
            data[i] = Module.HEAPU8[d+i]
            //console.log("%s ", data[i].toString(16))
        }
        //sendWSBinary(data)
        sourceBuffer.appendBuffer(data)
    }, 'vii');
    Module._open_muxer(videoCallback, 0)
    sendChannel.onclose = () => console.log('sendChannel has closed')
    sendChannel.onopen = () => console.log('sendChannel has opened')
    sendChannel.onmessage = e => {
        console.log(`Message from DataChannel '${sendChannel.label}' payload '${e.data}'`)
        const typedArray = new Uint8Array(e.data);
        const size = typedArray.length;
        const cacheBuffer = Module._malloc(size);
        Module.HEAPU8.set(typedArray, cacheBuffer);
        Module._mux_data(cacheBuffer, size)
        if (cacheBuffer != null) {
            Module._free(cacheBuffer);
        }
        console.log("receive source buffer.")
    }
    video.play()
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
