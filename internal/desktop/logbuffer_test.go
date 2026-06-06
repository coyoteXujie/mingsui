package desktop

import (
	"reflect"
	"testing"
)

func TestLogBufferKeepsRecentLines(t *testing.T) {
	buffer := NewLogBuffer(2)
	if _, err := buffer.Write([]byte("one\ntwo\nthree\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got, want := buffer.Lines(), []string{"two", "three"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() = %+v, want %+v", got, want)
	}
}

func TestLogBufferKeepsPartialLine(t *testing.T) {
	buffer := NewLogBuffer(3)
	if _, err := buffer.Write([]byte("one\npartial")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got, want := buffer.Lines(), []string{"one", "partial"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() = %+v, want %+v", got, want)
	}
	if _, err := buffer.Write([]byte(" done\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got, want := buffer.Lines(), []string{"one", "partial done"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() = %+v, want %+v", got, want)
	}
}
