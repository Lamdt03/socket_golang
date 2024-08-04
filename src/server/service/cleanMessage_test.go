package service_test

import (
	"reflect"
	"testing"
	"week6/src/server/service"
)

type data struct {
	rawMessage string
	expected   []string
}

func generateTest(rawMessage string, expected []string) data {
	return data{
		rawMessage: rawMessage,
		expected:   expected,
	}
}

func TestCleanMessage(t *testing.T) {
	testData := []data{generateTest("header;username;body:la:m243", []string{"username", "lam243"})}
	for i := 0; i < len(testData); i++ {
		get := service.CleanMessage(testData[i].rawMessage)
		want := testData[i].expected
		if !reflect.DeepEqual(get, want) {
			t.Errorf("expected value case %d is %v instead of %v", i, want, get)
		}
	}
}
