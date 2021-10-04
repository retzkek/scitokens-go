package scitokens

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

var (
	// WellKnown is a list of well-known URL suffixes to check for OAuth server
	// metadata. See
	// https://www.iana.org/assignments/well-known-uris/well-known-uris.xhtml
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-discovery-07
	WellKnown = []string{
		"oauth-authorization-server",
		"openid-configuration",
	}
)

// AuthServerMetadata per
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-discovery-07. Fields
// defined as OPTIONAL that aren't currently used are not included.
type AuthServerMetadata struct {
	Issuer          string   `json:"issuer"`
	AuthURL         string   `json:"authorization_endpoint"`
	TokenURL        string   `json:"token_endpoint"`
	JWKSURL         string   `json:"jwks_uri"`
	RegistrationURL string   `json:"registration_endpoint"`
	UserInfoURL     string   `json:"userinfo_endpoint"`
	Scopes          []string `json:"scopes_supported"`
	ResponseTypes   []string `json:"response_types_supported"`
}

// FetchMetadata retrieves the OAUTH 2.0 authorization server metadata from the
// given URL, which must include the complete well-known path to the resource.
func FetchMetadata(ctx context.Context, urlstring string) (*AuthServerMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlstring, nil)
	if err != nil {
		return nil, err
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	switch {
	case r.StatusCode == http.StatusNotFound:
		return nil, NotFoundError{}
	case r.StatusCode >= 400:
		return nil, fmt.Errorf("%s", r.Status)
	}

	var rr io.Reader = r.Body
	// DEBUG
	//rr = io.TeeReader(r.Body, os.Stderr)

	var meta AuthServerMetadata
	dec := json.NewDecoder(rr)
	err = dec.Decode(&meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// IssuerKeyURL determines the URL for JWKS keys for the issuer, based on its
// OAuth metadata.
func IssuerKeyURL(ctx context.Context, issuer string) (string, error) {
	var meta *AuthServerMetadata
	for _, wk := range WellKnown {
		var err error
		meta, err = FetchMetadata(ctx, issuer+path.Join("/.well-known", wk))
		if err == nil {
			break
		} else {
			switch err.(type) {
			case NotFoundError:
				continue
			}
			return "", fmt.Errorf("status code %s when fetching metadata for %s", err, issuer)
		}
	}
	if meta == nil {
		return "", fmt.Errorf("no server metadata found under %s", issuer)
	}
	return meta.JWKSURL, nil
}

// IssuerKeyManager handles fetching metadata, then fetching, caching, and
// refreshing keys for each issuer, and implements jwt.KeySetProvider, providing
// jwt.Parse... with the appropriate keys for the token issuer.
type IssuerKeyManager struct {
	// issuerKeyURLs is a cache of the JWKSURL for each issuer
	issuerKeyURLs map[string]string
	keysets       *jwk.AutoRefresh
}

// NewIssuerKeyManager initializes a new key manager. The Context controls the
// lifespan of the manager and its underlying objects.
func NewIssuerKeyManager(ctx context.Context) *IssuerKeyManager {
	return &IssuerKeyManager{
		issuerKeyURLs: make(map[string]string),
		keysets:       jwk.NewAutoRefresh(ctx),
	}
}

// AddIssuer determines the JSON Web Keys URL for the given issuer, and adds it
// to the list of issuers managed by this IssueKeyManager and accepted when
// using KeySetFrom() for validating tokens. Keys will be cached and refreshed
// at regular intervals, and can be accessed with GetIssuerKeys().
func (m *IssuerKeyManager) AddIssuer(ctx context.Context, issuer string) error {
	if m.keysets == nil {
		return fmt.Errorf("IssuerKeyManager not initialized")
	}
	if _, ok := m.issuerKeyURLs[issuer]; !ok {
		url, err := IssuerKeyURL(ctx, issuer)
		if err != nil {
			return err
		}
		m.keysets.Configure(url)
		m.issuerKeyURLs[issuer] = url
	}
	return nil
}

// GetIssuerKeys returns all JSON Web Keys for the given issuer, fetching from
// the jwks_uri specified in the issuer's OAuth metadata if necessary. The
// IssuerKeyManager will cache these keys, refreshing them at regular intervals.
// AddIssuer() must be called first for this issuer.
func (m *IssuerKeyManager) GetIssuerKeys(ctx context.Context, issuer string) (jwk.Set, error) {
	if m.keysets == nil {
		return nil, fmt.Errorf("IssuerKeyManager not initialized")
	}
	url, ok := m.issuerKeyURLs[issuer]
	if !ok {
		return nil, fmt.Errorf("issuer %s not in IssuerKeyManager", issuer)
	}
	return m.keysets.Fetch(ctx, url)
}

// KeySetFrom returns the key set for the token, based on the token's issuer.
// The issuer must first be added to the IssuerKeyManager with AddIssuer().
func (m *IssuerKeyManager) KeySetFrom(t jwt.Token) (jwk.Set, error) {
	if m.keysets == nil {
		return nil, fmt.Errorf("IssuerKeyManager not initialized")
	}
	return m.GetIssuerKeys(context.Background(), t.Issuer())
}

// GetIssuerKeys returns all JSON Web Keys for the given issuer, fetching from
// the jwks_uri specified in the issuer's OAuth metadata. This will fetch the
// metadata and keys with every call, use an IssuerKeyManager to cache them for
// long-running processes.
func GetIssuerKeys(ctx context.Context, issuer string) (jwk.Set, error) {
	url, err := IssuerKeyURL(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return jwk.Fetch(ctx, url)
}
