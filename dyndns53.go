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

	var recSet recordSet
	var logFn string
	flag.StringVar(&recSet.Name, "name", "", "record set name (domain)")
	flag.StringVar(&recSet.Type, "type", "A", `record set type; "A" or "AAAA"`)
	flag.Int64Var(&recSet.TTL, "ttl", 300, "TTL (time to live) in seconds")
	flag.StringVar(&recSet.HostedZoneId, "zone", "", "hosted zone id")
	flag.StringVar(&logFn, "log", "", "file name to log to (default is stdout)")
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	flag.Parse()

	recSet.Name = strings.TrimSuffix(recSet.Name, ".") + "." // append . if missing
	if err := recSet.validate(); err != nil {
		log.Fatal(err)
	}

	if logFn != "" {
		f, err := os.OpenFile(logFn, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("log file: %v", err)
		}
		defer f.Close()

		log.SetFlags(log.LstdFlags) // restore standard flags
		log.SetOutput(f)            // log to file
	}

	var err error
	recSet.Value, err = getCurrentIP()
	if err != nil {
		log.Fatal(err)
	}

	domain := strings.TrimSuffix(recSet.Name, ".")
	if domainResolvesToIP(domain, recSet.Value) {
		log.Fatalf("%s already resolves to %s; nothing to do", domain, recSet.Value)
	}

	resp, err := recSet.upsert()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(resp)
}

func getCurrentIP() (string, error) {
	resp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		return "", fmt.Errorf("getCurrentIP: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("getCurrentIP: %v", err)
	}
	ip := strings.TrimSpace(string(body))
	return ip, nil
}

func domainResolvesToIP(domain, checkIP string) bool {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.String() == checkIP {
			return true
		}
	}
	return false
}

func (rs *recordSet) upsert() (*route53.ChangeResourceRecordSetsOutput, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("(*recordSet).upsert: %v", err)
	}
	credentialsPath := path.Join(usr.HomeDir, ".aws", "credentials")
	credentials := credentials.NewSharedCredentials(credentialsPath, progName)

	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("(*recordSet).upsert: %v", err)
	}

	svc := route53.New(sess, &aws.Config{Credentials: credentials})

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(rs.Name),
						Type: aws.String(rs.Type),
						TTL:  aws.Int64(rs.TTL),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(rs.Value),
							},
						},
					},
				},
			},
		},
		HostedZoneId: aws.String(rs.HostedZoneId),
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	if err != nil {
		return nil, fmt.Errorf("(*recordSet).upsert: %v", err)
	}

	return resp, nil
}

func (rs *recordSet) validate() error {
	if rs.Name == "" {
		return fmt.Errorf("missing record set name")
	}

	if !strings.HasSuffix(rs.Name, ".") {
		return fmt.Errorf(`record set name must end with a "."`)
	}

	if rs.Type == "" {
		return fmt.Errorf("missing record set type")
	}

	if rs.Type != "A" && rs.Type != "AAAA" {
		return fmt.Errorf("invalid record set type: %s", rs.Type)
	}

	if rs.TTL < 1 {
		return fmt.Errorf("invalid record set TTL: %d", rs.TTL)
	}

	if rs.HostedZoneId == "" {
		return fmt.Errorf("missing hosted zone id")
	}

	return nil
}
