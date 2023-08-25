package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bwmarrin/discordgo"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

	content, err := paramCheck(ctx, event)
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

func paramCheck(ctx context.Context, event map[string]interface{}) (string, error) {
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
			ec2client, err := getEC2Client(ctx)
			if err != nil {
				return "", errors.New("err ec2client")
			}

			instanceID := os.Getenv("INSTANCE_ID")
			switch optionValue {
			case "start":
				result, err := ec2client.StartInstances(ctx, &ec2.StartInstancesInput{
					InstanceIds: []string{instanceID},
				})
				if err != nil {
					content = "server couldn't be started."
				} else {
					content = fmt.Sprintf("server started: %#v", result)
				}
			case "stop":
				result, err := ec2client.StopInstances(ctx, &ec2.StopInstancesInput{
					InstanceIds: []string{instanceID},
				})
				if err != nil {
					content = "server couldn't be stoped."
				} else {
					content = fmt.Sprintf("server stoped: %#v", result)
				}
			case "test":
				result, err := ec2client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
					InstanceIds: []string{instanceID},
				})
				if err != nil {
					content = "server couldn't be stoped."
				} else {
					content = "server stoped."
				}
				content = fmt.Sprintf("server test: %#v", result)
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

func getEC2Client(ctx context.Context) (*ec2.Client, error) {
	region := os.Getenv("REGION")

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	ec2client := ec2.NewFromConfig(cfg)

	return ec2client, nil
}
