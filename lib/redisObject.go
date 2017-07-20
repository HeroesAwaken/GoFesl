package lib

import (
	// Needed since we are using this for opening the connection
	"github.com/go-redis/redis"
)

// RedisObject Stores a hash-map in redis, provides basic crud-like actions
type RedisObject struct {
	redis      *redis.Client
	identifier string
}

// New - Initialize a new key
func (rS *RedisObject) New(redis *redis.Client, prefix string, identifier string) {
	rS.redis = redis
	rS.identifier = prefix + ":" + identifier
}

// Get - Get value from the hash-map
func (rS *RedisObject) Get(key string) string {
	stringCmd := rS.redis.HGet(rS.identifier, key)
	return stringCmd.Val()
}

// HKeys - Get a list of the keys in the hash-map
func (rS *RedisObject) HKeys() []string {
	stringSliceCmd := rS.redis.HKeys(rS.identifier)
	return stringSliceCmd.Val()
}

// Set - Set a value in the hash-map
func (rS *RedisObject) Set(key string, value string) error {
	statusCmd := rS.redis.HSet(rS.identifier, key, value)
	return statusCmd.Err()
}

// SetM - runs HMSET
func (rS *RedisObject) SetM(set map[string]interface{}) error {
	statusCmd := rS.redis.HMSet(rS.identifier, set)
	return statusCmd.Err()
}

// Delete - Deletes this key
func (rS *RedisObject) Delete() error {
	statusCmd := rS.redis.Del(rS.identifier)
	return statusCmd.Err()
}
