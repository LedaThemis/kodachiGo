package commands

import "github.com/bwmarrin/discordgo"

var noDM = false
var configPermission int64 = discordgo.PermissionAdministrator
var Commands = []*discordgo.ApplicationCommand{
	&welcomeCommand,
	&configCommand,
	&birthdayCommand,
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

var (
	minMonth float64 = 1
	maxMonth float64 = 12
	minDay   float64 = 1
	maxDay   float64 = 31
)

var birthdayCommand = discordgo.ApplicationCommand{
	Name:        "birthday",
	Description: "Various commands relating to birthdays",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "add",
			Description: "Add birthday entry",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_id",
					Description: "ID of user",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of user",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "month",
					Description: "Birth month",
					Required:    true,
					MinValue:    &minMonth,
					MaxValue:    maxMonth,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "day",
					Description: "Birth day",
					Required:    true,
					MinValue:    &minDay,
					MaxValue:    maxDay,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "update",
			Description: "Update birthday entry",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_id",
					Description: "ID of user to update",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "New name of user",
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "month",
					Description: "New birth month",
					MinValue:    &minMonth,
					MaxValue:    maxMonth,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "day",
					Description: "New birth day",
					MinValue:    &minDay,
					MaxValue:    maxDay,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "delete",
			Description: "Delete birthday entry",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_id",
					Description: "ID of user",
					Required:    true,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "list",
			Description: "List birthday entries",
		},
	},
}
