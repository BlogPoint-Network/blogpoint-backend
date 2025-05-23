package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"blogpoint-backend/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

// CreatePost создает новый пост
// @Summary      Создание поста
// @Description  Создает пост в указанном канале. Пользователь должен быть владельцем канала.
// @Tags         Post
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      CreatePostRequest true "Данные поста (channelId, title, content, tags)"
// @Success      200   {object}  DataResponse[PostResponse]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /api/createPost [post]
func CreatePost(c *fiber.Ctx) error {
	var data CreatePostRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	if data.ChannelId == 0 || data.Title == "" || data.Content == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Missing required fields",
		})
	}

	var channel models.Channel
	if result := repository.DB.Preload("Category").First(&channel, data.ChannelId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	if channel.OwnerId != uint(userId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(ErrorResponse{
			Message: "You are not the owner of this channel",
		})
	}

	var existingPost models.Post
	if result := repository.DB.Where("channel_id = ? AND title = ?", data.ChannelId, data.Title).First(&existingPost); result.Error == nil {
		c.Status(fiber.StatusConflict)
		return c.JSON(ErrorResponse{
			Message: "A post with the same title already exists in this channel",
		})
	}

	if len(data.Tags) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "No tag Ids provided"})
	}

	var previewFile *models.File
	if data.PreviewImageId != nil {
		if err := repository.DB.First(&previewFile, *data.PreviewImageId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error preview image does not exist",
			})
		}

		if previewFile.OwnerId != uint(userId) {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{Message: "You don't own the preview image"})
		}

		if strings.Split(previewFile.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Preview image file type is not allowed",
			})
		}
	}

	post := models.Post{
		ChannelId:      data.ChannelId,
		PreviewImageId: data.PreviewImageId,
		Title:          data.Title,
		Content:        data.Content,
	}

	var tags []models.Tag
	var postImages []models.File
	var postFiles []models.File

	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to create post")
		}

		if err := tx.Where("id IN ?", data.Tags).Find(&tags).Error; err != nil || len(tags) != len(data.Tags) {
			return fiber.NewError(fiber.StatusBadRequest, "One or more tag Ids are invalid")
		}

		if err := tx.Model(&post).Association("Tags").Append(tags); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to attach tags to post")
		}

		if len(data.PostImages) > 0 {
			if err := tx.Where("id IN ?", data.PostImages).Find(&postImages).Error; err != nil || len(postImages) != len(data.PostImages) {
				return fiber.NewError(fiber.StatusBadRequest, "One or more post image Ids are invalid")
			}

			for _, file := range postImages {
				if file.OwnerId != uint(userId) {
					return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("You don't own file with Id %d", file.Id))
				}
				if strings.Split(file.MimeType, "/")[0] != "image" {
					return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("File with Id %d is not an image", file.Id))
				}

			}

			if err := tx.Model(&post).Association("PostImages").Append(postImages); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to attach post images")
			}
		}

		if len(data.PostFiles) > 0 {
			if err := tx.Where("id IN ?", data.PostFiles).Find(&postFiles).Error; err != nil || len(postFiles) != len(data.PostFiles) {
				return fiber.NewError(fiber.StatusBadRequest, "One or more post file Ids are invalid")
			}
			if err := tx.Model(&post).Association("PostFiles").Append(postFiles); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to attach post files")
			}
		}

		return nil
	})

	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(ErrorResponse{
			Message: err.(*fiber.Error).Error(),
		})
	}

	if err := repository.DB.First(&post, post.Id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	var channelLogo *models.File
	if channel.LogoId != nil {
		if err := repository.DB.First(&channelLogo, *channel.LogoId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error logo does not exist"})
		}
		if strings.Split(channelLogo.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Logo image file type is not allowed"})
		}
	}

	channelResponse := ConvertChannelToResponse(channel, channelLogo)

	tagsResponse, err := GetTagResponsesForPost(post.Id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch tags for post",
		})
	}

	return c.JSON(DataResponse[PostResponse]{
		Data:    ConvertPostToResponse(post, previewFile, channelResponse, tagsResponse),
		Message: "Post created successfully",
	})
}

