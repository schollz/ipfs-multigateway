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
	"https://ipfs.io/ipfs/:hash",
	"https://gateway.ipfs.io/ipfs/:hash",
	"https://ipfs.infura.io/ipfs/:hash",
	"https://ninetailed.ninja/ipfs/:hash",
	"https://ipfs.globalupload.io/:hash",
	"https://ipfs.eternum.io/ipfs/:hash",
	"https://hardbin.com/ipfs/:hash",
	"https://gateway.blocksec.com/ipfs/:hash",
	"https://cloudflare-ipfs.com/ipfs/:hash",
	"https://ipns.co/:hash",
	"https://ipfs.mrh.io/ipfs/:hash",
	"https://gateway.originprotocol.com/ipfs/:hash",
	"https://gateway.pinata.cloud/ipfs/:hash",
	"https://ipfs.doolta.com/ipfs/:hash",
	"https://ipfs.sloppyta.co/ipfs/:hash",
	"https://ipfs.busy.org/ipfs/:hash",
	"https://ipfs.greyh.at/ipfs/:hash",
	"https://gateway.serph.network/ipfs/:hash",
	"https://jorropo.ovh/ipfs/:hash",
	"https://gateway.temporal.cloud/ipfs/:hash",
	"https://ipfs.fooock.com/ipfs/:hash",
	"https://cdn.cwinfo.net/ipfs/:hash",
	"https://ipfs.privacytools.io/ipfs/:hash",
	"https://ipfs.jeroendeneef.com/ipfs/:hash",
	"https://permaweb.io/ipfs/:hash",
	"https://ipfs.stibarc.com/ipfs/:hash",
	"https://ipfs.best-practice.se/ipfs/:hash",
	"https://ipfs.2read.net/ipfs/:hash",
	"https://storjipfs-gateway.com/ipfs/:hash",
	"https://ipfs.runfission.com/ipfs/:hash",
	"https://trusti.id/ipfs/:hash",
}

const (
	checkHash   = "Qmaisz6NMhDB51cCvNWa1GMS7LU1pAxdF4Ld6Ft9kZEP2a"
	checkString = "Hello from IPFS Gateway Checker\n"
)

func main() {
	var port string
	var debug bool
	flag.StringVar(&port, "port", "8085", "port to host on")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	// toggle debug mode
	if debug {
		log.SetLevel("debug")
	} else {
		log.SetLevel("info")
	}
	log.Debugf("debug mode: %+v", debug)

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

	for w := 0; w < len(gateways); w++ {
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
	log.Infof("checking gateways...")
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
	resp, err := client.Get(strings.Replace(gateway, ":hash", checkHash, 1))
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

	cancel := make(chan struct{}, len(gateways))
	result := make(chan *http.Response, len(gateways))

	for _, gateway := range gateways {
		go cancelableRequest(result, cancel, strings.Replace(gateway, ":hash", ipfsContentHash, 1), r.Header)
	}

	for i := 0; i < len(gateways); i++ {
		res := <-result
		if res == nil {
			continue
		}
		log.Debugf("res.StatusCod: %+v", res.StatusCode)
		// if res.StatusCode != http.StatusOK && res.Body != nil {
		// 	continue
		// }
		log.Debugf("%s", res.Request.URL.String())

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

		for k, vv := range res.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		io.Copy(w, res.Body)
		res.Body.Close()

		break
	}
	return nil
}

func cancelableRequest(result chan *http.Response, cancel chan struct{}, urlToGet string, requestHeader http.Header) {
	u, err := url.Parse(urlToGet)
	if err != nil {
		panic(err)
	}
	req, _ := http.NewRequest("GET", urlToGet, nil)
	for k := range requestHeader {
		for _, v := range requestHeader[k] {
			req.Header.Add(k, v)
		}
	}
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
