package scraper

import (
	"database/sql"
	"encoding/json"
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
			Source:      "ldlc",
		}

		data, err := json.Marshal(p)
		if err != nil {
			log.Panicf("could not json marshal the product %s", title)
		}
		database.InsertProduct(db, p)
		fileMu.Lock()
		file.Write(append(data, '\n'))
		fileMu.Unlock()

	}

	log.Printf("scraped %d products from %s", len(products), subCategory)
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

	slots := make(chan struct{}, 3)
	done := make(chan bool, pageAmount-1)

	for i := 2; i <= pageAmount; i++ { // we start at the first page, so we need to continue with 2+
		go func() {
			slots <- struct{}{}
			newPage, err := browser.NewPage()
			if err != nil {
				log.Panicf("could not create page %v", err)
			}
			defer newPage.Close()
			newPage.Goto(config.LDLC_URL + subCategory + "page" + strconv.Itoa(i) + "/")
			handleProductsListing(newPage, category, subCategory, file, db)
			<-slots
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
		log.Printf("could not create page: %v", err)
		return
	}
	defer page.Close()

	page.Goto(config.LDLC_URL + subCategory)

	// page has a product listing
	amount := page.Locator("#listing > div.wrap-list > div.head-list.fix-list > div.title-2")
	err = amount.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	})
	if err == nil {
		// products found, scrape them
		pagination := page.Locator("#listing > div.wrap-list > div.listing-product > ul.pagination")
		err = pagination.WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(5000),
		})

		if err != nil {
			handleProductsListing(page, category, subCategory, file, db)
		} else {
			handlePagination(page, category, subCategory, browser, file, db)
		}
		return
	}

	// page has sub-subcategories
	catBloc := page.Locator("div.sbloc.cat-bloc > ul")
	err = catBloc.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(5000),
	})
	if err != nil {
		log.Printf("skipping %s: no products or subcategories found", subCategory)
		return
	}

	links, err := catBloc.Locator("a").All()
	if err != nil {
		log.Printf("could not get sub-subcategory links for %s: %v", subCategory, err)
		return
	}

	var hrefs []string
	for _, a := range links {
		href, _ := a.GetAttribute("href")
		hrefs = append(hrefs, href)
	}
	page.Close()

	log.Printf("found %d sub-subcategories in %s, diving deeper", len(hrefs), subCategory)

	slots := make(chan struct{}, 3)
	done := make(chan bool, len(hrefs))

	for _, href := range hrefs {
		go func() {
			slots <- struct{}{}
			ScrapeCategory(db, category, href, browser, file)
			<-slots
			done <- true
		}()
	}

	for range hrefs {
		<-done
	}
}
