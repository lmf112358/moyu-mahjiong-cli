package netplay

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"

	"github.com/LimitlessMindForce/moyu-mahjiong-cli/internal/game"
)

func TestPeerDecisionProtocol(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	p := &Peer{conn: server, enc: json.NewEncoder(server), dec: json.NewDecoder(bufio.NewReader(server))}
	done := make(chan game.Action, 1)
	go func() {
		done <- p.Decide(game.Decision{Prompt: "test", Options: []game.Option{{Label: "pass", Action: game.Action{Kind: game.ActPass}}}})
	}()
	dec := json.NewDecoder(bufio.NewReader(client))
	enc := json.NewEncoder(client)
	var msg Message
	if err := dec.Decode(&msg); err != nil || msg.Decision == nil {
		t.Fatalf("decode decision: %v", err)
	}
	want := game.Action{Kind: game.ActPass}
	if err := enc.Encode(Message{Type: "action", Action: &want}); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got.Kind != game.ActPass {
		t.Fatalf("got %v", got.Kind)
	}
}

func TestPeerDecisionRejectsWrongMessageType(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	p := &Peer{conn: server, enc: json.NewEncoder(server), dec: json.NewDecoder(bufio.NewReader(server))}
	done := make(chan game.Action, 1)
	go func() { done <- p.Decide(game.Decision{Prompt: "test"}) }()
	dec := json.NewDecoder(bufio.NewReader(client))
	enc := json.NewEncoder(client)
	var msg Message
	if err := dec.Decode(&msg); err != nil {
		t.Fatal(err)
	}
	action := game.Action{Kind: game.ActPass}
	if err := enc.Encode(Message{Type: "settlement_ack", Action: &action}); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got.Kind != game.ActQuit {
		t.Fatalf("wrong message type accepted as %v", got.Kind)
	}
}

func TestPeerSettlementProtocol(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	p := &Peer{conn: server, enc: json.NewEncoder(server), dec: json.NewDecoder(bufio.NewReader(server))}
	done := make(chan struct{})
	go func() { p.ShowSettlement(game.Settlement{HandNumber: 1}); close(done) }()
	dec := json.NewDecoder(bufio.NewReader(client))
	enc := json.NewEncoder(client)
	var msg Message
	if err := dec.Decode(&msg); err != nil || msg.Type != "settlement" || msg.Settlement == nil {
		t.Fatalf("message=%+v err=%v", msg, err)
	}
	if err := enc.Encode(Message{Type: "settlement_ack"}); err != nil {
		t.Fatal(err)
	}
	<-done
}
