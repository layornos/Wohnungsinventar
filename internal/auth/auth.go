// Package auth hasht Passwörter mit argon2id und erzeugt Zufalls-Tokens.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameter nach OWASP-Empfehlung; auf dem Pi 5 dauert ein Hash deutlich unter einer Sekunde.
const (
	argonZeit     = 2
	argonSpeicher = 19 * 1024
	argonThreads  = 1
	argonLaenge   = 32
	saltLaenge    = 16
)

const MinPasswortLaenge = 8

var ErrPasswortZuKurz = fmt.Errorf("Passwort muss mindestens %d Zeichen haben", MinPasswortLaenge)

func PasswortHashen(passwort string) (string, error) {
	if len(passwort) < MinPasswortLaenge {
		return "", ErrPasswortZuKurz
	}
	salt := make([]byte, saltLaenge)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(passwort), salt, argonZeit, argonSpeicher, argonThreads, argonLaenge)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonSpeicher, argonZeit, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func PasswortPruefen(hash, passwort string) (bool, error) {
	teile := strings.Split(hash, "$")
	if len(teile) != 6 || teile[1] != "argon2id" {
		return false, errors.New("unbekanntes Hash-Format")
	}
	var speicher, zeit uint32
	var threads uint8
	if _, err := fmt.Sscanf(teile[3], "m=%d,t=%d,p=%d", &speicher, &zeit, &threads); err != nil {
		return false, err
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(teile[4])
	if err != nil {
		return false, err
	}
	erwartet, err := b64.DecodeString(teile[5])
	if err != nil {
		return false, err
	}
	ist := argon2.IDKey([]byte(passwort), salt, zeit, speicher, threads, uint32(len(erwartet)))
	return subtle.ConstantTimeCompare(ist, erwartet) == 1, nil
}

// DummyHash wird geprüft, wenn es die E-Mail nicht gibt, damit die Antwortzeit nichts verrät.
var DummyHash, _ = PasswortHashen("dummy-passwort-ohne-nutzer")

// NeuesToken liefert ein zufälliges, URL-sicheres Token (256 Bit) und seinen Hash zum Speichern.
func NeuesToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, TokenHash(token), nil
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
