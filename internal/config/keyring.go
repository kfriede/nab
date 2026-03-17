package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const keyringService = "nab"

// StoreSecret stores a secret (YNAB personal access token) in the OS keyring.
// Falls back to config file if the keyring is not available.
func StoreSecret(profile, secret string) error {
	user := keyringUser(profile)
	err := keyring.Set(keyringService, user, secret)
	if err != nil {
		return fmt.Errorf("storing secret in keyring: %w (you may need to store it in the config file instead)", err)
	}
	return nil
}

// GetSecret retrieves a secret from the OS keyring.
// Returns empty string and nil error if no secret is stored.
func GetSecret(profile string) (string, error) {
	user := keyringUser(profile)
	secret, err := keyring.Get(keyringService, user)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("reading secret from keyring: %w", err)
	}
	return secret, nil
}

// DeleteSecret removes a secret from the OS keyring.
func DeleteSecret(profile string) error {
	user := keyringUser(profile)
	err := keyring.Delete(keyringService, user)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("deleting secret from keyring: %w", err)
	}
	return nil
}

// KeyringAvailable returns true if the OS keyring is usable.
func KeyringAvailable() bool {
	// Test by trying to get a non-existent key — if the keyring
	// backend itself errors (not just "not found"), it's unavailable.
	_, err := keyring.Get(keyringService, "__nab_keyring_probe__")
	return err == nil || err == keyring.ErrNotFound
}

func keyringUser(profile string) string {
	if profile == "" {
		return "default"
	}
	return profile
}
