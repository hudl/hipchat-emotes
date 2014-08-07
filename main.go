package main

import (
	"encoding/json"
	"fmt"
	"github.com/pmylund/go-cache"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var EmoticonCache *cache.Cache = cache.New(60*time.Minute, 5*time.Minute)
var logger *log.Logger = log.New(os.Stdout, "log: ", log.LstdFlags)

const CacheKey string = "emoticons"

type EmoticonResponse struct {
	Items []Emoticon        `json:"items"`
	Links map[string]string `json:"links"`
}

type Emoticon struct {
	ImageUrl string `json:"url"`
	Shortcut string `json:"shortcut"`
}

func main() {
	http.HandleFunc("/", handle)
	staticfs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticfs))
	logger.Println(http.ListenAndServe(":6070", nil))
}

func handle(writer http.ResponseWriter, request *http.Request) {
	logger.Println("handling")
	emoticons := GetEmoticons()
	if emoticons == nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	mainTemplate, err := template.ParseFiles("main.tmpl")
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		logger.Fatalln(err)
		return
	}

	writer.WriteHeader(http.StatusOK)
	mainTemplate.Execute(writer, emoticons)
}

func GetEmoticons() *[]Emoticon {
	if emoticons, found := EmoticonCache.Get(CacheKey); found {
		logger.Println("Emoticons retrieved from cache")
		return emoticons.(*[]Emoticon)
	}

	url := "https://api.hipchat.com/v2/emoticon"
	emoticons := make([]Emoticon, 0, 200)
	for url != "" {
		response := getEmoticonsPage(url)
		emoticons = append(emoticons, response.Items...)
		url = response.Links["next"]
	}

	EmoticonCache.Set(CacheKey, &emoticons, 0)

	return &emoticons
}

func getEmoticonsPage(url string) *EmoticonResponse {
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("Authorization", "Bearer "+os.Getenv("HIPCHAT_API_TOKEN"))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		logger.Println("Something went wrong:", err)
		return nil
	}

	defer response.Body.Close()

	emoticonData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Something went wrong:", err)
		return nil
	}
	var emoticons EmoticonResponse
	json.Unmarshal(emoticonData, &emoticons)

	return &emoticons
}
