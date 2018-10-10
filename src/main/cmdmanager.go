package main

import (
	"strings"
	"unicode"

	"github.com/nlopes/slack"
)

type CommandManager struct {
	handlers []*CommandHandler
}

type CommandSource *slack.MessageEvent

func (manager *CommandManager) addCommand(handler *CommandHandler) {
	manager.handlers = append(manager.handlers, handler)
}

func (manager *CommandManager) getHandler(cmd string) *CommandHandler {
	for _, handler := range manager.handlers {
		if strings.Compare(handler.cmd, strings.ToLower(cmd)) == 0 {
			return handler
		}
	}
	return nil
}

func (manager *CommandManager) getUsage() string {
	var buf strings.Builder

	buf.WriteString("Usage:\n")
	for _, handler := range manager.handlers {
		if handler.options.customOnly {
			continue
		}
		buf.WriteString(".")
		buf.WriteString(handler.cmd)
		buf.WriteString(" ")
		buf.WriteString(handler.options.usage)
		buf.WriteString("\n")
	}
	return buf.String()
}

func (manager *CommandManager) execute(api *slack.RTM, event CommandSource) {
	inCommand := false
	inArgs := false
	inArg := false
	inQuote := false

	var handler *CommandHandler = nil

	var command strings.Builder
	var arg strings.Builder

	var args []string

	for _, h := range manager.handlers {
		if len(h.options.matchMessages) > 0 {
			for _, regexp := range h.options.matchMessages {
				if regexp.MatchString(event.Msg.Text) {
					cmdEvent := CommandEvent{}
					cmdEvent.api = api
					cmdEvent.source = event
					h.execute(cmdEvent)
					return
				}
			}
		}
	}

	for _, char := range event.Msg.Text {
		if !(inCommand || inArgs) {
			if char == '.' {
				inCommand = true
				continue
			} else if !unicode.IsSpace(char) {
				return
			}
		}

		if inCommand {
			if unicode.IsSpace(char) {
				inCommand = false
				inArgs = true
				handler = manager.getHandler(command.String())
				if handler == nil {
					return
				}
				continue
			}
			command.WriteRune(char)
		}

		if inArgs {
			// Commands can be configured to capture all remaining text after some
			// argument as the final argument
			capture := handler.options.captureAfter
			if capture != -1 && len(args) >= capture {
				inArg = true
				arg.WriteRune(char)
			} else if !unicode.IsSpace(char) || inQuote {
				inArg = true
				if char == '"' && handler.options.quotesEnabled {
					inQuote = !inQuote
					continue
				}
				arg.WriteRune(char)
			} else if inArg {
				inArg = false

				args = append(args, arg.String())
				arg.Reset()
			}
		}
	}

	if inCommand {
		handler = manager.getHandler(command.String())
	}

	if inArg {
		args = append(args, arg.String())
	}

	if handler == nil {
		return
	}

	if handler.options.customOnly {
		return
	}

	if len(handler.options.matchChannels) > 0 {
		matches := false
		for _, match := range handler.options.matchChannels {
			if match.MatchString(event.Channel) {
				matches = true
			}
		}
		if !matches {
			return
		}
	}

	cmdEvent := CommandEvent{}
	cmdEvent.api = api
	cmdEvent.source = event
	cmdEvent.args = args

	if len(args) < handler.options.minArgs {
		var buf strings.Builder
		buf.WriteString("Usage: .")
		buf.WriteString(handler.cmd)
		buf.WriteString(" ")
		buf.WriteString(handler.options.usage)
		api.SendMessage(api.NewOutgoingMessage(buf.String(), event.Channel))
		return
	}
	handler.execute(cmdEvent)
}

func (manager *CommandManager) getCommandCount() int {
	return len(manager.handlers)
}
