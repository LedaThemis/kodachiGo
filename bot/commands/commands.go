package commands

import "github.com/bwmarrin/discordgo"

var noDM = false
var configPermission int64 = discordgo.PermissionAdministrator
var Commands = []*discordgo.ApplicationCommand{
	&welcomeCommand,
	&configCommand,
}

var welcomeCommand = discordgo.ApplicationCommand{
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
}

var configCommand = discordgo.ApplicationCommand{
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
}
