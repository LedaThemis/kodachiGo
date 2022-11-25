package utils

import (
	"fmt"
	"kodachi/packages/trees"
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

// Returns true if (m1/d1) is earlier than (m2/d2)
func CompareDates(m1, d1, m2, d2 int64) bool {
	return m1 < m2 || (m1 == m2 && d1 < d2)
}

// Returns true if (m1/d1) and (m2/d2) are the same
func SameDates(m1, d1, m2, d2 int64) bool {
	return m1 == m2 && d1 == d2
}

// Returns TreeNode from list of {"ParentId": Names of children}
func ConstructTreeNode(m map[string][]string, rootParent string) trees.TreeNode {
	root := trees.TreeNode{
		Name:     rootParent,
		Children: []*trees.TreeNode{},
	}

	if _, ok := m[rootParent]; !ok {
		return root
	}

	childNodes := []*trees.TreeNode{}

	for _, childName := range m[rootParent] {
		node := ConstructTreeNode(m, childName)

		childNodes = append(childNodes, &node)
	}

	root.Children = childNodes

	return root
}
