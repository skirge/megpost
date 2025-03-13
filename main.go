package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"net/url"
	"regexp"
	"strconv"
)

// json to eventually save results to json output in future
type FileMetrics struct {
	Status                  string    `json:"status"`
	Length                  int    `json:"length"`
	Words                   int    `json:"words"`
	Lines                   int    `json:"lines"`
	ContentType             string `json:"content-type"`
	RedirectLocation        string `json:"redirectlocation"`
	Resultfile              string `json:"resultfile"`
	Url                     string `json:"url"`
	Host                    string `json:"host"`
	CountHeaders            string `json:"count-headers"`
	RedirectDomain          string `json:"redirect-domain"`
	CountRedirectParameters string `json:"count-redirect-parameters"`
	LengthTitle             string `json:"length-title"`
	WordsTitle              string `json:"words-title"`
	CountCssFiles           string `json:"count-css-files"`
	CountJsFiles            string `json:"count-js-files"`
	CountTags               string `json:"count-tags"`
	KeepReason              string `json:"keepreason"`
	IsInteresting	        bool `json:"interesting"`
}


var (
	statusRegex = regexp.MustCompile("HTTP/.*?\\s+(\\d+)\\s+")
	contentTypeRegex = regexp.MustCompile("Content-Type: (.*)")
	redirectLocationRegex = regexp.MustCompile("Location: (.*)")
	titleRegex  = regexp.MustCompile("(?mi)<title>(.*?)</title>")
	cssRegex    = regexp.MustCompile(`\.css(\?|)`)
	jsRegex     = regexp.MustCompile(`\.js(\?|)`)
	headerRegex = regexp.MustCompile("(.*):(.*)")
	tagsRegex   = regexp.MustCompile("<(.*?)>")
	jsonRegex   = regexp.MustCompile("(\"|')(\\s|):(\\s|)(\"|'|)")
	interestingContent = regexp.MustCompile("(?mi)(PUDAX|DUPA|INJECTX)")
)

func GetRedirectLocation(headers string) string {
	redirects := redirectLocationRegex.FindStringSubmatch(headers)
	if len(redirects) > 0 {
		return redirects[1]
	} else {
		return ""
	}
}

func IsInterestingContent(headers string, body string) bool {
	return len(interestingContent.FindAllString(headers,-1)) > 0 ||
		len(interestingContent.FindAllString(body, -1)) > 0
}

func GetStatus(headers string) string {
	status := statusRegex.FindStringSubmatch(headers)
	if len(status) > 0 {
		return status[1]
	} else { 
		return ""
	}
}

func GetContentType(headers string) string {
	content := contentTypeRegex.FindStringSubmatch(headers)
	if len(content) > 0 { 
		return content[1]
	} else {
		return ""
	}
}

func CountTags(contentType, body string) string {
	if strings.Contains(contentType, "html") || strings.Contains(contentType, "xml") {
		return strconv.Itoa(len(tagsRegex.FindAllString(body, -1)))
	}
	if strings.Contains(contentType, "json") {
		return strconv.Itoa(len(jsonRegex.FindAllString(body, -1)))
	}
	return "0"
}

func CountCssFiles(body string) string {
	return strconv.Itoa(len(cssRegex.FindAllString(body, -1)))
}

func CountJsFiles(body string) string {
	return strconv.Itoa(len(jsRegex.FindAllString(body, -1)))
}

func CalculateTitleLength(body string) string {
	matches := titleRegex.FindStringSubmatch(body)
	if len(matches) == 2 {
		return strconv.Itoa(len(matches[1]))
	}
	return "0"
}

func CalculateTitleWords(body string) string {
	matches := titleRegex.FindStringSubmatch(body)
	if len(matches) == 2 {
		return strconv.Itoa(len(strings.Fields(matches[1])))
	}
	return "0"
}

func ExtractRedirectDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}

func CountRedirectParameters(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "0"
	}
	return strconv.Itoa(len(parsedURL.Query()))
}

func CountHeaders(headerString string) string {
	return strconv.Itoa(len(headerRegex.FindAllString(headerString, -1)))
}