// EditPost редактирует пост
// @Summary      Редактирование поста
// @Description  Изменяет заголовок, содержимое и теги поста. Пользователь должен быть владельцем канала.
// @Tags         Post
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      EditPostRequest true "Данные для обновления поста (postId, title?, content?, tags?)"
// @Success      200   {object}  DataResponse[PostResponse]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/editPost [patch]
func EditPost(c *fiber.Ctx) error {
	var data EditPostRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	uintId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	if data.PostId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Post id is required",
		})
	}

	var post models.Post
	if result := repository.DB.First(&post, data.PostId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Post not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.Preload("Category").First(&channel, post.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(ErrorResponse{
			Message: "You are not the owner of this post",
		})
	}

	if data.Title != "" {
		post.Title = data.Title
	}
	if data.Content != "" {
		post.Content = data.Content
	}

	var previewFile *models.File
	if data.PreviewImageId != nil {
		if err := repository.DB.First(&previewFile, *data.PreviewImageId).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Preview image does not exist")
		}

		if previewFile.OwnerId != uint(uintId) {
			return fiber.NewError(fiber.StatusForbidden, "You don't own the preview image")
		}

		if strings.Split(previewFile.MimeType, "/")[0] != "image" {
			return fiber.NewError(fiber.StatusBadRequest, "Preview image file type is not allowed")
		}

		post.PreviewImageId = data.PreviewImageId
	} else if post.PreviewImageId != nil {
		if err := repository.DB.First(&previewFile, *post.PreviewImageId).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Preview image does not exist")
		}

		if strings.Split(previewFile.MimeType, "/")[0] != "image" {
			return fiber.NewError(fiber.StatusBadRequest, "Preview image file type is not allowed")
		}
	}

	// Начинаем транзакцию
	err = repository.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&post).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update post")
		}

		if data.Tags != nil {
			var tags []models.Tag
			if err := tx.Where("id IN ?", data.Tags).Find(&tags).Error; err != nil || len(tags) != len(data.Tags) {
				return fiber.NewError(fiber.StatusBadRequest, "One or more tag IDs are invalid")
			}

			if err := tx.Model(&post).Association("Tags").Replace(tags); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update tags")
			}
		}

		if data.PostImages != nil {
			var postImages []models.File
			if err := tx.Where("id IN ?", data.PostImages).Find(&postImages).Error; err != nil || len(postImages) != len(data.PostImages) {
				return fiber.NewError(fiber.StatusBadRequest, "One or more post image IDs are invalid")
			}
			for _, file := range postImages {
				if file.OwnerId != uint(uintId) {
					return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("You don't own image with ID %d", file.Id))
				}
				if strings.Split(file.MimeType, "/")[0] != "image" {
					return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("File with ID %d is not an image", file.Id))
				}
			}
			if err := tx.Model(&post).Association("PostImages").Replace(&postImages); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update post images")
			}
		}

		if data.PostFiles != nil {
			var postFiles []models.File
			if err := tx.Where("id IN ?", data.PostFiles).Find(&postFiles).Error; err != nil || len(postFiles) != len(data.PostFiles) {
				return fiber.NewError(fiber.StatusBadRequest, "One or more post file IDs are invalid")
			}
			for _, file := range postFiles {
				if file.OwnerId != uint(uintId) {
					return fiber.NewError(fiber.StatusForbidden, fmt.Sprintf("You don't own file with ID %d", file.Id))
				}
			}
			if err := tx.Model(&post).Association("PostFiles").Replace(&postFiles); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update post files")
			}
		}

		return nil
	})

	if err != nil {
		return c.JSON(ErrorResponse{
			Message: err.Error(),
		})
	}

	if err := repository.DB.First(&post, data.PostId).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	var channelLogo *models.File
	if channel.LogoId != nil {
		if err := repository.DB.First(&channelLogo, *channel.LogoId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error logo does not exist"})
		}
		if strings.Split(channelLogo.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Logo image file type is not allowed"})
		}
	}

	channelResponse := ConvertChannelToResponse(channel, channelLogo)

	tagsResponse, err := GetTagResponsesForPost(post.Id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch tags for post",
		})
	}

	return c.JSON(DataResponse[PostResponse]{
		Data:    ConvertPostToResponse(post, previewFile, channelResponse, tagsResponse),
		Message: "Post updated successfully",
	})
}

