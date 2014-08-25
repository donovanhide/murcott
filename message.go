package murcott

import (
	"errors"
	"mime"
)

type Content struct {
	Mime string `msgpack:"mime"`
	Data string `msgpack:"data"`
}

type ChatMessage struct {
	Contents []Content `msgpack:"contents"`
}

// NewPlainChatMessage generates a new ChatMessage with a plain text.
func NewPlainChatMessage(text string) ChatMessage {
	return NewMimeChatMessage("text/plain", text)
}

// NewHtmlChatMessage generates a new ChatMessage with a html text.
func NewHtmlChatMessage(html string) ChatMessage {
	return NewMimeChatMessage("text/html", html)
}

// NewMimeChatMessage generates a new ChatMessage with the given mimetype.
func NewMimeChatMessage(mimetype string, data string) ChatMessage {
	return NewChatMessage([]Content{
		Content{mimetype, data},
	})
}

// NewChatMessage generates a new ChatMessage with the given Content array.
func NewChatMessage(contents []Content) ChatMessage {
	return ChatMessage{Contents: contents}
}

// Text returns the first text/plain content.
func (m *ChatMessage) Text() string {
	c, _ := m.First("text/plain")
	return c
}

// Html returns the first text/html content.
func (m *ChatMessage) Html() string {
	c, _ := m.First("text/html")
	return c
}

// First returns the first content that has the given mimetype.
func (m *ChatMessage) First(mimetype string) (string, error) {
	for _, c := range m.Contents {
		t, _, err := mime.ParseMediaType(c.Mime)
		if err == nil && t == mimetype {
			return c.Data, nil
		}
	}
	return "", errors.New("not found")
}

// At returns the content at position i in the message.
func (m *ChatMessage) At(i int) (Content, error) {
	if i >= 0 && i < m.Len() {
		return m.Contents[i], nil
	} else {
		return Content{}, errors.New("out of bounds")
	}
}

// PushBack adds a new content to the message.
func (m *ChatMessage) Push(c Content) {
	m.Contents = append(m.Contents, c)
}

// Len returns the number of contents.
func (m *ChatMessage) Len() int {
	return len(m.Contents)
}

type messageAck struct {
}