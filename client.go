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

var modsdir, _ = filepath.Abs("testmods")
var remotehostport = "flungloaf.myftp.org:12312"

func remote() (files []string) {
	resp, err := http.Get("http://" + remotehostport + "/filelist")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	var list Filelist
	err = dec.Decode(&list)

	if err != nil {
		log.Fatal(err)
	}

	files = list.Files
	return
}

func local() (files []string) {
	list, err := ioutil.ReadDir(modsdir)

	if err != nil {
		log.Fatal(err)
	}

	for _, fi := range list {
		if !fi.IsDir() {
			files = append(files, fi.Name())
		}
	}

	return
}

func StringsToInterfaces(strings []string) []interface{} {
	vals := make([]interface{}, len(strings))
	for i, v := range strings {
		vals[i] = v
	}
	return vals
}

func main() {
	fmt.Println("Working with directory: " + modsdir)
	fmt.Println("Loading mod list from the server on " + remotehostport)

	remote := remote()
	local := local()

	remoteSet := mapset.NewSetFromSlice(StringsToInterfaces(remote))
	localSet := mapset.NewSetFromSlice(StringsToInterfaces(local))

	filesToDownload := remoteSet.Difference(localSet)
	filesToDelete := localSet.Difference(remoteSet)

	// Remove files
	for fileName := range filesToDelete.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Removing " + fileNameS)

		err := os.Remove(path.Join(modsdir, fileNameS))

		if err != nil {
			log.Fatal(err)
		}
	}

	// Download files
	for fileName := range filesToDownload.Iter() {
		fileNameS, _ := fileName.(string)

		fmt.Println("Downloading " + fileNameS)

		out, err := os.Create(path.Join(modsdir, fileNameS))
		defer out.Close()

		if err != nil {
			log.Fatal(err)
		}

		resp, err := http.Get("http://" + remotehostport + "/files/" + fileNameS)
		defer resp.Body.Close()

		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(out, resp.Body)

		if err != nil {
			log.Fatal(err)
		}
	}

	// fmt.Printf("%v\n", filesToDelete)
	// fmt.Printf("%v\n", filesToDownload)

	fmt.Println("Done!")
}
