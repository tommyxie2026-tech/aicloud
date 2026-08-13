package authn

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxMetadataBytes = 1 << 20

type remoteKeySet struct {
	client     *http.Client
	url        string
	cacheTTL   time.Duration
	algorithms []string
	now        func() time.Time

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	keysUntil time.Time
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func newRemoteKeySet(client *http.Client, rawURL string, cacheTTL time.Duration, algorithms []string) *remoteKeySet {
	return &remoteKeySet{
		client:     client,
		url:        rawURL,
		cacheTTL:   cacheTTL,
		algorithms: append([]string(nil), algorithms...),
		now:        time.Now,
		keys:       map[string]*rsa.PublicKey{},
	}
}

func (s *remoteKeySet) verify(ctx context.Context, alg, kid string, signingInput []byte, encodedSignature string) error {
	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return ErrTokenInvalid
	}
	if err := s.verifyOnce(ctx, alg, kid, signingInput, signature, false); err == nil {
		return nil
	}
	// Refresh once to handle normal key rotation. A second failure is final.
	return s.verifyOnce(ctx, alg, kid, signingInput, signature, true)
}

func (s *remoteKeySet) verifyOnce(ctx context.Context, alg, kid string, signingInput, signature []byte, forceRefresh bool) error {
	key, err := s.keyFor(ctx, kid, forceRefresh)
	if err != nil {
		return err
	}
	hash, digest := digestForAlgorithm(alg, signingInput)
	if hash == 0 {
		return ErrTokenInvalid
	}
	return rsa.VerifyPKCS1v15(key, hash, digest, signature)
}

func digestForAlgorithm(alg string, input []byte) (crypto.Hash, []byte) {
	switch alg {
	case "RS256":
		sum := sha256.Sum256(input)
		return crypto.SHA256, sum[:]
	case "RS384":
		sum := sha512.Sum384(input)
		return crypto.SHA384, sum[:]
	case "RS512":
		sum := sha512.Sum512(input)
		return crypto.SHA512, sum[:]
	default:
		return 0, nil
	}
}

func supportedAlgorithm(alg string) bool {
	return alg == "RS256" || alg == "RS384" || alg == "RS512"
}

func (s *remoteKeySet) keyFor(ctx context.Context, kid string, forceRefresh bool) (*rsa.PublicKey, error) {
	if !forceRefresh {
		s.mu.RLock()
		key := s.keys[kid]
		valid := key != nil && s.now().Before(s.keysUntil)
		s.mu.RUnlock()
		if valid {
			return key, nil
		}
	}
	if err := s.refresh(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := s.keys[kid]
	if key == nil {
		return nil, fmt.Errorf("no matching signing key")
	}
	return key, nil
}

func (s *remoteKeySet) refresh(ctx context.Context) error {
	var document jwksDocument
	if err := fetchJSON(ctx, s.client, s.url, &document); err != nil {
		return fmt.Errorf("JWKS refresh failed: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.Kty != "RSA" || item.Kid == "" || (item.Use != "" && item.Use != "sig") {
			continue
		}
		if item.Alg != "" && !contains(s.algorithms, item.Alg) {
			continue
		}
		key, err := rsaPublicKey(item.N, item.E)
		if err != nil || key.N.BitLen() < 2048 {
			continue
		}
		if _, exists := keys[item.Kid]; exists {
			return fmt.Errorf("duplicate JWKS kid %q", item.Kid)
		}
		keys[item.Kid] = key
	}
	if len(keys) == 0 {
		return fmt.Errorf("JWKS contains no acceptable signing keys")
	}
	s.mu.Lock()
	s.keys = keys
	s.keysUntil = s.now().Add(s.cacheTTL)
	s.mu.Unlock()
	return nil
}

func rsaPublicKey(encodedN, encodedE string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(encodedN)
	if err != nil || len(nBytes) == 0 {
		return nil, fmt.Errorf("invalid RSA modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(encodedE)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
		return nil, fmt.Errorf("invalid RSA exponent")
	}
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 | int(b)
	}
	if exponent < 3 || exponent%2 == 0 {
		return nil, fmt.Errorf("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

func fetchJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Request == nil || resp.Request.URL == nil || resp.Request.URL.Scheme != "https" {
		return fmt.Errorf("OIDC metadata redirect left HTTPS")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMetadataBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxMetadataBytes {
		return fmt.Errorf("OIDC metadata response is too large")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	return decoder.Decode(target)
}
