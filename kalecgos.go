package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"strings"
	"log"
	"regexp"
	"net/http"
	"os"
	"io"
	"github.com/skratchdot/open-golang/open"
)

type addon struct {
	id string
	version string
	newVersion string
	hasNewVersion bool
	url string
	//successful bool
}

func main() {
	addonsDirectoryPointer := flag.String("addons-directory", "Interface/AddOns/", "Path to the Addons folder")

	flag.Parse()
	addonsDirectory := *addonsDirectoryPointer

	fmt.Println("Dir:", addonsDirectory)

	addons := getAddons(addonsDirectory)

	addons = addVersionDataToAddons(addons)

	f, err := os.Create("addons.html")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.WriteString("<html><body><h1>Addons:</h1><h2>Outdated:</h2><ul>\n")

	for _,addon := range addons {
		if addon.hasNewVersion {
			fmt.Println("Found newer version of", addon.id, "(", addon.version, "->", addon.newVersion, "): ", addon.url)
			f.WriteString("<li>Newer version of " + addon.id + " ( " + addon.version + " -> " + addon.newVersion + " ): <a href=\"" + addon.url + "\">Curse link</a></li>\n")
		}
	}

	f.WriteString("</ul><h2>Up do date:</h2><ul>\n")

	for _,addon := range addons {
		if !addon.hasNewVersion {
			fmt.Println("Addon", addon.id, "(", addon.version, ") is at the latest version")
			f.WriteString("<li>Addon " + addon.id + " ( " + addon.version + " ) is at the latest version</li>\n")
		}
	}

	f.WriteString("</ul></body></html>")
	f.Sync()
	open.Run("addons.html")
}

// Takes the path to the addons directory and returns a slice of addons
func getAddons(addonsDirectory string) []addon {
	addons := make([]addon, 0)
	addonDirectories, _ := ioutil.ReadDir(addonsDirectory)
	for _, addonDirectory := range addonDirectories {
		filesInAddonDirectory, err := ioutil.ReadDir(addonsDirectory + "/" + addonDirectory.Name() + "/")
		if err != nil {
			log.Fatal(err)
		}
		for _, addonFile := range filesInAddonDirectory {
			if strings.HasSuffix(addonFile.Name(), ".toc") {
				tocFile, err := ioutil.ReadFile(addonsDirectory + "/" + addonDirectory.Name() + "/" + addonFile.Name())
				if err != nil {
					log.Fatal(err)
				}
				id, version := getAddonProperties(addonDirectory.Name(), string(tocFile))
				addon := addon{id: id, version: version}
				if id != "" && !contains(addons, addon){
					addons = append(addons, addon)
				}
			}
		}
	}
	return addons
}

func addVersionDataToAddons(addons []addon) []addon {
	result := make([]addon, 0)
	for _, oldAddon := range addons {
		oldAddon.url = createAddonUrl(oldAddon.id)
		page := getWebpage(oldAddon.url)
		newestVersion := getAddonVersionFromCurseWebpage(page)
		id := oldAddon.id
		version := oldAddon.version
		if oldAddon.version != newestVersion {
			newAddon := addon{id: id, version: version, newVersion: newestVersion, hasNewVersion: true}
			result = append(result, newAddon)
		} else {
			newAddon := addon{id: id, version: version, hasNewVersion: false}
			result = append(result, newAddon)
		}
	}
	return result
}

func getAddonProperties(addon string, tocFile string) (string, string) {
	pattern, err := regexp.Compile(`X-Curse-Project-ID: (.*)`)
	if err != nil {
		log.Fatal(err)
	}
	rawId := pattern.FindStringSubmatch(tocFile)
	if len(rawId) == 0 {
		log.Println("Didn't find X-Curse-Project-ID for addon :" + addon)
		return "", ""
	}
	fixedId := fixParsedString(rawId[1])

	pattern, err = regexp.Compile(`X-Curse-Packaged-Version: (.*)`)
	if err != nil {
		log.Fatal(err)
	}

	rawVersion := pattern.FindStringSubmatch(tocFile)
	fixedVersion := fixParsedString(rawVersion[1])

	return fixedId, fixedVersion
}

func fixParsedString(str string) string {
	lastCharacter := str[len(str) - 1:]
	if lastCharacter[0] == 13 {
		// For some reason we sometimes get a stray ascii character 13 at the end
		return str[:len(str) - 1]
	}
	return str
}

func createAddonUrl(id string) string {
	return "https://mods.curse.com/addons/wow/" + id
}

func getWebpage(url string) string {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
		return ""
	} else {
		defer response.Body.Close()
		if response.StatusCode != 200 {
			log.Println("Wrong status code: " + response.Status)
			log.Fatal("Body: " + responseToString(response.Body))
		}
		return string(responseToString(response.Body))
	}
}

func responseToString(body io.ReadCloser) string {
	bs, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bs)
}

func getAddonVersionFromCurseWebpage(html string) string {
	pattern, err := regexp.Compile(`<li class="newest-file">Newest File: (.*)</li>`)
	if err != nil {
		log.Fatal(err)
	}
	version := pattern.FindStringSubmatch(html)
	return version[1]
}

func contains(s []addon, e addon) bool {
    for _, a := range s {
        if a.id == e.id {
            return true
        }
    }
    return false
}
