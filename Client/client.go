package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net"
	"sync"
	"time"

	helper "../helper"
	rsa "../rsa"
)

var conn net.Conn
var myPort string
var ipInput string
var portInput string
var addrToDoTransaction = make([]string, 0)
var peerList = make([]string, 0)
var shouldConnectToPeer bool = false
var localLedger *Ledger = MakeLedger()

var transactionList = make([]int, 0)

var to string
var from string
var amount int

//rsa
var pk *big.Int = big.NewInt(3) // public key
var sk *big.Int = big.NewInt(3) // secret key
var signature = big.NewInt(3)   // signature, pk for hash , messageHash

//mutex to handle concurrency
var mutex = sync.Mutex{}

var isSequencer bool = false // The initial user should be the designated sequencer
var sequencerIP string
var transactionCounter int = 0
var sequencerTransactionList = make([]*Transaction, 0)
var isFirstBlock bool = true
var lastBlockNumber int = 0
var processedBlocks = make([]int, 0)

//sequencer rsa
var SequencerSignature = big.NewInt(3)
var SequencerPk = big.NewInt(3)
var SequencerSk = big.NewInt(3)

var blockChain = make([]*Block, 0)
var transactionsCompleted = make([]int, 0)

//This is a message struct, used for encoding and decoding messages
type Communication struct {
	Peerlist    []string
	MsgPort     string
	MsgID       int
	From        string //transaction
	To          string //transaction
	Amount      int    //transaction
	Ledger      *Ledger
	Signature   string
	Pk          string
	Block       *Block
	Transaction *Transaction
	Blockchain  []*Block
}

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

type Transaction struct {
	To            string
	From          string
	Amount        int
	TransactionId int
}

type Block struct {
	Number        int
	Transactions  *Transaction
	PreviousHash  string
	Seed          int
	InitialLedger *Ledger
}

func main() {
	pk, sk = rsa.KeyGen() // generate keys for each peer
	fmt.Println("My public key is n:", rsa.Hash(pk))
	signature = rsa.GenerateSignature(rsa.Hash(pk), sk, pk) // sign own public key
	askForIPAndPort()
	initialize()
}

func askForIPAndPort() {
	// fmt.Print("Please enter the IP address that you want to connect to: ")
	// fmt.Scan(&ipInput)
	ipInput = "localhost"
	fmt.Print("Please enter the port: ")
	fmt.Scan(&portInput)
}

func createGenesisBlock() {
	fmt.Println("Creating genesis block")
	ledger := MakeLedger()

	block := &Block{}
	block.Number = 1
	block.Seed = 42
	block.InitialLedger = ledger
	blockChain = append(blockChain, block)
	//fmt.Println("Genesis block", block.)
	fmt.Println("Blockchain", blockChain)
}

func sendLotteryRequest() {
	com := &Communication{}
	com.Pk = rsa.Hash(pk).String()
	com.MsgID = 777
	broadcast(com)
}

func initialize() {
	var ipAndPort = ipInput + ":" + portInput
	conn, _ = net.Dial("tcp", ipAndPort)

	if conn == nil {
		fmt.Println("No client with that IP/port exsists in the network")
		fmt.Println("Starting new network")
		//fmt.Println("Sequencer public key: ", SequencerPk)
		createGenesisBlock()
		addPkToGenesisBlock(rsa.Hash(pk).String()) // add myself to genesis block
		startServer(false)
	} else {
		fmt.Println("Found peer")
		fmt.Println("Starting server")
		startServer(true)
	}
}

//Starting the server side
func startServer(shouldConnect bool) {
	ln, _ := net.Listen("tcp", ":") // Listen on random port
	myPort = ln.Addr().String()
	if shouldConnect {
		go connectToPeer(conn)
	} else {
		peerList = append(peerList, ln.Addr().String())
	}

	defer ln.Close()
	fmt.Println("Listening for connection on " + ln.Addr().String())

	for {
		conn, _ := ln.Accept()
		fmt.Println("Got a connection...")
		go handleConnection(conn)
	}
}

// Connecting to a peer, and are then setting up a thread for writing to this peer,
//and letting the main thread wait for a response.
func connectToPeer(conn net.Conn) {
	fmt.Println("Connecting to peer " + conn.RemoteAddr().String())
	writeToPeer(conn)
}

// Writing to a peer
func writeToPeer(conn net.Conn) {
	defer conn.Close()
	sendMyPort(conn)

	msg := &Communication{}
	dec := gob.NewDecoder(conn)
	err := dec.Decode(msg)
	if err == io.EOF {
		return
	}
	if err != nil {
		log.Println("decoding error:", err.Error())
	}
	fmt.Println("Received message from: "+":\n", msg)
	handleMessageID(msg, conn)
}

