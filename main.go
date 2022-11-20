package main

import (
	"flag"
	kodachiEvents "kodachi/bot/events"
	"kodachi/bot/handlers"
	"kodachi/bot/models"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	REGISTER_COMMANDS = flag.Bool("register-commands", true, "True by default (useful in development)")
	TESTING           = flag.Bool("testing", false, "")
)

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

	if !db.Migrator().HasTable(&models.Config{}) {
		db.Migrator().CreateTable(&models.Config{})
	}

	if !db.Migrator().HasColumn(&models.Config{}, "welcome_channel_id") {
		db.Migrator().AddColumn(&models.Config{}, "welcome_channel_id")
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

func init() {
	s.AddHandler(handlers.InteractionCreateHandler(db))
}

func main() {
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	s.AddHandler(kodachiEvents.WelcomeMessageEventHandler(db))

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
