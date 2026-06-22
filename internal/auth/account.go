package auth

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

func userFromSessionRow(row *dbsql.UserBySessionTokenRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func userFromRegisteredRow(row *dbsql.UserByRegisteredIdRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func userFromIDOrRegisteredRow(row *dbsql.UserByIdorRegisteredIdRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func stringPtr(value string) *string {
	return &value
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func numericCode(size int) (string, error) {
	var b strings.Builder
	for i := 0; i < size; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String(), nil
}