// Response received from a peer
func handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		msg := &Communication{}
		dec := gob.NewDecoder(conn)
		err := dec.Decode(msg)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Println("decoding error:", err.Error())
		}
		handleMessageID(msg, conn)
	}
}

func askAgainEvery() {
	time.Sleep(15 * time.Second)
	go askForTransactionDetails()
}

func handleMessageID(msg *Communication, conn net.Conn) {
	switch msg.MsgID {
	case 111: //receiving a port and adding it to peerlist.
		checkIfPeerIsInList(msg, conn)
	case 222: // receving Peerlist and blockchain
		if msg.Blockchain == nil {
			return
		}
		blockChain = msg.Blockchain
		updateLedgerFromBlockChain()
		handlePeerListMessage(msg, conn)
		// fmt.Println("Genesis block is", blockChain[0].InitialLedger.Accounts)
		accounts := blockChain[0].InitialLedger.Accounts
		// TODO change to 10
		if len(accounts) == networkSize { // After initilaztion, peers update their ledger to the same as the genesis block
			localLedger = blockChain[0].InitialLedger
			fmt.Print("Added genesis block ledger to my ledger")
			fmt.Print("Ledger: ", localLedger)
			com := &Communication{}
			com.MsgID = 666
			com.Ledger = localLedger
			broadcast(com)
		}
	case 333: //receiving a transaction
		handleTransactionMessage(msg)
	case 444: // recieve a relayed transaction
		handleRelayedTransactionMessage(msg)
	case 555: // recieve block message
		handleBlockMessage(msg)
	case 666: // backway that overrides ledger for setup purposes.sshhh
		localLedger = msg.Ledger
		fmt.Println("Initiazation finished")
		fmt.Println("Ledger: ", localLedger)
		go helper.DoEvery(10000*time.Millisecond, sendLotteryRequest)
	case 777:
		n := new(big.Int)
		n, _ = n.SetString(msg.Pk, 10)
		//fmt.Println("received pk", n)
		if !uniqueLotteryTest(n) {
			//fmt.Println("not unique lottery")
			return
		}
		calculateLottery(n)
		broadcast(msg)

	default:
		fmt.Println("UNKNOWN MESSAGE ID")
	}
}

var testedPks = make([]*big.Int, 0)

func uniqueLotteryTest(n *big.Int) bool {
	for _, pk := range testedPks {
		if n.Cmp(pk) == 0 {
			return false
		}
	}
	return true
}

var largestLotteryNumber *big.Int = big.NewInt(42)
var roundWinner *big.Int = big.NewInt(42)
var networkSize int = 3

func calculateLottery(pub *big.Int) {

	lotteryNumber := runLottery(pub)
	// fmt.Println("lottery number", lotteryNumber)
	if lotteryNumber.Cmp(largestLotteryNumber) > 0 {
		largestLotteryNumber = lotteryNumber
		roundWinner = pub
	}
	//check if i am the winner
	lotteryNumber = runLottery(rsa.Hash(pk))
	//fmt.Println("lottery number", lotteryNumber)
	if lotteryNumber.Cmp(largestLotteryNumber) > 0 {
		largestLotteryNumber = lotteryNumber
		roundWinner = rsa.Hash(pk)
	}
	//fmt.Println("testedpks len ", len(testedPks))
	if roundWinner.Cmp(rsa.Hash(pk)) == 0 && len(testedPks) == networkSize-1 {
		fmt.Println("I'm the winner and sequencer!")
		//create RSA key pair
		SequencerPk, SequencerSk = rsa.KeyGen() // generate keys for each peer
		SequencerSignature = rsa.GenerateSignature(rsa.Hash(SequencerPk), SequencerSk, SequencerPk)
		isSequencer = true
		checkForTransactions() // check new transactions every 10 second
	} else {
		isSequencer = false
	}
	fmt.Println("roundWinner is:", roundWinner)
}

func runLottery(pubkey *big.Int) *big.Int {
	//fmt.Println("ledger accounts ", localLedger.Accounts)
	tickets := localLedger.Accounts[pubkey.String()]
	//seed := big.NewInt(int64(blockChain[0].Seed))

	lotteryNumber := helper.BigInt().Mul(big.NewInt(int64(tickets)), rsa.Hash(pubkey))
	//fmt.Println("Ln:", lotteryNumber, " , ")
	testedPks = append(testedPks, pubkey)
	return lotteryNumber
}

