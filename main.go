package main

import (
	"flag"
	"kodachi/bot/commands"
	kodachiEvents "kodachi/bot/events"
	"kodachi/bot/handlers"
	"kodachi/bot/models"
	kodachiTasks "kodachi/bot/tasks"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-co-op/gocron"
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

// Parse CLI Arguments
func init() { flag.Parse() }

// Initiate discord session
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
	s.Identify.Intents |= discordgo.IntentGuildWebhooks
	s.Identify.Intents |= discordgo.IntentMessageContent
}

// Initiate database connection
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

	if !db.Migrator().HasTable(&models.Birthday{}) {
		db.Migrator().CreateTable(&models.Birthday{})
	}
}

// Add Handlers
func init() {
	s.AddHandler(handlers.InteractionCreateHandler(db))

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	s.AddHandler(kodachiEvents.WelcomeMessageEventHandler(db))
}

// Create websocket connection to Discord
func init() {
	err := s.Open()

	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
}

// Add Tasks
func init() {
	scheduler := gocron.NewScheduler(time.UTC)

	scheduler.Every(1).Day().At("00:00").Do(kodachiTasks.BirthdayCheck(db, s))

	scheduler.StartAsync()
}

func main() {
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands.Commands))
	guildId := "" // Empty to register global commands
	if *REGISTER_COMMANDS {
		log.Println("Adding commands...")

		for i, command := range commands.Commands {

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
