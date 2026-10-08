package signaling

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/guanshanh/wavecall/internal/auth"
	"github.com/guanshanh/wavecall/internal/config"
	"github.com/guanshanh/wavecall/internal/room"
	"github.com/guanshanh/wavecall/pkg/proto"
)

func TestJoinRejectsBadTokenWithoutCreatingRoom(t *testing.T) {
	users, err := auth.NewDirectory("top-secret", []auth.User{{Account: "achi", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	manager := room.NewManager()
	h, err := NewHandler(manager, &config.Config{}, users)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	client := &Client{send: make(chan []byte, 1)}
	h.handleJoin(client, &proto.JoinMessage{
		Type:     proto.TypeJoin,
		RoomID:   "room-1",
		UserName: "玩家",
		Token:    "not-a-token",
	})
	if manager.RoomCount() != 0 {
		t.Fatalf("room count %d", manager.RoomCount())
	}
	var msg proto.ErrorMessage
	if err := json.Unmarshal(<-client.send, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Code != "UNAUTHORIZED" || msg.Message != "登录已失效，请重新登录" {
		t.Fatalf("error %+v", msg)
	}
}

func TestJoinWrongRoomPasswordStillWrongPassword(t *testing.T) {
	users, err := auth.NewDirectory("top-secret", []auth.User{{Account: "achi", Password: "pw"}})
	if err != nil {
		t.Fatal(err)
	}
	token, err := users.Issue("achi", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	manager := room.NewManager()
	manager.GetOrCreate("room-1", "right-secret")
	h, err := NewHandler(manager, &config.Config{}, users)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	client := &Client{send: make(chan []byte, 1)}
	h.handleJoin(client, &proto.JoinMessage{
		Type:     proto.TypeJoin,
		RoomID:   "room-1",
		UserName: "玩家",
		Password: "wrong-secret",
		Token:    token,
	})
	if manager.RoomCount() != 1 {
		t.Fatalf("room count %d", manager.RoomCount())
	}
	var msg proto.ErrorMessage
	if err := json.Unmarshal(<-client.send, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Code != "WRONG_PASSWORD" {
		t.Fatalf("code %s", msg.Code)
	}
}
