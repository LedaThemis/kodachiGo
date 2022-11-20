package events

import (
	"errors"
	"fmt"
	"kodachi/bot/models"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func WelcomeMessageEventHandler(db *gorm.DB) func(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	return func(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
		var guildConfig = models.Config{}

		result := db.Where(&models.Config{GuildId: e.GuildID}).First(&guildConfig)

		switch {
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			log.Println("Server is not configured.")
		case result.Error != nil:
			log.Print(result.Error)
		default:
			if guildConfig.WelcomeChannelId == "" {
				log.Printf("Welcome channel is not configured")
				return
			}

			if guildConfig.WelcomeMessage == "" {
				log.Printf("Welcome message is not configured")
				return
			}

			messageContent := strings.ReplaceAll(guildConfig.WelcomeMessage, "<@USER_ID>", fmt.Sprintf("<@%s>", e.User.ID))

			var messageFiles []*discordgo.File

			if guildConfig.WelcomeMessageAttachmentURL != "" {
				validAttachmentURL, err := url.ParseRequestURI(guildConfig.WelcomeMessageAttachmentURL)
				if err != nil {
					s.ChannelMessageSendComplex(guildConfig.WelcomeChannelId, &discordgo.MessageSend{
						Content: "Attachment url is invalid.",
					})
					return
				}

				resp, err := http.Get(validAttachmentURL.String())

				if err != nil {
					log.Printf("An error occurred while fetching image: %v", err)
				}

				messageFiles = []*discordgo.File{
					{
						ContentType: resp.Header.Get("Content-Type"),
						Name:        "welcome.png",
						Reader:      resp.Body,
					},
				}

				defer resp.Body.Close()
			}

			_, err := s.ChannelMessageSendComplex(guildConfig.WelcomeChannelId, &discordgo.MessageSend{
				Content: messageContent,
				Files:   messageFiles,
			})

			if err != nil {
				log.Printf("Failed to send welcome message in %v: %v", e.GuildID, err)
			}
		}
	}
}
