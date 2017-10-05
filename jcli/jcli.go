package jcli

import (
	"encoding/json"
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/goadapp/goad/goad"
	"github.com/goadapp/goad/result"
	"github.com/goadapp/goad/version"

)

const (
	iniFile        = "goad.ini"
	general        = "general"
	urlKey         = "url"
	methodKey      = "method"
	bodyKey        = "body"
	concurrencyKey = "concurrency"
	requestsKey    = "requests"
	timelimitKey   = "timelimit"
	timeoutKey     = "timeout"
	jsonOutputKey  = "json-output"
	headerKey      = "header"
	regionKey      = "region"
	writeIniKey    = "create-ini-template"
	runDockerKey   = "run-docker"
)

var (
	app             = kingpin.New("goad", "An AWS Lambda powered load testing tool")
	urlArg          = app.Arg(urlKey, "[http[s]://]hostname[:port]/path optional if defined in goad.ini")
	url             = urlArg.String()
	requestsFlag    = app.Flag(requestsKey, "Number of requests to perform. Set to 0 in combination with a specified timelimit allows for unlimited requests for the specified time.").Short('n').Default("1000")
	requests        = requestsFlag.Int()
	concurrencyFlag = app.Flag(concurrencyKey, "Number of multiple requests to make at a time").Short('c').Default("10")
	concurrency     = concurrencyFlag.Int()
	timelimitFlag   = app.Flag(timelimitKey, "Seconds to max. to spend on benchmarking").Short('t').Default("3600")
	timelimit       = timelimitFlag.Int()
	timeoutFlag     = app.Flag(timeoutKey, "Seconds to max. wait for each response").Short('s').Default("15")
	timeout         = timeoutFlag.Int()
	headersFlag     = app.Flag(headerKey, "Add Arbitrary header line, eg. 'Accept-Encoding: gzip' (repeatable)").Short('H')
	headers         = headersFlag.Strings()
	methodFlag      = app.Flag(methodKey, "HTTP method").Short('m').Default("GET")
	method          = methodFlag.String()
	bodyFlag        = app.Flag(bodyKey, "HTTP request body")
	body            = bodyFlag.String()
	outputFileFlag  = app.Flag(jsonOutputKey, "Optional path to file for JSON result storage")
	outputFile      = outputFileFlag.String()
	regionsFlag     = app.Flag(regionKey, "AWS regions to run in. Repeat flag to run in more then one region. (repeatable)")
	regions         = regionsFlag.Strings()
	runDockerFlag   = app.Flag(runDockerKey, "execute in docker container instead of aws lambda")
	runDocker       = runDockerFlag.Bool()
	writeIniFlag    = app.Flag(writeIniKey, "create sample configuration file \""+iniFile+"\" in current working directory")
	writeIni        = writeIniFlag.Bool()
)

func Runner() {
	app.HelpFlag.Short('h')
	app.Version(version.String())
	app.VersionFlag.Short('V')

	config := aggregateConfiguration()
	err := config.Check()
	goad.HandleErr(err)

	// Not currently being used
	//sigChan := make(chan os.Signal, 1)
	//signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	//

	resultChan, teardown := goad.Start(config)
	defer teardown()

	for results := range resultChan {
		message, jsonerr := jsonFromRegionsAggData(results)
		if jsonerr != nil {
			fmt.Println(jsonerr)
			break
		}
		fmt.Println(message)
	}
}

func jsonFromRegionsAggData(result *result.LambdaResults) (string, error) {
	data, jsonerr := json.Marshal(result)
	if jsonerr != nil {
		return "", jsonerr
	}
	return string(data), nil
}
