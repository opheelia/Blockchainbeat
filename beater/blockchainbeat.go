package beater

import (
	"time"
  "fmt"
  "net/http"
  "encoding/json"
	"strconv"
	"os"
	//"io/ioutil"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/opheelia/blockchainbeat/config"
)

var number_of_blocks = 10000

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
		last_existing_block := GetLastBlock()
		logp.Info(fmt.Sprintf("Last block is %v, type %T", last_existing_block, last_existing_block))

		// Get last indexed block (stored into last_indexed_block.txt)
		// if file does not exit, create it (f is file)
		if _, err := os.Stat("beater/last_indexed_block.txt"); os.IsNotExist(err) {
			f, err := os.Create("beater/last_indexed_block.txt")
			d := []byte("06925596")
			_, err = f.Write(d)
			if err != nil {
				return err
			}
			f.Sync()
			f.Close()
		}
		f, err := os.OpenFile("beater/last_indexed_block.txt", os.O_RDWR, 0666)
		if err != nil {
			return err
		}

		b1 := make([]byte, 8)
		_, err = f.Read(b1)
		if err != nil {
			return err
		}

		last_indexed_block, err := strconv.Atoi(string(b1))
		if err != nil {
			return err
		}
		logp.Info("Last read line %d for file\n ", last_indexed_block)

		// Index blocks
		if last_existing_block > last_indexed_block {
			// Index new blocks
			SendBlockTransactionsToElastic(last_indexed_block, last_existing_block, b, bt, counter)

			// Write last existing block into the file
			write_last_block := fmt.Sprintf("%08d\n", last_existing_block)
			_, err = f.Seek(0, 0)
			if err != nil {
				return err
				}
			d := []byte(write_last_block)
			_, err = f.Write(d)
			if err != nil {
				return err
				}

			logp.Info("Write last read line : %s", write_last_block)
		} else {
			logp.Info("All the blocks are already indexed")
		}

		// Close file
		f.Sync()
		f.Close()

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

func SendBlockTransactionsToElastic(last_indexed_block int, last_block int, b *beat.Beat, bt *Blockchainbeat, counter int){
	//Get block transactions up to last block number

	for index:=last_indexed_block+1; index<=last_block; index++ {
		logp.Info(fmt.Sprint(index))
		var result map[string]interface{}
		GetBlockTransaction(&result, index)
		logp.Info(fmt.Sprintf("Transaction for block %v", index))
		logp.Info(fmt.Sprint(result))

		// Parse timeStamp field to get a date
		data := result["result"].(map[string]interface{})
		block_date := getBlockTimeFormat(data["timeStamp"].(string))
		logp.Info(fmt.Sprintf("New time is %v, %T", block_date, block_date))
		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    b.Info.Name,
				"counter": counter,
				"message": result["message"],
				"blockNumber": data["blockNumber"],
				"blockTime": block_date,
				"blockMiner": data["blockMiner"],
			},
		}
		bt.client.Publish(event)
		logp.Info("Event sent")
	}
}

func getBlockTimeFormat(timestamp string) (time.Time) {
	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		logp.Info(fmt.Sprint("Timestamp cannot be parsed"))
	}
	//logp.Info(fmt.Sprintf("Before parsing %v is a %T", i, i))
	tm := time.Unix(i,0)
	return tm
}
