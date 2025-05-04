package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terkoizmy/go-blog-api/config"
	"github.com/terkoizmy/go-blog-api/internal/auth"
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

// UserHandler handles user-related routes
type UserHandler struct{}

// NewUserHandler creates a new UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// @Summary Register a new user
// @Description Register a new user with the provided details
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.RegisterRequest true "User registration details"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	var existingUser models.User
	if result := db.DB.Where("username = ?", req.Username).First(&existingUser); result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	// Check if email already exists
	if result := db.DB.Where("email = ?", req.Email).First(&existingUser); result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Create user
	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      "user", // Default role
	}

	if result := db.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Don't return the password
	user.Password = ""

	c.JSON(http.StatusCreated, user)
}

// @Summary Login user
// @Description Login with username and password
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.TokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by username
	var user models.User
	if result := db.DB.Where("username = ?", req.Username).First(&user); result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Check password
	if !auth.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate JWT token
	cfg, err := config.LoadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't load config"})
		return
	}

	token, err := auth.GenerateToken(user, cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.TokenResponse{
		Token:     token,
		TokenType: "Bearer",
	})
}

// @Summary Get user profile
// @Description Get the profile of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	var user models.User
	if result := db.DB.Where("id = ?", id).First(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Don't return the password
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// @Summary Get all users
// @Description Get all users (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (h *UserHandler) GetAll(c *gin.Context) {
	var users []models.User
	if result := db.DB.Find(&users); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get users"})
		return
	}

	// Remove passwords
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}

// @Summary Update user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param user body models.User true "Updated user information"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse the UUID
	userUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	// Check if user is updating their own profile or is an admin
	requestingUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	// Only allow users to update their own profile or admins to update any profile
	if userUUID != requestingUserID && roleStr != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var user models.User
	if result := db.DB.Where("id = ?", userUUID).First(&user); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get update data
	var updateData struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Password  string `json:"password,omitempty"`
		Role      string `json:"role,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only update fields that were provided
	if updateData.FirstName != "" {
		user.FirstName = updateData.FirstName
	}

	if updateData.LastName != "" {
		user.LastName = updateData.LastName
	}

	if updateData.Email != "" && updateData.Email != user.Email {
		// Check if the email is already taken
		var existingUser models.User
		if result := db.DB.Where("email = ? AND id != ?", updateData.Email, userUUID).First(&existingUser); result.RowsAffected > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
			return
		}
		user.Email = updateData.Email
	}

	// Only admins can update roles
	if updateData.Role != "" && roleStr == "admin" {
		user.Role = updateData.Role
	}

	// Update password if provided
	if updateData.Password != "" {
		hashedPassword, err := auth.HashPassword(updateData.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		user.Password = hashedPassword
	}

	if result := db.DB.Save(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	// Don't return the password
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// @Summary Delete user
// @Description Delete a user (admin only or own account)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse the UUID
	userUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	// Check if user is deleting their own account or is an admin
	requestingUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	// Only allow users to delete their own account or admins to delete any account
	if userUUID != requestingUserID && roleStr != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var user models.User
	if result := db.DB.Where("id = ?", userUUID).First(&user); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Delete user
	if result := db.DB.Delete(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}
