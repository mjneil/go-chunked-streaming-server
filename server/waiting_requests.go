package server

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultRequestExpiration time.Duration = 1000 * time.Millisecond
const defaultRequestCleanUpEvery time.Duration = 100 * time.Millisecond

const (
	cancelSignal = 0
	isCancelled  = 1
	dataArrived  = 2
)

// WaitingRequests stores data of blocked requests
type WaitingRequests struct {
	requests     map[string]*WaitingRequestArrayBlock
	requestsLock sync.RWMutex

	cleanUpChannelBidi chan bool
}

type WaitingRequestArrayBlock struct {
	requests []*WaitingRequest
}

// WaitingRequest Definition of blocked request waiting for data
type WaitingRequest struct {
	uidStr       string
	receivedAt   time.Time
	expirationAt time.Time

	channelBidi chan int
}

// NewCors Creates a new Cors object
func NewWaitingRequests() *WaitingRequests {
	brs := WaitingRequests{
		requests:           map[string]*WaitingRequestArrayBlock{},
		cleanUpChannelBidi: make(chan bool),
	}

	go brs.runRequestCleanupEvery(defaultRequestCleanUpEvery)

	return &brs
}

// AddWaitingRequest Adds a new requests to wait for data, and blocks the execution
func (brs *WaitingRequests) AddWaitingRequest(name string, headers http.Header) (found bool, waited time.Duration) {
	found = false
	nowStart := time.Now()
	// This is modified Expires, instead of HTTP-date timestamp uses duration in seconds (Ex: "Expires: in=10")
	expiration := brs.getExpiresInOr(headers.Get("Expires"), defaultRequestExpiration)

	uidStr := uuid.New().String()
	br := WaitingRequest{
		uidStr:       uidStr,
		receivedAt:   nowStart,
		expirationAt: nowStart.Add(expiration),
		channelBidi:  make(chan int),
	}

	brs.requestsLock.Lock()

	//Add waiting request
	reqArrayBlock, exists := brs.requests[name]
	if !exists {
		brs.requests[name] = &WaitingRequestArrayBlock{}
		reqArrayBlock = brs.requests[name]
	}
	reqArrayBlock.requests = append(reqArrayBlock.requests, &br)

	brs.requestsLock.Unlock()

	// Wait on signal
	msg := <-br.channelBidi
	if msg == dataArrived {
		found = true
	}

	// Remove request
	brs.requestsLock.Lock()

	brs.removeRequestByUID(name, uidStr)

	brs.requestsLock.Unlock()

	waited = time.Since(nowStart)

	return
}

// AddBlockedRequest Adds a new blocked requests
func (brs *WaitingRequests) Close() {
	brs.cancelRemoveAllRequests()

	brs.stopCleanUp()
}

func (brs *WaitingRequests) ReceivedDataFor(name string) {
	now := time.Now()

	brs.requestsLock.Lock()
	defer brs.requestsLock.Unlock()

	for nameWaiting, reqArrayBlock := range brs.requests {
		if name == nameWaiting {
			for _, bReq := range reqArrayBlock.requests {
				if now.Before(bReq.expirationAt) {
					brs.responseRequest(bReq)
				}
			}
		}
	}
}

func (brs *WaitingRequests) getExpiresInOr(s string, def time.Duration) time.Duration {
	ret := def
	r := regexp.MustCompile(`in=(?P<in>\d*)`)
	match := r.FindStringSubmatch(s)
	for i, name := range r.SubexpNames() {
		if i > 0 && i <= len(match) {
			if name == "in" {
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

	for name, reqArrayBlock := range brs.requests {
		for i, bReq := range reqArrayBlock.requests {
			// Add expired requests to delete array
			if now.After(bReq.expirationAt) {
				brs.cancelRequest(brs.requests[name].requests[i])
			}
		}
	}
}

func (brs *WaitingRequests) cancelRemoveAllRequests() {
	for _, reqArrayBlock := range brs.requests {
		for _, bReq := range reqArrayBlock.requests {
			brs.cancelRequest(bReq)
		}
	}
}

func (brs *WaitingRequests) removeRequestByUID(name string, uidStr string) {
	reqArrayBlock, exists := brs.requests[name]
	if exists {
		for i, bReq := range reqArrayBlock.requests {
			if bReq.uidStr == uidStr {
				if len(reqArrayBlock.requests) > 1 {
					reqArrayBlock.requests = append(reqArrayBlock.requests[:i], reqArrayBlock.requests[i+1:]...)
				} else {
					reqArrayBlock.requests = []*WaitingRequest{}
				}
			}
		}
		// Remove name entry if no waiting requests
		if len(reqArrayBlock.requests) <= 0 {
			delete(brs.requests, name)
		}
	}
}

func (brs *WaitingRequests) cancelRequest(br *WaitingRequest) {
	br.channelBidi <- cancelSignal
}

func (brs *WaitingRequests) responseRequest(br *WaitingRequest) {
	br.channelBidi <- dataArrived
}
