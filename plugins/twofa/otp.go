package twofa

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "math/rand"
    "time"
    "github.com/rs/xid"
)

// SendOTP generates and stores a one-time password; returns the code for delivery
func (s *Service) SendOTP(ctx context.Context, userID string) (string, error) {
    // 6-digit numeric code
    rand.Seed(time.Now().UnixNano())
    code := 100000 + rand.Intn(900000)
    ch := sha256.Sum256([]byte(fmtCode(code)))
    hash := hex.EncodeToString(ch[:])
    expires := time.Now().Add(5 * time.Minute)
    uid, err := xid.FromString(userID)
    if err != nil { return "", err }
    if err := s.repo.CreateOTPCode(ctx, uid, hash, expires); err != nil {
        return "", err
    }
    return fmtCode(code), nil
}

// VerifyOTP verifies a one-time password against stored hash
func (s *Service) VerifyOTP(ctx context.Context, userID, code string) (bool, error) {
    uid, err := xid.FromString(userID)
    if err != nil { return false, err }
    ch := sha256.Sum256([]byte(code))
    hash := hex.EncodeToString(ch[:])
    return s.repo.VerifyOTPCode(ctx, uid, hash, time.Now(), 5)
}

func fmtCode(n int) string { return fmt.Sprintf("%06d", n) }