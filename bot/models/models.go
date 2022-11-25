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

type Birthday struct {
	gorm.Model
	UserId     string
	Name       string
	BirthDay   int64
	BirthMonth int64
	AuthorId   string // User that added birthday entry
}

type TreeMember struct {
	gorm.Model
	UserId   string
	Name     string
	ParentId string // "" if no parent
	GuildId  string
}
