package main


import helpers "./helpers"
import lib "./lib"
import xmlFormats "./xmlFormats"
import (
    "flag"
	"fmt"
	"strconv"
	"net/http"
	"encoding/xml"
	"time"
	"errors"
	"io/ioutil"
    "os"
    "bufio"
	"strings"
	"io"
	"path/filepath"
	"sync"
)

var THIS_FOLDER_NAME string = "RSS youtube feed"

// default behaviour:
// go through file named `feeds.txt` and try to GET videos
// not older than `weekOld` from every channel in that file
// by storing data from youtube.com into xml structure defined in Feed.go
// then output to console 1.video title 2.how long ago was video published 3.video url
func main() {
	
	const fileWithUnixTimeName string = "last_update_UnixTime.txt"
	
	const fi string = "feeds.txt"
	
	const fileWithUnixTimeAppendName string = "backup/passedChangedUnixTimes.txt"
	
	
	// parse args
	aRet, bRet, cRet, dRet, eRet := parseArgs(fileWithUnixTimeName, fi, fileWithUnixTimeAppendName)
	var opmlFileName 							string = aRet
	var PARAM_OPML_DEFAULT_VALUE	string = bRet
	var hrs 											int = cRet
	var ignore404 									bool = dRet
	var openall 										bool = eRet
	
	
	// this gets populated by either list of urls from `fi` text file or by urls from opmlFileName opml file
	var fetchedDataItems []FetchedDataItem
	populate_fetchedDataItems(&fetchedDataItems, opmlFileName, PARAM_OPML_DEFAULT_VALUE, fi)
	
	
	var lstOfUrls []string
	
	var saveStr []string // holds all 404ed channel names and urls
	
	fmt.Printf("Getting videos not older than %d hours.\n", int64(hrs))
	
	var wg sync.WaitGroup
	
	for i := 0; i < len(fetchedDataItems); i++ {
		
		wg.Add(1)
		
		go func(fdiPtr *FetchedDataItem) {
			
			fetch(int64(hrs), fdiPtr, ignore404)
			
			// Decrement the counter when the go routine completes
			defer wg.Done()
			
		}(&fetchedDataItems[i])
		
	}
	// Wait for all the checkWebsite calls to finish
	wg.Wait()
	
	var returnedUrlIndx int = 0
	for _, di := range fetchedDataItems {
		
		if di.titleArray != nil && di.publishedArray != nil && di.publishedLinks != nil && di.hrsPublished != nil {
			fmt.Printf("Videos found for %s (xmlUrl:%s):\n", di.channelName, di.xmlUrl)
			for i := 0; i < len(di.titleArray); i++ {// or di.numOfUrls
				fmt.Println("("+strconv.Itoa(returnedUrlIndx)+")", di.titleArray[i], di.publishedArray[i], "\tURL:", di.publishedLinks[i], "\n")
				
				returnedUrlIndx = returnedUrlIndx+1
			}
		} else {
			fmt.Println(di.channelName, "(xmlUrl:"+di.xmlUrl+") returned nothing\n")
		}
		
		if len(di.publishedLinks) == 1 {
			if strings.HasPrefix(di.publishedLinks[0], "https://www.google.com/search?q=404")  {
				saveStr = append(saveStr, di.channelName +"(at index "+strconv.Itoa(returnedUrlIndx-1)+") 404ed  (url in file:"+di.xmlUrl+")\n")
			}
		}

		for _, s := range di.publishedLinks {
			lstOfUrls = append(lstOfUrls, s)
		}
		
	}
	
	
	if ignore404 == false {
		fmt.Println("Missing channels:")
		for _, erroredLink := range saveStr {
			fmt.Println(erroredLink)
		}
		var waitStr string
		fmt.Printf("Press any key to contine...")
		fmt.Scan(&waitStr)
	}
	
	var indexToOpen int = 65535-1
	if openall {
		indexToOpen = -1 // open all urls from lstOfUrls
	} else {
		if len(lstOfUrls) == 0 { // nothing returned
			indexToOpen = -1
		} else {
			fmt.Print("\nEnter (index) number to open in browser or negative number to open all urls:")
			fmt.Scan(&indexToOpen) // get indexToOpen number
		}
	}
	
	if indexToOpen < 0 {
		// open all links in lstOfUrls
		var stopAfterThisMany int = 20
		for i, url := range lstOfUrls {
			if i % stopAfterThisMany == 0 && i != 0 {
				var exitProgram string = "\n"
				fmt.Printf("\nId of last shown url is (%d), type e or q to end program or any other key to show next %d links:", i-1, stopAfterThisMany)
				fmt.Scanln(&exitProgram)
				if exitProgram == "e" || exitProgram == "q" || exitProgram == "E" || exitProgram == "Q" {
					os.Exit(0)
				}
			}
			helpers.Openbrowser(url)
		}
    } else if indexToOpen < len(lstOfUrls) {
		// open single url link
    	fmt.Println(lstOfUrls[indexToOpen])
    	helpers.Openbrowser(lstOfUrls[indexToOpen])
    } else {
		fmt.Printf("Number entered too large. Only numbers from 0 to %d are valid.", len(lstOfUrls)-1)
	}
	
	
	//if revertDate { return } // just in case as this is redundant because after reverting date parseArgs will call os.Exit() and never communicate with server or reach here
	
	
	// if all returned without internet error
	
	fpUnixTimeFile, errWithOpeningUnixTimeFile := os.OpenFile(fileWithUnixTimeName, os.O_WRONLY, 0644)
	if errWithOpeningUnixTimeFile != nil {
		_, errTryingToCreateUnixTimeFile := os.Create(fileWithUnixTimeName)
		if errTryingToCreateUnixTimeFile != nil {
			fmt.Println(errTryingToCreateUnixTimeFile.Error())
		}
		panic(errWithOpeningUnixTimeFile)
	}
	defer fpUnixTimeFile.Close()
	
	timeNowNano := time.Now().UnixNano()
	
	lastGetTime := strconv.FormatInt(timeNowNano, 10)
	
	// overwrite fileWithUnixTimeName with new UnixTime
	fpUnixTimeFile.WriteString(lastGetTime)
	
	
	// make sure to backup times just in case
	
	// add new request GET time
	
	var num_of_tries_to_make_backup_folder_append_file int = 0
	var fpUnixTimeFileAppend *os.File
	for num_of_tries_to_make_backup_folder_append_file < 2 {// try only once to create new file
		var errWithOpeningUnixTimeAppendFile error
		fpUnixTimeFileAppend, errWithOpeningUnixTimeAppendFile = os.OpenFile(fileWithUnixTimeAppendName, os.O_APPEND, 0644)
		if errWithOpeningUnixTimeAppendFile != nil {
			absPath, _ := filepath.Abs("../"+THIS_FOLDER_NAME+"/"+fileWithUnixTimeAppendName)
			_, errTryingToCreateUnixTimeAppendFile := os.Create(absPath)
			if errTryingToCreateUnixTimeAppendFile != nil {
				fmt.Println(errTryingToCreateUnixTimeAppendFile.Error())
			}
		} else {
			break
		}
		num_of_tries_to_make_backup_folder_append_file = num_of_tries_to_make_backup_folder_append_file+1
	}
	
	_, errWs := fpUnixTimeFileAppend.WriteString("\n"+strconv.FormatInt(timeNowNano, 10))
	if errWs != nil {
		fmt.Println(errWs)
	}
	
	defer fpUnixTimeFileAppend.Close()

}

