package sessions

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-seatbelt/seatbelt/internal/rand"
	"github.com/go-seatbelt/seatbelt/internal/trace"
	"github.com/sirupsen/logrus"
)

// session is the default connection to Redis.
var session *sessionService

// A sessionService is used for managing data stored in Redis.
type sessionService struct {
	rdb        *redis.Client
	prefix     string
	authPrefix string
}

// init connects to the Redis database.
func init() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set.
		DB:       0,  // Use default db.
	})

	if err := rdb.Ping().Err(); err != nil {
		logrus.Fatalf("%s: Failed to connect to Redis: %+v", trace.Getfl(), err)
	}
	logrus.Infoln("Connected to Redis")

	session = &sessionService{rdb: rdb, prefix: "sessions:", authPrefix: "auth:"}
}

// Save saves an arbitrary session value in Redis. It will generate and return
// the key associated with the value.
func Save(v interface{}) (string, error) {
	key := rand.NewString(12)

	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(v); err != nil {
		return "", err
	}

	err := session.rdb.Set(session.prefix+string(key), buf.Bytes(), time.Duration(1*time.Hour)).Err()
	return key, err
}

// Get returns the value for the given key, and decodes it into v.
func Get(key string, v interface{}) error {
	b, err := session.rdb.Get(session.prefix + key).Bytes()
	if err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewReader(b)).Decode(v)
}

// Put updates the session for the given key.
func Put(key string, v interface{}) error {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(v); err != nil {
		return err
	}
	return session.rdb.Set(session.prefix+key, buf.Bytes(), time.Duration(1*time.Hour)).Err()
}

// Delete deletes the given key and associated value from the Redis session
// store.
func Delete(key string) error {
	return session.rdb.Del(session.prefix + key).Err()
}

// Authorize creates a new session for the given data. The session will expire
// in one month.
func Authorize(model interface{}) (string, error) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(model); err != nil {
		return "", err
	}

	key := rand.NewString(64)
	err := session.rdb.Set(session.authPrefix+key, buf.Bytes(), time.Duration(24*30*time.Hour)).Err()
	return key, err
}

// Find decodes the value of identifier associated with the given token into
// the given model.
func Find(token string, model interface{}) error {
	b, err := session.rdb.Get(session.authPrefix + token).Bytes()
	if err != nil {
		return err
	}

	return gob.NewDecoder(bytes.NewReader(b)).Decode(model)
}

// Unauthorize deletes the authorization session for the given token.
func Unauthorize(token string) error {
	return session.rdb.Del(session.authPrefix + token).Err()
}
