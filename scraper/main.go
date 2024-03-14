package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"text/template"
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

const (
	initConfTemplate = `
terraform {
	required_providers {
		itsi = {
			source  = "TiVo/splunk-itsi"
			version = "~> 1.0"
		}
	}
	required_version = "~> 1.7.0"
}

provider "itsi" {
	{{if .BearerToken}}
	access_token = "{{.BearerToken}}"
	{{else}}
	password = "{{.Password}}"
	user     = "{{.User}}"
	{{end}}
	host     = "{{.Host}}"
	port     = {{.Port}}
}`
	importSchemaTemplate = `
import {
	id = "{{.Id}}"
	to = {{.ObjectType}}.{{escape .TFID}}
}
`
)

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

	mode := parser.Selector("m", "mode", []string{"scraper", "terraform_gen"}, &argparse.Options{Required: false, Help: "scrape_mode", Default: "scraper"})

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
		if *mode == "terraform_gen" {
			wg.Add(1)
			go runGenerateCommand(clientConfig, objType, logsCh, &wg)
		} else if formatter, ok := provider.Formatters[objType]; ok || !strings.HasPrefix(*format, "tf") {
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

	logger := &DualLogger{}
	logger.New(f)

	for log := range logsCh {
		logger.Print(fmt.Sprintf("Scrapped %s: \n", log.ObjectType))

		if len(log.Errors) == 0 {
			logger.Print("Success\n")
		}

		for _, logErr := range log.Errors {
			logger.Print(logErr.Error() + "\n")
		}
	}
}

type DualLogger struct {
	fileLogger *log.Logger
	stdLogger  *log.Logger
}

func (cl *DualLogger) New(f *os.File) {
	cl.fileLogger = log.New(f, "", log.Ldate|log.Ltime|log.Lshortfile)
	cl.stdLogger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func (cl *DualLogger) Print(msg string) {
	cl.fileLogger.Println(msg)
	cl.stdLogger.Println(msg)
}

type Logs struct {
	ObjectType string
	Errors     []error
}

func runTerraformCommand(folder string, args ...string) (errors []error) {
	cmd := exec.Command("terraform", args...)
	var b bytes.Buffer
	cmd.Stderr = &b
	cmd.Stdout = os.Stdout
	cmd.Dir = folder
	err := cmd.Run()
	errors = append(errors, fmt.Errorf(string(b.Bytes())))
	if err != nil {
		errors = append(errors, err)
	}
	return
}

func runGenerateCommand(clientConfig models.ClientConfig, objectType string, logsCh chan Logs, wg *sync.WaitGroup) {
	defer wg.Done()
	errors := []error{}

	base := models.NewBase(clientConfig, "", "", objectType)

	// Parse Templates for import.tf & versions.tf
	importT := template.Must(template.New("import").Funcs(template.FuncMap{
		"escape": provider.Escape,
	}).Parse(importSchemaTemplate))
	initT := template.Must(template.New("init").Parse(initConfTemplate))

	// Make a folder dump/objectType
	folder := fmt.Sprintf("%s/%s", outputBasePath, objectType)
	err := os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		logsCh <- Logs{
			ObjectType: objectType,
			Errors:     []error{err},
		}
		return
	}

	versions_fn := fmt.Sprintf("%s/versions.tf", folder)
	versionsF, err := os.OpenFile(versions_fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	defer versionsF.Close()

	if err != nil {
		logsCh <- Logs{
			ObjectType: objectType,
			Errors:     []error{err},
		}
		return
	}

	err = initT.Execute(versionsF, clientConfig)
	if err != nil {
		logsCh <- Logs{
			ObjectType: objectType,
			Errors:     []error{err},
		}
		return
	}

	import_fn := fmt.Sprintf("%s/import.tf", folder)
	importF, err := os.OpenFile(import_fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	defer importF.Close()

	if err != nil {
		logsCh <- Logs{
			ObjectType: objectType,
			Errors:     []error{err},
		}
		return
	}

	errors = append(errors, runTerraformCommand(folder, "init")...)

	for count, offset := base.GetPageSize(), 0; offset >= 0; offset += count {
		ctx := context.Background()
		items, err := base.Dump(ctx, &models.Parameters{Offset: offset, Count: count, Fields: []string{"_key", "title"}})
		if err != nil {
			errors = append(errors, err)
			break
		}
		for _, item := range items {
			err = importT.Execute(importF, map[string]string{
				"Id":         item.RESTKey,
				"TFID":       item.TFID,
				"ObjectType": "itsi_" + objectType,
			})
			if err != nil {
				errors = append(errors, err)
			}
		}

		if len(items) < count {
			break
		}
	}
	errors = append(errors, runTerraformCommand(folder, "plan", "-generate-config-out=generated.tf")...)
	diags := NewFmtCommand([]string{folder + "/generated.tf"}, false).Run()

	if diags.HasError() {
		for _, diag := range diags {
			errors = append(errors, fmt.Errorf(diag.Detail))
		}
	}

	logsCh <- Logs{
		ObjectType: objectType,
		Errors:     errors,
	}

}
func dump(clientConfig models.ClientConfig, objectType, format string, formatter provider.TFFormatter, logsCh chan Logs, wg *sync.WaitGroup) {
	defer wg.Done()

	base := models.NewBase(clientConfig, "", "", objectType)
	errors := []error{}

	fieldsMap := map[string]bool{}
	for count, offset := base.GetPageSize(), 0; offset >= 0; offset += count {
		ctx := context.Background()
		items, err := base.Dump(ctx, &models.Parameters{Offset: offset, Count: count, Fields: nil})
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
		if err != nil {
			errors = append(errors, err)
		}
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
	return os.WriteFile(filename, by, 0644)
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
	decryption := true

	param, err := client.GetParameters(
		context.Background(),
		&ssm.GetParametersInput{
			Names:          []string{tokenPath},
			WithDecryption: &decryption,
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
