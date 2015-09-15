package web_logs

import (
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"sync"
)

//LOGS
var GCMActiveClients = make(map[ClientConn]int)
var GCMActiveClientsRWMutex sync.RWMutex

var APNSActiveClients = make(map[ClientConn]int)
var APNSActiveClientsRWMutex sync.RWMutex

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ClientConn struct {
	websocket *websocket.Conn
	clientIP  net.Addr
}

func AddGCMClient(cc ClientConn) {
	GCMActiveClientsRWMutex.Lock()
	GCMActiveClients[cc] = 0
	GCMActiveClientsRWMutex.Unlock()
}

func AddAPNSClient(cc ClientConn) {
	APNSActiveClientsRWMutex.Lock()
	APNSActiveClients[cc] = 0
	APNSActiveClientsRWMutex.Unlock()
}

func DeleteGCMClient(cc ClientConn) {
	GCMActiveClientsRWMutex.Lock()
	delete(GCMActiveClients, cc)
	GCMActiveClientsRWMutex.Unlock()
}

func DeleteAPNSClient(cc ClientConn) {
	APNSActiveClientsRWMutex.Lock()
	delete(APNSActiveClients, cc)
	APNSActiveClientsRWMutex.Unlock()
}

func GCMLogs(log string) {
	GCMActiveClientsRWMutex.RLock()
	defer GCMActiveClientsRWMutex.RUnlock()

	for client, _ := range GCMActiveClients {
		if err := client.websocket.WriteMessage(1, []byte(log)); err != nil {
			return
		}
	}
}

func APNSLogs(log string) {
	APNSActiveClientsRWMutex.RLock()
	defer APNSActiveClientsRWMutex.RUnlock()

	for client, _ := range APNSActiveClients {
		if err := client.websocket.WriteMessage(1, []byte(log)); err != nil {
			return
		}
	}
}

func SockGCM(w http.ResponseWriter, r *http.Request) {
	ws, err := Upgrader.Upgrade(w, r, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	client := ws.RemoteAddr()
	sockCli := ClientConn{ws, client}
	AddGCMClient(sockCli)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			DeleteGCMClient(sockCli)
			return
		}
	}
}

func SockAPNS(w http.ResponseWriter, r *http.Request) {
	ws, err := Upgrader.Upgrade(w, r, nil)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Println(err)
		return
	}
	client := ws.RemoteAddr()
	sockCli := ClientConn{ws, client}
	AddAPNSClient(sockCli)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			DeleteAPNSClient(sockCli)
			return
		}
	}
}
