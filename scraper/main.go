package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/akamensky/argparse"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
	"github.com/tivo/terraform-provider-splunk-itsi/provider"

	"gopkg.in/yaml.v3"
)

var outputBasePath = "dump"

func main() {
	parser := argparse.NewParser("ITSI scraper", "Dump ITSI resources via REST interface and format them in a file.")

	host := parser.String("t", "host", &argparse.Options{Required: false, Help: "host", Default: "localhost"})
	port := parser.Int("o", "port", &argparse.Options{Required: false, Help: "port", Default: 8089})
	verbose := parser.Selector("v", "verbose", []string{"true", "false"}, &argparse.Options{Required: false, Help: "verbose mode", Default: "false"})
	skipTLS := parser.Selector("s", "skip-tls", []string{"true", "false"}, &argparse.Options{Required: false, Help: "skip TLS check", Default: "false"})
	format := parser.Selector("f", "format", []string{"json", "yaml", "tf", "tfjson"}, &argparse.Options{Required: false, Help: "output format. json|yaml|tf", Default: "yaml"})

	objectTypes := []string{}
	for k := range models.RestConfigs {
		objectTypes = append(objectTypes, k)
	}
	objs := parser.StringList("b", "obj", &argparse.Options{Required: false, Help: "object types", Default: objectTypes})

	userCredCommand := parser.NewCommand("creds", "configure user/password  credential")
	user := userCredCommand.String("u", "user", &argparse.Options{Required: true, Help: "user"})
	password := userCredCommand.String("p", "password", &argparse.Options{Required: true, Help: "password"})

	ssmTokenCommand := parser.NewCommand("token", "Bearer token to retrieve from AWS SSM")
	profile := ssmTokenCommand.String("i", "profile", &argparse.Options{Required: true, Help: "Profile - to retrieve token from AWS SSM"})
	region := ssmTokenCommand.String("r", "region", &argparse.Options{Required: true, Help: "Region - to retrieve token from AWS SSM"})
	tokenPath := ssmTokenCommand.String("l", "path", &argparse.Options{Required: true, Help: "The auth token path from AWS SSM"})

	err := parser.Parse(os.Args)
	if err != nil {
		log.Fatal(parser.Usage(err))
	}
	models.InitItsiApiLimiter(10)
	provider.InitSplunkSearchLimiter(10)

	models.Verbose = (*verbose == "true")
	models.Cache = models.NewCache(1000)
	bearerToken := ""

	if ssmTokenCommand.Happened() {
		bearerToken, err = getTokenFromSSM(*tokenPath, *profile, *region)
		if err != nil {
			log.Fatal(err)
		}
	} else if !userCredCommand.Happened() {
		log.Fatal(parser.Usage("Access should be configured via the user credentials or bearer token."))
	}

	clientConfig := models.ClientConfig{
		BearerToken: bearerToken,
		Host:        *host,
		Port:        *port,
		User:        *user,
		Password:    *password,
		SkipTLS:     (*skipTLS == "true"),
	}

	clientConfig.RetryPolicy = backoff.Exponential(
		backoff.WithMinInterval(500*time.Millisecond),
		backoff.WithMaxInterval(time.Minute),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(3),
	)
	err = os.RemoveAll(outputBasePath)
	if err != nil {
		panic(err)
	}

	logFolder := fmt.Sprintf("%s/%s", outputBasePath, "log")
	err = os.MkdirAll(logFolder, os.ModePerm)
	if err != nil {
		panic(err)
	}
	logFilename := fmt.Sprintf("%s/log.txt", logFolder)
	f, err := os.OpenFile(logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var wg sync.WaitGroup
	logsCh := make(chan Logs, len(*objs))
	for _, objType := range *objs {
		if formatter, ok := provider.Formatters[objType]; ok || !strings.HasPrefix(*format, "tf") {
			wg.Add(1)
			go dump(clientConfig, objType, *format, formatter, logsCh, &wg)
		} else {
			log.Printf("No formatter for %s, skipping...\n", objType)
		}
	}

	go func() {
		wg.Wait()
		close(logsCh)
	}()

	for log := range logsCh {
		_, err = f.WriteString(fmt.Sprintf("Scrapped %s: \n", log.ObjectType))
		if err != nil {
			panic(err)
		}

		if len(log.Errors) == 0 {
			_, err = f.WriteString("Success\n")
			if err != nil {
				panic(err)
			}
		}

		for _, logErr := range log.Errors {
			_, err = f.WriteString(logErr.Error() + "\n")
			if err != nil {
				panic(err)
			}
		}
	}
}

type Logs struct {
	ObjectType string
	Errors     []error
}

func dump(clientConfig models.ClientConfig, objectType, format string, formatter provider.TFFormatter, logsCh chan Logs, wg *sync.WaitGroup) {
	defer wg.Done()

	base := models.NewBase(clientConfig, "", "", objectType)
	errors := []error{}

	fieldsMap := map[string]bool{}
	for count, offset := base.GetPageSize(), 0; offset >= 0; offset += count {
		ctx := context.Background()
		items, err := base.Dump(ctx, offset, count)
		if err != nil {
			errors = append(errors, err)
			break
		}

		_errors := auditLog(items, objectType, format, formatter)
		if len(*_errors) > 0 {
			errors = append(errors, *_errors...)
			continue
		}

		for _, item := range items {
			for _, f := range item.Fields {
				fieldsMap[f] = true
			}
		}

		if len(items) < count {
			break
		}
	}
	err := auditFields(fieldsMap, objectType)
	if err != nil {
		errors = append(errors, err)
	}

	if format == "tf" {
		diags := NewFmtCommand([]string{outputBasePath}, false).Run()
		if diags.HasError() {
			log.Fatalf("%+v", diags)
		}
	}

	logsCh <- Logs{
		ObjectType: objectType,
		Errors:     errors,
	}
}

func auditLog(items []*models.Base, objectType, format string, formatter provider.TFFormatter) *[]error {
	folder := fmt.Sprintf("%s/%s", outputBasePath, format)
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return &[]error{err}
	}
	filename := fmt.Sprintf("%s/%s.%s", folder, objectType, format)

	var by []byte
	errors := []error{}

	switch format {
	case "json":
		objects := []json.RawMessage{}
		for _, item := range items {
			objects = append(objects, json.RawMessage(item.RawJson))
		}

		by, err = json.MarshalIndent(objects, "", "  ")
		if err != nil {
			return &[]error{err}
		}
	case "yaml":
		objects := []interface{}{}
		for _, item := range items {
			by, err := json.Marshal(item.RawJson)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			var i interface{}
			err = json.Unmarshal(by, &i)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			objects = append(objects, i)
		}
		by, err = yaml.Marshal(objects)
	case "tf":
		if formatter == nil {
			return &[]error{fmt.Errorf("formatter for %s is nil for %s format", objectType, format)}
		}
		var objects string
		for _, item := range items {
			out, err := formatter(item)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			objects = strings.Join([]string{objects, out}, "\n")
		}

		by = []byte(objects)
	case "tfjson":
		if formatter == nil {
			return &[]error{fmt.Errorf("formatter for %s is nil for %s format", objectType, format)}
		}
		var objects []json.RawMessage
		for _, item := range items {
			out, err := provider.JSONify(item, formatter)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			objects = append(objects, out)
		}
		by, err = json.Marshal(objects)
		if err != nil {
			errors = append(errors, err)
		}
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &[]error{err}
	}

	defer f.Close()
	_, err = f.Write(by)
	if err != nil {
		return &[]error{err}
	}

	return &errors
}

func auditFields(fieldsMap map[string]bool, objectType string) error {
	folder := fmt.Sprintf("%s/fields", outputBasePath)

	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s/%s.yaml", folder, objectType)

	fields := []string{}
	for field := range fieldsMap {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	by, err := yaml.Marshal(fields)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, by, 0644)
}

func getTokenFromSSM(tokenPath, profile, region string) (accessToken string, err error) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithSharedConfigProfile(profile))
	if err != nil {
		return "", err
	}

	cfg.Region = region
	client := ssm.NewFromConfig(cfg)

	param, err := client.GetParameters(
		context.Background(),
		&ssm.GetParametersInput{
			Names:          []string{tokenPath},
			WithDecryption: true,
		})
	if err != nil {
		return "", err
	}

	secretsInfo := map[string]string{}
	for _, item := range param.Parameters {
		secretsInfo[*item.Name] = *item.Value
	}

	token, ok := secretsInfo[tokenPath]
	if !ok {
		return "", fmt.Errorf("client token not found from SSM")
	}

	return token, nil
}