type FetchedDataItem struct {
    channelName string
    numOfUrls int
	xmlUrl string
	hrsPublished []int
	publishedLinks []string
	titleArray []string
	publishedArray []string
}

// modifies FetchedDataItem
func fetch(hrs int64, di *FetchedDataItem, ignore404 bool)  {
	var url string = (*di).xmlUrl
	var channelName string = (*di).channelName

	var titleArray, publishedArray, hrefArray []string
	var hrsAgoArray []int
	titleArray, publishedArray, hrefArray, hrsAgoArray = getVideosNotOlderThan(hrs, url, ignore404)
	
	if len(titleArray)==1 && titleArray[0] == "404" { titleArray[0] = channelName+" 404" }
	
	(*di).hrsPublished = hrsAgoArray
	(*di).publishedLinks = hrefArray
	
	(*di).titleArray = titleArray
	(*di).publishedArray = publishedArray
	
		
	(*di).numOfUrls = len((*di).publishedLinks)

	fmt.Println("fetched", len(titleArray), "links from", (*di).channelName)
	//fmt.Printf("%+v\n", *di)
}

// returns titleArray, publishedArray, hrefArray, publishedHrsAgoArray
func getVideosNotOlderThan(hours int64, url string, ignore404 bool) ([]string, []string, []string, []int) {

	// populate arrays with Entry-s which are less than `hours` ago from now
	var titleArray, publishedArray, hrefArray []string
	var publishedHrsAgoArray []int
	
	// get xmlData for url
	_, err, xmlData := communicate(url)
	if err != nil {
		fmt.Println("communicate(url) failed on url:", url)
		fmt.Println(err.Error())
		if ignore404 == false {
			titleArray = append(titleArray, "404")
			publishedArray = append(publishedArray, "")
			hrefArray = append(hrefArray, "https://www.google.com/search?q=404+" + url)
			publishedHrsAgoArray = append(publishedHrsAgoArray, 0)
		}
		return titleArray, publishedArray, hrefArray, publishedHrsAgoArray
		//return nil, nil, nil, nil
	}
	//fmt.Printf("%+v\n", xmlData)
	fmt.Println("\n")

	// going through every xmlData Entry(ie. video published)
	for _, ent := range xmlData.Entry {
		//fmt.Printf("%+q %+q %+q\n", ent.Title, ent.Published, ent.Link.Href)

		// translate Entry.Published which is in `layout` format into `type Time stuct`
		layout := "2006-01-02T15:04:05-07:00"
		input := ent.Published
		
		t, err := time.Parse(layout, input)
		
		if err != nil {
			fmt.Println("Error Entry.Published format not same as `layout` format:"+layout+"\n", err.Error())
		} else {
			
			// how long ago was it
			passedNanosec := time.Now().UnixNano() - t.UnixNano()
			passedMilisec := passedNanosec / (int64(time.Millisecond)/int64(time.Nanosecond))
			passedSeconds := passedMilisec / (int64(time.Second)/int64(time.Millisecond))
			passedMin := passedSeconds / 60
			passedHrs := passedMin / 60

			if passedHrs <= hours {
				numerator, denominator := passedHrs, int64(24)
				quotient, remainder := numerator/denominator, numerator%denominator
				fmt.Printf("found one published %ddays, %dhrs ago", quotient, remainder)

				// append to arrays items to return
				titleArray = append(titleArray, ent.Title)
				publishedArray = append(publishedArray, "\npublished "+strconv.FormatInt(quotient, 10)+"days, "+strconv.FormatInt(remainder, 10)+"hrs ago")
				publishedHrsAgoArray = append(publishedHrsAgoArray, int(passedHrs))
				hrefArray = append(hrefArray, ent.Link.Href)
			}

		}

	}

	return titleArray, publishedArray, hrefArray, publishedHrsAgoArray

}