// DeletePost удаляет пост
// @Summary      Удаление поста
// @Description  Удаляет пост, если пользователь — владелец канала
// @Tags         Post
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int true "Id поста"
// @Success      200   {object}  MessageResponse "Сообщение об успешном удалении"
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/deletePost/{id} [delete]
func DeletePost(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	Id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || Id == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid post id",
		})
	}

	var post models.Post
	if result := repository.DB.First(&post, Id); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Post not found",
		})
	}

	var channel models.Channel
	if result := repository.DB.Preload("Category").First(&channel, post.ChannelId); result.Error != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	if channel.OwnerId != uint(userId) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(ErrorResponse{
			Message: "You are not the owner of this post",
		})
	}

	if err := repository.DB.Delete(&post).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to delete post",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Post successfully deleted",
	})
}

// GetPost возвращает пост по Id
// @Summary      Получение поста
// @Description  Возвращает пост с тегами по Id
// @Tags         Post
// @Accept       json
// @Produce      json
// @Param        id      path      int true "Id поста"
// @Success      200     {object}  DataResponse[PostResponse]
// @Failure      400     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Router       /api/getPost/{id} [get]
func GetPost(c *fiber.Ctx) error {
	Id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || Id == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid post id",
		})
	}

	var post models.Post
	if err := repository.DB.Preload("PostImages").Preload("PostFiles").
		First(&post, Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch post",
		})
	}

	var channel models.Channel
	if err := repository.DB.Preload("Category").First(&channel, post.ChannelId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Channel not found",
			})
		}
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to retrieve channel",
		})
	}

	var channelLogo *models.File
	if channel.LogoId != nil {
		if err := repository.DB.First(&channelLogo, *channel.LogoId).Error; err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Error logo does not exist"})
		}
		if strings.Split(channelLogo.MimeType, "/")[0] != "image" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(ErrorResponse{
				Message: "Logo image file type is not allowed"})
		}
	}

	channelResponse := ConvertChannelToResponse(channel, channelLogo)

	tags, err := GetTagResponsesForPost(post.Id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch tags for post",
		})
	}

	if err := repository.DB.Model(&post).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to update view count",
		})
	}

	var previewFile *models.File
	if post.PreviewImageId != nil {
		var file models.File
		if err := repository.DB.First(&file, *post.PreviewImageId).Error; err == nil {
			previewFile = &file
		}
	}

	return c.JSON(DataResponse[PostResponse]{
		Data: ConvertPostToResponse(post, previewFile, channelResponse, tags),
	})
}

// GetPosts получает посты по Id канала с пагинацией
// @Summary      Get posts
// @Description  Получает список постов по Id канала с пагинацией (по 10 на страницу)
// @Tags         Post
// @Accept       json
// @Produce      json
// @Param        channelId  path      int true  "Id канала"
// @Param        page       query     int false "Номер страницы (по умолчанию 1)"
// @Success      200        {array}   DataResponse[[]PostResponse]
// @Failure      400        {object}  ErrorResponse
// @Failure      500        {object}  ErrorResponse
// @Router       /api/getPosts/{channelId} [get]
func GetPosts(c *fiber.Ctx) error {
	channelId, err := strconv.ParseUint(c.Params("channelId"), 10, 64)
	if err != nil || channelId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid channel id",
		})
	}

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid page value",
		})
	}

	offset := (page - 1) * 10

	var posts []models.Post
	if err := repository.DB.Preload("PostImages").Preload("PostFiles").
		Where("channel_id = ?", channelId).Limit(10).Offset(offset).Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch posts",
		})
	}

	var postsResponse []PostResponse
	for _, post := range posts {

		var preview *models.File
		if post.PreviewImageId != nil {
			if err := repository.DB.First(&preview, *post.PreviewImageId).Error; err != nil {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Error logo does not exist",
				})
			}

			if strings.Split(preview.MimeType, "/")[0] != "image" {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Logo image file type is not allowed",
				})
			}
		}

		var channel models.Channel
		if err := repository.DB.Preload("Category").First(&channel, channelId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.Status(fiber.StatusNotFound)
				return c.JSON(fiber.Map{
					"message": "Channel not found",
				})
			}
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(ErrorResponse{
				Message: "Failed to retrieve channel",
			})
		}

		var channelLogo *models.File
		if channel.LogoId != nil {
			if err := repository.DB.First(&channelLogo, *channel.LogoId).Error; err != nil {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Error logo does not exist"})
			}
			if strings.Split(channelLogo.MimeType, "/")[0] != "image" {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Logo image file type is not allowed"})
			}
		}

		channelResponse := ConvertChannelToResponse(channel, channelLogo)

		tagsResponse, err := GetTagResponsesForPost(post.Id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Message: "Failed to fetch tags for post",
			})
		}

		postsResponse = append(postsResponse, ConvertPostToResponse(post, preview, channelResponse, tagsResponse))
	}

	return c.JSON(DataResponse[[]PostResponse]{
		Data: postsResponse,
	})
}

