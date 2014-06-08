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

type Filelist struct {
	Files []string
}

type Config struct {
	ModsDir, Server string
}

func Assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Must(val interface{}, err error) interface{} {
	Assert(err)
	return val
}

func FetchRemoteList(url string) (files []string) {
	resp, err := http.Get(url)
	Assert(err)
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	var list Filelist
	err = dec.Decode(&list)
	Assert(err)

	files = list.Files
	return
}

func FetchLocalList(dir string) []string {
	list, err := ioutil.ReadDir(dir)
	Assert(err)

	// Allocate space for at most len(list) strings
	files := make([]string, 0, len(list))

	for _, fi := range list {
		if !fi.IsDir() { // skip all the directories
			files = append(files, fi.Name())
		}
	}

	return files
}

func StringsToInterfaces(strings []string) []interface{} {
	vals := make([]interface{}, len(strings))
	for i, v := range strings {
		vals[i] = v
	}
	return vals
}

func LoadConfig(filename string) (config Config) {
	configFile, err := os.Open(filename)
	Assert(err)

	defer configFile.Close()
	dec := json.NewDecoder(configFile)
	err = dec.Decode(&config)
	Assert(err)

	config.Server = os.ExpandEnv(config.Server)
	config.ModsDir, err = filepath.Abs(os.ExpandEnv(config.ModsDir))
	Assert(err)

	return
}

func main() {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}

	config := LoadConfig(configFileName)

	fmt.Println("Working with directory: " + config.ModsDir)
	fmt.Println("Loading mod list from the server on " + config.Server)

	remote := FetchRemoteList("http://" + config.Server + "/filelist")
	local := FetchLocalList(config.ModsDir)

	remoteSet := mapset.NewSetFromSlice(StringsToInterfaces(remote))
	localSet := mapset.NewSetFromSlice(StringsToInterfaces(local))

	filesToDownload := remoteSet.Difference(localSet)
	filesToDelete := localSet.Difference(remoteSet)

	// Remove files
	for fileName := range filesToDelete.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Removing " + fileNameS)

		err := os.Remove(path.Join(config.ModsDir, fileNameS))
		Assert(err)
	}

	// Download files
	for fileName := range filesToDownload.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Downloading " + fileNameS)

		out, err := os.Create(path.Join(config.ModsDir, fileNameS))
		defer out.Close()
		Assert(err)

		resp, err := http.Get("http://" + config.Server + "/files/" + fileNameS)
		Assert(err)
		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)

		Assert(err)
	}

	// fmt.Printf("%v\n", filesToDelete)
	// fmt.Printf("%v\n", filesToDownload)

	fmt.Println("Done!")
}
