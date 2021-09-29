package scitokens

import (
	"fmt"
	"io"
	"strings"

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

// GetScopes parses the scope claim and returns a list of all scopes.
func GetScopes(t jwt.Token) ([]Scope, error) {
	scopeint, ok := t.Get("scope")
	if !ok {
		return nil, &TokenValidationError{fmt.Errorf("scope claim missing")}
	}
	scopestr, ok := scopeint.(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast scopes claim to string")
	}
	scopestrs := strings.Split(scopestr, " ")
	scopes := make([]Scope, len(scopestrs))
	for _, s := range scopestrs {
		scopes = append(scopes, ParseScope(s))
	}
	return scopes, nil
}

// GetGroups parses the wlcg.groups claim and returns a list of all groups.
func GetGroups(t jwt.Token) ([]string, error) {
	groupint, ok := t.Get("wlcg.groups")
	if !ok {
		return nil, &TokenValidationError{fmt.Errorf("wlcg.groups claim missing")}
	}
	groupints, ok := groupint.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to cast wlcg.groups claim to slice")
	}
	groups := make([]string, len(groupints))
	for _, g := range groupints {
		gs, ok := g.(string)
		if !ok {
			return nil, fmt.Errorf("unable to cast wlcg.group \"%v\" to string", g)
		}
		groups = append(groups, gs)
	}
	return groups, nil
}
