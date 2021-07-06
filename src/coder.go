package main

import (
	"context"
	"encoding/hex"
	"strings"

	"go.uber.org/zap"
)

const encryptKey = "abc321"

func encode(key string, source string) string {
	result := ""
	for i, val := range []byte(strings.ToUpper(hex.EncodeToString([]byte(source)))) {
		var c byte
		if len(key) > 0 {
			c = byte((key[i%len(key)])) ^ val
		} else {
			c = val
		}

		result += hex.EncodeToString([]byte{c})
	}
	return result
}

func decode(key string, source string) string {
	decoded, err := hex.DecodeString(source)
	if err != nil {
		msu.Error(context.Background(), err, zap.Any("source", source))
		return ""
	}

	result := []byte{}
	for i, val := range decoded {
		if len(key) > 0 {
			result = append(result, byte(key[i%len(key)])^val)
		}
	}

	decoded, err = hex.DecodeString(string(result))
	if err != nil {
		msu.Error(context.Background(), err, zap.Any("source", source))
		return ""
	}
	return string(decoded)
}
