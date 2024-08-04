package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"os"
	"strings"
	"sync"
	"week6/src/server/model"
	"week6/src/server/service"
)

type Server struct {
	listenAddr string
	ln         net.Listener
	quitCh     chan struct{}                //close server
	pool       map[string]*model.Connection //handle multi client
	poolLock   sync.RWMutex
	grpCh      map[string][]model.Connection //to communicate with client
}

func NewServer(listenAddr string) *Server {
	grpCh := make(map[string][]model.Connection)
	//available channel
	grpCh["ch1"] = []model.Connection{}
	grpCh["ch2"] = []model.Connection{}
	grpCh["ch3"] = []model.Connection{}
	grpCh["ch4"] = []model.Connection{}
	grpCh["ch5"] = []model.Connection{}
	return &Server{
		listenAddr: listenAddr,
		quitCh:     make(chan struct{}),
		pool:       make(map[string]*model.Connection),
		grpCh:      grpCh,
	}
}

// set nick name of a connection
func (s *Server) setUsername(connection *model.Connection, username string) string {
	for _, conn := range s.pool {
		if username == conn.Username {
			return "header:NICK;body:username exist"
		}
	}
	connection.Username = username
	// change username in channel if exists
	if connection.GroupCh != "" {
		for i := 0; i < len(s.grpCh[connection.GroupCh]); i++ {
			if s.grpCh[connection.GroupCh][i].Conn.RemoteAddr().String() == connection.Conn.RemoteAddr().String() {
				s.grpCh[connection.GroupCh][i] = *connection
				break
			}
		}
	}
	return "header:NICK;body:" + connection.Username
}

// join channel if it does not exist, create
func (s *Server) joinCreateChannel(connection *model.Connection, grpChannel string) (string, int) {
	if connection.GroupCh != "" {
		return "header:JOIN;body:you must quit current channel", 0
	}
	if grpChannel == "" {
		return "header:JOIN;body:channel name can not be empty", 0
	}
	if _, ok := s.grpCh[grpChannel]; !ok {
		s.grpCh[grpChannel] = make([]model.Connection, 0)
	}
	connection.GroupCh = grpChannel
	s.grpCh[grpChannel] = append(s.grpCh[grpChannel], *connection)
	return "header:JOIN;body:" + grpChannel, 1
}

func (s *Server) getAllChannel() string {
	var listCh strings.Builder
	_, err := fmt.Fprintln(&listCh, "header:LIST;body:")
	if err != nil {
		fmt.Println(err)
	}
	for ch, mems := range s.grpCh {
		_, err2 := fmt.Fprintln(&listCh, ch, ": ", len(mems))
		if err2 != nil {
			fmt.Println(err2)
		}
	}
	return listCh.String()
}

// get all member info of a channel
func (s *Server) getChannelInfo(grpCh string) string {
	if _, ok := s.grpCh[grpCh]; !ok {
		return "header:WHO;body:channel not exist"
	}
	var listMem strings.Builder
	_, err := fmt.Fprintln(&listMem, "header:WHO;body:")
	if err != nil {
		fmt.Println(err)
	}
	for _, mem := range s.grpCh[grpCh] {
		_, err2 := fmt.Fprintln(&listMem, mem.ToString())
		if err2 != nil {
			fmt.Println(err2)
		}
	}
	return listMem.String()
}

// left current channel if exists
func (s *Server) part(connection *model.Connection) string {
	if _, ok := s.grpCh[connection.GroupCh]; !ok {
		return "header:PART;body:channel not exist"
	} else {
		for i := 0; i < len(s.grpCh[connection.GroupCh]); i++ {
			if s.grpCh[connection.GroupCh][i].Conn.RemoteAddr().String() == connection.Conn.RemoteAddr().String() {
				s.grpCh[connection.GroupCh] = append(s.grpCh[connection.GroupCh][:i], s.grpCh[connection.GroupCh][(i+1):]...)
				break
			}
		}
		connection.GroupCh = ""
		return "header:PART;body:quit channel success"
	}
}

