package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/syumai/workers"
	"github.com/syumai/workers/cloudflare/fetch"
)

func main() {
	workers.Serve(http.HandlerFunc(index))
}

// NB: cannot use net/http, text/template in tinygo

const zhihuHost = "https://zhihu.com"

func index(w http.ResponseWriter, req *http.Request) {
	if !isBot(req) {
		http.Redirect(w, req, zhihuHost+req.URL.Path, http.StatusFound)
		return
	}
	tags, err := generate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(tags))
	return
}

func isBot(req *http.Request) bool {
	ua := strings.Join(req.Header.Values("User-Agent"), ",")
	ua = strings.ToLower(ua)
	if strings.Contains(ua, "bot") {
		return true
	}
	if strings.Contains(ua, "https://github.com/sindresorhus/got") {
		return true
	}
	return false
}

const tmplHTML = `
<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8" />
    <meta property="og:locale" content="zh_CN" />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="Zhihu" />
    <meta property="og:title" content="%[1]s" />
    <meta property="og:description" content="%[2]s" />
    <meta property="og:image" content="%[3]s" />
    <meta property="og:url" content="%[4]s" />

    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="%[1]s" />
    <meta name="twitter:description" content="%[2]s" />
    <meta name="twitter:image:src" content="%[3]s" />
    <meta name="twitter:url" content="%[4]s" />
</head>

<body>
    <p>ZhiHu Link Preview</p>
</body>

</html>
`

func generate(req *http.Request) (string, error) {
	// TODO: cache
	url := zhihuHost + req.URL.Path
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	html, err := fetchHTML(ctx, url)
	if err != nil {
		return "", err
	}
	tags, err := parseHTML(html)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		tmplHTML,
		tags.Title,
		tags.Description,
		tags.Image,
		url,
	), nil
}

func fetchHTML(ctx context.Context, url string) (string, error) {
	c := fetch.NewClient()
	req, _ := fetch.NewRequest(ctx, "GET", url, nil)
	resp, err := c.Do(req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type Tags struct {
	Title       string
	Description string
	Image       string
}

func parseHTML(html string) (Tags, error) {
	title := regexp.MustCompile(`<title [^>]*?>(.+)</title>`).FindStringSubmatch(html)
	description := regexp.MustCompile(`<meta [^>]*?name="description"[^>]*content="(.+?)"/>`).FindStringSubmatch(html)
	if len(title) < 2 || len(description) < 2 {
		return Tags{}, fmt.Errorf("failed to parse html")
	}
	return Tags{
		Title:       title[1],
		Description: description[1],
		Image:       "https://pic2.zhimg.com/80/v2-f6b1f64a098b891b4ea1e3104b5b71f6_720w.png",
	}, nil
}
