package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type MessageHandler func(string)

type Notifications struct {
	url   string
	token string
	conn  *websocket.Conn
	done  chan struct{}
	exit  chan struct{}
	hdl   MessageHandler
}

func NewNotifications(url, token string) *Notifications {
	return &Notifications{
		url:   url,
		token: token,
		done:  make(chan struct{}),
		exit:  make(chan struct{}),
	}
}

func (n *Notifications) Connect() error {
	// TODO: if already connected, return error
	n.conn = nil

	fmt.Printf("Connecting to notifications server at %q\n", n.url)

	auth := http.Header{}
	auth.Set("Authorization", "Bearer "+n.token)
	conn, res, err := websocket.DefaultDialer.Dial(n.url, auth)
	if err != nil {
		return fmt.Errorf("websocket connection failed with status %v, error %v", res.StatusCode, err)
		return err
	}

	n.conn = conn
	n.done = make(chan struct{})
	n.exit = make(chan struct{})

	go n.loop()
	go n.read()

	return nil
}

func (n *Notifications) Disconnect() {
	close(n.exit)
}

func (n *Notifications) onDisconnected() {
	fmt.Println("Notifications disconnected")
	// TODO: Lock
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}

}

func (n *Notifications) loop() {
	fmt.Println("Start write loop...")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer n.onDisconnected()

	for {
		select {
		case <-n.done:
			return
		case <-n.exit:
			// close the connection by sending a close message
			close := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			err := n.conn.WriteMessage(websocket.CloseMessage, close)
			if err != nil {
				fmt.Println("write close:", err)
				return
			}
			// wait for server to close the connection (or timeout)
			select {
			case <-n.done:
			case <-time.After(time.Second):
			}
			return
		case t := <-ticker.C:
			err := n.conn.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func (n *Notifications) read() {
	fmt.Println("Start read loop...")
	defer close(n.done)
	for {
		_, msg, err := n.conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			// server closed connection
			return
		}
		n.onMessage(string(msg))
	}
}

func (n *Notifications) onMessage(msg string) {
	// TODO Lock()
	if n.hdl == nil {
		return
	}

	fmt.Println(msg)
	// parse JSON?
	n.hdl(msg)
}

func (n *Notifications) OnMessage(f MessageHandler) {
	// TODO Lock()
	n.hdl = f
}
