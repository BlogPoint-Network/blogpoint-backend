package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"errors"
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
// @Success      200   {object}  DataResponse[models.Post]
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

	uintId, err := strconv.ParseUint(strId, 10, 32)
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
	if result := repository.DB.First(&channel, data.ChannelId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	if channel.OwnerId != uint(uintId) {
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

	tagNames := strings.Split(strings.TrimSpace(data.Tags), ",")
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
		ChannelId: data.ChannelId,
		Title:     data.Title,
		Content:   data.Content,
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
		return c.JSON(ErrorResponse{
			Message: "Transaction error",
		})
	}

	if err := repository.DB.Preload("Tags").First(&post, post.Id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	return c.JSON(DataResponse[models.Post]{
		Data:    post,
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
// @Success      200   {object}  DataResponse[models.Post]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/editPost [put]
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
	if result := repository.DB.First(&channel, post.ChannelId); result.Error != nil {
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

	tagNames := strings.Split(strings.TrimSpace(data.Tags), ",")
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
		return c.JSON(ErrorResponse{
			Message: "Transaction error",
		})
	}

	if err := repository.DB.Preload("Tags").First(&post, data.PostId).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to load post with tags"})
	}

	return c.JSON(DataResponse[models.Post]{
		Data:    post,
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
	if result := repository.DB.First(&channel, post.ChannelId); result.Error != nil {
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
// @Success      200     {object}  DataResponse[models.Post]
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
	if err := repository.DB.Preload("Tags").First(&post, Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch post",
		})
	}

	var comments []models.Comment
	if err := repository.DB.Where("post_id = ? AND parent_id IS NULL", post.Id).
		Order("created_at ASC").Find(&comments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to load comments",
		})
	}

	userIds := make([]uint, 0)
	for _, c := range comments {
		userIds = append(userIds, c.UserId)
	}

	var users []models.User
	if len(userIds) > 0 {
		repository.DB.Where("id IN ?", userIds).Find(&users)
	}

	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.Id] = u
	}

	type CountResult struct {
		ParentID uint
		Count    int
	}

	var counts []CountResult
	repository.DB.Table("comments").
		Select("parent_id, COUNT(*) as count").
		Where("post_id = ? AND parent_id IS NOT NULL", post.Id).
		Group("parent_id").
		Scan(&counts)

	replyCountMap := make(map[uint]int)
	for _, c := range counts {
		replyCountMap[c.ParentID] = c.Count
	}

	var commentResponses []CommentResponse
	for _, cmt := range comments {
		resp := CommentResponse{
			Id:           cmt.Id,
			PostId:       cmt.PostId,
			ParentId:     nil,
			Content:      cmt.Content,
			IsDeleted:    cmt.IsDeleted,
			RepliesCount: replyCountMap[cmt.Id],
		}

		if user, ok := userMap[cmt.UserId]; ok {
			resp.User = struct {
				Id       uint   `json:"id"`
				Username string `json:"username"`
				//AvatarURL string `json:"avatarUrl"`
			}{
				Id:       user.Id,
				Username: user.Login,
				//AvatarURL: user.AvatarUrl,
			}
		}

		commentResponses = append(commentResponses, resp)
	}

	if err := repository.DB.Model(&post).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to update view count",
		})
	}

	postResponse := PostResponse{
		Post:     post,
		Comments: commentResponses,
	}

	return c.JSON(DataResponse[PostResponse]{
		Data: postResponse,
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
// @Success      200        {array}   DataResponse[[]models.Post]
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
	if err := repository.DB.Preload("Tags").Where("channel_id = ?", channelId).Limit(10).Offset(offset).Find(&posts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch posts",
		})
	}

	return c.JSON(DataResponse[[]models.Post]{
		Data: posts,
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

// GetChildComments получает вложенные комментарии по parentId
// @Summary      Получение ответов на комментарий
// @Description  Возвращает дочерние комментарии по parentId
// @Tags         Comment
// @Accept       json
// @Produce      json
// @Param        parentId  path  int true "Id родительского комментария"
// @Success      200  {array}  DataResponse[[]models.Comment]
// @Failure      400  {object}  ErrorResponse
// @Router       /api/getReplies [get]
func GetChildComments(c *fiber.Ctx) error {
	parentId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || parentId == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid parent id",
		})
	}

	var replies []models.Comment
	if err := repository.DB.
		Where("parent_id = ?", parentId).
		Order("created_at ASC").
		Find(&replies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to fetch replies",
		})
	}

	// Собираем userIds
	userIds := make([]uint, 0, len(replies))
	for _, r := range replies {
		userIds = append(userIds, r.UserId)
	}

	var users []models.User
	if len(userIds) > 0 {
		repository.DB.Where("id IN ?", userIds).Find(&users)
	}

	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.Id] = u
	}

	// Получаем количество ответов на каждый комментарий
	var counts []struct {
		ParentID uint
		Count    int
	}
	repository.DB.Table("comments").
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN ?", userIds).
		Group("parent_id").
		Scan(&counts)

	replyCountMap := make(map[uint]int)
	for _, c := range counts {
		replyCountMap[c.ParentID] = c.Count
	}

	var response []CommentResponse
	for _, r := range replies {
		resp := CommentResponse{
			Id:           r.Id,
			PostId:       r.PostId,
			ParentId:     r.ParentId,
			Content:      r.Content,
			IsDeleted:    r.IsDeleted,
			RepliesCount: replyCountMap[r.Id],
		}

		if user, ok := userMap[r.UserId]; ok {
			resp.User = struct {
				Id       uint   `json:"id"`
				Username string `json:"username"`
			}{
				Id:       user.Id,
				Username: user.Login,
			}
		}

		response = append(response, resp)
	}

	return c.JSON(DataResponse[[]CommentResponse]{Data: response})
}
