package main

import (
	"flag"
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/travisperson/dydo-dns/dydosyncer"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func getIpAddr() (string, error) {
	res, _ := http.Get("https://api.ipify.org")
	ipRaw, _ := ioutil.ReadAll(res.Body)

	if net.ParseIP(string(ipRaw)) == nil {
		return "", fmt.Errorf("Failed to parse IP response")
	}

	return string(ipRaw), nil
}

func main() {
	var token = flag.String("token", "", "Digital Ocean API token")
	var domain = flag.String("domain", "", "Domain")
	var rtype = flag.String("record-type", "A", "Record type")
	var rname = flag.String("record-name", "@", "Record name")
	var frequency = flag.Int("frequency", 5, "Frequency in seconds to sync")

	flag.Parse()

	frequency_duration := time.Duration(*frequency) * time.Second

	tokenSource := &TokenSource{
		AccessToken: *token,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	dydo := dydosyncer.NewDydoSyncer(*domain, *rtype, *rname, client, frequency_duration)

	for {
		ip, err := getIpAddr()
		if err != nil {
			fmt.Printf("[%s] Failed to retrieve IP\n", *domain)
			time.Sleep(frequency_duration)
			continue
		}

		fmt.Printf("[%s] External IP [%s]\n", *domain, ip)

		changed, last, err := dydo.Sync(ip)

		if err != nil {
			fmt.Println(err)
			break
		}

		if changed == true {
			fmt.Printf("[%s] Updated IP Address [%s] => [%s]\n", *domain, last, ip)
		}

		time.Sleep(frequency_duration)
	}
}
