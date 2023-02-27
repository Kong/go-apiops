package deckfile

import (
	"testing"
)

func Test_NewKongService_panic(t *testing.T) {
	cases := []struct {
		desc     string
		data     map[string]interface{}
		deckfile *DeckFile
	}{
		{"Panics if data is nil", nil, &DeckFile{}},
		{"Panics if the deckfile is nil", make(map[string]interface{}), nil},
	}

	for _, testcase := range cases {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("failed; %s", testcase.desc)
			}
		}()
		NewKongService(testcase.data, testcase.deckfile)
	}
}
