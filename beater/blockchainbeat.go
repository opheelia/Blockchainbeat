package beater

import (
	"time"
  "fmt"
  "net/http"
  "encoding/json"
	//"io/ioutil"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/opheelia/blockchainbeat/config"
)

// Blockchainbeat configuration.
type Blockchainbeat struct {
	done   chan struct{}
	config config.Config
	client beat.Client
}

// New creates an instance of blockchainbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Blockchainbeat{
		done:   make(chan struct{}),
		config: c,
	}
	return bt, nil
}

// Run starts blockchainbeat.
func (bt *Blockchainbeat) Run(b *beat.Beat) error {
	logp.Info("blockchainbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	counter := 1
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		last_block := GetLastBlock()
		logp.Info(fmt.Sprint(last_block))
		//last_block = last_block.(string)
		/*if err != nil {
			logp.Info("")
		}*/
		//logp.Info("Last block of the chain is " + last_block_str)
		var result map[string]interface{}
		GetRequest(&result)
		/*if err != nil {
			logp.Info("GET request error")
		}*/
		logp.Info(fmt.Sprint(result))

		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    b.Info.Name,
				"counter": counter,
				"message": result,
			},
		}
		bt.client.Publish(event)
		logp.Info("Event sent")
		counter++
	}
}

// Stop stops blockchainbeat.
func (bt *Blockchainbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
func GetLastBlock() (float64){
	resp, err := http.Get("https://api.blockcypher.com/v1/eth/main")
	if err != nil {
		logp.Info("ERROR : HTTP REQUEST LAST BLOCK")
	}
	defer resp.Body.Close()
	var resp_body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&resp_body)
	last_block := resp_body["height"].(float64)
	return last_block

}

func GetRequest(result interface{}) {
	resp, err := http.Get("https://api.etherscan.io/api?module=block&action=getblockreward&blockno=2165403&apikey=YourApiKeyToken")
	if err != nil {
		logp.Info("")
	}
	defer resp.Body.Close()
	/*body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logp.Info("")
	}*/
  json.NewDecoder(resp.Body).Decode(result)
}
