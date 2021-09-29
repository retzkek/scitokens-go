package scitokens

import (
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/jwt"
)

// PrintToken pretty-prints the token claims to w.
func PrintToken(w io.Writer, t jwt.Token) {
	fmt.Fprintf(w, "Subject: %s\n", t.Subject())
	fmt.Fprintf(w, "Issuer: %s\n", t.Issuer())
	fmt.Fprintf(w, "Audience: %s\n", t.Audience())
	fmt.Fprintf(w, "Issued at: %s, Expires at: %s\n", t.IssuedAt(), t.Expiration())
	fmt.Fprintf(w, "Claims:\n")
	for k, v := range t.PrivateClaims() {
		fmt.Fprintf(w, "\t%v: %v\n", k, v)
	}
}
