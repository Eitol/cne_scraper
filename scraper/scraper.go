package scraper

import (
	"encoding/gob"
	"fmt"
	"github.com/Eitol/cne_scraper/cne"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	StartIDX          int    `env:"START_IDX"`
	EndIDX            int    `env:"END_ID"`
	NumThreads        int    `env:"NUM_THREADS"`
	ChunkSize         int    `env:"CHUNK_SIZE"`
	PersonOutputDir   string `env:"PERSON_OUTPUT_DIR"`
	FailedOutputDir   string `env:"FAILED_OUTPUT_DIR"`
	LatestIDXFileName string `env:"LATEST_IDX_FILE_NAME"`
}

type Scraper struct {
	config Config

	personMutex       sync.Mutex
	failedPersonList  []int
	failedPersonMutex sync.Mutex
	wg                sync.WaitGroup
	personList        []cne.Person
	idChan            chan int
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Fatalf("Error closing file: %v", err)
	}
}

func (s *Scraper) readLatestIDXFile() int {
	var latestID int
	file, err := os.Open(s.config.LatestIDXFileName)
	if err != nil {
		return s.config.StartIDX
	}
	defer closeFile(file)
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&latestID)
	if err != nil {
		return s.config.StartIDX
	}
	return latestID
}

func (s *Scraper) saveLatestIDXFile(id int) {
	file, err := os.Create(s.config.LatestIDXFileName)
	if err != nil {
		return
	}
	defer closeFile(file)
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(id)
	if err != nil {
		return
	}
}

func (s *Scraper) savePersonList(personList []cne.Person, failedPersonList []int, id int) {
	err := saveList(personList, id, s.config.PersonOutputDir)
	if err != nil {
		log.Fatalf("Error saving person list: %v", err)
	}
	err = saveList(failedPersonList, id, s.config.FailedOutputDir)
	if err != nil {
		log.Fatalf("Error saving failed person list: %v", err)
	}
}

func saveList[V any](personList V, id int, path string) error {
	fileName := filepath.Join(path, strconv.Itoa(id)+".gob")
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer closeFile(file)
	err = gob.NewEncoder(file).Encode(personList)
	if err != nil {
		return err
	}
	return nil
}

func (s *Scraper) Scrap() {
	startIDX := s.readLatestIDXFile()
	log.Printf("Starting from ID: %d\n", startIDX)
	s.idChan = make(chan int)
	s.initializeOutputDirectories()
	s.startScrapJobs()
	s.startDataSavingJob(startIDX)
	s.wg.Wait()
}

func (s *Scraper) startDataSavingJob(startIDX int) {
	latestChunkProcessedID := 0
	beginTime := time.Now()
	// Enviar los IDs a las goroutines
	for id := startIDX + 1; id <= s.config.EndIDX; id++ {
		s.idChan <- id
		if id%1000 == 0 {
			fmt.Printf("ID: %d | success %d | failed | %d\n", id, len(s.personList), len(s.failedPersonList))
		}
		if len(s.personList) >= s.config.ChunkSize {
			totalTime := time.Since(beginTime)
			fmt.Printf("Saving chunk | ID: %d | time: %v | total %d ", id, totalTime, id-latestChunkProcessedID)
			s.personMutex.Lock()
			s.failedPersonMutex.Lock()

			s.savePersonList(s.personList, s.failedPersonList, id)
			s.saveLatestIDXFile(id)
			// clear lists
			s.personList = s.personList[:0]
			s.failedPersonList = s.failedPersonList[:0]

			s.personMutex.Unlock()
			s.failedPersonMutex.Unlock()
			latestChunkProcessedID = id
			beginTime = time.Now()
		}
	}
	// Cerrar el canal y esperar a que todas las goroutines terminen
	close(s.idChan)
}

func (s *Scraper) startScrapJobs() {
	for i := 0; i < s.config.NumThreads; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			client := cne.NewCNEClient()
			for id := range s.idChan {
				person, err := client.GetPersonByDocID(id)
				if err != nil || person.Name == "" {
					s.failedPersonMutex.Lock()
					s.failedPersonList = append(s.failedPersonList, id)
					s.failedPersonMutex.Unlock()
					continue
				}
				s.personMutex.Lock()
				s.personList = append(s.personList, *person)
				s.personMutex.Unlock()
			}
		}()
	}
}

func (s *Scraper) initializeOutputDirectories() {
	err := os.MkdirAll(s.config.PersonOutputDir, 0777)
	if err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}
	err = os.MkdirAll(s.config.FailedOutputDir, 0777)
	if err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}
}
