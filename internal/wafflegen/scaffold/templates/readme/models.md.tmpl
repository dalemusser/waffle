# Domain Models Directory

This directory contains **domain models and business entities** for your WAFFLE application.

## Purpose

Domain models represent the core business concepts in your application. They are:

- **Data structures** that represent real-world entities
- **Independent** of HTTP, database, or framework concerns
- **Reusable** across features, stores, and policies

## What Belongs Here

- **Entity structs** — User, Product, Order, etc.
- **Value objects** — Address, Money, DateRange, etc.
- **Enums and constants** — Status values, type definitions
- **Business methods** — Logic that belongs to the entity itself

## Example Structure

```
domain/
└── models/
    ├── user.go           # User entity
    ├── product.go        # Product entity
    ├── order.go          # Order aggregate
    ├── address.go        # Address value object
    └── common.go         # Shared types (timestamps, IDs, etc.)
```

## Usage Example

Define a domain entity:

```go
// domain/models/user.go
package models

import "time"

type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusInactive UserStatus = "inactive"
    UserStatusBanned   UserStatus = "banned"
)

type User struct {
    ID          string     `json:"id" bson:"_id"`
    Email       string     `json:"email" bson:"email"`
    Name        string     `json:"name" bson:"name"`
    Status      UserStatus `json:"status" bson:"status"`
    Permissions []string   `json:"permissions" bson:"permissions"`
    CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" bson:"updated_at"`
}

// IsActive returns true if the user can log in.
func (u *User) IsActive() bool {
    return u.Status == UserStatusActive
}

// HasPermission checks if the user has a specific permission.
func (u *User) HasPermission(perm string) bool {
    for _, p := range u.Permissions {
        if p == perm {
            return true
        }
    }
    return false
}
```

Define a value object:

```go
// domain/models/address.go
package models

type Address struct {
    Street     string `json:"street" bson:"street"`
    City       string `json:"city" bson:"city"`
    State      string `json:"state" bson:"state"`
    PostalCode string `json:"postal_code" bson:"postal_code"`
    Country    string `json:"country" bson:"country"`
}

// IsComplete returns true if all required fields are present.
func (a Address) IsComplete() bool {
    return a.Street != "" && a.City != "" && a.Country != ""
}
```

## Using Models

Models are used throughout the application:

```go
// In stores (data access)
func (s *mongoStore) FindByID(ctx context.Context, id string) (*models.User, error)

// In handlers (HTTP layer)
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var user models.User
    json.NewDecoder(r.Body).Decode(&user)
    // ...
}

// In policies (authorization)
func CanEdit(actor *models.User, target *models.User) bool {
    return actor.HasPermission("users:edit")
}
```

## Guidelines

- Keep models free of database or HTTP-specific logic
- Use struct tags for JSON and database field mapping
- Add business methods that operate on the entity's data
- Prefer value objects for concepts that don't have identity (Address, Money)
- Consider validation methods that return errors for invalid states
