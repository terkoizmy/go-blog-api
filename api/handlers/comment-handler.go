package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terkoizmy/go-blog-api/internal/db"
	"github.com/terkoizmy/go-blog-api/internal/models"
)

// CommentHandler handles comment-related routes
type CommentHandler struct{}

// NewCommentHandler creates a new CommentHandlerz
func NewCommentHandler() *CommentHandler {
	return &CommentHandler{}
}

// @Summary Create a new comment
// @Description Create a new comment on a post
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param postId path string true "Post ID"
// @Param comment body models.CommentRequest true "Comment details"
// @Success 201 {object} models.Comment
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/posts/{postId} [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
	postID := c.Param("postId")
	fmt.Println(postID, "postID")
	// Parse the UUID
	postUUID, err := uuid.Parse(postID)
	fmt.Println(postUUID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	// Check if post exists and is published
	var post models.Post
	if result := db.DB.Where("id = ?", postUUID).First(&post); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	// Only allow comments on published posts
	if post.Status != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot comment on unpublished posts"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authorID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	var req models.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create comment
	comment := models.Comment{
		Content:  req.Content,
		PostID:   postUUID,
		AuthorID: authorID,
		ParentID: req.ParentID,
	}

	// If ParentID is provided, check if parent comment exists
	if req.ParentID != nil {
		var parentComment models.Comment
		if result := db.DB.Where("id = ?", *req.ParentID).First(&parentComment); result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment not found"})
			return
		}

		// Check if parent comment belongs to the same
		if parentComment.PostID != postUUID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parent comment does not belong to the same post"})
			return
		}
	}

	// Don't return the password
	// comment.Author.Password = ""

	if err := db.DB.Preload("Author").Preload("Parent").Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)

}

// @Summary Get comments by POST ID
// @Description Get a Comments post by its post ID
// @Tags comments
// @Accept json
// @Produce json
// @Param postId path string true "Post Id"
// @Success 200 {object} models.Comment
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/posts/{postId} [get]
func (h *CommentHandler) GetAllCommentsFromPostId(c *gin.Context) {
	postId := c.Param("postId")

	fmt.Println(postId, "postId")

	// Parse the UUID
	postUUID, err := uuid.Parse(postId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID format"})
		return
	}

	var comments []models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Where("post_id = ?", postUUID).Find(&comments); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// @Summary Get comment by comment ID
// @Description Get a Comment by its id
// @Tags comments
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} models.Comment
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /comment/{id} [get]
func (h *CommentHandler) GetCommentById(c *gin.Context) {
	commentID := c.Param("id")

	// Parse the UUID
	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment ID format"})
		return
	}

	var comment models.Comment
	if result := db.DB.Preload("Author").Preload("Parent").Where("id = ?", commentUUID).First(&comment); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, comment)

}