// GetRecommendedPosts возвращает список рекомендуемых постов по кастомной формуле
// @Summary      Рекомендуемые посты
// @Description  Получает список рекомендуемых постов за последнюю неделю, сортируя по формуле: views + likes*3 - dislikes*2 + comments*2
// @Tags         Post
// @Accept       json
// @Produce      json
// @Param        page  query     int false "Номер страницы (по умолчанию 1)"
// @Success      200   {object}  DataResponse[[]PostResponse]
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/getRecommendedPosts [get]
func GetRecommendedPosts(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid page value",
		})
	}

	offset := (page - 1) * 10

	type PostWithRating struct {
		Id            uint
		CommentsCount int `json:"comments_count"`
		Rating        int `json:"-"`
	}

	var postInfos []PostWithRating

	subQuery := repository.DB.
		Table("comments").
		Select("post_id, COUNT(*) as comments_count").
		Group("post_id")

	if err := repository.DB.
		Table("posts").
		Select(`posts.id, 
		COALESCE(c.comments_count, 0) as comments_count,
		(views_count + likes_count * 3 - dislikes_count * 2 + COALESCE(c.comments_count, 0) * 2) as rating`).
		Joins("LEFT JOIN (?) as c ON posts.id = c.post_id", subQuery).
		Where("posts.created_at >= NOW() - INTERVAL '7 days'").
		Order("rating DESC").
		Limit(10).Offset(offset).
		Scan(&postInfos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch popular post IDs",
		})
	}

	ids := make([]uint, len(postInfos))
	indexMap := make(map[uint]int, len(postInfos))
	for i, p := range postInfos {
		ids[i] = p.Id
		indexMap[p.Id] = i
	}

	var posts []models.Post
	if err := repository.DB.
		Preload("PostImages").
		Preload("PostFiles").
		Where("id IN ?", ids).
		Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch posts",
		})
	}

	sortedPosts := make([]models.Post, len(posts))
	for _, post := range posts {
		if idx, ok := indexMap[post.Id]; ok {
			sortedPosts[idx] = post
		}
	}

	postsResponse := make([]PostResponse, 0, len(sortedPosts))
	for _, post := range sortedPosts {
		var preview *models.File
		if post.PreviewImageId != nil {
			var file models.File
			if err := repository.DB.First(&file, *post.PreviewImageId).Error; err == nil {
				preview = &file
			}
		}

		var channel models.Channel
		if err := repository.DB.Preload("Category").First(&channel, post.ChannelId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.Status(fiber.StatusNotFound)
				return c.JSON(fiber.Map{
					"message": "Channel not found",
				})
			}
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(ErrorResponse{
				Message: "Failed to retrieve channel",
			})
		}

		var channelLogo *models.File
		if channel.LogoId != nil {
			if err := repository.DB.First(&channelLogo, *channel.LogoId).Error; err != nil {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Error logo does not exist"})
			}
			if strings.Split(channelLogo.MimeType, "/")[0] != "image" {
				c.Status(fiber.StatusBadRequest)
				return c.JSON(ErrorResponse{
					Message: "Logo image file type is not allowed"})
			}
		}

		channelResponse := ConvertChannelToResponse(channel, channelLogo)

		tagsResponse, err := GetTagResponsesForPost(post.Id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Message: "Failed to fetch tags for post",
			})
		}

		postsResponse = append(postsResponse, ConvertPostToResponse(post, preview, channelResponse, tagsResponse))
	}

	return c.JSON(DataResponse[[]PostResponse]{
		Data: postsResponse,
	})
}

