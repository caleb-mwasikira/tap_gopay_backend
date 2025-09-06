package encrypt

import (
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Key struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	Salt    []byte
	Key     []byte
}

func NewArgon2Key(
	memory, timeTaken uint32,
	threads uint8, salt, key []byte,
) (*Argon2Key, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("Argon2Key missing derived key data")
	}

	return &Argon2Key{
		Memory:  memory,
		Time:    timeTaken,
		Threads: threads,
		Salt:    salt,
		Key:     key,
	}, nil
}

func (key Argon2Key) String() string {
	encodedKey := fmt.Sprintf("$id=argon2id$version=%v$memory=%v$time=%v$threads=%v$salt=%v$hash=%v",
		argon2.Version, key.Memory, key.Time, key.Threads,
		base64.StdEncoding.EncodeToString(key.Salt),
		base64.StdEncoding.EncodeToString(key.Key),
	)
	return encodedKey
}

// Generates a 32 byte password using the Argon2 KDF.
// Returns an Argon2Key and an error (if any).
// Key is encoded in the format:
//
//	$id=argon2id$version=%d$memory=%d$time=%d$threads=%d$salt=b64-encoded$key=b64-encoded
//
// Note: salt and key are base64 encoded values
func DeriveKey(password string, salt []byte) (*Argon2Key, error) {
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Argon2 parameters
	const (
		timeTaken uint32 = 1
		memory    uint32 = 64 * 1024
		threads   uint8  = 4
		keyLen    uint32 = 32
	)

	// Derive key
	derivedKey := argon2.IDKey(
		[]byte(password), salt, timeTaken, memory, threads, keyLen,
	)
	return NewArgon2Key(
		memory, timeTaken, threads, salt, derivedKey,
	)
}
