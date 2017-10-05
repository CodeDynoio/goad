package jcli

import (
	"fmt"
	"os"
	"strings"
	"reflect"
	"strconv"
	"bytes"
	"io/ioutil"
	"io"

	ini "gopkg.in/ini.v1"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/goadapp/goad/goad/types"
	//"github.com/goadapp/goad/goad"
)

func writeIniFile() {
	stream := bytes.NewBuffer(make([]byte, 0))
	writeConfigStream(stream)
	ioutil.WriteFile(iniFile, stream.Bytes(), 0644)
}

func writeConfigStream(writer io.Writer) {
	stream := bytes.NewBufferString(template)
	stream.WriteTo(writer)
}

func aggregateConfiguration() *types.TestConfig {
	config := parseSettings()
	applyDefaultsFromConfig(config)
	config = parseCommandline()
	applyExtendedConfiguration(config)
	return config
}

func applyDefaultsFromConfig(config *types.TestConfig) {
	applyDefaultIfNotZero(bodyFlag, config.Body)
	applyDefaultIfNotZero(concurrencyFlag, prepareInt(config.Concurrency))
	applyDefaultIfNotZero(headersFlag, config.Headers)
	applyDefaultIfNotZero(methodFlag, config.Method)
	applyDefaultIfNotZero(outputFileFlag, config.Output)
	applyDefaultIfNotZero(regionsFlag, config.Regions)
	applyDefaultIfNotZero(requestsFlag, prepareInt(config.Requests))
	applyDefaultIfNotZero(timelimitFlag, prepareInt(config.Timelimit))
	applyDefaultIfNotZero(timeoutFlag, prepareInt(config.Timeout))
	if config.URL != "" {
		urlArg.Default(config.URL)
	}
	if len(config.Regions) == 0 {
		regionsFlag.Default("us-west-2", "us-east-1", "eu-west-1", "ap-northeast-1")
	}
	if config.RunDocker {
		runDockerFlag.Default("true")
	}
}

func applyDefaultIfNotZero(flag *kingpin.FlagClause, def interface{}) {
	value := reflect.ValueOf(def)
	kind := value.Kind()
	if isNotZero(value) {
		if kind == reflect.Slice || kind == reflect.Array {
			strs := make([]string, 0)
			for i := 0; i < value.Len(); i++ {
				strs = append(strs, value.Index(i).String())
			}
			flag.Default(strs...)
		} else {
			flag.Default(value.String())
		}
	}
}

func prepareInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func isNotZero(v reflect.Value) bool {
	return !isZero(v)
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func loadIni() *ini.File {
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, iniFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println(err.Error())
		}
		return nil
	}
	return cfg
}

func parseSettings() *types.TestConfig {
	config := &types.TestConfig{}
	cfg := loadIni()
	if cfg == nil {
		return config
	}

	generalSection := cfg.Section(general)
	config.URL = generalSection.Key(urlKey).String()
	config.Method = generalSection.Key(methodKey).String()
	config.Body = generalSection.Key(bodyKey).String()
	config.Concurrency, _ = generalSection.Key(concurrencyKey).Int()
	config.Requests, _ = generalSection.Key(requestsKey).Int()
	config.Timelimit, _ = generalSection.Key(timelimitKey).Int()
	config.Timeout, _ = generalSection.Key(timeoutKey).Int()
	config.Output = generalSection.Key(jsonOutputKey).String()
	config.RunDocker, _ = generalSection.Key(runDockerKey).Bool()

	regionsSection := cfg.Section("regions")
	config.Regions = regionsSection.KeyStrings()

	headersSection := cfg.Section("headers")
	headerHash := headersSection.KeysHash()
	config.Headers = foldHeaders(headerHash)

	return config
}

func applyExtendedConfiguration(config *types.TestConfig) {
	cfg := loadIni()
	if cfg == nil {
		return
	}
	taskSection := cfg.Section("task")
	runnerPathKey, err := taskSection.GetKey("runner")
	if err != nil {
		return
	}
	config.RunnerPath = runnerPathKey.String()
}

func foldHeaders(hash map[string]string) []string {
	headersList := make([]string, 0)
	for k, v := range hash {
		headersList = append(headersList, fmt.Sprintf("%s: %s", k, v))
	}
	return headersList
}

func parseCommandline() *types.TestConfig {
	args := os.Args[1:]

	kingpin.MustParse(app.Parse(args))
	if *writeIni {
		writeIniFile()
		fmt.Printf("Sample configuration written to: %s\n", iniFile)
		os.Exit(0)
	}

	if *url == "" {
		fmt.Println("No URL provided")
		app.Usage(args)
		os.Exit(1)
	}

	regionsArray := parseRegionsForBackwardsCompatibility(*regions)

	config := &types.TestConfig{}
	config.URL = *url
	config.Concurrency = *concurrency
	config.Requests = *requests
	config.Timelimit = *timelimit
	config.Timeout = *timeout
	config.Regions = regionsArray
	config.Method = *method
	config.Body = *body
	config.Headers = *headers
	config.Output = *outputFile
	config.RunDocker = *runDocker
	return config
}

func parseRegionsForBackwardsCompatibility(regions []string) []string {
	parsedRegions := make([]string, 0)
	for _, str := range regions {
		parsedRegions = append(parsedRegions, strings.Split(str, ",")...)
	}
	return parsedRegions
}