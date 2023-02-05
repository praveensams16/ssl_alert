package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
    "os"
    "strconv"
)

type (
	result struct {
		url_s, url   string
		days         int
		valid        bool
		reachability bool
	}

	stats struct {
		c     chan result
		count int
	}
)

var (
	wg sync.WaitGroup
)

func url_validator(url string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	_, err := client.Get("https://" + url)

	if err == nil {

		return true
	} else {
		return false
	}

	return true
}

func (s *stats) getdate(url string) {
	defer wg.Done()

	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	url_f := fmt.Sprintf("%s:443", (url))

	conn, err := tls.Dial("tcp", url_f, conf)
	if err != nil {
		wg.Done()
		//goto SAM
		println("errorss")
	}

	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	//for _, cert := range certs {
	cert := certs[0]
	currentTime := time.Now()

	d1, _ := time.Parse("2006-01-02 ", cert.NotAfter.Format("2006-01-02"))
	d2, _ := time.Parse("2006-01-02 ", currentTime.Format("2006-01-02"))
    val,_ := strconv.Atoi(os.Getenv("days"))
	if int(d1.Sub(d2).Hours()/24) > val {
		s.c <- result{url: url, url_s: "Enough days", days: int(d1.Sub(d2).Hours() / 24), valid: true, reachability: true}
		s.count = s.count + 1
	} else {
		s.c <- result{url: url, url_s: "Enough days", days: int(d1.Sub(d2).Hours() / 24), valid: false, reachability: true}
		s.count = s.count + 1
	}
	//	}

}

func results(w http.ResponseWriter, req *http.Request) {

	s := &stats{c: make(chan result, 1000)}
	files, _ := ioutil.ReadFile("/mnt/url.txt")
	lists := strings.Split(string(files), "\n")
	for _, i := range lists {
        if len(i) > 0 {
		if url_validator(i) {
			wg.Add(1)
			go s.getdate(i)

		} else {
			prt := fmt.Sprintf("unifonic_ssl{url=%s,days=%d,valid=%s,reachability=%s,reason=%s} = 0\n", i, 0, "false", "false", "Unable to reach the url")
			fmt.Fprintf(w, prt)
			println("run", i)
		}
      }
	}
	wg.Wait()
	for i := 0; i < s.count; i++ {
		select {
		case d := <-s.c:
			if d.valid && d.reachability {
				prt := fmt.Sprintf("unifonic_ssl{url=%s,days=%d,valid=%t,reachability=%t,reason=\"\"} = 1\n", d.url, d.days, d.valid, d.reachability)
				fmt.Fprintf(w, prt)
			} else {

				prt := fmt.Sprintf("unifonic_ssl{url=%s,days=%d,valid=%t,reachability=%t,reason=\"About to expire\"} = 0\n", d.url, d.days, d.valid, d.reachability)
				fmt.Fprintf(w, prt)
			}
		}
	}

}

func main() {

	http.HandleFunc("/metrics", results)

	http.ListenAndServe(":8090", nil)
}

