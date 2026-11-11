package mercadopago

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled           bool
	LiveEnabled       bool
	Environment       string
	ClientID          string
	ClientSecret      string
	PlatformToken     string
	PlatformCollector string
	WebhookSecret     string
	PublicOrigin      string
	APIOrigin         string
	KeyID             string
	Keys              map[string]string
}

func LoadConfig() Config {
	c := Config{Enabled: os.Getenv("MP_ENABLED") == "true", LiveEnabled: os.Getenv("MP_LIVE_ENABLED") == "true", Environment: os.Getenv("MP_ENVIRONMENT"), ClientID: os.Getenv("MP_CLIENT_ID"), ClientSecret: os.Getenv("MP_CLIENT_SECRET"), PlatformToken: os.Getenv("MP_PLATFORM_ACCESS_TOKEN"), PlatformCollector: os.Getenv("MP_PLATFORM_COLLECTOR_ID"), WebhookSecret: os.Getenv("MP_WEBHOOK_SECRET"), PublicOrigin: strings.TrimRight(os.Getenv("PUBLIC_SITE_URL"), "/"), APIOrigin: strings.TrimRight(os.Getenv("MP_PUBLIC_ORIGIN"), "/"), KeyID: os.Getenv("MP_ENCRYPTION_KEY_ID"), Keys: map[string]string{}}
	if c.Environment == "" {
		c.Environment = "sandbox"
	}
	if c.KeyID == "" {
		c.KeyID = "v1"
	}
	if key := os.Getenv("MP_ENCRYPTION_KEY"); key != "" {
		c.Keys[c.KeyID] = key
	}
	if raw := os.Getenv("MP_ENCRYPTION_KEYS"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &c.Keys)
	}
	return c
}
func (c Config) ready() bool {
	return c.Enabled && (c.Environment == "sandbox" || c.Environment == "production" && c.LiveEnabled) && validOrigin(c.APIOrigin) && validOrigin(c.PublicOrigin) && c.APIOrigin == c.PublicOrigin && c.WebhookSecret != ""
}
func validOrigin(value string) bool {
	u, e := url.Parse(value)
	return e == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path == "" && (u.Port() == "" || u.Port() == "443")
}
func (c Config) seal(value, domain string) (string, error) {
	key, e := base64.StdEncoding.DecodeString(c.Keys[c.KeyID])
	if e != nil || len(key) != 32 {
		return "", errors.New("missing encryption key")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	a, _ := cipher.NewGCM(block)
	nonce := make([]byte, a.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return c.KeyID + "." + base64.RawURLEncoding.EncodeToString(a.Seal(nonce, nonce, []byte(value), []byte(domain))), nil
}
func (c Config) open(value, domain string) (string, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid ciphertext")
	}
	key, e := base64.StdEncoding.DecodeString(c.Keys[parts[0]])
	if e != nil || len(key) != 32 {
		return "", errors.New("unknown encryption key")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	a, _ := cipher.NewGCM(block)
	b, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || len(b) < a.NonceSize() {
		return "", errors.New("invalid ciphertext")
	}
	plain, e := a.Open(nil, b[:a.NonceSize()], b[a.NonceSize():], []byte(domain))
	return string(plain), e
}
func hash(value string) string { b := sha256.Sum256([]byte(value)); return hex.EncodeToString(b[:]) }
func random() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func jsonString(v any) string { b, _ := json.Marshal(v); return string(b) }

var resourcePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,120}$`)
var operationPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,160}$`)
var moneyPattern = regexp.MustCompile(`^[0-9]{1,14}(\.[0-9]{1,2})?$`)

