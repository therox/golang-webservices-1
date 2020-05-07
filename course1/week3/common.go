package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	// "log"
)

const filePath string = "./data/users.txt"

func SlowSearch(out io.Writer) {
	//file, err := os.Open(filePath)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fileContents, err := ioutil.ReadAll(file)
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	//lines := strings.Split(string(fileContents), "\n")
	//contents := bytes.Join([][]byte{[]byte("["), bytes.Replace(fileContents, []byte("\n"), []byte(","), -1), []byte("]")}, []byte(""))
	txt := new(bytes.Buffer)
	txt.Grow(len(fileContents) + 2)
	txt.WriteString("[")

	txt.Write(bytes.ReplaceAll(fileContents, []byte("\n"), []byte(",")))
	txt.WriteString("]")
	//contents := []byte(txt.String())

	var users []map[string]interface{}
	err = json.Unmarshal(txt.Bytes(), &users)
	if err != nil {
		panic(err)
	}

	//for _, line := range lines {
	//	user := make(map[string]interface{})
	//	// fmt.Printf("%v %v\n", err, line)
	//	err := json.Unmarshal([]byte(line), &user)
	//	if err != nil {
	//		panic(err)
	//	}
	//	users = append(users, user)
	//}

	for i, user := range users {

		isAndroid := false
		isMSIE := false

		browsers, ok := user["browsers"].([]interface{})
		if !ok {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}
			if strings.Contains(browser, "Android") {
				//if ok, err := regexp.MatchString("Android", browser); ok && err == nil {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}
			if strings.Contains(browser, "MSIE") {
				//if ok, err := regexp.MatchString("MSIE", browser); ok && err == nil {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := r.ReplaceAllString(user["email"].(string), " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
