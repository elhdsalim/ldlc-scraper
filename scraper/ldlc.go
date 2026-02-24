package scraper

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"ldlcscraper.com/config"
	"ldlcscraper.com/database"
	"ldlcscraper.com/models"

	"github.com/playwright-community/playwright-go"
)

var fileMu sync.Mutex

func handleProductsListing(page playwright.Page, category string, subCategory string, file *os.File, db *sql.DB) {
	page.Locator(".pdt-item").First().WaitFor()
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

		price = strings.ReplaceAll(price, "€", ".")
		price = strings.ReplaceAll(price, " ", "")
		price = strings.ReplaceAll(price, "\u00a0", "")

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
		database.InsertProduct(db, p)
		fmt.Println(string(data))
		fileMu.Lock()
		file.Write(append(data, '\n'))
		fileMu.Unlock()

	}
}

func handlePagination(page playwright.Page, category string, subCategory string, browser playwright.Browser, file *os.File, db *sql.DB) {
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
			handleProductsListing(newPage, category, subCategory, file, db)
			done <- true
		}()
	}

	for i := 2; i <= pageAmount; i++ {
		<-done
	}

}

func ScrapeCategory(db *sql.DB, category string, subCategory string, browser playwright.Browser, file *os.File) {
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
		handleProductsListing(page, category, subCategory, file, db)
		return
	}

	handlePagination(page, category, subCategory, browser, file, db)
}
