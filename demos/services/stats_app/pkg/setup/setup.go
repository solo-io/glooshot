package setup

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

func GetOptsFromEnv() (Opts, error) {
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

func (o *Opts) Name() string {
	return fmt.Sprintf("neighbor-%v", o.StreetAddress)
}
