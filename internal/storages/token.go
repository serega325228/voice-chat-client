package storage

import "github.com/zalando/go-keyring"

type TokenStorage struct {
	service string
	access  string
}

const (
	access  = "access"
	refresh = "refresh"
)

func NewTokenStorage(service string) *TokenStorage {
	return &TokenStorage{service: service}
}

func (s *TokenStorage) Save(accessToken, refreshToken string) error {
	if err := keyring.Set(s.service, refresh, refreshToken); err != nil {
		return err
	}

	s.access = accessToken

	return nil
}

func (s *TokenStorage) GetAccess() string {
	return s.access
}

func (s *TokenStorage) GetRefresh() (string, error) {
	return keyring.Get(s.service, refresh)
}

func (s *TokenStorage) Clear() error {
	s.access = ""
	if err := keyring.Delete(s.service, refresh); err != nil {
		return err
	}
	return nil
}
