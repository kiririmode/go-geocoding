package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/sync/errgroup"
)

func main() {
	var infile = flag.String("i", "", "住所が入ったCSVファイル")
	var outfile = flag.String("o", "", "出力するCSVファイル")

	flag.Parse()

	if *infile == "" {
		fmt.Fprintf(os.Stderr, "specify csv file with -i\n")
		os.Exit(1)
	}
	if err := newApp(*infile, *outfile).run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
}

type geo struct {
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

// App represents this application
type App struct {
	AddressFile   string
	GeoDecodeFile string
	Client        *http.Client
}

// NewApp creates a new application with input and output file pointer.
func newApp(infile, outfile string) *App {
	return &App{
		AddressFile:   infile,
		GeoDecodeFile: outfile,
		Client:        &http.Client{},
	}
}

func (app *App) run() error {

	infp, err := os.Open(app.AddressFile)
	if err != nil {
		return fmt.Errorf("open %s: ", app.AddressFile)
	}
	defer infp.Close()

	var outfp *os.File
	if app.GeoDecodeFile != "" {
		outfp, err = os.Create(app.GeoDecodeFile)
	} else {
		outfp = os.Stdout
	}
	if err != nil {
		return fmt.Errorf("open %s: ", app.GeoDecodeFile)
	}
	defer outfp.Close()

	eg, ctx := errgroup.WithContext(context.Background())
	q := make(chan string, 1000)

	eg.Go(func() error {
		return app.enqueue(ctx, infp, q)
	})

	eg.Go(func() error {
		return app.putGeocode(ctx, outfp, q)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (app *App) enqueue(ctx context.Context, fp *os.File, q chan<- string) error {
	reader := csv.NewReader(fp)
	reader.LazyQuotes = true

	for {
		records, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error while reading %s", fp.Name())
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case q <- records[0]:
		}
	}
	close(q)
	return nil
}

func (app *App) putGeocode(ctx context.Context, fp *os.File, q <-chan string) error {
	for address := range q {
		lat, lng, err := app.geocode(ctx, address)
		if err != nil {
			return fmt.Errorf("decode %s", err)
		}
		fmt.Fprintf(fp, "%s,%sN,%sE\n", address, convert(lat), convert(lng))
	}
	fp.Sync()
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

func (app *App) geocode(ctx context.Context, address string) (lat, lng float64, err error) {
	values := url.Values{}
	values.Add("address", address)

	req, err := http.NewRequest("GET", "https://maps.googleapis.com/maps/api/geocode/json", nil)
	if err != nil {
		return -1, -1, err
	}
	req = req.WithContext(ctx)

	req.URL.RawQuery = values.Encode()
	resp, err := app.Client.Do(req)
	if err != nil {
		return -1, -1, err
	}
	defer resp.Body.Close()

	var geo geo
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&geo)
	if err != nil {
		return -1, -1, err
	}

	l := geo.Results[0].Geometry.Location
	return l.Lat, l.Lng, nil
}
