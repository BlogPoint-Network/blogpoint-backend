package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"strconv"
)

func CreatePost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer id",
		})
	}

	if data["channelId"] == "" || data["title"] == "" || data["content"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Missing required fields",
		})
	}

	channelId, err := strconv.ParseUint(data["channelId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid channel id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, uint(channelId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(fiber.Map{
			"message": "You are not the owner of this channel",
		})
	}

	var existingPost models.Post
	if result := repository.DB.Where("channel_id = ? AND title = ?", uint(channelId), data["title"]).First(&existingPost); result.Error == nil {
		c.Status(fiber.StatusConflict)
		return c.JSON(fiber.Map{
			"message": "A post with the same title already exists in this channel",
		})
	}

	post := models.Post{
		ChannelId: uint(channelId),
		Title:     data["title"],
		Content:   data["content"],
	}

	if err := repository.DB.Create(&post).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to create post",
		})
	}

	return c.JSON(post)
}

func EditPost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer id",
		})
	}

	if data["postId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Post id is required",
		})
	}

	postId, err := strconv.ParseUint(data["postId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid post id",
		})
	}

	var post models.Post
	if result := repository.DB.First(&post, uint(postId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Post not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, post.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(fiber.Map{
			"message": "You are not the owner of this post",
		})
	}

	if data["title"] != "" {
		post.Title = data["title"]
	}
	if data["content"] != "" {
		post.Content = data["content"]
	}

	if err := repository.DB.Save(&post).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to update post",
		})
	}

	return c.JSON(post)
}

func DeletePost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(data["token"], jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer id",
		})
	}

	if data["postId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Post id is required",
		})
	}

	postId, err := strconv.ParseUint(data["postId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid post id",
		})
	}

	var post models.Post
	if result := repository.DB.First(&post, uint(postId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Post not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, post.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(fiber.Map{
			"message": "You are not the owner of this post",
		})
	}

	if err := repository.DB.Delete(&post).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to delete post",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Post successfully deleted",
	})
}

func GetPost(c fiber.Ctx) error {
	postId := c.Query("postId")
	channelId := c.Query("channelId")

	if postId == "" || channelId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Post Id and Channel Id are required",
		})
	}

	var post models.Post

	if err := repository.DB.Where("id = ? AND channel_id = ?", postId, channelId).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch post",
		})
	}

	return c.JSON(post)
}

func GetPosts(c fiber.Ctx) error {
	channelID := c.Query("channelId")
	if channelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Channel ID is required",
		})
	}

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid page value",
		})
	}

	offset := (page - 1) * 10

	var posts []models.Post
	if err := repository.DB.Where("channel_id = ?", channelID).Limit(10).Offset(offset).Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch posts",
		})
	}

	return c.JSON(posts)
}
