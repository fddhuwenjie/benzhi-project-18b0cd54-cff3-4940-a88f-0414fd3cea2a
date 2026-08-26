package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Digest(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func MustDigest(value any) string {
	d, err := Digest(value)
	if err != nil {
		panic(err)
	}
	return d
}
