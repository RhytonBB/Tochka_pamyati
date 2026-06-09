package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2idParams struct {
	TimeCost uint32
	MemCost  uint32
	Threads  uint8
	KeyLen   uint32
}

func DefaultArgon2idParams() Argon2idParams {
	return Argon2idParams{
		TimeCost: 2,
		MemCost:  64 * 1024,
		Threads:  2,
		KeyLen:   32,
	}
}

func HashArgon2idPHC(password string, params Argon2idParams) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, params.TimeCost, params.MemCost, params.Threads, params.KeyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.MemCost,
		params.TimeCost,
		params.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyArgon2idPHC(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[1] != "argon2id" {
		return false
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false
	}

	memCost, timeCost, threads, ok := parseArgon2Params(parts[3])
	if !ok {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return false
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) < 16 {
		return false
	}

	derived := argon2.IDKey([]byte(password), salt, timeCost, memCost, threads, uint32(len(hash)))
	return subtleConstantTimeBytesEq(derived, hash)
}

func parseArgon2Params(s string) (mem uint32, timeCost uint32, threads uint8, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	var m uint64
	var t uint64
	var p uint64
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "m=") {
			v := strings.TrimPrefix(part, "m=")
			n, err := parseUint(v)
			if err != nil {
				return 0, 0, 0, false
			}
			m = n
		} else if strings.HasPrefix(part, "t=") {
			v := strings.TrimPrefix(part, "t=")
			n, err := parseUint(v)
			if err != nil {
				return 0, 0, 0, false
			}
			t = n
		} else if strings.HasPrefix(part, "p=") {
			v := strings.TrimPrefix(part, "p=")
			n, err := parseUint(v)
			if err != nil {
				return 0, 0, 0, false
			}
			p = n
		} else {
			return 0, 0, 0, false
		}
	}
	if m == 0 || t == 0 || p == 0 || p > 255 {
		return 0, 0, 0, false
	}
	return uint32(m), uint32(t), uint8(p), true
}

func parseUint(s string) (uint64, error) {
	var n uint64
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + uint64(ch-'0')
	}
	return n, nil
}

func subtleConstantTimeBytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
