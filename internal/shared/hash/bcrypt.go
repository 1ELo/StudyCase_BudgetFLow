package hash

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// Hash hashes a plain text password using bcrypt with cost 12.
func Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare checks if a plain text password matches a bcrypt hash.
func Compare(plain, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}
