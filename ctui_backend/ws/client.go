package ws

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
  writeWait = 10 * time.Second
  pongWait = 60 * time.Second
  pingPeriod = (pongWait * 9)   / 10
  maxMessageSize = 512
)

var (
  newLine = []byte{'\n'}
  space = []byte{' '}
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}

// Client => The middleman between the Websocket connection and the Hub.
type Client struct {
  hub *Hub
  conn *websocket.Conn

  send chan []byte
}

// readPump pumps messages from the websocket connection to the Hub.
//
// The application runs readPump in a per-connection Goroutine. The
// application ensures that there is at most one reader on a connection
// by executing all reads from this Goroutine.
func(c *Client)readPump() {
  defer func() {
		c.hub.unregister <-c
    c.conn.Close()
  }()

  c.conn.SetReadLimit(maxMessageSize+64)// Plus Metadata.
  c.conn.SetReadDeadline(time.Now().Add(pongWait))
  c.conn.SetPongHandler(
    func(string) error {
      c.conn.SetReadDeadline(time.Now().Add(pongWait));
      return nil
  })

  for {
    _, message, err := c.conn.ReadMessage()
    if err != nil {
      if websocket.IsUnexpectedCloseError(
        err,
        websocket.CloseGoingAway,
        websocket.CloseAbnormalClosure,
      ){
        log.Printf("error: %v", err)
      }
      break
    }
    fmt.Println(" -> READING MSG")
    message = bytes.TrimSpace(bytes.Replace(message, newLine, space, -1))

    // Add to Database.

    c.hub.broadcast <- message
  }
}

// writePump pumps messages from the hub to the webscoket connection.
//
// A Goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection
// by executing all writes from this Goroutine.
func(c *Client)writePump() {
  ticker := time.NewTicker(pingPeriod)
  defer func(){
    ticker.Stop()
    c.conn.Close()
}()
  for {
    select {
    case message, ok := <-c.send:
      c.conn.SetWriteDeadline(time.Now().Add(writeWait))
      if !ok {
        // The hub closed the channel
        c.conn.WriteMessage(websocket.CloseMessage, []byte{})
        return
      }
      w, err := c.conn.NextWriter(websocket.TextMessage)
      if err != nil {
        log.Printf(" -> Connection NextWriter Error: %s", err)
        return
      }
      // Store on DB

      w.Write(message)
      n := len(c.send)

      for i := 0; i < n; i++ {
        w.Write(newLine)
        w.Write(<-c.send)
      }
      fmt.Println(" -> WROTE MSG")
      if err := w.Close(); err != nil {
        return
      }
    case <-ticker.C:
      c.conn.SetWriteDeadline(time.Now().Add(writeWait))
      if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
        return
      }
    }
  }
}

// Handle Websocket requests from the Peer
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request){
  conn, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Println(err)
    return
  }
  client := &Client{ hub:hub, conn:conn, send:make(chan []byte, 256) }
  client.hub.register <- client

  // Allow collection of memory referenced by the caller by doing all work in
  // new Goroutines.
  go client.writePump()
  go client.readPump()
}
