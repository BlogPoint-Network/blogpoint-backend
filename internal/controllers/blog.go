package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
)

func CreateBlog(c fiber.Ctx) error {
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

	var existingBlog models.Blog
	if result := repository.DB.Where("channel_id = ? AND title = ?", uint(channelId), data["title"]).First(&existingBlog); result.Error == nil {
		c.Status(fiber.StatusConflict)
		return c.JSON(fiber.Map{
			"message": "A blog with the same title already exists in this channel",
		})
	}

	blog := models.Blog{
		ChannelId: uint(channelId),
		Title:     data["title"],
		Content:   data["content"],
	}

	if err := repository.DB.Create(&blog).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to create blog",
		})
	}

	return c.JSON(blog)
}

func EditBlog(c fiber.Ctx) error {
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

	if data["blogId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Blog id is required",
		})
	}

	blogId, err := strconv.ParseUint(data["blogId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid blog id",
		})
	}

	var blog models.Blog
	if result := repository.DB.First(&blog, uint(blogId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Blog not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, blog.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(fiber.Map{
			"message": "You are not the owner of this blog",
		})
	}

	if data["title"] != "" {
		blog.Title = data["title"]
	}
	if data["content"] != "" {
		blog.Content = data["content"]
	}

	if err := repository.DB.Save(&blog).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to update blog",
		})
	}

	return c.JSON(blog)
}

func DeleteBlog(c fiber.Ctx) error {
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

	if data["blogId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Blog id is required",
		})
	}

	blogId, err := strconv.ParseUint(data["blogId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid blog id",
		})
	}

	var blog models.Blog
	if result := repository.DB.First(&blog, uint(blogId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Blog not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, blog.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(fiber.Map{
			"message": "You are not the owner of this blog",
		})
	}

	if err := repository.DB.Delete(&blog).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to delete blog",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Blog successfully deleted",
	})
}
