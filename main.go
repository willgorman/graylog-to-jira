package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"os"

	"github.com/andygrunwald/go-jira"
	clientrun "github.com/go-openapi/runtime/client"
	glog "github.com/martinbaillie/go-graylog/pkg/client"
	"github.com/martinbaillie/go-graylog/pkg/client/search_decorators"
	"github.com/martinbaillie/go-graylog/pkg/models"
)

// TODO: generate a filename
// TODO: post a comment with a link to the search
// TODO: input: search query, fields, jira #, endpoint

func main() {
	graylog := glog.NewHTTPClientWithConfig(nil, &glog.TransportConfig{
		Host:     os.Getenv("GRAYLOG_URL"),
		Schemes:  []string{"https"},
		BasePath: "/api",
	})

	results, err := graylog.SearchDecorators.SearchAbsolute(search_decorators.NewSearchAbsoluteParamsWithContext(context.Background()).
		WithQuery("backfill").WithFrom("2022-11-30 23:02:29").WithTo("2022-11-30 23:06:29"),
		clientrun.BasicAuth("nbs", os.Getenv("GRAYLOG_PASSWORD")))
	if err != nil {
		panic(err)
	}

	csvData, err := messagesToCSV(results.Payload.Messages)
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
	_, _, err = jiras.Issue.PostAttachmentWithContext(context.Background(), "NBS-1728", csvData, "test.csv")
	if err != nil {
		panic(err)
	}
	// spew.Dump(*attachment[])
}

func messagesToCSV(messages models.SearchResponseMessages) (io.Reader, error) {
	buffer := []byte{}
	buf := bytes.NewBuffer(buffer)
	// rw := bufio.NewReadWriter(bytes.NewBuffer(buffer), (*bufio.Writer)(bytes.NewBuffer(input)))
	w := csv.NewWriter(buf)
	err := w.Write([]string{"time", "message", "source"})
	if err != nil {
		return nil, err
	}

	for _, m := range messages {
		vals := m.Message.(map[string]any)
		err = w.Write([]string{vals["timestamp"].(string), vals["message"].(string), vals["source"].(string)})
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
