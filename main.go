package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	_ "github.com/joho/godotenv/autoload" // auto laod .env file
)

type Pieces struct {
	wB  int
	bB  int
	wN  int
	bN  int
	wP  int
	bP  int
	wQ  int
	bQ  int
	wR  int
	bR  int
	min int
	max int
}

// countPieces scans a FEN string and sets the number of pieces found
// returns the index of the last scanned byte
func countPieces(line *[]byte, pieces *Pieces) int {
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
			*pieces = Pieces{wB, bB, wN, bN, wP, bP, wQ, bQ, wR, bR, 0, 0}
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

func processChunk(filePath string, start int64, chunkSize int64) int {
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

	var pieces Pieces
	centiPawn := []byte("\"cp\"")
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Bytes()

		lastScanned := countPieces(&line, &pieces)
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
		pieces.max = maxEval
		pieces.min = minEval

		lineNum += 1
		if lineNum == 10_000_000 {
			fmt.Printf("%v %d\n", pieces, lineNum)
		}
	}
	return lineNum
}

func main() {

	// cpu profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	filePath := os.Getenv("EVAL_FILE")
	offsets := splitFile(filePath)

	var wg sync.WaitGroup
	for i := range len(offsets) - 1 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			lineNum := processChunk(filePath, offsets[i], offsets[i+1]-offsets[i])
			fmt.Printf("Chunk %d complete, read %d lines\n", i, lineNum)
		}()
	}
	wg.Wait()

}
