package main

import (
	"bufio"
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"os"
	"strings"
	"time"
)

func writeRequest(request *strings.Builder, scanner *bufio.Scanner, instruction string) error {
	fmt.Printf("%s", instruction)
	scanner.Scan()
	text := scanner.Text()
	for len(text) > 20 {
		fmt.Printf("cannot contain more than 20 characters, enter again: ")
		scanner.Scan()
		text = scanner.Text()
	}
	_, err := fmt.Fprintf(request, ";body:%s", text)
	return err
}

func sendRequest(command string, scanner *bufio.Scanner) (string, error) {
	var request strings.Builder
	fmt.Fprintf(&request, "header:%s", command)
	var err error
	switch command {
	case "NICK":
		err = writeRequest(&request, scanner, "enter username: ")
	case "LIST":
		return request.String(), nil
	case "JOIN":
		err = writeRequest(&request, scanner, "enter channel: ")
	case "PART":
		return request.String(), nil
	case "QUIT":
		return request.String(), nil
	case "WHO":
		err = writeRequest(&request, scanner, "enter channel: ")
	case "PRIVMSG":
		err = writeRequest(&request, scanner, "enter receiver: ")
		err = writeRequest(&request, scanner, "enter content: ")
		err = writeRequest(&request, scanner, "press y if you pm: ")
	case "HELP":
		return request.String(), nil
	default:
		return request.String(), nil
	}
	return request.String(), err
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	SERVER_PORT := os.Getenv("SERVER_PORT")
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", SERVER_PORT)
		if err != nil {
			fmt.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}
		scanner := bufio.NewScanner(os.Stdin)
		go func() {
			for {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					fmt.Println("cannot read data from server:", err)
					return
				}
				fmt.Println("server:", string(buf[:n]))
			}
		}()
		for {
			scanner.Scan()
			command := scanner.Text()
			command = strings.TrimSpace(command)
			if len(command) == 0 {
				continue
			}
			request, err := sendRequest(command, scanner)
			if err != nil {
				fmt.Println(err)
				fmt.Println("enter command again")
			} else {
				_, err2 := conn.Write([]byte(request))
				if err2 != nil {
					conn.Close()
					return
				}
			}

		}
	}
}
