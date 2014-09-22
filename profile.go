package murcott

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"reflect"

	"github.com/vmihailenco/msgpack"
)

type UserProfile struct {
	Nickname  string            `msgpack:"nickname"`
	Avatar    UserAvatar        `msgpack:"avatar"`
	Extension map[string]string `msgpack:"ext"`
}

type UserAvatar struct {
	Image image.Image
}

func init() {
	msgpack.Register(reflect.TypeOf(UserAvatar{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			avatar := v.Interface().(UserAvatar)
			if avatar.Image == nil {
				return e.Encode(map[string]string{})
			}
			var b bytes.Buffer
			err := png.Encode(&b, avatar.Image)
			if err != nil {
				return err
			}
			return e.Encode(map[string]string{
				"type": "png",
				"data": string(b.Bytes()),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return nil
			}
			if m, ok := i.(map[interface{}]interface{}); ok {
				if t, ok := m["type"].(string); ok {
					if data, ok := m["data"].(string); ok {
						if t == "png" {
							b := bytes.NewBuffer([]byte(data))
							img, err := png.Decode(b)
							if err != nil {
								return err
							}
							v.Set(reflect.ValueOf(UserAvatar{Image: img}))
						} else {
							errors.New("unsupported image type")
						}
					}
				}
			} else {
				v.Set(reflect.ValueOf(UserAvatar{}))
			}
			return nil
		})
}
