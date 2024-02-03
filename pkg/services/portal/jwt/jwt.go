package jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"

	jwt "github.com/golang-jwt/jwt"
	uuid "github.com/satori/go.uuid"

	cache "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
)

var minSecretLen = 128

// New generates new JWT service necessary for auth middleware
func New(
	algo, secret string,
	ttlAccessMinutes, ttlRefreshMinutes, ttlAutoLogoffMinutes, minSecretLength int,
	cache *cache.Cache, cacheEnabled bool,
	platform *platform.Platform,
	db *db.DB) (*Service, error) {

	if minSecretLength > 0 {
		minSecretLen = minSecretLength
	}

	log.Printf("secret length: %d minSecretLen: %d\n", len(secret), minSecretLen)
	if len(secret) < minSecretLen {
		return nil, fmt.Errorf("jwt secret length is %d, which is less than required %d", len(secret), minSecretLen)
	}
	signingMethod := jwt.GetSigningMethod(algo)
	if signingMethod == nil {
		return nil, fmt.Errorf("invalid jwt signing method: %s", algo)
	}

	return &Service{
		key:           []byte(secret),
		algo:          signingMethod,
		ttlAccess:     time.Duration(ttlAccessMinutes) * time.Minute,
		ttlRefresh:    time.Duration(ttlRefreshMinutes) * time.Minute,
		ttlAutoLogoff: time.Duration(ttlAutoLogoffMinutes) * time.Minute,
		cache:         cache,
		cacheEnabled:  cacheEnabled,
		db:            db,
		platform:      platform,
	}, nil
}

// Service provides a Json-Web-Token authentication implementation
type Service struct {
	// Secret key used for signing.
	key []byte

	// Duration for which the jwt access token is valid.
	ttlAccess time.Duration

	// Duration for which the jwt refresh token is valid.
	ttlRefresh time.Duration

	// Duration for which the jwt token lives in te auth cache i.e. autologoff time.
	ttlAutoLogoff time.Duration

	// JWT signing algorithm
	algo jwt.SigningMethod

	// Cache to store token
	cache *cache.Cache

	// Enabled cache
	cacheEnabled bool

	// Platform repositories
	platform *platform.Platform

	// DB connection
	db *db.DB
}

type CustomClaims struct {
	ID  string `json:"id"`
	UID string `json:"uid"`
	jwt.StandardClaims
}

type CachedTokens struct {
	AccessUID  string `json:"access"`
	RefreshUID string `json:"refresh"`
}

func (s *Service) ParseToken(tokenString string) (claims *CustomClaims, err error) {

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if s.algo != token.Method {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
			}
			return s.key, nil
		})
	if err != nil {
		return
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

func (s *Service) ValidateToken(claims *CustomClaims, isRefresh bool) (models.User, error) {
	var (
		err  error
		g    errgroup.Group
		user models.User
	)

	g.Go(func() error {
		cacheJSON, _ := s.cache.Get(fmt.Sprintf("token-%s", claims.ID))
		cachedTokens := new(CachedTokens)
		err = json.Unmarshal([]byte(cacheJSON), cachedTokens)

		var tokenUID string
		if isRefresh {
			tokenUID = cachedTokens.RefreshUID
		} else {
			tokenUID = cachedTokens.AccessUID
		}

		if err != nil || tokenUID != claims.UID {
			return errors.New("token not found")
		}

		return nil
	})

	g.Go(func() error {
		user, err = s.platform.UserDB.View(s.db, claims.ID)
		if err != nil {
			return platform.ErrUserDoesNotExist
		}
		return nil
	})

	err = g.Wait()

	return user, err
}

// GenerateTokenPair generates new JWT tokens and populates it with user data
func (s *Service) GenerateTokenPair(user *models.User) (accessToken, refreshToken string, err error) {
	var (
		accessUID, refreshUID string
		cacheJSON             []byte
	)

	if accessToken, accessUID, err = s.createToken(user.ID.Hex(), s.ttlAccess); err != nil {
		return
	}

	if refreshToken, refreshUID, err = s.createToken(user.ID.Hex(), s.ttlRefresh); err != nil {
		return
	}

	cacheJSON, err = json.Marshal(CachedTokens{
		AccessUID:  accessUID,
		RefreshUID: refreshUID,
	})
	if err != nil {
		return
	}

	if s.cacheEnabled {
		err = s.cache.Set(fmt.Sprintf("token-%s", user.ID.Hex()), string(cacheJSON), s.ttlAutoLogoff)
		if err != nil {
			return
		}
	}

	return
}

func (s *Service) createToken(userID string, expire time.Duration) (token string, uid string, err error) {
	exp := time.Now().Add(expire).Unix()
	uid = uuid.NewV4().String()
	claims := &CustomClaims{
		ID:  userID,
		UID: uid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}
	jwtToken := jwt.NewWithClaims(s.algo, claims)
	token, err = jwtToken.SignedString(s.key)

	return
}

func (s *Service) ExpireAuth(key string) (bool, error) {
	return s.cache.Expire(key, s.ttlAutoLogoff)
}
