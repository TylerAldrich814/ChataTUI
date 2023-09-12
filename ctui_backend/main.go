package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	// api?
	// "github.com/dgrijalva/jwt-go"
	// "cloud.google.com/go/pubsub"
	"chatatui_backend/ws"

	"github.com/gorilla/mux"
)

const PORT = ":8080"
var addr = flag.String("addr", PORT, "htto service address")
var live_chatrooms sync.Map

type User string

type Message struct {
  creator User
  content string
}

type Chatroom struct {
  id  int
  name string
  url  string
  users []User
  messages []Message
}
func newChatroom(
  id       int,
  name     string,
  url      string,
  users    []User,
  messages []Message,
) Chatroom {
  return Chatroom {
    id,
    name,
    url,
    users,
    messages,
  }
}
func(cm *Chatroom)room_url() string {
  return cm.url
}

func(cm *Chatroom)room_name() string {
  return cm.name
}

func(cm *Chatroom)meta(){
  fmt.Println(" <--------------------->")
  fmt.Println(" -> ID:    {}", cm.id)
  fmt.Println(" -> NAME:  {}", cm.name)
  fmt.Println(" -> URL:   {}", cm.url)
  if len(cm.users) != 0 {
    fmt.Print(" -> USERS:")
    for i, user := range cm.users {
      if i != 0 && i % 5 == 0{
        fmt.Println()
      }
      fmt.Print(" {},", user)
    }
  }
}


func getChatrooms() []Chatroom{
  chatrooms := []Chatroom{
    newChatroom(
      2,
      "BeaverLovers",
      "beaverlovers1",
      make([]User, 0),
      make([]Message, 1024*64),
    ),
    newChatroom(
      1,
      "MyNameIsTyler",
      "mynameistyler2",
      make([]User, 0),
      make([]Message, 1024*64),
    ),
    newChatroom(
      3,
      "LydiaIsACutie",
      "lydiaisacutire3",
      make([]User, 0),
      make([]Message, 1024*64),
    ),
  }
  return chatrooms
}

func main(){
  r := mux.NewRouter()

  r.HandleFunc("/", homeHandler)
  r.HandleFunc("/chatrooms", listChatroomsHandler)
  r.HandleFunc("/chatrooms/{room_id}/ws", websocketHandler)

  log.Println(" --> ChataTUI-Server Started <--")
  http.Handle("/", r)
  if err := http.ListenAndServe(PORT, nil); err != nil {
    log.Fatalf("ListenAndServe ERROR: %s", err.Error())
  }
}

func homeHandler(w http.ResponseWriter, r *http.Request){
  welcome_message := map[string]string{
    "message": `
    Welcome to Chatatui!
    use '/chatrooms' to retreive a list of chatrooms to join
    use '/chatrooms/{room_id}' to join the room!
    `,
  }
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(welcome_message)
}

func listChatroomsHandler(w http.ResponseWriter, r *http.Request){
  chatrooms := getChatrooms()
  names := make([]string, len(chatrooms))

  for i, room := range chatrooms {
    names[i] = room.room_name()
  }
  rooms := map[string][]string{
    "rooms": names,
  }

  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(rooms)
}

func websocketHandler(w http.ResponseWriter, r *http.Request){
  log.Println(" -> websocketHandler called")
  vars := mux.Vars(r)
  roomID := vars["room_id"]
  log.Printf(" -> Room ID: %s\n", roomID)

  hub, ok := live_chatrooms.LoadOrStore(roomID, ws.NewHub())
  if !ok {
    go hub.(*ws.Hub).Run()
  }

  ws.ServeWs(hub.(*ws.Hub), w, r)
}

func mainx(){
  flag.Parse()
  fmt.Println("Just a test: ADDR: {}", *addr)
  //

  chatrooms := getChatrooms()
  var live_chatrooms sync.Map

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    log.Println(w, " -- ChataTUI-Server --")
  })

  http.HandleFunc("/chatrooms/", func(w http.ResponseWriter, r *http.Request){
    roomURL := strings.TrimPrefix(r.URL.Path, "/chatrooms/")
    if roomURL == "" {
      log.Print("List of created chatrooms:")
      for i, room := range chatrooms {
        if i != 0 && i % 5 == 0 {
          log.Println()
        }
        log.Print(" {}, ", room)
      }
      return
    }

    hub, ok := live_chatrooms.LoadOrStore(roomURL, ws.NewHub())
    if ok {
      go hub.(*ws.Hub).Run()
    }

    fullURL := fmt.Sprintf("/chatrooms/%s/ws", roomURL)
    http.HandleFunc(fullURL, func(w http.ResponseWriter, r *http.Request) {
      ws.ServeWs(hub.(*ws.Hub), w, r)
    })
  })

  log.Println("Server Started at ", PORT)
  if err := http.ListenAndServe(PORT, nil); err != nil {
    log.Fatalf("ListenAndServe ERROR: %s", err.Error())
  }
}

// func startChatroom(roomURL string){
//   hub := ws.NewHub()
//   go hub.Run()
//
//   full_url := fmt.Sprintf("/chatrooms/%s/ws", roomURL)
//
//   http.HandleFunc(full_url, func(w http.ResponseWriter, r *http.Request) {
//     ws.ServeWs(hub, w, r)
//   })
//
//   log.Println(" --> WEBSOCKET IS LIVE <--")
//
//   server := &http.Server{
//     Addr:              PORT,
//     ReadHeaderTimeout: 3 * time.Second,
//   }
//
//   if err := server.ListenAndServe(); err != nil {
//     log.Fatal("ListenAndServe: ", err)
//   }
// }
