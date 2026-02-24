package main

import (
	"log"
	"os"

	"ldlcscraper.com/config"
	"ldlcscraper.com/database"
	"ldlcscraper.com/scraper"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panicf("could not load .env: %v", err)
	}

	db, err := database.InitDatabase("products.db")
	if err != nil {
		log.Fatalf("could not init database %v", err)
	}
	defer db.Close()

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

	file, err := os.OpenFile("products.jsonl", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("could not open products.jsonl: %v", err)
	}
	defer file.Close()

	done := make(chan bool, len(config.Categories))

	for category, path := range config.Categories {
		go func() {
			log.Printf("scraping category: %s", category)

			page, err := browser.NewPage()
			if err != nil {
				log.Printf("could not create page for %s: %v", category, err)
				done <- true
				return
			}

			if _, err = page.Goto(config.LDLC_URL + path); err != nil {
				log.Printf("could not goto %s: %v", category, err)
				page.Close()
				done <- true
				return
			}

			ulist := page.Locator("body > div.main > div.sbloc.cat-bloc > ul")
			err = ulist.WaitFor()
			if err != nil {
				log.Printf("could not find subcategories for %s: %v", category, err)
				page.Close()
				done <- true
				return
			}

			list, err := ulist.Locator("a").All()
			if err != nil {
				log.Printf("could not find links for %s: %v", category, err)
				page.Close()
				done <- true
				return
			}

			var hrefs []string
			for _, a := range list {
				href, _ := a.GetAttribute("href")
				hrefs = append(hrefs, href)
			}
			page.Close()

			slots := make(chan struct{}, 3)
			subDone := make(chan bool, len(hrefs))

			for _, href := range hrefs {
				go func() {
					slots <- struct{}{}
					scraper.ScrapeCategory(db, category, href, browser, file)
					<-slots
					subDone <- true
				}()
			}

			for range hrefs {
				<-subDone
			}

			log.Printf("done scraping %s (%d subcategories)", category, len(hrefs))
			done <- true
		}()
	}

	for range config.Categories {
		<-done
	}
}
