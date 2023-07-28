package main

import (
    "context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type Config struct {
    AWSRegion string `yaml:"aws_region"`
    Record string `yaml:"record"`
    HostedZoneId string `yaml:"hosted_zone_id"`
}

type IPResponse struct {
	IP string `json:"ip"`
}

func main() {
    config, err := config()

    if err != nil {
		panic(err)
    }

    ip, err := getIP()

    if err != nil {
		panic(err)
    }

    fmt.Printf(*ip)

    rErr := upsert(config, ip)

    if rErr != nil {
		panic(rErr)
    }
}

func config() (*Config, error) {
    f, err := os.Open("config.yml")
    if err != nil {
        return nil, err
    }

    defer f.Close()

    var cfg Config
    decoder := yaml.NewDecoder(f)
    err = decoder.Decode(&cfg)
    if err != nil {
        return nil, err
    }

    return &cfg, nil
}

func getIP() (*string, error) {
    r, err := http.NewRequest("GET", "https://api.seeip.org/jsonip", nil)

    if err != nil {
        return nil, err
    }

    client := &http.Client{}
    res, err := client.Do(r)

    if err != nil {
        return nil, err
    }

    defer res.Body.Close()

    response := &IPResponse{}
    derr := json.NewDecoder(res.Body).Decode(response)

    if derr != nil {
        return nil, derr
    }

    return &response.IP, nil
}

func upsert(config *Config, value *string) error {
	cfg, err := aws_config.LoadDefaultConfig(
	    context.TODO(),
	    aws_config.WithRegion(config.AWSRegion),
    )

    if err != nil {
        log.Fatalf("unable to load SDK config, %v", err)
    }

	svc := route53.NewFromConfig(cfg)

	params := &route53.ChangeResourceRecordSetsInput{
        ChangeBatch: &types.ChangeBatch{
            Changes: []types.Change{
                {
                    Action: types.ChangeActionUpsert,
                    ResourceRecordSet: &types.ResourceRecordSet{
                        Name: aws.String(config.Record),
                        Type: types.RRTypeA,
                        ResourceRecords: []types.ResourceRecord{
                            {
                                Value: aws.String(*value),
                            },
                        },
                        TTL: aws.Int64(3600),
                    },
                },
            },
        },
        HostedZoneId: aws.String(config.HostedZoneId),
    }

	_, err = svc.ChangeResourceRecordSets(context.TODO(), params)

    return err
}
