package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/schollz/logger"
)

var gateways = []string{
	"https://ipfs.io/ipfs/",
	"https://gateway.ipfs.io/ipfs/",
	"https://ipfs.infura.io/ipfs/",
	"https://rx14.co.uk/ipfs/",
	"https://ninetailed.ninja/ipfs/",
	"https://upload.global/ipfs/",
	"https://ipfs.globalupload.io/ipfs/",
	"https://ipfs.jes.xxx/ipfs/",
	"https://catalunya.network/ipfs/",
	"https://siderus.io/ipfs/",
	"https://eu.siderus.io/ipfs/",
	"https://na.siderus.io/ipfs/",
	"https://ap.siderus.io/ipfs/",
	"https://ipfs.eternum.io/ipfs/",
	"https://hardbin.com/ipfs/",
	"https://ipfs.macholibre.org/ipfs/",
	"https://ipfs.works/ipfs/",
	"https://ipfs.wa.hle.rs/ipfs/",
	"https://api.wisdom.sh/ipfs/",
	"https://gateway.blocksec.com/ipfs/",
	"https://ipfs.renehsz.com/ipfs/",
	"https://cloudflare-ipfs.com/ipfs/",
	"https://ipns.co/",
	"https://ipfs.netw0rk.io/ipfs/",
	"https://gateway.swedneck.xyz/ipfs/",
	"https://ipfs.mrh.io/ipfs/",
	"https://gateway.originprotocol.com/ipfs/",
	"https://ipfs.dapps.earth/ipfs/",
	"https://gateway.pinata.cloud/ipfs/",
	"https://ipfs.doolta.com/ipfs/",
	"https://ipfs.sloppyta.co/ipfs/",
	"https://ipfs.busy.org/ipfs/",
	"https://ipfs.greyh.at/ipfs/",
	"https://gateway.serph.network/ipfs/",
	"https://jorropo.ovh/ipfs/",
	"https://gateway.temporal.cloud/ipfs/",
	"https://ipfs.fooock.com/ipfs/",
	"https://ipfstube.erindachtler.me/ipfs/",
	"https://cdn.cwinfo.net/ipfs/",
}

const (
	checkHash   = "Qmaisz6NMhDB51cCvNWa1GMS7LU1pAxdF4Ld6Ft9kZEP2a"
	checkString = "Hello from IPFS Gateway Checker\n"
)

func main() {
	var port string
	var debug bool
	flag.StringVar(&port, "port", "8085", "port to host on")
	flag.BoolVar(&debug, "debug", true, "debug mode")
	flag.Parse()

	// toggle debug mode
	if debug {
		log.SetLevel("debug")
	} else {
		log.SetLevel("info")
	}

	checkGateways()
	go func() {
		// check gateways every hour
		time.Sleep(60 * time.Minute)
		checkGateways()
	}()

	log.Info("running on :" + port)
	http.HandleFunc("/ipfs/", handler)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Error(err)
	}
}

func checkGateways() {
	jobs := make(chan string, len(gateways))
	type result struct {
		err     error
		gateway string
	}
	results := make(chan result, len(gateways))

	for w := 0; w < 8; w++ {
		go func(jobs <-chan string, results chan<- result) {
			for j := range jobs {
				results <- result{checkGateway(j), j}
			}
		}(jobs, results)
	}

	for _, gateway := range gateways {
		jobs <- gateway
	}
	close(jobs)
	newgateways := []string{}
	for i := 0; i < len(gateways); i++ {
		r := <-results
		if r.err != nil {
			log.Infof("%s ❌", r.gateway)
		} else {
			log.Infof("%s ✔️", r.gateway)
			newgateways = append(newgateways, r.gateway)
		}
	}
	gateways = newgateways
	log.Infof("found %d functional gateways", len(gateways))
}

func checkGateway(gateway string) (err error) {
	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	resp, err := client.Get(gateway + checkHash)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response code: %d", resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	bodyString := string(bodyBytes)
	if bodyString != checkString {
		err = fmt.Errorf("'%s' != '%s'", checkString, bodyString)
	}
	return
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := handle(w, r)
	if err != nil {
		log.Error(err)
	}
	log.Infof("%s %s", r.URL.Path[1:], time.Since(start))
}

func handle(w http.ResponseWriter, r *http.Request) (err error) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if len(r.URL.Path[1:]) < 10 {
		fmt.Fprintf(w, "bad ipfs hash: "+r.URL.Path[1:])
		return
	}
	ipfsContentHash := strings.TrimPrefix(r.URL.Path[1:], "ipfs/")
	if !strings.Contains(ipfsContentHash, "/") {
		http.Redirect(w, r, r.URL.Path+"/", 302)
		return nil
	}

	cancel := make(chan struct{})
	result := make(chan *http.Response)

	for _, gateway := range gateways {
		go cancelableRequest(result, cancel, gateway+ipfsContentHash)
	}

	for i := 0; i < len(gateways); i++ {
		res := <-result
		if res == nil {
			continue
		}
		// log.Println(res)
		if res.StatusCode != http.StatusOK && res.Body != nil {
			continue
		}
		log.Debugf("%s", res.Request.URL.String())

		copyHeader(w.Header(), res.Header)
		io.Copy(w, res.Body)
		res.Body.Close()

		go func() {
			// cancel other requests
			for j := i + 1; j < len(gateways); j++ {
				cancel <- struct{}{}
			}
			// cancel other requests
			for j := i + 1; j < len(gateways); j++ {
				_ = <-result
			}
			close(result)
			close(cancel)
		}()
		break
	}
	return nil
}

func cancelableRequest(result chan *http.Response, cancel chan struct{}, urlToGet string) {
	u, err := url.Parse(urlToGet)
	if err != nil {
		panic(err)
	}
	req, _ := http.NewRequest("GET", urlToGet, nil)
	tr := &http.Transport{} // TODO: copy defaults from http.DefaultTransport
	client := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}
	c := make(chan *http.Response, 1)
	go func() {
		resp, _ := client.Do(req)
		c <- resp
	}()

	for {
		select {
		case <-cancel:
			log.Debugf("Cancelling request for %s", u.Host)
			tr.CancelRequest(req)
			result <- nil
			return
		case r := <-c:
			log.Debugf("Got content from %s", u.Host)
			result <- r
			return
		}
	}
}
