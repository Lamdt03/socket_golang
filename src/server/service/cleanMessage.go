package service

import "strings"

func CleanMessage(rawMessage string) []string {
	newMessage := strings.Split(rawMessage, ";")
	for i := 0; i < len(newMessage); i++ {
		a := strings.Split(newMessage[i], ":")
		if len(a) == 2 {
			newMessage[i] = strings.Split(newMessage[i], ":")[1]
		}
	}
	return newMessage
}
