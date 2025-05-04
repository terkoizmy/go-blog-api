package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model with UUID instead of auto-incrementing integer
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate will set a UUID rather than numeric ID
func (base *Base) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}

type User struct {
	Base
	Username  string `gorm:"uniqueIndex;size:255;not null" json:"username"`
	Email     string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password  string `gorm:"size:255;not null" json:"-"`
	FirstName string `gorm:"size:255" json:"first_name"`
	LastName  string `gorm:"size:255" json:"last_name"`
	Role      string `gorm:"size:50;default:'user'" json:"role"`
	Posts     []Post `gorm:"foreignKey:AuthorID" json:"-"`
}

type Post struct {
	Base
	Title       string     `gorm:"size:255;not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Slug        string     `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	AuthorID    uuid.UUID  `gorm:"type:uuid;not null" json:"author_id"`
	Author      User       `gorm:"foreignKey:AuthorID" json:"author"`
	Status      string     `gorm:"size:50;default:'draft'" json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Categories  []Category `gorm:"many2many:post_categories;" json:"categories"`
	Comments    []Comment  `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

type Category struct {
	Base
	Name  string `gorm:"uniqueIndex;size:255;not null" json:"name"`
	Slug  string `gorm:"uniqueIndex;size:255;not null" json:"slug"`
	Posts []Post `gorm:"many2many:post_categories;" json:"-"`
}

type Comment struct {
	Base
	Content  string     `gorm:"type:text;not null" json:"content"`
	PostID   uuid.UUID  `gorm:"type:uuid;not null" json:"post_id"`
	Post     Post       `gorm:"foreignKey:PostID" json:"-"`
	AuthorID uuid.UUID  `gorm:"type:uuid;not null" json:"author_id"`
	Author   User       `gorm:"foreignKey:AuthorID" json:"author"`
	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent   *Comment   `gorm:"foreignKey:ParentID" json:"-"`
	Replies  []Comment  `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

// Request and response structures
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type TokenResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
}

type PostRequest struct {
	Title       string      `json:"title" binding:"required"`
	Content     string      `json:"content" binding:"required"`
	Slug        string      `json:"slug"`
	Status      string      `json:"status"`
	CategoryIDs []uuid.UUID `json:"category_ids"`
}

type CommentRequest struct {
	Content  string     `json:"content" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type CategoryRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug"`
}
