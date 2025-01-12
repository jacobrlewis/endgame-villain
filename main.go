package main

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/jacobrlewis/endgame-villain/db"
	_ "github.com/joho/godotenv/autoload" // auto laod .env file
)

func main() {

	// cpu profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	inputFilePath := os.Getenv("EVAL_FILE")
	dbFilePath := os.Getenv("DB_FILE")

	db.Connect(dbFilePath)
	db.LoadData(inputFilePath)

}
