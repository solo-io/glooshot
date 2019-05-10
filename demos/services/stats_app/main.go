package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/solo-io/go-utils/stats"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

type Opts struct {
	StreetAddress int
	Neighbors     []string
	BindAddress   string
}

const (
	EnvSelfNeighborIndex   = "NEIGHBOR_INDEX"
	EnvNeighborServiceList = "NEIGHBOR_SERVICE_LIST"
	EnvBindAddress         = "BIND_ADDRESS"

	NeighborhoodBindAddress = "localhost:8080"
)

var about = `BLOCK PARTY
This app exists to demonstrate the capabilities of Glooshot.
It expects to have a set of "neighbor" services. All services run the same code with different configs.
According to its configuration, the service can identify itself and its neighbor services.
Upon initialization, each service gets the following from its environment (through the pod spec):
- a list of all services in the neighborhood, self included
- a way to identify itself among the members of the neighborhood
The service provides various metrics that can be used to verify affect of a Glooshot experiment. These include:
- number of messages received from each neighbor
- number of seconds that the service has been running
During runtime, certain properties can be configured.
- to be determined/added as necessary
`

func getOptsFromEnv() (Opts, error) {
	neighborList := strings.Split(os.Getenv(EnvNeighborServiceList), ",")
	if len(neighborList) == 0 {
		return Opts{}, fmt.Errorf("no neighbors found, please pass a comma-separated list of neighbor services through %v env var", EnvNeighborServiceList)
	}
	streetAddress := os.Getenv(EnvSelfNeighborIndex)
	if streetAddress == "" {
		return Opts{}, fmt.Errorf("no street address found, please provide an integer between 0 and %v", len(neighborList))
	}
	streetAddressInt, err := strconv.Atoi(streetAddress)
	envBindAddress := os.Getenv(EnvBindAddress)
	if envBindAddress == "" {
		envBindAddress = NeighborhoodBindAddress
	}
	if err != nil {
		return Opts{}, nil
	}
	return Opts{
		StreetAddress: streetAddressInt,
		Neighbors:     neighborList,
		BindAddress:   envBindAddress,
	}, nil
}

func main() {
	ctx := context.Background()
	opts, err := getOptsFromEnv()
	if err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("unable to get options from env", zap.Error(err))
	}

	stats.StartStatsServer()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", newChatterHandler(ctx, opts))
		contextutils.LoggerFrom(ctx).Fatal(http.ListenAndServe(opts.BindAddress, mux))
	}()

	makeSmallTalk(opts)
}

func makeSmallTalk(opts Opts) {
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
	ctx context.Context
}

func newChatterHandler(ctx context.Context, opts Opts) chatter {
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
		From:    "f",
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
