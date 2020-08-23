package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tormoder/fit"
)

type Summary struct {
	Sport     string
	Timestamp int64
	Duration  string
	Distance  float32
}

func Summarize(fitd *fit.File) (*Summary, error) {
	activity, err := fitd.Activity()
	if err != nil {
		return nil, err
	}
	if len(activity.Sessions) != 1 {
		return nil, errors.New("got activity with num sessions != 1")
	}
	sess := activity.Sessions[0]

	dur := fmt.Sprintf("%s", time.Duration(activity.Activity.TotalTimerTime)*time.Millisecond)

	return &Summary{
		Sport:     sess.Sport.String(),
		Timestamp: fitd.FileId.TimeCreated.Unix(),
		Duration:  dur,
		Distance:  float32(sess.TotalDistance) / 100.0 / 1000.0, // cm to km
	}, nil
}

type GPS struct {
	Time time.Time
	Lat  float64
	Lng  float64
}

type Data struct {
	Summary *Summary
	Coords  []GPS
}

func ReadData(fname string) (*Data, error) {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	fitd, err := fit.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	sum, err := Summarize(fitd)
	if err != nil {
		return nil, err
	}

	activity, err := fitd.Activity()
	if err != nil {
		return nil, err
	}

	var d Data
	d.Summary = sum
	for _, rec := range activity.Records {
		if rec.PositionLat.Invalid() || rec.PositionLong.Invalid() {
			continue
		}
		d.Coords = append(d.Coords, GPS{
			Time: rec.Timestamp,
			Lat:  rec.PositionLat.Degrees(),
			Lng:  rec.PositionLong.Degrees(),
		})
	}

	return &d, nil
}

type Server struct {
	apikey       string
	basedir      string
	htmlTemplate *template.Template
	jsTemplate   *template.Template
}

func New(apikey, basedir string) (*Server, error) {
	html, err := template.ParseFiles("templates/map.html")
	if err != nil {
		return nil, err
	}
	js, err := template.ParseFiles("templates/map.js")
	if err != nil {
		return nil, err
	}

	return &Server{
		apikey:       apikey,
		basedir:      basedir,
		htmlTemplate: html,
		jsTemplate:   js,
	}, nil
}

func (s *Server) ServeJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	s.jsTemplate.Execute(w, "")
}

func (s *Server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(fmt.Sprintf("%s/*.FIT", s.basedir))
	for idx, f := range files {
		rel, err := filepath.Rel(s.basedir, f)
		if err != nil {
			panic(err)
		}
		files[idx] = rel
	}
	if err != nil {
		panic(err)
	}
	s.htmlTemplate.Execute(w, struct {
		Key   string
		Files []string
	}{
		Key:   s.apikey,
		Files: files,
	})
}

func (s *Server) ServeActivity(w http.ResponseWriter, r *http.Request) {
	acts, ok := r.URL.Query()["fit"]
	if !ok || len(acts) != 1 {
		return
	}
	act := acts[0]
	data, err := ReadData(path.Join(s.basedir, act))
	if err != nil {
		log.Printf("%s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func run() error {
	basedir := flag.String("basedir", "", "base directory for activities")
	flag.Parse()
	if *basedir == "" {
		return errors.New("basedir must be set")
	}
	key, err := ioutil.ReadFile("gcp.key")
	if err != nil {
		return err
	}
	s, err := New(strings.TrimSpace(string(key)), *basedir)
	if err != nil {
		return err
	}
	http.HandleFunc("/", s.ServeIndex)
	http.HandleFunc("/map.js", s.ServeJS)
	http.HandleFunc("/activity", s.ServeActivity)
	return http.ListenAndServe(":8080", nil)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
