package vivo

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	skipAuthKey = "X-Skip-Auth"
)

const (
	host = "https://api-push.vivo.com.cn"
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
	appID, err := strconv.Atoi(cfg.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed convert appid to int: %v", err)
	}

	var endpoints *Endpoints

	options := []httptransport.ClientOption{
		httptransport.ClientBefore(func(ctx context.Context, r *http.Request) context.Context {
			if r.Header.Get(skipAuthKey) != "" {
				r.Header.Del(skipAuthKey)
				return ctx
			}

			token, err := endpoints.getToken(ctx)
			if err != nil {
				fmt.Println(err)
			}

			r.Header.Set("authToken", token)
			return ctx
		}),
	}

	endpoints = &Endpoints{
		GetTokenEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = "/message/auth"

			timestamp := time.Now().UTC().UnixNano() / (1e6)

			body := map[string]interface{}{
				"appId":     cfg.AppID,
				"appKey":    cfg.AppKey,
				"timestamp": timestamp,
				"sign":      endpoints.sign(ctx, cfg, timestamp),
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(body)
			if err != nil {
				return err
			}

			r.Body = io.NopCloser(&buf)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Add(skipAuthKey, "True")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			defer resp.Body.Close()

			body := &struct {
				Result    int    `json:"result,omitempty"`
				Desc      string `json:"desc,omitempty"`
				AuthToken string `json:"authToken,omitempty"`
			}{}

			err = json.NewDecoder(resp.Body).Decode(body)
			if err != nil {
				return nil, fmt.Errorf("failed decode token: %v", err)
			}

			if body.Result != 0 {
				return nil, fmt.Errorf("failed decode token: %v", body.Desc)
			}
			return body.AuthToken, nil
		}, options...).Endpoint(),
		PushNoticeEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*push.Message)

			// more info: https://dev.vivo.com.cn/documentCenter/doc/362#w2-98542835
			body := struct {
				AppID           int               `json:"appId"`
				RegID           string            `json:"regId"`
				NotifyType      int               `json:"notifyType"`
				Title           string            `json:"title"`
				Content         string            `json:"content"`
				TimeToLive      int               `json:"timeToLive"`
				SkipType        int               `json:"skipType"`
				SkipContent     string            `json:"skipContent"`
				Classification  int               `json:"classification"`
				NetworkType     string            `json:"networkType"`
				ClientCustomMap map[string]string `json:"clientCustomMap"`
				Extra           map[string]string `json:"extra"`
				RequestID       string            `json:"requestId"`
				Category        string            `json:"category"`
			}{
				AppID: appID,
				RegID: strings.Join(req.DeviceTokens, ","),
				// 通知类型 1:无，2:响铃，3:振动，4:响铃和振动
				NotifyType: 1,
				// 消息缓存时间，单位是秒。在用户设备没有网络时，消息在Push服务器进行缓存，在消息缓存时间内用户设备重新连接网络，消息会下发，超过缓存时间后消息会丢弃。
				// 取值至少60秒，最长7天。
				TimeToLive: 60 * 60 * 24,
				Title:      req.Payload.Title,
				Content:    req.Payload.Content,
				// 点击跳转类型 1：打开APP首页 2：打开链接 3：自定义 4:打开app内指定页面
				SkipType: 1,
				// 消息类型 0：运营类消息，1：系统类消息。不填默认为0
				Classification: 1,
				RequestID:      req.Payload.BusinessID,
				Category:       "IM",
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(body)
			if err != nil {
				return err
			}

			r.Body = io.NopCloser(&buf)
			r.URL.Path = "/message/send"
			r.Header.Set("Content-Type", "application/json")
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			defer resp.Body.Close()

			body := &struct {
				Result int    `json:"result,omitempty"`
				Desc   string `json:"desc,omitempty"`
				TaskID string `json:"taskId,omitempty"`
			}{}

			err = json.NewDecoder(resp.Body).Decode(body)
			if err != nil {
				return nil, fmt.Errorf("failed decode push result: %v", err)
			}

			if body.Result != 0 {
				return nil, fmt.Errorf("failed push notice: %s, taskID: %s", body.Desc, body.TaskID)
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

func (endpoints *Endpoints) sign(ctx context.Context, conf *Config, timestamp int64) string {
	signStr := fmt.Sprintf("%s%s%d%s", conf.AppID, conf.AppKey, timestamp, conf.AppSecret)
	hash := md5.New()
	_, _ = hash.Write([]byte(signStr))

	return hex.EncodeToString(hash.Sum(nil))
}
