package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/qu0ta/MTSHack/internal/platform"
)

func main() {
	if err := godotenv.Load(".env.example"); err != nil {
		log.Fatal(err)
	}
	db, err := platform.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("connected to PostgreSQL")

}
