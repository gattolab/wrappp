package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gattolab/wrappp/pkg/cache"
)

func PrintToJSON(v interface{}) {
	fmt.Println("PrintToJSON")

	s, _ := json.MarshalIndent(v, "", "\t")
	fmt.Println(string(s))
}

func IsEqueThen(input, target, success, failure string) string {
	if input == target {
		return success
	}

	return failure
}

func UseCache[T any](ctx context.Context, cache cache.Engine, key string, executeFunc func(context.Context) (T, error), cacheTime time.Duration) (T, error) {
	var result T
	cachedData, err := cache.Get(key)
	if err == nil {
		if err := json.Unmarshal(cachedData, &result); err == nil {
			return result, nil
		}
	}

	result, err = executeFunc(ctx)
	if err != nil {
		return result, err
	}

	dataToCache, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	_ = cache.Set(key, dataToCache, cacheTime)

	return result, nil
}
