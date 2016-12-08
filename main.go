package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/nlopes/slack"
)

func closeResponse(resp *http.Response) {
	if resp != nil {
		resp.Body.Close()
	}
}

func getPostMessageParameters(metrics map[string]*mkr.MetricValue) slack.PostMessageParameters {
	params := slack.PostMessageParameters{
		Markdown: true,
	}
	params.Attachments = []slack.Attachment{}

	params.Attachments = append(params.Attachments, slack.Attachment{
		Title: "Load Average",
		Text:  fmt.Sprintf("%0.2f", metrics["loadavg5"].Value),
	})
	params.Attachments = append(params.Attachments, slack.Attachment{
		Title: "CPU User %",
		Text:  fmt.Sprintf("%0.2f%%", metrics["cpu.user.percentage"].Value),
	})
	params.Attachments = append(params.Attachments, slack.Attachment{
		Title: "Memory Usage %",
		Text:  fmt.Sprintf("%0.2f%%", metrics["memory.used"].Value.(float64)/metrics["memory.total"].Value.(float64)),
	})

	return params
}

func main() {
	slackAPIToken := os.Getenv("SLACK_API_TOKEN")

	r := regexp.MustCompile(`^mkr ([a-zA-Z0-9_\-\.]+$)`)

	api := slack.New(slackAPIToken)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				m := r.FindSubmatch([]byte(ev.Text))
				if len(m) != 2 {
					continue
				}

				hostName := string(m[1])
				if ev.Text == fmt.Sprintf("mkr %s", hostName) {
					mackerelAPIKey := os.Getenv("MACKEREL_API_KEY")
					client := myClient{
						mkr.NewClient(mackerelAPIKey),
					}

					hosts, err := client.FindHosts(&mkr.FindHostsParam{
						Name: hostName,
					})
					if err != nil {
						log.Fatal(err)
					}

					if len(hosts) == 0 {
						continue
					}

					host := *hosts[0]
					metrics, err := client.fetchLatestMetricValues(host)

					params := getPostMessageParameters(metrics)
					_, _, err = api.PostMessage(ev.Channel, "*"+host.Name+"*", params)
					if err != nil {
						fmt.Printf("%s\n", err)
						return
					}
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// Ignore other events..
			}
		}
	}
}
