package log

import (
	"fmt"
	"os"
	"sync"
)

var debug bool

func init() {
	debug = (len(os.Getenv("GO_MURCOTT_LOGGING")) > 0)
}

type Logger struct {
	ch    chan int
	b     [1024]string
	begin int
	size  int
	mutex sync.RWMutex
}

func NewLogger() *Logger {
	return &Logger{
		ch: make(chan int),
	}
}

func (l *Logger) Read(p []byte) (n int, err error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if l.size == 0 {
		<-l.ch
	}
	msg := l.b[l.begin]
	l.begin = (l.begin + 1) % len(l.b)
	l.size--
	return copy(p, []byte(msg)), nil
}

func (l *Logger) write(msg string) {
	if debug {
		fmt.Println(msg)
	}
	l.b[(l.begin+l.size)%len(l.b)] = msg
	if l.size < len(l.b) {
		l.size++
	} else {
		l.begin = (l.begin + 1) % len(l.b)
	}
	select {
	case l.ch <- 0:
	default:
	}
}

func (l *Logger) Info(format string, a ...interface{}) {
	l.write("[INFO]  " + fmt.Sprintf(format, a...))
}

func (l *Logger) Warning(format string, a ...interface{}) {
	l.write("[WARN]  " + fmt.Sprintf(format, a...))
}

func (l *Logger) Error(format string, a ...interface{}) {
	l.write("[ERROR] " + fmt.Sprintf(format, a...))
}

func (l *Logger) Fatal(format string, a ...interface{}) {
	l.write("[FATAL] " + fmt.Sprintf(format, a...))
}
