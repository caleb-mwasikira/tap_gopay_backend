package encrypt

import (
	"io"
	"math/rand/v2"
	"strings"
)

const (
	SEED_PHRASE_WORD_COUNT int = 3
	SEED_PHRASE_CHAR_COUNT int = 4
)

type SeedPhraseReader struct {
	seed []byte
}

func (s SeedPhraseReader) Read(p []byte) (int, error) {
	if len(s.seed) == 0 {
		return 0, io.EOF
	}

	if len(p) <= len(s.seed) {
		p = s.seed[:len(p)]
		return len(p), nil
	}

	index := 0
	numBytesRead := 0

	// Since seed is smaller than len(p), we are going to circularly
	// read from seed until p is full
	for {
		if numBytesRead == len(p) {
			break
		}
		p[numBytesRead] = s.seed[index]
		index = (index + 1) % len(s.seed)
		numBytesRead++
	}

	return numBytesRead, nil
}

func NewSeedPhraseReader(data []byte) *SeedPhraseReader {
	return &SeedPhraseReader{
		seed: data,
	}
}

func GenerateSeedPhrase() string {
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	ln := len(alphabet)
	str := strings.Builder{}

	for range SEED_PHRASE_WORD_COUNT {
		for range SEED_PHRASE_CHAR_COUNT {
			index := rand.IntN(ln - 1)
			char := alphabet[index]
			str.WriteByte(char)
		}
		str.WriteString(" ")
	}
	return str.String()
}
