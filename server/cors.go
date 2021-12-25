package server

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

//  corsData Raw data of CORS config
type CorsData struct {
	AllowedMethods []string `json:"AllowedMethods"`
	AllowedOrigins []string `json:"AllowedOrigins"`
	AllowedHeaders []string `json:"AllowedHeaders"`
}

//  corsData Raw data of CORS config
type Cors struct {
	Data   CorsData
	Loaded bool
}

// NewCors Creates a new Cors object
func NewCors() *Cors {
	c := new(Cors)

	c.loadDefault()

	return c
}

// LoadFromDisc Initializes CORS from config file
func (c *Cors) LoadFromDisc(configFilePath string) error {
	data, errLoad := c.loadJSONDataFromDisc(configFilePath)
	if errLoad != nil {
		return errLoad
	}

	errJson := json.Unmarshal(data, &c.Data)
	if errJson != nil {
		return errJson
	}

	c.Loaded = true

	return nil
}

func (c *Cors) String() string {
	ret := ""

	b, err := json.Marshal(c.Data)
	if err == nil {
		ret = string(b)
	}

	return ret
}

func (c *Cors) GetAllowedOriginsStr() string {
	return strings.Join(c.Data.AllowedOrigins[:], ", ")
}

func (c *Cors) GetAllowedMethodsStr() string {
	return strings.Join(c.Data.AllowedMethods[:], ", ")
}

func (c *Cors) GetAllowedHeadersStr() string {
	return strings.Join(c.Data.AllowedHeaders[:], ", ")
}

func (c *Cors) loadJSONDataFromDisc(configFilePath string) (data []byte, err error) {
	jsonFile, errOpen := os.Open(configFilePath)
	if errOpen != nil {
		err = errOpen
		return
	}
	defer jsonFile.Close()

	data, errRead := ioutil.ReadAll(jsonFile)
	if errRead != nil {
		err = errRead
	}

	return
}

func (c *Cors) loadDefault() {
	c.Data.AllowedMethods = []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"}
	c.Data.AllowedHeaders = []string{"Content-Type"}
	c.Data.AllowedOrigins = []string{"*"}
}
