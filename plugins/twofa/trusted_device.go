package twofa

// Trusted device support (stub)
func (s *Service) MarkTrustedDevice(userID, deviceID string, days int) error {
    _ = userID; _ = deviceID; _ = days
    return nil
}

func (s *Service) IsTrustedDevice(userID, deviceID string) bool {
    _ = userID; _ = deviceID
    return false
}