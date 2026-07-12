// hashgen generates a bcrypt hash for a password, using the same hashing
// parameters as the application. Useful for resetting admin passwords
// directly in the database:
//
//	go run ./cmd/hashgen 'my-new-password'
//
// It can also verify a password against an existing hash:
//
//	go run ./cmd/hashgen 'my-password' '$2a$12$...'
package main

import (
	"fmt"
	"os"

	"skintrader-go/internal/pkg/hash"
)

func main() {
	switch len(os.Args) {
	case 2:
		h, err := hash.HashPassword(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Println(h)
	case 3:
		if hash.CheckPassword(os.Args[1], os.Args[2]) {
			fmt.Println("MATCH")
		} else {
			fmt.Println("MISMATCH")
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: hashgen <password> [hash-to-verify-against]")
		os.Exit(2)
	}
}
