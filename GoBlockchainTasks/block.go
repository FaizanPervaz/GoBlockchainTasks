package main

import (
	"fmt"
	"time"
)

type Block struct {
	Index     int
	Timestamp time.Time
	Data      string
	PrevHash  string
	Hash      string
}

type Blockchain struct {
	Blocks []Block
}

func DisplayAllBlocks(chain Blockchain) {
	for _, block := range chain.Blocks {
		fmt.Printf("Block #%d\n", block.Index)
		fmt.Printf("Timestamp: %s\n", block.Timestamp.Format(time.RFC3339))
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("PrevHash: %s\n", block.PrevHash)
		fmt.Printf("Hash: %s\n", block.Hash)
		fmt.Println("--------------------------------------")
	}
}

func NewBlock(data string, prevHash string, chain Blockchain) Block {
	newIndex := len(chain.Blocks) + 1
	newBlock := Block{
		Index:     newIndex,
		Timestamp: time.Now(),
		Data:      data,
		PrevHash:  prevHash,
		Hash:      "", 
	}
	newBlock.Hash = calculateHash(newBlock)
	return newBlock
}

func ModifyBlock(block Block, newData string) Block {
	block.Data = newData
	block.Hash = calculateHash(block)
	return block
}

func calculateHash(block Block) string {
	return "fakehash"
}

func main() {
	// Main function
}