// SetReaction устанавливает реакцию пользователя на пост
// @Summary      Set reaction to post
// @Description  Устанавливает реакцию (лайк/дизлайк) пользователя на пост. Повторное нажатие удаляет реакцию
// @Tags         Post
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      SetReactionRequest true "Данные реакции"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/setReaction [post]
func SetReaction(c *fiber.Ctx) error {
	var data SetReactionRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	if data.PostId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Post id is required",
		})
	}

	if data.Reaction != "like" && data.Reaction != "dislike" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid reaction type",
		})
	}

	reactionValue := data.Reaction == "like"

	var existingReaction models.PostReaction
	if err := repository.DB.Where("post_id = ? AND user_id = ?", data.PostId, userId).
		First(&existingReaction).Error; err == nil {
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
		return c.JSON(ErrorResponse{
			Message: "Reaction updated",
		})
	}

	newReaction := models.PostReaction{
		PostId:   data.PostId,
		UserId:   uint(userId),
		Reaction: reactionValue,
	}

	if err := repository.DB.Create(&newReaction).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to add reaction",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Reaction added",
	})
}

// CreateComment создает комментарий к посту или ответ на комментарий
// @Summary      Создание комментария
// @Description  Создает новый комментарий к посту или ответ на существующий комментарий
// @Tags         Comment
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      CreateCommentRequest true "Данные комментария"
// @Success      200   {object}  DataResponse[models.Comment]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/createComment [post]
func CreateComment(c *fiber.Ctx) error {
	var data CreateCommentRequest

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid input",
		})
	}

	if data.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Content is required",
		})
	}

	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	var post models.Post
	if result := repository.DB.First(&post, data.PostId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Post not found",
		})
	}

	if data.ParentId != nil {
		var parent models.Comment
		if err := repository.DB.First(&parent, *data.ParentId).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Message: "Parent comment not found",
			})
		}
		if parent.PostId != data.PostId {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Message: "Parent comment does not belong to this post",
			})
		}
	}

	comment := models.Comment{
		PostId:   data.PostId,
		UserId:   uint(userId),
		Content:  data.Content,
		ParentId: data.ParentId,
	}

	if err := repository.DB.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to create comment",
		})
	}

	return c.JSON(DataResponse[models.Comment]{
		Data:    comment,
		Message: "Comment created successfully",
	})
}

// DeleteComment удаляет комментарий пользователя
// @Summary      Удаление комментария
// @Description  Удаляет комментарий, если нет ответов — полностью, иначе помечает как удаленный
// @Tags         Comment
// @Security     ApiKeyAuth
// @Produce      json
// @Param        id   path      int true  "Id комментария"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/deleteComment/{id} [delete]
func DeleteComment(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	strId, ok := token.Claims.(jwt.MapClaims)["iss"].(string)
	if !ok {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid token",
		})
	}

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	Id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || Id == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid comment id",
		})
	}

	var comment models.Comment
	if err := repository.DB.First(&comment, Id).Error; err != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Comment not found",
		})
	}

	if comment.UserId != uint(userId) {
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Message: "You are not the author of this comment",
		})
	}

	var repliesCount int64
	repository.DB.Model(&models.Comment{}).Where("parent_id = ?", comment.Id).Count(&repliesCount)

	if repliesCount > 0 {
		comment.IsDeleted = true
		comment.Content = ""
		if err := repository.DB.Save(&comment).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Message: "Failed to mark comment as deleted",
			})
		}
		return c.JSON(MessageResponse{
			Message: "Comment marked as deleted",
		})
	}

	if err := repository.DB.Delete(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to delete comment",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Comment deleted successfully",
	})
}

