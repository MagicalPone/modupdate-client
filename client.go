package main

import (
	"encoding/json"
	"fmt"
	"github.com/deckarep/golang-set"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

type file string

func (f file) String() string {
	return string(f)
}

type config struct {
	ModsDir, Server string
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func fetchRemoteList(url string) []file {
	resp, err := http.Get(url)
	assert(err)
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	// Local struct for decoding input from the server
	type filesList struct {
		Files []file
	}

	list := new(filesList)
	err = dec.Decode(list)
	assert(err)

	return (*list).Files
}

func fetchLocalList(dir string) []file {
	list, err := ioutil.ReadDir(dir)
	assert(err)

	// Allocate space for at most len(list) items
	files := make([]file, 0, len(list))

	for _, fi := range list {
		if !fi.IsDir() { // skip all the directories
			files = append(files, file(fi.Name()))
		}
	}

	return files
}

func loadConfig(filename string) (config config) {
	configFile, err := os.Open(filename)
	assert(err)
	defer configFile.Close()

	dec := json.NewDecoder(configFile)
	err = dec.Decode(&config)
	assert(err)

	config.Server = os.ExpandEnv(config.Server)
	config.ModsDir, err = filepath.Abs(os.ExpandEnv(config.ModsDir))
	assert(err)

	return
}

func newFilesSet(files []file) mapset.Set {
	set := mapset.NewSet()
	for f := range files {
		set.Add(f)
	}
	return set
}

func asyncFetchSet(f func() mapset.Set) <-chan mapset.Set {
	c := make(chan mapset.Set, 1)
	go func() {
		c <- f()
	}()
	return c
}

func main() {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}

	config := loadConfig(configFileName)

	fmt.Println("Working with directory: " + config.ModsDir)
	fmt.Println("Loading mod list from the server on " + config.Server)

	// Run async tasks
	remotec := asyncFetchSet(func() mapset.Set {
		return newFilesSet(fetchRemoteList("http://" + config.Server + "/filelist"))
	})
	localc := asyncFetchSet(func() mapset.Set {
		return newFilesSet(fetchLocalList(config.ModsDir))
	})

	// Wait for data
	localSet := <-localc
	remoteSet := <-remotec

	filesToDownload := remoteSet.Difference(localSet)
	filesToDelete := localSet.Difference(remoteSet)

	// Remove files
	for fileName := range filesToDelete.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Removing " + fileNameS)

		err := os.Remove(path.Join(config.ModsDir, fileNameS))
		assert(err)
	}

	// Download files
	for fileName := range filesToDownload.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Downloading " + fileNameS)

		out, err := os.Create(path.Join(config.ModsDir, fileNameS))
		defer out.Close()
		assert(err)

		resp, err := http.Get("http://" + config.Server + "/files/" + fileNameS)
		assert(err)
		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)

		assert(err)
	}

	// fmt.Printf("%v\n", filesToDelete)
	// fmt.Printf("%v\n", filesToDownload)

	fmt.Println("Done!")
}
