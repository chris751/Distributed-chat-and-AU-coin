package rsa

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

// example of how to use libary
// returns a big n and d
// func GenerateKeys() (*big.Int, *big.Int) {

// 	// message := n
// 	// c := rsa.Encrypt(message, n)

// 	// signature := rsa.NewGenerateSignature(message, d, n)
// 	// if rsa.NewVerifySignature(signature, message, n) {
// 	// 	rsa.Decrypt(c, d, n)
// 	// }

// 	fmt.Println("My public key is n:", rsa.Hash(n))

// 	return n, d
// }

var e *big.Int = big.NewInt(3) // assignment says that e should be 3

// KeyGen generates two prime numbers, p, q.KeyGen
// The product of p, q is of bit length k
// returns n and d
func KeyGen() (*big.Int, *big.Int) {
	n, _ := rand.Prime(rand.Reader, 2) // put some value in n to avoid nil pointer
	p, _ := rand.Prime(rand.Reader, 2) // put some value in n to avoid nil pointer
	q, _ := rand.Prime(rand.Reader, 2) // put some value in n to avoid nil pointer
	d, _ := rand.Prime(rand.Reader, 2) // put some value in n to avoid nil pointer
	bitLengthP := 1024
	bitLengthQ := 1024
	shouldSwap := false // switch between incrementing p and q

	// loop stops when n = k bit length is found
	for {
		if !shouldSwap {
			bitLengthP++
			shouldSwap = true
		} else {
			bitLengthQ++
			shouldSwap = false
		}
		// if n.BitLen() > m.BitLen() {
		// 	bitLengthP = bitLengthP - 10
		// 	bitLengthQ = bitLengthQ - 10
		// }
		p, _ = rand.Prime(rand.Reader, bitLengthP)
		q, _ = rand.Prime(rand.Reader, bitLengthQ)
		// fmt.Println("m/2", m.BitLen()/2)
		n = n.Mul(p, q) // calculate the product of p, q and save in n
		// fmt.Println("n", n.BitLen())
		// fmt.Println("m", m.BitLen())

		// fmt.Println("length was the same")
		pMin1 := bigInt().Sub(p, big.NewInt(1))
		qMin1 := bigInt().Sub(q, big.NewInt(1))
		r := bigInt().Mul(pMin1, qMin1) // (p-1)*(q-1)
		// fmt.Println("p' Greatest common divisor is: ", bigInt().GCD(nil, nil, e, pMin1))
		// fmt.Println("Q' Greatest common divisor is: ", bigInt().GCD(nil, nil, e, qMin1))

		d = bigInt().ModInverse(e, r) // inverse of e modulo (p−1)(q−1),
		if d != nil {
			break
		} else {
			bitLengthP = 1024
			bitLengthQ = 1024
			shouldSwap = false
		}
	}
	return n, d
}

// c = m^e mod n
// returns c
func Encrypt(message *big.Int, n *big.Int) *big.Int {
	fmt.Println("Encrypting...")
	c := bigInt().Exp(message, e, n) // m^e mod n
	return c
}

// takes message, secret key and public key
// returns signature
func GenerateSignature(m *big.Int, d *big.Int, n *big.Int) *big.Int {
	fmt.Println("Generating signature...")
	m = Hash(m)
	signature := bigInt().Exp(m, d, n)
	return signature
}

// takes signature, message, and public key
// returns true if valid and false if invalid
func VerifySignature(signature *big.Int, m *big.Int, n *big.Int) bool {
	fmt.Println("Verifying signature")
	// fmt.Println("signature:", signature)
	// fmt.Println("m:", m)
	// fmt.Println("n:", n)
	m = Hash(m)
	s := bigInt().Exp(signature, e, n)

	if s.Cmp(m) == 0 {
		fmt.Println("Signature and message is valid")
		return true
	} else {
		fmt.Println("Signature:", s)
		fmt.Println("Message:  ", m)
		fmt.Println("Message is invalid")
		return false
	}
}

// t = c^d mod n
// returns c,
func Decrypt(c *big.Int, d *big.Int, n *big.Int) *big.Int {
	fmt.Println("decrypting...")
	// fmt.Println("c", c)
	// fmt.Println("d", d)
	// fmt.Println("n", n)
	plaintext := c.Exp(c, d, n)
	fmt.Println("plaintext:", plaintext)
	return plaintext
}

func bigInt() *big.Int {
	return big.NewInt(42)
}

func Hash(m *big.Int) *big.Int {
	hash := sha256.New()
	hash.Write([]byte(m.Bytes()))
	return bigInt().SetBytes(hash.Sum(nil))

}
