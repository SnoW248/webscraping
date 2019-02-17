package main

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
	"webscraping/utils"

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
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func GetPage(baseUrl string) {
	base, _ := url.Parse(baseUrl)
	for page := 1; page < 61; page++ {
		reference, _ := url.Parse(strconv.Itoa(page) + "/?Srt=D&SrtT=rt&sort_mode=1&sk=%E3%83%A9%E3%83%BC%E3%83%A1%E3%83%B3&svd=20190216&svt=1900&svps=2")
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
		if err := ioutil.WriteFile("ramen/"+title, []byte(body), 0666); err != nil {
			fmt.Println("write file err")
		}
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

func GetInfo(dir string, page int, writeMode string) {
	filesInfo, _ := ioutil.ReadDir(dir)
	var stores []StoreInfo
	for i, fileInfo := range filesInfo {
		if i > page {
			break
		}
		file, _ := ioutil.ReadFile(dir + "/" + fileInfo.Name())
		stringReader := strings.NewReader(string(file))
		doc, _ := goquery.NewDocumentFromReader(stringReader)

		var store StoreInfo
		store.Genre = dir
		doc.Find("ul.js-rstlist-info li.list-rst").Each(func(_ int, s *goquery.Selection) {
			store.URL, _ = s.Find("a.list-rst__rst-name-target.cpy-rst-name").Attr("href")
			store.Address = GetAddress(store.URL)
			store.Latitude, store.Longitude = GetGeocode(store.Address)
			store.StoreName = s.Find("a.list-rst__rst-name-target.cpy-rst-name").Text()
			store.Point, _ = strconv.ParseFloat(s.Find("span.c-rating__val.c-rating__val--strong.list-rst__rating-val").Text(), 64)
			stores = append(stores, store)
		})
		log.Println(stores)
	}
	
	if writeMode == "csv" {
		WriteCSV(stores)
	} else if writeMode == "json" {
		WriteJson(stores)
	}
}

func WriteCSV(stores []StoreInfo) {
	file, err := os.OpenFile("csv/ramen.csv", os.O_CREATE|os.O_APPEND, 0600)
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
			store.Address,
			strconv.FormatFloat(store.Latitude, 'f', 4, 64),
			strconv.FormatFloat(store.Longitude, 'f', 4, 64),
		})
	}
	writer.Flush()
}

func WriteJson(stores []StoreInfo) {
	jsonStore, err := json.Marshal(stores)
	if err != nil {
		fmt.Println("JSON Marshal error:", err)
		return
	}

	out := new(bytes.Buffer)
	// プリフィックスなし、スペース4つでインデント
	json.Indent(out, jsonStore, "", "    ")

	ioutil.WriteFile("csv/ramen.json", out.Bytes(), 0664)
}

func main() {
	utils.LoggingSettings("webscraping.log")
	// GetPage("https://tabelog.com/rstLst/ramen/")
	GetInfo("ramen", 3, "csv")
}
