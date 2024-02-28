package notify

type TooLTT struct {
	Notification Notification `json:"notification"`
}
type Content struct {
	Body       string   `json:"body"`
	Msgtype    string   `json:"msgtype"`
	NewContent *Content `json:"m.new_content"`
	RelatesTO  *struct {
		EventID string `json:"event_id"`
		RelType string `json:"rel_type"`
	} `json:"m.relates_to"`

	DisplayName string `json:"displayname"`
	Membership  string `json:"membership"`

	CallID       string `json:"call_id"`
	Capabilities struct {
		Dtmf       bool `json:"m.call.dtmf"`
		Transferee bool `json:"m.call.transferee"`
	} `json:"capabilities"`
	// Lifetime int
	PartyID string `json:"party_id"`
	Version string `json:"version"`
}

type Counts struct {
	MissedCalls int `json:"missed_calls"`
	Unread      int `json:"unread"`
}
type Data struct {
}
type Tweaks struct {
	Sound string `json:"sound"`
}
type Devices struct {
	AppID     string `json:"app_id"`
	Data      Data   `json:"data"`
	PushKey   string `json:"pushkey"`
	PushKeyTs int    `json:"pushkey_ts"`
	Tweaks    Tweaks `json:"tweaks"`
}
type Notification struct {
	Content           Content   `json:"content"`
	Counts            Counts    `json:"counts"`
	Devices           []Devices `json:"devices"`
	EventID           string    `json:"event_id"`
	Prio              string    `json:"prio"`
	RoomAlias         string    `json:"room_alias"`
	RoomID            string    `json:"room_id"`
	RoomName          string    `json:"room_name"`
	Sender            string    `json:"sender"`
	SenderDisplayName string    `json:"sender_display_name"`
	Type              string    `json:"type"`
}

type Params struct {
	Notification Notification `json:"notification"`
}
