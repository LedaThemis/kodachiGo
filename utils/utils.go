package utils

import (
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// Convert month days to contain ordinal indicators
func Ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		suffix = "st"
	case 2:
		suffix = "nd"
	case 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%v%s", n, suffix)
}

func MessageURL(guildId, channelId, messageId string) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildId, channelId, messageId)
}

func AttachmentToFile(attachment discordgo.MessageAttachment) (discordgo.File, error) {
	resp, err := http.Get(attachment.ProxyURL)

	if err != nil {
		return discordgo.File{}, err
	}

	return discordgo.File{
		ContentType: attachment.ContentType,
		Name:        attachment.Filename,
		Reader:      resp.Body,
	}, nil
}

func AttachmentsToFile(attachments []*discordgo.MessageAttachment) ([]*discordgo.File, error) {
	files := make([]*discordgo.File, len(attachments))

	for i, attachment := range attachments {
		file, err := AttachmentToFile(*attachment)

		if err != nil {
			return []*discordgo.File{}, err
		}

		files[i] = &file
	}

	return files, nil
}
