package server

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const defaultRequestExpiration time.Duration = 5000 * time.Millisecond
const defaultRequestCleanUpEvery time.Duration = 100 * time.Millisecond

// WaitingRequests stores data of blocked requests
type WaitingRequests struct {
	requests     map[string][]*WaitingRequest
	requestsLock sync.RWMutex

	cleanUpChannelBidi chan bool

	// Callbacks
	callbackResponse func(string, http.ResponseWriter, *http.Request, *Cors, *File, string)
	callbackCancel   func(string, http.ResponseWriter)
}

// WaitingRequest Definition of blocked request waiting for data
type WaitingRequest struct {
	receivedAt   time.Time
	expirationAt time.Time

	cors     *Cors
	basePath string
	w        http.ResponseWriter
	r        *http.Request
}

// NewCors Creates a new Cors object
func NewWaitingRequests(callbackResponse func(string, http.ResponseWriter, *http.Request, *Cors, *File, string), callbackCancel func(string, http.ResponseWriter)) *WaitingRequests {
	brs := WaitingRequests{
		requests:           map[string][]*WaitingRequest{},
		cleanUpChannelBidi: make(chan bool),

		callbackResponse: callbackResponse,
		callbackCancel:   callbackCancel,
	}

	go brs.runRequestCleanupEvery(defaultRequestCleanUpEvery)

	return &brs
}

// AddWaitingRequest Adds a new requests to wait for data
func (brs *WaitingRequests) AddWaitingRequest(name string, headers http.Header, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	expiration := brs.getMaxAgeOr(headers.Get("Cache-Control"), defaultRequestExpiration)

	br := WaitingRequest{
		receivedAt:   now,
		expirationAt: now.Add(expiration),
		cors:         cors,
		basePath:     basePath,
		w:            w,
		r:            r,
	}

	brs.requestsLock.Lock()
	defer brs.requestsLock.Unlock()
	//Add waiting request
	brs.requests[name] = append(brs.requests[name], &br)

	// Wait on signal
	// TODO
}

// AddBlockedRequest Adds a new blocked requests
func (brs *WaitingRequests) Close() {
	brs.cancelRemoveAllRequests()

	brs.stopCleanUp()
}

func (brs *WaitingRequests) ReceivedDataFor(name string, f *File) {
	now := time.Now()

	brs.requestsLock.Lock()
	defer brs.requestsLock.Unlock()

	for nameWaiting, reqArray := range brs.requests {
		if name == nameWaiting {
			for _, bReq := range reqArray {
				if now.Before(bReq.expirationAt) {
					brs.responseRequest(name, bReq, f)
				}
			}
		}
	}
}

func (brs *WaitingRequests) getMaxAgeOr(s string, def time.Duration) time.Duration {
	ret := def
	r := regexp.MustCompile(`max-age=(?P<maxage>\d*)`)
	match := r.FindStringSubmatch(s)
	for i, name := range r.SubexpNames() {
		if i > 0 && i <= len(match) {
			if name == "maxage" {
				valInt, err := strconv.ParseInt(match[i], 10, 64)
				if err == nil {
					ret = time.Duration(valInt) * time.Second
					break
				}
			}
		}
	}
	return ret
}

func (brs *WaitingRequests) runRequestCleanupEvery(period time.Duration) {
	timeCh := time.NewTicker(period)
	exit := false

	for !exit {
		select {
		// Wait for the next tick
		case tm := <-timeCh.C:
			brs.expireRequests(tm)

		case <-brs.cleanUpChannelBidi:
			exit = true
		}
	}
	// Indicates finished
	brs.cleanUpChannelBidi <- true
}

func (brs *WaitingRequests) stopCleanUp() {
	// Send finish signal
	cleanUpChannel <- true

	// Wait to finish
	<-cleanUpChannel
}

func (brs *WaitingRequests) expireRequests(now time.Time) {

	brs.requestsLock.Lock()
	defer brs.requestsLock.Unlock()

	// This could be heavily improved, we will be blocking the thread a lot of time here if we have a big number of waiting requests.
	// Create a deletion flag and do it async during idle time would be better

	toDelReqName := []string{}
	for name, reqArray := range brs.requests {
		toDelReqIndex := []int{}
		for i, bReq := range reqArray {
			// Add expired requests to delete array
			if now.After(bReq.expirationAt) {
				toDelReqIndex = append(toDelReqIndex, i)
			}
		}
		// Execute deletion
		for i := len(toDelReqIndex) - 1; i >= 0; i-- {
			index := toDelReqIndex[i]
			brs.cancelRequest(name, brs.requests[name][index])
			reqArray = append(reqArray[:index], reqArray[index+1:]...)
		}
		// Add URL that has NO waiting requests for removal
		if len(reqArray) <= 0 {
			toDelReqName = append(toDelReqName, name)
		}
	}
	// Execute removal of waiting requests URLs
	for _, name := range toDelReqName {
		delete(brs.requests, name)
	}
}

func (brs *WaitingRequests) cancelRemoveAllRequests() {
	for name, reqArray := range brs.requests {
		for _, bReq := range reqArray {
			brs.cancelRequest(name, bReq)
		}
	}
	// Remove all requests
	for name := range brs.requests {
		brs.requests[name] = nil
	}
	// Clear map
	brs.requests = map[string][]*WaitingRequest{}
}

func (brs *WaitingRequests) cancelRequest(name string, br *WaitingRequest) {
	if brs.callbackCancel != nil {
		brs.callbackCancel(name, br.w)
	}
}

func (brs *WaitingRequests) responseRequest(name string, br *WaitingRequest, f *File) {
	if brs.callbackResponse != nil {
		brs.callbackResponse(name, br.w, br.r, br.cors, f, br.basePath)
	}
}
