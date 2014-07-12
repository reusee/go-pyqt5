package pyqt5

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	Init()
	defer Finalize()

	Emit("foo", 1, 2, 3)
	Connect("FOO", func(a, b, c float64) {
		if a != 1 || b != 2 || c != 3 {
			t.Fail()
		}
		fmt.Printf("FOO %f %f %f\n", a, b, c)
		Emit("quit")
	})

	RunString(`
Connect("foo", lambda a, b, c: [
	print(a, b, c),
	Emit('FOO', a, b, c)
])
Connect("quit", lambda: app.quit())
	`)

	Main()
}
