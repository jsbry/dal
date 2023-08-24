package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bwmarrin/discordgo"
)

const (
	DiscordRequestTypePing    = 1
	DiscordRequestTypeCommand = 4
)

type Response struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest(ctx context.Context, event map[string]interface{}) (Response, error) {
	r := Response{
		IsBase64Encoded: false,
		StatusCode:      http.StatusOK,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            "",
	}

	content, err := paramCheck(event)
	if err != nil {
		r.StatusCode = http.StatusUnauthorized
		r.Body = err.Error()
	}
	r.Body = content

	return r, nil
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-type
type PingResponse struct {
	Type int `json:"type"`
}

type CommandResponse struct {
	Type int         `json:"type"`
	Data CommandData `json:"data"`
}

type CommandData struct {
	Content string `json:"content"`
}

func paramCheck(event map[string]interface{}) (string, error) {
	pubKeyBytes, err := hex.DecodeString(os.Getenv("DISCORD_PUBKEY"))
	if err != nil {
		return "", errors.New("err pubkey")
	}
	DiscordPublicKey := ed25519.PublicKey(pubKeyBytes)

	body, ok := event["body"].(string)
	if !ok {
		return "", errors.New("not found body")
	}
	var v interface{}
	err = json.Unmarshal([]byte(body), &v)
	if err != nil {
		return "", errors.New("err json format")
	}
	param, ok := v.(map[string]interface{})
	if !ok {
		return "", errors.New("not found body")
	}
	requestType, ok := param["type"].(float64)
	if !ok {
		return "", errors.New("not found type")
	}

	headers, ok := event["headers"].(map[string]interface{})
	if !ok {
		return "", errors.New("not found headers")
	}

	ed25519, ok := headers["x-signature-ed25519"].(string)
	if !ok {
		return "", errors.New("not found ed25519")
	}

	timestamp, ok := headers["x-signature-timestamp"].(string)
	if !ok {
		return "", errors.New("not found timestamp")
	}

	request, err := http.NewRequest("POST", "", strings.NewReader(body))
	if err != nil {
		return "", errors.New("err request")
	}

	request.Header.Set("X-Signature-Timestamp", timestamp)
	request.Header.Set("X-Signature-Ed25519", ed25519)

	if !discordgo.VerifyInteraction(request, DiscordPublicKey) {
		return "", errors.New("")
	}

	content := ""
	// https://zenn.dev/drumath2237/articles/112fd0bfa7ea4f836195
	if int(requestType) == DiscordRequestTypePing {
		r := PingResponse{
			Type: DiscordRequestTypePing,
		}
		pr, err := json.Marshal(r)
		if err != nil {
			return "", errors.New("err ping response")
		}
		content = string(pr)
	} else {
		data, ok := param["data"].(map[string]interface{})
		if !ok {
			return "", errors.New("not found data")
		}
		options, ok := data["options"].([]interface{})
		if !ok {
			return "", errors.New("not found options")
		}

		if len(options) < 1 {
			return "", errors.New("too little options")
		}
		option := options[0]
		op, ok := option.(map[string]interface{})
		if !ok {
			return "", errors.New("err options")
		}

		optionName, ok := op["name"].(string)
		if !ok {
			return "", errors.New("not found name")
		}
		optionType, ok := op["type"].(float64)
		if !ok {
			return "", errors.New("not found type")
		}
		optionValue, ok := op["value"].(string)
		if !ok {
			return "", errors.New("not found value")
		}

		if optionName == "action" && int(optionType) == 3 {
			switch optionValue {
			case "start":
				content = "server start"
			case "stop":
				content = "server stop"
			case "test":
				content = "server test"
			}
		}

		r := CommandResponse{
			Type: DiscordRequestTypeCommand,
			Data: CommandData{
				Content: content,
			},
		}
		cr, err := json.Marshal(r)
		if err != nil {
			return "", errors.New("err command response")
		}
		content = string(cr)
	}
	return content, nil
}
