package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		fmt.Fprintf(os.Stderr, "Could not generate token: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(base64.RawURLEncoding.EncodeToString(token))
}
