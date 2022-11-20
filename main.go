package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	gorm.Model
	GuildId                     string
	WelcomeMessage              string
	WelcomeMessageAttachmentURL string
	WelcomeChannelId            string
	PinsChannelId               string
}

var (
	REGISTER_COMMANDS = flag.Bool("register-commands", true, "True by default (useful in development)")
	TESTING           = flag.Bool("testing", false, "")
)

var genericErrorResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "An unknown error occurred, please try again.",
	},
}

var s *discordgo.Session
var db *gorm.DB

func init() { flag.Parse() }

func init() {
	// Load .env only if --testing=true
	if *TESTING {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// Load BotToken
	BotToken := os.Getenv("BOT_TOKEN")

	var err error
	s, err = discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
	s.Identify.Intents |= discordgo.IntentGuildMembers
}

func init() {
	var err error
	db, err = gorm.Open(postgres.Open(os.Getenv("POSTGRES_DSN")), &gorm.Config{})

	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	if !db.Migrator().HasTable(&Config{}) {
		db.Migrator().CreateTable(&Config{})
	}

	if !db.Migrator().HasColumn(&Config{}, "welcome_channel_id") {
		db.Migrator().AddColumn(&Config{}, "welcome_channel_id")
	}
}

var noDM = false
var configPermission int64 = discordgo.PermissionAdministrator
var commands = []*discordgo.ApplicationCommand{
	{
		Name:         "welcome",
		Description:  "Various commands related to welcome",
		DMPermission: &noDM,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "test",
				Description: "Test welcome message",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "User to welcome",
						Required:    false,
					},
				},
			},
		},
	},
	{
		Name:                     "config",
		Description:              "Various commands related to configuration",
		DMPermission:             &noDM,
		DefaultMemberPermissions: &configPermission,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "list",
				Description: "Lists available config options with their current values",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Name:        "set",
				Description: "Updates config with provided values",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Name:        "welcome_message",
						Description: "Set Welcome Message",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "message",
								Description: "New welcome message",
								Required:    true,
							},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Name:        "welcome_message_attachment",
						Description: "Set Welcome Message Attachment",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionString,
								Name:        "attachment_url",
								Description: "New attachment url",
								Required:    true,
							},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Name:        "pins_channel_id",
						Description: "Set Pins Channel ID",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionChannel,
								Name:        "channel",
								Description: "New pins channel",
								Required:    true,
							},
						},
					},
					{
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Name:        "welcome_channel_id",
						Description: "Set Welcome Channel ID",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Type:        discordgo.ApplicationCommandOptionChannel,
								Name:        "channel",
								Description: "New welcome channel",
								Required:    true,
							},
						},
					},
				},
			},
		},
	},
}

type CommandHandler = func(s *discordgo.Session, i *discordgo.InteractionCreate)

var commandHandlers = map[string]CommandHandler{
	"welcome": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

			var guildConfig = Config{}

			result := db.Where(&Config{GuildId: i.GuildID}).First(&guildConfig)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, genericErrorResponse)

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
	},
	"config": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		options := i.ApplicationCommandData().Options

		switch options[0].Name {
		case "list":
			var config = Config{GuildId: i.GuildID}

			result := db.Where(&config).FirstOrCreate(&config)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, genericErrorResponse)

			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Configuration for %s:\n\nWelcome message: %s\nWelcome message attachment: <%s>\nPins channel: %s", i.GuildID, config.WelcomeMessage, config.WelcomeMessageAttachmentURL, config.PinsChannelId),
					},
				})
			}
		case "set":
			var newGuildConfig = Config{GuildId: i.GuildID}

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

			result := db.Model(&Config{}).Where(&Config{GuildId: i.GuildID}).Updates(&newGuildConfig)

			switch {
			case result.Error != nil:
				log.Print(result.Error)
				s.InteractionRespond(i.Interaction, genericErrorResponse)

			default:
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Successfully updated config!",
					},
				})
			}

		}
	},
}

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// If command handler exists
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			// Call with session and interaction
			handler(s, i)
		}
	})
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	s.AddHandler(func(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
		var guildConfig = Config{}

		result := db.Where(&Config{GuildId: e.GuildID}).First(&guildConfig)

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
	})

	err := s.Open()

	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	guildId := "" // Empty to register global commands
	if *REGISTER_COMMANDS {
		log.Println("Adding commands...")

		for i, command := range commands {

			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildId, command)

			if err != nil {
				log.Panicf("Cannot create '%v' command: %v", command.Name, err)
			}

			registeredCommands[i] = cmd
		}
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	CLEAN_COMMANDS_AFTER_SHUTDOWN := os.Getenv("CLEAN_COMMANDS_AFTER_SHUTDOWN")

	if CLEAN_COMMANDS_AFTER_SHUTDOWN == "true" {
		log.Println("Removing commands...")

		for _, command := range registeredCommands {
			err := s.ApplicationCommandDelete(s.State.User.ID, guildId, command.ID)

			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", command.Name, err)
			}

		}
	}

	log.Println("Gracefully shutting down.")
}
