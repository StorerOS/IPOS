package crypto

import (
	"encoding/hex"
	"strings"
)

func ParseMasterKey(envArg string) (KMS, error) {
	values := strings.SplitN(envArg, ":", 2)
	if len(values) != 2 {
		return nil, Errorf("Invalid KMS master key: %s does not contain a ':'", envArg)
	}
	var (
		keyID  = values[0]
		hexKey = values[1]
	)
	if len(hexKey) != 64 {
		return nil, Errorf("Invalid KMS master key: %s not a 32 bytes long HEX value", hexKey)
	}
	var masterKey [32]byte
	if _, err := hex.Decode(masterKey[:], []byte(hexKey)); err != nil {
		return nil, Errorf("Invalid KMS master key: %v", err)
	}
	return NewMasterKey(keyID, masterKey), nil
}
