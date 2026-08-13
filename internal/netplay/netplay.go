package netplay

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/lmf112358/moyu-mahjiong-cli/internal/game"
)

type Message struct {
	Type       string            `json:"type"`
	Name       string            `json:"name,omitempty"`
	Decision   *game.Decision    `json:"decision,omitempty"`
	Action     *game.Action      `json:"action,omitempty"`
	Result     *game.MatchResult `json:"result,omitempty"`
	Settlement *game.Settlement  `json:"settlement,omitempty"`
	Text       string            `json:"text,omitempty"`
}

type Peer struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
	mu   sync.Mutex
	Name string
}

func Listen(addr string, want int, onJoin func(int, string)) (net.Listener, []*Peer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	peers := []*Peer{}
	for len(peers) < want {
		conn, err := ln.Accept()
		if err != nil {
			ln.Close()
			return nil, nil, err
		}
		p := &Peer{conn: conn, enc: json.NewEncoder(conn), dec: json.NewDecoder(bufio.NewReader(conn))}
		var hello Message
		if err = p.dec.Decode(&hello); err != nil || hello.Type != "hello" {
			conn.Close()
			continue
		}
		p.Name = hello.Name
		peers = append(peers, p)
		p.enc.Encode(Message{Type: "welcome", Text: fmt.Sprintf("已加入房间（%d/%d）", len(peers), want)})
		if onJoin != nil {
			onJoin(len(peers), p.Name)
		}
	}
	return ln, peers, nil
}

func (p *Peer) Decide(d game.Decision) game.Action {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.enc.Encode(Message{Type: "decision", Decision: &d}); err != nil {
		return game.Action{Kind: game.ActQuit}
	}
	var m Message
	if err := p.dec.Decode(&m); err != nil || m.Action == nil {
		return game.Action{Kind: game.ActQuit}
	}
	return *m.Action
}
func (p *Peer) Finish(r game.MatchResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enc.Encode(Message{Type: "result", Result: &r})
	p.conn.Close()
}

func (p *Peer) ShowSettlement(s game.Settlement) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.enc.Encode(Message{Type: "settlement", Settlement: &s}); err != nil {
		return
	}
	var ack Message
	_ = p.dec.Decode(&ack)
}

func Join(addr, name string, controller game.Controller) (game.MatchResult, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return game.MatchResult{}, err
	}
	defer conn.Close()
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(bufio.NewReader(conn))
	if err = enc.Encode(Message{Type: "hello", Name: name}); err != nil {
		return game.MatchResult{}, err
	}
	for {
		var m Message
		if err = dec.Decode(&m); err != nil {
			return game.MatchResult{}, err
		}
		switch m.Type {
		case "welcome":
			fmt.Println(m.Text)
		case "decision":
			if m.Decision == nil {
				continue
			}
			a := controller.Decide(*m.Decision)
			if err = enc.Encode(Message{Type: "action", Action: &a}); err != nil {
				return game.MatchResult{}, err
			}
		case "settlement":
			if m.Settlement == nil {
				continue
			}
			if viewer, ok := controller.(game.SettlementViewer); ok {
				viewer.ShowSettlement(*m.Settlement)
			}
			if err = enc.Encode(Message{Type: "settlement_ack"}); err != nil {
				return game.MatchResult{}, err
			}
		case "result":
			if m.Result == nil {
				return game.MatchResult{}, errors.New("服务器返回空结果")
			}
			return *m.Result, nil
		}
	}
}
