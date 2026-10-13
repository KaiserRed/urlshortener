package auth

import "github.com/golang-jwt/jwt/v5"

// Claims is the JWT payload issued by the SSO service.
type Claims struct {
	UID   int64  `json:"uid"`
	Email string `json:"email"`
	AppID int    `json:"app_id"`
	jwt.RegisteredClaims
}
