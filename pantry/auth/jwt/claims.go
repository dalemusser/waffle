// auth/jwt/claims.go
package jwt

// Terminology: User Identifiers
//   - UserID / userID / user_id: The MongoDB ObjectID (_id) that uniquely identifies a user record
//   - LoginID / loginID / login_id: The human-readable string users type to log in

import (
	"time"
)

// CustomClaims embeds standard claims and allows custom fields.
type CustomClaims[T any] struct {
	Claims
	Custom T `json:"custom,omitempty"`
}

// Valid validates both standard and custom claims.
func (c *CustomClaims[T]) Valid() error {
	return c.Claims.Valid()
}

// UserClaims is a common pattern for user authentication tokens.
type UserClaims struct {
	Claims
	UserID   string   `json:"uid,omitempty"`
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

// Valid validates the claims.
func (c *UserClaims) Valid() error {
	return c.Claims.Valid()
}

// HasRole checks if the user has a specific role.
func (c *UserClaims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles.
func (c *UserClaims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the user has all of the specified roles.
func (c *UserClaims) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !c.HasRole(role) {
			return false
		}
	}
	return true
}

// Builder provides a fluent API for creating claims.
type Builder struct {
	claims Claims
}

// NewBuilder creates a new claims builder.
func NewBuilder() *Builder {
	return &Builder{
		claims: Claims{
			IssuedAt: NewTime(time.Now()),
		},
	}
}

// Issuer sets the issuer claim.
func (b *Builder) Issuer(iss string) *Builder {
	b.claims.Issuer = iss
	return b
}

// Subject sets the subject claim.
func (b *Builder) Subject(sub string) *Builder {
	b.claims.Subject = sub
	return b
}

// Audience sets the audience claim.
func (b *Builder) Audience(aud ...string) *Builder {
	b.claims.Audience = aud
	return b
}

// ExpiresIn sets the expiration relative to now.
func (b *Builder) ExpiresIn(d time.Duration) *Builder {
	b.claims.ExpiresAt = NewTime(time.Now().Add(d))
	return b
}

// ExpiresAt sets the expiration time.
func (b *Builder) ExpiresAt(t time.Time) *Builder {
	b.claims.ExpiresAt = NewTime(t)
	return b
}

// NotBefore sets the not-before time.
func (b *Builder) NotBefore(t time.Time) *Builder {
	b.claims.NotBefore = NewTime(t)
	return b
}

// NotBeforeIn sets the not-before time relative to now.
func (b *Builder) NotBeforeIn(d time.Duration) *Builder {
	b.claims.NotBefore = NewTime(time.Now().Add(d))
	return b
}

// ID sets the JWT ID.
func (b *Builder) ID(jti string) *Builder {
	b.claims.ID = jti
	return b
}

// Build returns the claims.
func (b *Builder) Build() Claims {
	return b.claims
}

// UserBuilder provides a fluent API for creating user claims.
type UserBuilder struct {
	claims UserClaims
}

// NewUserBuilder creates a new user claims builder.
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		claims: UserClaims{
			Claims: Claims{
				IssuedAt: NewTime(time.Now()),
			},
		},
	}
}

// Issuer sets the issuer claim.
func (b *UserBuilder) Issuer(iss string) *UserBuilder {
	b.claims.Issuer = iss
	return b
}

// Subject sets the subject claim.
func (b *UserBuilder) Subject(sub string) *UserBuilder {
	b.claims.Subject = sub
	return b
}

// Audience sets the audience claim.
func (b *UserBuilder) Audience(aud ...string) *UserBuilder {
	b.claims.Audience = aud
	return b
}

// ExpiresIn sets the expiration relative to now.
func (b *UserBuilder) ExpiresIn(d time.Duration) *UserBuilder {
	b.claims.ExpiresAt = NewTime(time.Now().Add(d))
	return b
}

// ExpiresAt sets the expiration time.
func (b *UserBuilder) ExpiresAt(t time.Time) *UserBuilder {
	b.claims.ExpiresAt = NewTime(t)
	return b
}

// ID sets the JWT ID.
func (b *UserBuilder) ID(jti string) *UserBuilder {
	b.claims.ID = jti
	return b
}

// UserID sets the user ID.
func (b *UserBuilder) UserID(uid string) *UserBuilder {
	b.claims.UserID = uid
	return b
}

// Username sets the username.
func (b *UserBuilder) Username(username string) *UserBuilder {
	b.claims.Username = username
	return b
}

// Email sets the email.
func (b *UserBuilder) Email(email string) *UserBuilder {
	b.claims.Email = email
	return b
}

// Roles sets the user roles.
func (b *UserBuilder) Roles(roles ...string) *UserBuilder {
	b.claims.Roles = roles
	return b
}

// Build returns the user claims.
func (b *UserBuilder) Build() UserClaims {
	return b.claims
}
