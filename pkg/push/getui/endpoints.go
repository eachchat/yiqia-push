package getui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

// host is the default host of the GETUI push service.
var host = "https://restapi.getui.com"

type Endpoints struct {
	sync.Mutex
	token *Token

	GetTokenEndpoint   endpoint.Endpoint
	PushNoticeEndpoint endpoint.Endpoint
}

func newEndpoints(ctx context.Context, cfg *Config) (*Endpoints, error) {
	tgt, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host: %v", err)
	}
	tgt.Path = ""
	var endpoints *Endpoints

	options := []httptransport.ClientOption{
		httptransport.ClientBefore(
			func(ctx context.Context, r *http.Request) context.Context {
				token, err := endpoints.getToken(ctx)
				if err != nil {
					r.Close = true
					return ctx
				}
				r.Header.Set("Token", token)
				return ctx
			}),
	}

	endpoints = &Endpoints{
		// more info: https://docs.getui.com/getui/server/rest_v2/token/
		GetTokenEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, i interface{}) error {
			r.URL.Path = fmt.Sprintf("/v2/%s/auth", cfg.AppID)

			timeStamp := time.Now().UnixMilli()
			sign, err := endpoints.sign(ctx, cfg.AppKey, cfg.MasterSecret, timeStamp)
			if err != nil {
				return fmt.Errorf("failed to sign: %v", err)
			}
			body := map[string]interface{}{
				"sign":      sign,
				"timestamp": fmt.Sprintf("%d", timeStamp),
				"appkey":    cfg.AppKey,
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(body)
			if err != nil {
				return err
			}

			r.Body = io.NopCloser(&buf)
			r.Header.Set("Content-Type", "application/json;charset=utf-8")

			return nil
		}, func(ctx context.Context, r *http.Response) (interface{}, error) {
			if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusBadRequest {
				return nil, fmt.Errorf("failed to get token: %s", r.Status)
			}
			defer r.Body.Close()

			var resp = struct {
				baseResponse
				Data struct {
					Token      string `json:"token"`
					ExpireTime string `json:"expire_time"`
				}
			}{}

			err = json.NewDecoder(r.Body).Decode(&resp)
			if err != nil {
				return nil, fmt.Errorf("failed to decode response: %v", err)
			}

			if resp.Code != 0 {
				return nil, fmt.Errorf("failed to get token: %s", resp.Message)
			}

			timeStamp, err := strconv.ParseInt(resp.Data.ExpireTime, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse expire time: %v", err)
			}

			return &Token{
				Token:      resp.Data.Token,
				ExpireTime: timeStamp,
			}, nil
		}).Endpoint(),
		// more info: https://docs.getui.com/getui/server/rest_v2/push/
		PushNoticeEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, i interface{}) error {
			r.URL.Path = fmt.Sprintf("/v2/%s/push/single/cid", cfg.AppID)

			req := i.(*push.Message)
			body := map[string]interface{}{
				"request_id": req.Payload.BusinessID,
				"audience": map[string]interface{}{
					"cid": req.DeviceTokens,
				},
				"push_message": map[string]interface{}{
					"notification": map[string]interface{}{
						"title":      req.Payload.Title,
						"body":       req.Payload.Content,
						"click_type": "startapp",
					},
				},
				"push_channel": map[string]interface{}{
					"android": map[string]interface{}{
						"ups": map[string]interface{}{
							"notification": map[string]interface{}{
								"title":      req.Payload.Title,
								"body":       req.Payload.Content,
								"click_type": "startapp",
							},
						},
					},
				},
			}

			var buf bytes.Buffer
			err = json.NewEncoder(&buf).Encode(body)
			if err != nil {
				return fmt.Errorf("failed to encode body: %v", err)
			}

			r.Body = io.NopCloser(&buf)
			r.Header.Set("Content-Type", "application/json;charset=utf-8")

			return nil
		}, func(ctx context.Context, r *http.Response) (interface{}, error) {
			if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusBadRequest {
				return nil, fmt.Errorf("failed to push notice: %s", r.Status)
			}
			defer r.Body.Close()

			var resp = struct {
				baseResponse
				Data map[string]interface{} `json:"data"`
			}{}
			err = json.NewDecoder(r.Body).Decode(&resp)
			if err != nil {
				return nil, fmt.Errorf("failed to decode response: %v", err)
			}

			if resp.Code != 0 {
				return nil, fmt.Errorf("failed to push notice: %s", resp.Message)
			}

			return resp, nil
		}, options...).Endpoint(),
	}

	return endpoints, nil
}

type baseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// sign is the signature of the GETUI push service.
// timestamp: current timestamp in milliseconds
func (e *Endpoints) sign(ctx context.Context, appKey string, masterSecret string, timestamp int64) (string, error) {
	atm := fmt.Sprintf("%s%d%s", appKey, timestamp, masterSecret)
	hash := sha256.New()
	_, err := hash.Write([]byte(atm))
	if err != nil {
		return "", fmt.Errorf("failed to write hash: %v", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type Token struct {
	ExpireTime int64  `json:"expire_time"`
	Token      string `json:"token"`
}

func (e *Endpoints) getToken(ctx context.Context) (string, error) {
	if e.token == nil || time.Now().Add(5*time.Minute).UnixMilli() > e.token.ExpireTime {
		if !e.TryLock() {
			<-time.After(time.Millisecond * 500)
			return e.getToken(ctx)
		}
		token, err := e.GetTokenEndpoint(ctx, nil)
		if err != nil {
			e.Unlock()
			return "", fmt.Errorf("failed to get token: %v", err)
		}
		e.token = token.(*Token)
		e.Unlock()
	}
	return e.token.Token, nil
}
