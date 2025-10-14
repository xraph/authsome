package twofa

// BackupCodes manages generation and verification of recovery codes (stub)
func (s *Service) BackupCodes(userID string, count int) ([]string, error) {
    _ = userID; _ = count
    return []string{"backup-1", "backup-2"}, nil
}

func (s *Service) VerifyBackupCode(userID, code string) (bool, error) {
    _ = userID; _ = code
    return true, nil
}