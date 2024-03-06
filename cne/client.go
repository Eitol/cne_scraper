package cne

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/valyala/fasthttp"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Person struct {
	Cedula       int
	Name         string
	State        string
	Municipality string
	Parish       string
}

type Client struct {
	client     *fasthttp.HostClient
	readBuffer []byte
}

func NewCNEClient() *Client {
	return &Client{
		client:     httpClientSingleton,
		readBuffer: make([]byte, responseSizeLimit),
	}
}

func (c *Client) GetPersonByDocID(id int) (*Person, error) {
	doc, err := c.getPersonByDocID(id, 0)
	if err != nil {
		return nil, err
	}
	return c.extractPersonDataFromHTMLDoc(id, doc)
}

func (c *Client) getPersonByDocID(id int, attempt int) (*goquery.Document, error) {
	url := baseURL + strconv.Itoa(id)
	isError := false
	statusCode, body, err := c.client.Get(c.readBuffer, "http://"+url)
	if err != nil {
		isError = true
		log.Printf("error doing request: %s", err)
	}
	if statusCode != fasthttp.StatusOK {
		isError = true
		log.Printf("Unexpected status code: %d. Expecting %d", statusCode, fasthttp.StatusOK)
	}
	if isError {
		if attempt < 5 {
			randSleepTime := rand.Int() % (20)
			sleepTime := 10 + attempt*3 + randSleepTime
			time.Sleep(time.Second * time.Duration(sleepTime))
			return c.getPersonByDocID(id, attempt+1)
		}
		return nil, err
	}
	if strings.Contains(string(body), "dula de identidad no se encuentra inscrito en el Registro Electoral") {
		return nil, ErrNotFound
	}
	if strings.Contains(string(body), "dula de identidad presenta una objeción por lo que no podrá ejercer su derecho al voto") {
		return nil, ErrUserBlocked
	}
	reader := io.NopCloser(bytes.NewReader(body))

	// Analizar la respuesta HTML con goquery
	return goquery.NewDocumentFromReader(reader)
}

func (c *Client) extractPersonDataFromHTMLDoc(id int, doc *goquery.Document) (*Person, error) {
	// Extraer información del documento
	person := &Person{Cedula: id}
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		s.Find("tr").Each(func(j int, s *goquery.Selection) {
			s.Find("td").Each(func(k int, s *goquery.Selection) {
				switch {
				case strings.Contains(s.Text(), "Nombre:"):
					person.Name = strings.TrimSpace(s.Next().Text())
				case strings.Contains(s.Text(), "Estado:"):
					st := strings.TrimSpace(s.Next().Text())
					person.State = estateReplacer.Replace(st)
				case strings.Contains(s.Text(), "Municipio:"):
					mp := strings.TrimSpace(s.Next().Text())
					person.Municipality = munReplacer.Replace(mp)
				case strings.Contains(s.Text(), "Parroquia:"):
					p := strings.TrimSpace(s.Next().Text())
					person.Parish = parishReplacer.Replace(p)
				}
			})
		})
	})
	if person.Name == "" {
		return nil, errors.New("empty name")
	}
	if person.State == "" {
		return nil, errors.New("empty state")
	}
	if person.Municipality == "" {
		return nil, errors.New("empty municipality")
	}
	if person.Parish == "" {
		return nil, errors.New("empty parish")
	}
	return person, nil
}
