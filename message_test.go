package murcott

import (
	"testing"
)

func TestChatMessage(t *testing.T) {
	text := "plain"
	plainmsg := NewPlainChatMessage(text)
	if plainmsg.Text() != text {
		t.Errorf("Text() returns wrong value: %s; expects %s", plainmsg.Text(), text)
	}

	html := "<html></html>"
	htmlmsg := NewHTMLChatMessage(html)
	if htmlmsg.HTML() != html {
		t.Errorf("HTML() returns wrong value: %s; expects %s", htmlmsg.Text(), html)
	}

	contents := []Content{
		Content{"image/png", "png image1"},
		Content{"text/plain; charset=ISO-2022-JP", "plain text1"},
		Content{"image/png", "png image2"},
		Content{"text/plain", "plain text2"},
	}

	multi := NewChatMessage(contents)
	multi.Push(Content{"application/json", "{}"})

	if multi.Len() != 5 {
		t.Errorf("Len() returns wrong value: %d; expects %d", multi.Len(), 5)
	}

	data, err := multi.First("text/plain")
	if err != nil {
		t.Errorf("First(\"text/plain\") returns error: %v", err)
	}
	if data != contents[1].Data {
		t.Errorf("First(\"text/plain\") returns wrong value: %s; expects %s", data, contents[1].Data)
	}

	data, err = multi.First("image/png")
	if err != nil {
		t.Errorf("First(\"image/png\") returns error: %v", err)
	}
	if data != contents[0].Data {
		t.Errorf("First(\"image/png\") returns wrong value: %s; expects %s", data, contents[0].Data)
	}

	data, err = multi.First("application/json")
	if err != nil {
		t.Errorf("First(\"application/json\") returns error: %v", err)
	}
	if data != "{}" {
		t.Errorf("First(\"application/json\") returns wrong value: %s; expects %s", data, "{}")
	}

	_, err = multi.First("application/xml")
	if err == nil {
		t.Errorf("First(\"application/xml\") should return error")
	}
}
