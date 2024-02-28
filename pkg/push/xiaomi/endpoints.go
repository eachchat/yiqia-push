package xiaomi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

const (
	host = "https://api.xmpush.xiaomi.com"
)

type Endpoints struct {
	PushNoticeEndpoint endpoint.Endpoint
}

func newEndpoints(ctx context.Context, conf *Config) (*Endpoints, error) {
	tgt, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	tgt.Path = ""

	var endpoints *Endpoints
	options := []httptransport.ClientOption{}

	endpoints = &Endpoints{
		PushNoticeEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*push.Message)
			values := url.Values{}
			values.Add("payload", url.QueryEscape(req.Payload.Content))
			values.Add("restricted_package_name", conf.AppPkgName)
			values.Add("title", req.Payload.Title)
			values.Add("description", req.Payload.Content)
			// values.Add("notify_type", "-1")
			values.Add("time_to_live", "86400")
			values.Add("extra.notify_foreground", "1")
			values.Add("extra.notify_effect", "1")
			values.Add("extra.channel_id", conf.ChannelID)
			values.Add("registration_id", strings.Join(req.DeviceTokens, ","))

			body := strings.NewReader(values.Encode())
			r.Body = io.NopCloser(body)
			r.ContentLength = int64(len(values.Encode()))

			r.URL.Path = "/v3/message/regid"
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Authorization", fmt.Sprintf("key=%s", conf.AppSecret))
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			defer resp.Body.Close()

			body := &struct {
				Result      string            `json:"result,omitempty"`
				Description string            `json:"description,omitempty"`
				Data        map[string]string `json:"data,omitempty"`
				Code        int               `json:"code,omitempty"`
				Info        string            `json:"info,omitempty"`
				Reason      string            `json:"reason,omitempty"`
			}{}
			err = json.NewDecoder(resp.Body).Decode(body)
			if err != nil {
				return nil, fmt.Errorf("failed decode body: %v", err)
			}

			if body.Code != 0 {
				return nil, fmt.Errorf("failed push notice: %s, info: %s", body.Reason, body.Info)
			}
			return body, nil
		}, options...).Endpoint(),
	}
	return endpoints, nil
}