func updateLedgerFromBlockChain() {
	fmt.Println("updating ledger from blockchain")
	fmt.Println(blockChain)
	for i, block := range blockChain {
		if i != 0 { // skip genesis block
			// fmt.Println(len(transactionsCompleted))
			// fmt.Println(transactionsCompleted[0])
			// fmt.Println(transactionsCompleted[1])
			if len(transactionsCompleted) == 0 { // we have done no transactions it should be safe to do this
				updateLedger(block)
			}
			for _, transactionID := range transactionsCompleted {
				fmt.Println(transactionID)
				fmt.Println(block.Transactions.TransactionId)
				if transactionID != block.Transactions.TransactionId {
					//fmt.Println("Updating ledger...")
					updateLedger(block)
				}
			}
		}
	}
	fmt.Println("updated ledger from blockchain: ", localLedger)
}

// add new peer to the genesis block
func addPkToGenesisBlock(pk string) {
	genesisBLock := blockChain[0]
	genesisBLock.InitialLedger.Accounts[pk] = 10000
	blockChain[0] = genesisBLock
	fmt.Println(blockChain[0].InitialLedger)
}

func handleBlockMessage(msg *Communication) {
	currentBlockNumber := msg.Block.Number
	if isFirstBlock || currentBlockNumber == lastBlockNumber+1 {
		fmt.Println("blocknumber accepted")
		//check signature
		signature := new(big.Int)
		from := new(big.Int)
		pk := new(big.Int)
		signature.SetString(msg.Signature, 0)
		from.SetString(msg.From, 0)
		pk.SetString(msg.Pk, 0)
		if rsa.VerifySignature(signature, from, pk) { //verify the signature
			if msg.From == msg.To { // peers should not be allowed to send messages to themselves
				fmt.Println("DECLINED - You cant transfer money to yourself!")
				return
			}
			// if !validTransaction(msg) {
			// 	fmt.Println("Transaction will cause overdraft - discarding")
			// 	return
			// }
			//fmt.Println("length of blockchain ", len(blockChain))
			if isBlockInChain(msg.Block) {
				fmt.Println("block was already in the block chain")
				fmt.Println(blockChain)
				return
			}
			updateLedger(msg.Block)                    //process transactions
			blockChain = append(blockChain, msg.Block) // add block to our blockchain
			fmt.Println("Added block to blockchain", blockChain)

			for _, block := range blockChain {
				fmt.Println(block.Transactions)
			}
			fmt.Println("ledger:", localLedger)
			fmt.Println("broadCasting block to network")
			broadcast(msg) //broadcast to network
			lastBlockNumber = currentBlockNumber
			isFirstBlock = false
		}
	} else {
		//fmt.Println("currentBlockNumber:", currentBlockNumber)
		//fmt.Println("lastBlockNumber:", lastBlockNumber+1)
		fmt.Println("rejected block - block number not the next in line")
		return
	}
}

func handleRelayedTransactionMessage(msg *Communication) {
	if !isUniqueTransaction(msg) {
		return
	}

	com := &Communication{}
	com.Transaction = msg.Transaction
	com.MsgID = 444
	fmt.Println("Broadcasting transaction")
	broadcast(com)
}

func handleTransactionMessage(msg *Communication) {
	if isUniqueTransaction(msg) {
		if math.Signbit(float64(msg.Amount)) { // is it a negative number ?
			fmt.Println("DECLINED - You cant transfer a negative amount of money")
			return
		}
		signature := new(big.Int)
		from := new(big.Int)
		pk := new(big.Int)
		signature.SetString(msg.Signature, 0)
		from.SetString(msg.From, 0)
		pk.SetString(msg.Pk, 0)
		if rsa.VerifySignature(signature, from, pk) { // verify the signature
			if msg.From == msg.To { // peers should not be allowed to send messages to themselves
				fmt.Println("DECLINED - You cant transfer money to yourself!")
				return
			}

			com := &Communication{}
			com.Transaction = msg.Transaction
			com.MsgID = 444
			if isSequencer {
				sequencerTransactionList = append(sequencerTransactionList, msg.Transaction)
			}
			broadcast(com)
		}
	} else {
		fmt.Println("Amount in ledger not unique")
	}
}

func handlePeerListMessage(msg *Communication, conn net.Conn) {
	if isUniquePeerList(msg) {
		peerList = msg.Peerlist
		fmt.Println("MY PEERLIST IS NOW:", peerList)
		// If the way the peerList is managed is changed this does not work!
		sequencerIP = peerList[0] // The peer should know who the sequencer is
		//fmt.Println("Sequencer is:", sequencerIP)
		broadcastList() //broadcast my new list to the 10 peers after me on my list
		go askForTransactionDetails()
		return
	} else {
		fmt.Println("list is not unique")
	}
}

