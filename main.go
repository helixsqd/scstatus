package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"html"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SERVER_NUM_STR = "%d"

var username *string
var password *string
var sortField *string
var urls = make([]string, 0, 100)

// holds the resulting entries after fetching and parsing
var entries []*Entry = nil

func main() {
	sortField = flag.String("s", "host", "Field to sort on: [host, uri, time, ip]")
	username = flag.String("u", "", "Username")
	password = flag.String("p", "", "Password")
	var timeout = flag.Uint("t", uint(3000), "Request timeout")
	var startHttpServer = flag.Bool("w", false, "Start HTTP server on port 8000")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] server1 [server2]...[serverN]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// initial hosts entries that still may need %d exploding
	baseHosts := flag.Args()
	// read line by line entries from stdin if not a terminal
	if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println(err)
		} else {
			hostsStr := string(bytes)
			baseHosts = append(baseHosts, strings.Split(hostsStr, "\n")...)
		}
	}
	// set http default timeout
	http.DefaultClient.Timeout = time.Millisecond * time.Duration(*timeout)

	// resolve entries containing %d
	for _, host := range baseHosts {
		urls = append(urls, resolveDNS(host)...)
	}

	// no urls to grab, show usage
	if len(urls) == 0 {
		flag.Usage()
		return
	}
	gatherData()
	if *startHttpServer {
		fmt.Println("Starting server on port " + httpPort)
		startServer()
	}
}

// primary function that kicks off fetching, processing, and printing of status data
func gatherData() {
	entries = make([]*Entry, 0, 100)
	var wg = new(sync.WaitGroup)
	urlChan := make(chan string)
	resultChans := make([]chan *Entry, 0, 100)

	// start 100 fetchers, get their result chans to listen on
	for i := 0; i < 100; i++ {
		wg.Add(1)
		resultChan := fetcher(urlChan, wg)
		resultChans = append(resultChans, resultChan)
	}

	// add urls to chan for fetchers to process
	for i := 0; i < len(urls); i++ {
		urlChan <- urls[i]
	}
	close(urlChan)
	wg.Wait()

	// listen for results
	for i := 0; i < len(resultChans); i++ {
		for entry := range resultChans[i] {
			entries = append(entries, entry)
		}
	}

	// sort and print
	if len(entries) > 0 {
		Sort(*sortField, entries)
		fmt.Printf("%-30.28s%-40.38s%-11.9s%-18.16s\n", "Host", "URI", "Time(ms)", "Remote IP")
		for _, e := range entries {
			printEntry(e)
		}
	}
}

func printEntry(e *Entry) {
	fmt.Printf("%-30.28s%-40.38s%-11.9s%-18.16s\n", e.Host, e.Attrs["uri"], e.Attrs["requestProcessingTime"], e.Attrs["remoteAddr"])
}

// starts a go func listening on the urlChan and sending results on the returned channel
func fetcher(urlChan chan string, wg *sync.WaitGroup) chan *Entry {
	resChan := make(chan *Entry, 0)

	// runs in bg until no urls are left to process
	go func() {
		// close result channel when done
		defer func() {
			close(resChan)
		}()
		entries := make([]*Entry, 0, 100)
		for tcUrl := range urlChan {
			entries = append(entries, fetchStatus(tcUrl)...)
		}
		wg.Done()

		for _, e := range entries {
			if e != nil {
				resChan <- e
			}
		}
	}()

	return resChan
}

// given a tomcat url, fetch the status and process it
func fetchStatus(tcUrl string) []*Entry {
	// we support entries like: server1.prod:8080 so we need to tack on a scheme
	if !strings.Contains(tcUrl, "://") {
		tcUrl = "http://" + tcUrl
	}
	url, err := url.Parse(tcUrl)
	if err != nil {
		fmt.Printf("Error in host %v  %v\n", tcUrl, err)
		return nil
	}
	if url.Path == "" {
		// add default status URL if we have no path
		tcUrl += "/manager/status"
	}
	status := fetchStatusXML(tcUrl)
	entries := processStatus(url.Host, status)
	return entries
}

// parses a single status XML
func processStatus(host string, statusXML string) []*Entry {
	entries := make([]*Entry, 0, 100)
	// can switch to XML processing packages if we need more complete XML handling
	// TODO this should be compiled once and reused, move into tomcat specific package
	regex, err := regexp.Compile("<worker\\s+[^>]+")
	if err != nil {
		fmt.Printf("Error compiling regex %v\n", err)
		return entries
	}

	matches := regex.FindAllString(statusXML, 10000)

	if len(matches) < 1 {
		fmt.Printf("Unknown status format from %v\n", host)
	}

	for i := 0; i < len(matches); i++ {
		entryAttrs := processEntry(matches[i])
		// check if the line is a "processing" entry
		if _, ok := entryAttrs["uri"]; ok {
			e := new(Entry)
			e.Attrs = entryAttrs
			e.Host = host
			entries = append(entries, e)
		}
	}
	return entries
}

// given a single <worker> xml tag, parse it into an entry
func processEntry(entry string) map[string]string {
	var attrs map[string]string
	entry = html.UnescapeString(entry)
	uri := parseAttr(entry, "currentUri")
	// if uri == ? then its an idle TC thread
	if uri != "?" {
		attrs = map[string]string{}
		attrs["uri"] = uri
		attrs["requestProcessingTime"] = parseAttr(entry, "requestProcessingTime")
		attrs["remoteAddr"] = parseAttr(entry, "remoteAddr")
	}
	return attrs
}

// parse an attribute from a <worker> tag
func parseAttr(s string, attr string) string {
	regex, _ := regexp.Compile(attr + "=\"([^\"]+)")
	return regex.FindStringSubmatch(s)[1]
}

// fetches the status XML from the supplied url
func fetchStatusXML(tcUrl string) string {
	req, err := http.NewRequest("GET", tcUrl+"?XML=true&XML=true", nil)
	if err != nil {
		fmt.Printf("Error fetching from %v: %v\n", tcUrl, err)
		return ""
	}
	if *username != "" {
		req.SetBasicAuth(*username, *password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error fetching from %v: %v\n", tcUrl, err)
		return ""
	}

	if resp.StatusCode != 200 {
		fmt.Printf("Failure getting status from %v: %v\n", tcUrl, resp.Status)
	}
	if resp.StatusCode == 200 && resp.Body != nil {
		contents, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil && contents != nil {
			return string(contents)
		} else if err != nil {
			fmt.Printf("Error fetching from %v: %v\n", tcUrl, err)
		}
	}
	return ""
}

// explode entries containing %d using DNS lookups
func resolveDNS(e string) []string {
	entries := make([]string, 0, 100)
	if e == "" {
		return entries
	}
	if strings.Contains(e, SERVER_NUM_STR) {
		for i := 1; i < 1000; i++ {
			newUrl := strings.Replace(e, SERVER_NUM_STR, strconv.Itoa(i), -1)
			host := parseHostname(newUrl)
			addrs, err := net.LookupHost(host)
			if err == nil && len(addrs) > 0 {
				entries = append(entries, newUrl)
			} else {
				// lookup failed, stop looking
				break
			}
		}
	} else {
		entries = append(entries, e)
	}

	return entries
}

// return hostname out of a url string for DNS lookup
func parseHostname(urlStr string) string {
	url, err := url.Parse(urlStr)
	host := ""
	if err == nil && url != nil {
		if strings.Contains(url.Host, ":") {
			host = strings.Split(url.Host, ":")[0]
		} else {
			host = url.Host
		}
	}

	return host
}
