package controllers

import (
	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
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
			"message": "Invalid issuer ID",
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
