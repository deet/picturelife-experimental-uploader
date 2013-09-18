package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
)

func CalculateSignature(filePath string) string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Signature error:", err)
		}
	}()

	hash := sha256.New()

	// TODO stream file so it's not all in memory

	file, err := os.Open(filePath)

	//fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("Could not read file")
	}

	var blockSize int64 = 1000000

	//log.Println("Calculating signature")

	blockNum := 1
	for {
		//log.Println("Sig block", blockNum)
		blockNum++
		_, err = io.CopyN(hash, file, blockSize)
		if err != nil {
			//log.Println(err)
			break
		}
	}

	//hash.Write(fileBytes)
	sig := hex.EncodeToString(hash.Sum(nil))

	return string(sig)
}
