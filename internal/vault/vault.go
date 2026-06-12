package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

const (
	PINHashMemory  = 64 * 1024
	PINHashTime    = 3
	PINHashThreads = 2
	PINHashKeyLen  = 32
	TOTPDigits     = 6
	TOTPStep       = 30 * time.Second
)

var (
	ErrInvalidPIN  = errors.New("invalid PIN/password")
	ErrInvalidCode = errors.New("invalid authenticator code")
)

type Metadata struct {
	Configured     bool   `json:"configured"`
	PINSalt        string `json:"pinSalt"`
	PINVerifier    string `json:"pinVerifier"`
	LocalSalt      string `json:"localSalt"`
	SealedVaultKey string `json:"sealedVaultKey"`
	VaultKeyNonce  string `json:"vaultKeyNonce"`
	SealedTOTP     string `json:"sealedTotp"`
	TOTPNonce      string `json:"totpNonce"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
	FailedAttempts int    `json:"failedAttempts"`
	LastFailedAt   int64  `json:"lastFailedAt"`
}

type SetupBundle struct {
	Secret      string `json:"secret"`
	OtpauthURL  string `json:"otpauthUrl"`
	QRCodeSVG   string `json:"qrCodeSvg"`
	ManualKey   string `json:"manualKey"`
	AccountName string `json:"accountName"`
	Issuer      string `json:"issuer"`
}

type PlainEntry struct {
	Title       string `json:"title,omitempty"`
	ContentType string `json:"contentType"`
	Text        string `json:"text,omitempty"`
	ImagePNG    []byte `json:"imagePng,omitempty"`
	ImageThumb  string `json:"imageThumb,omitempty"`
	ImageW      int    `json:"imageW,omitempty"`
	ImageH      int    `json:"imageH,omitempty"`
}

func NewSetupBundle(account string) (SetupBundle, error) {
	secretBytes, err := RandomBytes(20)
	if err != nil {
		return SetupBundle{}, err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	issuer := "clipd"
	if strings.TrimSpace(account) == "" {
		account = "Private Vault"
	}
	otpauth := OtpauthURL(issuer, account, secret)
	qrHTML, err := qrImageHTML(otpauth)
	if err != nil {
		return SetupBundle{}, err
	}
	return SetupBundle{
		Secret:      secret,
		OtpauthURL:  otpauth,
		QRCodeSVG:   qrHTML,
		ManualKey:   groupedSecret(secret),
		AccountName: account,
		Issuer:      issuer,
	}, nil
}

func NewMetadata(pin, totpSecret string) (Metadata, []byte, error) {
	pinSalt, err := RandomBytes(16)
	if err != nil {
		return Metadata{}, nil, err
	}
	localSalt, err := RandomBytes(16)
	if err != nil {
		return Metadata{}, nil, err
	}
	vaultKey, err := RandomBytes(32)
	if err != nil {
		return Metadata{}, nil, err
	}
	localKey := LocalKey(localSalt)
	sealedVaultKey, vaultNonce, err := Seal(localKey, vaultKey)
	if err != nil {
		return Metadata{}, nil, err
	}
	sealedTOTP, totpNonce, err := Seal(localKey, []byte(totpSecret))
	if err != nil {
		return Metadata{}, nil, err
	}
	now := time.Now().Unix()
	return Metadata{
		Configured:     true,
		PINSalt:        base64.StdEncoding.EncodeToString(pinSalt),
		PINVerifier:    base64.StdEncoding.EncodeToString(PINVerifier(pin, pinSalt)),
		LocalSalt:      base64.StdEncoding.EncodeToString(localSalt),
		SealedVaultKey: base64.StdEncoding.EncodeToString(sealedVaultKey),
		VaultKeyNonce:  base64.StdEncoding.EncodeToString(vaultNonce),
		SealedTOTP:     base64.StdEncoding.EncodeToString(sealedTOTP),
		TOTPNonce:      base64.StdEncoding.EncodeToString(totpNonce),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, vaultKey, nil
}

func (m Metadata) VaultKey() ([]byte, error) {
	localSalt, err := base64.StdEncoding.DecodeString(m.LocalSalt)
	if err != nil {
		return nil, err
	}
	sealed, err := base64.StdEncoding.DecodeString(m.SealedVaultKey)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(m.VaultKeyNonce)
	if err != nil {
		return nil, err
	}
	return Open(LocalKey(localSalt), sealed, nonce)
}

func (m Metadata) TOTPSecret() (string, error) {
	localSalt, err := base64.StdEncoding.DecodeString(m.LocalSalt)
	if err != nil {
		return "", err
	}
	sealed, err := base64.StdEncoding.DecodeString(m.SealedTOTP)
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(m.TOTPNonce)
	if err != nil {
		return "", err
	}
	plain, err := Open(LocalKey(localSalt), sealed, nonce)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (m Metadata) VerifyPIN(pin string) bool {
	salt, err := base64.StdEncoding.DecodeString(m.PINSalt)
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(m.PINVerifier)
	if err != nil {
		return false
	}
	got := PINVerifier(pin, salt)
	return hmac.Equal(got, want)
}

func (m Metadata) WithNewPIN(pin string) (Metadata, error) {
	salt, err := RandomBytes(16)
	if err != nil {
		return Metadata{}, err
	}
	m.PINSalt = base64.StdEncoding.EncodeToString(salt)
	m.PINVerifier = base64.StdEncoding.EncodeToString(PINVerifier(pin, salt))
	m.UpdatedAt = time.Now().Unix()
	m.FailedAttempts = 0
	m.LastFailedAt = 0
	return m, nil
}

func PINVerifier(pin string, salt []byte) []byte {
	return argon2.IDKey([]byte(pin), salt, PINHashTime, PINHashMemory, PINHashThreads, PINHashKeyLen)
}

func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func LocalKey(salt []byte) []byte {
	seed := sha256.Sum256([]byte("clipd-private-vault-local-protection"))
	kdf := hkdf.New(sha256.New, seed[:], salt, []byte("vault-local-key"))
	out := make([]byte, 32)
	if _, err := kdf.Read(out); err != nil {
		panic(err)
	}
	return out
}

func Seal(key, plain []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce, err := RandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, plain, nil), nonce, nil
}

func Open(key, sealed, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, sealed, nil)
}

func SealEntry(key []byte, entry PlainEntry) ([]byte, []byte, error) {
	plain, err := json.Marshal(entry)
	if err != nil {
		return nil, nil, err
	}
	return Seal(key, plain)
}

func OpenEntry(key, sealed, nonce []byte) (PlainEntry, error) {
	plain, err := Open(key, sealed, nonce)
	if err != nil {
		return PlainEntry{}, err
	}
	var out PlainEntry
	if err := json.Unmarshal(plain, &out); err != nil {
		return PlainEntry{}, err
	}
	return out, nil
}

func ValidTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != TOTPDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	for skew := -1; skew <= 1; skew++ {
		at := now.Add(time.Duration(skew) * TOTPStep)
		want, err := TOTPCode(secret, at)
		if err == nil && hmac.Equal([]byte(want), []byte(code)) {
			return true
		}
	}
	return false
}

func TOTPCode(secret string, at time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return "", err
	}
	counter := uint64(at.Unix() / int64(TOTPStep.Seconds()))
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	mod := uint32(math.Pow10(TOTPDigits))
	return fmt.Sprintf("%0"+strconv.Itoa(TOTPDigits)+"d", binCode%mod), nil
}

func OtpauthURL(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", strconv.Itoa(TOTPDigits))
	q.Set("period", strconv.Itoa(int(TOTPStep.Seconds())))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

func EncodeMetadata(m Metadata) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeMetadata(s string) (Metadata, error) {
	if strings.TrimSpace(s) == "" {
		return Metadata{}, nil
	}
	var m Metadata
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

func HashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return "t:" + hex.EncodeToString(h[:])
}

func HashImage(png []byte) string {
	h := sha256.Sum256(png)
	return "i:" + hex.EncodeToString(h[:])
}

func groupedSecret(secret string) string {
	var parts []string
	for len(secret) > 4 {
		parts = append(parts, secret[:4])
		secret = secret[4:]
	}
	if secret != "" {
		parts = append(parts, secret)
	}
	return strings.Join(parts, " ")
}

func qrImageHTML(data string) (string, error) {
	q, err := qrcode.New(data, qrcode.High)
	if err != nil {
		return "", err
	}
	// Fixed square PNG — scales cleanly inside the vault QR frame.
	png, err := q.PNG(256)
	if err != nil {
		return "", err
	}
	src := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	return `<img alt="Authenticator setup QR code" width="256" height="256" src="` + src + `"/>`, nil
}
