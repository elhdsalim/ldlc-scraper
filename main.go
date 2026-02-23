package main

import (
	"log"
	"os"

	"ldlcscraper.com/config"
	"ldlcscraper.com/scraper"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panicf("could not load .env: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Panicf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Proxy: &playwright.Proxy{
			Server:   os.Getenv("PROXY_SERVER"),
			Username: playwright.String(os.Getenv("PROXY_USERNAME")),
			Password: playwright.String(os.Getenv("PROXY_PASSWORD")),
		},
	})
	if err != nil {
		log.Panicf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Panicf("could not create page: %v", err)
	}
	defer page.Close()

	if _, err = page.Goto(config.LDLC_URL + config.LAPTOPS); err != nil {
		log.Fatalf("could not goto: %v", err)
	}

	ulist := page.Locator("body > div.main > div.sbloc.cat-bloc > ul")
	err = ulist.WaitFor()
	if err != nil {
		log.Panicf("could not find the items list: %v", err)
	}

	list, err := ulist.Locator("a").All()
	if err != nil {
		log.Panicf("could not find the hypedlink text")
	}

	var hrefs []string

	for _, a := range list {
		href, _ := a.GetAttribute("href")
		hrefs = append(hrefs, href)
	}

	done := make(chan bool, len(hrefs))

	for _, href := range hrefs[:4] {
		go func() {
			scraper.ScrapeCategory(config.LAPTOPS, href, browser)
			done <- true
		}()
	}

	for range hrefs[:4] {
		<-done
	}
}