func parseArgs(fileWithUnixTimeName string, fi string, fileWithUnixTimeAppendName string) (string, string, int, bool, bool) {
	
    const PARAM_OPML_DEFAULT_VALUE string = ""
	var opmlFileName string = PARAM_OPML_DEFAULT_VALUE
	flag.StringVar(&opmlFileName, "opml", PARAM_OPML_DEFAULT_VALUE, "add -opml=FILE_NAME.opml to get urls from opml instead of from "+fi)
	
	const weekOld int = 24*7
    const PARAM_HRS_DEFAULT_VALUE int = weekOld
	var hrs int = PARAM_HRS_DEFAULT_VALUE
	flag.IntVar(&hrs, "hrs", PARAM_HRS_DEFAULT_VALUE, "add -hrs=HOURS_NUMBER to get urls not older than that number of hours.\nIf HOURS_NUMBER equals 0 then UnixTime time from file "+fileWithUnixTimeName+" will be used as not older than time and if all are fetched successfuly UnixTime time now will overwrite old time in "+fileWithUnixTimeName)
	
    const PARAM_IGNORE404_DEFAULT_VALUE bool = false
	var ignore404 bool = PARAM_IGNORE404_DEFAULT_VALUE
	flag.BoolVar(&ignore404, "ignore404", PARAM_IGNORE404_DEFAULT_VALUE, "add -ignore404=true to ignore urls from opml which are can not be reached")
	
    const PARAM_ADD_CHANNELS_CHANNELID_TO_OPML_DEFAULT_VALUE string = ""
	var addRss string = PARAM_ADD_CHANNELS_CHANNELID_TO_OPML_DEFAULT_VALUE
	flag.StringVar(&addRss, "addRss", PARAM_ADD_CHANNELS_CHANNELID_TO_OPML_DEFAULT_VALUE, "add -addRss=URL to get channelId from URL and if laso -opml was given then channelId will be saved to that opml file")
	
    const PARAM_REVERT_DATE_DEFAULT_VALUE bool = false
	var revertDate bool = PARAM_REVERT_DATE_DEFAULT_VALUE
	flag.BoolVar(&revertDate, "revertDate", PARAM_REVERT_DATE_DEFAULT_VALUE, "add -revertDate=true to revert to date previous of last request")
	
    const PARAM_OPENALL_DEFAULT_VALUE bool = false
	var openall bool = PARAM_OPENALL_DEFAULT_VALUE
	flag.BoolVar(&openall, "openall", PARAM_OPENALL_DEFAULT_VALUE, "add -openall=true to get urls from opml instead of from "+fi)
	
	flag.Parse()
	
	
	if addRss != "" {
	
		var addXmlLinkToRss string = ""// if it remains empty nothing will be added
		var urlToGetRss string = addRss// ex. "https://www.youtube.com/@pewdiepie"
		
		_, err, pageData := communicateGettingHtmlPage(urlToGetRss)
		if err != nil {
			fmt.Println("communicateGettingHtmlPage("+urlToGetRss+") failed because:", err.Error())
			os.Exit(1)
		}
		if len(strings.Split(pageData, "\"channelId\":\"")) < 2 {
			fmt.Println("got html page from "+urlToGetRss+" but it does not have channelId:")
			os.Exit(1)
		}
		channelId := strings.Split(pageData, "\"channelId\":\"")[1]
		channelId = strings.Split(channelId, "\"")[0]
		if len(channelId) == 24 {
			fmt.Println("rss feed of length 24 found:", channelId, "\n")
			// add channelId to opmlFileName.opml
			addXmlLinkToRss = "https://www.youtube.com/feeds/videos.xml?channel_id="+channelId
			
			if opmlFileName != "" {
				// open .opml file and add new url and name
				addThisNameToOpml := strings.Split(pageData, "\"canonicalBaseUrl\":\"")[1]
				addThisNameToOpml = strings.Split(addThisNameToOpml, "\"")[0]
				addThisNameToOpml = strings.Replace(addThisNameToOpml, "/", "", -1)
				addThisNameToOpml = strings.Replace(addThisNameToOpml, "@", "", -1)
				errorReadingOrSavingToOpml := addToOpml(opmlFileName, addXmlLinkToRss, addThisNameToOpml, urlToGetRss)
				if errorReadingOrSavingToOpml != nil {
					fmt.Println("Failed to save to opml.", errorReadingOrSavingToOpml.Error())
				} else {
					fmt.Println("Successfully added", addThisNameToOpml, "to", opmlFileName, "file.")
				}
			} else {
					fmt.Println("Unable to save to .opml file because no -opml=FILE_NAME.opml provided")
			}
		} else {
			fmt.Println("rss feed not of length 24 (but should be!):", channelId, "skipped adding to opml file")
		}
	
		os.Exit(1)
	}
	
	var lastGetTime string
	var beforeLastGetTime string
	if revertDate {
		//get last line from fileWithUnixTimeAppendName and make that lastGetTime
		fpUnixTimeBackFile, errWithOpeningUnixTimeBackFile := os.Open(fileWithUnixTimeAppendName)
		defer fpUnixTimeBackFile.Close()
		if errWithOpeningUnixTimeBackFile == nil {
			// get line before last from file fileWithUnixTimeAppendName
			bReader := bufio.NewReader(fpUnixTimeBackFile)
			line, e := helpers.Readln(bReader)
			for e == nil {
				beforeLastGetTime = string(lastGetTime)
				lastGetTime = string(line[:])
				line, e = helpers.Readln(bReader)
			}
			lastGetTime = beforeLastGetTime
			
			// find fileWithUnixTimeAppendName length
			fileWithUnixTimesPtr, errWithUnixTimeAppendName := fpUnixTimeBackFile.Stat()
			if errWithUnixTimeAppendName != nil {
				panic(errWithUnixTimeAppendName)
			}
			unixTimeLen := int64(20)// this is size of nanoseconds unix time string; cast to int64 so it can be compared with io.File.Size() which is of type int64
			
			// truncate fileWithUnixTimeAppendName by unixTimeLen
			if unixTimeLen > fileWithUnixTimesPtr.Size() {
				fmt.Printf("\nUnable to revert any further.")
				fpUnixTimeBackFile.Close()
				os.Exit(1)
			}
			errTruncate := os.Truncate(fileWithUnixTimeAppendName, fileWithUnixTimesPtr.Size()-unixTimeLen)
			if errTruncate != nil {
				panic(errTruncate)
			}

			// overwrite fileWithUnixTimeName with new UnixTime
			fpUnixTimeFile, errWithOpeningUnixTimeFile := os.OpenFile(fileWithUnixTimeName, os.O_WRONLY, 0644)
			if errWithOpeningUnixTimeFile != nil {
				var errTryingToCreateUnixTimeFile error
				fpUnixTimeFile, errTryingToCreateUnixTimeFile = os.Create(fileWithUnixTimeName)
				if errTryingToCreateUnixTimeFile != nil {
					fmt.Println(errWithOpeningUnixTimeFile.Error())
					panic(errTryingToCreateUnixTimeFile)
				}
			}
			fpUnixTimeFile.WriteString(lastGetTime)
	
		} else {
			panic(errWithOpeningUnixTimeBackFile)
		}
		
		os.Exit(1)
	}
	
	if hrs == 0 {
		fpUnixTimeFile, errWithOpeningUnixTimeFile := os.Open(fileWithUnixTimeName)
		if errWithOpeningUnixTimeFile != nil {
			_, errTryingToCreateUnixTimeFile := os.Create(fileWithUnixTimeName)
			if errTryingToCreateUnixTimeFile != nil {
				fmt.Println(errTryingToCreateUnixTimeFile.Error())
			} else {
				fmt.Printf("Created new %s now add to it up to last time to check when sending rss request in utc epoch time format", fileWithUnixTimeName)
			}
			panic(errWithOpeningUnixTimeFile)
		}
		scannerUnixTimeFile := bufio.NewScanner(fpUnixTimeFile)
		var strUnixTime string
		for scannerUnixTimeFile.Scan() {
			strUnixTime = scannerUnixTimeFile.Text()
		}
		int64UnixTime, errConvertingUnixTimeToint64 := strconv.ParseInt(strUnixTime, 10, 64)
		defer fpUnixTimeFile.Close()
		if errConvertingUnixTimeToint64 != nil {
			panic(errConvertingUnixTimeToint64)
		}
		nanosecAgo := time.Now().UnixNano() - int64UnixTime
		milisecAgo := nanosecAgo / (int64(time.Millisecond)/int64(time.Nanosecond))
		secondsAgo := milisecAgo / (int64(time.Second)/int64(time.Millisecond))
		minAgo:= secondsAgo / 60
		hrsAgo := minAgo / 60
		hrs = int(hrsAgo)
	}
	if hrs < 0 { hrs *= -1 }
	
	
	return opmlFileName, PARAM_OPML_DEFAULT_VALUE, hrs, ignore404, openall
}


