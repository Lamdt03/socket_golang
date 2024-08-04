package model

import "net"

type Connection struct {
	Conn     net.Conn
	Username string
	MsgCh    chan Message
	GroupCh  string
}

type Message struct {
	From    string
	Payload []byte
}

func (c Connection) ToString() string {
	return c.Conn.RemoteAddr().String() + " " + c.Username + " " + c.GroupCh
}
