package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters from OWASP's password storage recommendations (19 MiB, 2 passes, 1 thread): modest enough for
// a small server. They are stored in every hash, so changing them later doesn't invalidate existing passwords.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
)

var b64 = base64.RawStdEncoding

var errMalformedHash = errors.New("malformed password hash")

// ErrBusy means a password couldn't be hashed or checked because too many others were for too long.
var ErrBusy = errors.New("too many password checks at once")

// argonSlots bounds the argon2id computations running at once: each takes argonMemory, so a burst of sign-ins could
// otherwise exhaust a small server's memory. One that waits longer than argonWait gives up with ErrBusy.
var (
	argonSlots = make(chan struct{}, 2)
	argonWait  = 5 * time.Second
)

func argonKey(ctx context.Context, password string, salt []byte, passes, memory uint32, threads uint8, keyLen uint32) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, argonWait)
	defer cancel()
	select {
	case argonSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ErrBusy
	}
	defer func() { <-argonSlots }()
	return argon2.IDKey([]byte(password), salt, passes, memory, threads, keyLen), nil
}

// dummyHash has the current parameters and a key no password is known to match. An unknown username's password is
// checked against it, so that refusing it takes as long as refusing a wrong password.
var dummyHash = phc(make([]byte, saltLen), make([]byte, argonKeyLen))

func phc(salt, key []byte) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

// hashPassword returns the password's argon2id hash in the PHC string format:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>
func hashPassword(ctx context.Context, password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := argonKey(ctx, password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	if err != nil {
		return "", err
	}
	return phc(salt, key), nil
}

// checkPassword reports whether password matches hash, a string made by hashPassword.
func checkPassword(ctx context.Context, hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return false, errMalformedHash
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errMalformedHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, errMalformedHash
	}
	got, err := argonKey(ctx, password, salt, time, memory, threads, uint32(len(key)))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(got, key) == 1, nil
}

// TakePasswordChecks takes every slot for password checks, making the ones that follow wait at most wait, and returns
// the function giving them back. It lets the tests of callers see how they handle ErrBusy.
func TakePasswordChecks(wait time.Duration) (release func()) {
	saved := argonWait
	argonWait = wait
	for range cap(argonSlots) {
		argonSlots <- struct{}{}
	}
	return func() {
		for range cap(argonSlots) {
			<-argonSlots
		}
		argonWait = saved
	}
}
