package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bwmarrin/discordgo"
)

const (
	DiscordRequestType1 = 1
	DiscordRequestType4 = 4
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
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            "",
	}

	errr := Response{
		IsBase64Encoded: false,
		StatusCode:      401,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            "",
	}

	pubKeyBytes, err := hex.DecodeString(os.Getenv("DISCORD_PUBKEY"))
	if err != nil {
		return errr, nil
	}
	DiscordPublicKey := ed25519.PublicKey(pubKeyBytes)

	body, ok := event["body"].(string)
	if !ok {
		return errr, nil
	}
	var v interface{}
	err = json.Unmarshal([]byte(body), &v)
	if err != nil {
		return errr, nil
	}
	param, ok := v.(map[string]interface{})
	if !ok {
		return errr, nil
	}
	requestType, ok := param["type"].(float64)
	if !ok {
		return errr, nil
	}

	headers, ok := event["headers"].(map[string]interface{})
	if !ok {
		return errr, nil
	}

	ed25519, ok := headers["x-signature-ed25519"].(string)
	if !ok {
		return errr, nil
	}

	timestamp, ok := headers["x-signature-timestamp"].(string)
	if !ok {
		return errr, nil
	}

	request, err := http.NewRequest("POST", "", strings.NewReader(body))
	if err != nil {
		return errr, nil
	}

	request.Header.Set("X-Signature-Timestamp", timestamp)
	request.Header.Set("X-Signature-Ed25519", ed25519)

	if !discordgo.VerifyInteraction(request, DiscordPublicKey) {
		r.StatusCode = 401
		r.Body = ""
		log.Printf("!VerifyInteraction: %#v", r.StatusCode)
		return errr, nil
	}

	log.Print("VerifyInteraction: ok")
	// https://zenn.dev/drumath2237/articles/112fd0bfa7ea4f836195
	if int(requestType) == DiscordRequestType1 {
		r.Body = fmt.Sprintf(`{"type":%d}`, DiscordRequestType1)
	} else {
		data, ok := param["data"].(map[string]interface{})
		if !ok {
			return errr, nil
		}
		options, ok := data["options"].([]interface{})
		if !ok {
			return errr, nil
		}

		if len(options) < 1 {
			return errr, nil
		}
		option := options[0]
		op, ok := option.(map[string]interface{})
		if !ok {
			return errr, nil
		}

		optionName, ok := op["name"].(string)
		if !ok {
			return errr, nil
		}
		optionType, ok := op["type"].(float64)
		if !ok {
			return errr, nil
		}
		optionValue, ok := op["value"].(string)
		if !ok {
			return errr, nil
		}

		content := "ok"
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

		r.Body = fmt.Sprintf(`{"type":%d,"data":{"content":"%s"}}`, DiscordRequestType4, content)
	}
	return r, nil
}
