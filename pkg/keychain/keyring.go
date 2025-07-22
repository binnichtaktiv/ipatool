package keychain

import (
	"encoding/json"
	"os"
	"sync"
)

type Item struct {
	Key         string
	Data        []byte
	Label       string
	Description string
}

// JSONKeyring implements Keyring interface using a JSON file
type JSONKeyring struct {
	filePath string
	data     map[string]Item
	mu       sync.Mutex
}

//go:generate go run go.uber.org/mock/mockgen -source=keyring.go -destination=keyring_mock.go -package keychain
type Keyring interface {
	Get(key string) (Item, error)
	Set(item Item) error
	Remove(key string) error
}

// NewJSONKeyring creates a new JSON-based keyring
func NewJSONKeyring(filePath string) (*JSONKeyring, error) {
	keyring := &JSONKeyring{
		filePath: filePath,
		data:     make(map[string]Item),
	}

	if _, err := os.Stat(filePath); err == nil {
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(fileData, &keyring.data); err != nil {
			return nil, err
		}
	}

	return keyring, nil
}

func (k *JSONKeyring) save() error {
	data, err := json.MarshalIndent(k.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(k.filePath, data, 0600)
}

func (k *JSONKeyring) Get(key string) (Item, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	item, exists := k.data[key]
	if !exists {
		return Item{}, os.ErrNotExist
	}
	return item, nil
}

func (k *JSONKeyring) Set(item Item) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.data[item.Key] = item
	return k.save()
}

func (k *JSONKeyring) Remove(key string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	delete(k.data, key)
	return k.save()
}
