package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"ldlcscraper.com/config"
	"ldlcscraper.com/models"

	"github.com/playwright-community/playwright-go"
)

func handleProductsListing(page playwright.Page, category string, subCategory string) {
	page.Locator(".pdt-item").First().WaitFor()
	products, err := page.Locator(".pdt-item").All()

	if err != nil {
		log.Panicf("could not get products items")
	}

	file, err := os.OpenFile("products.jsonl", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panicf("could not save in products.jsonl")
	}

	defer file.Close()

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

		p := models.Product{
			Title:       title,
			Price:       price,
			Link:        link,
			Pic:         pic,
			Desc:        desc,
			Stock:       stock,
			Category:    category,
			SubCategory: subCategory,
		}

		data, err := json.Marshal(p)
		if err != nil {
			log.Panicf("could not json marshal the product %s", title)
		}
		fmt.Println(string(data))
		file.Write(append(data, '\n'))

	}
}

func handlePagination(page playwright.Page, category string, subCategory string, browser playwright.Browser) {
	pages, err := page.Locator("ul.pagination > li:not(.next) > a[data-page]").Last().GetAttribute("data-page")
	if err != nil {
		log.Panicf("could not get page amount (pagination)")
	}

	pageAmount, err := strconv.Atoi(pages)
	if err != nil {
		log.Panicf("could not convert pageAmount to int %v", err)
	}

	done := make(chan bool, pageAmount-1)

	for i := 2; i <= pageAmount; i++ { // we start at the first page, so we need to continue with 2+
		go func() {
			newPage, err := browser.NewPage()
			if err != nil {
				log.Panicf("could not create page %v", err)
			}
			defer newPage.Close()
			newPage.Goto(config.LDLC_URL + subCategory + "page" + strconv.Itoa(i) + "/")
			handleProductsListing(newPage, category, subCategory)
			done <- true
		}()
	}

	for i := 2; i <= pageAmount; i++ {
		<-done
	}

}

func ScrapeCategory(category string, subCategory string, browser playwright.Browser) {
	page, err := browser.NewPage()
	if err != nil {
		log.Panicf("could not create page: %v", err)
	}
	defer page.Close()

	page.Goto(config.LDLC_URL + subCategory)
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
		handleProductsListing(page, category, subCategory)
		return
	}

	handlePagination(page, category, subCategory, browser)
}
