package models

import (
	"dgatewayWebBrowser/chainapi"
	"dgatewayWebBrowser/datastruct"
	"dgatewayWebBrowser/dboperation"
	"strconv"
	"time"

	"github.com/astaxie/beego"

	"github.com/gorilla/websocket"
)

var (
	//WsClients :WebSocket的连接客户端map
	WsClients = make(map[*websocket.Conn]bool)

	BroadCastBlockChan = make(chan *datastruct.Block, 50)
	BroadCastTxChan    = make(chan *datastruct.DgatewayTx, 50)
)

func init() {
	go handleBroadCastMessage()

}

func mockaddBlockChan() {
	height := int64(1000)
	for {
		block := &datastruct.Block{
			ID:     "test",
			Height: height,
		}
		BroadCastBlockChan <- block
		height++

		time.Sleep(3 * time.Second)
	}
}

func mockaddTxChan() {
	amount := int64(1000)
	for {
		block := &datastruct.DgatewayTx{
			Id:        999,
			FromChain: "bch",
			Amount:    amount,
		}
		BroadCastTxChan <- block
		amount++
		time.Sleep(5 * time.Second)
	}
}

func handleBroadCastMessage() {
	for {
		select {
		case blockInfo := <-BroadCastBlockChan:
			sendBlockMessage(blockInfo)
		case txInfo := <-BroadCastTxChan:
			sendTxMessage(txInfo)
		}
	}
}

func sendBlockMessage(blockInfo *datastruct.Block) bool {
	nodeNum, _ := chainapi.GetNodeNum()
	txNum := dboperation.GetTxCount("")
	infoResp := &datastruct.WsInfoResp{
		Type:          1,
		HighestBolck:  blockInfo.Height,
		LastBlockTime: blockInfo.CreatedUsed,
		NodeNum:       nodeNum,
		TxNum:         txNum,
	}
	blockResp := &datastruct.WsBlockResp{
		Type:        2,
		Height:      blockInfo.Height,
		ID:          blockInfo.ID,
		Miner:       blockInfo.Miner,
		CreatedUsed: blockInfo.CreatedUsed,
		TxCnt:       blockInfo.TxCnt,
		Time:        blockInfo.Time,
	}
	for client := range WsClients {
		err := client.WriteJSON(infoResp)
		if err != nil {
			beego.Error(err)
			client.Close()
			delete(WsClients, client)
		}
		err1 := client.WriteJSON(blockResp)
		if err1 != nil {
			beego.Error(err)
			client.Close()
			delete(WsClients, client)
		}
	}
	return true
}

func sendTxMessage(txInfo *datastruct.DgatewayTx) bool {
	wsTxResp := &datastruct.WsTxResp{
		Type:       3,
		DgatewayTx: *txInfo,
	}
	for client := range WsClients {
		err := client.WriteJSON(wsTxResp)
		if err != nil {
			beego.Error(err)
			client.Close()
			delete(WsClients, client)
		}
	}
	return true
}

//首页图表信息(15天)
func GetInfoTableData() (*datastruct.InfoTableResp, error) {

	timeNow := time.Now() //获取当前时间
	zeroHour := time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), 0, 0, 0, 0, timeNow.Location())
	//获取下一个零时
	startTime := zeroHour.AddDate(0, 0, -15)
	endTime := zeroHour.AddDate(0, 0, -1)

	data, err := dboperation.GetInfoDataFromMysql(startTime, endTime)
	if err != nil {
		return nil, err
	}
	var timeinfo []time.Time
	timeCount := make(map[time.Time]int64)
	timeAmount := make(map[time.Time]float64)
	for i := -15; i < 0; i++ {
		before := zeroHour.AddDate(0, 0, i)
		timeinfo = append(timeinfo, before)
		timeCount[before] = 0
		timeAmount[before] = 0

	}
	//amount目前的判断逻辑：from_chain 和to_chain，哪个不是eth，就是哪一个。。
	for _, d := range data {
		timeCount[d.Time] += d.Count
		timeAmount[d.Time] += d.CurrencyAmount
	}
	date := []string{}
	count := []int64{}
	amount := []float64{}
	for _, info := range timeinfo {
		month := strconv.Itoa(int(info.Month()))
		day := strconv.Itoa(info.Day())
		infoString := month + "/" + day
		date = append(date, infoString)
		count = append(count, timeCount[info])
		amount = append(amount, timeAmount[info])

	}
	return &datastruct.InfoTableResp{
		Code: 0,
		Msg:  "",
		Data: map[string]interface{}{"time": date, "count": count, "amount": amount},
	}, nil
}

func GetInfoOverview() (*datastruct.InfoOverviewResp, error) {
	nodeNum, _ := chainapi.GetNodeNum()
	txNum := dboperation.GetTxCount("")
	lastBlock, err := dboperation.GetLatestBlock()
	if err != nil {
		return nil, err
	}
	resp := &datastruct.InfoOverviewResp{
		Code: 0,
		Msg:  "",
		Data: datastruct.WsInfoResp{
			Type:          1,
			HighestBolck:  lastBlock.Height,
			LastBlockTime: lastBlock.CreatedUsed,
			NodeNum:       nodeNum,
			TxNum:         txNum,
		},
	}
	return resp, nil
}