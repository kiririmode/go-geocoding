package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
)

func main() {
	var infile = flag.String("i", "", "住所が入ったCSVファイル")
	var outfile = flag.String("o", "", "出力するCSVファイル")

	flag.Parse()

	if *infile == "" {
		fmt.Fprintf(os.Stderr, "specify csv file with -i\n")
		os.Exit(1)
	}
	if *outfile == "" {
		fmt.Fprintf(os.Stderr, "specify output file with -o\n")
		os.Exit(1)
	}

	run(infile, outfile)
}

type Geo struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

type App struct {
	AddressFile   *os.File
	GeoDecodeFile *os.File
}

// NewApp creates a new application with input and output file pointer.
func newApp(addressFile, decodedFile *os.File) *App {
	return &App{
		AddressFile:   addressFile,
		GeoDecodeFile: decodedFile,
	}
}

func run(infile, outfile *string) error {

	infp, err := os.Open(*infile)
	if err != nil {
		return fmt.Errorf("open %s: ", *infile)
	}
	defer infp.Close()

	outfp, err := os.Create(*outfile)
	if err != nil {
		return fmt.Errorf("open %s: ", *outfile)
	}
	defer outfp.Close()

	reader := csv.NewReader(infp)
	reader.LazyQuotes = true

	client := &http.Client{}

	for {
		records, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		lat, lng, err := geoDecode(records[0], client)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(outfp, "%s,%sN,%s,%s\n", records[0], convert(lat), fmt.Sprint(lat), convert(lng))
	}
	outfp.Sync()

	return nil
}

// 10 進数による座標を 60 進数(度分秒)に変換する
// ref. http://www.benricho.org/map_latlng_10-60conv/
func convert(n float64) string {
	degree := math.Trunc(n)
	leftover := (n - degree) * 60

	minute := (int)(math.Trunc(leftover))
	leftover -= (float64)(minute)

	second := leftover * 60
	return fmt.Sprintf("%d°%d'%3.1f\"", int(degree), minute, second)
}

func geoDecode(address string, client *http.Client) (lat, lng float64, err error) {
	values := url.Values{}
	values.Add("address", address)

	// geo decoding using google maps api
	req, err := http.NewRequest("GET", "https://maps.googleapis.com/maps/api/geocode/json", nil)
	if err != nil {
		return -1, -1, err
	}
	req.URL.RawQuery = values.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return -1, -1, err
	}

	var geo Geo
	err2 := decodeBody(resp, &geo)
	if err2 != nil {
		return -1, -1, err
	}

	l := geo.Results[0].Geometry.Location
	return l.Lat, l.Lng, nil
}

func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(out)
}
