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
	"strings"
)

func CreatePost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
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

	tagNames := strings.Split(strings.TrimSpace(data["tags"]), ",")
	tagSet := make(map[string]bool) // Для фильтрации дублей
	var tagList []string

	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName != "" && !tagSet[tagName] {
			tagSet[tagName] = true
			tagList = append(tagList, tagName)
		}
	}

	if len(tagList) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "No valid tags provided"})
	}

	post := models.Post{
		ChannelId: uint(channelId),
		Title:     data["title"],
		Content:   data["content"],
	}

	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create post")
		}

		var tags []models.Tag
		for _, tagName := range tagList {
			var tag models.Tag
			if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "A non-existent tag was provided")
			}
			tags = append(tags, tag)
		}

		// Привязываем теги к посту
		if err := tx.Model(&post).Association("Tags").Append(tags); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create tags")
		}

		return nil
	})

	if err != nil {
		return c.JSON(fiber.Map{
			"message": err,
		})
	}

	if err := repository.DB.Preload("Tags").First(&post, post.Id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	return c.JSON(post)
}

func EditPost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
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

	tagNames := strings.Split(strings.TrimSpace(data["tags"]), ",")
	tagSet := make(map[string]bool) // Для фильтрации дублей
	var tagList []string

	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName != "" && !tagSet[tagName] {
			tagSet[tagName] = true
			tagList = append(tagList, tagName)
		}
	}

	// Начинаем транзакцию
	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		// Сохраняем изменения в посте
		if err := tx.Save(&post).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update post")

		}

		if len(tagList) != 0 {
			var tags []models.Tag
			for _, tagName := range tagList {
				var tag models.Tag
				if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "A non-existent tag was provided")
				}
				tags = append(tags, tag)
			}

			// Очищаем старые теги и добавляем новые
			if err := tx.Model(&post).Association("Tags").Replace(tags); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update tags")
			}
		}

		return nil
	})

	if err != nil {
		return c.JSON(fiber.Map{
			"message": err,
		})
	}

	if err := repository.DB.Preload("Tags").First(&post, uint(postId)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	return c.JSON(post)
}

func DeletePost(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
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

	if postId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Post Id is required",
		})
	}

	var post models.Post

	if err := repository.DB.Preload("Tags").First(&post, postId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to fetch post",
		})
	}

	if err := repository.DB.Model(&post).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update view count",
		})
	}

	return c.JSON(post)
}

func GetPosts(c fiber.Ctx) error {
	channelId := c.Query("channelId")
	if channelId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Channel Id is required",
		})
	}

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid page value",
		})
	}

	offset := (page - 1) * 10

	var posts []models.Post
	if err := repository.DB.Preload("Tags").Where("channel_id = ?", channelId).Limit(10).Offset(offset).Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to fetch posts",
		})
	}

	return c.JSON(posts)
}

func SetReaction(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
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

	userId, err := strconv.ParseUint(strId, 10, 32)
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

	if data["reaction"] != "like" && data["reaction"] != "dislike" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid reaction type",
		})
	}

	reactionValue := data["reaction"] == "like"

	var existingReaction models.PostReaction
	if err := repository.DB.Where("post_id = ? AND user_id = ?", postId, userId).First(&existingReaction).Error; err == nil {
		if existingReaction.Reaction == reactionValue {
			if err := repository.DB.Delete(&existingReaction).Error; err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Failed to remove reaction",
				})
			}
			return c.JSON(fiber.Map{
				"message": "Reaction removed",
			})
		}

		existingReaction.Reaction = reactionValue
		if err := repository.DB.Save(&existingReaction).Error; err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Failed to update reaction",
			})
		}
		return c.JSON(fiber.Map{
			"message": "Reaction updated",
		})
	}

	newReaction := models.PostReaction{
		PostId:   uint(postId),
		UserId:   uint(userId),
		Reaction: reactionValue,
	}

	if err := repository.DB.Create(&newReaction).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to add reaction",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Reaction added",
	})
}
