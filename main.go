package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var s *discordgo.Session

func init() {
	if os.Getenv("PRODUCTION") == "false" {
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
}

var dmPermission = false
var commands = []*discordgo.ApplicationCommand{
	{
		Name:         "welcome",
		Description:  "Various commands related to welcome",
		DMPermission: &dmPermission,
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

			userId := i.Interaction.Member.User.ID

			if option, ok := testCommandOptionMap["user"]; ok {
				userId = option.UserValue(nil).ID
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Welcome <@%s>!", userId),
				},
			})
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

	err := s.Open()

	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	guildId := "" // Empty to register global commands
	for i, command := range commands {

		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildId, command)

		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", command.Name, err)
		}

		registeredCommands[i] = cmd
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
