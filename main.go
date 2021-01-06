package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"websocket-demo/imp"
)

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 0,
		ReadBufferSize:   0,
		WriteBufferSize:  0,
		WriteBufferPool:  nil,
		Subprotocols:     nil,
		Error:            nil,
		// 在请求的时候，遇到跨域问题是允许跨域的
		CheckOrigin:       func(r *http.Request) bool { return true },
		EnableCompression: false,
	}
)

// 处理websocket通信
func WSHandler(rw http.ResponseWriter, r *http.Request) {
	var (
		wsConn *websocket.Conn
		conn   *imp.Connection
		err    error
		data   []byte
	)

	// 下面这一步会给 http 请求的 response header 中返回 "Upgrade: websocket"；并返回一个 websocket 长连接
	if wsConn, err = upgrader.Upgrade(rw, r, nil); err != nil {
		// 如果出错，websocket连接默认就会被中断掉
		log.Println(err)
		return
	}

	// 如果得到的websocket没有出错，那么就可以进行 websocket 长连接消息的收发
	if conn, err = imp.InitConnection(wsConn); err != nil {
		goto ERR
	}

	for {
		if data, err = conn.Read(); err != nil {
			goto ERR
		}
		if err = conn.Send(data); err != nil {
			goto ERR
		}
	}
ERR:
	conn.Close() // 关闭连接的操作
}

func main() {
	http.HandleFunc("/ws", WSHandler)        // 配一个 http 协议的路由；当请求 "/ws" 接口时，WSHandler 这个回调函数会被调用
	http.ListenAndServe("0.0.0.0:8080", nil) // 监听在本机所有网卡的 8080 端口
}
