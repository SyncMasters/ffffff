// Package security provides conservative output redaction, not secret storage.
package security

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

const Redacted = "[REDACTED]"

// Secret requires explicit Reveal at the integration boundary. Formatting,
// structured logging and JSON/YAML encoding never expose its value.
// Go strings cannot be reliably zeroed; do not retain secrets unnecessarily.
type Secret struct{ value string }

func NewSecret(value string) Secret           { return Secret{value: value} }
func (s Secret) Reveal() string               { return s.value }
func (s Secret) String() string               { return Redacted }
func (s Secret) GoString() string             { return Redacted }
func (s Secret) LogValue() slog.Value         { return slog.StringValue(Redacted) }
func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(Redacted) }
func (s Secret) MarshalYAML() (any, error)    { return Redacted, nil }

func SensitiveKey(key string) bool {
	key = strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(key))
	switch key {
	case "password", "passwd", "passwordhash", "apikey", "xapikey", "authorization", "proxyauthorization", "token", "accesstoken", "refreshtoken", "secret", "clientsecret", "cookie", "setcookie":
		return true
	}
	return false
}

var userinfo = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^\s/@]+@`)
var headers = regexp.MustCompile(`(?im)((?:proxy-)?authorization\s*[:=]\s*)[^\r\n]+`)
var keyed = regexp.MustCompile(`(?i)((?:password|passwd|api[_-]?key|access[_-]?token|token|client[_-]?secret)\s*[=:]\s*)[^&\s,;]+`)

// Redact removes known secrets (including URL-encoded forms), URL userinfo,
// and common header/key-value secrets. It is a safety net, not a substitute for
// keeping secrets out of free-form evidence, metadata and error messages.
func Redact(text string, secrets ...string) string {
	for _, secret := range secrets {
		if secret == "" || secret == Redacted {
			continue
		}
		for _, value := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
			text = strings.ReplaceAll(text, value, Redacted)
		}
	}
	text = userinfo.ReplaceAllString(text, "${1}"+Redacted+"@")
	text = headers.ReplaceAllString(text, "${1}"+Redacted)
	return keyed.ReplaceAllString(text, "${1}"+Redacted)
}

// Fields returns a redacted copy; input maps (including request headers) remain intact.
func Fields(fields map[string]string, secrets ...string) map[string]string {
	if fields == nil {
		return nil
	}
	out := make(map[string]string, len(fields))
	for k, v := range fields {
		if SensitiveKey(k) {
			v = Redacted
		} else {
			v = Redact(v, secrets...)
		}
		out[Redact(k, secrets...)] = v
	}
	return out
}
