package main

import (
	"flag"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

func main() {
	var port string
	var debug bool
	flag.StringVar(&port, "port", "8085", "port to host on")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.Parse()

	// toggle debug mode
	if debug {
		setLogLevel("debug")
	} else {
		setLogLevel("info")
	}

	log.Info("running on :" + port)
	http.HandleFunc("/ipfs/", handler)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Error(err)
	}
}

var gateways = []string{
	"https://ipfs.io/ipfs/",
	"https://gateway.ipfs.io/ipfs/",
	"https://ipfs.infura.io/ipfs/",
	"https://xmine128.tk/ipfs/",
	"https://ipfs.jes.xxx/ipfs/",
	"https://siderus.io/ipfs/",
	"https://www.eternum.io/ipfs/",
	"https://hardbin.com/ipfs/",
	"https://ipfs.wa.hle.rs/ipfs/",
	"https://ipfs.renehsz.com/ipfs/",
	"https://cloudflare-ipfs.com/ipfs/",
	"https://ipfs.netw0rk.io/ipfs/",
	"https://gateway.swedneck.xyz/ipfs/",
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

func cancelableRequest(result chan *http.Response, cancel chan struct{}, url string) {
	req, _ := http.NewRequest("GET", url, nil)
	tr := &http.Transport{} // TODO: copy defaults from http.DefaultTransport
	client := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}
	c := make(chan *http.Response, 1)
	go func() {
		resp, _ := client.Do(req)
		// if err != nil {
		// 	log.Println(err)
		// }
		c <- resp
	}()

	for {
		select {
		case <-cancel:
			// log.Println("Cancelling request for " + url)
			tr.CancelRequest(req)
			result <- nil
			return
		case r := <-c:
			// log.Println("Client finished", r)
			result <- r
			return
		}
	}
}

// setLogLevel determines the log level
func setLogLevel(level string) (err error) {

	// https://en.wikipedia.org/wiki/ANSI_escape_code#3/4_bit
	// https://github.com/cihub/seelog/wiki/Log-levels
	appConfig := `
	<seelog minlevel="` + level + `">
	<outputs formatid="stdout">
	<filter levels="debug,trace">
		<console formatid="debug"/>
	</filter>
	<filter levels="info">
		<console formatid="info"/>
	</filter>
	<filter levels="critical,error">
		<console formatid="error"/>
	</filter>
	<filter levels="warn">
		<console formatid="warn"/>
	</filter>
	</outputs>
	<formats>
		<format id="stdout"   format="%Date %Time [%LEVEL] %File %FuncShort:%Line %Msg %n" />
		<format id="debug"   format="%Date %Time %EscM(37)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="info"    format="%Date %Time %EscM(36)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="warn"    format="%Date %Time %EscM(33)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
		<format id="error"   format="%Date %Time %EscM(31)[%LEVEL]%EscM(0) %File %FuncShort:%Line %Msg %n" />
	</formats>
	</seelog>
	`
	logger, err := log.LoggerFromConfigAsBytes([]byte(appConfig))
	if err != nil {
		return
	}
	log.ReplaceLogger(logger)
	return
}