func populate_fetchedDataItems(fetchedDataItems *[]FetchedDataItem, opmlFileName string, PARAM_OPML_DEFAULT_VALUE string, fi string) {
	
	var io_ReaderWithUrls *io.Reader = new(io.Reader)
	if opmlFileName != PARAM_OPML_DEFAULT_VALUE {
		// get channel urls from args opml file
		
		fmt.Printf("Using %s to read urls.\n", opmlFileName)
		
		// open .opml file and read urls
		lstOfUrls_FromOpmlFile, errorReadingOpml := getUrlsFromOpml(opmlFileName)
		if errorReadingOpml == nil {
			// get channel urls from fi text file

			var temp_lstOfURLs []string

			for _, s := range lstOfUrls_FromOpmlFile {
				split := strings.Split(s, "|")
				(*fetchedDataItems) = append((*fetchedDataItems), FetchedDataItem{channelName: split[0], numOfUrls: 0})
				temp_lstOfURLs = append(temp_lstOfURLs, split[1])
			}
			
			(*io_ReaderWithUrls) = strings.NewReader(strings.Join(temp_lstOfURLs, "\n"))
		
		} else {
			fmt.Printf("Error trying to get urls from opml file: %s\n", errorReadingOpml.Error())
			os.Exit(1)
		}
		
	} else {
		// get channel urls from fi text file
		var errWithOpeningFile error
		*io_ReaderWithUrls, errWithOpeningFile = os.Open(fi)
		if errWithOpeningFile != nil {
			fmt.Printf("Error opening file: %v\n", errWithOpeningFile)
			fmt.Printf("must have %s file adjacent with rss links in format 'https://www.youtube.com/feeds/videos.xml?channel_id=XXXXXXXXXXXXXXXXXXXXXXXX'\n", fi)
			os.Exit(1)
		} else {
			linesNum, _ := helpers.LineCounter(*io_ReaderWithUrls)
			linesNum = linesNum+1
			for i := 1; i <= linesNum; i++ {
				(*fetchedDataItems) = append((*fetchedDataItems), FetchedDataItem{channelName: "channel_on_line_"+strconv.Itoa(i), numOfUrls: 0})
			}
			// assign again because helpers.LineCounter doesnt rewind io_ReaderWithUrls passed to it so it is not reusable
			*io_ReaderWithUrls, _ = os.Open(fi)
			fmt.Printf("Reed urls from file: %s\n", fi)
		}
		
	}
	// populate xmlUrl fields of fetchedDataItems array using io_ReaderWithUrls
	r := bufio.NewReader(*io_ReaderWithUrls)
	if fetchedDataItems == nil {
		panic(fmt.Errorf("Error: no lines in file"))
	}
	
	var lastError error
	
	var i int = 0
	
	// get line from file
	url, e := helpers.Readln(r)
	for e == nil {
		
		(*fetchedDataItems)[i].xmlUrl = url
		
		// get line from file
		url, e = helpers.Readln(r)
		
		i = i+1
		
		lastError = e
	}
	
	if lastError != io.EOF { fmt.Print("Last error from urls file: ", lastError) }
	
}
	
