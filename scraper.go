package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/playwright-community/playwright-go"
)

func handleNoPagination(page playwright.Page) {
	products, err := page.Locator(".pdt-item").All()

	if err != nil {
		log.Panicf("could not get products items")
	}

	for _, product := range products {
		price, err := product.Locator("div.price").Last().InnerText()
		if err != nil {
			log.Panicf("could not get .pic")
		}

		title, err := product.Locator("h3.title-3").InnerText()
		if err != nil {
			log.Panicf("could not get title")
		}

		link, err := product.Locator("h3.title-3 > a").GetAttribute("href")
		if err != nil {
			log.Panicf("could not get link")
		}

		pic, err := product.Locator("div.pic > a > img").GetAttribute("src")
		if err != nil {
			log.Panicf("could not get pic")
		}

		desc, err := product.Locator("p.desc").InnerText()
		if err != nil {
			log.Panicf("could not get desc")
		}

		stock, err := product.Locator("div[data-stock-web]").GetAttribute("data-stock-web")
		if err != nil {
			log.Printf("could not get stock: %v", err)
		}

		p := Product{
			Title: title,
			Price: price,
			Link:  link,
			Pic:   pic,
			Desc:  desc,
			Stock: stock,
		}

		data, err := json.Marshal(p)
		if err != nil {
			log.Panicf("could not json marshal the product %s", title)
		}
		fmt.Println(string(data))

	}

}

func ScrapeCategory(url string, browser playwright.Browser) {
	page, err := browser.NewPage()
	if err != nil {
		log.Panicf("could not create page: %v", err)
	}
	defer page.Close()

	page.Goto(LDLC_URL + url)
	amount := page.Locator("#listing > div.wrap-list > div.head-list.fix-list > div.title-2")
	err = amount.WaitFor()
	if err != nil {
		log.Panicf("could not find the items list: %v", err)
	}

	pagination := page.Locator("#listing > div.wrap-list > div.listing-product > ul.pagination")
	err = pagination.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	})

	if err != nil {
		fmt.Printf("1 %s\n", url)
		handleNoPagination(page)
		return
	}

	text, err := pagination.Locator("li").Last().InnerText()
	if err != nil {
		log.Panicf("could not get pagination text : %v", err)
	}
	fmt.Printf("%s %s\n", text, url)
}
