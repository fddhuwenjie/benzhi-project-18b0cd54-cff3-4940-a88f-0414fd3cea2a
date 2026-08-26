package validation

import (
	"encoding/hex"
	"strings"
)

func Required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalid(name + " 不能为空")
	}
	if len(value) > 200 {
		return invalid(name + " 长度超过限制")
	}
	return nil
}

func Digest(value string) error {
	if len(value) != 64 {
		return invalid("evidence_digest 必须是 64 位 SHA-256 十六进制摘要")
	}
	_, err := hex.DecodeString(value)
	if err != nil {
		return invalid("evidence_digest 必须是合法十六进制摘要")
	}
	return nil
}
