package murcott

import (
	"fmt"
)

type Logger struct {
	ch chan string
}

func newLogger() *Logger {
	return &Logger{
		ch: make(chan string),
	}
}

func (l *Logger) Read(p []byte) (n int, err error) {
	msg := <-l.ch
	return copy(p, []byte(msg)), nil
}

func (l *Logger) write(msg string) {
	go func(ch chan<- string) {
		ch <- msg
	}(l.ch)
}

func (l *Logger) info(format string, a ...interface{}) {
	l.write("[INFO]  " + fmt.Sprintf(format, a...))
}

func (l *Logger) warning(format string, a ...interface{}) {
	l.write("[WARN]  " + fmt.Sprintf(format, a...))
}

func (l *Logger) error(format string, a ...interface{}) {
	l.write("[ERROR] " + fmt.Sprintf(format, a...))
}

func (l *Logger) fatal(format string, a ...interface{}) {
	l.write("[FATAL] " + fmt.Sprintf(format, a...))
}
