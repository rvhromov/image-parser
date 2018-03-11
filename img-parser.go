package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/html"
)

var (
	index  int
	chLink = make(chan string)
	chDone = make(chan bool)
)

func main() {
	var foundImgLinks []string
	whereToLook := os.Args[1:]
	defer close(chLink)

	// start looking for links in specific page
	for _, url := range whereToLook {
		go seek(url)
	}

	// add found links to slice
	for i := 0; i < len(whereToLook); {
		select {
		case link := <-chLink:
			foundImgLinks = append(foundImgLinks, link)
		case <-chDone:
			i++
		}
	}

	// show the list of found links
	fmt.Println("\n", len(foundImgLinks), "images were found and saved:\n")
	for i, link := range foundImgLinks {
		fmt.Println(i+1, " - ", link)
	}
}

func seek(url string) {
	// notify when we finished looking for links
	defer func() {
		chDone <- true
	}()

	// make a GET-request to desired page
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	parseResponse(resp)
}

func parseResponse(resp *http.Response) {
	// get the root of the parse tree as a *Node
	body, _ := ioutil.ReadAll(resp.Body)
	r := bytes.NewReader(body)
	doc, err := html.Parse(r)
	if err != nil {
		fmt.Println(err)
		return
	}

	// process each img node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			chLink <- extractLink(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}

func extractLink(n *html.Node) string {
	var src string
	// look for src attribute and get its value
	for _, a := range n.Attr {
		if a.Key == "src" {
			src = a.Val
		}
	}

	saveImage(src)
	return src
}

func saveImage(src string) {
	// get image via link
	resp, err := http.Get(src)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	// save image as file
	img, err := os.Create("image" + strconv.Itoa(index) + src[len(src)-4:])
	if err != nil {
		fmt.Println("ERROR: ", err)
	}

	io.Copy(img, resp.Body)
	resp.Body.Close()
	img.Close()
	index++
}
