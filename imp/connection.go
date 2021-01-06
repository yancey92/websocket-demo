package imp

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

// 封装一个websocket连接。
// 实现原理：
//		会启动读协程循环的读取 websocket，将消息投递到 inChan
//		会启动写协程循环的读取 outChan，将消息投递给 websocket
type Connection struct {
	wsConn  *websocket.Conn // websocket 长连接
	inChan  chan []byte     // websocket 接收到的消息
	outChan chan []byte     // websocket 要发送的消息
	// 定义 closeChan：如果在消息传递过程中出现错误，就会触发 Close 方法，
	// 如果 closeChan 被关闭，就说明 websocket 底层连接出错，此时要关闭该 Connection 的所有连接并退出 select 语句
	closeChan chan []byte

	mutex    sync.Mutex
	isClosed bool
}

func InitConnection(wsConn *websocket.Conn) (conn *Connection, err error) {
	conn = &Connection{
		wsConn:    wsConn,
		inChan:    make(chan []byte, 1000), // 最大1000条消息，如果积累的消息大于1000，消息就会被丢弃
		outChan:   make(chan []byte, 1000),
		closeChan: make(chan []byte, 1),
	}
	go conn.readLoop()  // 启动读协程，将 websocket 收到的数据，存放到 inChan
	go conn.writeLoop() // 启动写协程，将 outChan 收到的数据，通过 websocket 发出去
	return
}

// ------------------------Connection 的API，提供Send、Read、Close等线程安全的接口------------------------

func (conn *Connection) Send(data []byte) (err error) {
	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

func (conn *Connection) Read() (data []byte, err error) {
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

func (conn *Connection) Close() {
	conn.wsConn.Close() // websocket 的close()是线程安全的，并且可以多次被调用

	// 要保证下面的 closeChan 只被关闭一次，且是线程安全的
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan) // 关闭 closeChan，readLoop 和 writeLoop 的 select 语句就会退出
		conn.isClosed = true
	}
	conn.mutex.Unlock()
}

// ------------------------

// 不停的从websocket中读取数据，
func (conn *Connection) readLoop() {
	var (
		data []byte
		err  error
	)
	for {
		if _, data, err = conn.wsConn.ReadMessage(); err != nil {
			goto ERR
		}
		select {
		case conn.inChan <- data:
		case <-conn.closeChan: // 如果closeChan被关闭，就会关闭此处的阻塞
			goto ERR
		}

	}

ERR:
	conn.Close()
}

// 不停的websocket中读取数据，
func (conn *Connection) writeLoop() {
	var (
		data []byte
		err  error
	)
	for {
		select {
		case data = <-conn.outChan:
			if err = conn.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
				goto ERR
			}
		case <-conn.closeChan: // 如果closeChan被关闭，就会关闭此处的阻塞
			goto ERR
		}
	}

ERR:
	conn.Close()
}
