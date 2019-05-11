package chitchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/solo-io/glooshot/demos/services/stats_app/pkg/stats"

	"github.com/solo-io/glooshot/demos/services/stats_app/pkg/setup"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

func MakeSmallTalk(opts setup.Opts, selfStats *stats.Stats) {
	for {
		for i, n := range opts.Neighbors {
			if i != opts.StreetAddress {
				fmt.Printf("calling %v\n", n)
				msg, delta, err := curl(fmt.Sprintf("http://%v/", n))
				if err != nil {
					selfStats.IncrementErrors()
					fmt.Printf("ouch: %v\n", err)
				}
				selfStats.RecordDelta(delta)
				fmt.Println(msg)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

type chatter struct {
	ctx   context.Context
	opts  setup.Opts
	stats *stats.Stats
}

func NewChatterHandler(ctx context.Context, opts setup.Opts, rootStats *stats.Stats) chatter {
	loggingContext := []interface{}{"type", "stats"}
	return chatter{
		ctx:   contextutils.WithLoggerValues(ctx, loggingContext...),
		stats: rootStats,
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
	d.stats.IncrementConversation(r.RemoteAddr)
	err := json.NewEncoder(w).Encode(chitChat{
		From:    d.opts.Name(),
		To:      r.RemoteAddr,
		Message: fmt.Sprintf("reqs: %v, errs: %v", d.stats.TotalOutboundRequests(), d.stats.TotalOutboundRequestErrors()),
	})
	if err != nil {
		d.stats.IncrementErrors()
		d.reportError(err, http.StatusInternalServerError, w)
		return
	}
}

func curl(url string) (string, int, error) {
	body := bytes.NewReader([]byte(url))
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return "", 0, err
	}

	start := time.Now().UnixNano()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	finish := time.Now().UnixNano()
	delta := int(finish - start)
	p := new(bytes.Buffer)
	_, err = io.Copy(p, resp.Body)
	defer resp.Body.Close()

	return p.String(), delta, nil
}
