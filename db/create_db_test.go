package db

import (
	"fmt"
	"testing"
)

func TestGetCentipawnEval(t *testing.T) {
	var tests = []struct {
		line   []byte
		output int
		length int
	}{
		{[]byte("123 456"), 123, 3},
		{[]byte("98999\"456"), 98999, 5},
		{[]byte("-123 456"), -123, 4},
		{[]byte(" 1"), 0, 0},
	}

	for _, tt := range tests {

		testname := fmt.Sprintf("%s,%d", tt.line, tt.output)
		t.Run(testname, func(t *testing.T) {
			ans, length := getCentipawnEval(&tt.line)
			if ans != tt.output || length != tt.length {
				t.Errorf("got %d %d, wanted %d, %d", ans, length, tt.output, tt.length)
			}
		})
	}
}
