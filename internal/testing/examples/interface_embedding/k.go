package interface_embedding

import (
	"hash"
)

type Hasher111 interface {
	hash.Hash

	Hash([]byte) []byte
}