// returns list of strings where each string is textCaption|url (ie. youtube channel name|https://www.youtube.com/feeds/videos.xml?channel_id=XXXXXXXXXXXXXXXXXXXXXXXX)
func getUrlsFromOpml(fileName string) ([]string, error) {
	
	fileContent, err := ioutil.ReadFile(fileName)

    if err != nil {
		//return nil, errors.New("Error with Unmarshal to Feed =" + err.Error())
		return nil, err
	}

    //var err error
	var xmlData xmlFormats.OPML
	
	err = xml.Unmarshal(fileContent, &xmlData)
	if err != nil {
		//return xmlData, errors.New("Error with Unmarshal to Feed =" + err.Error())
		return nil, err
	}
	
	var lstOfUrls []string
	var url string
	for _, outline := range xmlData.Body.Outlines {
		
		for _, o := range outline.Outlines {
			url = o.Text+"|"+o.XMLURL// possible to use o.Title instead of o.Text
			lstOfUrls = append(lstOfUrls, url)
		}
		
	}
	
	return lstOfUrls, nil
	
}


func addToOpml(opmlFileName string, addXmlLinkToRss string, addNameToOpml string, urlToSaveToOpml string) error {
	
	fileContent, err := ioutil.ReadFile(opmlFileName)

    if err != nil {
		return err
	}

    //var err error
	var xmlData xmlFormats.OPML
	
	err = xml.Unmarshal(fileContent, &xmlData)
	if err != nil {
		return err
	}
	
	var outlineWithNewUrl xmlFormats.Outline = xmlFormats.Outline{ Text:addNameToOpml, Title:addNameToOpml, Type:"rss", XMLURL:addXmlLinkToRss, HTMLURL:urlToSaveToOpml }
	// change xmlData.Body.Outlines by chaning last Outline in its Outlines array so that its Outlines has outlineWithNewUrl added to it
	xmlData.Body.Outlines[len(xmlData.Body.Outlines)-1].Outlines = append(xmlData.Body.Outlines[len(xmlData.Body.Outlines)-1].Outlines, outlineWithNewUrl)
	
	output, err := xml.MarshalIndent(xmlData, "", "  ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return err
	}
	//fmt.Printf("MarshalIndent output: %s\n", output) // debug

	err = ioutil.WriteFile(opmlFileName, output, 0644)
	if err != nil { 
		fmt.Printf("Error writting to file %s: %s\n", opmlFileName, err.Error())
		return err
	}

	return nil
}


