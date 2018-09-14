package bytes

import (
	"testing"
)

func TestBufferAt(t *testing.T) {
	buf := new(BufferAt)

	expected := "test string"
	n, err := buf.Write([]byte(expected))
	if err != nil {
		t.Fatal(err)
	}
	if n != len(expected) {
		t.Fatalf("write size: get %d, want %d", n, len(expected))
	}

	addtext := "abc"
	expected = "tabc string"
	n, err = buf.WriteAt([]byte(addtext), 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(addtext) {
		t.Fatalf("write size: get %d, want %d", n, len(addtext))
	}
	get := buf.String()
	if get != expected {
		t.Fatalf("write: get %q, want %q", get, expected)
	}
}
