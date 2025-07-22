package keychain

import (
	"fmt"
)

func (k *keychain) Set(key string, data []byte) error {
	err := k.keyring.Set(Item{
		Key:  key,
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to set item: %w", err)
	}

	return nil
}
