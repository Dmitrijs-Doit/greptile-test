package domain

import (
	"crypto/sha1"
	"encoding/binary"
	"math/rand"
	"strings"
)

func HashCustomerIdIntoABillingAccountId(customerID string) string {
	hexLetters := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}
	r := rand.New(rand.NewSource(Hash(customerID)))
	r.Shuffle(len(hexLetters), func(i, j int) { hexLetters[i], hexLetters[j] = hexLetters[j], hexLetters[i] })

	return strings.Join([]string{strings.Join(hexLetters[0:6], ""), strings.Join(hexLetters[3:9], ""), strings.Join(hexLetters[9:], "")}, "-")
}

func GetCustomerHexLetters(customerID string) []string {
	hexLetters := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
	r := rand.New(rand.NewSource(Hash(customerID)))
	r.Shuffle(len(hexLetters), func(i, j int) { hexLetters[i], hexLetters[j] = hexLetters[j], hexLetters[i] })

	return hexLetters
}

func Hash(input string) int64 {
	hash := sha1.New()
	hash.Write([]byte(input))
	inputAsHex := hash.Sum(nil)

	return int64(binary.BigEndian.Uint32(inputAsHex))
}
