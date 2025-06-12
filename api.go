package main

// Send message

type SendChatMessageRequestBody struct {
	BroadcasterID        string `json:"broadcaster_id"`
	SenderID             string `json:"sender_id"`
	Message              string `json:"message"`
	ReplyParentMessageID string `json:"reply_parent_message_id"`
}

// Subscription

type createSubscriptionTransport struct {
	Method    string `json:"method"` // websocket
	SessionID string `json:"session_id"`
}

type CreateSubscriptionRequestBody struct {
	Type      string                      `json:"type"`      // "channel.chat.message"
	Version   string                      `json:"version"`   // 1
	Condition map[string]any              `json:"condition"` // broadcaster_user_id: {icosatess}, user_id: {icosabot}
	Transport createSubscriptionTransport `json:"transport"`
}

// Websocket message types

type WelcomeMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "session_welcome"
	} `json:"metadata"`
	Payload struct {
		Session struct {
			ID                      string `json:"id"`
			KeepaliveTimeoutSeconds int    `json:"keepalive_timeout_seconds"`
		} `json:"session"`
	} `json:"payload"`
}

type KeepaliveMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "session_keepalive"
	} `json:"metadata"`
	Payload struct{} `json:"payload"`
}

type channelChatMessageEvent struct {
	ChatterUserName string `json:"chatter_user_name"`
	MessageID       string `json:"message_id"`
	Message         struct {
		Text string `json:"text"`
	} `json:"message"`
}

type NotificationMessage struct {
	Metadata struct {
		MessageType string `json:"message_type"` // "notification"
	} `json:"metadata"`
	Payload struct {
		Event channelChatMessageEvent `json:"event"`
	} `json:"payload"`
}
