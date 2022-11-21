package tasks

import (
	"fmt"
	"kodachi/bot/models"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func BirthdayCheck(db *gorm.DB, s *discordgo.Session) func() {
	return func() {
		current := time.Now().UTC()

		currentMonth := int64(current.Month())
		currentDay := int64(current.Day())

		userBirthdays := []models.Birthday{}

		result := db.Where(&models.Birthday{BirthMonth: currentMonth, BirthDay: currentDay}).Find(&userBirthdays)

		switch {
		case result.Error != nil:
			log.Printf("An error ocurred while querying for birthdays: %v", result.Error)
		default:
			for _, birthday := range userBirthdays {
				ch, err := s.UserChannelCreate(birthday.AuthorId)

				if err != nil {
					log.Printf("Could not initiate DMs with %v: %v", birthday.AuthorId, err)
				} else {
					s.ChannelMessageSend(ch.ID, fmt.Sprintf("Friendly Reminder: Today, %s (<@%s>, %s) was born!\n\nIt's their birthday 🎉🥳", birthday.Name, birthday.UserId, birthday.UserId))
				}

				time.Sleep(5 * time.Second)
			}
		}
	}
}
