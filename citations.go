package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// AuthorType holds family and given name
type AuthorType struct {
	Family string `json:"family"`
	Given  string `json:"given"`
}

// IssuedType holds the date
type IssuedType struct {
	DateParts [][]interface{} `json:"date-parts"`
}

// ResultType holds the real reply
type ResultType struct {
	Page           string       `json:"page"`
	Title          string       `json:"title"`
	ContainerTitle string       `json:"container-title"`
	Author         []AuthorType `json:"author"`
	Issued         IssuedType   `json:"issued"`
	DOI            string       `json:"DOI"`
}

// ZoteroReply defines the parsed JSON stream
type ZoteroReply struct {
	Jsonrpc string       `json:"jsonrpc"`
	Result  []ResultType `json:"result"`
}

func getBibliographyFromCitekey(citekey string) (ResultType, error) {
	// Request message to Zotero local server
	unformattedRequest := "{\"jsonrpc\": \"2.0\", \"method\": \"item.search\", \"params\": [\"%s\"] }"
	formattedRequest := fmt.Sprintf(unformattedRequest, citekey)
	requestMessage := strings.NewReader(formattedRequest)

	// Create an http request object to be able to set the headers
	req, err := http.NewRequest(
		"POST",
		"http://localhost:23119/better-bibtex/json-rpc",
		requestMessage)
	if err != nil {
		return ResultType{}, err
	}

	// The request body will be in JSON format
	req.Header.Add("Content-Type", "application/json")

	// Send the actual request to the server and receive the reply
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return ResultType{}, err
	}

	// Read everything
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ResultType{}, err
	}

	var reply ZoteroReply

	// Decode answer as JSON
	err = json.Unmarshal(responseBody, &reply)
	if err != nil {
		return ResultType{}, err
	}

	// reply.Result should be an array of ResultType
	// Check that we have at least one item
	if len(reply.Result) == 0 {
		log.Fatalln("There were no results to the query")
	}

	// get the contents of the first result
	r := reply.Result[0]
	return r, nil

}

func buildAsciidocBibligraphyItem(citekey string, index int, result ResultType) string {

	var b strings.Builder

	// Initial part of bibliography line
	fmt.Fprintf(&b, "- [[[%s, %s]]] ", citekey, citekey)

	// Add the authors
	for i, author := range result.Author {
		fmt.Fprintf(&b, "%s %s", author.Given, author.Family)
		if i < len(result.Author)-1 {
			b.WriteString(" and ")
		} else {
			b.WriteString(". ")
		}
	}

	// Add the title
	fmt.Fprintf(&b, "\"%s\"", result.Title)

	// Add the date issued
	fmt.Fprintf(&b, " (%s).", result.Issued.DateParts[0][0])

	// Add the container title
	if len(result.ContainerTitle) > 0 {
		fmt.Fprintf(&b, " %s.", result.ContainerTitle)
	}

	// Add the DOI
	if len(result.DOI) > 0 {
		fmt.Fprintf(&b, " DOI: %s.", result.DOI)
	}

	return b.String()

}

func main() {

	// Define the regex for detecting the citekeys in the Asciidoc document
	re := regexp.MustCompile(`<<.+?>>`)

	// Read the file entirely in memory
	content, err := ioutil.ReadFile("README.asc")
	if err != nil {
		log.Fatal(err)
	}

	// Find all citation keys
	citekeys := re.FindAll([]byte(content), -1)

	sort.Slice(citekeys, func(i, j int) bool {
		return string(citekeys[i]) < string(citekeys[j])
	})

	for i, citekey := range citekeys {
		c := strings.Trim(string(citekey), "<>")
		r, err := getBibliographyFromCitekey(c)
		if err != nil {
			log.Fatal(err)
		}

		s := buildAsciidocBibligraphyItem(c, i, r)
		fmt.Printf("%s\n\n", s)

	}

}
