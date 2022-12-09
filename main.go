package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/davecgh/go-spew/spew"
	clientrun "github.com/go-openapi/runtime/client"
	glog "github.com/martinbaillie/go-graylog/pkg/client"
	"github.com/martinbaillie/go-graylog/pkg/client/search_decorators"
	"github.com/martinbaillie/go-graylog/pkg/models"
)

// TODO: input: better config and validation
// TODO: config for extra fields
// TODO: handle paging
type Config struct {
	GraylogURL string
}

func main() {
	cfg := Config{
		GraylogURL: os.Getenv("GRAYLOG_URL"),
	}
	graylog := glog.NewHTTPClientWithConfig(nil, &glog.TransportConfig{
		Host:     cfg.GraylogURL,
		Schemes:  []string{"https"},
		BasePath: "/api",
	})
	if len(os.Args) != 4 {
		fmt.Println("Usage: logsjira <query> <lookback> <jira key>")
		os.Exit(1)
	}

	query := os.Args[1]
	lookback, err := time.ParseDuration(os.Args[2])
	if err != nil {
		panic(err)
	}
	jiraKey := os.Args[3]

	// TODO: use total results, limit and offset to collect everything
	results, err := graylog.SearchDecorators.SearchAbsolute(search_decorators.NewSearchAbsoluteParamsWithContext(context.Background()).
		WithQuery(query).WithFrom(time.Now().UTC().Add(-lookback).Format("2006-01-02 03:04:05")).WithTo(time.Now().UTC().Format("2006-01-02 03:04:05")),
		clientrun.BasicAuth(os.Getenv("GRAYLOG_USERNAME"), os.Getenv("GRAYLOG_PASSWORD")))
	if err != nil {
		panic(err)
	}

	spew.Dump(results.Payload.TotalResults)

	csvData, err := messagesToCSV(results.Payload.Messages, []string{"solidfire_request", "foobar"})
	if err != nil {
		panic(err)
	}

	tp := jira.BasicAuthTransport{
		Username: os.Getenv("JIRA_USERNAME"),
		Password: os.Getenv("JIRA_PASSWORD"),
	}
	jiras, err := jira.NewClient(tp.Client(), os.Getenv("JIRA_BASE_URL"))
	if err != nil {
		panic(err)
	}
	filename := cfg.GraylogURL + time.Now().Format(time.RFC3339) + ".csv"
	_, _, err = jiras.Issue.PostAttachmentWithContext(context.Background(), jiraKey, csvData, filename)
	if err != nil {
		panic(err)
	}
	searchURL := fmt.Sprintf(`https://%s/search?q=%s&rangetype=relative&relative=%f`,
		cfg.GraylogURL, url.QueryEscape(query), lookback.Seconds())

	// _, _, err = jiras.Issue.AddComment(jiraKey, &jira.Comment{
	// 	Body: searchURL,
	// })
	// if err != nil {
	// 	panic(err)
	// }
	_, _, err = jiras.Issue.AddRemoteLink(jiraKey, &jira.RemoteLink{
		Application: &jira.RemoteLinkApplication{
			Type: "Web",
		},
		Object: &jira.RemoteLinkObject{
			URL:   searchURL,
			Title: fmt.Sprintf("%s - %s", cfg.GraylogURL, query),
		},
	})
	if err != nil {
		panic(err)
	}
}

func messagesToCSV(messages models.SearchResponseMessages, extraFields []string) (io.Reader, error) {
	buffer := []byte{}
	buf := bytes.NewBuffer(buffer)
	w := csv.NewWriter(buf)
	err := w.Write(append([]string{"time", "message", "source"}, extraFields...))
	if err != nil {
		return nil, err
	}

	for _, m := range messages {
		vals := m.Message.(map[string]any)
		record := []string{vals["timestamp"].(string), vals["message"].(string), vals["source"].(string)}
		for _, f := range extraFields {
			if val, ok := vals[f]; ok {
				record = append(record, val.(string))
			}
		}
		err = w.Write(record)
		if err != nil {
			return nil, err
		}
	}
	w.Flush()
	if w.Error() != nil {
		return nil, w.Error()
	}
	return buf, nil
}
