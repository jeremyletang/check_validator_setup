package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	dnapipb "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const gqlPayload = `{"query": "{epoch{id}}"}`

var (
	//go:embed testnet_config.json
	testnetBuf []byte

	//go:embed mainnet_config.json
	mainnetBuf []byte

	timeout = 2 * time.Second

	testnetConfig bool
	only          string
	output        string
)

type config struct {
	Validators []struct {
		Name string `json:"name"`
		GRPC string `json:"grpc"`
		REST string `json:"rest"`
		GQL  string `json:"gql"`
	} `json:"validators"`
}

type aPIResult struct {
	API       string        `json:"api"`
	TimeTaken time.Duration `json:"time_taken"`
	Error     string        `json:"error"`
}

type results struct {
	Name       string      `json:"name"`
	APIResults []aPIResult `json:"api_results"`
}

func init() {
	flag.BoolVar(&testnetConfig, "testnet", false, "check testnet")
	flag.StringVar(&only, "only", "", "check a single validator")
	flag.StringVar(&output, "output", "human", "results output [human|json]")
}

func main() {
	flag.Parse()
	var buf = mainnetBuf
	if testnetConfig {
		buf = testnetBuf
	}
	if len(only) > 0 {
		only = strings.ToLower(only)
	}

	var isJsonOutput bool
	switch output {
	case "human":
		break
	case "json":
		isJsonOutput = true
	default:
		log.Fatalf("invalid output format: %v", output)
	}

	cfg := config{}
	err := json.Unmarshal(buf, &cfg)
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// validate only is a correct validator if specified
	if len(only) > 0 {
		var exists bool
		for _, v := range cfg.Validators {
			if strings.EqualFold(only, v.Name) {
				exists = true
				break
			}
		}
		if !exists {
			log.Fatalf("not an existing validator: %v", only)
		}
	}

	var bar *progressbar.ProgressBar
	if !isJsonOutput {
		if len(only) > 0 {
			bar = progressbar.Default(4)
		} else {
			bar = progressbar.Default(int64(len(cfg.Validators) * 4))
		}
	}

	res := []results{}

	for _, v := range cfg.Validators {
		if len(only) > 0 && !strings.EqualFold(only, v.Name) {
			continue
		}

		newRes := results{
			Name: v.Name,
		}

		errStr := ""
		timeTaken, err := checkGRPC(v.GRPC)
		if err != nil {
			errStr = err.Error()
		}
		newRes.APIResults = append(newRes.APIResults, aPIResult{
			API:       "core",
			TimeTaken: timeTaken,
			Error:     errStr,
		})
		if !isJsonOutput {
			bar.Add(1)
		}

		errStr = ""
		timeTaken, err = checkGRPCDN(v.GRPC)
		if err != nil {
			errStr = err.Error()
		}
		newRes.APIResults = append(newRes.APIResults, aPIResult{
			API:       "datanode",
			TimeTaken: timeTaken,
			Error:     errStr,
		})
		if !isJsonOutput {
			bar.Add(1)
		}

		errStr = ""
		timeTaken, err = checkREST(v.REST)
		if err != nil {
			errStr = err.Error()
		}
		newRes.APIResults = append(newRes.APIResults, aPIResult{
			API:       "rest",
			TimeTaken: timeTaken,
			Error:     errStr,
		})

		if !isJsonOutput {
			bar.Add(1)
		}

		errStr = ""
		timeTaken, err = checkGQL(v.GQL)
		if err != nil {
			errStr = err.Error()
		}
		newRes.APIResults = append(newRes.APIResults, aPIResult{
			API:       "gql",
			TimeTaken: timeTaken,
			Error:     errStr,
		})
		if !isJsonOutput {
			bar.Add(1)
		}

		res = append(res, newRes)
	}

	if output == "human" {
		printResults(res)
	} else {
		buf, err := json.Marshal(res)
		if err != nil {
			log.Fatalf("could not format output: %v", err)
		}
		fmt.Printf("%v\n", string(buf))
	}
}

func printResults(results []results) {
	t := table.NewWriter()
	t.AppendHeader(table.Row{"validator", "core", "datanode", "rest", "graphql"})

	t2 := table.NewWriter()
	t2.AppendHeader(table.Row{"validator", "api", "error"})

	for _, v := range results {
		resMap := map[string]aPIResult{}
		for _, vr := range v.APIResults {
			resMap[vr.API] = vr
			if len(vr.Error) > 0 {
				t2.AppendRow(table.Row{v.Name, vr.API, vr.Error})
			}
		}

		t.AppendRow(table.Row{
			v.Name,
			coloredDuration(resMap["core"]),
			coloredDuration(resMap["datanode"]),
			coloredDuration(resMap["rest"]),
			coloredDuration(resMap["gql"]),
		})
	}

	fmt.Println(t.Render())
	fmt.Println(t2.Render())
}

func coloredDuration(res aPIResult) string {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	if len(res.Error) > 0 {
		return red(res.TimeTaken.String())
	}

	return green(res.TimeTaken.String())
}

func checkREST(address string) (time.Duration, error) {
	s, err := url.JoinPath(address, "api/v2/info")
	if err != nil {
		return 0, err
	}

	now := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	s := address

	now := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = connDT.Info(ctx, &dnapipb.InfoRequest{})

	return time.Since(now), err
}