// GetPostComments возвращает корневые или дочерние комментарии поста с пагинацией
// @Summary      Получение комментариев
// @Description  Возвращает комментарии к посту, поддерживает пагинацию и фильтрацию по parentId
// @Tags         Comment
// @Accept       json
// @Produce      json
// @Param        postId     query     int  true  "Id поста"
// @Param        parentId   query     int  false "Id родительского комментария (для подкомментариев)"
// @Param        offset     query     int  false "Смещение (для пагинации)"
// @Param        limit      query     int  false "Количество комментариев на странице"
// @Success      200  {object}  DataResponse[[]CommentResponse]
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/getPostComments [get]
func GetPostComments(c *fiber.Ctx) error {
	postId, err := strconv.ParseUint(c.Query("postId"), 10, 64)
	if err != nil || postId == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Message: "Invalid postId"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var parentId *uint
	if pid := c.Query("parentId"); pid != "" {
		if parsed, err := strconv.ParseUint(pid, 10, 64); err == nil {
			p := uint(parsed)
			parentId = &p
		}
	}

	// Загружаем комментарии
	var comments []models.Comment
	query := repository.DB.Where("post_id = ?", postId)
	if parentId != nil {
		query = query.Where("parent_id = ?", *parentId)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	query = query.Order("created_at ASC").Limit(limit).Offset(offset)
	if err := query.Find(&comments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Message: "Failed to load comments"})
	}

	// Собираем userIds
	userIds := make([]uint, 0)
	for _, c := range comments {
		userIds = append(userIds, c.UserId)
	}

	// Загружаем пользователей и их лого
	var users []models.User
	if len(userIds) > 0 {
		repository.DB.Where("id IN ?", userIds).Find(&users)
	}

	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.Id] = u
	}

	// Загружаем файлы (лого)
	var fileIds []uint
	for _, u := range users {
		if u.LogoId != nil {
			fileIds = append(fileIds, *u.LogoId)
		}
	}
	var files []models.File
	if len(fileIds) > 0 {
		repository.DB.Where("id IN ?", fileIds).Find(&files)
	}
	fileMap := make(map[uint]models.File)
	for _, f := range files {
		fileMap[f.Id] = f
	}

	// Подсчёт количества ответов
	type CountResult struct {
		ParentID uint
		Count    int
	}
	var counts []CountResult
	repository.DB.Table("comments").
		Select("parent_id, COUNT(*) as count").
		Where("post_id = ? AND parent_id IS NOT NULL", postId).
		Group("parent_id").
		Scan(&counts)

	replyCountMap := make(map[uint]int)
	for _, c := range counts {
		replyCountMap[c.ParentID] = c.Count
	}

	// Формирование ответа
	commentResponses := make([]CommentResponse, 0, len(comments))
	for _, cmt := range comments {
		user := userMap[cmt.UserId]
		var fileResponse *FileResponse
		if user.LogoId != nil {
			if file, ok := fileMap[*user.LogoId]; ok {
				fileResponse = &FileResponse{
					Id:  file.Id,
					Url: storage.GetUrl(file.Filename),
				}
			}
		}
		resp := CommentResponse{
			Id:           cmt.Id,
			PostId:       cmt.PostId,
			ParentId:     cmt.ParentId,
			Content:      cmt.Content,
			IsDeleted:    cmt.IsDeleted,
			RepliesCount: replyCountMap[cmt.Id],
			User: struct {
				Id    uint          `json:"id"`
				Login string        `json:"login"`
				Logo  *FileResponse `json:"logo"`
			}{
				Id:    user.Id,
				Login: user.Login,
				Logo:  fileResponse,
			},
		}
		commentResponses = append(commentResponses, resp)
	}

	return c.JSON(DataResponse[[]CommentResponse]{
		Data: commentResponses,
	})
}

func ConvertPostToResponse(post models.Post, previewImage *models.File, channel *ChannelResponse, tags []TagResponse) PostResponse {
	var preview *FileResponse
	if previewImage != nil {
		preview = &FileResponse{
			Id:  previewImage.Id,
			Url: storage.GetUrl(previewImage.Filename)}

	}

	postImages := make([]FileResponse, 0, len(post.PostImages))
	for _, f := range post.PostImages {
		postImages = append(postImages, FileResponse{
			Id:  f.Id,
			Url: storage.GetUrl(f.Filename),
		})
	}

	postFiles := make([]FileResponse, 0, len(post.PostFiles))
	for _, f := range post.PostFiles {
		postFiles = append(postFiles, FileResponse{
			Id:  f.Id,
			Url: storage.GetUrl(f.Filename),
		})
	}

	return PostResponse{
		Id:            post.Id,
		Channel:       channel,
		PreviewImage:  preview,
		Title:         post.Title,
		Content:       post.Content,
		LikesCount:    post.LikesCount,
		DislikesCount: post.DislikesCount,
		ViewsCount:    post.ViewsCount,
		PostImages:    postImages,
		PostFiles:     postFiles,
		Tags:          tags,
		CreatedAt:     post.CreatedAt,
	}
}

func GetTagResponsesForPost(postId uint) ([]TagResponse, error) {
	var tags []TagResponse
	err := repository.DB.
		Table("tags").
		Select("tags.id, tags.category_id, tags.name, categories.color").
		Joins("LEFT JOIN categories ON tags.category_id = categories.id").
		Joins("LEFT JOIN post_tags ON post_tags.tag_id = tags.id").
		Where("post_tags.post_id = ?", postId).
		Scan(&tags).Error
	return tags, err
}
