package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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
	name         string
	value        string // ip
	rsType       string
	ttl          int64
	hostedZoneID string
}

const (
	progName   = "dyndns53"
	ipFileName = "." + progName + "-ip"
)

func main() {
	log.SetPrefix(progName + ": ")
	log.SetFlags(0)

	var recSet recordSet
	var logFn string
	flag.StringVar(&recSet.name, "name", "", "record set name (domain)")
	flag.StringVar(&recSet.rsType, "type", "A", `record set type; "A" or "AAAA"`)
	flag.Int64Var(&recSet.ttl, "ttl", 300, "TTL (time to live) in seconds")
	flag.StringVar(&recSet.hostedZoneID, "zone", "", "hosted zone id")
	flag.StringVar(&logFn, "log", "", "file name to log to (default is stdout)")
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	flag.Parse()

	recSet.name = strings.TrimSuffix(recSet.name, ".") + "." // append . if missing
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

	ip, err := currentIPAddress()
	if err != nil {
		log.Fatal(err)
	}

	if ip == lastIPAddress() {
		log.Printf("current IP address is %s; nothing to do", ip)
		os.Exit(0)
	}

	recSet.value = ip
	_, err = recSet.upsert()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("current IP address is %s; upsert request sent", ip)

	if err := updateLastIPAddress(ip); err != nil {
		log.Fatal(err)
	}
}

func currentIPAddress() (string, error) {
	resp, err := http.Get("http://checkip.amazonaws.com/")
	if err != nil {
		return "", fmt.Errorf("currentIPAddress: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("currentIPAddress: %v", err)
	}
	ip := strings.TrimSpace(string(body))
	return ip, nil
}

func lastIPAddress() string {
	data, err := ioutil.ReadFile(ipFileName)
	if err != nil {
		return ""
	}
	return string(data)
}

func updateLastIPAddress(ip string) error {
	if err := ioutil.WriteFile(ipFileName, []byte(ip), 0644); err != nil {
		return fmt.Errorf("updateLastIPAddress: %v", err)
	}
	return nil
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
						Name: aws.String(rs.name),
						Type: aws.String(rs.rsType),
						TTL:  aws.Int64(rs.ttl),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(rs.value),
							},
						},
					},
				},
			},
		},
		HostedZoneId: aws.String(rs.hostedZoneID),
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	if err != nil {
		return nil, fmt.Errorf("(*recordSet).upsert: %v", err)
	}
	return resp, nil
}

func (rs *recordSet) validate() error {
	if rs.name == "" {
		return fmt.Errorf("missing record set name")
	}
	if !strings.HasSuffix(rs.name, ".") {
		return fmt.Errorf(`record set name must end with a "."`)
	}
	if rs.rsType == "" {
		return fmt.Errorf("missing record set type")
	}
	if rs.rsType != "A" && rs.rsType != "AAAA" {
		return fmt.Errorf("invalid record set type: %s", rs.rsType)
	}
	if rs.ttl < 1 {
		return fmt.Errorf("invalid record set TTL: %d", rs.ttl)
	}
	if rs.hostedZoneID == "" {
		return fmt.Errorf("missing hosted zone id")
	}
	return nil
}
