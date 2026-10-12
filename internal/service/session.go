package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{secret: []byte(secret)}
}

func (s *SessionManager) Sign(userID uint) string {
	payload := fmt.Sprintf("%d:%d", userID, time.Now().Add(24*time.Hour).Unix())
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

func (s *SessionManager) Verify(token string) (uint, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	payload := string(payloadBytes)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return 0, false
	}
	fields := strings.Split(payload, ":")
	if len(fields) != 2 {
		return 0, false
	}
	id, err1 := strconv.ParseUint(fields[0], 10, 64)
	exp, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil || id == 0 || time.Now().Unix() > exp {
		return 0, false
	}
	return uint(id), true
}
