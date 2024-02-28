package oppo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

const (
	host = "https://api.push.oppomobile.com"
)

type Endpoints struct {
	locker sync.Mutex
	token  *Token

	GetTokenEndpoint   endpoint.Endpoint
	PushNoticeEndpoint endpoint.Endpoint
}

func newEndpoints(ctx context.Context, cfg *Config) (*Endpoints, error) {
	tgt, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	tgt.Path = ""
	var endpoints *Endpoints

	options := []httptransport.ClientOption{}

	endpoints = &Endpoints{
		GetTokenEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = "/server/v1/auth"

			values := url.Values{}
			values.Add("app_key", cfg.AppKey)

			timestamp := time.Now().UnixNano() / (1e6)
			values.Add("timestamp", strconv.Itoa(int(timestamp)))
			values.Add("sign", endpoints.sign(ctx, cfg, timestamp))

			body := strings.NewReader(values.Encode())
			r.Body = io.NopCloser(body)
			r.ContentLength = int64(len(values.Encode()))

			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("get token failed, status code: %d", resp.StatusCode)
			}
			defer resp.Body.Close()

			body := &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    struct {
					// default 24 hours
					AuthToken  string `json:"auth_token"`
					CreateTime int    `json:"create_time"`
				} `json:"data"`
			}{}

			err = json.NewDecoder(resp.Body).Decode(body)
			if err != nil {
				return nil, fmt.Errorf("failed decode token: %v", err)
			}

			return body.Data.AuthToken, nil
		}, options...).Endpoint(),
		PushNoticeEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = "/server/v1/message/notification/unicast"

			authToken, err := endpoints.getToken(ctx)
			if err != nil {
				return fmt.Errorf("failed get token: %v", err)
			}

			values := url.Values{}
			values.Add("auth_token", authToken)

			req := request.(*push.Message)
			// more info: https://open.oppomobile.com/new/developmentDoc/info?id=11238
			message := struct {
				// 推送的目标类型 2: registration_id 5: 别名
				TargetType int `json:"target_type"`
				// 推送目标，按taget_type对应填入，仅接受一个值。
				TargetValue string `json:"target_value"`
				// 消息到达客户端后是否校验registration_id。 true表示推送目标与客户端registration_id进行比较，如果一致则继续展示，不一致则就丢弃；false表示不校验
				VerifyRegistrationID bool `json:"verify_registration_id"`

				Notification struct {
					// 通知栏样式 default 1
					Style int `json:"style"`
					// 设置在通知栏展示的通知栏标题, 【字数串长度限制在50个字符内，中英文字符及特殊符号（如emoji）均视为一个字符】
					Title   string `json:"title"`
					Content string `json:"content"`
					// 点击通知栏后触发的动作类型。 0.启动应用；1.跳转指定应用内页（action标签名）；2.跳转网页；4.跳转指定应用内页（全路径类名）；【非必填，默认值为0】; 5.跳转Intent scheme URL
					ClickActionType     int    `json:"click_action_type"`
					ClickActionActivity string `json:"click_action_activity"`
					// 是否是离线消息。如果是离线消息，OPPO PUSH在设备离线期间缓存消息一段时间，等待设备上线接收。 default true
					OffLine bool `json:"off_line"`
					// 离线消息的存活时间，单位是秒。存活时间最大允许设置为10天，参数超过10天以10天传入。 default 3600
					OffLineTTL int `json:"off_line_ttl"`
					// 通知栏通道（NotificationChannel），从Android9开始，Android设备发送通知栏消息必须要指定通道ID，（如果是快应用，必须带置顶的通道Id:OPPO PUSH推送）
					ChannelID string `json:"channel_id"`
				} `json:"notification"`
			}{
				TargetType:           2,
				TargetValue:          strings.Join(req.DeviceTokens, ","),
				VerifyRegistrationID: false,
			}

			message.Notification.Title = req.Payload.Title
			message.Notification.Content = req.Payload.Content
			message.Notification.Style = 1
			message.Notification.ClickActionType = 0
			message.Notification.OffLine = true
			message.Notification.OffLineTTL = 60 * 60 * 24 * 10
			message.Notification.ChannelID = cfg.ChannelID

			messageByte, _ := json.Marshal(message)
			values.Add("message", string(messageByte))

			body := strings.NewReader(values.Encode())
			r.Body = io.NopCloser(body)
			r.ContentLength = int64(len(values.Encode()))

			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("failed push message, code: %d", resp.StatusCode)
			}
			defer resp.Body.Close()

			body := &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    struct {
					MessageID      string `json:"messageId"`
					RegistrationID string `json:"registrationId"`
				}
			}{}

			err = json.NewDecoder(resp.Body).Decode(body)
			if err != nil {
				return nil, fmt.Errorf("failed decode push result: %v", err)
			}

			if body.Code != 0 {
				return nil, fmt.Errorf("failed push message: %s, messageID: %s, registrationID: %s",
					body.Message, body.Data.MessageID, body.Data.RegistrationID)
			}

			return body, nil
		}, options...).Endpoint(),
	}
	return endpoints, nil
}

type Token struct {
	AccessToken string
	ExpiresIn   int64
}

// getToken get token from OPPO server
func (endpoints *Endpoints) getToken(ctx context.Context) (string, error) {
	if endpoints.token == nil || time.Now().Add(time.Minute*5).Unix() > endpoints.token.ExpiresIn {
		if !endpoints.locker.TryLock() {
			<-time.After(time.Millisecond * 500)
			return endpoints.getToken(ctx)
		}

		token, err := endpoints.GetTokenEndpoint(ctx, nil)
		if err != nil {
			endpoints.locker.Unlock()
			return "", fmt.Errorf("failed get token: %v", err)
		}
		endpoints.token = &Token{
			AccessToken: token.(string),
			// default token expires in 24 hours
			ExpiresIn: time.Now().Unix() + 60*60*24,
		}
		endpoints.locker.Unlock()
	}
	return endpoints.token.AccessToken, nil
}

func (endpoints *Endpoints) sign(ctx context.Context, cfg *Config, timestamp int64) string {
	signStr := fmt.Sprintf("%s%d%s", cfg.AppKey, timestamp, cfg.MasterSecret)
	hash := sha256.New()
	_, _ = hash.Write([]byte(signStr))

	return hex.EncodeToString(hash.Sum(nil))
}
