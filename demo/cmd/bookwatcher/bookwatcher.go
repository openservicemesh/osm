package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openservicemesh/osm/demo/cmd/common"
)

const (
	bookBuyerPort   = 8080
	bookStoreV1Port = 8081
	bookStoreV2Port = 8082
	bookThiefPort   = 8083
)

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func printGreenln(msg string) {
	fmt.Printf("\033[32m%s\033[0m\n", msg)
}

func printYellowln(msg string) {
	fmt.Printf("\033[33m%s\033[0m\n", msg)
}

func printRedln(msg string) {
	fmt.Printf("\033[31m%s\033[0m\n", msg)
}

func getBookData(dest interface{}, port int, wg *sync.WaitGroup, errc chan<- error) {
	defer wg.Done()

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/raw", port))
	if err != nil {
		errc <- fmt.Errorf("error fetching data (port %d): %v", port, err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			errc <- fmt.Errorf("error closing response (port %d): %v", port, err)
		}
	}()

	output, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errc <- fmt.Errorf("error reading data (port %d): %v", port, err)
		return
	}

	err = json.Unmarshal(output, dest)
	if err != nil {
		errc <- fmt.Errorf("error unmarshalling data (port %d): %v", port, err)
	}
}

func main() {
	bookBuyerPurchases := &common.BookBuyerPurchases{}
	bookThiefThievery := &common.BookThiefThievery{}
	bookStorePurchasesV1 := &common.BookStorePurchases{}
	bookStorePurchasesV2 := &common.BookStorePurchases{}
	wg := &sync.WaitGroup{}
	errc := make(chan error, 4)

	for {
		clearScreen()

		bookBuyerPurchasesTemp := *bookBuyerPurchases
		wg.Add(1)
		go getBookData(bookBuyerPurchases, bookBuyerPort, wg, errc)

		bookThiefThieveryTemp := *bookThiefThievery
		wg.Add(1)
		go getBookData(bookThiefThievery, bookThiefPort, wg, errc)

		bookStorePurchasesV1Temp := *bookStorePurchasesV1
		wg.Add(1)
		go getBookData(bookStorePurchasesV1, bookStoreV1Port, wg, errc)

		bookStorePurchasesV2Temp := *bookStorePurchasesV2
		wg.Add(1)
		go getBookData(bookStorePurchasesV2, bookStoreV2Port, wg, errc)

		complete := make(chan bool)
		go func() {
			wg.Wait()
			close(complete)
		}()

		select {
		case err := <-errc:
			wg.Wait()
			close(errc)
			log.Fatal(err)
		case <-complete:
		}

		bookBuyerHasChanged := bookBuyerPurchases.BooksBought-bookBuyerPurchasesTemp.BooksBought != 0 ||
			bookBuyerPurchases.BooksBoughtV1-bookBuyerPurchasesTemp.BooksBoughtV1 != 0 ||
			bookBuyerPurchases.BooksBoughtV2-bookBuyerPurchasesTemp.BooksBoughtV2 != 0

		bookThiefHasChanged := bookThiefThievery.BooksStolen-bookThiefThieveryTemp.BooksStolen != 0 ||
			bookThiefThievery.BooksStolenV1-bookThiefThieveryTemp.BooksStolenV1 != 0 ||
			bookThiefThievery.BooksStolenV2-bookThiefThieveryTemp.BooksStolenV2 != 0

		bookStoreV1HasChanged := bookStorePurchasesV1.BooksSold-bookStorePurchasesV1Temp.BooksSold != 0
		bookStoreV2HasChanged := bookStorePurchasesV2.BooksSold-bookStorePurchasesV2Temp.BooksSold != 0

		printFunc := printYellowln
		if bookBuyerHasChanged {
			printFunc = printGreenln
		}
		printFunc(fmt.Sprintf(
			"bookbuyer     Books bought: %d  V1 books bought: %d  V2 books bought: %d",
			bookBuyerPurchases.BooksBought,
			bookBuyerPurchases.BooksBoughtV1,
			bookBuyerPurchases.BooksBoughtV2,
		))

		printFunc = printYellowln
		if bookThiefHasChanged {
			printFunc = printRedln
		}
		printFunc(fmt.Sprintf(
			"bookthief     Books stolen: %d  V1 books stolen: %d  V2 books stolen: %d",
			bookThiefThievery.BooksStolen,
			bookThiefThievery.BooksStolenV1,
			bookThiefThievery.BooksStolenV2,
		))

		printFunc = printYellowln
		if bookStoreV1HasChanged {
			printFunc = printGreenln
		}
		printFunc(fmt.Sprintf("bookstore v1  Books sold: %d", bookStorePurchasesV1.BooksSold))

		printFunc = printYellowln
		if bookStoreV2HasChanged {
			printFunc = printGreenln
		}
		printFunc(fmt.Sprintf("bookstore v2  Books sold: %d", bookStorePurchasesV2.BooksSold))

		time.Sleep(1 * time.Second)
	}
}
