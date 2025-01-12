package db

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dbFilePath string) {

	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	DB = db

	fmt.Println("database connection established")
	DB.AutoMigrate(&Position{})
}

func LoadData(inputFilePath string) {
	var position Position
	result := DB.First(&position)

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		fmt.Println("Database is already loaded")
		return
	}

	fmt.Println("Database is empty. Loading data...")
	offsets := splitFile(inputFilePath)

	batches := make(chan []Position, len(offsets))
	doneProcessing := make(chan bool)

	go dbWriter(batches, doneProcessing)

	var wg sync.WaitGroup
	totalLines := 0
	for i := range len(offsets) - 1 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			lineNum := processChunk(inputFilePath, offsets[i], offsets[i+1]-offsets[i], batches)
			totalLines += lineNum
			fmt.Printf("Chunk %d complete, read %d lines\n", i, lineNum)
		}()
	}
	wg.Wait()
	doneProcessing <- true
	fmt.Printf("Should expect %d lines in db\n", totalLines)
}

// dbWriter waits on a batches channel, to insert large batches of records to the database at a time
func dbWriter(batches <-chan []Position, doneProcessing <-chan bool) {
	sum := 0
	for {
		select {
		case batch := <-batches:
			size := len(batch)
			sum += size
			DB.CreateInBatches(batch, size)
		case <-doneProcessing:
			fmt.Printf("Wrote %d lines to db\n", sum)
			return
		}
	}
}

// countPieces scans a FEN string and sets the number of pieces found
// returns the index of the last scanned byte
func countPieces(line *[]byte, pieces *Position) int {
	wB, bB, wN, bN, wP, bP, wQ, bQ, wR, bR := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0

	fenStart := 8 // lines always start with {"fen": (7 chars)

	for i, b := range (*line)[fenStart:] {
		switch b {
		case 'B':
			wB += 1
		case 'b':
			bB += 1
		case 'N':
			wN += 1
		case 'n':
			bN += 1
		case 'P':
			wP += 1
		case 'p':
			bP += 1
		case 'Q':
			wQ += 1
		case 'q':
			bQ += 1
		case 'R':
			wR += 1
		case 'r':
			bR += 1
		case ' ':
			*pieces = Position{0, wB, bB, wN, bN, wP, bP, wQ, bQ, wR, bR, 0, 0}
			return fenStart + i
		}
	}
	// should never get here
	panic("did not find a space in the line, invalid input")
}

// getCentipawnEval scans an int from the start of a slice of bytes
// returns the evaluation, and the length of the int parsed
func getCentipawnEval(bytes *[]byte) (int, int) {
	var intEnd int
	negative := false
	line := *bytes

	if line[0] == '-' {
		// integer could start with a negative sign
		negative = true
		line = line[1:]
	}

	sum := 0
	for i := range line {
		if line[i] < '0' || line[i] > '9' {
			intEnd = i
			break
		}
		sum = sum*10 + int(line[i]-'0')
	}

	if negative {
		sum *= -1
		intEnd += 1
	}
	return sum, intEnd
}

// split file returns a list of offsets, approximately equal in size, one for each available cpu
// each offset is the start of a new line
// the final offset is the end of the file
func splitFile(filePath string) []int64 {
	numChunks := runtime.NumCPU()

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	stat, err := file.Stat()
	if err != nil {
		panic(err)
	}

	chunkSize := stat.Size() / int64(numChunks)

	offsets := make([]int64, numChunks+1) // +1 because we include last byte as the final offset
	offsets[0] = 0                        // 0 as first offset
	offsets[numChunks] = stat.Size()      // file size as final offset

	// skip 0, don't want to move first chunk start
	for i := 1; i < numChunks; i++ {
		offset, err := file.Seek(chunkSize*int64(i), io.SeekStart)
		if err != nil {
			panic(err)
		}

		// read exact number of bytes until the next newline
		reader := bufio.NewReader(file)
		restOfLine, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			panic(err)
		}

		offset += int64(len(restOfLine))
		offsets[i] = offset
	}

	fmt.Println(offsets)

	return offsets
}

func processChunk(filePath string, start int64, chunkSize int64, batches chan<- []Position) int {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// seek to start of our chunk
	_, err = file.Seek(start, io.SeekStart)
	if err != nil {
		panic(err)
	}

	// limit file reading to our given chunk
	reader := io.LimitReader(file, chunkSize)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	// var position Position
	centiPawn := []byte("\"cp\"")
	lineNum := 0

	batchSize := 1000
	positions := make([]Position, batchSize)

	for scanner.Scan() {
		line := scanner.Bytes()

		position := &positions[lineNum%batchSize]

		lastScanned := countPieces(&line, position)
		line = line[lastScanned:]
		foundIndex := bytes.Index(line, centiPawn)
		minEval, maxEval := 99_999, -99_999
		for foundIndex != -1 {

			// shift slice 5 over
			// want to move past "cp": and start on the number
			line = line[foundIndex+5:]

			parsedInt, intLength := getCentipawnEval(&line)

			minEval = min(parsedInt, minEval)
			maxEval = max(parsedInt, maxEval)

			line = line[intLength:]
			foundIndex = bytes.Index(line, centiPawn)
		}
		position.MaxEval = maxEval
		position.MinEval = minEval

		if lineNum%batchSize == batchSize-1 {
			batches <- positions
		}

		lineNum += 1
		if lineNum%100_000 == 0 {
			fmt.Printf("%v %d\n", position, lineNum)
		}
	}

	batches <- positions[:(lineNum%batchSize)+1]

	return lineNum
}
