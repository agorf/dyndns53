package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

const progName = "dyndns"

func main() {
	log.SetPrefix(progName + ": ")
	log.SetFlags(0)

	var recordSetName, recordSetType, hostedZoneId string
	var recordSetTTL int64

	flag.StringVar(&recordSetName, "name", "", `record set name; must end with "."`)
	flag.StringVar(&recordSetType, "type", "A", `record set type; "A" or "AAAA"`)
	flag.Int64Var(&recordSetTTL, "ttl", 300, "TTL (time to live) in seconds")
	flag.StringVar(&hostedZoneId, "zone", "", "hosted zone id")

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	flag.Parse()

	if recordSetName == "" {
		log.Fatal("missing record set name")
	}
	if recordSetName[len(recordSetName)-1:] != "." {
		log.Fatal(`record set name must end with a "."`)
	}
	if recordSetType == "" {
		log.Fatal("missing record set type")
	}
	if recordSetType != "A" && recordSetType != "AAAA" {
		log.Fatalf("invalid record set type: %s", recordSetType)
	}
	if recordSetTTL < 1 {
		log.Fatalf("invalid record set TTL: %d", recordSetTTL)
	}
	if hostedZoneId == "" {
		log.Fatal("missing hosted zone id")
	}

	httpResp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	currentIP := strings.TrimSuffix(string(body), "\n")

	domain := strings.TrimSuffix(recordSetName, ".")
	ips, err := net.LookupIP(domain)
	if err != nil {
		log.Fatal(err)
	}
	for _, ip := range ips {
		if ip.String() == currentIP {
			log.Fatalf("%s already resolves to %s; nothing to do", domain, currentIP)
		}
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	homeDir := usr.HomeDir
	credentials := credentials.NewSharedCredentials(path.Join(homeDir, ".aws", "credentials"), progName)

	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	svc := route53.New(sess, &aws.Config{Credentials: credentials})

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(recordSetName),
						Type: aws.String(recordSetType),
						TTL:  aws.Int64(recordSetTTL),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(currentIP),
							},
						},
					},
				},
			},
			Comment: aws.String(progName),
		},
		HostedZoneId: aws.String(hostedZoneId),
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp)
}
