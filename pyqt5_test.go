package pyqt5

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	Init()
	defer Finalize()

	Emit("foo", 1, 2, "你好")
	Connect("FOO", func(a, b float64, c string) {
		if a != 1 || b != 2 || c != "你好" {
			t.Fail()
		}
		fmt.Printf("FOO %f %f %s\n", a, b, c)
		Emit("quit")
	})

	RunString(`
Connect("foo", lambda a, b, c:
	Emit('FOO', a, b, c))
Connect("quit", lambda: App.quit())
	`)

	Main()
}
