package log

import (
	"fmt"
	"os"
)

var debug bool

func init() {
	debug = (len(os.Getenv("GO_MURCOTT_LOGGING")) > 0)
}

type Logger struct {
	ch chan string
}

func NewLogger() *Logger {
	return &Logger{
		ch: make(chan string, 1024),
	}
}

func (l *Logger) Read(p []byte) (n int, err error) {
	msg := <-l.ch
	return copy(p, []byte(msg)), nil
}

func (l *Logger) write(msg string) {
	if debug {
		fmt.Println(msg)
	}
	l.ch <- msg
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
