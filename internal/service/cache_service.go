package service

import (
	"context"
	"fmt"
	"time"

	"skintrader-go/internal/config"
	"skintrader-go/internal/repository/cache"
)

type CacheService struct {
	cache  *cache.RedisCache
	config config.CacheConfig
}

func NewCacheService(cacheRepo *cache.RedisCache, cfg config.CacheConfig) *CacheService {
	return &CacheService{cache: cacheRepo, config: cfg}
}

func (s *CacheService) GetPost(ctx context.Context, postID string, dest any) error {
	key := fmt.Sprintf(cache.KeyPostByID, postID)
	return s.cache.Get(ctx, key, dest)
}

func (s *CacheService) SetPost(ctx context.Context, postID string, data any) error {
	key := fmt.Sprintf(cache.KeyPostByID, postID)
	return s.cache.Set(ctx, key, data, s.config.PostsTTL)
}

func (s *CacheService) InvalidatePost(ctx context.Context, postID string) error {
	key := fmt.Sprintf(cache.KeyPostByID, postID)
	return s.cache.Delete(ctx, key)
}

func (s *CacheService) GetGamesList(ctx context.Context, dest any) error {
	return s.cache.Get(ctx, cache.KeyGamesList, dest)
}

func (s *CacheService) SetGamesList(ctx context.Context, data any) error {
	return s.cache.Set(ctx, cache.KeyGamesList, data, s.config.GamesTTL)
}

func (s *CacheService) GetGame(ctx context.Context, gameID string, dest any) error {
	key := fmt.Sprintf(cache.KeyGameByID, gameID)
	return s.cache.Get(ctx, key, dest)
}

func (s *CacheService) SetGame(ctx context.Context, gameID string, data any) error {
	key := fmt.Sprintf(cache.KeyGameByID, gameID)
	return s.cache.Set(ctx, key, data, s.config.GamesTTL)
}

func (s *CacheService) InvalidateGame(ctx context.Context, gameID string) error {
	key := fmt.Sprintf(cache.KeyGameByID, gameID)
	return s.cache.Delete(ctx, key, cache.KeyGamesList)
}

func (s *CacheService) GetUserProfile(ctx context.Context, userID string, dest any) error {
	key := fmt.Sprintf(cache.KeyUserProfile, userID)
	return s.cache.Get(ctx, key, dest)
}

func (s *CacheService) SetUserProfile(ctx context.Context, userID string, data any) error {
	key := fmt.Sprintf(cache.KeyUserProfile, userID)
	return s.cache.Set(ctx, key, data, s.config.UserTTL)
}

func (s *CacheService) InvalidateUserProfile(ctx context.Context, userID string) error {
	key := fmt.Sprintf(cache.KeyUserProfile, userID)
	return s.cache.Delete(ctx, key)
}

func (s *CacheService) IncrementViewCount(ctx context.Context, postID string) (int64, error) {
	key := fmt.Sprintf("views:%s", postID)
	count, err := s.cache.Increment(ctx, key)
	if err != nil {
		return 0, err
	}
	if count == 1 {
		_ = s.cache.Expire(ctx, key, 1*time.Hour)
	}
	return count, nil
}
