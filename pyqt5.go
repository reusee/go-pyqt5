package pyqt5

/*
#include <Python.h>
#include <stdlib.h>
#cgo pkg-config: python2
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"
	"unsafe"
)

var conn *net.UnixConn
var connReady = make(chan bool)

var cbs = make(map[string][]reflect.Value)

func Init() {
	rand.Seed(time.Now().UnixNano())
	C.Py_Initialize()
	runtime.LockOSThread()
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", rand.Uint32()))
	RunString(fmt.Sprintf(`
from PyQt5.QtWidgets import QApplication
from PyQt5.QtNetwork import QLocalServer
import json

App = QApplication([])

_gopyqt5_signals = dict()

server = QLocalServer()
socket = None
if not server.listen("%s"):
	raise Exception("server listen error")
def onNewConn():
	global socket
	socket = server.nextPendingConnection()
	buf = [bytearray()]
	def onReady():
		for b in bytearray(socket.readAll()):
			if b == 0:
				data = json.loads(buf[0].decode('utf8'))
				if data['Signal'] in _gopyqt5_signals:
					for cb in _gopyqt5_signals[data['Signal']]:
						if data['Args']:
							cb(*data['Args'])
						else:
							cb()
				buf[0] = bytearray()
			else:
				buf[0].append(b)
	socket.readyRead.connect(onReady)
server.newConnection.connect(onNewConn)

def Connect(signal, cb):
	_gopyqt5_signals.setdefault(signal, [])
	_gopyqt5_signals[signal].append(cb)

def Emit(signal, *args):
	buf = bytearray(json.dumps({
		'Signal': signal,
		'Args': args,
	}).encode('utf8'))
	buf.append(0)
	socket.write(buf)

	`, socketPath))

	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		log.Fatalf("ResolveUnixAddr %v", err)
	}
	conn, err = net.DialUnix("unix", nil, addr)
	if err != nil {
		log.Fatalf("DialUnix %v", err)
	}
	close(connReady)
	go func() {
		var buf []byte
		c := make([]byte, 1)
		for {
			_, err := conn.Read(c)
			if err != nil {
				return
			}
			if c[0] == '\x00' {
				var msg _Message
				json.NewDecoder(bytes.NewReader(buf)).Decode(&msg)
				if len(cbs[msg.Signal]) > 0 {
					var values []reflect.Value
					for _, arg := range msg.Args {
						values = append(values, reflect.ValueOf(arg))
					}
					for _, cb := range cbs[msg.Signal] {
						cb.Call(values)
					}
				}
				buf = buf[0:0]
			} else {
				buf = append(buf, c[0])
			}
		}
	}()

}

func Finalize() {
	conn.Close()
	RunString(`server.close()`)
	C.Py_Finalize()
}

func RunString(code string) {
	cCode := C.CString(code)
	C.PyRun_SimpleStringFlags(cCode, nil)
	C.free(unsafe.Pointer(cCode))
}

func Main() {
	RunString(`App.exec_()`)
}

type _Message struct {
	Signal string
	Args   []interface{}
}

func Emit(signal string, args ...interface{}) {
	<-connReady
	msg := _Message{
		Signal: signal,
		Args:   args,
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(msg)
	buf.WriteByte(byte(0))
	conn.Write(buf.Bytes())
}

func Connect(signal string, cb interface{}) {
	cbs[signal] = append(cbs[signal], reflect.ValueOf(cb))
}
