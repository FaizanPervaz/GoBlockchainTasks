package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	ID            int
	Port          int
	IP            string
	Neighbors     []string
	CurrentBlock  Block
	ReceivedBlock Block
	mu            sync.Mutex
}

var BootstrapNode = Node{
	ID:   0,
	Port: 5000,
	IP:   "127.0.0.1",
}

var nodeIDCounter = 1
var portCounter = 6000

func RegisterNode(node *Node) {
	BootstrapNode.mu.Lock()
	defer BootstrapNode.mu.Unlock()

	node.ID = nodeIDCounter
	nodeIDCounter++

	node.Port = portCounter
	portCounter++

	BootstrapNode.Neighbors = append(BootstrapNode.Neighbors, fmt.Sprintf("%s:%d", node.IP, node.Port))
}

func assigningNeighbor(node *Node) {

	existingNodes := append([]string{}, BootstrapNode.Neighbors...)
	rand.Shuffle(len(existingNodes), func(i, j int) {
		existingNodes[i], existingNodes[j] = existingNodes[j], existingNodes[i]
	})

	maxNeighbors := 5
	for _, potentialNeighbor := range existingNodes {
		if potentialNeighbor != node.IP {
			node.Neighbors = append(node.Neighbors, potentialNeighbor)
			maxNeighbors--
		}

		if maxNeighbors == 0 {
			break
		}
	}
}

func StartNode(node *Node) {
	node.CurrentBlock = Block{
		prevBlockHash: blockchain.head.data.currentBlockHash,
		timestamp:     time.Now().Unix(),
		nonce:         0,
	}
	go node.startServer()
	go node.startClient()
	go node.mineCheck()
}

func (node *Node) startServer() {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", node.Port))
	if err != nil {
		fmt.Printf("Node %d: There was error starting server: %v\n", node.ID, err)
		return
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Printf("Node %d: There was error accepting connection: %v\n", node.ID, err)
			continue
		}

		go node.handleClient(conn)
	}
}

func (node *Node) handleTranscation(trx []Transaction) {

	flood_arr := []Transaction{}
	for _, upcoming_trx := range trx {
		check := false
		for _, exisiting_trx := range node.CurrentBlock.BlockTransactions {
			if exisiting_trx.Data == upcoming_trx.Data {
				check = true
				return
			}
		}
		if !check {
			flood_arr = append(flood_arr, upcoming_trx)
		}
	}
	node.floodingTrx(flood_arr)

}

func (node *Node) mineCheck() {
	if len(node.CurrentBlock.BlockTransactions) >= 4 {
		node.CurrentBlock.mineBlock()
		if node.ReceivedBlock.prevBlockHash == "" {
			fmt.Println("Node ", node.ID, " is the first node to brodcast")
			node.floodingBlock(node.CurrentBlock)
		}
	}
	node.CurrentBlock = Block{}
}

func (node *Node) handleBlock(block Block) {
	if block.prevBlockHash != node.ReceivedBlock.prevBlockHash {
		node.ReceivedBlock = block
		if node.ReceivedBlock.currentBlockHash == node.ReceivedBlock.blockHashCalculation() {
			fmt.Println("Block is valid")
			node.floodingBlock(node.ReceivedBlock)
		} else {
			fmt.Println("Block is invalid : ", node.ReceivedBlock.currentBlockHash, " | ", node.ReceivedBlock.blockHashCalculation())
		}
	}
	node.ReceivedBlock = Block{}
}

func (node *Node) handleClient(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		data := scanner.Text()
		//fmt.Printf("Node %d: Received data from neighbor: %s\n", node.ID, data)
		var receivedTransactions []Transaction
		if err := json.Unmarshal([]byte(data), &receivedTransactions); err == nil {
			node.handleTranscation(receivedTransactions)
			continue
		} else {
			re := regexp.MustCompile(`\{([^ ]+) ([^ ]+) %!s\(int64=([0-9]+)\) %!s\(int=([0-9]+)\) ([^ ]+) \[([^\]]+)\]\}`)

			matches := re.FindStringSubmatch(data)
			if matches == nil || len(matches) != 7 {
				//sreturn nil, fmt.Errorf("invalid input format")
			}

			currentBlockHash := matches[1]
			prevBlockHash := matches[2]
			timestamp, _ := strconv.ParseInt(matches[3], 10, 64)
			nonce, _ := strconv.Atoi(matches[4])
			merkleRoot := matches[5]
			transactionsStr := matches[6]

			re1 := regexp.MustCompile(`\{([^}]+)\}`)

			matches1 := re1.FindAllStringSubmatch(transactionsStr, -1)
			if matches1 == nil {
			}

			var extractedTransactions []Transaction
			for _, match := range matches1 {
				extractedTransactions = append(extractedTransactions, Transaction{Data: match[1]})
			}

			block := &Block{
				currentBlockHash:  currentBlockHash,
				prevBlockHash:     prevBlockHash,
				timestamp:         timestamp,
				nonce:             nonce,
				merkleroot:        merkleRoot,
				BlockTransactions: extractedTransactions,
			}

			//fmt.Println("received block: ", block)
			node.handleBlock(*block)

		}

	}
}

