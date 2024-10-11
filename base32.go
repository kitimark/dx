package dx

import "encoding/base32"

var b32encoding *base32.Encoding

func init() {
	b32encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").
		WithPadding(base32.NoPadding)
}

func b32en(name string) string {
	return b32encoding.EncodeToString([]byte(name))
}

func b32de(encode string) (string, error) {
	b, err := b32encoding.DecodeString(encode)
	return string(b), err
}
