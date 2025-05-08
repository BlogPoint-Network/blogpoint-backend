package utils

import (
	"log"
	"time"

	"blogpoint-backend/internal/models"
	"blogpoint-backend/internal/repository"
)

func StartCleanupTask() {
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			log.Println("Запуск фона очистки старых данных...")

			// Удаляем коды, которые не подтверждены
			verificationCodeResult := repository.DB.
				Where("created_at < ?", time.Now().Add(-1*time.Hour)).
				Delete(&models.VerificationCode{})
			log.Printf("Удалено %d старых кодов подтверждения", verificationCodeResult.RowsAffected)

			// Удаляем неподтверждённые аккаунты
			userResult := repository.DB.
				Where("is_verified = ? AND created_at < ?", false, time.Now().Add(-24*time.Hour)).
				Delete(&models.User{})
			log.Printf("🧹 Удалено %d неподтверждённых аккаунтов", userResult.RowsAffected)
		}
	}()
}

func StartStatisticsTask() {
	go func() {
		for {
			log.Println("📊 Обновление статистики каналов...")

			today := time.Now().Truncate(24 * time.Hour)
			repository.DB.Where("date = ?", today).Delete(&models.ChannelStatistics{})

			var channels []models.Channel
			if err := repository.DB.Find(&channels).Error; err != nil {
				log.Printf("❌ Ошибка при получении каналов: %v", err)
				continue
			}

			for _, ch := range channels {
				stats := models.ChannelStatistics{
					ChannelId: ch.Id,
					Date:      today,
				}

				repository.DB.
					Model(&models.Post{}).
					Where("channel_id = ?", ch.Id).
					Select("COALESCE(SUM(likes_count), 0), COALESCE(SUM(dislikes_count), 0), COALESCE(SUM(views_count), 0), COUNT(*)").
					Row().
					Scan(&stats.Likes, &stats.Dislikes, &stats.Views, &stats.Posts)

				var commentsCount int64
				repository.DB.
					Model(&models.Comment{}).
					Joins("JOIN posts ON comments.post_id = posts.id").
					Where("posts.channel_id = ?", ch.Id).
					Count(&commentsCount)

				stats.Comments = int(commentsCount)

				if err := repository.DB.Create(&stats).Error; err != nil {
					log.Printf("❌ Ошибка при создании статистики для канала %d: %v", ch.Id, err)
				} else {
					log.Printf("✅ Статистика обновлена для канала %d", ch.Id)
				}
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}
