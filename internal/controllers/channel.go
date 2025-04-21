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
)

// CreateChannel создает новый канал
// @Summary      Создание канал
// @Description  Создает канал для текущего пользователя
// @Tags         Channel
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      CreateChannelRequest true "Данные канала"
// @Success      200   {object}  DataResponse[models.Channel]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /api/createChannel [post]
func CreateChannel(c *fiber.Ctx) error {

	var data CreateChannelRequest

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

	var uintId uint
	if parsedId, err := strconv.ParseUint(strId, 10, 32); err == nil {
		uintId = uint(parsedId)
	} else {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer Id",
		})
	}

	if data.Name == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Incorrect data",
		})
	}

	if data.CategoryId != nil {
		var category models.Category
		if err := repository.DB.First(&category, data.CategoryId).Error; err != nil {
			c.Status(fiber.StatusNotFound)
			return c.JSON(ErrorResponse{
				Message: "Category not found",
			})
		}
	}

	channel := models.Channel{
		Name:        data.Name,
		Description: data.Description,
		CategoryId:  data.CategoryId,
		OwnerId:     uintId,
	}
	repository.DB.Create(&channel)
	repository.DB.Preload("Category").First(&channel, channel.Id)

	return c.JSON(DataResponse[models.Channel]{
		Data:    channel,
		Message: "Channel created successfully",
	})
}

// EditChannel редактирует существующий канал
// @Summary      Редактироввние канала
// @Description  Изменяет название и описание канала пользователя
// @Tags         Channel
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        data  body      EditChannelRequest true "Обновленные данные канала"
// @Success      200   {object}  DataResponse[models.Channel]
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/editChannel [patch]
func EditChannel(c *fiber.Ctx) error {
	var data EditChannelRequest

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

	Id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Invalid issuer id",
		})
	}

	if data.ChannelId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Channel id is required",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, data.ChannelId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	if channel.OwnerId != uint(Id) {
		c.Status(fiber.StatusForbidden)
		return c.JSON(ErrorResponse{
			Message: "You are not the owner of this channel",
		})
	}

	if data.Name != "" {
		channel.Name = data.Name
	}
	if data.Description != "" {
		channel.Description = data.Description
	}

	if data.CategoryId != nil {
		if *data.CategoryId == 0 {
			channel.CategoryId = nil
		} else {
			var category models.Category
			if err := repository.DB.First(&category, *data.CategoryId).Error; err != nil {
				c.Status(fiber.StatusNotFound)
				return c.JSON(ErrorResponse{
					Message: "Category not found",
				})
			}
			channel.CategoryId = data.CategoryId
		}
	}

	if err := repository.DB.Save(&channel).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to update channel",
		})
	}
	repository.DB.Preload("Category").First(&channel, channel.Id)

	return c.JSON(DataResponse[models.Channel]{
		Data:    channel,
		Message: "Channel edited successfully",
	})
}

// DeleteChannel удаляет канал
// @Summary      Удаление канала
// @Description  Удаляет канал, если пользователь является его владельцем
// @Tags         Channel
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int true "ID канала"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/deleteChannel/{id} [delete]
func DeleteChannel(c *fiber.Ctx) error {
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

	channelId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || channelId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid channel id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, channelId); result.Error != nil {
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

	if err := repository.DB.Delete(&channel).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to delete channel",
		})
	}

	return c.JSON(MessageResponse{
		Message: "Channel successfully deleted",
	})
}

// GetUserSubscriptions возвращает список каналов, на которые подписан пользователь
// @Summary      Получение подписок пользователя
// @Description  Возвращает список каналов, на которые подписан текущий пользователь
// @Tags         Channel
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {array}   DataResponse[models.Channel]
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/getUserSubscriptions [get]
func GetUserSubscriptions(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var channels []models.Channel
	if err := repository.DB.Joins("JOIN subscriptions ON subscriptions.channel_id = channels.id").
		Where("subscriptions.user_id = ?", claims["iss"]).
		Find(&channels).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to retrieve subscriptions",
		})
	}

	if len(channels) == 0 {
		return c.JSON(ErrorResponse{
			Message: "No subscriptions found",
		})
	}

	return c.JSON(DataResponse[[]models.Channel]{
		Data: channels,
	})
}

