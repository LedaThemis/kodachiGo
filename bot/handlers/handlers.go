package handlers

import (
	"errors"
	"fmt"
	"kodachi/bot/models"
	"kodachi/bot/responses"
	"kodachi/utils"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func InteractionCreateHandler(db *gorm.DB) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var commandHandlers = map[string]CommandHandler{
		"welcome":     welcomeCommandHandler(db),
		"config":      configCommandHandler(db),
		"birthday":    birthdayCommandHandler(db),
		"Pin Message": pinCommandHandler(db),
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

func birthdayCommandHandler(db *gorm.DB) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options

		var subCommand = options[0]
		var subCommandOptions = subCommand.Options

		subCommandOptionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(subCommandOptions))
		for _, opt := range subCommandOptions {
			subCommandOptionMap[opt.Name] = opt
		}

		switch options[0].Name {
		case "add":
			birthdayUserId := subCommandOptionMap["user_id"].StringValue()

			var authorId string

			if i.User != nil {
				authorId = i.User.ID
			} else {
				authorId = i.Member.User.ID
			}

			userBirthday := models.Birthday{AuthorId: authorId, UserId: birthdayUserId}

			result := db.Where(&userBirthday).First(&userBirthday)

			switch {
			// Birthday does not exist, create it
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				userBirthday.Name = subCommandOptionMap["name"].StringValue()
				userBirthday.BirthDay = subCommandOptionMap["day"].IntValue()
				userBirthday.BirthMonth = subCommandOptionMap["month"].IntValue()

				result := db.Create(&userBirthday)

				switch {
				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully added birthday entry.",
						},
					})
				}

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Birthday exists
			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You've already added a birthday for this user.",
					},
				})
			}

		case "update":
			birthdayUserId := subCommandOptionMap["user_id"].StringValue()

			var birthdayUpdate models.Birthday

			if subCommandOptionMap["name"] != nil {
				birthdayUpdate.Name = subCommandOptionMap["name"].StringValue()
			}

			if subCommandOptionMap["day"] != nil {
				birthdayUpdate.BirthDay = subCommandOptionMap["day"].IntValue()
			}

			if subCommandOptionMap["month"] != nil {
				birthdayUpdate.BirthMonth = subCommandOptionMap["month"].IntValue()
			}

			var authorId string

			if i.User != nil {
				authorId = i.User.ID
			} else {
				authorId = i.Member.User.ID
			}

			userBirthday := models.Birthday{AuthorId: authorId, UserId: birthdayUserId}

			result := db.Where(&userBirthday).First(&models.Birthday{})

			switch {
			// Birthday does not exist, inform user
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Birthday entry does not exist.",
					},
				})

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Birthday updated
			default:

				result := db.Where(&userBirthday).Updates(&birthdayUpdate)

				switch {

				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully updated birthday entry.",
						},
					})
				}
			}

		case "delete":
			birthdayUserId := subCommandOptionMap["user_id"].StringValue()

			var authorId string

			if i.User != nil {
				authorId = i.User.ID
			} else {
				authorId = i.Member.User.ID
			}

			userBirthday := models.Birthday{AuthorId: authorId, UserId: birthdayUserId}

			result := db.Where(&userBirthday).First(&userBirthday)

			switch {
			// Birthday does not exist, inform user
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Birthday entry does not exist.",
					},
				})

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Birthday exists, delete it
			default:
				result := db.Where(&userBirthday).Delete(&userBirthday)

				switch {
				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully deleted birthday entry.",
						},
					})
				}
			}
		case "list":
			var author *discordgo.User

			if i.User != nil {
				author = i.User
			} else {
				author = i.Member.User
			}

			var authorId = author.ID

			userBirthdays := []models.Birthday{}

			result := db.Where(&models.Birthday{AuthorId: authorId}).Find(&userBirthdays)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			default:
				var birthdaysListMessage string

				birthdaysListMessage += fmt.Sprintf("**List of Birthdays**\nAuthor: %v#%v\n\n", author.Username, author.Discriminator)

				if len(userBirthdays) == 0 {
					birthdaysListMessage += fmt.Sprintf("\n%v has not added any birthdays yet.", author.Username)
				}

				for i, birthday := range userBirthdays {
					birthdaysListMessage += fmt.Sprintf("%v. %s born %s of %s\n- Mention: <@%s> | Discord ID: %s\n\n", i+1, birthday.Name, utils.Ordinal(int(birthday.BirthDay)), time.Month(birthday.BirthMonth), birthday.UserId, birthday.UserId)
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: birthdaysListMessage,
					},
				})
			}
		}
	}
}

func pinCommandHandler(db *gorm.DB) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		messageId := i.ApplicationCommandData().TargetID

		message, err := s.ChannelMessage(i.ChannelID, messageId)

		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "There was an error fetching message to be pinned.",
				},
			})
			return
		}

		var config = models.Config{GuildId: i.GuildID}

		result := db.Where(&config).First(&config)

		switch {
		// Pins channel is not configured, inform user
		case errors.Is(result.Error, gorm.ErrRecordNotFound) || config.PinsChannelId == "":
			s.InteractionRespond(i.Interaction, responses.NoPinsChannelConfigured)
		case result.Error != nil:
			s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

		// Pins channel is configured, send message to it
		default:
			var usableWebhook *discordgo.Webhook
			author := i.Member.User

			_, err := s.Channel(config.PinsChannelId)

			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Configured pins channel does not exist.",
					},
				})
				return
			}

			webhooks, err := s.ChannelWebhooks(config.PinsChannelId)

			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to get guild webhooks, check bot permissions.",
					},
				})
				return
			}

			exists := false

			for _, webhook := range webhooks {
				if webhook.ApplicationID == s.State.User.ID {
					exists = true

					usableWebhook = webhook
				}
			}

			if !exists {
				// Create a webhook
				usableWebhook, err = s.WebhookCreate(config.PinsChannelId, fmt.Sprintf("Pins [%s]", s.State.User.Username), "")

				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Failed to create a webhook, please create one yourself or check bot permissions.",
						},
					})
					return
				}
			}

			wait := false

			buttonRow := discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "Jump",
						Style: discordgo.LinkButton,
						URL:   utils.MessageURL(i.GuildID, message.ChannelID, message.ID),
					},
				},
			}

			messageFiles, err := utils.AttachmentsToFile(message.Attachments)

			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to download message attachment, please try again.",
					},
				})
				return
			}

			_, err = s.WebhookExecute(usableWebhook.ID, usableWebhook.Token, wait, &discordgo.WebhookParams{
				Username:   author.Username,
				AvatarURL:  author.AvatarURL(""),
				Content:    message.Content,
				Embeds:     message.Embeds,
				TTS:        message.TTS,
				Files:      messageFiles,
				Components: append(message.Components, buttonRow),
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{},
				},
			})

			if err != nil {
				log.Printf("An error occurred while sending pin message: %v", err)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Failed to send the pinned message, please try again.",
					},
				})
			} else {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("<@%s> pinned a message from this channel. See all pinned messages <#%s>", author.ID, config.PinsChannelId),
					},
				})
			}
		}
	}
}
