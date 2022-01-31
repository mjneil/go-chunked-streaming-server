package server

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultMaxAgeForTranslated time.Duration = 1000 * time.Millisecond

// UrlTranslator keeps the URL entries
type UrlTranslator struct {
	streamEntries map[string]map[int64]urlTranslatorStreamUnit

	translatorLock sync.RWMutex
}

// urlTranslatorStreamUnit One of this stream entry
type urlTranslatorStreamUnit struct {
	receivedAt time.Time
}

// NewCors Creates a new Cors object
func NewUrlTranslator() *UrlTranslator {
	urlT := UrlTranslator{
		streamEntries: map[string]map[int64]urlTranslatorStreamUnit{},
	}
	return &urlT
}

func (urlt *UrlTranslator) GetTranslatedMaxAge() time.Duration {
	return defaultMaxAgeForTranslated
}

func (urlt *UrlTranslator) AddNewEntry(uriStr string, receivedAt time.Time) {

	baseUri, uriId := urlt.getBaseIdFromUrl(uriStr)
	if baseUri == "" || uriId < 0 {
		return
	}

	urlt.translatorLock.Lock()

	// Check if base URL exists and adds if not
	stream, exists := urlt.streamEntries[baseUri]
	if !exists {
		// Create stream if not exists
		urlt.streamEntries[baseUri] = map[int64]urlTranslatorStreamUnit{}
		stream = urlt.streamEntries[baseUri]
	}
	// Add entry
	entry := urlTranslatorStreamUnit{
		receivedAt: receivedAt,
	}
	stream[uriId] = entry

	urlt.translatorLock.Unlock()
}

func (urlt *UrlTranslator) RemoveEntry(uriStr string) {

	baseUri, uriId := urlt.getBaseIdFromUrl(uriStr)
	if baseUri == "" || uriId < 0 {
		return
	}

	urlt.translatorLock.Lock()

	// Check if base URL exists and adds if not
	stream, existsStream := urlt.streamEntries[baseUri]
	if existsStream {
		_, existsEntry := stream[uriId]
		if existsEntry {
			delete(stream, uriId)
		}
		// Check if stream is empty and delete it
		if len(stream) <= 0 {
			delete(urlt.streamEntries, baseUri)
		}
	}

	urlt.translatorLock.Unlock()
}

func (urlt *UrlTranslator) GetTranslated(uriStr string) (retUriStr string, isTranslated bool) {
	retUriStr = uriStr
	isTranslated = false

	baseStr, cmdStr := urlt.parseBaseUri(uriStr)
	if strings.HasPrefix(cmdStr, "EDGE") {
		isTranslated = true
		urlt.translatorLock.Lock()

		retUriStr = urlt.getLatest(baseStr) // Inside will NOT Lock

		urlt.translatorLock.Unlock()
	} else if strings.HasPrefix(cmdStr, "OLD_S") {
		secsStrArr := strings.Split(cmdStr, "=")
		if len(secsStrArr) > 1 {
			secsStr := secsStrArr[1]
			secs, err := strconv.Atoi(secsStr)
			if err == nil {
				isTranslated = true
				urlt.translatorLock.Lock()

				retUriStr = urlt.getOld(baseStr, secs) // Inside will NOT Lock

				urlt.translatorLock.Unlock()
			}
		}
	}
	return
}

func (urlt *UrlTranslator) getLatest(baseUri string) (retBasePath string) {
	retBasePath = ""
	var retSeqId int64 = -1

	// Better used a sorted map, but since this call would be infrequent is OK
	// consume some resources

	stream, existsStream := urlt.streamEntries[baseUri]
	if existsStream {
		// Find lastest
		for key := range stream {
			if key > retSeqId {
				retSeqId = key
			}
		}
	}

	if retSeqId >= 0 {
		retBasePath = baseUri + "/" + strconv.FormatInt(retSeqId, 10)
	}

	return
}

func (urlt *UrlTranslator) getOld(baseUri string, secOld int) (retBasePath string) {
	retBasePath = ""
	timeStart := time.Now().Add(time.Duration(-1*secOld) * time.Second)
	var retSeqId int64 = -1

	// Better used a sorted map, but since this call would be infrequent is OK
	// consume some resources

	stream, existsStream := urlt.streamEntries[baseUri]
	if existsStream {
		// Find latest
		for key, unitToCheck := range stream {
			if unitToCheck.receivedAt.After(timeStart) {
				if retSeqId < 0 || stream[retSeqId].receivedAt.After(unitToCheck.receivedAt) {
					retSeqId = key
				}
			}
		}
	}

	if retSeqId >= 0 {
		retBasePath = baseUri + "/" + strconv.FormatInt(retSeqId, 10)
	}

	return
}

func (urlt *UrlTranslator) parseBaseUri(uriStr string) (base string, last string) {
	base = ""
	last = ""

	s := strings.Split(uriStr, "/")
	sLen := len(s)
	if sLen > 1 {
		last = s[sLen-1]
		s = s[:len(s)-1]
		base = strings.Join(s, "/")
	}
	return
}

func (urlt *UrlTranslator) getBaseIdFromUrl(urlStr string) (base string, id int64) {
	base = ""
	id = -1

	baseTmp, lastTmp := urlt.parseBaseUri(urlStr)
	if baseTmp != "" && lastTmp != "" {
		base = baseTmp
		idTest, err := strconv.ParseInt(lastTmp, 10, 64)
		if err == nil {
			id = idTest
		}
	}
	return
}