func isUniqueTransaction(msg *Communication) bool {
	if len(transactionList) == 0 {
		fmt.Println("transactionList was empty, adding it to list")
		transactionList = append(transactionList, msg.Amount)
		return true
	}
	for i := range transactionList {
		if msg.Transaction.TransactionId == transactionList[i] {
			fmt.Println("transaction was not unique")
			return false
		}
	}
	fmt.Println("transaction was unique, adding it to list")
	transactionList = append(transactionList, msg.Transaction.TransactionId)
	return true
}

func checkIfPeerIsInList(msg *Communication, conn net.Conn) {
	remotePeer := msg.MsgPort
	isInPeerList := false
	for i := range peerList {
		if remotePeer == peerList[i] {
			isInPeerList = true
			fmt.Println("Peer is already in the list, no need to add them!")
			return
		}
	}
	if !isInPeerList {
		addPkToGenesisBlock(msg.Pk)
		peerList = append(peerList, remotePeer)
		fmt.Println("Peer was not in the list..adding it to the list..Peerlist is now:", peerList)
		//send blockchain to new peer
		// enc := gob.NewEncoder(conn)
		// com := &Communication{}
		// com.MsgID = 666
		// com.Blockchain = blockChain
		// enc.Encode(com)
		go sendPeerList(conn)
	}
}

//func sendBlockChain

func isUniquePeerList(msg *Communication) bool {
	var myPeerString string
	var incommingPeerString string
	incommingPeerList := msg.Peerlist

	for i := range peerList {
		myPeerString = myPeerString + peerList[i]
	}
	for j := range incommingPeerList {
		incommingPeerString = incommingPeerString + incommingPeerList[j]
	}

	// fmt.Println("myPeerString is:", myPeerString)
	// fmt.Println("incommingPeerString is:", incommingPeerString)

	if myPeerString == incommingPeerString {
		return false
	}
	return true
}

func sendMyPort(conn net.Conn) {
	fmt.Println("Sending my server port to:", conn.RemoteAddr().String())
	ts := &Communication{}
	enc := gob.NewEncoder(conn)
	ts.MsgID = 111
	ts.Ledger = localLedger //send initial ledger as a new peer
	ts.MsgPort = myPort
	ts.Pk = rsa.Hash(pk).String()
	enc.Encode(ts)
}

func sendPeerList(conn net.Conn) {

	fmt.Println("Sending my peerlist to:", conn.RemoteAddr().String())
	ts := &Communication{}
	enc := gob.NewEncoder(conn)
	ts.Peerlist = peerList
	ts.MsgID = 222
	ts.Ledger = localLedger
	ts.Blockchain = blockChain
	enc.Encode(ts)
}

var isCurrentlyAsking bool = false

func askForTransactionDetails() {
	isCurrentlyAsking = true
	ts := &Communication{}
	fmt.Println("")
	fmt.Print("Send to: ")
	fmt.Scan(&to)
	fmt.Print("Amount: ")
	fmt.Scan(&amount)
	// for i := range addrToDoTransaction {
	// 	conn, _ = net.Dial("tcp", addrToDoTransaction[i])

	// 	if conn == nil {
	// 		fmt.Println("Could not connect to peer:", addrToDoTransaction[i])
	// 		return
	// 	}

	// 	defer conn.Close()

	// 	fmt.Println("Sending a transaction to:", addrToDoTransaction[i])
	ts = &Communication{}
	transaction := &Transaction{to, rsa.Hash(pk).String(), amount, helper.RandomNumber()}
	ts.From = rsa.Hash(pk).String() // hashed public key
	ts.To = to
	ts.Amount = amount
	ts.MsgID = 333
	ts.Signature = signature.String()
	ts.Pk = pk.String()
	ts.Transaction = transaction
	transactionList = append(transactionList, amount) // adding the transaction to our list
	//enc.Encode(ts)
	broadcast(ts)
	isCurrentlyAsking = false
	// }
	if !isCurrentlyAsking {
		askAgainEvery()
	}
}

