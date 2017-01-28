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

type recordSet struct {
	Name         string
	Value        string // ip
	Type         string
	TTL          int64
	HostedZoneId string
}

const progName = "dyndns53"

func main() {
	log.SetPrefix(progName + ": ")
	log.SetFlags(0)

	var err error
	var recSet recordSet

	flag.StringVar(&recSet.Name, "name", "", `record set name; must end with "."`)
	flag.StringVar(&recSet.Type, "type", "A", `record set type; "A" or "AAAA"`)
	flag.Int64Var(&recSet.TTL, "ttl", 300, "TTL (time to live) in seconds")
	flag.StringVar(&recSet.HostedZoneId, "zone", "", "hosted zone id")

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	flag.Parse()

	if recSet.Name == "" {
		log.Fatal("missing record set name")
	}
	if !strings.HasSuffix(recSet.Name, ".") {
		log.Fatal(`record set name must end with a "."`)
	}
	if recSet.Type == "" {
		log.Fatal("missing record set type")
	}
	if recSet.Type != "A" && recSet.Type != "AAAA" {
		log.Fatalf("invalid record set type: %s", recSet.Type)
	}
	if recSet.TTL < 1 {
		log.Fatalf("invalid record set TTL: %d", recSet.TTL)
	}
	if recSet.HostedZoneId == "" {
		log.Fatal("missing hosted zone id")
	}

	recSet.Value, err = getCurrentIP()
	if err != nil {
		log.Fatal(err)
	}

	domain := strings.TrimSuffix(recSet.Name, ".")
	if domainResolvesToIP(domain, recSet.Value) {
		log.Fatalf("%s already resolves to %s; nothing to do", domain, recSet.Value)
	}

	resp, err := upsertRecordSet(&recSet)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp)
}

func getCurrentIP() (string, error) {
	httpResp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	ip := strings.TrimSuffix(string(body), "\n")
	return ip, nil
}

func domainResolvesToIP(domain, checkIP string) bool {
	if ips, err := net.LookupIP(domain); err == nil {
		for _, ip := range ips {
			if ip.String() == checkIP {
				return true
			}
		}
	}
	return false
}

func upsertRecordSet(recSet *recordSet) (*route53.ChangeResourceRecordSetsOutput, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	credentialsPath := path.Join(usr.HomeDir, ".aws", "credentials")
	credentials := credentials.NewSharedCredentials(credentialsPath, progName)

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	svc := route53.New(sess, &aws.Config{Credentials: credentials})

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(recSet.Name),
						Type: aws.String(recSet.Type),
						TTL:  aws.Int64(recSet.TTL),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(recSet.Value),
							},
						},
					},
				},
			},
		},
		HostedZoneId: aws.String(recSet.HostedZoneId),
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
