package domain

import "crypto/rand"

const publicIDSize = 21

// Alphabet/size preserve references/server/src/common/lib/nanoid.ts behavior.
const publicIDAlphabet = "useandom-26T198340PX75pxJACKVERYMINDBUSHWOLF_GQZbfghjklqvwyzrict"

func NewPublicID() (string, error) {
	var data [publicIDSize]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	for i, b := range data {
		data[i] = publicIDAlphabet[b&63]
	}
	return string(data[:]), nil
}