func (node *Node) startClient() {
	for {
		node.mu.Lock()
		neighbors := append([]string{}, node.Neighbors...)
		node.mu.Unlock()

		for _, neighbor := range neighbors {
			go node.contactNeighbor(neighbor)
		}
	}
}

func (node *Node) contactNeighbor(neighbor string) {
}

func (node *Node) floodingTrx(transactions []Transaction) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.CurrentBlock.BlockTransactions = append(node.CurrentBlock.BlockTransactions, transactions...)

	for _, neighbor := range node.Neighbors {
		go node.brodcastingTrxToNeigborNodes(neighbor, node.CurrentBlock.BlockTransactions)
	}
	node.CurrentBlock.merkleroot = merkleRoot(node.CurrentBlock.BlockTransactions).hash
}

func (node *Node) floodingBlock(block Block) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.CurrentBlock = block
	for _, neighbor := range node.Neighbors {
		go node.brodcastingBlockToNeigborNodes(neighbor, node.CurrentBlock)
	}
}

func (node *Node) brodcastingBlockToNeigborNodes(neighbor string, block Block) {
	fmt.Printf("Node %d: Broadcasting block to neighbor %s\n", node.ID, neighbor)
	node.mu.Lock()
	defer node.mu.Unlock()
	time.Sleep(1 * time.Second)
	conn, err := net.Dial("tcp", neighbor)
	if err != nil {
		fmt.Printf("Node %d: There was error connecting to neighbor: %v\n", node.ID, err)
		return
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s\n", block)

}

func (node *Node) brodcastingTrxToNeigborNodes(neighbor string, transactions []Transaction) {

	fmt.Printf("Node %d: Broadcasting transactions to neighbor %s\n", node.ID, neighbor)
	node.mu.Lock()
	defer node.mu.Unlock()
	time.Sleep(1 * time.Second)
	conn, err := net.Dial("tcp", neighbor)
	if err != nil {
		fmt.Printf("Node %d: There was error connecting to neighbor: %v\n", node.ID, err)
		return
	}
	defer conn.Close()
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(transactions); err != nil {
		fmt.Printf("Node %d: There was an error encoding and sending transactions to neighbor: %v\n", node.ID, err)
		return
	}
}

func DisplayP2PNetwork(nodes []Node) {
	fmt.Println("P2P Network:")
	for i, node := range nodes {
		fmt.Printf("Node %d: ID=%d, IP=%s, Port=%d, Neighbors=%v, Block=%v\n", i+1, node.ID, node.IP, node.Port, node.Neighbors, node.CurrentBlock)
	}
	fmt.Println("Bootstrap Node:", BootstrapNode)
}

var blockchain = BlockChain{}

func main() {

	var choice int

	fmt.Println("Select an option:")
	fmt.Println("1. Part #01")
	fmt.Println("2. Part #02")

	fmt.Print("Enter your choice (1 or 2): ")
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		initializeBlockchain()
		part01()
	case 2:
		part02()
	default:
		fmt.Println("Invalid choice. Please enter 1 or 2.")
	}
}
func (bc *BlockChain) length() int {
	length := 0
	currentNode := bc.head
	for currentNode != nil {
		length++
		currentNode = currentNode.next
	}
	return length
}
func initializeBlockchain() {
	genesisBlock := &Block{
		prevBlockHash:     "",
		timestamp:         time.Now().Unix(),
		nonce:             0,
		merkleroot:        "",
		BlockTransactions: []Transaction{},
	}

	genesisBlock.currentBlockHash = genesisBlock.blockHashCalculation()
	blockchain.addBlock(genesisBlock)
}

var transactions []Transaction

