package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

func WebsocketAndHTML() {

	// 设置静态文件服务器，托管位于 "static" 目录下的文件
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)
	log.Println("web 服务部署在 http://localhost:9998 ")
	log.Fatal(http.ListenAndServe(":9998", nil))

}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// 注册客户端
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	for {
		// 等待接收新消息，但这里我们不做处理
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
	}
}

func SendWebsocket(message map[string]string) {
	// 这里我们使用一个简单的死循环来模拟消息发送
	// 在实际应用中，你可能会根据特定的逻辑来触发消息发送
	clientsMu.Lock()
	for conn := range clients {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Println("Write error:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
	clientsMu.Unlock()
}
