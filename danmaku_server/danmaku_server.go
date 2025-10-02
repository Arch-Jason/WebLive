package danmaku_server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有连接
		return true
	},
}

func removeConn(conn *websocket.Conn) {
	for i, c := range ws_conns {
		if c == conn {
			ws_conns = append(ws_conns[:i], ws_conns[i+1:]...)
			break
		}
	}
}

var ws_conns = []*websocket.Conn{}
var msg_type = make(chan int)
var msg = make(chan []byte)

func InitHttpRoutes(http_mux *http.ServeMux) {
	http_mux.HandleFunc("/danmaku", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		ws_conns = append(ws_conns, conn)
		fmt.Println("New Websocket connection, total:", len(ws_conns))
		go func(c *websocket.Conn) {
			defer func() {
				fmt.Println("One Websocket connection closed")
				removeConn(c)
				c.Close()
			}()

			for {
				messageType, p, err := c.ReadMessage()
				if err != nil {
					return
				}
				msg_type <- messageType
				msg <- p
			}
		}(conn)
	})
}

func RxAndBroadcast() {
	for {
		messageType := <-msg_type
		msg := <-msg
		fmt.Printf("New message: %s\n", string(msg))
		for _, conn := range ws_conns {
			if err := conn.WriteMessage(messageType, msg); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
