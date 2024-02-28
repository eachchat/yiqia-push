package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/eachchat/yiqia-push/pkg/push"
	"github.com/eachchat/yiqia-push/pkg/push/overall"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
	"github.com/google/uuid"
)

type Pusher struct {
	cfg    *Config
	logger log.Logger

	overall overall.OverAll
}

type Config struct {
	PmrConfig `yaml:"pmr"`
}

func New(ctx context.Context, cfg *Config, overall *overall.OverAll, logger log.Logger) *Pusher {
	return &Pusher{
		cfg:     cfg,
		logger:  logger,
		overall: *overall,
	}
}

func (p *Pusher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u := uuid.New()
	requestID := u.String()

	logger := log.With(p.logger, "requestID", requestID)

	rr, _, err := newReusableRequest(r, 1024*1024*5)
	if err != nil {
		level.Error(logger).Log("msg", "fail new reusable request", "err", err)
		errorW(w, http.StatusBadRequest, "Fail new reusable request")
		return
	}

	if len(rr.body) == 0 {
		level.Error(logger).Log("msg", "body is missing")
		errorW(w, http.StatusBadRequest, "body is missing")
		return
	}

	params := new(Params)
	err = json.Unmarshal(rr.body, params)
	if err != nil {
		level.Error(logger).Log("msg", "fail unmarshal request body", "err", err)
		errorW(w, http.StatusBadRequest, "Fail unmarshal request body")
		return
	}

	logger = log.With(logger, "eventID", params.Notification.EventID)
	level.Info(logger).Log("msg", "receive request", "sender", params.Notification.SenderDisplayName)
	if len(params.Notification.Devices) == 0 {
		level.Error(logger).Log("msg", "devices field is missing")
		errorW(w, http.StatusBadRequest, "Devices field  is missing")
		return
	}

	if params.Notification.Sender == "" {
		w.Write([]byte("{}"))
		w.Header().Set("Content-Type", "application/json")
		return
	}

	// 按照设备类型分组
	deviceMap := make(map[string][]Devices)
	for i, device := range params.Notification.Devices {
		tag := "default"
		if strings.HasPrefix(device.AppID, "android_") {
			tag = strings.TrimPrefix(device.AppID, "android_")
		}

		devices, ok := deviceMap[tag]
		if !ok {
			devices = make([]Devices, 0)
		}

		devices = append(devices, params.Notification.Devices[i])
		deviceMap[tag] = devices
	}

	for tag, devices := range deviceMap {
		level.Info(logger).Log("msg", "try to push message", "tag", tag)
		for _, device := range devices {
			pusher, err := p.overall.GetPushClient(tag)
			if err != nil {
				level.Error(logger).Log("msg", "fail get push client", "err", err, "tag", tag)
				continue
			}

			ctx := context.Background()

			message := &push.Message{
				DeviceTokens: []string{
					device.PushKey,
				},
				Payload: &push.Payload{
					BusinessID: requestID,
				},
			}

			parseMessage(ctx, params.Notification, message, &p.cfg.PmrConfig)

			err = pusher.PushNotice(ctx, message)

			level.Info(logger).Log("msg", "push message", "deviceToken", device.PushKey)
			if err != nil {
				level.Error(logger).Log("msg", "fail push message", "err", err)
				continue
			}
		}
	}

	w.Write([]byte("{}"))
	w.Header().Set("Content-Type", "application/json")

}

func errorW(w http.ResponseWriter, code int, message string) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

var errBodyTooLarge = errors.New("request body too large")

// reusableRequest keeps in memory the body of the given request,
// so that the request can be fully cloned by each mirror.
type reusableRequest struct {
	req  *http.Request
	body []byte
}

// if the returned error is errBodyTooLarge, newReusableRequest also returns the
// bytes that were already consumed from the request's body.
func newReusableRequest(req *http.Request, maxBodySize int64) (*reusableRequest, []byte, error) {
	if req == nil {
		return nil, nil, errors.New("nil input request")
	}
	if req.Body == nil || req.ContentLength == 0 {
		return &reusableRequest{req: req}, nil, nil
	}

	// unbounded body size
	if maxBodySize < 0 {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, nil, err
		}
		return &reusableRequest{
			req:  req,
			body: body,
		}, nil, nil
	}

	// we purposefully try to read _more_ than maxBodySize to detect whether
	// the request body is larger than what we allow for the mirrors.
	body := make([]byte, maxBodySize+1)
	n, err := io.ReadFull(req.Body, body)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, err
	}

	// we got an ErrUnexpectedEOF, which means there was less than maxBodySize data to read,
	// which permits us sending also to all the mirrors later.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return &reusableRequest{
			req:  req,
			body: body[:n],
		}, nil, nil
	}

	// err == nil , which means data size > maxBodySize
	return nil, body[:n], errBodyTooLarge
}