// returns in string and Feed formats the response when sending GET request defined to given url
func communicate(url string) (string, error, xmlFormats.Feed) {

	var myClient http.Client
	var req *http.Request
	myClient, req = lib.GetHeader(url)

	resp, err := myClient.Do(req)// same as "resp, err := myClient.Get(url)" but with header parameter
	
	
	var xmlData xmlFormats.Feed
	
	if err == nil {
		return getXML_Feed_FromResponseBody(resp, xmlData)
	} else {
		return "Error", errors.New("Error with request:" + err.Error()), xmlData
	}
	
}
// returns in string and Feed formats the response when sending GET request defined to given url
func communicateGettingHtmlPage(url string) (string, error, string) {

	var myClient http.Client
	var req *http.Request
	myClient, req = lib.GetHeader(url)

	resp, err := myClient.Do(req)// same as "resp, err := myClient.Get(url)" but with header parameter
	// resp type is *http.Response
	
	if err != nil {
		return "Error", errors.New("Error with request:" + err.Error()), ""
	} else {
		body, errIoutilReadAll := ioutil.ReadAll(resp.Body)
		if errIoutilReadAll != nil {
			return "Error", errors.New("Error with io.util.ReadAll:" + errIoutilReadAll.Error()), ""
		}
		return "", nil, string(body[:])
	}
	
}

// populate xmlData with result from resp
// returns: Error in string format or other error as string, errors instance, xmlFormats.Feed
func getXML_Feed_FromResponseBody(resp *http.Response, xmlData xmlFormats.Feed) (string, error, xmlFormats.Feed) {

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "Error", errors.New("Error with ioutil translation" + err.Error()), xmlData
	}
	if len(string(body)) > 1000000 {
		return "Error", errors.New("Error returned size >1000000"), xmlData
	}

	if resp.StatusCode != 200 {
		fmt.Println("%s", string(body))
		return "Fail"+strconv.Itoa(resp.StatusCode), errors.New("Fail because code("+strconv.Itoa(resp.StatusCode)+")"), xmlData
	}
	
	err = xml.Unmarshal(body, &xmlData)
	if err != nil {
		return "Error", errors.New("Error with Unmarshal on Feed:" + err.Error()), xmlData
	}

	if resp.StatusCode == 200 {
		return string(body[:]), nil, xmlData
	}

	return "communicate response:" + string(body), errors.New("Response code:" + strconv.Itoa(resp.StatusCode)), xmlData

}
