package jwt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/cache"
	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGenerateTokenPairs(t *testing.T) {
	cases := map[string]struct {
		algo         string
		secret       string
		minSecretLen int
		req          *models.User
		wantErr      bool
		wantAccess   string
		wantRefresh  string
	}{
		"invalid algo": {
			algo:    "invalid",
			wantErr: true,
		},
		"secret not set": {
			algo:    "HS256",
			wantErr: true,
		},
		"invalid secret length": {
			algo:    "HS256",
			secret:  "123",
			wantErr: true,
		},
		"invalid secret length with min defined": {
			algo:         "HS256",
			minSecretLen: 4,
			secret:       "123",
			wantErr:      true,
		},
		"success": {
			algo:         "HS256",
			secret:       "g0r$kt3$t1ng",
			minSecretLen: 1,
			req: &models.User{
				ID:       primitive.NewObjectID(),
				Username: "johndoe",
				Email:    "johndoe@mail.com",
			},
			wantAccess:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantRefresh: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Log(tt.minSecretLen)
			jwtSvc, err := jwt.New(tt.algo, tt.secret, 60, 60, 60, tt.minSecretLen, &cache.Cache{}, false, &platform.Platform{}, &db.DB{})
			assert.Equal(t, tt.wantErr, err != nil)
			if err == nil && !tt.wantErr {
				access, refresh, _ := jwtSvc.GenerateTokenPair(tt.req)
				assert.Contains(t, access, tt.wantAccess)
				assert.Contains(t, refresh, tt.wantRefresh)
			}
		})
	}
}
