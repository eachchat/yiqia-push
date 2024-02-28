package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/eachchat/yiqia-push/pkg/push"
)

type PmrConfig struct {
	DefaultTitle   string `yaml:"default_title"`
	DefaultContent string `yaml:"default_content"`
	ImageContent   string `yaml:"image_content"`
	FileContent    string `yaml:"file_content"`
}

func (c *PmrConfig) Validate() error {
	if c.DefaultTitle == "" {
		return fmt.Errorf("default title is required")
	}
	if c.DefaultContent == "" {
		return fmt.Errorf("default content is required")
	}
	if c.ImageContent == "" {
		c.ImageContent = "[image]"
	}
	if c.FileContent == "" {
		c.FileContent = "[file]"
	}
	return nil
}

// pmr 处理消息实体
type pmr func(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig)

func parseMessage(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig) {
	var pmr pmr
	switch {
	case notification.Content.Msgtype == "m.text":
		pmr = textPMR
	case notification.Content.Msgtype == "m.image":
		pmr = imagePMR
	case notification.Content.Msgtype == "m.file":
		pmr = filePMR
	// case notification.Content.Membership == "invite":
	// 各种事件消息，如 邀请
	// case notification.Content.CallID != "":
	// 语音通话\视频通话
	default:
		pmr = defaultPMR
	}

	if notification.RoomName != "" {
		message.Payload.Title = notification.RoomName
	} else if notification.SenderDisplayName != "" {
		message.Payload.Title = notification.SenderDisplayName
	} else {
		message.Payload.Title = cfg.DefaultTitle
	}

	pmr(ctx, notification, message, cfg)
}

func defaultPMR(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig) {
	message.Payload.Content = cfg.DefaultContent
}

func textPMR(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig) {
	pruneBody := func(str string) string {
		var max = 35 * 2
		var second int
		var counter int
		var suffix string
		for first := range str {
			if first-second > 1 {
				counter += 5
			} else {
				counter += 2
			}
			second = first
			if counter > max {
				suffix = "..."
				break
			}
		}

		if len(str)-second < 4 {
			second = len(str)
		}

		return str[:second] + suffix
	}

	if notification.Content.NewContent != nil {
		// 如果是修改消息，则推送修改后的内容
		message.Payload.Content = pruneBody(notification.Content.NewContent.Body)
	} else if notification.Content.RelatesTO != nil {
		// 如果是回复消息，则推送回复内容
		// 回复消息和修改消息是可以叠加的，即修改回复消息
		if ss := strings.SplitAfterN(notification.Content.Body, "\n\n", 2); len(ss) > 1 {
			message.Payload.Content = pruneBody(ss[1])
		} else {
			message.Payload.Content = pruneBody(notification.Content.Body)
		}
	} else {
		message.Payload.Content = pruneBody(notification.Content.Body)
	}
}

func imagePMR(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig) {
	message.Payload.Content = cfg.ImageContent
}

func filePMR(ctx context.Context, notification Notification, message *push.Message, cfg *PmrConfig) {
	message.Payload.Content = cfg.FileContent
}
