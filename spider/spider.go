/*
*/

package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	MAXWORKERCNT = 5 	// cap the number of workers
	MAXSPANCNT = 100 	// cap the number of pages it can get
	MAXIDLECNT = 10 	// number of times a worker can see an empty queue before exiting
)

var (
	gWork chan string
	gDone chan bool
	gDoing int

	gScheme, gHost string

	gHrefs []string
	gHrefseen map[string]bool
)

func Args() (ok bool) {
	var (
		uu *url.URL
		err error
	)
	if len(os.Args) != 2 {
		goto out
	}

	if uu, err = url.Parse(os.Args[1]); err != nil {
		fmt.Printf("failed to parse root: %s, %s\n", os.Args[1], err.Error())
		goto out
	}

	gScheme = uu.Scheme
	gHost = uu.Host

	gHrefs = append(gHrefs, os.Args[1])

	ok = true

out:
	return
}

func main() {
	var (
		err error
		href string
		ok bool
		rsp *http.Response

		killswitch int
	)

	if !Args() {
		fmt.Printf("bad args\n")
		goto out
	}

	gWork = make(chan string)
	gDone = make(chan bool)
	gHrefseen = make(map[string]bool)

	// There is some dodgy html out there...
	gHrefseen[""] = true

	killswitch = 0
	for len(gHrefs) > 0 {
		killswitch++
		if killswitch > MAXSPANCNT {
			break
		}

		href, gHrefs = gHrefs[0], gHrefs[1:]
		if _, ok = gHrefseen[href]; ok {
			continue
		}
		gHrefseen[href] = true

		if href, ok = cleanup_url(href); !ok {
			continue
		}

		fmt.Printf("at %s (%d)\n", href, killswitch)

		if rsp, err = http.Get(href); err != nil {
			fmt.Printf("failed to GET %s: %s\n", href, err.Error())
			continue
		}
		defer rsp.Body.Close()

		tokenize(rsp.Body)
	}

	close(gWork)

	for gDoing > 0 {
		<- gDone
		gDoing--
	}

out:
}

func tokenize(body io.ReadCloser) {
	var (
		zz *html.Tokenizer
		tt html.TokenType
		to html.Token

		href, hh string
		err error
		ok bool
	)

	zz = html.NewTokenizer(body)

	for tt = zz.Next(); tt != html.ErrorToken; tt = zz.Next() {
		to = zz.Token()
		switch to.Type {
		case html.ErrorToken:
			panic("unexpected error token (is this even possible?)")

		case html.StartTagToken:
			if to.Data != "a" {
				// only want links
				break
			}
			if href, err = getattr("href", to.Attr); err != nil {
				// anchor without an href
				break
			}
			if href[0] == '#' {
				// dont care, its the same page
				break
			}

			// add to the list of things to do
			gHrefs = append(gHrefs, href)

		case html.SelfClosingTagToken:
			if to.Data != "img" {
				// only want images
				break
			}
			if href, err = getattr("src", to.Attr); err != nil {
				fmt.Printf("dodgy img with no src: %s\n", to.Data)
				break
			}
			if hh, ok = cleanup_url(href); !ok {
				fmt.Printf("dodgy img src: %s\n", href)
				break
			}
			href = hh

			if !strings.HasSuffix(href, "jpg") {
				// only want some images
				break
			}

			if _, ok = gHrefseen[href]; ok {
				break
			}
			gHrefseen[href] = true

			getimg(href)

		case html.DoctypeToken, html.TextToken, html.EndTagToken, html.CommentToken:
			// dont care

		default:
			fmt.Printf("unimplemented token: %s\n", to.Type.String())
		}
	}
}

func getattr(key string, attrs []html.Attribute) (href string, err error) {
	for _, attr := range attrs {
		if attr.Key == key {
			href = attr.Val
			break
		}
	}

	if href == "" {
		err = fmt.Errorf("missing attribute: %s", key)
	}

	return
}

func getimg(href string) {
	// allow stale workers to exit before getting a new image
	select {
	case <- gDone:
		gDoing--

	default:
		// dont block
	}

	select {
		case gWork <- href:
			// all good
		default:
			// work queue needs workers, but not too many
			if gDoing < MAXWORKERCNT {
				go getworker(gWork, gDone)
				gDoing++
			}
			// might block here if there are too many workers in play
			gWork <- href
	}
}

func cleanup_url(href string) (cleaned string, ok bool) {
	var (
		err error
		uu *url.URL
	)
	// First clean up the href
	if uu, err = url.Parse(href); err != nil {
		fmt.Printf("dodgy url: %s, %s\n", href, err.Error())
		goto out
	}

	if uu.Scheme == "" {
		uu.Scheme = "https"
	}
	if uu.Host == "" {
		uu.Host = gHost
	}
	if uu.Path == "" {
		fmt.Printf("dodgy url: %s, %s\n", href, err.Error())
		goto out
	}

	cleaned = uu.String()
	ok = true

out:
	return
}

/*
goroutine
*/
func getworker(work chan string, done chan bool) {
	var (
		href string
		more bool
		idle int
	)

	// allow the scheduler to run so the master can set up
	time.Sleep(10 * time.Millisecond)

	for {
		select {
		case href, more = <- work:
		default:
			time.Sleep(1 * time.Second)	
			href = ""
			idle++
		}
		
		if !more {
			break
		}
		if idle > MAXIDLECNT {
			// waiting too long for work, so die
			break
		}
		if href == "" {
			// came through idle
			continue
		}

		getworker_01(href)

		// Throttle
		time.Sleep(time.Duration(2 + rand.Intn(9)) * time.Second)
	}

	done <- true
}

func getworker_01(href string) {
	var (
		idx int
		fn string
		fh *os.File
		err error
		rsp *http.Response
	)

	if idx = strings.LastIndexByte(href, '/'); idx < 0 {
		fmt.Printf("missing file: %s\n", href)
		goto out
	}
	fn = href[idx+1:]

	if fh, err = os.Create(fn); err != nil {
		fmt.Printf("failed to create %s: %s\n", fn, err.Error())
		goto out
	}
	defer fh.Close()

	if rsp, err = http.Get(href); err != nil {
		fmt.Printf("failed to get %s: %s\n", href, err.Error())
		goto out
	}
	defer rsp.Body.Close()

	io.Copy(fh, rsp.Body)

out:
}
