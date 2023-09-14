package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	// "strings"
	"sync"

	// JWT Webtokens - OAuth & Bearer security
	// "github.com/dgrijalva/jwt-go"
	// "cloud.google.com/go/pubsub"
	"chatatui_backend/db"
	"chatatui_backend/ws"
	"chatatui_backend/router"
	"github.com/gorilla/mux"
)

var live_chatrooms sync.Map

func testData(cache *db.Chatrooms){
  cache.CreateRoom("LydiaIsCute");
  cache.CreateRoom("TylerIsAwesome");
  cache.CreateRoom("ReannaIsBeautiful");
}

var Chatrooms = db.Chatrooms{
    Rooms: make(map[db.RoomName]db.Chatroom),
  }

func main(){
  testData(&Chatrooms)

  r := mux.NewRouter()

  r.HandleFunc("/", homeHandler)
  r.HandleFunc("/chatrooms", listChatroomsHandler)
  r.HandleFunc("/chatrooms/{room_id}", chatroomMeta)
  r.HandleFunc("/chatrooms/{room_id}/ws", websocketHandler)

  log.Println(" --> ChataTUI-Server Started <--")
  http.Handle("/", r)
  if err := http.ListenAndServe(router.PORT, nil); err != nil {
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

func chatroomMeta(w http.ResponseWriter, r *http.Request) {
  // chatrooms := getChatrooms()
  vars := mux.Vars(r)
  roomName := vars["room_name"]

  _, connect := r.URL.Query()["connect"]

  w.Header().Set("Content-Type", "application/json")
  room, exists := Chatrooms.Rooms[roomName]
  if !exists {
    if connect {
      json.NewEncoder(w).Encode(map[string]bool{"exists": true})
      return
    } else {
      json.NewEncoder(w).Encode(map[string]string{
        "NULL": fmt.Sprintf("Chatroom \"%s\" doesn't exist.", roomName),
      })
      return
    }
  }
  json.NewEncoder(w).Encode(room)
}

func listChatroomsHandler(w http.ResponseWriter, r *http.Request){
  names := make([]string, len(Chatrooms.Rooms))

  for name := range Chatrooms.Rooms {
    names = append(names, name)
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

  // hub, ok := live_chatrooms.Load(roomID)
  // if !ok {
  //   response := map[string]string{
  //     "NotFound": fmt.Sprintf("The Chatroom %s is not created yet, maybe create it?", roomID),
  //   }
  //   w.Header().Set("Content-Type", "application/json")
  //   json.NewEncoder(w).Encode(response)
  //   return
  // }
  hub, ok := live_chatrooms.LoadOrStore(roomID, ws.NewHub())
  if !ok {
    go hub.(*ws.Hub).Run()
  }

  ws.ServeWs(hub.(*ws.Hub), w, r)
}
