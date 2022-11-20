package models

import "gorm.io/gorm"

type Config struct {
	gorm.Model
	GuildId                     string
	WelcomeMessage              string
	WelcomeMessageAttachmentURL string
	WelcomeChannelId            string
	PinsChannelId               string
}