func part01() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to the Blockchain Console!")
	fmt.Println("Enter 'exit' to quit.")

	for {
		fmt.Println("Options:")
		fmt.Println("1. Add transaction")
		fmt.Println("2. Create block")
		fmt.Println("3. Display blockchain and Merkle root")
		fmt.Println("4. Change transactions")
		fmt.Println("5. Check validity of blockchain")
		fmt.Println("6. Exit")
		fmt.Print("Enter your choice: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Print("Enter transaction data: ")
			data, _ := reader.ReadString('\n')
			data = strings.TrimSpace(data)
			newTransaction := Transaction{Data: data}
			transactions = append(transactions, newTransaction)
			fmt.Println("Transaction added successfully.")
		case "2":
			if len(transactions) == 0 {
				fmt.Println("No transactions to include in the block.")
				continue
			}
			// Get the last block in the blockchain
			lastBlock := blockchain.head
			for lastBlock.next != nil {
				lastBlock = lastBlock.next
			}
			prevBlockHash := lastBlock.data.currentBlockHash
			fmt.Println("Creating block with current transactions...")
			newBlock := blockCreation(prevBlockHash, transactions)
			blockchain.addBlock(newBlock)
			transactions = []Transaction{}
			fmt.Println("Block created successfully.")
		case "3":
			fmt.Println("*****BLOCKCHAIN*****")
			currentNode := blockchain.head
			for currentNode != nil {
				fmt.Println(currentNode.data)
				currentNode = currentNode.next
			}
		case "4":
			if blockchain.length() == 0 {
				fmt.Println("No blocks in the blockchain.")
				continue
			}
			if blockchain.length() == 1 {
				fmt.Println("The blockchain contains only the genesis block. There are no transactions to change.")
				continue
			}
			fmt.Println("Blocks in the blockchain:")
			currentNode := blockchain.head.next // Start from the second block, skipping the genesis block
			for i := 2; currentNode != nil; i++ {
				fmt.Printf("%d. Block Hash: %s\n", i, currentNode.data.currentBlockHash)
				currentNode = currentNode.next
			}
			fmt.Print("Enter the index of the block to change transactions: ")
			blockIndexInput, _ := reader.ReadString('\n')
			blockIndexInput = strings.TrimSpace(blockIndexInput)
			blockIndex, err := strconv.Atoi(blockIndexInput)
			if err != nil || blockIndex < 2 || blockIndex > blockchain.length() {
				fmt.Println("Invalid block index.")
				continue
			}
			currentNode = blockchain.head.next
			for i := 2; i < blockIndex; i++ {
				currentNode = currentNode.next
			}
			fmt.Println("Transactions in the selected block:")
			for i, txn := range currentNode.data.BlockTransactions {
				fmt.Printf("%d. %s\n", i+1, txn.Data)
			}
			fmt.Print("Enter the index of the transaction to change: ")
			txnIndexInput, _ := reader.ReadString('\n')
			txnIndexInput = strings.TrimSpace(txnIndexInput)
			txnIndex, err := strconv.Atoi(txnIndexInput)
			if err != nil || txnIndex < 1 || txnIndex > len(currentNode.data.BlockTransactions) {
				fmt.Println("Invalid transaction index.")
				continue
			}
			fmt.Printf("Enter new data for transaction %d: ", txnIndex)
			newData, _ := reader.ReadString('\n')
			newData = strings.TrimSpace(newData)
			currentNode.data.BlockTransactions[txnIndex-1].Data = newData
			// Recalculate the merkle root and block hash
			currentNode.data.merkleroot = merkleRoot(currentNode.data.BlockTransactions).hash
			currentNode.data.currentBlockHash = currentNode.data.blockHashCalculation()
			fmt.Println("Transaction changed successfully.")

		case "5":
			fmt.Println("Checking validity of blockchain...")
			if blockchain.validityCheck() {
				fmt.Println("Blockchain is valid.")
			} else {
				fmt.Println("Blockchain is not valid.")
			}
		case "6":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}

func part02() {

	genesisBlock := Block{
		prevBlockHash:     "",
		timestamp:         time.Now().Unix(),
		nonce:             0,
		merkleroot:        "",
		BlockTransactions: []Transaction{},
	}

	genesisBlock.currentBlockHash = genesisBlock.blockHashCalculation()
	blockchain.addBlock(&genesisBlock)

	transactions := []Transaction{
		{Data: "1700USD Sent"},
		{Data: "1800USD Sent"},
		{Data: "1500USD Sent"},
		{Data: "1600USD Sent"},
	}
	nodes := make([]Node, 10)

	for i := range nodes {
		nodes[i].IP = "127.0.0.1"
		RegisterNode(&nodes[i])
	}
	for i := range nodes {
		assigningNeighbor(&nodes[i])
		StartNode(&nodes[i])
	}
	nodes[4].floodingTrx(transactions)
	time.Sleep(12 * time.Second)
	DisplayP2PNetwork(nodes)

}