func SeperateContentIntoHeadersAndBody(Content string) (string, string) {

	var prev rune
	var EntireResponse string

	for i, c := range Content {
		if prev == '\n' && c == '<' {
			EntireResponse = Content[i:]
			break
		}
		prev = c
	}

	//println("Content:")
	//print(Content)

	var HeaderBuilder, BodyBuilder strings.Builder
	inHeaders := true

	scanner := bufio.NewScanner(strings.NewReader(EntireResponse))
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if inHeaders {
			if line == "" {
				inHeaders = false
				continue
			}
			HeaderBuilder.WriteString(strings.Replace(line,"< ","",1)) // replace < marker of response
			HeaderBuilder.WriteByte('\n')
		} else {
			BodyBuilder.WriteString(line)
			BodyBuilder.WriteByte('\n')
		}
	}

	HeaderString := strings.TrimSpace(HeaderBuilder.String())
	BodyString := strings.TrimSpace(BodyBuilder.String())
	return HeaderString, BodyString
}

func computeMetrics(path string) (FileMetrics, error) {
	var metrics FileMetrics

	file, err := os.Open(path)
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	content := make([]byte, 1000000)
	n, err := io.ReadFull(file, content)
	if err != nil && err != io.ErrUnexpectedEOF {
		return metrics, err
	}
	content = content[:n]

	Headers, Body := SeperateContentIntoHeadersAndBody(string(content))

	//println("Headers:")
	//println(Headers)
	//println("Body:")
	//println(Body)

	metrics.Status = GetStatus(Headers)
	metrics.ContentType = GetContentType(Headers)
	metrics.RedirectLocation = GetRedirectLocation(Headers)
	metrics.RedirectDomain = ExtractRedirectDomain(metrics.RedirectLocation)
	metrics.CountRedirectParameters = CountRedirectParameters(metrics.RedirectLocation)
	metrics.IsInteresting = IsInterestingContent(Headers, Body)
	metrics.CountHeaders = CountHeaders(Headers)
        metrics.LengthTitle = CalculateTitleLength(Body)
        metrics.WordsTitle = CalculateTitleWords(Body)
        metrics.CountCssFiles = CountCssFiles(Body)
        metrics.CountJsFiles = CountJsFiles(Body)
        metrics.CountTags = CountTags(metrics.ContentType, Body)
	
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		metrics.Words++
	}

	if err := scanner.Err(); err != nil {
		return metrics, err
	}

	return metrics, nil
}

func keyForMetrics(m FileMetrics) string {
	key := fmt.Sprintf("%s-%d-%s-%s-%s-%s-%s-%s-%s-%s-%s-%s-%t", m.Status, m.Words, m.ContentType, m.RedirectLocation,
		m.CountHeaders, m.RedirectDomain, m.CountRedirectParameters, m.LengthTitle,
		m.WordsTitle, m.CountCssFiles, m.CountJsFiles, m.CountTags, m.IsInteresting)
	//println("Calculated key:")
	//println(key)
	return key
}

func main() {

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <meg responses directory>")
	}
	rootDir := os.Args[1]

	groupedFiles := make(map[string][]string)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error processing %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		metrics, err := computeMetrics(path)
		if err != nil {
			log.Printf("Error calculating metrics %s: %v", path, err)
			return nil
		}
		key := keyForMetrics(metrics)
		groupedFiles[key] = append(groupedFiles[key], path)

		return nil
	})
	if err != nil {
		log.Fatalf("Error searching: %v", err)
	}

	removed := 0
	for key, files := range groupedFiles {
		if len(files) > 1 {
			fmt.Printf("Group %s contains %d files. To save: %s\n", key, len(files), files[0])
			for _, dup := range files[1:] {
				err := os.Remove(dup)
				//_, err := os.Stat(dup)
				if err != nil {
					log.Printf("[-] Can't remove %s: %v", dup, err)
				} else {
					fmt.Printf("[*] Duplicate removed: %s\n", dup)
					removed ++
				}
			}
		}
	}
	fmt.Printf("[!] Removed %d files total\n", removed)
}

