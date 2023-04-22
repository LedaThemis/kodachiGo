package handlers

import (
	"errors"
	"fmt"
	"kodachi/bot/models"
	"kodachi/bot/responses"
	"kodachi/packages/trees"
	"kodachi/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
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
		"tree":        treeCommandHandler(db),
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

				sort.SliceStable(userBirthdays, func(i, j int) bool {
					return utils.CompareDates(userBirthdays[i].BirthMonth, userBirthdays[i].BirthDay, userBirthdays[j].BirthMonth, userBirthdays[j].BirthDay)
				})

				current := time.Now().UTC()
				currentMonth := int64(current.Month())
				currentDay := int64(current.Day())

				last := userBirthdays[len(userBirthdays)-1]
				lastEarlierThanCurrent := utils.CompareDates(last.BirthMonth, last.BirthDay, currentMonth, currentDay)

				addedNextBirthdayMarker := false

				for i, birthday := range userBirthdays {
					birthdayToday := utils.SameDates(currentMonth, currentDay, birthday.BirthMonth, birthday.BirthDay)
					laterThanCurrent := utils.CompareDates(currentMonth, currentDay, birthday.BirthMonth, birthday.BirthDay)

					if birthdayToday {
						birthdaysListMessage += fmt.Sprintf("**Happy Birthday %v!!** 🎉🥳\n", birthday.Name)
					}

					// If last is earlier than current, first is the next birthday, otherwise any date later than current is next.
					if !addedNextBirthdayMarker {
						if lastEarlierThanCurrent && i == 0 || laterThanCurrent {
							birthdaysListMessage += "**Next birthday** ⬇️\n"
							addedNextBirthdayMarker = true
						}
					}
					birthdaysListMessage += fmt.Sprintf("%v. %s born %s of %s\n- Mention: <@%s> | Discord ID: %s\n\n", i+1, birthday.Name, utils.Ordinal(int(birthday.BirthDay)), time.Month(birthday.BirthMonth), birthday.UserId, birthday.UserId)
				}

				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Description: birthdaysListMessage,
							},
						},
					},
				})

				if err != nil {
					log.Print(err)
					s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
						Content: responses.GenericErrorResponse.Data.Content,
					})
				}
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

func treeCommandHandler(db *gorm.DB) CommandHandler {
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
			treeUserId := subCommandOptionMap["user"].UserValue(nil).ID
			treeUserName := subCommandOptionMap["name"].StringValue()

			parentUserId := ""

			if subCommandOptionMap["parent"] != nil {
				parentUserId = subCommandOptionMap["parent"].UserValue(nil).ID
			}

			treeMember := models.TreeMember{UserId: treeUserId, GuildId: i.GuildID}

			result := db.Where(&treeMember).First(&treeMember)

			switch {
			// Tree member does not exist, create it
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				treeMember.Name = treeUserName
				treeMember.ParentId = parentUserId

				result := db.Create(&treeMember)

				switch {
				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully added tree member.",
						},
					})
				}

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Tree member exists
			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You've already added this user to the tree.",
					},
				})
			}
		case "update":
			treeUserId := subCommandOptionMap["user"].UserValue(nil).ID

			parentUserId := ""

			if subCommandOptionMap["parent"] != nil {
				parentUserId = subCommandOptionMap["parent"].UserValue(nil).ID
			}

			var treeMemberUpdate models.TreeMember

			if subCommandOptionMap["name"] != nil {
				treeMemberUpdate.Name = subCommandOptionMap["name"].StringValue()
			}

			if subCommandOptionMap["parent"] != nil {
				treeMemberUpdate.ParentId = parentUserId
			}

			treeMember := models.TreeMember{UserId: treeUserId, GuildId: i.GuildID}

			result := db.Where(&treeMember).First(&models.TreeMember{})

			switch {
			// Tree member does not exist, inform user
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "User is not added to the tree.",
					},
				})

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Tree member updated
			default:
				result := db.Where(&treeMember).Updates(&treeMemberUpdate)

				switch {
				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)

				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully updated tree member.",
						},
					})
				}
			}
		case "delete":
			treeUserId := subCommandOptionMap["user_id"].StringValue()

			treeMember := models.TreeMember{UserId: treeUserId, GuildId: i.GuildID}

			result := db.Where(&treeMember).First(&treeMember)

			switch {
			// Tree member does not exist, inform user
			case errors.Is(result.Error, gorm.ErrRecordNotFound):
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "User is not added to the tree.",
					},
				})

			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			// Tree member exists, delete it
			default:
				result := db.Where(&treeMember).Delete(&treeMember)

				switch {
				case result.Error != nil:
					log.Print(result.Error)
					s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
				default:
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Successfully deleted user from tree.",
						},
					})
				}
			}
		case "view":

			treeMembers := []models.TreeMember{}

			result := db.Where(&models.TreeMember{GuildId: i.GuildID}).Find(&treeMembers)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, responses.GenericErrorResponse)
			default:
				if len(treeMembers) == 0 {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "There are no users added to the tree.",
						},
					})
					return
				}

				treeMembersIdMap := make(map[string]string)

				for _, member := range treeMembers {
					treeMembersIdMap[member.UserId] = member.Name
				}

				treeMembersMap := make(map[string][]string)

				for _, member := range treeMembers {
					parentName := ""

					if name, ok := treeMembersIdMap[member.ParentId]; ok || member.ParentId == "" {
						parentName = name
					} else {
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: fmt.Sprintf("There is a member whose parent is not in the tree.\n\n Member: %s (<@%s>), Parent: <@%s>", member.Name, member.UserId, member.ParentId),
							},
						})
						return
					}

					if children, ok := treeMembersMap[parentName]; ok {
						treeMembersMap[parentName] = append(children, member.Name)
					} else {
						treeMembersMap[parentName] = []string{member.Name}
					}
				}

				if len(treeMembersMap[""]) > 1 {
					membersWithNoParents := strings.Join(treeMembersMap[""], ", ")

					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("There are multiple origins (users with no parents), which is not supported.\n\nList of members: %s", membersWithNoParents),
						},
					})
					return
				}

				tree := utils.ConstructTreeNode(treeMembersMap, treeMembersMap[""][0])

				trees.DrawTree(&tree, 75.0, 75.0, 5.0, 50.0, 50.0, 50.0, 50.0, 50.0, "out.png")

				file, err := os.Open("out.png")

				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "An error occurred while rendering image.",
						},
					})
				} else {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "",
							Files: []*discordgo.File{
								{
									Name:        "tree.png",
									ContentType: "image/png",
									Reader:      file,
								},
							},
						},
					})

					file.Close()
				}

			}

		}
	}
}
