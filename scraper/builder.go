package scraper

import (
	"github.com/Eitol/goconf"
	"log"
)

const (
	defaultStartIDX   = 1_000_000
	endID             = 30000000
	numThreads        = 100
	chunkSize         = 10_000
	personOutputDir   = "success"
	failedOutputDir   = "failed"
	latestIDXFileName = "latest.idx"
)

func BuildScraper() *Scraper {
	config := Config{
		StartIDX:          defaultStartIDX,
		EndIDX:            endID,
		NumThreads:        numThreads,
		ChunkSize:         chunkSize,
		PersonOutputDir:   personOutputDir,
		FailedOutputDir:   failedOutputDir,
		LatestIDXFileName: latestIDXFileName,
	}
	err := goconf.Extract(goconf.ExtractorArgs{
		Options: goconf.ExtractorOptions{
			EnvFile:               ".env", // env file path
			OmitEnvFileIfNotExist: true,
		},
		Configs: []interface{}{
			//  Config struct | env name prefix
			&config, "CNE_SCRAPER",
		},
	})
	if err != nil {
		log.Fatalf("Error extracting configuration: %v", err)
	}
	return &Scraper{
		config: config,
	}
}