func money(value string) (*big.Rat, error) {
	if !moneyPattern.MatchString(value) {
		return nil, errors.New("invalid monetary amount")
	}
	n, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, errors.New("invalid monetary amount")
	}
	return n, nil
}
func positiveMoney(value string) bool { n, e := money(value); return e == nil && n.Sign() > 0 }
func sameMoney(a, b string) bool {
	x, e := money(a)
	y, f := money(b)
	return e == nil && f == nil && x.Cmp(y) == 0
}
func subtract(a, b string) (string, error) {
	x, e := money(a)
	y, f := money(b)
	if e != nil || f != nil {
		return "", errors.New("invalid money")
	}
	x.Sub(x, y)
	if x.Sign() < 0 {
		return "", errors.New("negative money")
	}
	return x.FloatString(2), nil
}
func plus(a, b string) (string, error) {
	x, e := money(a)
	y, f := money(b)
	if e != nil || f != nil {
		return "", errors.New("invalid money")
	}
	return x.Add(x, y).FloatString(2), nil
}

type Provider struct {
	client *http.Client
	base   string
}

func NewProvider() *Provider {
	return &Provider{base: "https://api.mercadopago.com", client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type providerFailure struct {
	Status    int
	Ambiguous bool
}

func (e *providerFailure) Error() string { return "provider operation unconfirmed" }
func (p *Provider) callValue(ctx context.Context, method, path, token, key string, input any) (any, error) {
	var body io.Reader
	if input != nil {
		b, e := json.Marshal(input)
		if e != nil {
			return nil, e
		}
		body = bytes.NewReader(b)
	}
	r, e := http.NewRequestWithContext(ctx, method, p.base+path, body)
	if e != nil {
		return nil, e
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	if key != "" {
		r.Header.Set("X-Idempotency-Key", key)
	}
	response, e := p.client.Do(r)
	if e != nil {
		return nil, &providerFailure{Ambiguous: method != "GET"}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, &providerFailure{Status: response.StatusCode, Ambiguous: method != "GET" && (response.StatusCode >= 500 || response.StatusCode == 429 || response.StatusCode == 408)}
	}
	d := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	d.UseNumber()
	var result any
	if e = d.Decode(&result); e != nil || result == nil {
		return nil, &providerFailure{Ambiguous: method != "GET"}
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, &providerFailure{Ambiguous: method != "GET"}
	}
	return result, nil
}
func (p *Provider) call(ctx context.Context, method, path, token, key string, input any) (map[string]any, error) {
	v, e := p.callValue(ctx, method, path, token, key, input)
	if e != nil {
		return nil, e
	}
	result, ok := v.(map[string]any)
	if !ok {
		return nil, &providerFailure{Ambiguous: method != "GET"}
	}
	return result, nil
}
func text(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	}
	return ""
}
func object(v any) map[string]any               { result, _ := v.(map[string]any); return result }
func providerTime(value any) (time.Time, error) { return time.Parse(time.RFC3339Nano, text(value)) }
func checkoutURL(raw string, environment string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	return u.Host == "www.mercadopago.com.ar" || u.Host == "www.mercadopago.com" || environment == "sandbox" && (u.Host == "sandbox.mercadopago.com.ar" || u.Host == "sandbox.mercadopago.com")
}
func signatureValid(signature, requestID, id, secret string, now time.Time) bool {
	if !resourcePattern.MatchString(id) || !operationPattern.MatchString(requestID) || len(signature) > 512 || secret == "" {
		return false
	}
	parts := strings.Split(signature, ",")
	if len(parts) != 2 {
		return false
	}
	values := map[string]string{}
	for _, part := range parts {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 || values[pair[0]] != "" {
			return false
		}
		values[pair[0]] = pair[1]
	}
	if len(values) != 2 || values["ts"] == "" || values["v1"] == "" {
		return false
	}
	ts, e := strconv.ParseInt(values["ts"], 10, 64)
	if e != nil {
		return false
	}
	seconds := ts
	if ts > 1e12 {
		seconds = ts / 1000
	}
	delta := now.Unix() - seconds
	if delta < -300 || delta > 300 {
		return false
	}
	received, e := hex.DecodeString(values["v1"])
	if e != nil || len(received) != 32 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "id:%s;request-id:%s;ts:%s;", strings.ToLower(id), requestID, values["ts"])
	return subtle.ConstantTimeCompare(received, mac.Sum(nil)) == 1
}
