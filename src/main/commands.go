package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/nlopes/slack"
)

var clUrlBase = "http://vancouver.craigslist.ca/search/sss?sort=rel&is_paid=all&srchType=T&min_price=50&query=%s"
var foodUrl = "http://data.streetfoodapp.com/1.1/schedule/gnw"

func dumpArgs(buf *strings.Builder, args []string) {
	for i, arg := range args {
		buf.WriteString(fmt.Sprintf("%d\t >", i))
		buf.WriteString(arg)
		buf.WriteString("\n")
	}
}

func registerCommands(cmdManager *CommandManager) {
	userNameEx, _ := regexp.Compile(`(?:<@)(\w+)`)
	channelEx, _ := regexp.Compile(`(?:<#)(\w+)`)

	cmdManager.addCommand(newHandler("sayas", func(event CommandEvent) {
		userId := userNameEx.FindStringSubmatch(event.args[0])
		channelId := channelEx.FindStringSubmatch(event.args[1])
		if len(userId) < 2 || len(channelId) < 2 {
			return
		}

		users, _ := event.api.GetUsers()

		for _, user := range users {
			if strings.Compare(user.ID, userId[1]) == 0 {
				params := slack.PostMessageParameters{}
				params.AsUser = false
				params.User = user.ID
				params.Username = user.RealName
				params.IconURL = user.Profile.Image192
				params.LinkNames = 1
				event.api.PostMessage(channelId[1], event.args[2], params)
			}
		}
	}, newOptions().
		RequiredArgs(3, "@user #channel <message>").
		CaptureAfter(2).
		MatchChannel("^D").
		QuotesEnabled(false).
		Build(),
	))

	cmdManager.addCommand(newHandler("say", func(event CommandEvent) {
		channelId := channelEx.FindStringSubmatch(event.args[0])
		if len(channelId) < 2 {
			return
		}

		event.api.SendMessage(event.api.NewOutgoingMessage(event.args[1], channelId[1]))
	}, newOptions().
		RequiredArgs(2, "#channel <message>").
		MatchChannel("^D").
		CaptureAfter(1).
		Build(),
	))

	cmdManager.addCommand(newHandler("cl", func(event CommandEvent) {
		query := strings.Replace(event.args[0], " ", "+", -1)
		query = fmt.Sprintf(clUrlBase, query)
		event.api.SendMessage(event.api.NewOutgoingMessage(query, event.source.Channel))

		doc, _ := htmlquery.LoadURL(query)

		var costs []int

		for _, n := range htmlquery.Find(doc, "//span/span[@class='result-price']") {
			contents := htmlquery.InnerText(n)
			val, _ := strconv.ParseInt(contents[1:len(contents)], 10, 64)
			costs = append(costs, int(val))
		}

		sort.Ints(costs)

		index := int(math.Ceil(0.5 * float64(len(costs))))
		event.api.SendMessage(event.api.NewOutgoingMessage(fmt.Sprintf("50th percentile price is $%d", costs[index]), event.source.Channel))
	}, newOptions().
		MatchChannel("^D").
		RequiredArgs(1, "\"search query\"").
		Build(),
	))

	cmdManager.addCommand(newHandler("help", func(event CommandEvent) {
		usage := cmdManager.getUsage()
		event.api.SendMessage(event.api.NewOutgoingMessage(usage, event.source.Channel))
	}, newOptions().MatchChannel("^D").Build()))

	cmdManager.addCommand(newHandler("foodtruck", func(event CommandEvent) {
		var obj map[string]interface{}
		response, _ := http.Get(foodUrl)
		jsonData, _ := ioutil.ReadAll(response.Body)
		json.Unmarshal(jsonData, &obj)

		vendors := obj["vendors"].(map[string]interface{})

		var attachments []slack.Attachment

		for key, val := range vendors {
			vendor := val.(map[string]interface{})
			hash := md5.Sum([]byte(key))
			hashString := hex.EncodeToString(hash[:])

			openList := vendor["open"].([]interface{})
			open := openList[0].(map[string]interface{})

			from := time.Unix(int64(open["start"].(float64)), 0)
			to := time.Unix(int64(open["end"].(float64)), 0)

			var attachment slack.Attachment
			attachment.Color = hashString[0:6]

			attachment.Fields = []slack.AttachmentField{
				slack.AttachmentField{
					Title: "Location",
					Value: open["display"].(string),
				},
				slack.AttachmentField{
					Title: "Time",
					Value: fmt.Sprintf("From %s To %s", from.Local().Format("Jan 2 3:04PM"), to.Local().Format("Jan 2 3:04PM")),
				},
				slack.AttachmentField{
					Title: "Website",
					Value: vendor["url"].(string),
				},
			}

			attachments = append(attachments, attachment)
		}

		params := slack.NewPostMessageParameters()
		params.AsUser = true
		params.Attachments = attachments

		event.api.PostMessage(event.source.Channel, "Food trucks at Great Northern Way", params)
	}, newOptions().
		MatchChannel("^C").
		MatchMsg("^Reminder: foodtruck").
		Build(),
	))

}
