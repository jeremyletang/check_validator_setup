package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	dnapipb "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed config.json
var buf []byte

const gqlPayload = `{"query": "{epoch{id}}"}`

type config struct {
	Validators []struct {
		Name string `json:"name"`
		GRPC string `json:"grpc"`
		REST string `json:"rest"`
		GQL  string `json:"gql"`
	} `json:"validators"`
}

func main() {
	cfg := config{}
	err := json.Unmarshal(buf, &cfg)
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	columnFmt := color.New(color.FgYellow).SprintfFunc()
	headerFmt := color.New(color.FgWhite, color.Underline).SprintfFunc()
	tbl := table.New("name", "core", "datanode", "rest", "graphql")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	bar := progressbar.Default(int64(len(cfg.Validators) * 4))

	for _, v := range cfg.Validators {
		var core, dn, rest, gql string
		timeTaken, err := checkGRPC(v.GRPC)
		if err != nil {
			core = red(timeTaken.String())
		} else {
			core = green(timeTaken.String())
		}
		bar.Add(1)

		timeTaken, err = checkGRPCDN(v.GRPC)
		if err != nil {
			dn = red(timeTaken.String())
		} else {
			dn = green(timeTaken.String())
		}
		bar.Add(1)

		timeTaken, err = checkREST(v.REST)
		if err != nil {
			rest = red(timeTaken.String())
		} else {
			rest = green(timeTaken.String())
		}
		bar.Add(1)

		timeTaken, err = checkGQL(v.GQL)
		if err != nil {
			gql = red(timeTaken.String())
		} else {
			gql = green(timeTaken.String())
		}
		bar.Add(1)

		tbl.AddRow(v.Name, core, dn, rest, gql)
	}

	tbl.Print()
}

func checkREST(address string) (time.Duration, error) {
	s, err := url.JoinPath(address, "api/v2/info")
	if err != nil {
		return 0, err
	}

	now := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s, nil)
	if err == nil {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return time.Since(now), err
		}
		if resp.StatusCode != http.StatusOK {
			return time.Since(now), fmt.Errorf("unexpected http status code: %v", resp.StatusCode)
		}
	}
	return time.Since(now), err
}

func checkGQL(address string) (time.Duration, error) {
	s, err := url.JoinPath(address, "graphql")
	if err != nil {
		return 0, err
	}

	now := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s, bytes.NewBuffer([]byte(gqlPayload)))
	if err == nil {
		req.Header.Add("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return time.Since(now), err
		}
		if resp.StatusCode != http.StatusOK {
			return time.Since(now), fmt.Errorf("unexpected http status code: %v", resp.StatusCode)
		}
	}

	return time.Since(now), err
}

func checkGRPC(address string) (time.Duration, error) {
	useTLS := strings.HasPrefix(address, "tls://")

	var creds credentials.TransportCredentials
	if useTLS {
		address = address[6:]
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	connection, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return 0, err
	}

	now := time.Now()

	connCore := apipb.NewCoreServiceClient(connection)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = connCore.Statistics(ctx, &apipb.StatisticsRequest{})

	return time.Since(now), err
}

func checkGRPCDN(address string) (time.Duration, error) {
	useTLS := strings.HasPrefix(address, "tls://")

	var creds credentials.TransportCredentials
	if useTLS {
		address = address[6:]
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	connection, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return 0, err
	}

	now := time.Now()

	connDT := dnapipb.NewTradingDataServiceClient(connection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = connDT.Info(ctx, &dnapipb.InfoRequest{})

	return time.Since(now), err
}
