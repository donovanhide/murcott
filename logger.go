package murcott

import (
	"fmt"
)

type Logger struct {
	ch chan string
}

func NewLogger() *Logger {
	return &Logger{
		ch: make(chan string),
	}
}

func (p *Logger) Channel() <-chan string {
	return p.ch
}

func (p *Logger) write(msg string) {
	go func(ch chan<- string) {
		ch <- msg
	}(p.ch)
}

func (p *Logger) Info(format string, a ...interface{}) {
	p.write("[INFO]  " + fmt.Sprintf(format, a...))
}

func (p *Logger) Warning(format string, a ...interface{}) {
	p.write("[WARN]  " + fmt.Sprintf(format, a...))
}

func (p *Logger) Error(format string, a ...interface{}) {
	p.write("[ERROR] " + fmt.Sprintf(format, a...))
}

func (p *Logger) Fatal(format string, a ...interface{}) {
	p.write("[FATAL] " + fmt.Sprintf(format, a...))
}
