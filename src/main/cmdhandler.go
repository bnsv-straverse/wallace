package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

type CommandEvent struct {
	source CommandSource
	args   []string
	api    *slack.RTM
}

type CommandCallback func(event CommandEvent)

type CommandHandler struct {
	cmd      string
	callback CommandCallback
	options  *CommandOptions
}

type CommandOptions struct {
	captureAfter  int
	quotesEnabled bool
	minArgs       int
	usage         string
	matchChannels []*regexp.Regexp
	matchMessages []*regexp.Regexp
	customOnly    bool
}

type CommandOptionsBuilder struct {
	options *CommandOptions
}

func (b *CommandOptionsBuilder) CaptureAfter(after int) *CommandOptionsBuilder {
	b.options.captureAfter = after
	return b
}

func (b *CommandOptionsBuilder) QuotesEnabled(enabled bool) *CommandOptionsBuilder {
	b.options.quotesEnabled = enabled
	return b
}

func (b *CommandOptionsBuilder) RequiredArgs(min int, usage string) *CommandOptionsBuilder {
	b.options.minArgs = min
	b.options.usage = usage
	return b
}

func (b *CommandOptionsBuilder) MatchChannel(match string) *CommandOptionsBuilder {
	regex, err := regexp.Compile(match)
	if err == nil {
		b.options.matchChannels = append(b.options.matchChannels, regex)
	} else {
		fmt.Printf("Invalid regexp '%s'\n", match)
	}
	return b
}

func (b *CommandOptionsBuilder) MatchMsg(match string) *CommandOptionsBuilder {
	regex, err := regexp.Compile(match)
	if err == nil {
		b.options.matchMessages = append(b.options.matchMessages, regex)
	} else {
		fmt.Printf("Invalid regexp '%s'\n", match)
	}
	return b
}

func (b *CommandOptionsBuilder) MatchMsgOnly(enabled bool) *CommandOptionsBuilder {
	b.options.customOnly = enabled
	return b
}

func (b *CommandOptionsBuilder) Build() *CommandOptions {
	if b.options.captureAfter != -1 || b.options.minArgs != -1 {
		if b.options.customOnly || len(b.options.matchMessages) > 0 {
			fmt.Printf("Command requires arguments and cannot have message matching, this feature will be disabled\n")
		}
		b.options.matchMessages = []*regexp.Regexp{}
		b.options.customOnly = false
	}
	return b.options
}

func newOptions() *CommandOptionsBuilder {
	options := CommandOptions{}
	options.captureAfter = -1
	options.quotesEnabled = true
	options.minArgs = -1
	options.usage = ""
	options.customOnly = false

	builder := CommandOptionsBuilder{}
	builder.options = &options
	return &builder
}

func (ch *CommandHandler) execute(event CommandEvent) {
	ch.callback(event)
}

func newHandler(cmd string, callback CommandCallback, options *CommandOptions) *CommandHandler {
	handler := new(CommandHandler)
	handler.cmd = strings.ToLower(cmd)
	handler.callback = callback
	handler.options = options
	return handler
}
