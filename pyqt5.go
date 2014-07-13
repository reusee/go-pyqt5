package pyqt5

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type PyQt struct {
	conn  net.Conn
	ready chan bool
	cbs   map[string][]reflect.Value
}

type _Message struct {
	Signal string
	Args   []interface{}
}

func New() (*PyQt, error) {
	qt := &PyQt{
		ready: make(chan bool),
		cbs:   make(map[string][]reflect.Value),
	}
	qt.Connect("__exception__", func(desc string) {
		log.Fatal(desc)
	})

	// start unix domain socket
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", rand.Uint32()))
	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, err
	}
	ln, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("Accept %v", err)
		}
		qt.conn = conn
		close(qt.ready)
		var buf []byte
		c := make([]byte, 1)
		for {
			_, err := conn.Read(c)
			if err != nil {
				return //TODO python closed
			}
			if c[0] == '\x00' {
				var msg _Message
				json.NewDecoder(bytes.NewReader(buf)).Decode(&msg)
				if len(qt.cbs[msg.Signal]) > 0 {
					var values []reflect.Value
					for _, arg := range msg.Args {
						values = append(values, reflect.ValueOf(arg))
					}
					for _, cb := range qt.cbs[msg.Signal] {
						cb.Call(values)
					}
				}
				buf = buf[0:0]
			} else {
				buf = append(buf, c[0])
			}
		}
	}()

	// start python
	cmd := exec.Command("python", "-c", `
from PyQt5.QtWidgets import QApplication
from PyQt5.QtNetwork import QLocalSocket
import sys
import json
import traceback
App = QApplication([])
def run(code):
	c = compile(code, '<string>', 'exec')
	exec(c)
_gopyqt5_signals = {
	'__run__': [run],
}
socket = QLocalSocket()
socket.connectToServer(sys.argv[1])
buf = bytearray()
def onReady():
	for b in bytearray(socket.readAll()):
		if b == 0:
			data = json.loads(buf.decode('utf8'))
			if data['Signal'] in _gopyqt5_signals:
				for cb in _gopyqt5_signals[data['Signal']]:
					if data['Args']:
						cb(*data['Args'])
					else:
						cb()
			buf.clear()
		else:
			buf.append(b)
socket.readyRead.connect(onReady)
socket.disconnected.connect(lambda: App.quit())
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
def excepthook(t, v, tb):
	Emit('__exception__', '%s %s\n%s' % (str(t), str(v), ''.join(traceback.format_tb(tb))))
sys.excepthook = excepthook
App.exec_()
	`, socketPath)
	cmd.Start()

	return qt, nil
}

func (qt *PyQt) Close() {
	qt.conn.Close()
}

func (qt *PyQt) Emit(signal string, args ...interface{}) {
	<-qt.ready
	msg := _Message{
		Signal: signal,
		Args:   args,
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(msg)
	buf.WriteByte(byte(0))
	qt.conn.Write(buf.Bytes())
}

func (qt *PyQt) Connect(signal string, cb interface{}) {
	qt.cbs[signal] = append(qt.cbs[signal], reflect.ValueOf(cb))
}

func (qt *PyQt) Run(code string) {
	qt.Emit("__run__", code)
}
