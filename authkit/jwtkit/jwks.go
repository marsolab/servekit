package jwtkit

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/cristalhq/jwt/v5"
	"github.com/marsolab/servekit/errkit"
)

const defaultClientTimeout = 10 * time.Second

// Key represents a single key in a JWK set.
type Key struct {
	Use string `json:"use"`
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// KeyStore represents a set of keys from a JWKS endpoint.
type KeyStore struct {
	Keys []Key `json:"keys"`
}

// JWKSProvider is a token provider that uses a JWKS endpoint to verify tokens.
type JWKSProvider struct {
	mu         sync.RWMutex
	client     *http.Client
	jwksURL    string
	keyStore   *KeyStore
	keyCache   map[string]*rsa.PublicKey
	refreshInt time.Duration
	lastFetch  time.Time
}

// NewJWKSProvider creates a new JWKSProvider.
func NewJWKSProvider(jwksURL string, refreshInterval time.Duration) (*JWKSProvider, error) {
	p := &JWKSProvider{
		client:     &http.Client{Timeout: defaultClientTimeout},
		jwksURL:    jwksURL,
		keyCache:   make(map[string]*rsa.PublicKey),
		refreshInt: refreshInterval,
	}

	refreshErr := p.refresh(context.Background())
	if refreshErr != nil {
		return nil, fmt.Errorf("initial jwks refresh: %w", refreshErr)
	}

	return p, nil
}

func (p *JWKSProvider) refresh(ctx context.Context) error {
	if p == nil {
		return errors.New("jwks provider is nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastFetch) < p.refreshInt {
		return nil
	}

	if err := p.fetchJWKS(ctx); err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	// Reset the key cache to avoid potential stale data.
	p.keyCache = make(map[string]*rsa.PublicKey)

	for _, key := range p.keyStore.Keys {
		if key.Kty == "RSA" {
			pubKey, err := p.convertKey(key.E, key.N)
			if err != nil {
				continue
			}

			p.keyCache[key.Kid] = pubKey
		}
	}

	p.lastFetch = time.Now()

	return nil
}

func (p *JWKSProvider) fetchJWKS(ctx context.Context) error {
	req, newRequestErr := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, http.NoBody)
	if newRequestErr != nil {
		return fmt.Errorf("create jwks request: %w", newRequestErr)
	}

	resp, doErr := p.client.Do(req)
	if doErr != nil {
		return fmt.Errorf("fetch jwks: %w", doErr)
	}

	if resp == nil {
		return errors.New("fetch jwks: empty response")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&p.keyStore); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	return nil
}

func (*JWKSProvider) convertKey(e, n string) (*rsa.PublicKey, error) {
	decodedE, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	decodedN, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(decodedN),
		E: int(new(big.Int).SetBytes(decodedE).Int64()),
	}

	return pubKey, nil
}

func (p *JWKSProvider) getVerifier(kid string) (jwt.Verifier, error) {
	if p == nil {
		return nil, errors.New("jwks provider is nil")
	}

	refreshErr := p.refresh(context.Background())
	if refreshErr != nil {
		return nil, fmt.Errorf("refresh jwks: %w", refreshErr)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	pubKey, ok := p.keyCache[kid]
	if !ok {
		return nil, fmt.Errorf("kid '%s' not found in jwks", kid)
	}

	verifier, err := jwt.NewVerifierRS(jwt.RS256, pubKey)
	if err != nil {
		return nil, fmt.Errorf("create verifier: %w", err)
	}

	return verifier, nil
}

// ParseVerify parses and verifies a token using the key from the JWKS endpoint.
func (p *JWKSProvider) ParseVerify(token string) (*Token, error) {
	if p == nil {
		return nil, errors.New("jwks provider is nil")
	}

	unverifiedToken, err := jwt.ParseNoVerify([]byte(token))
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token header: %w", err))
	}

	kid := unverifiedToken.Header().KeyID
	if kid == "" {
		return nil, errors.Join(errkit.ErrTokenInvalid, errors.New("missing kid in token header"))
	}

	verifier, err := p.getVerifier(kid)
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, err)
	}

	raw, err := jwt.Parse([]byte(token), verifier)
	if err != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token: %w", err))
	}

	t := Token{
		raw: raw,
	}

	decodeClaimsErr := raw.DecodeClaims(&t)
	if decodeClaimsErr != nil {
		return nil, errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode claims: %w", decodeClaimsErr))
	}

	validateErr := t.Validate(time.Now())
	if validateErr != nil {
		return nil, validateErr
	}

	return &t, nil
}

// ParseVerifyClaims parses and verifies a token using the key from the JWKS endpoint.
func (p *JWKSProvider) ParseVerifyClaims(token string, claims any) error {
	if p == nil {
		return errors.New("jwks provider is nil")
	}

	unverifiedToken, err := jwt.ParseNoVerify([]byte(token))
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token header: %w", err))
	}

	kid := unverifiedToken.Header().KeyID
	if kid == "" {
		return errors.Join(errkit.ErrTokenInvalid, errors.New("missing kid in token header"))
	}

	verifier, err := p.getVerifier(kid)
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, err)
	}

	raw, err := jwt.Parse([]byte(token), verifier)
	if err != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("parse token: %w", err))
	}

	decodeCustomClaimsErr := raw.DecodeClaims(claims)
	if decodeCustomClaimsErr != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode custom claims: %w", decodeCustomClaimsErr))
	}

	t := Token{}

	decodeStandardClaimsErr := raw.DecodeClaims(&t)
	if decodeStandardClaimsErr != nil {
		return errors.Join(errkit.ErrTokenInvalid, fmt.Errorf("decode standard claims: %w", decodeStandardClaimsErr))
	}

	return t.Validate(time.Now())
}

// Verify verifies a token.
func (p *JWKSProvider) Verify(token string) error {
	if p == nil {
		return errors.New("jwks provider is nil")
	}

	_, parseErr := p.ParseVerify(token)
	if parseErr != nil {
		return parseErr
	}

	return nil
}

// Sign is not supported for JWKSProvider.
func (*JWKSProvider) Sign(_ *Token) (string, error) {
	return "", errors.New("signing is not supported by JWKSProvider")
}
