package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"math/rand"
	"time"
)

func BigInt() *big.Int {
	return big.NewInt(42)
}

// Code from here https://gist.github.com/ryanfitz/4191392
func DoEvery(d time.Duration, f func()) {
	for range time.Tick(d) {
		f()
	}
}

// Picks winner that should add add the block to the chain
func PickWinner() {

}

func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

func RandomNumber() int {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	return r1.Intn(100000000) // 1 in a billion
}