// Broadcast peerlist to the 10 peers after you on the list
func broadcastList() {
	fmt.Println("broadcasting list...")
	//fmt.Println("length of array:", len(peerList))
	nextPeerInList := findIndexOfYourselfInPeerList() + 1
	if nextPeerInList == 9999 { // ensure that we find ourselves
		return
	}
	j := 0
	stopParam := 10
	for i := nextPeerInList; j < stopParam; j++ { // connect to the 10 peers after our position

		if 10 > len(peerList) { // we should not iterate 10 times if there is not 10 peers
			stopParam = len(peerList) - 1
		}

		if i == len(peerList) { // wrap-around if we reach the end of the array
			i = 0
		}
		if peerList[i] != myPort { // we dont want to try to dial ourselves
			conn, _ = net.Dial("tcp", peerList[i])
			addrToDoTransaction = append(addrToDoTransaction, peerList[i])
		}
		if conn == nil {
			fmt.Println("Could not connect to peer:", peerList[i])
			return
		}
		defer conn.Close()
		go sendPeerList(conn)
		i++
	}
}

func (l *Ledger) Transaction(t *Communication) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Accounts[t.From] -= t.Amount
	l.Accounts[t.To] += t.Amount
}

func (l *Ledger) blockTransaction(t *Block) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Accounts[t.Transactions.From] -= t.Transactions.Amount
	l.Accounts[t.Transactions.To] += t.Transactions.Amount
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}

// Finds and return index of yourself in list
func findIndexOfYourselfInPeerList() int {
	for i := range peerList {
		if peerList[i] == myPort {
			//fmt.Println("index of myself in peerlist:", i)
			return i
		}
	}
	return 9999 //error
}

// sequencer calls this every 10 seconds
func checkForTransactions() {
	testedPks = make([]*big.Int, 0) // reset tested lotteries
	//fmt.Println("Checking for new transactions")
	if len(sequencerTransactionList) == 0 {
		fmt.Println("Checking for new blocks...")
		return
	}

	block := &Block{}

	for i := range sequencerTransactionList {
		block.Number = transactionCounter
		block.Transactions = sequencerTransactionList[i]
		block.Transactions.Amount = block.Transactions.Amount - 1 // receiver
		fmt.Println("Block:", block)
		ts := &Communication{}
		ts.MsgID = 555
		ts.Block = block
		ts.From = rsa.Hash(SequencerPk).String()
		ts.Signature = SequencerSignature.String()
		ts.Pk = SequencerPk.String()
		go broadcast(ts) // Broadcast transactionblock to network
		updateLedger(block)
		blockChain = append(blockChain, block)
		// delete transaction from list after i has been sent
		sequencerTransactionList = append(sequencerTransactionList[:i], sequencerTransactionList[i+1:]...)
		transactionCounter++
	}
}

func updateLedger(block *Block) {
	if isBlockUnique(block) {
		localLedger.blockTransaction(block)
		fmt.Println("block ledger is ", localLedger)
		processedBlocks = append(processedBlocks, block.Transactions.TransactionId)
		transactionsCompleted = append(transactionsCompleted, block.Transactions.TransactionId)
	}
}

func isBlockUnique(block *Block) bool {
	for i := range processedBlocks {
		if i != 0 {
			if block.Transactions.TransactionId == processedBlocks[i] {
				fmt.Println("Block was not unique")
				return false
			}
		}
	}
	return true
}

func isBlockInChain(block *Block) bool {
	if len(blockChain) == 1 {
		return false
	}
	for i := range blockChain {
		//fmt.Println(blockChain[0].Transactions.TransactionId)
		// fmt.Println(blockChain)
		// fmt.Println(blockChain[1])
		// fmt.Println(blockChain[0])
		if i != 0 {
			// fmt.Println("block", block.Transactions)
			// fmt.Println("chain", blockChain[i].Transactions)
			if block.Transactions.TransactionId == blockChain[i].Transactions.TransactionId {
				return true
			}
		}
	}
	return false
}

//Broadcast function that broadcast anything to 10 peers below the peer itself on the list
func broadcast(ts *Communication) {
	var conn net.Conn
	nextPeerInList := findIndexOfYourselfInPeerList() + 1
	if nextPeerInList == 9999 { // ensure that we find ourselves
		return
	}
	j := 0
	stopParam := 10
	for i := nextPeerInList; j < stopParam; j++ { // connect to the 10 peers after our position
		if 10 > len(peerList) { // we should not iterate 10 times if there is not 10 peers
			stopParam = len(peerList) - 1
		}
		if i == len(peerList) { // wrap-around if we reach the end of the array
			i = 0
		}
		if peerList[i] != myPort { // we dont want to try to dial ourselves
			conn, _ = net.Dial("tcp", peerList[i])
		}
		if conn != nil {
			defer conn.Close()
			fmt.Println("Sending to:", peerList[i])
			enc := gob.NewEncoder(conn)
			enc.Encode(ts)
			i++
		} else {
			fmt.Println("Could not connect to peer:", peerList[i])
			i++
		}
	}
}
