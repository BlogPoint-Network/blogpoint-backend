package utils

import (
	"fmt"
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
			fmt.Printf("Удалено %d старых кодов подтверждения", verificationCodeResult.RowsAffected)

			// Удаляем неподтверждённые аккаунты
			userResult := repository.DB.
				Where("is_verified = ? AND created_at < ?", false, time.Now().Add(-24*time.Hour)).
				Delete(&models.User{})
			log.Printf("🧹 Удалено %d неподтверждённых аккаунтов", userResult.RowsAffected)
		}
	}()
}
