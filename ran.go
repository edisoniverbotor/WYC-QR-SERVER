package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// Generate a random number, 0..N, returned as a string with 6 to 8 non-zero digits.
func GenRandNumber(nDigits int) (buf string, err error) {
	var n int64
	for {
		binary.Read(rand.Reader, binary.LittleEndian, &n)
		if n < 0 {
			n = -n
		}
		if n > 1000000 {
			break
		}
	}
	n = n % 100000000
	buf = fmt.Sprintf("%08d", n)
	return
}

// Should move to aesccm package
func GenRandBytes(nRandBytes int) (buf []byte, err error) {
	buf = make([]byte, nRandBytes)
	_, err = rand.Read(buf)
	if err != nil {
		fmt.Printf(`{"msg":"Error generaintg random numbers :%s"}\n`, err)
		return nil, err
	}
	return
}

/* vim: set noai ts=4 sw=4: */
