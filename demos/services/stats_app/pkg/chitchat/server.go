package chitchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/solo-io/glooshot/demos/services/stats_app/pkg/setup"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

func MakeSmallTalk(opts setup.Opts) {
	for {
		for i, n := range opts.Neighbors {
			if i != opts.StreetAddress {
				fmt.Printf("calling %v\n", n)
				msg, err := curl(fmt.Sprintf("http://%v/", n))
				if err != nil {
					fmt.Printf("ouch: %v\n", err)
				}
				fmt.Println(msg)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

type chatter struct {
	ctx  context.Context
	opts setup.Opts
}

func NewChatterHandler(ctx context.Context, opts setup.Opts) chatter {
	loggingContext := []interface{}{"type", "stats"}
	return chatter{
		ctx: contextutils.WithLoggerValues(ctx, loggingContext...),
	}
}

func (d chatter) reportError(err error, status int, w http.ResponseWriter) {
	contextutils.LoggerFrom(d.ctx).Errorw("error getting client", zap.Error(err))
	w.WriteHeader(status)
	fmt.Fprint(w, err)
}

type chitChat struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Message string `json:"message"`
}

func (d chatter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(chitChat{
		From:    d.opts.Name(),
		To:      "t",
		Message: "m",
	})
	if err != nil {
		d.reportError(err, http.StatusInternalServerError, w)
		return
	}
}

func curl(url string) (string, error) {
	body := bytes.NewReader([]byte(url))
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	p := new(bytes.Buffer)
	_, err = io.Copy(p, resp.Body)
	defer resp.Body.Close()

	return p.String(), nil
}
