package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"strconv"
)

func CreateChannel(c fiber.Ctx) error {

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

	var uintId uint
	if parsedId, err := strconv.ParseUint(strId, 10, 32); err == nil {
		uintId = uint(parsedId)
	} else {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid issuer Id",
		})
	}

	var name string
	var description string

	if data["name"] != "" {
		name = data["name"]
	} else {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Incorrect data",
		})
	}
	if data["description"] != "" {
		description = data["description"]
	}

	channel := models.Channel{
		Name:        name,
		Description: description,
		OwnerId:     uintId,
	}

	repository.DB.Create(&channel)

	return c.JSON(channel)
}

func EditChannel(c fiber.Ctx) error {
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

	if data["channelId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Channel id is required",
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

	if data["name"] != "" {
		channel.Name = data["name"]
	}
	if data["description"] != "" {
		channel.Description = data["description"]
	}

	if err := repository.DB.Save(&channel).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to update channel",
		})
	}

	return c.JSON(channel)
}

func DeleteChannel(c fiber.Ctx) error {
	var data map[string]string

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		fmt.Println(err)
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

	if data["channelId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Channel id is required",
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

	if err := repository.DB.Delete(&channel).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to delete channel",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Channel successfully deleted",
	})
}

func GetUserSubscriptions(c fiber.Ctx) error {
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

	claims := token.Claims.(jwt.MapClaims)

	var channels []models.Channel
	if err := repository.DB.Joins("JOIN subscriptions ON subscriptions.channel_id = channels.id").
		Where("subscriptions.user_id = ?", claims["iss"]).
		Find(&channels).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to retrieve subscriptions",
		})
	}

	if len(channels) == 0 {
		return c.JSON(fiber.Map{
			"message": "No subscriptions found",
		})
	}

	return c.JSON(channels)
}

func GetUserChannels(c fiber.Ctx) error {
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

	claims := token.Claims.(jwt.MapClaims)

	var channels []models.Channel
	if err := repository.DB.Where("owner_id = ?", claims["iss"]).Find(&channels).Error; err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to retrieve channels",
		})
	}

	if len(channels) == 0 {
		return c.JSON(fiber.Map{
			"message": "No channels found",
		})
	}

	return c.JSON(channels)
}

func GetChannel(c fiber.Ctx) error {
	var data map[string]uint

	if err := json.Unmarshal(c.Body(), &data); err != nil {
		return err
	}

	_, ok := data["channelId"]
	if !ok {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Channel Id is required",
		})
	}

	var channel models.Channel
	if err := repository.DB.First(&channel, data["channelId"]).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Channel not found",
			})
		}
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Failed to retrieve channel",
		})
	}

	return c.JSON(channel)
}

func SubscribeChannel(c fiber.Ctx) error {
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

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid user Id",
		})
	}

	if data["channelId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Channel Id is required",
		})
	}

	channelId, err := strconv.ParseUint(data["channelId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid channel Id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, uint(channelId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	var subscription models.Subscription
	if result := repository.DB.Where("user_id = ? AND channel_id = ?", userId, channelId).First(&subscription); result.Error == nil {
		c.Status(fiber.StatusConflict)
		return c.JSON(fiber.Map{
			"message": "Already subscribed",
		})
	}

	subscription = models.Subscription{
		UserId:    uint(userId),
		ChannelId: uint(channelId),
	}

	repository.DB.Create(&subscription)

	repository.DB.Model(&channel).Update("subs_count", channel.SubsCount+1)

	return c.JSON(fiber.Map{
		"message": "Subscription successful",
	})

}

func UnsubscribeChannel(c fiber.Ctx) error {
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

	userId, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Invalid user Id",
		})
	}

	if data["channelId"] == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Channel Id is required",
		})
	}

	channelId, err := strconv.ParseUint(data["channelId"], 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid channel Id",
		})
	}

	var channel models.Channel
	if result := repository.DB.First(&channel, uint(channelId)); result.Error != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Channel not found",
		})
	}

	result := repository.DB.Where("user_id = ? AND channel_id = ?", userId, channelId).Delete(&models.Subscription{})
	if result.RowsAffected == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Not subscribed",
		})
	}

	if channel.SubsCount > 0 {
		repository.DB.Model(&channel).Update("subs_count", channel.SubsCount-1)
	}

	return c.JSON(fiber.Map{
		"message": "Unsubscribed successfully",
	})
}
