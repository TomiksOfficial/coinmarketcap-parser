package main

import (
	"encoding/json"
	"fmt"
	"github.com/gebleksengek/useragents"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"math/rand"
	"os"
	"runtime"
	"time"
)

const PageCount = 2

type CoinItem struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
}

func randomSleep(min, max int) {
	duration := time.Duration(rand.Intn(max-min)+min) * time.Millisecond
	time.Sleep(duration)
}

func main() {
	var parsedCoins []CoinItem
	channel := make(chan []CoinItem)

	runtime.GOMAXPROCS(4)

	service, err := selenium.NewChromeDriverService("./chrome/chromedriver", 4444)
	if err != nil {
		panic(err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{
		Args: []string{
			"--window-size=1920,1080",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--disable-extensions",
			"--headless",
			"--disable-images",
			fmt.Sprintf("--user-agent=%s", useragents.ChromeLatest()),
		},
	})

	caps["pageLoadStrategy"] = "eager"

	for i := 1; i <= PageCount; i++ {
		go ParseCoinsPage(&caps, i, channel)
	}

	for i := 1; i <= PageCount; i++ {
		parsedCoins = append(parsedCoins, <-channel...)
	}

	f, _ := os.Create("output.json")
	defer f.Close()
	js, _ := json.MarshalIndent(parsedCoins, "", "\t")

	f.Write(js)
}

func ParseCoinsPage(caps *selenium.Capabilities, page int, channel chan []CoinItem) {
	var parsedCoins []CoinItem

	driver, err := selenium.NewRemote(*caps, "")
	if err != nil {
		fmt.Println(err)
		return
	}

	script := `
		  Object.defineProperty(navigator, 'webdriver', {
		      get: () => undefined,
		  });
		  window.navigator.chrome = {
		      runtime: {},
		  };
		  Object.defineProperty(navigator, 'languages', {
		      get: () => ['en-US', 'en'],
		  });
		  Object.defineProperty(navigator, 'plugins', {
		      get: () => [1, 2, 3],
		  });
		`
	_, err = driver.ExecuteScript(script, nil)
	if err != nil {
		fmt.Println("Error injecting JavaScript:", err)
		return
	}

	err = driver.Get(fmt.Sprintf("https://coinmarketcap.com/?page=%d", page))
	if err != nil {
		fmt.Println("Error driver.Get():", err)
		return
	}

	randomSleep(1000, 2000)

	script = `	window.scrollBy({
					  top: document.body.scrollHeight / 2,
					  left: 0,
					  behavior: "smooth"
					})`

	_, err = driver.ExecuteScript(script, nil)
	if err != nil {
		fmt.Println("Error scroll to footer:", err)
		return
	}

	randomSleep(500, 1000)

	_, err = driver.ExecuteScript(script, nil)
	if err != nil {
		fmt.Println("Error scroll to footer:", err)
		return
	}

	randomSleep(1000, 2000)

	basic, err := driver.FindElement(selenium.ByCSSSelector, "table.sc-ae0cff98-3 > tbody")
	if err != nil {
		fmt.Println("Failed to get table of elements:", err)
		return
	}

	elements, err := basic.FindElements(selenium.ByCSSSelector, "tr")

	for _, item := range elements {

		pc, err1 := item.FindElement(selenium.ByCSSSelector, "p.sc-71024e3e-0.ehyBa-d")
		randomSleep(50, 100)
		sn, err2 := item.FindElement(selenium.ByCSSSelector, "p.coin-item-symbol")
		randomSleep(50, 100)
		if err1 != nil || err2 != nil {
			if err1 != nil {
				fmt.Println("Failed to get name element:", err1)
			} else {
				name, _ := pc.Text()
				fmt.Println("Name:", name)
			}

			if err2 != nil {
				fmt.Println("Failed to get short_name element:", err2)
			} else {
				shortName, _ := sn.Text()
				fmt.Println("ShortName:", shortName)
			}
			continue
		}

		name, _ := pc.Text()
		shortName, _ := sn.Text()
		parsedCoins = append(parsedCoins, CoinItem{name, shortName})
	}

	channel <- parsedCoins
}
