package responses

import "github.com/bwmarrin/discordgo"

var GenericErrorResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "An unknown error occurred, please try again.",
	},
}

var NoPinsChannelConfigured = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "This guild does not have a pins channel configured.",
	},
}