// GetUserChannels возвращает каналы, созданные пользователем
// @Summary      Получение каналов пользователя
// @Description  Возвращает список каналов, созданных текущим пользователем
// @Tags         Channel
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200  {array}   DataResponse[models.Channel]
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/getUserChannels [get]
func GetUserChannels(c *fiber.Ctx) error {
	token, err := jwt.ParseWithClaims(c.Cookies("jwt"), jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(ErrorResponse{
			Message: "Unauthenticated",
		})
	}

	claims := token.Claims.(jwt.MapClaims)

	var channels []models.Channel
	if err := repository.DB.Where("owner_id = ?", claims["iss"]).Find(&channels).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(ErrorResponse{
			Message: "Failed to retrieve channels",
		})
	}

	if len(channels) == 0 {
		return c.JSON(ErrorResponse{
			Message: "No channels found",
		})
	}

	return c.JSON(DataResponse[[]models.Channel]{
		Data: channels,
	})
}

// GetChannel возвращает информацию о канале
// @Summary      Получение канала
// @Description  Возвращает информацию о канале по ID
// @Tags         Channel
// @Accept       json
// @Produce      json
// @Param        id    path      int true "ID канала"
// @Success      200   {object}  DataResponse[models.Channel]
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/getChannel/{id} [get]
func GetChannel(c *fiber.Ctx) error {
	Id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || Id == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid channel id",
		})
	}

	var channel models.Channel
	if err := repository.DB.First(&channel, Id).Error; err != nil {
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

	return c.JSON(DataResponse[models.Channel]{
		Data: channel,
	})
}

// GetPopularChannels возвращает популярные каналы
// @Summary      Получение популярных каналов
// @Description  Возвращает список каналов, отсортированных по количеству подписчиков по убыванию
// @Tags         Channel
// @Produce      json
// @Success      200  {array}   DataResponse[models.Channel]
// @Failure      500  {object}  ErrorResponse
// @Router       /api/getPopularChannels [get]
func GetPopularChannels(c *fiber.Ctx) error {
	var channels []models.Channel

	if err := repository.DB.Order("subs_count DESC").Find(&channels).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: "Failed to retrieve popular channels"})
	}

	if len(channels) == 0 {
		return c.JSON(ErrorResponse{
			Message: "No channels found",
		})
	}

	return c.JSON(DataResponse[[]models.Channel]{
		Data: channels,
	})
}

// SubscribeChannel подписывает пользователя на канал
// @Summary      Подписка на канал
// @Description  Подписывает пользователя на указанный канал
// @Tags         Channel
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int true "Id канала"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /api/subscribeChannel/{id} [post]
func SubscribeChannel(c *fiber.Ctx) error {
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
			Message: "Invalid user Id",
		})
	}

	channelId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || channelId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid channel id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, channelId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	var subscription models.Subscription
	if result := repository.DB.Where("user_id = ? AND channel_id = ?", userId, channelId).First(&subscription); result.Error == nil {
		c.Status(fiber.StatusConflict)
		return c.JSON(ErrorResponse{
			Message: "Already subscribed",
		})
	}

	subscription = models.Subscription{
		UserId:    uint(userId),
		ChannelId: uint(channelId),
	}

	repository.DB.Create(&subscription)

	repository.DB.Model(&channel).Update("subs_count", channel.SubsCount+1)

	return c.JSON(MessageResponse{
		Message: "Subscription successful",
	})

}

// UnsubscribeChannel отписывает пользователя от канала
// @Summary      Отписаться от канала
// @Description  Отписывает пользователя от указанного канала
// @Tags         Channel
// @Security     ApiKeyAuth
// @Accept       json
// @Produce      json
// @Param        id    path      int true "ID канала"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /api/unsubscribeChannel/{id} [delete]
func UnsubscribeChannel(c *fiber.Ctx) error {
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
			Message: "Invalid user Id",
		})
	}

	channelId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || channelId == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ErrorResponse{
			Message: "Invalid channel id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, channelId); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Channel not found",
		})
	}

	result := repository.DB.Where("user_id = ? AND channel_id = ?", userId, channelId).Delete(&models.Subscription{})
	if result.RowsAffected == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(ErrorResponse{
			Message: "Not subscribed",
		})
	}

	if channel.SubsCount > 0 {
		repository.DB.Model(&channel).Update("subs_count", channel.SubsCount-1)
	}

	return c.JSON(MessageResponse{
		Message: "Unsubscribed successfully",
	})
}
