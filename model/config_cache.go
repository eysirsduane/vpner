package model

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const configCacheRefreshInterval = 30 * time.Second

type configCacheStore struct {
	mu     sync.RWMutex
	values map[string]string
	loaded bool
}

func (s *configCacheStore) get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.loaded {
		return "", false
	}
	value, ok := s.values[key]
	return value, ok
}

func (s *configCacheStore) replace(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = values
	s.loaded = true
}

var (
	systemConfigCache       = &configCacheStore{}
	payConfigCache          = &configCacheStore{}
	inviteConfigCache       = &configCacheStore{}
	configCacheRefresherOne sync.Once
)

func InitConfigCaches() error {
	if err := RefreshConfigCaches(); err != nil {
		return err
	}
	configCacheRefresherOne.Do(func() {
		go refreshConfigCachesLoop()
	})
	return nil
}

func RefreshConfigCaches() error {
	if err := refreshSystemConfigCache(); err != nil {
		return err
	}
	if err := refreshPayConfigCache(); err != nil {
		return err
	}
	if err := refreshInviteConfigCache(); err != nil {
		return err
	}
	return nil
}

func refreshConfigCachesLoop() {
	ticker := time.NewTicker(configCacheRefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := RefreshConfigCaches(); err != nil {
			log.Printf("refresh config cache failed: %v", err)
		}
	}
}

func refreshSystemConfigCache() error {
	var configs []Config
	if err := DB.Find(&configs).Error; err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	values := make(map[string]string, len(configs))
	for _, config := range configs {
		values[config.Code] = config.Value
	}
	systemConfigCache.replace(values)
	return nil
}

func refreshPayConfigCache() error {
	var configs []PayConfig
	if err := DB.Find(&configs).Error; err != nil {
		return fmt.Errorf("load pay_config: %w", err)
	}
	values := make(map[string]string, len(configs))
	for _, config := range configs {
		values[config.Code] = config.Value
	}
	payConfigCache.replace(values)
	return nil
}

func refreshInviteConfigCache() error {
	var configs []InviteConfig
	if err := DB.Find(&configs).Error; err != nil {
		return fmt.Errorf("load invite_config: %w", err)
	}
	values := make(map[string]string, len(configs))
	for _, config := range configs {
		values[config.ConfigKey] = config.ConfigValue
	}
	inviteConfigCache.replace(values)
	return nil
}
