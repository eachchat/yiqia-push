package huawei

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

const (
	skipAuthKey = "X-Skip-Auth"
)

const (
	host      = "https://push-api.cloud.huawei.com"
	tokenHost = "https://oauth-login.cloud.huawei.com"
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

	options := []httptransport.ClientOption{
		httptransport.ClientBefore(func(ctx context.Context, r *http.Request) context.Context {
			if r.Header.Get(skipAuthKey) != "" {
				r.Header.Del(skipAuthKey)
				return ctx
			}

			token, err := endpoints.getToken(ctx)
			if err != nil {
				// FIXME:
				fmt.Println(err)
			}

			r.Header.Set("Authorization", token)
			return ctx
		}),
	}

	endpoints = &Endpoints{
		GetTokenEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			var err error
			r.URL, err = url.Parse(tokenHost)
			if err != nil {
				return fmt.Errorf("failed parse auth host: %v", err)
			}
			r.URL.Path = "/oauth2/v3/token"
			r.Header.Add(skipAuthKey, "True")

			values := url.Values{}
			values.Set("grant_type", "client_credentials")
			values.Set("client_id", cfg.ClientId)
			values.Set("client_secret", cfg.ClientSecret)

			body := strings.NewReader(values.Encode())
			r.Body = io.NopCloser(body)
			r.ContentLength = int64(len(values.Encode()))

			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			defer resp.Body.Close()

			token := new(Token)
			err = json.NewDecoder(resp.Body).Decode(token)
			if err != nil {
				return nil, fmt.Errorf("failed decode token: %v", err)
			}
			return token, nil
		}, options...).Endpoint(),
		PushNoticeEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = fmt.Sprintf("/v1/%s/messages:send", cfg.ClientId)
			req := request.(*push.Message)

			// more info: https://developer.huawei.com/consumer/cn/doc/development/HMSCore-References/https-send-api-0000001050986197#section13271045101216
			body := &struct {
				ValidateOnly bool `json:"validate_only,omitempty"`
				Message      struct {
					Android struct {
						Category       string `json:"category,omitempty"`
						TargetUserType int    `json:"target_user_type,omitempty"`
						Notification   struct {
							Title       string `json:"title,omitempty"`
							Body        string `json:"body,omitempty"`
							ClickAction struct {
								Type int `json:"type"`
							} `json:"click_action"`
						} `json:"notification,omitempty"`
					} `json:"android,omitempty"`

					Token []string `json:"token,omitempty"`
				} `json:"message,omitempty"`
			}{
				ValidateOnly: false,
			}

			body.Message.Android.Notification.Title = req.Payload.Title
			body.Message.Android.Notification.Body = req.Payload.Content
			body.Message.Android.Notification.ClickAction.Type = 3
			body.Message.Android.Category = "IM"
			body.Message.Android.TargetUserType = cfg.TargetUserType
			body.Message.Token = req.DeviceTokens

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(body)
			if err != nil {
				return err
			}

			r.Body = io.NopCloser(&buf)
			r.Header.Set("Content-Type", "application/json")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			defer resp.Body.Close()

			body := &struct {
				Code      string `json:"code,omitempty"`
				Msg       string `json:"msg,omitempty"`
				RequestID string `json:"requestId,omitempty"`
			}{}

			if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
				return body, fmt.Errorf("failed decode push notice result: %v", err)
			}

			if body.Code != "80000000" {
				return body, fmt.Errorf("failed push notice: %s, requestID: %s", body.Msg, body.RequestID)
			}

			return body, nil
		}, options...).Endpoint(),
	}
	return endpoints, nil
}

type Token struct {
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
}

func (endpoints *Endpoints) getToken(ctx context.Context) (string, error) {
	if endpoints.token == nil || time.Now().Add(time.Minute*5).Unix() > endpoints.token.ExpiresIn {
		if !endpoints.locker.TryLock() {
			<-time.After(time.Millisecond * 500)
			return endpoints.getToken(ctx)
		}

		tokenInterface, err := endpoints.GetTokenEndpoint(ctx, nil)
		if err != nil {
			endpoints.locker.Unlock()
			return "", fmt.Errorf("failed get token: %v", err)
		}
		token := tokenInterface.(*Token)
		token.ExpiresIn += time.Now().Unix()
		endpoints.token = token
		endpoints.locker.Unlock()
	}

	return fmt.Sprintf("%s %s", endpoints.token.TokenType, endpoints.token.AccessToken), nil
}