// disconnect a user from server
func (s *Server) disconnect(connection *model.Connection) {
	err := connection.Conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (s *Server) quit(connection *model.Connection) {
	//left group
	if connection.GroupCh != "" {
		for i := 0; i < len(s.grpCh[connection.GroupCh]); i++ {
			if s.grpCh[connection.GroupCh][i].Conn.RemoteAddr().String() == connection.Conn.RemoteAddr().String() {
				s.grpCh[connection.GroupCh] = append(s.grpCh[connection.GroupCh][:i], s.grpCh[connection.GroupCh][(i+1):]...)
			}
		}
		// notify other user in channel
		for _, conn := range s.pool {
			if conn.GroupCh == connection.GroupCh && conn.Username != connection.Username {
				_, err := conn.Conn.Write([]byte("header:QUIT;body:" + connection.Username + " quit the channel"))
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
	}
	// disconnect from server
	err := connection.Conn.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	//delete from pool of client
	delete(s.pool, connection.Conn.RemoteAddr().String())
	close(connection.MsgCh)
}

// directed message
func (s *Server) pm(connection model.Connection, receiver string, msg string) string {
	for _, conn := range s.pool {
		if conn.Username == receiver {
			_, err := conn.Conn.Write([]byte("header:NEWMSG;sender:" + connection.Username + ";content:" + msg))
			if err != nil {
				fmt.Println(err)
				return "header:PRIVMSG;body:cannot send message"
			}
			return "header:PRIVMSG;body:send message success"
		}
	}
	return "header:PRIVMSG;body:receiver not exist"
}

// send message to a channel
func (s *Server) msgCh(connection model.Connection, ch string, msg string) string {
	if _, ok := s.grpCh[ch]; !ok {
		return "header:PRIVMSG;body:channel not exist"
	}
	for _, receiver := range s.grpCh[ch] {
		if receiver.Username == connection.Username {
			continue
		}
		_, err := receiver.Conn.Write([]byte("header:NEWMSG;sender:" + connection.Username + ";content:" + msg))
		if err != nil {
			fmt.Println(err)
			return "header:PRIVMSG;body:cannot send message"
		}
	}
	return "header:PRIVMSG;body:send message success"
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	s.ln = ln
	go s.acceptLoop()
	go s.HandleMessage()
	<-s.quitCh
	return nil
}

// listen connection from client
func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("new connection: ", conn.RemoteAddr())
		go s.Handler(conn, conn.RemoteAddr().String())
	}
}

// Handler create a handler for each client to communicate with server
func (s *Server) Handler(conn net.Conn, username string) {
	c := &model.Connection{
		Conn:     conn,
		Username: username,
		MsgCh:    make(chan model.Message, 2048),
		GroupCh:  "",
	}
	s.poolLock.Lock()
	s.pool[conn.RemoteAddr().String()] = c
	s.poolLock.Unlock()
	defer func() {
		s.poolLock.Lock()
		s.quit(c)
		s.poolLock.Unlock()

	}()
	buf := make([]byte, 2048)
	for {
		n, err := c.Conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			return
		}
		msg := model.Message{
			From:    username,
			Payload: buf[:n],
		}
		fmt.Println(string(msg.Payload))
		c.MsgCh <- msg
	}

}

// help command
func help() string {
	response := "header:HELP\n" + "NICK: set username\n" +
		"QUIT: end session\n" +
		"JOIN: join a channel\n" +
		"PART: quit current channel\n" +
		"LIST: get all channels info\n" +
		"PRIVMSG: send message to another user or a channel\n" +
		"WHO: get all user info of a channel\n" +
		"HELP: get commands\n"
	return response
}

// create a goroutine to resolve incoming message from clients
func (s *Server) HandleMessage() {
	for {
		select {
		case <-s.quitCh:
			return
		default:
			for connId := range s.pool {
				conn := s.pool[connId]
				s.poolLock.Lock()
				// get client message
				select {
				case msg := <-conn.MsgCh:

					fmt.Println("from ", msg.From, ": ", string(msg.Payload))
					cleanMsg := service.CleanMessage(string(msg.Payload))
					var response string
					var status int
					switch cleanMsg[0] {
					case "NICK":
						response = s.setUsername(conn, cleanMsg[1])
						s.pool[connId].Username = cleanMsg[1]
					case "LIST":
						response = s.getAllChannel()
					case "QUIT":
						s.disconnect(conn)
					case "JOIN":
						response, status = s.joinCreateChannel(conn, cleanMsg[1])
						if status == 1 {
							s.pool[connId].GroupCh = cleanMsg[1]
						}
					case "PART":
						response = s.part(conn)
						s.pool[connId].GroupCh = ""
					case "WHO":
						response = s.getChannelInfo(cleanMsg[1])
					case "PRIVMSG":
						if cleanMsg[3] == "y" {
							response = s.pm(*conn, cleanMsg[1], cleanMsg[2])
						} else {
							response = s.msgCh(*conn, cleanMsg[1], cleanMsg[2])
						}
					case "HELP":
						response = help()
					default:
						response = "ERR_UNKNOWNCOMMAND"
					}
					if response != "" {
						_, err := conn.Conn.Write([]byte(response))
						if err != nil {
							fmt.Println("cannot send message: ", err)
						}
					}
				default:
					// Do nothing, move to the next connection
				}
				s.poolLock.Unlock()
			}
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	SERVER_PORT := os.Getenv("SERVER_PORT")
	fmt.Println("port:", SERVER_PORT)
	server := NewServer(SERVER_PORT)
	err1 := server.Start()
	if err1 != nil {
		fmt.Println(err)
		return
	}
}
