package main

import (
	"log"

	"github.com/playwright-community/playwright-go"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Panicf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
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

	if _, err = page.Goto(LDLC_URL + LAPTOPS); err != nil {
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

	for _, href := range hrefs {
		go func() {
			ScrapeCategory(href, browser)
			done <- true
		}()
	}

	for range hrefs {
		<-done
	}
}
