package handlers

import (
	"fmt"
	"kodachi/bot/models"
	"kodachi/bot/responses"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func InteractionCreateHandler(db *gorm.DB) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var commandHandlers = map[string]CommandHandler{
		"welcome": welcomeCommandHandler(db),
		"config":  configCommandHandler(db),
	}

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// If command handler exists
		if commandHandler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			// Call with session and interaction
			commandHandler(s, i)
		}
	}
}

type CommandHandler = func(s *discordgo.Session, i *discordgo.InteractionCreate)

func welcomeCommandHandler(db *gorm.DB) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options

		switch options[0].Name {
		case "test":
			testCommandOptions := options[0].Options

			testCommandOptionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(testCommandOptions))
			for _, opt := range testCommandOptions {
				testCommandOptionMap[opt.Name] = opt
			}

			userId := i.Member.User.ID

			if option, ok := testCommandOptionMap["user"]; ok {
				userId = option.UserValue(nil).ID
			}

			var guildConfig = models.Config{}

			result := db.Where(&models.Config{GuildId: i.GuildID}).First(&guildConfig)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

			default:
				messageContent := strings.ReplaceAll(guildConfig.WelcomeMessage, "<@USER_ID>", fmt.Sprintf("<@%s>", userId))

				validAttachmentURL, err := url.ParseRequestURI(guildConfig.WelcomeMessageAttachmentURL)
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Attachment url is invalid.",
						},
					})
					return
				}

				resp, err := http.Get(validAttachmentURL.String())

				if err != nil {
					log.Printf("An error occurred while fetching image: %v", err)
				}

				defer resp.Body.Close()

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: messageContent,
						Files: []*discordgo.File{
							{
								ContentType: resp.Header.Get("Content-Type"),
								Name:        "welcome.png",
								Reader:      resp.Body,
							},
						},
					},
				})
			}
		}
	}
}

func configCommandHandler(db *gorm.DB) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options

		switch options[0].Name {
		case "list":
			var config = models.Config{GuildId: i.GuildID}

			result := db.Where(&config).FirstOrCreate(&config)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Configuration for %s:\n\nWelcome message: %s\nWelcome message attachment: <%s>\nPins channel: %s\nWelcome channel: %s", i.GuildID, config.WelcomeMessage, config.WelcomeMessageAttachmentURL, config.PinsChannelId, config.WelcomeChannelId),
					},
				})
			}
		case "set":
			var newGuildConfig = models.Config{GuildId: i.GuildID}

			subCommandOptions := options[0].Options
			subSubCommandOptions := subCommandOptions[0].Options

			subSubCommandOptionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(subSubCommandOptions))
			for _, opt := range subSubCommandOptions {
				subSubCommandOptionMap[opt.Name] = opt
			}

			switch subCommandOptions[0].Name {

			case "welcome_message":
				newGuildConfig.WelcomeMessage = subSubCommandOptionMap["message"].StringValue()
			case "welcome_message_attachment":
				validAttachmentURL, err := url.ParseRequestURI(subSubCommandOptionMap["attachment_url"].StringValue())
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Please provide a valid url.",
						},
					})
					return
				}

				newGuildConfig.WelcomeMessageAttachmentURL = validAttachmentURL.String()
			case "pins_channel_id":
				newGuildConfig.PinsChannelId = subSubCommandOptionMap["channel"].ChannelValue(s).ID
			case "welcome_channel_id":
				newGuildConfig.WelcomeChannelId = subSubCommandOptionMap["channel"].ChannelValue(s).ID
			}

			result := db.Model(&models.Config{}).Where(&models.Config{GuildId: i.GuildID}).Updates(&newGuildConfig)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Successfully updated config!",
					},
				})
			}

		}
	}
}
