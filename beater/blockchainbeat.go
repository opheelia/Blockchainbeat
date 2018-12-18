package beater

import (
	"time"
  "fmt"
  "net/http"
  "encoding/json"
	"strconv"
	//"io/ioutil"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/opheelia/blockchainbeat/config"
)

var number_of_blocks = 500

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

		//Get last block number
		last_block := GetLastBlock()
		logp.Info(fmt.Sprintf("Last block is %v, type %T", last_block, last_block))



		SendBlockTransactionsToElastic(last_block, b, bt, counter)

		counter++
	}
}

// Stop stops blockchainbeat.
func (bt *Blockchainbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
func GetLastBlock() (int){
	resp, err := http.Get("https://api.blockcypher.com/v1/eth/main")
	if err != nil {
		logp.Info("ERROR : HTTP REQUEST LAST BLOCK")
	}
	defer resp.Body.Close()
	var resp_body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&resp_body)
	last_block := resp_body["height"].(float64)
	return int(last_block)

}

func GetBlockTransaction(result interface{}, block_number int) {
	resp, err := http.Get("https://api.etherscan.io/api?module=block&action=getblockreward&blockno=" + strconv.Itoa(block_number) + "&apikey=YourApiKeyToken")
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

func SendBlockTransactionsToElastic(last_block int, b *beat.Beat, bt *Blockchainbeat, counter int){
	//Get block transactions up to last block number

	for index:=last_block-number_of_blocks; index<=last_block; index++ {
		logp.Info(fmt.Sprint(index))
		var result map[string]interface{}
		GetBlockTransaction(&result, index)
		logp.Info(fmt.Sprintf("Transaction for block %v", index))
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
	}

}
