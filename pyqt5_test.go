package pyqt5

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	qt, err := New(`
Connect('foo', lambda a, b, c: [
	print(a, b, c),
	Emit('bar', a, b, c),
])
	`)
	if err != nil {
		t.Fatalf("New %v", err)
	}
	defer qt.Close()
	done := make(chan bool)
	qt.Connect("bar", func(a, b float64, c string) {
		if a != 1 || b != 2 || c != "你好" {
			t.Fail()
		}
		fmt.Printf("bar %f %f %s\n", a, b, c)
		close(done)
	})
	qt.Emit("foo", 1, 2, "你好")
	select {
	case <-done:
	case <-time.After(time.Second * 3):
		t.Fatal("no signal")
	}
}
