package models

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-ini/ini"
	"googlemaps.github.io/maps"
)

type StoreInfo struct {
	URL       string  `json:"url"`
	StoreName string  `json:"storename"`
	Genre     string  `json:"genre"`
	Point     float64 `json:"point"`
	Img       string  `json:"img"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

var baseURL = "https://tabelog.com/rstLst/"

func GetPage(genre string) {
	base, _ := url.Parse(baseURL + genre + "/")
	if err := os.MkdirAll("html/"+genre, 0777); err != nil {
		log.Fatalln(err)
	}
	var wg sync.WaitGroup
	maxCh := make(chan int, 10)
	for page := 1; page < 61; page++ {
		wg.Add(1)
		maxCh <- 1
		time.Sleep(1 * time.Second)
		go func(page int) {
			defer wg.Done()
			reference, _ := url.Parse(strconv.Itoa(page) + "/?Srt=D&SrtT=rt&sort_mode=1&sk=%E3%83%A9%E3%83%BC%E3%83%A1%E3%83%B3&svt=1900&svps=2")
			endpoint := base.ResolveReference(reference).String()
			doc, err := goquery.NewDocument(endpoint)
			if err != nil {
				fmt.Print("url scarapping failed")
			}
			body, err := doc.Find("body").Html()
			if err != nil {
				fmt.Print("dom get failed")
			}
			title := fmt.Sprintf("%s.html", doc.Find("title").Text())
			if err := ioutil.WriteFile("html/"+genre+"/"+title, []byte(body), 0666); err != nil {
				fmt.Println("write file err")
			}
			<-maxCh
		}(page)
	}
}

func GetAddress(url string) string {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		fmt.Print("url scarapping failed")
	}
	address := doc.Find("p.rstinfo-table__address").Text()
	if err != nil {
		fmt.Print("dom get failed")
	}
	return address
}

func GetGeocode(address string) (latitude, longitude float64) {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		os.Exit(1)
	}

	ApiKey := cfg.Section("google").Key("api_key").String()
	c, err := maps.NewClient(maps.WithAPIKey(ApiKey))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}
	r := &maps.GeocodingRequest{
		Address: address,
	}

	result, _ := c.Geocode(context.Background(), r)

	return result[0].Geometry.Location.Lat, result[0].Geometry.Location.Lng
}

func GetInfo(genre string, page int, writeMode string) {
	if file, _ := ioutil.ReadFile("data/" + genre + "." + writeMode); file != nil {
		return
	}
	filesInfo, _ := ioutil.ReadDir("html/" + genre)
	if filesInfo == nil {
		GetPage(genre)
		filesInfo, _ = ioutil.ReadDir("html/" + genre)
	}
	var stores []StoreInfo
	var wg sync.WaitGroup
	maxCh := make(chan int, 10)
	for i, fileInfo := range filesInfo {
		wg.Add(1)
		maxCh <- 1
		if i > page {
			wg.Done()
			break
		}
		go func(i int, fileInfo os.FileInfo) {
			time.Sleep(1 * time.Second)
			defer wg.Done()
			file, err := ioutil.ReadFile("html/" + genre + "/" + fileInfo.Name())
			if err != nil {
				log.Println(err)
				return
			}
			stringReader := strings.NewReader(string(file))
			doc, _ := goquery.NewDocumentFromReader(stringReader)

			var store StoreInfo
			store.Genre = genre
			doc.Find("ul.js-rstlist-info li.list-rst").Each(func(_ int, s *goquery.Selection) {
				store.URL, _ = s.Find("a.list-rst__rst-name-target.cpy-rst-name").Attr("href")
				store.Address = GetAddress(store.URL)
				store.Latitude, store.Longitude = GetGeocode(store.Address)
				store.StoreName = s.Find("a.list-rst__rst-name-target.cpy-rst-name").Text()
				store.Img, _ = s.Find("img.js-cassette-img").Attr("data-original")
				store.Point, _ = strconv.ParseFloat(s.Find("span.c-rating__val.c-rating__val--strong.list-rst__rating-val").Text(), 64)
				stores = append(stores, store)
			})
			log.Println(stores)
			<-maxCh
		}(i, fileInfo)
	}
	wg.Wait()

	if writeMode == "csv" {
		WriteCSV(stores, genre)
	} else if writeMode == "json" {
		WriteJson(stores, genre)
	}
}

func WriteCSV(stores []StoreInfo, genre string) {
	file, err := os.OpenFile("data/"+genre+".csv", os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	writer := csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))
	for _, store := range stores {
		writer.Write([]string{
			store.StoreName,
			strconv.FormatFloat(store.Point, 'f', 4, 64),
			store.URL,
			store.Img,
			store.Address,
			strconv.FormatFloat(store.Latitude, 'f', 4, 64),
			strconv.FormatFloat(store.Longitude, 'f', 4, 64),
		})
	}
	writer.Flush()
}

func WriteJson(stores []StoreInfo, genre string) {
	jsonStore, err := json.Marshal(stores)
	if err != nil {
		fmt.Println("JSON Marshal error:", err)
		return
	}

	out := new(bytes.Buffer)
	// プリフィックスなし、スペース4つでインデント
	json.Indent(out, jsonStore, "", "    ")

	ioutil.WriteFile("data/"+genre+".json", out.Bytes(), 0664)
}

func ReadJson(genre string) ([]StoreInfo, error) {
	file, err := ioutil.ReadFile("data/" + genre + ".csv")
	if err != nil {
		log.Fatalln(err)
	}
	var stores []StoreInfo

	if err = json.Unmarshal(file, &stores); err != nil {
		fmt.Println("JSON Marshal error:", err)
		return nil, err
	}
	return stores, nil
}
