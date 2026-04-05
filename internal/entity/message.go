package entity

import "encoding/json"

type TextContent struct {
	Text string `json:"text"`
}

type ImageContent struct {
	Url string `json:"url"`
}

type VideoContent struct {
	Url string `json:"url"`
}

type AudioContent struct {
	Url string `json:"url"`
}

type FileContent struct {
	Url  string `json:"url"`
	Name string `json:"name,omitempty"`
}

// MessageContent is the internal typed content payload stored in JSON.
type MessageContent struct {
	Text   *TextContent    `json:"text,omitempty"`
	Image  *ImageContent   `json:"image,omitempty"`
	Video  *VideoContent   `json:"video,omitempty"`
	Audio  *AudioContent   `json:"audio,omitempty"`
	File   *FileContent    `json:"file,omitempty"`
	Custom json.RawMessage `json:"custom,omitempty"`
}

// FlatMessageContent keeps the external API shape stable.
type FlatMessageContent struct {
	Text   string `json:"text,omitempty"`
	Image  string `json:"image,omitempty"`
	Video  string `json:"video,omitempty"`
	Audio  string `json:"audio,omitempty"`
	File   string `json:"file,omitempty"`
	Custom string `json:"custom,omitempty"`
}

func NewMessageContentFromFlat(c FlatMessageContent) MessageContent {
	content := MessageContent{}
	if c.Text != "" {
		content.Text = &TextContent{Text: c.Text}
	}
	if c.Image != "" {
		content.Image = &ImageContent{Url: c.Image}
	}
	if c.Video != "" {
		content.Video = &VideoContent{Url: c.Video}
	}
	if c.Audio != "" {
		content.Audio = &AudioContent{Url: c.Audio}
	}
	if c.File != "" {
		content.File = &FileContent{Url: c.File}
	}
	if c.Custom != "" {
		content.Custom = json.RawMessage(c.Custom)
	}
	return content
}

func (c MessageContent) ToFlat() FlatMessageContent {
	flat := FlatMessageContent{}
	if c.Text != nil {
		flat.Text = c.Text.Text
	}
	if c.Image != nil {
		flat.Image = c.Image.Url
	}
	if c.Video != nil {
		flat.Video = c.Video.Url
	}
	if c.Audio != nil {
		flat.Audio = c.Audio.Url
	}
	if c.File != nil {
		flat.File = c.File.Url
	}
	if len(c.Custom) > 0 {
		flat.Custom = string(c.Custom)
	}
	return flat
}

func (c MessageContent) PayloadCount() int {
	count := 0
	if c.Text != nil {
		count++
	}
	if c.Image != nil {
		count++
	}
	if c.Video != nil {
		count++
	}
	if c.Audio != nil {
		count++
	}
	if c.File != nil {
		count++
	}
	if len(c.Custom) > 0 {
		count++
	}
	return count
}

// Message represents a message
type Message struct {
	Id             int64          `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ConversationId string         `json:"conversation_id" gorm:"column:conversation_id"`
	Seq            int64          `json:"seq" gorm:"column:seq"`
	ClientMsgId    string         `json:"client_msg_id" gorm:"column:client_msg_id"`
	SenderId       string         `json:"sender_id" gorm:"column:sender_id"`
	RecvId         string         `json:"recv_id" gorm:"column:recv_id"`
	GroupId        string         `json:"group_id" gorm:"column:group_id"`
	SessionType    int32          `json:"session_type" gorm:"column:session_type"`
	MsgType        int32          `json:"msg_type" gorm:"column:msg_type"`
	Content        MessageContent `json:"content" gorm:"column:content;type:json;serializer:json"`
	Extra          *string        `json:"extra" gorm:"column:extra;type:json"`
	SendAt         int64          `json:"send_at" gorm:"column:send_at"`
	CreatedAt      int64          `json:"created_at" gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt      int64          `json:"updated_at" gorm:"column:updated_at;autoUpdateTime:milli"`
}

// TableName returns the table name for Message
func (Message) TableName() string {
	return "messages"
}

// MessageInfo represents message info for API response
type MessageInfo struct {
	Id             int64              `json:"id"`
	ConversationId string             `json:"conversation_id"`
	Seq            int64              `json:"seq"`
	ClientMsgId    string             `json:"client_msg_id"`
	SenderId       string             `json:"sender_id"`
	SessionType    int32              `json:"session_type"`
	MsgType        int32              `json:"msg_type"`
	Content        FlatMessageContent `json:"content"`
	SendAt         int64              `json:"send_at"`
}

// ToMessageInfo converts Message to MessageInfo
func (m *Message) ToMessageInfo() *MessageInfo {
	return &MessageInfo{
		Id:             m.Id,
		ConversationId: m.ConversationId,
		Seq:            m.Seq,
		ClientMsgId:    m.ClientMsgId,
		SenderId:       m.SenderId,
		SessionType:    m.SessionType,
		MsgType:        m.MsgType,
		Content:        m.Content.ToFlat(),
		SendAt:         m.SendAt,
	}
}
