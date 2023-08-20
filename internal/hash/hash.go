package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func ReadHashFile(path string) ([]byte, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read file %s: %s", path, err)
	}

	h, err := HashBytes(b)
	return b, h, err
}

func HashBytes(b []byte) (string, error) {
	s := sha256.New()
	_, err := s.Write(b)
	if err != nil {
		return "", fmt.Errorf("writing hash: %s", err)
	}

	return hex.EncodeToString(s.Sum(nil)), nil
}
