package main

import (
	"bytes"
	"golang.org/x/net/html/charset"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
	"io"
	"strings"
	//"strconv"
)

type RssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Items   []RssItem `xml:"channel>item"`
}

type RssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
}

type Material struct {
	Title       string
}

type Announcement struct {
	Title       string
	Description string
}

var titleType [2]string = [...]string {
	"Material",
	"Announcement",
}

func fetchURL(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("unable to GET '%s': %s", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("unable to read body '%s': %s", url, err)
	}
	return body
}
func BypassReader(label string, input io.Reader) (io.Reader, error) {
	return input, nil
}

func parseXML(xmlDoc []byte, target interface{}) {

	nr, err := charset.NewReaderLabel("utf-16", bytes.NewReader(xmlDoc))
	if err != nil {
		panic(err)
	}
	decoder := xml.NewDecoder(nr)
	decoder.CharsetReader = BypassReader
	 //Fixes "xml: encoding \"UTF-16\" declared but Decoder.CharsetReader is nil"

	if err := decoder.Decode(target); err != nil {
		log.Fatalf("unable to parse XML '%s':\n%s", err, xmlDoc)
	}
}

func UTF16Bom(b []byte) int8 {
	if len(b) < 2 {
		return -1
	}

	if b[0] == 0xFE && b[1] == 0xFF {
		return 1
	}

	if b[0] == 0xFF && b[1] == 0xFE {
		return 2
	}

	return 0
}

func DecodeUTF16(b []byte) (string, error) {

	if len(b)%2 != 0 {
		return "" , fmt.Errorf("Must have even length byte slice")
	}

	bom := UTF16Bom(b)
	if bom < 0 {
		return "" , fmt.Errorf("Buffer is too small")
	}

	u16s := make([]uint16, 1)
	ret := &bytes.Buffer{}
	b8buf := make([]byte, 4)
	lb := len(b)

	for i := 0; i < lb; i += 2 {
		//assuming bom is big endian if 0 returned
		if bom == 0 || bom == 1 {
			u16s[0] = uint16(b[i+1]) + (uint16(b[i]) << 8)
		}
		if bom == 2 {
			u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		}
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write([]byte(string(b8buf[:n])))
	}

	return ret.String(), nil
}

// func ParseRSSMetCourseFeed(id int) ([]Announcement,[]Material){
// 	var rssFeed = &RssFeed{}
// 	var rssMaterial []Material
// 	var rssAnnouncement []Announcement
// 	xmlDoc := fetchURL("http://met.guc.edu.eg/Feeds/Course.ashx?c="+strconv.Itoa(id))
// 	parseXML(xmlDoc, &rssFeed)
// 	for _, item := range rssFeed.Items {
// 		if strings.Contains(item.Title,titleType[0]){
// 			rssMaterial = append(rssMaterial,Material{item.Title})
// 		} else if strings.Contains(item.Title,titleType[1]){
// 			rssAnnouncement = append(rssAnnouncement,Announcement{item.Title,item.Description})
// 		}
// 	}
// 	return rssAnnouncement , rssMaterial
// }
func ParseRSSMetCourseFeed(id string) []Announcement {
	var rssFeed = &RssFeed{}
	var rssAnnouncement []Announcement
	xmlDoc := fetchURL("http://met.guc.edu.eg/Feeds/Course.ashx?c=" + id)
	parseXML(xmlDoc, &rssFeed)
	for _, item := range rssFeed.Items {
		if strings.Contains(item.Title, "Announcement") {
			rssAnnouncement = append(rssAnnouncement,Announcement{item.Title,item.Description})
		}
	}
	return rssAnnouncement
}

// func main(){
//   fmt.Println(ParseRSSMetCourseFeed("759"))
// }
