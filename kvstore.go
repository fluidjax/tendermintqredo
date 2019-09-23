package tendermintqredo

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
	cmn "github.com/tendermint/tendermint/tmlibs/common"
)

var (
	stateKey        = []byte("stateKey")
	kvPairPrefixKey = []byte("kvPairKey:")

	ProtocolVersion version.Protocol = 0x1
)

type State struct {
	db      dbm.DB
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}

func loadState(db dbm.DB) State {
	stateBytes := db.Get(stateKey)
	var state State
	if len(stateBytes) != 0 {
		err := json.Unmarshal(stateBytes, &state)
		if err != nil {
			panic(err)
		}
	}
	state.db = db
	return state
}

func saveState(state State) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	fmt.Println("stateKey %s", string(stateKey))
	fmt.Println("stateBytes %s", string(stateBytes))
	state.db.Set(stateKey, stateBytes)
}

func prefixKey(key []byte) []byte {
	return append(kvPairPrefixKey, key...)
}

//---------------------------------------------------

var _ types.Application = (*KVStoreApplication)(nil)

type KVStoreApplication struct {
	types.BaseApplication
	state State
}

func NewKVStoreApplication() *KVStoreApplication {
	state := loadState(dbm.NewMemDB())
	return &KVStoreApplication{state: state}
}

func (app *KVStoreApplication) Info(req types.RequestInfo) (resInfo types.ResponseInfo) {
	return types.ResponseInfo{
		Data:       fmt.Sprintf("{\"size\":%v}", app.state.Size),
		Version:    version.ABCIVersion,
		AppVersion: ProtocolVersion.Uint64(),
	}
}

type BlockChainTX struct {
	Processor   string
	SenderID    string
	RecipientID []string
	Payload     []byte
	TXhash      []byte
	Tags        map[string]string
}

// tx is either "key=value" or just arbitrary bytes
func (app *KVStoreApplication) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	//Use JSON format message for now - its quicker and easier to add new fields
	payload := BlockChainTX{}

	err := json.Unmarshal(req.Tx, &payload)

	if err != nil {
		return types.ResponseDeliverTx{}
	}

	//senderID := payload.SenderID
	//recipientID := payload.RecipientID
	TXHash := payload.TXhash

	var atts []cmn.KVPair
	atts = append(atts, cmn.KVPair{Key: []byte("name"), Value: []byte("chris")})
	atts = append(atts, cmn.KVpair{Key: []byte("key"), Value: TXHash})

	events := []types.Event{
		{
			Type:       "tag", // curl "localhost:26657/tx_search?query=\"tag.name='chris'\""
			Attributes: atts,
		},
	}

	// events := []types.Event{
	// 	{
	// 		Type: "tag", // curl "localhost:26657/tx_search?query=\"tag.name='chris'\""
	// 		Attributes: []cmn.KVPair{
	// 			{Key: []byte("name"), Value: []byte("john")},
	// 			{Key: []byte("name"), Value: []byte("matt")},
	// 			{Key: []byte("key"), Value: key},
	// 		},

	// var key, value []byte
	// parts := bytes.Split(req.Tx, []byte("="))
	// if len(parts) == 2 {
	// 	key, value = parts[0], parts[1]
	// } else {
	// 	key, value = req.Tx, req.Tx
	// }

	// app.state.db.Set(prefixKey(key), value)
	// app.state.Size += 1

	// events = []types.Event{
	// 	{
	// 		Type: "tag", // curl "localhost:26657/tx_search?query=\"tag.name='chris'\""
	// 		Attributes: []cmn.KVPair{
	// 			{Key: []byte("name"), Value: []byte("john")},
	// 			{Key: []byte("name"), Value: []byte("matt")},
	// 			{Key: []byte("key"), Value: key},
	// 		},
	// 	},
	// }

	return types.ResponseDeliverTx{Code: code.CodeTypeOK, Events: events}
}

func (app *KVStoreApplication) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}

func (app *KVStoreApplication) Commit() types.ResponseCommit {
	// Using a memdb - just return the big endian size of the db
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height += 1
	saveState(app.state)
	return types.ResponseCommit{Data: appHash}
}

// Returns an associated value or nil if missing.
func (app *KVStoreApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	if reqQuery.Prove {
		value := app.state.db.Get(prefixKey(reqQuery.Data))
		resQuery.Index = -1 // TODO make Proof return index
		resQuery.Key = reqQuery.Data
		resQuery.Value = value
		if value != nil {
			resQuery.Log = "exists"
		} else {
			resQuery.Log = "does not exist"
		}
		return
	} else {
		resQuery.Key = reqQuery.Data
		value := app.state.db.Get(prefixKey(reqQuery.Data))
		resQuery.Value = value
		if value != nil {
			resQuery.Log = "exists"
		} else {
			resQuery.Log = "does not exist"
		}
		return
	}
}
